//! Storage wrapper that records metrics for all operations.

use crate::api::{
    BlockStateDiff, OpProofsHashedCursor, OpProofsStorage, OpProofsStorageResult,
    OpProofsTrieCursor,
};
use alloy_primitives::{map::HashMap, B256, U256};
use metrics::{Counter, Histogram};
use reth_metrics::Metrics;
use reth_primitives_traits::Account;
use reth_trie::{BranchNodeCompact, Nibbles};
use std::{
    fmt::Debug,
    future::Future,
    sync::Arc,
    time::{Duration, Instant},
};
use strum::{EnumCount, EnumIter, IntoEnumIterator};

/// Types of storage operations that can be tracked.
#[derive(Debug, Clone, Copy, Eq, PartialEq, Hash, EnumCount, EnumIter)]
pub enum StorageOperation {
    /// Store account trie branch
    StoreAccountBranch,
    /// Store storage trie branch
    StoreStorageBranch,
    /// Store hashed account
    StoreHashedAccount,
    /// Store hashed storage
    StoreHashedStorage,
    /// Trie cursor seek exact operation
    TrieCursorSeekExact,
    /// Trie cursor seek
    TrieCursorSeek,
    /// Trie cursor next
    TrieCursorNext,
    /// Trie cursor current
    TrieCursorCurrent,
    /// Hashed cursor seek
    HashedCursorSeek,
    /// Hashed cursor next
    HashedCursorNext,
}

impl StorageOperation {
    /// Returns the operation as a string for metrics labels.
    pub const fn as_str(&self) -> &'static str {
        match self {
            Self::StoreAccountBranch => "store_account_branch",
            Self::StoreStorageBranch => "store_storage_branch",
            Self::StoreHashedAccount => "store_hashed_account",
            Self::StoreHashedStorage => "store_hashed_storage",
            Self::TrieCursorSeekExact => "trie_cursor_seek_exact",
            Self::TrieCursorSeek => "trie_cursor_seek",
            Self::TrieCursorNext => "trie_cursor_next",
            Self::TrieCursorCurrent => "trie_cursor_current",
            Self::HashedCursorSeek => "hashed_cursor_seek",
            Self::HashedCursorNext => "hashed_cursor_next",
        }
    }
}

/// Metrics for storage operations.
#[derive(Debug)]
pub struct StorageMetrics {
    /// Cache of operation metrics handles, keyed by (operation, context)
    operations: HashMap<StorageOperation, OperationMetrics>,
    /// Block-level metrics
    block_metrics: BlockMetrics,
}

impl StorageMetrics {
    /// Create a new metrics instance with pre-allocated handles.
    pub fn new() -> Self {
        Self {
            operations: Self::generate_operation_handles(),
            block_metrics: BlockMetrics::new_with_labels(&[] as &[(&str, &str)]),
        }
    }

    /// Generate metric handles for all operation and context combinations.
    fn generate_operation_handles() -> HashMap<StorageOperation, OperationMetrics> {
        let mut operations =
            HashMap::with_capacity_and_hasher(StorageOperation::COUNT, Default::default());
        for operation in StorageOperation::iter() {
            operations.insert(
                operation,
                OperationMetrics::new_with_labels(&[("operation", operation.as_str())]),
            );
        }
        operations
    }

    /// Record a storage operation with timing.
    pub fn record_operation<R>(&self, operation: StorageOperation, f: impl FnOnce() -> R) -> R {
        if let Some(metrics) = self.operations.get(&operation) {
            metrics.record(f)
        } else {
            f()
        }
    }

    /// Record a storage operation with timing (async version).
    pub async fn record_operation_async<F, R>(&self, operation: StorageOperation, f: F) -> R
    where
        F: Future<Output = R>,
    {
        let start = Instant::now();
        let result = f.await;
        let duration = start.elapsed();

        if let Some(metrics) = self.operations.get(&operation) {
            metrics.record_duration(duration);
        }

        result
    }

    /// Get block metrics for recording high-level timing.
    pub const fn block_metrics(&self) -> &BlockMetrics {
        &self.block_metrics
    }

    /// Record a pre-measured duration for an operation.
    pub fn record_duration(&self, operation: StorageOperation, duration: Duration) {
        if let Some(metrics) = self.operations.get(&operation) {
            metrics.record_duration(duration);
        }
    }

    /// Record multiple items with the same duration.
    pub fn record_duration_per_item(
        &self,
        operation: StorageOperation,
        duration: Duration,
        count: usize,
    ) {
        if let Some(metrics) = self.operations.get(&operation) {
            metrics.record_duration_per_item(duration, count);
        }
    }
}

impl Default for StorageMetrics {
    fn default() -> Self {
        Self::new()
    }
}

/// Metrics for individual storage operations.
#[derive(Metrics, Clone)]
#[metrics(scope = "external_proofs.storage.operation")]
struct OperationMetrics {
    /// Duration of storage operations in seconds
    duration_seconds: Histogram,
}

impl OperationMetrics {
    /// Record an operation with timing.
    fn record<R>(&self, f: impl FnOnce() -> R) -> R {
        let start = Instant::now();
        let result = f();
        self.duration_seconds.record(start.elapsed());
        result
    }

    /// Record a pre-measured duration.
    fn record_duration(&self, duration: Duration) {
        self.duration_seconds.record(duration);
    }

    fn record_duration_per_item(&self, duration: Duration, count_usize: usize) {
        if count_usize > 0 &&
            let Some(count) = u32::try_from(count_usize).ok()
        {
            self.duration_seconds.record_many(duration / count, count as usize);
        }
    }
}

/// High-level block processing metrics.
#[derive(Metrics, Clone)]
#[metrics(scope = "external_proofs.block")]
pub struct BlockMetrics {
    /// Total time to process a block (end-to-end) in seconds
    pub total_duration_seconds: Histogram,
    /// Time spent executing the block (EVM) in seconds
    pub execution_duration_seconds: Histogram,
    /// Time spent calculating state root in seconds
    pub state_root_duration_seconds: Histogram,
    /// Time spent writing trie updates to storage in seconds
    pub write_duration_seconds: Histogram,
    /// Number of trie updates written
    pub account_trie_updates_written_total: Counter,
    /// Number of storage trie updates written
    pub storage_trie_updates_written_total: Counter,
    /// Number of hashed accounts written
    pub hashed_accounts_written_total: Counter,
    /// Number of hashed storages written
    pub hashed_storages_written_total: Counter,
}

/// Wrapper around [`OpProofsStorage`] that records metrics for all operations.
#[derive(Debug, Clone)]
pub struct OpProofsStorageWithMetrics<S> {
    storage: S,
    metrics: Arc<StorageMetrics>,
}

impl<S> OpProofsStorageWithMetrics<S> {
    /// Create a new storage wrapper with metrics.
    pub const fn new(storage: S, metrics: Arc<StorageMetrics>) -> Self {
        Self { storage, metrics }
    }

    /// Get the underlying storage.
    pub const fn inner(&self) -> &S {
        &self.storage
    }

    /// Get the metrics.
    pub const fn metrics(&self) -> &Arc<StorageMetrics> {
        &self.metrics
    }
}

/// Wrapper for [`OpProofsTrieCursor`] that records metrics.
#[derive(Debug)]
pub struct TrieCursorWithMetrics<C> {
    cursor: C,
    metrics: Arc<StorageMetrics>,
}

impl<C> TrieCursorWithMetrics<C> {
    /// Create a new cursor wrapper with metrics.
    pub const fn new(cursor: C, metrics: Arc<StorageMetrics>) -> Self {
        Self { cursor, metrics }
    }
}

impl<C: OpProofsTrieCursor> OpProofsTrieCursor for TrieCursorWithMetrics<C> {
    fn seek_exact(
        &mut self,
        path: Nibbles,
    ) -> OpProofsStorageResult<Option<(Nibbles, BranchNodeCompact)>> {
        self.metrics.record_operation(StorageOperation::TrieCursorSeekExact, || {
            self.cursor.seek_exact(path)
        })
    }

    fn seek(
        &mut self,
        path: Nibbles,
    ) -> OpProofsStorageResult<Option<(Nibbles, BranchNodeCompact)>> {
        self.metrics.record_operation(StorageOperation::TrieCursorSeek, || self.cursor.seek(path))
    }

    fn next(&mut self) -> OpProofsStorageResult<Option<(Nibbles, BranchNodeCompact)>> {
        self.metrics.record_operation(StorageOperation::TrieCursorNext, || self.cursor.next())
    }

    fn current(&mut self) -> OpProofsStorageResult<Option<Nibbles>> {
        self.metrics.record_operation(StorageOperation::TrieCursorCurrent, || self.cursor.current())
    }
}

/// Wrapper for [`OpProofsHashedCursor`] that records metrics.
#[derive(Debug)]
pub struct HashedCursorWithMetrics<C> {
    cursor: C,
    metrics: Arc<StorageMetrics>,
}

impl<C> HashedCursorWithMetrics<C> {
    /// Create a new cursor wrapper with metrics.
    pub const fn new(cursor: C, metrics: Arc<StorageMetrics>) -> Self {
        Self { cursor, metrics }
    }
}

impl<C: OpProofsHashedCursor> OpProofsHashedCursor for HashedCursorWithMetrics<C> {
    type Value = C::Value;

    fn seek(&mut self, key: B256) -> OpProofsStorageResult<Option<(B256, Self::Value)>> {
        self.metrics.record_operation(StorageOperation::HashedCursorSeek, || self.cursor.seek(key))
    }

    fn next(&mut self) -> OpProofsStorageResult<Option<(B256, Self::Value)>> {
        self.metrics.record_operation(StorageOperation::HashedCursorNext, || self.cursor.next())
    }
}

impl<S> OpProofsStorage for OpProofsStorageWithMetrics<S>
where
    S: OpProofsStorage,
{
    type StorageTrieCursor = TrieCursorWithMetrics<S::StorageTrieCursor>;
    type AccountTrieCursor = TrieCursorWithMetrics<S::AccountTrieCursor>;
    type StorageCursor = HashedCursorWithMetrics<S::StorageCursor>;
    type AccountHashedCursor = HashedCursorWithMetrics<S::AccountHashedCursor>;

    async fn store_account_branches(
        &self,
        account_nodes: Vec<(Nibbles, Option<BranchNodeCompact>)>,
    ) -> OpProofsStorageResult<()> {
        let count = account_nodes.len();
        let start = Instant::now();
        let result = self.storage.store_account_branches(account_nodes).await;
        let duration = start.elapsed();

        // Record per-item duration
        if count > 0 {
            self.metrics.record_duration_per_item(
                StorageOperation::StoreAccountBranch,
                duration,
                count,
            );
        }

        result
    }

    async fn store_storage_branches(
        &self,
        hashed_address: B256,
        storage_nodes: Vec<(Nibbles, Option<BranchNodeCompact>)>,
    ) -> OpProofsStorageResult<()> {
        let count = storage_nodes.len();
        let start = Instant::now();
        let result = self.storage.store_storage_branches(hashed_address, storage_nodes).await;
        let duration = start.elapsed();

        // Record per-item duration
        if count > 0 {
            self.metrics.record_duration_per_item(
                StorageOperation::StoreStorageBranch,
                duration,
                count,
            );
        }

        result
    }

    async fn store_hashed_accounts(
        &self,
        accounts: Vec<(B256, Option<Account>)>,
    ) -> OpProofsStorageResult<()> {
        let count = accounts.len();
        let start = Instant::now();
        let result = self.storage.store_hashed_accounts(accounts).await;
        let duration = start.elapsed();

        // Record per-item duration
        if count > 0 {
            self.metrics.record_duration_per_item(
                StorageOperation::StoreHashedAccount,
                duration,
                count,
            );
        }

        result
    }

    async fn store_hashed_storages(
        &self,
        hashed_address: B256,
        storages: Vec<(B256, U256)>,
    ) -> OpProofsStorageResult<()> {
        let count = storages.len();
        let start = Instant::now();
        let result = self.storage.store_hashed_storages(hashed_address, storages).await;
        let duration = start.elapsed();

        // Record per-item duration
        if count > 0 {
            self.metrics.record_duration_per_item(
                StorageOperation::StoreHashedStorage,
                duration,
                count,
            );
        }

        result
    }

    async fn get_earliest_block_number(&self) -> OpProofsStorageResult<Option<(u64, B256)>> {
        self.storage.get_earliest_block_number().await
    }

    async fn get_latest_block_number(&self) -> OpProofsStorageResult<Option<(u64, B256)>> {
        self.storage.get_latest_block_number().await
    }

    fn storage_trie_cursor(
        &self,
        hashed_address: B256,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::StorageTrieCursor> {
        let cursor = self.storage.storage_trie_cursor(hashed_address, max_block_number)?;
        Ok(TrieCursorWithMetrics::new(cursor, self.metrics.clone()))
    }

    fn account_trie_cursor(
        &self,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::AccountTrieCursor> {
        let cursor = self.storage.account_trie_cursor(max_block_number)?;
        Ok(TrieCursorWithMetrics::new(cursor, self.metrics.clone()))
    }

    fn storage_hashed_cursor(
        &self,
        hashed_address: B256,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::StorageCursor> {
        let cursor = self.storage.storage_hashed_cursor(hashed_address, max_block_number)?;
        Ok(HashedCursorWithMetrics::new(cursor, self.metrics.clone()))
    }

    fn account_hashed_cursor(
        &self,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::AccountHashedCursor> {
        let cursor = self.storage.account_hashed_cursor(max_block_number)?;
        Ok(HashedCursorWithMetrics::new(cursor, self.metrics.clone()))
    }

    // no metrics for these
    async fn store_trie_updates(
        &self,
        block_number: u64,
        block_state_diff: BlockStateDiff,
    ) -> OpProofsStorageResult<()> {
        self.storage.store_trie_updates(block_number, block_state_diff).await
    }

    async fn fetch_trie_updates(&self, block_number: u64) -> OpProofsStorageResult<BlockStateDiff> {
        self.storage.fetch_trie_updates(block_number).await
    }

    async fn prune_earliest_state(
        &self,
        new_earliest_block_number: u64,
        diff: BlockStateDiff,
    ) -> OpProofsStorageResult<()> {
        self.storage.prune_earliest_state(new_earliest_block_number, diff).await
    }

    async fn replace_updates(
        &self,
        latest_common_block_number: u64,
        blocks_to_add: HashMap<u64, BlockStateDiff>,
    ) -> OpProofsStorageResult<()> {
        self.storage.replace_updates(latest_common_block_number, blocks_to_add).await
    }

    async fn set_earliest_block_number(
        &self,
        block_number: u64,
        hash: B256,
    ) -> OpProofsStorageResult<()> {
        self.storage.set_earliest_block_number(block_number, hash).await
    }
}

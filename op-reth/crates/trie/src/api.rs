//! Storage API for external storage of intermediary trie nodes.

use crate::{
    db::{HashedStorageKey, StorageTrieKey},
    OpProofsStorageResult,
};
use alloy_eips::{eip1898::BlockWithParent, BlockNumHash};
use alloy_primitives::{map::HashMap, B256, U256};
use auto_impl::auto_impl;
use derive_more::{AddAssign, Constructor};
use reth_primitives_traits::Account;
use reth_trie::{
    hashed_cursor::{HashedCursor, HashedStorageCursor},
    trie_cursor::{TrieCursor, TrieStorageCursor},
};
use reth_trie_common::{
    updates::TrieUpdatesSorted, BranchNodeCompact, HashedPostStateSorted, Nibbles, StoredNibbles,
};
use std::{fmt::Debug, time::Duration};

/// Diff of trie updates and post state for a block.
#[derive(Debug, Clone, Default)]
pub struct BlockStateDiff {
    /// Trie updates for branch nodes
    pub sorted_trie_updates: TrieUpdatesSorted,
    /// Post state for leaf nodes (accounts and storage)
    pub sorted_post_state: HashedPostStateSorted,
}

impl BlockStateDiff {
    /// Extend the [` BlockStateDiff`] from other latest [`BlockStateDiff`]
    pub fn extend_ref(&mut self, other: &Self) {
        self.sorted_trie_updates.extend_ref_and_sort(&other.sorted_trie_updates);
        self.sorted_post_state.extend_ref_and_sort(&other.sorted_post_state);
    }
}

/// Counts of trie updates written to storage.
#[derive(Debug, Clone, Default, AddAssign, Constructor, Eq, PartialEq)]
pub struct WriteCounts {
    /// Number of account trie updates written
    pub account_trie_updates_written_total: u64,
    /// Number of storage trie updates written
    pub storage_trie_updates_written_total: u64,
    /// Number of hashed accounts written
    pub hashed_accounts_written_total: u64,
    /// Number of hashed storages written
    pub hashed_storages_written_total: u64,
}

/// Duration metrics for block processing.
#[derive(Debug, Default, Clone)]
pub struct OperationDurations {
    /// Total time to process a block (end-to-end) in seconds
    pub total_duration_seconds: Duration,
    /// Time spent executing the block (EVM) in seconds
    pub execution_duration_seconds: Duration,
    /// Time spent calculating state root in seconds
    pub state_root_duration_seconds: Duration,
    /// Time spent writing trie updates to storage in seconds
    pub write_duration_seconds: Duration,
}

/// Trait for reading trie nodes from the database.
///
/// Only leaf nodes and some branch nodes are stored. The bottom layer of branch nodes
/// are not stored to reduce write amplification. This matches Reth's non-historical trie storage.
#[auto_impl(Arc)]
pub trait OpProofsStore: Send + Sync + Debug {
    /// Cursor for iterating over trie branches.
    type StorageTrieCursor<'tx>: TrieStorageCursor + 'tx
    where
        Self: 'tx;

    /// Cursor for iterating over account trie branches.
    type AccountTrieCursor<'tx>: TrieCursor + 'tx
    where
        Self: 'tx;

    /// Cursor for iterating over storage leaves.
    type StorageCursor<'tx>: HashedStorageCursor<Value = U256> + Send + Sync + 'tx
    where
        Self: 'tx;

    /// Cursor for iterating over account leaves.
    type AccountHashedCursor<'tx>: HashedCursor<Value = Account> + Send + Sync + 'tx
    where
        Self: 'tx;

    /// Store a batch of account trie branches. Used for saving existing state. For live state
    /// capture, use [store_trie_updates](OpProofsStore::store_trie_updates).
    fn store_account_branches(
        &self,
        account_nodes: Vec<(Nibbles, Option<BranchNodeCompact>)>,
    ) -> impl Future<Output = OpProofsStorageResult<()>> + Send;

    /// Store a batch of storage trie branches. Used for saving existing state.
    fn store_storage_branches(
        &self,
        hashed_address: B256,
        storage_nodes: Vec<(Nibbles, Option<BranchNodeCompact>)>,
    ) -> impl Future<Output = OpProofsStorageResult<()>> + Send;

    /// Store a batch of account trie leaf nodes. Used for saving existing state.
    fn store_hashed_accounts(
        &self,
        accounts: Vec<(B256, Option<Account>)>,
    ) -> impl Future<Output = OpProofsStorageResult<()>> + Send;

    /// Store a batch of storage trie leaf nodes. Used for saving existing state.
    fn store_hashed_storages(
        &self,
        hashed_address: B256,
        storages: Vec<(B256, U256)>,
    ) -> impl Future<Output = OpProofsStorageResult<()>> + Send;

    /// Get the earliest block number and hash that has been stored
    ///
    /// This is used to determine the block number of trie nodes with block number 0.
    /// All earliest block numbers are stored in 0 to reduce updates required to prune trie nodes.
    fn get_earliest_block_number(
        &self,
    ) -> impl Future<Output = OpProofsStorageResult<Option<(u64, B256)>>> + Send;

    /// Get the latest block number and hash that has been stored
    fn get_latest_block_number(
        &self,
    ) -> impl Future<Output = OpProofsStorageResult<Option<(u64, B256)>>> + Send;

    /// Get a trie cursor for the storage backend
    fn storage_trie_cursor<'tx>(
        &self,
        hashed_address: B256,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::StorageTrieCursor<'tx>>;

    /// Get a trie cursor for the account backend
    fn account_trie_cursor<'tx>(
        &self,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::AccountTrieCursor<'tx>>;

    /// Get a storage cursor for the storage backend
    fn storage_hashed_cursor<'tx>(
        &self,
        hashed_address: B256,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::StorageCursor<'tx>>;

    /// Get an account hashed cursor for the storage backend
    fn account_hashed_cursor<'tx>(
        &self,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::AccountHashedCursor<'tx>>;

    /// Store a batch of trie updates.
    ///
    /// If wiped is true, the entire storage trie is wiped, but this is unsupported going forward,
    /// so should only happen for legacy reasons.
    fn store_trie_updates(
        &self,
        block_ref: BlockWithParent,
        block_state_diff: BlockStateDiff,
    ) -> impl Future<Output = OpProofsStorageResult<WriteCounts>> + Send;

    /// Fetch all updates for a given block number.
    fn fetch_trie_updates(
        &self,
        block_number: u64,
    ) -> impl Future<Output = OpProofsStorageResult<BlockStateDiff>> + Send;

    /// Applies [`BlockStateDiff`] to the earliest state (updating/deleting nodes) and updates the
    /// earliest block number.
    fn prune_earliest_state(
        &self,
        new_earliest_block_ref: BlockWithParent,
    ) -> impl Future<Output = OpProofsStorageResult<WriteCounts>> + Send;

    /// Remove account, storage and trie updates from historical storage for all blocks till
    /// the specified block (inclusive).
    fn unwind_history(
        &self,
        to: BlockWithParent,
    ) -> impl Future<Output = OpProofsStorageResult<()>> + Send;

    /// Deletes all updates > `latest_common_block_number` and replaces them with the new updates.
    fn replace_updates(
        &self,
        latest_common_block_number: u64,
        blocks_to_add: HashMap<BlockWithParent, BlockStateDiff>,
    ) -> impl Future<Output = OpProofsStorageResult<()>> + Send;

    /// Set the earliest block number and hash that has been stored
    fn set_earliest_block_number(
        &self,
        block_number: u64,
        hash: B256,
    ) -> impl Future<Output = OpProofsStorageResult<()>> + Send;
}

/// Status of the initial state anchor.
#[derive(Debug, Clone, Copy, Default)]
pub enum InitialStateStatus {
    /// Init isn't yet started
    #[default]
    NotStarted,
    /// Init is in progress (some tables may already be populated)
    InProgress,
    /// Init completed successfully (all tables done + earliest block set)
    Completed,
}

/// Anchor for the initial state.
#[derive(Debug, Clone, Default)]
pub struct InitialStateAnchor {
    /// The block for which the initial state is being initialized. None if initialization is not
    /// yet started.
    pub block: Option<BlockNumHash>,
    /// Whether initialization is still running or completed.
    pub status: InitialStateStatus,
    /// The latest key stored for `AccountTrieHistory`.
    pub latest_account_trie_key: Option<StoredNibbles>,
    /// The latest key stored for `StorageTrieHistory`.
    pub latest_storage_trie_key: Option<StorageTrieKey>,
    /// The latest key stored for `HashedAccountHistory`.
    pub latest_hashed_account_key: Option<B256>,
    /// The latest key stored for `HashedStorageHistory`.
    pub latest_hashed_storage_key: Option<HashedStorageKey>,
}

/// Trait for storing and retrieving the initial state anchor.
#[auto_impl(Arc)]
pub trait OpProofsInitialStateStore: Send + Sync + Debug {
    /// Read the current anchor.
    fn initial_state_anchor(
        &self,
    ) -> impl Future<Output = OpProofsStorageResult<InitialStateAnchor>> + Send;

    /// Create the anchor if it doesn't exist.
    /// Returns `Err` if an anchor already exists (prevents accidental overwrite).
    fn set_initial_state_anchor(
        &self,
        anchor: BlockNumHash,
    ) -> impl Future<Output = OpProofsStorageResult<()>> + Send;

    /// Commit the initial state - mark the anchor as completed and also set the earliest block
    /// number to anchor.
    fn commit_initial_state(
        &self,
    ) -> impl Future<Output = OpProofsStorageResult<BlockNumHash>> + Send;
}

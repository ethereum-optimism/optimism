//! Storage API for external storage of intermediary trie nodes.

use crate::{
    OpProofsStorageResult,
    db::{HashedStorageKey, StorageTrieKey},
};
use alloy_eips::{BlockNumHash, NumHash, eip1898::BlockWithParent};
use alloy_primitives::{B256, U256};
use auto_impl::auto_impl;
use derive_more::{AddAssign, Constructor};
use reth_primitives_traits::Account;
use reth_trie::{
    hashed_cursor::{HashedCursor, HashedStorageCursor},
    trie_cursor::{TrieCursor, TrieStorageCursor},
};
use reth_trie_common::{
    BranchNodeCompact, HashedPostStateSorted, Nibbles, StoredNibbles, updates::TrieUpdatesSorted,
};
use std::{fmt::Debug, time::Duration};

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

/// The block range covered by the proof window: earliest persisted block and latest persisted
/// block. Both endpoints are inclusive.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct ProofWindowRange {
    /// Earliest block stored.
    pub earliest: NumHash,
    /// Latest block stored.
    pub latest: NumHash,
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

/// Provider for interacting with the proofs storage within a transaction.
#[auto_impl(Arc)]
pub trait OpProofsProviderRO: Send + Sync + Debug {
    /// Cursor for iterating over trie branches.
    type StorageTrieCursor<'tx>: TrieStorageCursor + 'tx
    where
        Self: 'tx;

    /// Cursor for iterating over account trie branches.
    type AccountTrieCursor<'tx>: TrieCursor + 'tx
    where
        Self: 'tx;

    /// Cursor for iterating over storage leaves.
    type StorageCursor<'tx>: HashedStorageCursor<Value = U256> + Send + 'tx
    where
        Self: 'tx;

    /// Cursor for iterating over account leaves.
    type AccountHashedCursor<'tx>: HashedCursor<Value = Account> + Send + 'tx
    where
        Self: 'tx;

    /// Get the earliest block number and hash that has been stored. Returns
    /// [`crate::OpProofsStorageError::NoBlocksFound`] if the proof window is empty.
    fn get_earliest_block(&self) -> OpProofsStorageResult<NumHash>;

    /// Get the latest block number and hash that has been stored. Returns
    /// [`crate::OpProofsStorageError::NoBlocksFound`] if the proof window is empty.
    fn get_latest_block(&self) -> OpProofsStorageResult<NumHash>;

    /// Get the proof window range — earliest and latest persisted blocks — in a single read.
    /// Prefer this over calling [`Self::get_earliest_block`] and [`Self::get_latest_block`]
    /// separately when you need both. Returns [`crate::OpProofsStorageError::NoBlocksFound`] if the
    /// proof window is empty.
    fn get_proof_window(&self) -> OpProofsStorageResult<ProofWindowRange>;

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

    /// Fetch all updates for a given block number.
    fn fetch_trie_updates(&self, block_number: u64) -> OpProofsStorageResult<BlockStateDiff>;
}

/// Blanket [`OpProofsProviderRO`] for shared references.
impl<'a, T: OpProofsProviderRO + 'a> OpProofsProviderRO for &'a T {
    type StorageTrieCursor<'tx>
        = T::StorageTrieCursor<'tx>
    where
        Self: 'tx,
        T: 'tx;
    type AccountTrieCursor<'tx>
        = T::AccountTrieCursor<'tx>
    where
        Self: 'tx,
        T: 'tx;
    type StorageCursor<'tx>
        = T::StorageCursor<'tx>
    where
        Self: 'tx,
        T: 'tx;
    type AccountHashedCursor<'tx>
        = T::AccountHashedCursor<'tx>
    where
        Self: 'tx,
        T: 'tx;

    fn get_earliest_block(&self) -> OpProofsStorageResult<NumHash> {
        T::get_earliest_block(self)
    }

    fn get_latest_block(&self) -> OpProofsStorageResult<NumHash> {
        T::get_latest_block(self)
    }

    fn get_proof_window(&self) -> OpProofsStorageResult<ProofWindowRange> {
        T::get_proof_window(self)
    }

    fn storage_trie_cursor<'tx>(
        &self,
        hashed_address: B256,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::StorageTrieCursor<'tx>>
    where
        'a: 'tx,
    {
        T::storage_trie_cursor(self, hashed_address, max_block_number)
    }

    fn account_trie_cursor<'tx>(
        &self,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::AccountTrieCursor<'tx>>
    where
        'a: 'tx,
    {
        T::account_trie_cursor(self, max_block_number)
    }

    fn storage_hashed_cursor<'tx>(
        &self,
        hashed_address: B256,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::StorageCursor<'tx>>
    where
        'a: 'tx,
    {
        T::storage_hashed_cursor(self, hashed_address, max_block_number)
    }

    fn account_hashed_cursor<'tx>(
        &self,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::AccountHashedCursor<'tx>>
    where
        'a: 'tx,
    {
        T::account_hashed_cursor(self, max_block_number)
    }

    fn fetch_trie_updates(&self, block_number: u64) -> OpProofsStorageResult<BlockStateDiff> {
        T::fetch_trie_updates(self, block_number)
    }
}

/// Provider for writing to the proofs storage within a transaction.
pub trait OpProofsProviderRw: OpProofsProviderRO {
    /// Store trie updates for a block.
    fn store_trie_updates(
        &self,
        block_ref: BlockWithParent,
        block_state_diff: BlockStateDiff,
    ) -> OpProofsStorageResult<WriteCounts>;

    /// Store a batch of trie updates for a block.
    fn store_trie_updates_batch(
        &self,
        updates: Vec<(BlockWithParent, BlockStateDiff)>,
    ) -> OpProofsStorageResult<WriteCounts>;

    /// Applies [`BlockStateDiff`] to the earliest state (updating/deleting nodes) and updates the
    /// earliest block number.
    fn prune_earliest_state(
        &self,
        new_earliest_block_ref: BlockWithParent,
    ) -> OpProofsStorageResult<WriteCounts>;

    /// Remove account, storage and trie updates from historical storage for all blocks till
    /// the specified block (inclusive).
    fn unwind_history(&self, to: BlockWithParent) -> OpProofsStorageResult<()>;

    /// Deletes all updates > `latest_common_block` and replaces them with the new updates.
    fn replace_updates(
        &self,
        latest_common_block: BlockNumHash,
        blocks_to_add: Vec<(BlockWithParent, BlockStateDiff)>,
    ) -> OpProofsStorageResult<()>;

    /// Commit the changes to the database.
    /// Consumes the provider.
    fn commit(self) -> OpProofsStorageResult<()>;
}

/// Provider for writing historical records for blocks older than the current window boundary,
/// and for the snapshot RW operations that ride on the same transaction.
///
/// Unlike [`OpProofsProviderRw::store_trie_updates`], which is strictly append-only (validates
/// parent hash against `latest` and advances `latest`), this provider is designed for
/// **prepend-style** writes that extend the window backward.  It does not touch the `latest`
/// marker, and it does not enforce parent-hash ordering against `latest`.
///
/// The typical call sequence for one snapshot-accelerated backfill step is:
/// ```ignore
/// let bp = storage.backfill_provider()?;
/// bp.update_snapshot(new_anchor, &trie_updates)?;
/// bp.prepend_block(block_ref, diff)?;
/// bp.commit()?;   // commits backfill + snapshot writes atomically
/// ```
pub trait OpProofsBackfillProvider: OpProofsSnapshotProviderRO + OpProofsProviderRO {
    /// Write historical changeset and history-bitmap entries for `block_ref`, and move the
    /// `earliest` marker to `block_ref.parent`.
    ///
    /// `diff` contains:
    /// - `sorted_trie_updates`: trie node **before-values** for `block_ref.block.number` (i.e. what
    ///   each changed node looked like *before* the block executed).
    /// - `sorted_post_state`: account / storage **before-values** for the same block.
    ///
    /// The implementation must **not** update the `latest` marker and must **not**
    /// validate `diff` against the current `latest` block.
    fn prepend_block(
        &self,
        block_ref: BlockWithParent,
        diff: BlockStateDiff,
    ) -> OpProofsStorageResult<WriteCounts>;

    /// Wipe both snapshot tables and the meta row.
    fn clear_snapshot(&self) -> OpProofsStorageResult<()>;

    /// Project `trie_updates` onto the snapshot and advance the anchor to `new_anchor`
    /// atomically. Direction is implicit in the diff: `(path, Some(node))` sets,
    /// `(path, None)` deletes.
    fn update_snapshot(
        &self,
        new_anchor: BlockNumHash,
        trie_updates: &TrieUpdatesSorted,
    ) -> OpProofsStorageResult<u64>;

    /// Commit the transaction. Consumes the provider. Flushes both the backfill writes
    /// and any pending snapshot writes atomically.
    fn commit(self) -> OpProofsStorageResult<()>;
}

/// Factory trait for creating providers to interact with the proofs storage.
#[auto_impl(Arc)]
pub trait OpProofsStore: Send + Sync + Debug {
    /// The read-only provider type created by the factory.
    type ProviderRO<'a>: OpProofsProviderRO + Clone + 'a
    where
        Self: 'a;

    /// The read-write provider type created by the factory.
    type ProviderRw<'a>: OpProofsProviderRw + 'a
    where
        Self: 'a;

    /// The initialization provider type created by the factory.
    type Initializer<'a>: OpProofsInitProvider + 'a
    where
        Self: 'a;

    /// Create a read-only provider for interacting with the proofs storage.
    fn provider_ro<'a>(&'a self) -> OpProofsStorageResult<Self::ProviderRO<'a>>;

    /// Create a read-write provider for interacting with the proofs storage.
    fn provider_rw<'a>(&'a self) -> OpProofsStorageResult<Self::ProviderRw<'a>>;

    /// Create an initialization provider for interacting with the proofs storage.
    fn initialization_provider<'a>(&'a self) -> OpProofsStorageResult<Self::Initializer<'a>>;
}

/// Factory extension for stores that support backfill — extending the `earliest` block of
/// the proof window backward.
///
/// Bundles the trie-state snapshot machinery (snapshot RO / RW / init providers) on the
/// same trait because snapshot is internal infrastructure that accelerates backfill.
#[auto_impl(Arc)]
pub trait OpProofsBackfillStore: OpProofsStore {
    /// The backfill provider type created by the factory.
    type BackfillProvider<'a>: OpProofsBackfillProvider + 'a
    where
        Self: 'a;

    /// The snapshot RO provider type created by the factory.
    type SnapshotProviderRO<'a>: OpProofsSnapshotProviderRO + Clone + 'a
    where
        Self: 'a;

    /// Init-time bulk writer (used by [`crate::snapshot::SnapshotInitJob`]).
    type SnapshotInitializer<'a>: OpProofsSnapshotInitProvider + 'a
    where
        Self: 'a;

    /// Open the writer provider for backfill and snapshot RW operations. Backfill writes,
    /// snapshot updates, and snapshot teardown all share this single tx and commit
    /// atomically through [`OpProofsBackfillProvider::commit`].
    fn backfill_provider<'a>(&'a self) -> OpProofsStorageResult<Self::BackfillProvider<'a>>;

    /// Open a RO snapshot provider.
    fn snapshot_provider_ro<'a>(&'a self) -> OpProofsStorageResult<Self::SnapshotProviderRO<'a>>;

    /// Open an init-time snapshot provider.
    fn snapshot_initialization_provider<'a>(
        &'a self,
    ) -> OpProofsStorageResult<Self::SnapshotInitializer<'a>>;
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
pub trait OpProofsInitProvider: Send + Sync + Debug {
    /// Read the current anchor.
    fn initial_state_anchor(&self) -> OpProofsStorageResult<InitialStateAnchor>;

    /// Create the anchor if it doesn't exist.
    /// Returns `Err` if an anchor already exists (prevents accidental overwrite).
    fn set_initial_state_anchor(&self, anchor: BlockNumHash) -> OpProofsStorageResult<()>;

    /// Store a batch of account trie branches. Used for saving existing state. For live state
    /// capture, use [`store_trie_updates`](OpProofsProviderRw::store_trie_updates).
    fn store_account_branches(
        &self,
        account_nodes: Vec<(Nibbles, Option<BranchNodeCompact>)>,
    ) -> OpProofsStorageResult<()>;

    /// Store a batch of storage trie branches. Used for saving existing state.
    fn store_storage_branches(
        &self,
        hashed_address: B256,
        storage_nodes: Vec<(Nibbles, Option<BranchNodeCompact>)>,
    ) -> OpProofsStorageResult<()>;

    /// Store a batch of account trie leaf nodes. Used for saving existing state.
    fn store_hashed_accounts(
        &self,
        accounts: Vec<(B256, Option<Account>)>,
    ) -> OpProofsStorageResult<()>;

    /// Store a batch of storage trie leaf nodes. Used for saving existing state.
    fn store_hashed_storages(
        &self,
        hashed_address: B256,
        storages: Vec<(B256, U256)>,
    ) -> OpProofsStorageResult<()>;

    /// Commit the initial state - mark the anchor as completed and also set the earliest block
    /// number to anchor.
    fn commit_initial_state(&self) -> OpProofsStorageResult<BlockNumHash>;

    /// Commit the changes to the database.
    /// Consumes the provider.
    fn commit(self) -> OpProofsStorageResult<()>;
}

/// Read access to the trie-state snapshot.
///
/// Cursors read the snapshot tables directly (no history-bitmap lookups) and
/// are only valid at [`Self::snapshot_anchor`].
#[auto_impl(Arc)]
pub trait OpProofsSnapshotProviderRO: OpProofsProviderRO {
    /// Cursor over the snapshot's account trie table.
    type SnapshotAccountTrieCursor<'tx>: TrieCursor + 'tx
    where
        Self: 'tx;

    /// Cursor over the snapshot's storage trie table.
    type SnapshotStorageTrieCursor<'tx>: TrieStorageCursor + 'tx
    where
        Self: 'tx;

    /// Anchor block of a `Ready` snapshot. Errors with
    /// [`OpProofsStorageError::SnapshotNotReady`](crate::OpProofsStorageError::SnapshotNotReady)
    /// otherwise.
    fn snapshot_anchor(&self) -> OpProofsStorageResult<BlockNumHash>;

    /// Open a cursor over the snapshot's account trie table.
    fn snapshot_account_trie_cursor<'tx>(
        &self,
    ) -> OpProofsStorageResult<Self::SnapshotAccountTrieCursor<'tx>>;

    /// Open a cursor over the snapshot's storage trie table for `hashed_address`.
    fn snapshot_storage_trie_cursor<'tx>(
        &self,
        hashed_address: B256,
    ) -> OpProofsStorageResult<Self::SnapshotStorageTrieCursor<'tx>>;
}

/// Blanket [`OpProofsSnapshotProviderRO`] for shared references — mirrors the
/// equivalent impl on [`OpProofsProviderRO`] above. Lets callers pass `&bp` to
/// owning cursor factories (e.g., [`crate::SnapshotTrieCursorFactory::new`])
/// without wrapping in [`std::sync::Arc`].
impl<'a, T: OpProofsSnapshotProviderRO + 'a> OpProofsSnapshotProviderRO for &'a T {
    type SnapshotAccountTrieCursor<'tx>
        = T::SnapshotAccountTrieCursor<'tx>
    where
        Self: 'tx,
        T: 'tx;
    type SnapshotStorageTrieCursor<'tx>
        = T::SnapshotStorageTrieCursor<'tx>
    where
        Self: 'tx,
        T: 'tx;

    fn snapshot_anchor(&self) -> OpProofsStorageResult<BlockNumHash> {
        T::snapshot_anchor(self)
    }

    fn snapshot_account_trie_cursor<'tx>(
        &self,
    ) -> OpProofsStorageResult<Self::SnapshotAccountTrieCursor<'tx>>
    where
        'a: 'tx,
    {
        T::snapshot_account_trie_cursor(self)
    }

    fn snapshot_storage_trie_cursor<'tx>(
        &self,
        hashed_address: B256,
    ) -> OpProofsStorageResult<Self::SnapshotStorageTrieCursor<'tx>>
    where
        'a: 'tx,
    {
        T::snapshot_storage_trie_cursor(self, hashed_address)
    }
}

/// Lifecycle of the snapshot init job. Mirrors [`InitialStateStatus`].
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub enum SnapshotInitStatus {
    /// Init has never run.
    #[default]
    NotStarted,
    /// Init is running; snapshot tables may be partially populated.
    InProgress,
    /// Init completed. Use [`OpProofsSnapshotProviderRO::snapshot_anchor`] to read.
    Completed,
}

/// Status + anchor block + resume keys for the snapshot init job. Mirrors
/// [`InitialStateAnchor`]'s shape.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct SnapshotInitAnchor {
    /// Anchor block, or `None` if init has never run.
    pub block: Option<BlockNumHash>,
    /// Lifecycle status.
    pub status: SnapshotInitStatus,
    /// Last key in [`crate::db::V2AccountsTrieSnapshot`]; resumes the account-trie phase.
    pub last_account_trie_key: Option<StoredNibbles>,
    /// Last entry in [`crate::db::V2StoragesTrieSnapshot`]; resumes the storage-trie phase.
    pub last_storage_trie_key: Option<StorageTrieKey>,
}

/// Init-time read + write surface for the trie-state snapshot. Mirrors
/// [`OpProofsInitProvider`]'s role. Driven by [`crate::snapshot::SnapshotInitJob`]
/// over short chunked rw-tx; meta stays `Building` mid-init so a crash
/// resumes from [`Self::snapshot_init_anchor`].
pub trait OpProofsSnapshotInitProvider: Send + Sync + Debug {
    /// Read status + anchor block + resume keys in one call.
    fn snapshot_init_anchor(&self) -> OpProofsStorageResult<SnapshotInitAnchor>;

    /// Plant the meta row at `anchor` with status `Building`. Errors if a
    /// meta row already exists — call
    /// [`OpProofsBackfillProvider::clear_snapshot`] first to rebuild.
    fn set_snapshot_init_anchor(&self, anchor: BlockNumHash) -> OpProofsStorageResult<()>;

    /// Append a chunk to [`crate::db::V2AccountsTrieSnapshot`]. Entries must
    /// be sorted and strictly greater than the table's current last key.
    fn store_account_trie_snapshot_branches(
        &self,
        entries: Vec<(StoredNibbles, BranchNodeCompact)>,
    ) -> OpProofsStorageResult<()>;

    /// Append a chunk to [`crate::db::V2StoragesTrieSnapshot`]. Entries must
    /// be sorted and strictly greater than the table's current last entry.
    fn store_storage_trie_snapshot_branches(
        &self,
        hashed_address: B256,
        storage_nodes: Vec<(Nibbles, Option<BranchNodeCompact>)>,
    ) -> OpProofsStorageResult<()>;

    /// Transition the meta row from `Building` to `Ready`. Errors if no meta
    /// exists or if it isn't `Building`.
    fn commit_snapshot(&self) -> OpProofsStorageResult<()>;

    /// Commit the transaction. Consumes the provider.
    fn commit(self) -> OpProofsStorageResult<()>;
}

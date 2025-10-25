//! Storage API for external storage of intermediary trie nodes.

use alloy_eips::eip1898::BlockWithParent;
use alloy_primitives::{map::HashMap, B256, U256};
use auto_impl::auto_impl;
use reth_db::DatabaseError;
use reth_primitives_traits::Account;
use reth_trie::{updates::TrieUpdates, BranchNodeCompact, HashedPostState, Nibbles};
use std::fmt::Debug;
use thiserror::Error;

/// Error type for storage operations
#[derive(Debug, Error)]
pub enum OpProofsStorageError {
    /// No blocks found
    #[error("No blocks found")]
    NoBlocksFound,
    /// Parent block number is less than earliest stored block number
    #[error("Parent block number is less than earliest stored block number")]
    UnknownParent,
    /// Block is out of order
    #[error("Block {block_number} is out of order (parent: {parent_block_hash}, latest stored block hash: {latest_block_hash})")]
    OutOfOrder {
        /// The block number being inserted
        block_number: u64,
        /// The parent hash of the block being inserted
        parent_block_hash: B256,
        /// block hash of the latest stored block
        latest_block_hash: B256,
    },
    /// Block update failed since parent state
    #[error("Cannot execute block updates for block {0} without parent state {1} (latest stored block number: {2})")]
    BlockUpdateFailed(u64, u64, u64),
    /// State root mismatch
    #[error("State root mismatch for block {0} (have: {1}, expected: {2})")]
    StateRootMismatch(u64, B256, B256),
    /// Error occurred while interacting with the database.
    #[error(transparent)]
    DatabaseError(#[from] DatabaseError),
    /// Other error
    #[error("Other error: {0}")]
    Other(eyre::Error),
}

impl From<OpProofsStorageError> for DatabaseError {
    fn from(error: OpProofsStorageError) -> Self {
        Self::Other(error.to_string())
    }
}

/// Result type for storage operations
pub type OpProofsStorageResult<T> = Result<T, OpProofsStorageError>;

/// Seeks and iterates over trie nodes in the database by path (lexicographical order)
pub trait OpProofsTrieCursorRO: Send + Sync {
    /// Seek to an exact path, otherwise return None if not found.
    fn seek_exact(
        &mut self,
        path: Nibbles,
    ) -> OpProofsStorageResult<Option<(Nibbles, BranchNodeCompact)>>;

    /// Seek to a path, otherwise return the first path greater than the given path
    /// lexicographically.
    fn seek(
        &mut self,
        path: Nibbles,
    ) -> OpProofsStorageResult<Option<(Nibbles, BranchNodeCompact)>>;

    /// Move the cursor to the next path and return it.
    fn next(&mut self) -> OpProofsStorageResult<Option<(Nibbles, BranchNodeCompact)>>;

    /// Get the current path.
    fn current(&mut self) -> OpProofsStorageResult<Option<Nibbles>>;
}

/// Seeks and iterates over hashed entries in the database by key.
pub trait OpProofsHashedCursorRO: Send + Sync {
    /// Value returned by the cursor.
    type Value: Debug;

    /// Seek an entry greater or equal to the given key and position the cursor there.
    /// Returns the first entry with the key greater or equal to the sought key.
    fn seek(&mut self, key: B256) -> OpProofsStorageResult<Option<(B256, Self::Value)>>;

    /// Move the cursor to the next entry and return it.
    fn next(&mut self) -> OpProofsStorageResult<Option<(B256, Self::Value)>>;

    /// Returns `true` if there are no entries for a given key.
    fn is_storage_empty(&mut self) -> OpProofsStorageResult<bool> {
        Ok(self.seek(B256::ZERO)?.is_none())
    }
}

/// Diff of trie updates and post state for a block.
#[derive(Debug, Clone, Default)]
pub struct BlockStateDiff {
    /// Trie updates for branch nodes
    pub trie_updates: TrieUpdates,
    /// Post state for leaf nodes (accounts and storage)
    pub post_state: HashedPostState,
}

/// Trait for reading trie nodes from the database.
///
/// Only leaf nodes and some branch nodes are stored. The bottom layer of branch nodes
/// are not stored to reduce write amplification. This matches Reth's non-historical trie storage.
#[auto_impl(Arc)]
pub trait OpProofsStore: Send + Sync + Debug {
    /// Cursor for iterating over trie branches.
    type StorageTrieCursor<'tx>: OpProofsTrieCursorRO + 'tx
    where
        Self: 'tx;

    /// Cursor for iterating over account trie branches.
    type AccountTrieCursor<'tx>: OpProofsTrieCursorRO + 'tx
    where
        Self: 'tx;

    /// Cursor for iterating over storage leaves.
    type StorageCursor<'tx>: OpProofsHashedCursorRO<Value = U256> + 'tx
    where
        Self: 'tx;

    /// Cursor for iterating over account leaves.
    type AccountHashedCursor<'tx>: OpProofsHashedCursorRO<Value = Account> + 'tx
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
    ) -> impl Future<Output = OpProofsStorageResult<()>> + Send;

    /// Fetch all updates for a given block number.
    fn fetch_trie_updates(
        &self,
        block_number: u64,
    ) -> impl Future<Output = OpProofsStorageResult<BlockStateDiff>> + Send;

    /// Applies [`BlockStateDiff`] to the earliest state (updating/deleting nodes) and updates the
    /// earliest block number.
    fn prune_earliest_state(
        &self,
        new_earliest_block_number: u64,
        diff: BlockStateDiff,
    ) -> impl Future<Output = OpProofsStorageResult<()>> + Send;

    /// Deletes all updates > `latest_common_block_number` and replaces them with the new updates.
    fn replace_updates(
        &self,
        latest_common_block_number: u64,
        blocks_to_add: HashMap<u64, BlockStateDiff>,
    ) -> impl Future<Output = OpProofsStorageResult<()>> + Send;

    /// Set the earliest block number and hash that has been stored
    fn set_earliest_block_number(
        &self,
        block_number: u64,
        hash: B256,
    ) -> impl Future<Output = OpProofsStorageResult<()>> + Send;
}

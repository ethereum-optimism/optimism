//! Traits for external storage for trie nodes.

use alloy_primitives::{map::HashMap, B256, U256};
use async_trait::async_trait;
use auto_impl::auto_impl;
use reth_primitives_traits::Account;
use reth_trie::{updates::TrieUpdates, BranchNodeCompact, HashedPostState, Nibbles};
use std::fmt::Debug;
use thiserror::Error;

/// Error type for storage operations
#[derive(Debug, Error)]
pub enum ExternalStorageError {
    // TODO: add more errors once we know what they are
    /// Other error
    #[error("Other error: {0}")]
    Other(eyre::Error),
}

/// Result type for storage operations
pub type ExternalStorageResult<T> = Result<T, ExternalStorageError>;

/// Seeks and iterates over trie nodes in the database by path (lexicographical order)
pub trait ExternalTrieCursor: Send + Sync {
    /// Seek to an exact path, otherwise return None if not found.
    fn seek_exact(
        &mut self,
        path: Nibbles,
    ) -> ExternalStorageResult<Option<(Nibbles, BranchNodeCompact)>>;

    /// Seek to a path, otherwise return the first path greater than the given path
    /// lexicographically.
    fn seek(
        &mut self,
        path: Nibbles,
    ) -> ExternalStorageResult<Option<(Nibbles, BranchNodeCompact)>>;

    /// Move the cursor to the next path and return it.
    fn next(&mut self) -> ExternalStorageResult<Option<(Nibbles, BranchNodeCompact)>>;

    /// Get the current path.
    fn current(&mut self) -> ExternalStorageResult<Option<Nibbles>>;
}

/// Seeks and iterates over hashed entries in the database by key.
pub trait ExternalHashedCursor: Send + Sync {
    /// Value returned by the cursor.
    type Value: std::fmt::Debug;

    /// Seek an entry greater or equal to the given key and position the cursor there.
    /// Returns the first entry with the key greater or equal to the sought key.
    fn seek(&mut self, key: B256) -> ExternalStorageResult<Option<(B256, Self::Value)>>;

    /// Move the cursor to the next entry and return it.
    fn next(&mut self) -> ExternalStorageResult<Option<(B256, Self::Value)>>;
}

/// Diff of trie updates and post state for a block.
#[derive(Debug, Clone)]
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
#[async_trait]
#[auto_impl(Arc)]
pub trait ExternalStorage: Send + Sync + Debug {
    /// Cursor for iterating over trie branches.
    type TrieCursor: ExternalTrieCursor;

    /// Cursor for iterating over storage leaves.
    type StorageCursor: ExternalHashedCursor<Value = U256>;

    /// Cursor for iterating over account leaves.
    type AccountHashedCursor: ExternalHashedCursor<Value = Account>;

    /// Store a batch of account trie branches. Used for saving existing state. For live state
    /// capture, use [store_trie_updates](ExternalStorage::store_trie_updates).
    async fn store_account_branches(
        &self,
        block_number: u64,
        updates: Vec<(Nibbles, Option<BranchNodeCompact>)>,
    ) -> ExternalStorageResult<()>;

    /// Store a batch of storage trie branches. Used for saving existing state.
    async fn store_storage_branches(
        &self,
        block_number: u64,
        hashed_address: B256,
        items: Vec<(Nibbles, Option<BranchNodeCompact>)>,
    ) -> ExternalStorageResult<()>;

    /// Store a batch of account trie leaf nodes. Used for saving existing state.
    async fn store_hashed_accounts(
        &self,
        accounts: Vec<(B256, Option<Account>)>,
        block_number: u64,
    ) -> ExternalStorageResult<()>;

    /// Store a batch of storage trie leaf nodes. Used for saving existing state.
    async fn store_hashed_storages(
        &self,
        hashed_address: B256,
        storages: Vec<(B256, U256)>,
        block_number: u64,
    ) -> ExternalStorageResult<()>;

    /// Get the earliest block number and hash that has been stored
    ///
    /// This is used to determine the block number of trie nodes with block number 0.
    /// All earliest block numbers are stored in 0 to reduce updates required to prune trie nodes.
    async fn get_earliest_block_number(&self) -> ExternalStorageResult<Option<(u64, B256)>>;

    /// Get the latest block number and hash that has been stored
    async fn get_latest_block_number(&self) -> ExternalStorageResult<Option<(u64, B256)>>;

    /// Get a trie cursor for the storage backend
    fn trie_cursor(
        &self,
        hashed_address: Option<B256>,
        max_block_number: u64,
    ) -> ExternalStorageResult<Self::TrieCursor>;

    /// Get a storage cursor for the storage backend
    fn storage_hashed_cursor(
        &self,
        hashed_address: B256,
        max_block_number: u64,
    ) -> ExternalStorageResult<Self::StorageCursor>;

    /// Get an account hashed cursor for the storage backend
    fn account_hashed_cursor(
        &self,
        max_block_number: u64,
    ) -> ExternalStorageResult<Self::AccountHashedCursor>;

    /// Store a batch of trie updates.
    ///
    /// If wiped is true, the entire storage trie is wiped, but this is unsupported going forward,
    /// so should only happen for legacy reasons.
    async fn store_trie_updates(
        &self,
        block_number: u64,
        block_state_diff: BlockStateDiff,
    ) -> ExternalStorageResult<()>;

    /// Fetch all updates for a given block number.
    async fn fetch_trie_updates(&self, block_number: u64) -> ExternalStorageResult<BlockStateDiff>;

    /// Applies `BlockStateDiff` to the earliest state (updating/deleting nodes) and updates the
    /// earliest block number.
    async fn prune_earliest_state(
        &self,
        new_earliest_block_number: u64,
        diff: BlockStateDiff,
    ) -> ExternalStorageResult<()>;

    /// Deletes all updates > `latest_common_block_number` and replaces them with the new updates.
    async fn replace_updates(
        &self,
        latest_common_block_number: u64,
        blocks_to_add: HashMap<u64, BlockStateDiff>,
    ) -> ExternalStorageResult<()>;

    /// Set the earliest block number and hash that has been stored
    async fn set_earliest_block_number(
        &self,
        block_number: u64,
        hash: B256,
    ) -> ExternalStorageResult<()>;
}

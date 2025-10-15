use crate::{
    db::{MdbxAccountCursor, MdbxStorageCursor, MdbxTrieCursor},
    BlockStateDiff, OpProofsStorage, OpProofsStorageError, OpProofsStorageResult,
};
use alloy_primitives::{map::HashMap, B256, U256};
use reth_db::{
    mdbx::{init_db_for, DatabaseArguments},
    DatabaseEnv,
};
use reth_primitives_traits::Account;
use reth_trie::{BranchNodeCompact, Nibbles};
use std::path::Path;

/// MDBX implementation of `OpProofsStorage`.
#[derive(Debug)]
pub struct MdbxProofsStorage {
    _env: DatabaseEnv,
}

impl MdbxProofsStorage {
    /// Creates a new `MdbxProofsStorage` instance with the given path.
    pub fn new(path: &Path) -> Result<Self, OpProofsStorageError> {
        let env = init_db_for::<_, super::models::Tables>(path, DatabaseArguments::default())
            .map_err(OpProofsStorageError::Other)?;
        Ok(Self { _env: env })
    }
}

impl OpProofsStorage for MdbxProofsStorage {
    type StorageTrieCursor = MdbxTrieCursor;
    type AccountTrieCursor = MdbxTrieCursor;
    type StorageCursor = MdbxStorageCursor;
    type AccountHashedCursor = MdbxAccountCursor;

    async fn store_account_branches(
        &self,
        _block_number: u64,
        _updates: Vec<(Nibbles, Option<BranchNodeCompact>)>,
    ) -> OpProofsStorageResult<()> {
        unimplemented!()
    }

    async fn store_storage_branches(
        &self,
        _block_number: u64,
        _hashed_address: B256,
        _items: Vec<(Nibbles, Option<BranchNodeCompact>)>,
    ) -> OpProofsStorageResult<()> {
        unimplemented!()
    }

    async fn store_hashed_accounts(
        &self,
        _accounts: Vec<(B256, Option<Account>)>,
        _block_number: u64,
    ) -> OpProofsStorageResult<()> {
        unimplemented!()
    }

    async fn store_hashed_storages(
        &self,
        _hashed_address: B256,
        _storages: Vec<(B256, U256)>,
        _block_number: u64,
    ) -> OpProofsStorageResult<()> {
        unimplemented!()
    }

    async fn get_earliest_block_number(&self) -> OpProofsStorageResult<Option<(u64, B256)>> {
        unimplemented!()
    }

    async fn get_latest_block_number(&self) -> OpProofsStorageResult<Option<(u64, B256)>> {
        unimplemented!()
    }

    fn storage_trie_cursor(
        &self,
        _hashed_address: B256,
        _max_block_number: u64,
    ) -> OpProofsStorageResult<Self::StorageTrieCursor> {
        unimplemented!()
    }

    fn account_trie_cursor(
        &self,
        _max_block_number: u64,
    ) -> OpProofsStorageResult<Self::AccountTrieCursor> {
        unimplemented!()
    }

    fn storage_hashed_cursor(
        &self,
        _hashed_address: B256,
        _max_block_number: u64,
    ) -> OpProofsStorageResult<Self::StorageCursor> {
        unimplemented!()
    }

    fn account_hashed_cursor(
        &self,
        _max_block_number: u64,
    ) -> OpProofsStorageResult<Self::AccountHashedCursor> {
        unimplemented!()
    }

    async fn store_trie_updates(
        &self,
        _block_number: u64,
        _block_state_diff: BlockStateDiff,
    ) -> OpProofsStorageResult<()> {
        unimplemented!()
    }

    async fn fetch_trie_updates(
        &self,
        _block_number: u64,
    ) -> OpProofsStorageResult<BlockStateDiff> {
        unimplemented!()
    }

    async fn prune_earliest_state(
        &self,
        _new_earliest_block_number: u64,
        _diff: BlockStateDiff,
    ) -> OpProofsStorageResult<()> {
        unimplemented!()
    }

    async fn replace_updates(
        &self,
        _latest_common_block_number: u64,
        _blocks_to_add: HashMap<u64, BlockStateDiff>,
    ) -> OpProofsStorageResult<()> {
        unimplemented!()
    }

    async fn set_earliest_block_number(
        &self,
        _block_number: u64,
        _hash: B256,
    ) -> OpProofsStorageResult<()> {
        unimplemented!()
    }
}

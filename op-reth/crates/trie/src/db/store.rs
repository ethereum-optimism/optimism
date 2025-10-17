use crate::{
    db::{
        models::{
            AccountTrieHistory, HashedAccountHistory, HashedStorageHistory, HashedStorageKey,
            MaybeDeleted, StorageTrieHistory, StorageTrieKey, StorageValue, VersionedValue,
        },
        MdbxAccountCursor, MdbxStorageCursor, MdbxTrieCursor,
    },
    BlockStateDiff, OpProofsStorage, OpProofsStorageError, OpProofsStorageResult,
};
use alloy_primitives::{map::HashMap, B256, U256};
use reth_db::{
    cursor::DbDupCursorRW,
    mdbx::{init_db_for, DatabaseArguments},
    Database, DatabaseEnv,
};
use reth_primitives_traits::Account;
use reth_trie::{BranchNodeCompact, Nibbles, StoredNibbles};
use std::path::Path;

/// MDBX implementation of `OpProofsStorage`.
#[derive(Debug)]
pub struct MdbxProofsStorage {
    env: DatabaseEnv,
}

impl MdbxProofsStorage {
    /// Creates a new `MdbxProofsStorage` instance with the given path.
    pub fn new(path: &Path) -> Result<Self, OpProofsStorageError> {
        let env = init_db_for::<_, super::models::Tables>(path, DatabaseArguments::default())
            .map_err(OpProofsStorageError::Other)?;
        Ok(Self { env })
    }
}

impl OpProofsStorage for MdbxProofsStorage {
    type StorageTrieCursor = MdbxTrieCursor;
    type AccountTrieCursor = MdbxTrieCursor;
    type StorageCursor = MdbxStorageCursor;
    type AccountHashedCursor = MdbxAccountCursor;

    async fn store_account_branches(
        &self,
        block_number: u64,
        updates: Vec<(Nibbles, Option<BranchNodeCompact>)>,
    ) -> OpProofsStorageResult<()> {
        let mut updates = updates;
        if updates.is_empty() {
            return Ok(());
        }

        updates.sort_by_key(|(key, _)| *key);

        self.env.update(|tx| {
            let mut cursor = tx.new_cursor::<AccountTrieHistory>()?;
            for (nibble, branch_node) in updates {
                let vv = VersionedValue { block_number, value: MaybeDeleted(branch_node) };
                cursor.append_dup(StoredNibbles::from(nibble), vv)?;
            }
            Ok(())
        })?
    }

    async fn store_storage_branches(
        &self,
        block_number: u64,
        hashed_address: B256,
        items: Vec<(Nibbles, Option<BranchNodeCompact>)>,
    ) -> OpProofsStorageResult<()> {
        let mut items = items;
        if items.is_empty() {
            return Ok(());
        }

        items.sort_by_key(|(key, _)| *key);

        self.env.update(|tx| {
            let mut cursor = tx.new_cursor::<StorageTrieHistory>()?;
            for (nibble, branch_node) in items {
                let key = StorageTrieKey::new(hashed_address, StoredNibbles::from(nibble));
                let vv = VersionedValue { block_number, value: MaybeDeleted(branch_node) };
                cursor.append_dup(key, vv)?;
            }
            Ok(())
        })?
    }

    async fn store_hashed_accounts(
        &self,
        accounts: Vec<(B256, Option<Account>)>,
        block_number: u64,
    ) -> OpProofsStorageResult<()> {
        let mut accounts = accounts;
        if accounts.is_empty() {
            return Ok(());
        }

        // sort the accounts by key to ensure insertion is efficient
        accounts.sort_by_key(|(key, _)| *key);

        self.env.update(|tx| {
            let mut cursor = tx.new_cursor::<HashedAccountHistory>()?;
            for (key, account) in accounts {
                let vv = VersionedValue { block_number, value: MaybeDeleted(account) };
                cursor.append_dup(key, vv)?;
            }
            Ok(())
        })?
    }

    async fn store_hashed_storages(
        &self,
        hashed_address: B256,
        storages: Vec<(B256, U256)>,
        block_number: u64,
    ) -> OpProofsStorageResult<()> {
        let mut storages = storages;
        if storages.is_empty() {
            return Ok(());
        }

        // sort the storages by key to ensure insertion is efficient
        storages.sort_by_key(|(key, _)| *key);

        self.env.update(|tx| {
            let mut cursor = tx.new_cursor::<HashedStorageHistory>()?;
            for (key, value) in storages {
                let vv =
                    VersionedValue { block_number, value: MaybeDeleted(Some(StorageValue(value))) };
                let storage_key = HashedStorageKey::new(hashed_address, key);
                cursor.append_dup(storage_key, vv)?;
            }
            Ok(())
        })?
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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::db::{
        models::{AccountTrieHistory, StorageTrieHistory},
        StorageTrieKey,
    };
    use alloy_primitives::B256;
    use reth_db::{cursor::DbDupCursorRO, transaction::DbTx};
    use reth_trie::{BranchNodeCompact, Nibbles, StoredNibbles};
    use tempfile::TempDir;

    const B0: u64 = 0;

    #[tokio::test]
    async fn store_hashed_accounts_writes_versioned_values() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        let addr = B256::from([0xAA; 32]);
        let account = Account::default();
        store.store_hashed_accounts(vec![(addr, Some(account))], B0).await.expect("write accounts");

        let tx = store.env.tx().expect("ro tx");
        let mut cur = tx.new_cursor::<HashedAccountHistory>().expect("cursor");
        let vv = cur.seek_by_key_subkey(addr, B0).expect("seek");
        let vv = vv.expect("entry exists");

        assert_eq!(vv.block_number, B0);
        assert_eq!(vv.value.0, Some(account));
    }

    #[tokio::test]
    async fn store_hashed_accounts_multiple_items_unsorted() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        // Unsorted input, mixed Some/None (deletion)
        let a1 = B256::from([0x01; 32]);
        let a2 = B256::from([0x02; 32]);
        let a3 = B256::from([0x03; 32]);
        let acc1 = Account { nonce: 2, balance: U256::from(1000u64), ..Default::default() };
        let acc3 = Account { nonce: 1, balance: U256::from(10000u64), ..Default::default() };

        store
            .store_hashed_accounts(vec![(a2, None), (a1, Some(acc1)), (a3, Some(acc3))], B0)
            .await
            .expect("write");

        let tx = store.env.tx().expect("ro tx");
        let mut cur = tx.new_cursor::<HashedAccountHistory>().expect("cursor");

        let v1 = cur.seek_by_key_subkey(a1, B0).expect("seek a1").expect("exists a1");
        assert_eq!(v1.block_number, B0);
        assert_eq!(v1.value.0, Some(acc1));

        let v2 = cur.seek_by_key_subkey(a2, B0).expect("seek a2").expect("exists a2");
        assert_eq!(v2.block_number, B0);
        assert!(v2.value.0.is_none(), "a2 is none");

        let v3 = cur.seek_by_key_subkey(a3, B0).expect("seek a3").expect("exists a3");
        assert_eq!(v3.block_number, B0);
        assert_eq!(v3.value.0, Some(acc3));
    }

    #[tokio::test]
    async fn store_hashed_accounts_multiple_calls() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        // Unsorted input, mixed Some/None (deletion)
        let a1 = B256::from([0x01; 32]);
        let a2 = B256::from([0x02; 32]);
        let a3 = B256::from([0x03; 32]);
        let a4 = B256::from([0x04; 32]);
        let a5 = B256::from([0x05; 32]);
        let acc1 = Account { nonce: 2, balance: U256::from(1000u64), ..Default::default() };
        let acc3 = Account { nonce: 1, balance: U256::from(10000u64), ..Default::default() };
        let acc4 = Account { nonce: 5, balance: U256::from(5000u64), ..Default::default() };
        let acc5 = Account { nonce: 10, balance: U256::from(20000u64), ..Default::default() };

        {
            store
                .store_hashed_accounts(vec![(a2, None), (a1, Some(acc1)), (a4, Some(acc4))], B0)
                .await
                .expect("write");

            let tx = store.env.tx().expect("ro tx");
            let mut cur = tx.new_cursor::<HashedAccountHistory>().expect("cursor");

            let v1 = cur.seek_by_key_subkey(a1, B0).expect("seek a1").expect("exists a1");
            assert_eq!(v1.block_number, B0);
            assert_eq!(v1.value.0, Some(acc1));

            let v2 = cur.seek_by_key_subkey(a2, B0).expect("seek a2").expect("exists a2");
            assert_eq!(v2.block_number, B0);
            assert!(v2.value.0.is_none(), "a2 is none");

            let v4 = cur.seek_by_key_subkey(a4, B0).expect("seek a4").expect("exists a4");
            assert_eq!(v4.block_number, B0);
            assert_eq!(v4.value.0, Some(acc4));
        }

        {
            // Second call
            store
                .store_hashed_accounts(vec![(a5, Some(acc5)), (a3, Some(acc3))], B0)
                .await
                .expect("write");

            let tx = store.env.tx().expect("ro tx");
            let mut cur = tx.new_cursor::<HashedAccountHistory>().expect("cursor");

            let v3 = cur.seek_by_key_subkey(a3, B0).expect("seek a3").expect("exists a3");
            assert_eq!(v3.block_number, B0);
            assert_eq!(v3.value.0, Some(acc3));

            let v5 = cur.seek_by_key_subkey(a5, B0).expect("seek a5").expect("exists a5");
            assert_eq!(v5.block_number, B0);
            assert_eq!(v5.value.0, Some(acc5));
        }
    }

    #[tokio::test]
    async fn store_hashed_storages_writes_versioned_values() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        let addr = B256::from([0x11; 32]);
        let slot = B256::from([0x22; 32]);
        let val = U256::from(0x1234u64);

        store.store_hashed_storages(addr, vec![(slot, val)], B0).await.expect("write storage");

        let tx = store.env.tx().expect("ro tx");
        let mut cur = tx.new_cursor::<HashedStorageHistory>().expect("cursor");
        let key = HashedStorageKey::new(addr, slot);
        let vv = cur.seek_by_key_subkey(key, B0).expect("seek");
        let vv = vv.expect("entry exists");

        assert_eq!(vv.block_number, B0);
        let inner = vv.value.0.as_ref().expect("Some(StorageValue)");
        assert_eq!(inner.0, val);
    }

    #[tokio::test]
    async fn store_hashed_storages_multiple_slots_unsorted() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        let addr = B256::from([0x11; 32]);
        let s1 = B256::from([0x01; 32]);
        let v1 = U256::from(1u64);
        let s2 = B256::from([0x02; 32]);
        let v2 = U256::from(2u64);
        let s3 = B256::from([0x03; 32]);
        let v3 = U256::from(3u64);

        store
            .store_hashed_storages(addr, vec![(s2, v2), (s1, v1), (s3, v3)], B0)
            .await
            .expect("write");

        let tx = store.env.tx().expect("ro tx");
        let mut cur = tx.new_cursor::<HashedStorageHistory>().expect("cursor");

        for (slot, expected) in [(s1, v1), (s2, v2), (s3, v3)] {
            let key = HashedStorageKey::new(addr, slot);
            let vv = cur.seek_by_key_subkey(key, B0).expect("seek").expect("exists");
            assert_eq!(vv.block_number, B0);
            let inner = vv.value.0.as_ref().expect("Some(StorageValue)");
            assert_eq!(inner.0, expected);
        }
    }

    #[tokio::test]
    async fn store_hashed_storages_multiple_calls() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        let addr = B256::from([0x11; 32]);
        let s1 = B256::from([0x01; 32]);
        let v1 = U256::from(1u64);
        let s2 = B256::from([0x02; 32]);
        let v2 = U256::from(2u64);
        let s3 = B256::from([0x03; 32]);
        let v3 = U256::from(3u64);
        let s4 = B256::from([0x04; 32]);
        let v4 = U256::from(4u64);
        let s5 = B256::from([0x05; 32]);
        let v5 = U256::from(5u64);

        {
            store
                .store_hashed_storages(addr, vec![(s2, v2), (s1, v1), (s5, v5)], B0)
                .await
                .expect("write");

            let tx = store.env.tx().expect("ro tx");
            let mut cur = tx.new_cursor::<HashedStorageHistory>().expect("cursor");

            for (slot, expected) in [(s1, v1), (s2, v2), (s5, v5)] {
                let key = HashedStorageKey::new(addr, slot);
                let vv = cur.seek_by_key_subkey(key, B0).expect("seek").expect("exists");
                assert_eq!(vv.block_number, B0);
                let inner = vv.value.0.as_ref().expect("Some(StorageValue)");
                assert_eq!(inner.0, expected);
            }
        }

        {
            // Second call
            store.store_hashed_storages(addr, vec![(s4, v4), (s3, v3)], B0).await.expect("write");

            let tx = store.env.tx().expect("ro tx");
            let mut cur = tx.new_cursor::<HashedStorageHistory>().expect("cursor");

            for (slot, expected) in [(s4, v4), (s3, v3)] {
                let key = HashedStorageKey::new(addr, slot);
                let vv = cur.seek_by_key_subkey(key, B0).expect("seek").expect("exists");
                assert_eq!(vv.block_number, B0);
                let inner = vv.value.0.as_ref().expect("Some(StorageValue)");
                assert_eq!(inner.0, expected);
            }
        }
    }

    #[tokio::test]
    async fn test_store_account_branches_writes_versioned_values() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        let nibble = Nibbles::from_nibbles_unchecked([0x12, 0x34]);
        let branch_node = BranchNodeCompact::new(0b1, 0, 0, vec![], Some(B256::random()));
        let updates = vec![(nibble, Some(branch_node.clone()))];

        store.store_account_branches(B0, updates).await.expect("write");

        let tx = store.env.tx().expect("ro tx");
        let mut cur = tx.cursor_dup_read::<AccountTrieHistory>().expect("cursor");

        let vv = cur
            .seek_by_key_subkey(StoredNibbles::from(nibble), B0)
            .expect("seek")
            .expect("entry exists");

        assert_eq!(vv.block_number, B0);
        assert_eq!(vv.value.0, Some(branch_node));
    }

    #[tokio::test]
    async fn test_store_account_branches_multiple_items_unsorted() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        let n1 = Nibbles::from_nibbles_unchecked([0x01]);
        let b1 = BranchNodeCompact::new(0b1, 0, 0, vec![], Some(B256::random()));
        let n2 = Nibbles::from_nibbles_unchecked([0x02]);
        let n3 = Nibbles::from_nibbles_unchecked([0x03]);
        let b3 = BranchNodeCompact::new(0b1, 0, 0, vec![], Some(B256::random()));

        let updates = vec![(n2, None), (n1, Some(b1.clone())), (n3, Some(b3.clone()))];
        store.store_account_branches(B0, updates.clone()).await.expect("write");

        let tx = store.env.tx().expect("ro tx");
        let mut cur = tx.cursor_dup_read::<AccountTrieHistory>().expect("cursor");

        for (nibble, branch) in updates {
            let v = cur
                .seek_by_key_subkey(StoredNibbles::from(nibble), B0)
                .expect("seek")
                .expect("exists");
            assert_eq!(v.block_number, B0);
            assert_eq!(v.value.0, branch);
        }
    }

    #[tokio::test]
    async fn store_account_branches_multiple_calls() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        let n1 = Nibbles::from_nibbles_unchecked([0x01]);
        let b1 = BranchNodeCompact::new(0b1, 0, 0, vec![], Some(B256::random()));
        let n2 = Nibbles::from_nibbles_unchecked([0x02]);
        let n3 = Nibbles::from_nibbles_unchecked([0x03]);
        let b3 = BranchNodeCompact::new(0b1, 0, 0, vec![], Some(B256::random()));
        let n4 = Nibbles::from_nibbles_unchecked([0x04]);
        let b4 = BranchNodeCompact::new(0b1, 0, 0, vec![], Some(B256::random()));
        let n5 = Nibbles::from_nibbles_unchecked([0x05]);
        let b5 = BranchNodeCompact::new(0b1, 0, 0, vec![], Some(B256::random()));

        {
            let updates1 = vec![(n2, None), (n1, Some(b1.clone())), (n4, Some(b4.clone()))];
            store.store_account_branches(B0, updates1.clone()).await.expect("write");

            let tx = store.env.tx().expect("ro tx");
            let mut cur = tx.cursor_dup_read::<AccountTrieHistory>().expect("cursor");

            for (nibble, branch) in updates1 {
                let v = cur
                    .seek_by_key_subkey(StoredNibbles::from(nibble), B0)
                    .expect("seek")
                    .expect("exists");
                assert_eq!(v.block_number, B0);
                assert_eq!(v.value.0, branch);
            }
        }

        {
            // Second call
            let updates2 = vec![(n5, Some(b5.clone())), (n3, Some(b3.clone()))];
            store.store_account_branches(B0, updates2.clone()).await.expect("write");

            let tx = store.env.tx().expect("ro tx");
            let mut cur = tx.cursor_dup_read::<AccountTrieHistory>().expect("cursor");

            for (nibble, branch) in updates2 {
                let v = cur
                    .seek_by_key_subkey(StoredNibbles::from(nibble), B0)
                    .expect("seek")
                    .expect("exists");
                assert_eq!(v.block_number, B0);
                assert_eq!(v.value.0, branch);
            }
        }
    }

    #[tokio::test]
    async fn test_store_storage_branches_writes_versioned_values() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        let hashed_address = B256::random();
        let nibble = Nibbles::from_nibbles_unchecked([0x12, 0x34]);
        let branch_node = BranchNodeCompact::new(0b1, 0, 0, vec![], Some(B256::random()));
        let items = vec![(nibble, Some(branch_node.clone()))];

        store.store_storage_branches(B0, hashed_address, items).await.expect("write");

        let tx = store.env.tx().expect("ro tx");
        let mut cur = tx.cursor_dup_read::<StorageTrieHistory>().expect("cursor");

        let key = StorageTrieKey::new(hashed_address, StoredNibbles::from(nibble));
        let vv = cur.seek_by_key_subkey(key, B0).expect("seek").expect("entry exists");

        assert_eq!(vv.block_number, B0);
        assert_eq!(vv.value.0, Some(branch_node));
    }

    #[tokio::test]
    async fn store_storage_branches_multiple_items_unsorted() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        let hashed_address = B256::random();
        let n1 = Nibbles::from_nibbles_unchecked([0x01]);
        let b1 = BranchNodeCompact::new(0b1, 0, 0, vec![], Some(B256::random()));
        let n2 = Nibbles::from_nibbles_unchecked([0x02]);
        let n3 = Nibbles::from_nibbles_unchecked([0x03]);
        let b3 = BranchNodeCompact::new(0b1, 0, 0, vec![], Some(B256::random()));

        let items = vec![(n2, None), (n1, Some(b1.clone())), (n3, Some(b3.clone()))];
        store.store_storage_branches(B0, hashed_address, items.clone()).await.expect("write");

        let tx = store.env.tx().expect("ro tx");
        let mut cur = tx.cursor_dup_read::<StorageTrieHistory>().expect("cursor");

        for (nibble, branch) in items {
            let key = StorageTrieKey::new(hashed_address, StoredNibbles::from(nibble));
            let v = cur.seek_by_key_subkey(key, B0).expect("seek").expect("exists");
            assert_eq!(v.block_number, B0);
            assert_eq!(v.value.0, branch);
        }
    }

    #[tokio::test]
    async fn store_storage_branches_multiple_calls() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        let hashed_address = B256::random();
        let n1 = Nibbles::from_nibbles_unchecked([0x01]);
        let b1 = BranchNodeCompact::new(0b1, 0, 0, vec![], Some(B256::random()));
        let n2 = Nibbles::from_nibbles_unchecked([0x02]);
        let n3 = Nibbles::from_nibbles_unchecked([0x03]);
        let b3 = BranchNodeCompact::new(0b1, 0, 0, vec![], Some(B256::random()));
        let n4 = Nibbles::from_nibbles_unchecked([0x04]);
        let b4 = BranchNodeCompact::new(0b1, 0, 0, vec![], Some(B256::random()));
        let n5 = Nibbles::from_nibbles_unchecked([0x05]);
        let b5 = BranchNodeCompact::new(0b1, 0, 0, vec![], Some(B256::random()));

        {
            let items1 = vec![(n2, None), (n1, Some(b1.clone())), (n5, Some(b5.clone()))];
            store.store_storage_branches(B0, hashed_address, items1.clone()).await.expect("write");

            let tx = store.env.tx().expect("ro tx");
            let mut cur = tx.cursor_dup_read::<StorageTrieHistory>().expect("cursor");

            for (nibble, branch) in items1 {
                let key = StorageTrieKey::new(hashed_address, StoredNibbles::from(nibble));
                let v = cur.seek_by_key_subkey(key, B0).expect("seek").expect("exists");
                assert_eq!(v.block_number, B0);
                assert_eq!(v.value.0, branch);
            }
        }

        {
            // Second call
            let items2 = vec![(n4, Some(b4.clone())), (n3, Some(b3.clone()))];
            store.store_storage_branches(B0, hashed_address, items2.clone()).await.expect("write");

            let tx = store.env.tx().expect("ro tx");
            let mut cur = tx.cursor_dup_read::<StorageTrieHistory>().expect("cursor");

            for (nibble, branch) in items2 {
                let key = StorageTrieKey::new(hashed_address, StoredNibbles::from(nibble));
                let v = cur.seek_by_key_subkey(key, B0).expect("seek").expect("exists");
                assert_eq!(v.block_number, B0);
                assert_eq!(v.value.0, branch);
            }
        }
    }
}

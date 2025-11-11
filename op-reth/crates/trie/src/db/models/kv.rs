use crate::db::{
    AccountTrieHistory, HashedAccountHistory, HashedStorageHistory, HashedStorageKey, MaybeDeleted,
    StorageTrieHistory, StorageTrieKey, StorageValue, VersionedValue,
};
use alloy_primitives::B256;
use reth_db::table::{DupSort, Table};
use reth_primitives_traits::Account;
use reth_trie::{BranchNodeCompact, Nibbles, StoredNibbles};

/// Helper to convert inputs into a table key or kv pair.
pub trait IntoKV<Tab: Table + DupSort> {
    /// Convert `self` into the table key.
    fn into_key(self) -> Tab::Key;

    /// Convert `self` into kv for the given `block_number`.
    fn into_kv(self, block_number: u64) -> (Tab::Key, Tab::Value);
}

impl IntoKV<AccountTrieHistory> for (Nibbles, Option<BranchNodeCompact>) {
    fn into_key(self) -> StoredNibbles {
        StoredNibbles::from(self.0)
    }

    fn into_kv(self, block_number: u64) -> (StoredNibbles, VersionedValue<BranchNodeCompact>) {
        let (path, node) = self;
        (StoredNibbles::from(path), VersionedValue { block_number, value: MaybeDeleted(node) })
    }
}

impl IntoKV<StorageTrieHistory> for (B256, Nibbles, Option<BranchNodeCompact>) {
    fn into_key(self) -> StorageTrieKey {
        let (hashed_address, path, _) = self;
        StorageTrieKey::new(hashed_address, StoredNibbles::from(path))
    }
    fn into_kv(self, block_number: u64) -> (StorageTrieKey, VersionedValue<BranchNodeCompact>) {
        let (hashed_address, path, node) = self;
        (
            StorageTrieKey::new(hashed_address, StoredNibbles::from(path)),
            VersionedValue { block_number, value: MaybeDeleted(node) },
        )
    }
}

impl IntoKV<HashedAccountHistory> for (B256, Option<Account>) {
    fn into_key(self) -> B256 {
        self.0
    }
    fn into_kv(self, block_number: u64) -> (B256, VersionedValue<Account>) {
        let (hashed_address, account) = self;
        (hashed_address, VersionedValue { block_number, value: MaybeDeleted(account) })
    }
}

impl IntoKV<HashedStorageHistory> for (B256, B256, Option<StorageValue>) {
    fn into_key(self) -> HashedStorageKey {
        let (hashed_address, hashed_storage_key, _) = self;
        HashedStorageKey::new(hashed_address, hashed_storage_key)
    }
    fn into_kv(self, block_number: u64) -> (HashedStorageKey, VersionedValue<StorageValue>) {
        let (hashed_address, hashed_storage_key, value) = self;
        (
            HashedStorageKey::new(hashed_address, hashed_storage_key),
            VersionedValue { block_number, value: MaybeDeleted(value) },
        )
    }
}

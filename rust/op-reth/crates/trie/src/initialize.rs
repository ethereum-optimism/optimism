//! Initialization job for proofs storage. Handles storing the existing state into the proofs
//! storage.

use crate::{
    OpProofsStorageError, OpProofsStore,
    api::{InitialStateAnchor, InitialStateStatus, OpProofsInitialStateStore},
    db::{HashedStorageKey, StorageTrieKey},
};
use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, U256};
use derive_more::Constructor;
use reth_db::{
    DatabaseError,
    cursor::{DbCursorRO, DbDupCursorRO},
    tables,
    transaction::DbTx,
};
use reth_primitives_traits::{Account, StorageEntry};
use reth_trie_common::{
    BranchNodeCompact, Nibbles, StorageTrieEntry, StoredNibbles, StoredNibblesSubKey,
};
use std::{collections::HashMap, time::Instant};
use tracing::{debug, info};

/// Batch size threshold for storing entries during initialization
const INITIALIZE_STORAGE_THRESHOLD: usize = 100000;

/// Threshold for logging progress during initialization
const INITIALIZE_LOG_THRESHOLD: usize = 100000;

/// Initialization job for external storage.
#[derive(Debug, Constructor)]
pub struct InitializationJob<Tx: DbTx, S: OpProofsStore + Send> {
    storage: S,
    tx: Tx,
}

/// Macro to generate simple cursor iterators for tables
macro_rules! define_simple_cursor_iter {
    ($iter_name:ident, $table:ty, $key_type:ty, $value_type:ty) => {
        struct $iter_name<C>(C);

        impl<C> $iter_name<C> {
            const fn new(cursor: C) -> Self {
                Self(cursor)
            }
        }

        impl<C: DbCursorRO<$table>> Iterator for $iter_name<C> {
            type Item = Result<($key_type, $value_type), DatabaseError>;

            fn next(&mut self) -> Option<Self::Item> {
                self.0.next().transpose()
            }
        }
    };
}

/// Macro to generate duplicate cursor iterators for tables with custom logic
macro_rules! define_dup_cursor_iter {
    ($iter_name:ident, $table:ty, $key_type:ty, $value_type:ty) => {
        struct $iter_name<C>(C);

        impl<C> $iter_name<C> {
            const fn new(cursor: C) -> Self {
                Self(cursor)
            }
        }

        impl<C: DbDupCursorRO<$table> + DbCursorRO<$table>> Iterator for $iter_name<C> {
            type Item = Result<($key_type, $value_type), DatabaseError>;

            fn next(&mut self) -> Option<Self::Item> {
                // First try to get the next duplicate value
                if let Some(res) = self.0.next_dup().transpose() {
                    return Some(res);
                }

                // If no more duplicates, find the next key with values
                let Some(Ok((next_key, _))) = self.0.next_no_dup().transpose() else {
                    // If no more entries, return None
                    return None;
                };

                // If found, seek to the first duplicate for this key
                return self.0.seek(next_key).transpose();
            }
        }
    };
}

// Generate iterators for all 4 table types
define_simple_cursor_iter!(HashedAccountsInit, tables::HashedAccounts, B256, Account);
define_dup_cursor_iter!(HashedStoragesInit, tables::HashedStorages, B256, StorageEntry);
define_simple_cursor_iter!(
    AccountsTrieInit,
    tables::AccountsTrie,
    StoredNibbles,
    BranchNodeCompact
);
define_dup_cursor_iter!(StoragesTrieInit, tables::StoragesTrie, B256, StorageTrieEntry);

/// Trait to estimate the progress of a initialization job based on the key.
trait CompletionEstimatable {
    // Returns a progress estimate as a percentage (0.0 to 1.0)
    fn estimate_progress(&self) -> f64;
}

impl CompletionEstimatable for B256 {
    fn estimate_progress(&self) -> f64 {
        // use the first 3 bytes as a progress estimate
        let progress = self.0[..3].to_vec();
        let mut val: u64 = 0;
        for nibble in &progress {
            val = (val << 8) | *nibble as u64;
        }
        val as f64 / (256u64.pow(3)) as f64
    }
}

impl CompletionEstimatable for StoredNibbles {
    fn estimate_progress(&self) -> f64 {
        // use the first 6 nibbles as a progress estimate
        let progress_nibbles =
            if self.0.is_empty() { Nibbles::new() } else { self.0.slice(0..(self.0.len().min(6))) };
        let mut val: u64 = 0;
        for nibble in progress_nibbles.iter() {
            val = (val << 4) | nibble as u64;
        }
        val as f64 / (16u64.pow(progress_nibbles.len() as u32)) as f64
    }
}

impl<Tx: DbTx + Sync, S: OpProofsStore + OpProofsInitialStateStore + Send>
    InitializationJob<Tx, S>
{
    /// Initialize a table from a source iterator to a storage function. Handles batching and
    /// logging.
    fn initialize<
        I: Iterator<Item = Result<(Key, Value), DatabaseError>> + InitTable<Key = Key, Value = Value>,
        Key: CompletionEstimatable + 'static,
        Value: 'static,
    >(
        &self,
        name: &str,
        source: I,
        storage_threshold: usize,
        log_threshold: usize,
    ) -> Result<u64, OpProofsStorageError> {
        info!("Starting {} initialization", name);
        let start_time = Instant::now();

        let mut source = source.peekable();
        let Some(first_entry) = source.peek() else {
            debug!(target: "reth::cli", "No entries to store for table");
            return Ok(0);
        };
        let initial_progress = match first_entry {
            Ok(i) => i.0.estimate_progress(),
            Err(e) => Err(e.clone())?,
        };

        let storage = &self.storage;
        let source_size_hint = source.size_hint().0;
        let mut batch = Vec::with_capacity(source_size_hint.min(storage_threshold));
        let mut total_entries: usize = 0;

        for entry in source {
            batch.push(entry?);
            total_entries += 1;

            if total_entries.is_multiple_of(log_threshold) {
                let progress = batch.last().expect("non-empty batch").0.estimate_progress();
                let elapsed = start_time.elapsed();
                let elapsed_secs = elapsed.as_secs_f64();

                let progress_per_second = if elapsed_secs.is_normal() {
                    (progress - initial_progress) / elapsed_secs
                } else {
                    0.0
                };
                let estimated_total_time = if progress_per_second.is_normal() {
                    (1.0 - progress) / progress_per_second
                } else {
                    0.0
                };
                let progress_pct = progress * 100.0;
                info!(
                    "Processed {} {}, progress: {progress_pct:.2}%, ETA: {}s",
                    name, total_entries, estimated_total_time,
                );
            }

            if batch.len() >= storage_threshold {
                info!("Storing {} entries, total entries: {}", name, total_entries);
                I::store_entries(storage, batch)?;
                batch = Vec::with_capacity(
                    (source_size_hint.saturating_sub(total_entries)).min(storage_threshold),
                );
            }
        }

        if !batch.is_empty() {
            info!("Storing final {} entries", name);
            I::store_entries(storage, batch)?;
        }

        info!("{} initialization complete: {} entries", name, total_entries);
        Ok(total_entries as u64)
    }

    /// Initialize hashed accounts data
    fn initialize_hashed_accounts(
        &self,
        start_key: Option<B256>,
    ) -> Result<(), OpProofsStorageError> {
        let mut start_cursor = self.tx.cursor_read::<tables::HashedAccounts>()?;

        if let Some(latest) = start_key {
            start_cursor
                .seek(latest)?
                .filter(|(k, _)| *k == latest)
                .ok_or(OpProofsStorageError::InitializeStorageInconsistentState)?;
        }

        let source = HashedAccountsInit::new(start_cursor);
        self.initialize(
            "hashed accounts",
            source,
            INITIALIZE_STORAGE_THRESHOLD,
            INITIALIZE_LOG_THRESHOLD,
        )?;

        Ok(())
    }

    /// Initialize hashed storage data
    fn initialize_hashed_storages(
        &self,
        start_key: Option<HashedStorageKey>,
    ) -> Result<(), OpProofsStorageError> {
        let mut start_cursor = self.tx.cursor_dup_read::<tables::HashedStorages>()?;

        if let Some(latest) = start_key {
            start_cursor
                .seek_by_key_subkey(latest.hashed_address, latest.hashed_storage_key)?
                .filter(|v| v.key == latest.hashed_storage_key)
                .ok_or(OpProofsStorageError::InitializeStorageInconsistentState)?;
        }

        let source = HashedStoragesInit::new(start_cursor);
        self.initialize(
            "hashed storage",
            source,
            INITIALIZE_STORAGE_THRESHOLD,
            INITIALIZE_LOG_THRESHOLD,
        )?;

        Ok(())
    }

    /// Initialize accounts trie data
    fn initialize_accounts_trie(
        &self,
        start_key: Option<StoredNibbles>,
    ) -> Result<(), OpProofsStorageError> {
        let mut start_cursor = self.tx.cursor_read::<tables::AccountsTrie>()?;

        if let Some(latest_key) = start_key {
            start_cursor
                .seek(latest_key.clone())?
                .filter(|(k, _)| *k == latest_key)
                .ok_or(OpProofsStorageError::InitializeStorageInconsistentState)?;
        }

        let source = AccountsTrieInit::new(start_cursor);
        self.initialize(
            "accounts trie",
            source,
            INITIALIZE_STORAGE_THRESHOLD,
            INITIALIZE_LOG_THRESHOLD,
        )?;

        Ok(())
    }

    /// Initialize storage trie data
    fn initialize_storages_trie(
        &self,
        start_key: Option<StorageTrieKey>,
    ) -> Result<(), OpProofsStorageError> {
        let mut start_cursor = self.tx.cursor_dup_read::<tables::StoragesTrie>()?;

        if let Some(latest_key) = start_key {
            start_cursor
                .seek_by_key_subkey(
                    latest_key.hashed_address,
                    StoredNibblesSubKey::from(latest_key.path.0),
                )?
                .filter(|v| v.nibbles.0 == latest_key.path.0)
                .ok_or(OpProofsStorageError::InitializeStorageInconsistentState)?;
        }

        let source = StoragesTrieInit::new(start_cursor);
        self.initialize(
            "storage trie",
            source,
            INITIALIZE_STORAGE_THRESHOLD,
            INITIALIZE_LOG_THRESHOLD,
        )?;

        Ok(())
    }

    /// Run complete initialization of all preimage data
    fn initialize_trie(&self, anchor: InitialStateAnchor) -> Result<(), OpProofsStorageError> {
        self.initialize_hashed_accounts(anchor.latest_hashed_account_key)?;
        self.initialize_hashed_storages(anchor.latest_hashed_storage_key)?;
        self.initialize_storages_trie(anchor.latest_storage_trie_key)?;
        self.initialize_accounts_trie(anchor.latest_account_trie_key)?;
        Ok(())
    }

    fn validate_anchor_block(
        &self,
        anchor: &InitialStateAnchor,
        best_number: u64,
        best_hash: B256,
    ) -> Result<(), OpProofsStorageError> {
        let block = anchor.block.ok_or(OpProofsStorageError::InitializeStorageInconsistentState)?;

        if block.number != best_number || block.hash != best_hash {
            return Err(OpProofsStorageError::InitializeStorageInconsistentState);
        }

        Ok(())
    }

    /// Run the initialization job.
    pub fn run(&self, best_number: u64, best_hash: B256) -> Result<(), OpProofsStorageError> {
        let anchor = self.storage.initial_state_anchor()?;

        match anchor.status {
            InitialStateStatus::Completed => return Ok(()),
            InitialStateStatus::NotStarted => {
                self.storage.set_initial_state_anchor(BlockNumHash::new(best_number, best_hash))?;
            }
            InitialStateStatus::InProgress => {
                self.validate_anchor_block(&anchor, best_number, best_hash)?;
            }
        }

        self.initialize_trie(anchor)?;
        self.storage.commit_initial_state()?;

        Ok(())
    }
}

/// Handles storing entries for a particular KV-pair type.
trait InitTable {
    /// Key of target table.
    type Key: CompletionEstimatable + 'static;
    /// Value of target table.
    type Value: 'static;

    /// Writes given entries to given storage.
    fn store_entries(
        store: &impl OpProofsInitialStateStore,
        entries: impl IntoIterator<Item = (Self::Key, Self::Value)>,
    ) -> Result<(), OpProofsStorageError>;
}

impl<C> InitTable for HashedAccountsInit<C> {
    type Key = B256;
    type Value = Account;

    /// Save mapping of hashed addresses to accounts to storage.
    fn store_entries(
        store: &impl OpProofsInitialStateStore,
        entries: impl IntoIterator<Item = (Self::Key, Self::Value)>,
    ) -> Result<(), OpProofsStorageError> {
        store.store_hashed_accounts(
            entries.into_iter().map(|(address, account)| (address, Some(account))).collect(),
        )?;
        Ok(())
    }
}

impl<C> InitTable for HashedStoragesInit<C> {
    type Key = B256;
    type Value = StorageEntry;

    /// Save mapping of hashed addresses to storage entries to storage.
    fn store_entries(
        store: &impl OpProofsInitialStateStore,
        entries: impl IntoIterator<Item = (Self::Key, Self::Value)>,
    ) -> Result<(), OpProofsStorageError> {
        let entries_iter = entries.into_iter();
        let mut by_address: HashMap<B256, Vec<(B256, U256)>> =
            HashMap::with_capacity(entries_iter.size_hint().0);

        // Group entries by hashed address
        for (address, entry) in entries_iter {
            by_address.entry(address).or_default().push((entry.key, entry.value));
        }
        // Store each address's storage entries
        for (address, storages) in by_address {
            store.store_hashed_storages(address, storages)?;
        }

        Ok(())
    }
}

impl<C> InitTable for AccountsTrieInit<C> {
    type Key = StoredNibbles;
    type Value = BranchNodeCompact;

    /// Save mapping of account trie paths to branch nodes to storage.
    fn store_entries(
        store: &impl OpProofsInitialStateStore,
        entries: impl IntoIterator<Item = (Self::Key, Self::Value)>,
    ) -> Result<(), OpProofsStorageError> {
        store.store_account_branches(
            entries.into_iter().map(|(path, branch)| (path.0, Some(branch))).collect(),
        )?;

        Ok(())
    }
}

impl<C> InitTable for StoragesTrieInit<C> {
    type Key = B256;
    type Value = StorageTrieEntry;

    /// Save mapping of hashed addresses to storage trie entries to storage.
    fn store_entries(
        store: &impl OpProofsInitialStateStore,
        entries: impl IntoIterator<Item = (Self::Key, Self::Value)>,
    ) -> Result<(), OpProofsStorageError> {
        let entries_iter = entries.into_iter();
        let mut by_address: HashMap<B256, Vec<(Nibbles, Option<BranchNodeCompact>)>> =
            HashMap::with_capacity(entries_iter.size_hint().0);

        // Group entries by hashed address
        for (hashed_address, storage_entry) in entries_iter {
            by_address
                .entry(hashed_address)
                .or_default()
                .push((storage_entry.nibbles.0, Some(storage_entry.node)));
        }
        // Store each address's storage trie branches
        for (address, branches) in by_address {
            store.store_storage_branches(address, branches)?;
        }

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::MdbxProofsStorage;
    use alloy_primitives::{Address, U256, keccak256};
    use reth_db::{
        Database, cursor::DbCursorRW, test_utils::create_test_rw_db, transaction::DbTxMut,
    };
    use reth_primitives_traits::Account;
    use reth_trie::{
        BranchNodeCompact, StorageTrieEntry, StoredNibbles, StoredNibblesSubKey, TrieMask,
        hashed_cursor::HashedCursor, trie_cursor::TrieCursor,
    };
    use std::sync::Arc;
    use tempfile::TempDir;

    /// Helper function to create a key
    fn k(b: u8) -> B256 {
        let mut bytes = [0u8; 32];
        bytes[0] = b;
        B256::from(bytes)
    }

    /// Helper function to create a test branch node
    fn create_test_branch_node() -> BranchNodeCompact {
        let mut state_mask = TrieMask::default();
        state_mask.set_bit(0);
        state_mask.set_bit(1);

        BranchNodeCompact {
            state_mask,
            tree_mask: TrieMask::default(),
            hash_mask: TrieMask::default(),
            hashes: Arc::new(vec![]),
            root_hash: None,
        }
    }

    #[test]
    fn test_initialize_hashed_accounts() {
        let db = create_test_rw_db();
        let dir = TempDir::new().unwrap();
        let storage = Arc::new(MdbxProofsStorage::new(dir.path()).expect("env"));

        // Insert test accounts into database
        let tx = db.tx_mut().unwrap();
        let mut cursor = tx.cursor_write::<tables::HashedAccounts>().unwrap();

        let mut accounts = vec![
            (
                keccak256(Address::repeat_byte(0x01)),
                Account { nonce: 1, balance: U256::from(100), bytecode_hash: None },
            ),
            (
                keccak256(Address::repeat_byte(0x02)),
                Account { nonce: 2, balance: U256::from(200), bytecode_hash: None },
            ),
            (
                keccak256(Address::repeat_byte(0x03)),
                Account { nonce: 3, balance: U256::from(300), bytecode_hash: None },
            ),
        ];

        // Sort accounts by address for cursor.append (which requires sorted order)
        accounts.sort_by_key(|(addr, _)| *addr);

        for (addr, account) in &accounts {
            cursor.append(*addr, account).unwrap();
        }
        drop(cursor);
        tx.commit().unwrap();

        // Run initialization
        let tx = db.tx().unwrap();
        let job = InitializationJob::new(storage.clone(), tx);
        job.initialize_hashed_accounts(None).unwrap();

        // Verify data was stored (will be in sorted order)
        let mut account_cursor = storage.account_hashed_cursor(100).unwrap();
        let mut count = 0;
        while let Some((key, account)) = account_cursor.next().unwrap() {
            // Find matching account in our test data
            let expected = accounts.iter().find(|(addr, _)| *addr == key).unwrap();
            assert_eq!((key, account), *expected);
            count += 1;
        }
        assert_eq!(count, 3);
    }

    #[test]
    fn test_initialize_hashed_storage() {
        let db = create_test_rw_db();
        let dir = TempDir::new().unwrap();
        let storage = Arc::new(MdbxProofsStorage::new(dir.path()).expect("env"));

        // Insert test storage into database
        let tx = db.tx_mut().unwrap();
        let mut cursor = tx.cursor_dup_write::<tables::HashedStorages>().unwrap();

        let addr1 = keccak256(Address::repeat_byte(0x01));
        let addr2 = keccak256(Address::repeat_byte(0x02));

        let storage_entries = vec![
            (
                addr1,
                StorageEntry { key: keccak256(B256::repeat_byte(0x10)), value: U256::from(100) },
            ),
            (
                addr1,
                StorageEntry { key: keccak256(B256::repeat_byte(0x20)), value: U256::from(200) },
            ),
            (
                addr2,
                StorageEntry { key: keccak256(B256::repeat_byte(0x30)), value: U256::from(300) },
            ),
        ];

        for (addr, entry) in &storage_entries {
            cursor.upsert(*addr, entry).unwrap();
        }
        drop(cursor);
        tx.commit().unwrap();

        // Run initialization
        let tx = db.tx().unwrap();
        let job = InitializationJob::new(storage.clone(), tx);
        job.initialize_hashed_storages(None).unwrap();

        // Verify data was stored for addr1
        let mut storage_cursor = storage.storage_hashed_cursor(addr1, 100).unwrap();
        let mut found = vec![];
        while let Some((key, value)) = storage_cursor.next().unwrap() {
            found.push((key, value));
        }
        assert_eq!(found.len(), 2);
        assert_eq!(found[0], (storage_entries[0].1.key, storage_entries[0].1.value));
        assert_eq!(found[1], (storage_entries[1].1.key, storage_entries[1].1.value));

        // Verify data was stored for addr2
        let mut storage_cursor = storage.storage_hashed_cursor(addr2, 100).unwrap();
        let mut found = vec![];
        while let Some((key, value)) = storage_cursor.next().unwrap() {
            found.push((key, value));
        }
        assert_eq!(found.len(), 1);
        assert_eq!(found[0], (storage_entries[2].1.key, storage_entries[2].1.value));
    }

    #[test]
    fn test_initialize_accounts_trie() {
        let db = create_test_rw_db();
        let dir = TempDir::new().unwrap();
        let storage = Arc::new(MdbxProofsStorage::new(dir.path()).expect("env"));

        // Insert test trie nodes into database
        let tx = db.tx_mut().unwrap();
        let mut cursor = tx.cursor_write::<tables::AccountsTrie>().unwrap();

        let branch = create_test_branch_node();
        let nodes = vec![
            (StoredNibbles(Nibbles::from_nibbles_unchecked(vec![1])), branch.clone()),
            (StoredNibbles(Nibbles::from_nibbles_unchecked(vec![2])), branch.clone()),
            (StoredNibbles(Nibbles::from_nibbles_unchecked(vec![3])), branch),
        ];

        for (path, node) in &nodes {
            cursor.append(path.clone(), node).unwrap();
        }
        drop(cursor);
        tx.commit().unwrap();

        // Run initialization
        let tx = db.tx().unwrap();
        let job = InitializationJob::new(storage.clone(), tx);
        job.initialize_accounts_trie(None).unwrap();

        // Verify data was stored
        let mut trie_cursor = storage.account_trie_cursor(100).unwrap();
        let mut count = 0;
        while let Some((path, _node)) = trie_cursor.next().unwrap() {
            assert_eq!(path, nodes[count].0.0);
            count += 1;
        }
        assert_eq!(count, 3);
    }

    #[test]
    fn test_initialize_storages_trie() {
        let db = create_test_rw_db();
        let dir = TempDir::new().unwrap();
        let storage = Arc::new(MdbxProofsStorage::new(dir.path()).expect("env"));

        // Insert test storage trie nodes into database
        let tx = db.tx_mut().unwrap();
        let mut cursor = tx.cursor_dup_write::<tables::StoragesTrie>().unwrap();

        let branch = create_test_branch_node();
        let addr1 = keccak256(Address::repeat_byte(0x01));
        let addr2 = keccak256(Address::repeat_byte(0x02));

        let nodes = vec![
            (
                addr1,
                StorageTrieEntry {
                    nibbles: StoredNibblesSubKey(Nibbles::from_nibbles_unchecked(vec![1])),
                    node: branch.clone(),
                },
            ),
            (
                addr1,
                StorageTrieEntry {
                    nibbles: StoredNibblesSubKey(Nibbles::from_nibbles_unchecked(vec![2])),
                    node: branch.clone(),
                },
            ),
            (
                addr2,
                StorageTrieEntry {
                    nibbles: StoredNibblesSubKey(Nibbles::from_nibbles_unchecked(vec![3])),
                    node: branch,
                },
            ),
        ];

        for (addr, entry) in &nodes {
            cursor.upsert(*addr, entry).unwrap();
        }
        drop(cursor);
        tx.commit().unwrap();

        // Run initialization
        let tx = db.tx().unwrap();
        let job = InitializationJob::new(storage.clone(), tx);
        job.initialize_storages_trie(None).unwrap();

        // Verify data was stored for addr1
        let mut trie_cursor = storage.storage_trie_cursor(addr1, 100).unwrap();
        let mut found = vec![];
        while let Some((path, _node)) = trie_cursor.next().unwrap() {
            found.push(path);
        }
        assert_eq!(found.len(), 2);
        assert_eq!(found[0], nodes[0].1.nibbles.0);
        assert_eq!(found[1], nodes[1].1.nibbles.0);

        // Verify data was stored for addr2
        let mut trie_cursor = storage.storage_trie_cursor(addr2, 100).unwrap();
        let mut found = vec![];
        while let Some((path, _node)) = trie_cursor.next().unwrap() {
            found.push(path);
        }
        assert_eq!(found.len(), 1);
        assert_eq!(found[0], nodes[2].1.nibbles.0);
    }

    #[test]
    fn test_full_initialize_run() {
        let db = create_test_rw_db();
        let dir = TempDir::new().unwrap();
        let storage = Arc::new(MdbxProofsStorage::new(dir.path()).expect("env"));

        // Insert some test data
        let tx = db.tx_mut().unwrap();

        // Add accounts
        let mut cursor = tx.cursor_write::<tables::HashedAccounts>().unwrap();
        let addr = keccak256(Address::repeat_byte(0x01));
        cursor
            .append(addr, &Account { nonce: 1, balance: U256::from(100), bytecode_hash: None })
            .unwrap();
        drop(cursor);

        // Add storage
        let mut cursor = tx.cursor_dup_write::<tables::HashedStorages>().unwrap();
        cursor
            .upsert(
                addr,
                &StorageEntry { key: keccak256(B256::repeat_byte(0x10)), value: U256::from(100) },
            )
            .unwrap();
        drop(cursor);

        // Add account trie
        let mut cursor = tx.cursor_write::<tables::AccountsTrie>().unwrap();
        cursor
            .append(
                StoredNibbles(Nibbles::from_nibbles_unchecked(vec![1])),
                &create_test_branch_node(),
            )
            .unwrap();
        drop(cursor);

        // Add storage trie
        let mut cursor = tx.cursor_dup_write::<tables::StoragesTrie>().unwrap();
        cursor
            .upsert(
                addr,
                &StorageTrieEntry {
                    nibbles: StoredNibblesSubKey(Nibbles::from_nibbles_unchecked(vec![1])),
                    node: create_test_branch_node(),
                },
            )
            .unwrap();
        drop(cursor);

        tx.commit().unwrap();

        // Run full initialization
        let tx = db.tx().unwrap();
        let job = InitializationJob::new(storage.clone(), tx);
        let best_number = 100;
        let best_hash = B256::repeat_byte(0x42);

        // Should be None initially
        assert_eq!(storage.initial_state_anchor().unwrap().block, None);
        assert_eq!(storage.get_earliest_block_number().unwrap(), None);

        job.run(best_number, best_hash).unwrap();

        // Should be set after initialization
        assert_eq!(storage.get_earliest_block_number().unwrap(), Some((best_number, best_hash)));

        // Verify data was initialized
        let mut account_cursor = storage.account_hashed_cursor(100).unwrap();
        assert!(account_cursor.next().unwrap().is_some());

        let mut storage_cursor = storage.storage_hashed_cursor(addr, 100).unwrap();
        assert!(storage_cursor.next().unwrap().is_some());

        let mut trie_cursor = storage.account_trie_cursor(100).unwrap();
        assert!(trie_cursor.next().unwrap().is_some());

        let mut storage_trie_cursor = storage.storage_trie_cursor(addr, 100).unwrap();
        assert!(storage_trie_cursor.next().unwrap().is_some());
    }

    #[test]
    fn test_initialize_run_skips_if_already_done() {
        let db = create_test_rw_db();
        let dir = TempDir::new().unwrap();
        let storage = Arc::new(MdbxProofsStorage::new(dir.path()).expect("env"));

        // set and commit initial state anchor
        storage
            .set_initial_state_anchor(BlockNumHash::new(50, B256::repeat_byte(0x01)))
            .expect("set anchor");
        storage.commit_initial_state().expect("commit anchor");

        let tx = db.tx().unwrap();
        let job = InitializationJob::new(storage.clone(), tx);

        // Run initialization - should skip
        job.run(100, B256::repeat_byte(0x42)).unwrap();

        // Should still have the old anchor
        let anchor_block =
            storage.initial_state_anchor().expect("get anchor").block.expect("block");
        assert_eq!(
            Some((anchor_block.number, anchor_block.hash)),
            Some((50, B256::repeat_byte(0x01)))
        );

        // Should still have the old earliest block
        assert_eq!(
            storage.get_earliest_block_number().unwrap(),
            Some((50, B256::repeat_byte(0x01)))
        );
    }

    #[test]
    fn test_initialize_resumes_hashed_accounts_with_no_dups() {
        let db = create_test_rw_db();
        let dir = TempDir::new().unwrap();
        let store = Arc::new(MdbxProofsStorage::new(dir.path()).expect("env"));

        store.set_initial_state_anchor(BlockNumHash::new(0, B256::default())).expect("set anchor");

        // Phase 1 in source: k1, k2
        let k1 = k(1);
        let k2 = k(2);
        {
            let tx = db.tx_mut().unwrap();
            let mut cur = tx.cursor_write::<tables::HashedAccounts>().unwrap();
            cur.append(k1, &Account { nonce: 1, balance: U256::from(100), bytecode_hash: None })
                .unwrap();
            cur.append(k2, &Account { nonce: 2, balance: U256::from(200), bytecode_hash: None })
                .unwrap();
            tx.commit().unwrap();
        }

        // Initialization #1
        {
            let tx = db.tx().unwrap();
            let job = InitializationJob::new(store.clone(), tx);
            job.initialize_hashed_accounts(None).unwrap();
        }

        // Resume point must be k2 (max)
        assert_eq!(
            store.initial_state_anchor().expect("get anchor").latest_hashed_account_key,
            Some(k2)
        );

        // Phase 2 in source: k3, k4
        let k3 = k(3);
        let k4 = k(4);
        {
            let tx = db.tx_mut().unwrap();
            let mut cur = tx.cursor_write::<tables::HashedAccounts>().unwrap();
            cur.append(k3, &Account { nonce: 3, balance: U256::from(300), bytecode_hash: None })
                .unwrap();
            cur.append(k4, &Account { nonce: 4, balance: U256::from(400), bytecode_hash: None })
                .unwrap();
            tx.commit().unwrap();
        }

        // Initialization #2 (restart)
        {
            let tx = db.tx().unwrap();
            let job = InitializationJob::new(store.clone(), tx);
            job.initialize_hashed_accounts(Some(k2)).unwrap();
        }

        // Now resume point must be k4
        assert_eq!(
            store.initial_state_anchor().expect("get anchor").latest_hashed_account_key,
            Some(k4)
        );

        // Verify order + no dupes by iterating proofs cursor
        let mut cur = store.account_hashed_cursor(0).unwrap();
        let mut got = Vec::new();
        while let Some((key, acct)) = cur.next().unwrap() {
            got.push((key, acct));
        }

        // Expect exactly 4, in increasing key order.
        assert_eq!(got.len(), 4);
        assert_eq!(got[0].0, k1);
        assert_eq!(got[1].0, k2);
        assert_eq!(got[2].0, k3);
        assert_eq!(got[3].0, k4);

        // No dupes
        for w in got.windows(2) {
            assert!(w[0].0 < w[1].0);
        }
    }

    #[test]
    fn test_initialize_resumes_hashed_storages_with_no_dups() {
        let db = create_test_rw_db();
        let dir = TempDir::new().unwrap();
        let store = Arc::new(MdbxProofsStorage::new(dir.path()).expect("env"));

        store.set_initial_state_anchor(BlockNumHash::new(0, B256::default())).expect("set anchor");

        let a1 = k(0x10);
        let a2 = k(0x20);

        let s11 = k(0x01);
        let s12 = k(0x02);
        let s21 = k(0x03);
        let s22 = k(0x04);

        // Phase 1 source:
        // a1: s11,s12
        // a2: s21
        {
            let tx = db.tx_mut().unwrap();
            let mut cur = tx.cursor_dup_write::<tables::HashedStorages>().unwrap();
            cur.upsert(a1, &StorageEntry { key: s11, value: U256::from(11) }).unwrap();
            cur.upsert(a1, &StorageEntry { key: s12, value: U256::from(12) }).unwrap();
            cur.upsert(a2, &StorageEntry { key: s21, value: U256::from(21) }).unwrap();
            tx.commit().unwrap();
        }

        // Initialization #1
        {
            let tx = db.tx().unwrap();
            let job = InitializationJob::new(store.clone(), tx);
            job.initialize_hashed_storages(None).unwrap();
        }

        // Latest key must be (a2, s21) because a2 > a1
        let last1 = store
            .initial_state_anchor()
            .expect("get anchor")
            .latest_hashed_storage_key
            .expect("ok");
        assert_eq!(last1.hashed_address, a2);
        assert_eq!(last1.hashed_storage_key, s21);

        // Phase 2 source: add s22 to a2
        {
            let tx = db.tx_mut().unwrap();
            let mut cur = tx.cursor_dup_write::<tables::HashedStorages>().unwrap();
            cur.upsert(a2, &StorageEntry { key: s22, value: U256::from(22) }).unwrap();
            tx.commit().unwrap();
        }

        // Initialization #2
        {
            let tx = db.tx().unwrap();
            let job = InitializationJob::new(store.clone(), tx);
            job.initialize_hashed_storages(Some(HashedStorageKey::new(a2, s21))).unwrap();
        }

        // Latest key now must be (a2, s22)
        let last2 = store
            .initial_state_anchor()
            .expect("get anchor")
            .latest_hashed_storage_key
            .expect("ok");
        assert_eq!(last2.hashed_address, a2);
        assert_eq!(last2.hashed_storage_key, s22);

        // Verify no dupes by iterating per-address
        {
            let mut c = store.storage_hashed_cursor(a1, 0).unwrap();
            let mut got = Vec::new();
            while let Some((slot, val)) = c.next().unwrap() {
                got.push((slot, val));
            }
            assert_eq!(got.len(), 2);
            assert_eq!(got[0].0, s11);
            assert_eq!(got[1].0, s12);
        }
        {
            let mut c = store.storage_hashed_cursor(a2, 0).unwrap();
            let mut got = Vec::new();
            while let Some((slot, val)) = c.next().unwrap() {
                got.push((slot, val));
            }
            assert_eq!(got.len(), 2);
            assert_eq!(got[0].0, s21);
            assert_eq!(got[1].0, s22);
        }
    }

    #[test]
    fn test_initialize_resumes_accounts_trie_with_no_dups() {
        let db = create_test_rw_db();
        let dir = TempDir::new().unwrap();
        let store = Arc::new(MdbxProofsStorage::new(dir.path()).expect("env"));

        store.set_initial_state_anchor(BlockNumHash::new(0, B256::default())).expect("set anchor");

        let p1 = StoredNibbles(Nibbles::from_nibbles_unchecked(vec![1]));
        let p2 = StoredNibbles(Nibbles::from_nibbles_unchecked(vec![2]));
        let p3 = StoredNibbles(Nibbles::from_nibbles_unchecked(vec![3]));
        let p4 = StoredNibbles(Nibbles::from_nibbles_unchecked(vec![4]));

        // Phase 1 source: p1,p2
        {
            let tx = db.tx_mut().unwrap();
            let mut cur = tx.cursor_write::<tables::AccountsTrie>().unwrap();
            cur.append(p1.clone(), &create_test_branch_node()).unwrap();
            cur.append(p2.clone(), &create_test_branch_node()).unwrap();
            tx.commit().unwrap();
        }

        // Initialization #1
        {
            let tx = db.tx().unwrap();
            let job = InitializationJob::new(store.clone(), tx);
            job.initialize_accounts_trie(None).unwrap();
        }

        assert_eq!(
            store.initial_state_anchor().expect("get anchor").latest_account_trie_key,
            Some(p2.clone())
        );

        // Phase 2 source: p3,p4
        {
            let tx = db.tx_mut().unwrap();
            let mut cur = tx.cursor_write::<tables::AccountsTrie>().unwrap();
            cur.append(p3.clone(), &create_test_branch_node()).unwrap();
            cur.append(p4.clone(), &create_test_branch_node()).unwrap();
            tx.commit().unwrap();
        }

        // Initialization #2
        {
            let tx = db.tx().unwrap();
            let job = InitializationJob::new(store.clone(), tx);
            job.initialize_accounts_trie(Some(p2.clone())).unwrap();
        }

        assert_eq!(
            store.initial_state_anchor().expect("get anchor").latest_account_trie_key,
            Some(p4.clone())
        );

        // Verify 4 ordered, no dupes
        let mut c = store.account_trie_cursor(0).unwrap();
        let mut got = Vec::new();
        while let Some((path, _node)) = c.next().unwrap() {
            got.push(path);
        }
        assert_eq!(got.len(), 4);
        assert_eq!(got[0], p1.0);
        assert_eq!(got[1], p2.0);
        assert_eq!(got[2], p3.0);
        assert_eq!(got[3], p4.0);
    }

    #[test]
    fn test_initialize_resumes_storages_trie_with_no_dups() {
        let db = create_test_rw_db();
        let dir = TempDir::new().unwrap();
        let store = Arc::new(MdbxProofsStorage::new(dir.path()).expect("env"));

        store.set_initial_state_anchor(BlockNumHash::new(0, B256::default())).expect("set anchor");

        let a1 = k(0x10);
        let a2 = k(0x20);

        let n1 = StoredNibblesSubKey(Nibbles::from_nibbles_unchecked(vec![1]));
        let n2 = StoredNibblesSubKey(Nibbles::from_nibbles_unchecked(vec![2]));
        let n3 = StoredNibblesSubKey(Nibbles::from_nibbles_unchecked(vec![3]));

        // Phase 1 source: (a1,n1), (a2,n2)
        {
            let tx = db.tx_mut().unwrap();
            let mut cur = tx.cursor_dup_write::<tables::StoragesTrie>().unwrap();
            cur.upsert(
                a1,
                &StorageTrieEntry { nibbles: n1.clone(), node: create_test_branch_node() },
            )
            .unwrap();
            cur.upsert(
                a2,
                &StorageTrieEntry { nibbles: n2.clone(), node: create_test_branch_node() },
            )
            .unwrap();
            tx.commit().unwrap();
        }

        // Initialization #1
        {
            let tx = db.tx().unwrap();
            let job = InitializationJob::new(store.clone(), tx);
            job.initialize_storages_trie(None).unwrap();
        }

        // Latest must be (a2, n2) because a2 > a1
        let last1 =
            store.initial_state_anchor().expect("get anchor").latest_storage_trie_key.expect("ok");
        assert_eq!(last1.hashed_address, a2);
        assert_eq!(last1.path.0, n2.0);

        // Phase 2 source: add (a2,n3)
        {
            let tx = db.tx_mut().unwrap();
            let mut cur = tx.cursor_dup_write::<tables::StoragesTrie>().unwrap();
            cur.upsert(
                a2,
                &StorageTrieEntry { nibbles: n3.clone(), node: create_test_branch_node() },
            )
            .unwrap();
            tx.commit().unwrap();
        }

        // Initialization #2
        {
            let tx = db.tx().unwrap();
            let job = InitializationJob::new(store.clone(), tx);
            job.initialize_storages_trie(Some(StorageTrieKey::new(a2, StoredNibbles::from(n2.0))))
                .unwrap();
        }

        // Latest must now be (a2,n3)
        let last2 =
            store.initial_state_anchor().expect("get anchor").latest_storage_trie_key.expect("ok");
        assert_eq!(last2.hashed_address, a2);
        assert_eq!(last2.path.0, n3.0);

        // Verify per-address no dupes and stable ordering
        {
            let mut c = store.storage_trie_cursor(a1, 0).unwrap();

            let mut got = Vec::new();

            // next returns the rest
            while let Some((path, _node)) = c.next().unwrap() {
                got.push(path);
            }

            assert_eq!(got, vec![n1.0]);
        }
        {
            let mut c = store.storage_trie_cursor(a2, 0).unwrap();

            let mut got = Vec::new();

            // next returns the rest
            while let Some((path, _node)) = c.next().unwrap() {
                got.push(path);
            }
            assert_eq!(got, vec![n2.0, n3.0]);
        }
    }
}

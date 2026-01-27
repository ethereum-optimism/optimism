//! In-memory implementation of [`OpProofsStore`] for testing purposes

use crate::{
    api::WriteCounts, BlockStateDiff, OpProofsStorageError, OpProofsStorageResult, OpProofsStore,
};
use alloy_eips::{eip1898::BlockWithParent, BlockNumHash};
use alloy_primitives::{B256, U256};
use parking_lot::RwLock;
use reth_db::DatabaseError;
use reth_primitives_traits::Account;
use reth_trie::{
    hashed_cursor::{HashedCursor, HashedStorageCursor},
    trie_cursor::{TrieCursor, TrieStorageCursor},
};
use reth_trie_common::{
    updates::TrieUpdatesSorted, BranchNodeCompact, HashedPostStateSorted, Nibbles,
};
use std::{collections::BTreeMap, sync::Arc};

/// In-memory implementation of [`OpProofsStore`] for testing purposes
#[derive(Debug, Clone)]
pub struct InMemoryProofsStorage {
    /// Shared state across all instances
    inner: Arc<RwLock<InMemoryStorageInner>>,
}

#[derive(Debug, Default)]
struct InMemoryStorageInner {
    /// Account trie branches: (`block_number`, path) -> `branch_node`
    account_branches: BTreeMap<(u64, Nibbles), Option<BranchNodeCompact>>,

    /// Storage trie branches: (`block_number`, `hashed_address`, path) -> `branch_node`
    storage_branches: BTreeMap<(u64, B256, Nibbles), Option<BranchNodeCompact>>,

    /// Hashed accounts: (`block_number`, `hashed_address`) -> account
    hashed_accounts: BTreeMap<(u64, B256), Option<Account>>,

    /// Hashed storages: (`block_number`, `hashed_address`, `hashed_slot`) -> value
    hashed_storages: BTreeMap<(u64, B256, B256), U256>,

    /// Trie updates by block number
    trie_updates: BTreeMap<u64, TrieUpdatesSorted>,

    /// Post state by block number
    post_states: BTreeMap<u64, HashedPostStateSorted>,

    /// Earliest block number and hash
    earliest_block: Option<(u64, B256)>,
}

impl InMemoryStorageInner {
    fn store_trie_updates(
        &mut self,
        block_number: u64,
        block_state_diff: BlockStateDiff,
    ) -> WriteCounts {
        let mut result = WriteCounts::default();

        // Store account branch nodes
        for (path, branch) in block_state_diff.sorted_trie_updates.account_nodes_ref() {
            self.account_branches.insert((block_number, *path), branch.clone());
            result.account_trie_updates_written_total += 1;
        }

        // Store storage branch nodes and removals
        for (address, storage_trie_updates) in
            block_state_diff.sorted_trie_updates.storage_tries_ref()
        {
            // Store storage branch nodes
            for (path, branch) in storage_trie_updates.storage_nodes_ref() {
                self.storage_branches.insert((block_number, *address, *path), branch.clone());
                result.storage_trie_updates_written_total += 1;
            }
        }

        for (address, account) in &block_state_diff.sorted_post_state.accounts {
            self.hashed_accounts.insert((block_number, *address), *account);
            result.hashed_accounts_written_total += 1;
        }

        for (hashed_address, storage) in &block_state_diff.sorted_post_state.storages {
            // Handle wiped storage: iterate all existing values and mark them as deleted
            // This is an expensive operation and should never happen for blocks going forward.
            if storage.wiped {
                // Collect latest values for each slot up to the current block
                let mut slot_to_latest: std::collections::BTreeMap<B256, (u64, U256)> =
                    std::collections::BTreeMap::new();

                for ((block, address, slot), value) in &self.hashed_storages {
                    if *block < block_number && *address == *hashed_address {
                        if let Some((existing_block, _)) = slot_to_latest.get(slot) {
                            if *block > *existing_block {
                                slot_to_latest.insert(*slot, (*block, *value));
                            }
                        } else {
                            slot_to_latest.insert(*slot, (*block, *value));
                        }
                    }
                }

                // Store zero values for all non-zero slots to mark them as deleted
                for (slot, (_, value)) in slot_to_latest {
                    if !value.is_zero() {
                        self.hashed_storages
                            .insert((block_number, *hashed_address, slot), U256::ZERO);
                        result.hashed_storages_written_total += 1;
                    }
                }
            } else {
                for (slot, value) in storage.storage_slots_ref() {
                    self.hashed_storages.insert((block_number, *hashed_address, *slot), *value);
                    result.hashed_storages_written_total += 1;
                }
            }
        }

        self.trie_updates.insert(block_number, block_state_diff.sorted_trie_updates.clone());
        self.post_states.insert(block_number, block_state_diff.sorted_post_state);

        result
    }
}

impl Default for InMemoryProofsStorage {
    fn default() -> Self {
        Self::new()
    }
}

impl InMemoryProofsStorage {
    /// Create a new in-memory op proofs storage instance
    pub fn new() -> Self {
        Self { inner: Arc::new(RwLock::new(InMemoryStorageInner::default())) }
    }
}

/// In-memory implementation of [`TrieCursor`].
#[derive(Debug)]
pub struct InMemoryTrieCursor {
    /// inner storage reference
    inner: Arc<RwLock<InMemoryStorageInner>>,
    /// Hashed address being queried
    hashed_address: Option<B256>,
    /// max block number for the cursor
    max_block_number: u64,
    /// Current position in the iteration (-1 means not positioned yet)
    position: isize,

    /// Whether the entries have been populated
    is_populated: bool,
    /// Sorted entries that match the query parameters
    entries: Vec<(Nibbles, BranchNodeCompact)>,
}

impl InMemoryTrieCursor {
    const fn new(
        inner: Arc<RwLock<InMemoryStorageInner>>,
        hashed_address: Option<B256>,
        max_block_number: u64,
    ) -> Self {
        Self {
            inner,
            hashed_address,
            max_block_number,
            position: -1,

            is_populated: false,
            entries: Vec::new(),
        }
    }

    fn ensure_entries_populated(&mut self) -> Result<(), DatabaseError> {
        if self.is_populated {
            return Ok(());
        }

        let storage = self.inner.try_read().ok_or(OpProofsStorageError::TryLockError)?;

        // Common logic: collect latest values for each path
        let mut path_to_latest: std::collections::BTreeMap<
            Nibbles,
            (u64, Option<BranchNodeCompact>),
        > = std::collections::BTreeMap::new();

        let mut collected_entries: Vec<(Nibbles, BranchNodeCompact)> =
            if let Some(addr) = self.hashed_address {
                // Storage trie cursor
                for ((block, address, path), branch) in &storage.storage_branches {
                    if *block <= self.max_block_number && *address == addr {
                        if let Some((existing_block, _)) = path_to_latest.get(path) {
                            if *block > *existing_block {
                                path_to_latest.insert(*path, (*block, branch.clone()));
                            }
                        } else {
                            path_to_latest.insert(*path, (*block, branch.clone()));
                        }
                    }
                }

                path_to_latest
                    .into_iter()
                    .filter_map(|(path, (_, branch))| branch.map(|b| (path, b)))
                    .collect()
            } else {
                // Account trie cursor
                for ((block, path), branch) in &storage.account_branches {
                    if *block <= self.max_block_number {
                        if let Some((existing_block, _)) = path_to_latest.get(path) {
                            if *block > *existing_block {
                                path_to_latest.insert(*path, (*block, branch.clone()));
                            }
                        } else {
                            path_to_latest.insert(*path, (*block, branch.clone()));
                        }
                    }
                }

                path_to_latest
                    .into_iter()
                    .filter_map(|(path, (_, branch))| branch.map(|b| (path, b)))
                    .collect()
            };

        // Sort by path for consistent ordering
        collected_entries.sort_by_key(|(a, _)| *a);
        self.entries = collected_entries;
        self.is_populated = true;
        Ok(())
    }
}

impl TrieCursor for InMemoryTrieCursor {
    fn seek_exact(
        &mut self,
        path: Nibbles,
    ) -> Result<Option<(Nibbles, BranchNodeCompact)>, DatabaseError> {
        self.ensure_entries_populated()?;

        if let Some(pos) = self.entries.iter().position(|(p, _)| *p == path) {
            self.position = pos as isize;
            Ok(Some(self.entries[pos].clone()))
        } else {
            Ok(None)
        }
    }

    fn seek(
        &mut self,
        path: Nibbles,
    ) -> Result<Option<(Nibbles, BranchNodeCompact)>, DatabaseError> {
        self.ensure_entries_populated()?;

        if let Some(pos) = self.entries.iter().position(|(p, _)| *p >= path) {
            self.position = pos as isize;
            Ok(Some(self.entries[pos].clone()))
        } else {
            Ok(None)
        }
    }

    fn next(&mut self) -> Result<Option<(Nibbles, BranchNodeCompact)>, DatabaseError> {
        self.ensure_entries_populated()?;

        self.position += 1;
        if self.position >= 0 && (self.position as usize) < self.entries.len() {
            Ok(Some(self.entries[self.position as usize].clone()))
        } else {
            Ok(None)
        }
    }

    fn current(&mut self) -> Result<Option<Nibbles>, DatabaseError> {
        self.ensure_entries_populated()?;

        if self.position >= 0 && (self.position as usize) < self.entries.len() {
            Ok(Some(self.entries[self.position as usize].0))
        } else {
            Ok(None)
        }
    }

    fn reset(&mut self) {
        self.position = -1;
    }
}

impl TrieStorageCursor for InMemoryTrieCursor {
    fn set_hashed_address(&mut self, hashed_address: B256) {
        self.hashed_address = Some(hashed_address);
        self.is_populated = false;
        self.entries.clear();
        self.reset();
    }
}

/// In-memory implementation of [`HashedCursor`] for storage slots
#[derive(Debug)]
pub struct InMemoryStorageCursor {
    /// inner storage reference
    inner: Arc<RwLock<InMemoryStorageInner>>,
    /// hashed address for which the cursor is iterating
    hashed_address: B256,
    /// max block number for the cursor
    max_block_number: u64,
    /// Current position in the iteration (-1 means not positioned yet)
    position: isize,

    /// Whether the entries have been populated
    is_populated: bool,
    /// Sorted entries that match the query parameters
    entries: Vec<(B256, U256)>,
}

impl InMemoryStorageCursor {
    const fn new(
        storage: Arc<RwLock<InMemoryStorageInner>>,
        hashed_address: B256,
        max_block_number: u64,
    ) -> Self {
        Self {
            inner: storage,
            hashed_address,
            max_block_number,
            position: -1,

            is_populated: false,
            entries: Vec::new(),
        }
    }

    fn ensure_entries_populated(&mut self) -> Result<(), DatabaseError> {
        if self.is_populated {
            return Ok(());
        }

        let storage = self.inner.try_read().ok_or(OpProofsStorageError::TryLockError)?;

        // Collect latest values for each slot
        let mut slot_to_latest: std::collections::BTreeMap<B256, (u64, U256)> =
            std::collections::BTreeMap::new();

        for ((block, address, slot), value) in &storage.hashed_storages {
            if *block <= self.max_block_number && *address == self.hashed_address {
                if let Some((existing_block, _)) = slot_to_latest.get(slot) {
                    if *block > *existing_block {
                        slot_to_latest.insert(*slot, (*block, *value));
                    }
                } else {
                    slot_to_latest.insert(*slot, (*block, *value));
                }
            }
        }

        // Filter out zero values - they represent deleted/empty storage slots
        let mut entries: Vec<(B256, U256)> = slot_to_latest
            .into_iter()
            .filter_map(
                |(slot, (_, value))| {
                    if value.is_zero() {
                        None
                    } else {
                        Some((slot, value))
                    }
                },
            )
            .collect();

        entries.sort_by_key(|(slot, _)| *slot);
        self.entries = entries;
        self.is_populated = true;
        Ok(())
    }
}

impl HashedCursor for InMemoryStorageCursor {
    type Value = U256;

    fn seek(&mut self, key: B256) -> Result<Option<(B256, Self::Value)>, DatabaseError> {
        self.ensure_entries_populated()?;

        if let Some(pos) = self.entries.iter().position(|(k, _)| *k >= key) {
            self.position = pos as isize;
            Ok(Some(self.entries[pos]))
        } else {
            Ok(None)
        }
    }

    fn next(&mut self) -> Result<Option<(B256, Self::Value)>, DatabaseError> {
        self.ensure_entries_populated()?;

        self.position += 1;
        if self.position >= 0 && (self.position as usize) < self.entries.len() {
            Ok(Some(self.entries[self.position as usize]))
        } else {
            Ok(None)
        }
    }

    fn reset(&mut self) {
        self.position = -1;
    }
}

impl HashedStorageCursor for InMemoryStorageCursor {
    fn is_storage_empty(&mut self) -> Result<bool, DatabaseError> {
        Ok(self.seek(B256::ZERO)?.is_none())
    }

    fn set_hashed_address(&mut self, hashed_address: B256) {
        self.hashed_address = hashed_address;
        self.is_populated = false;
        self.entries.clear();
        self.reset();
    }
}

/// In-memory implementation of [`HashedCursor`] for accounts
#[derive(Debug)]
pub struct InMemoryAccountCursor {
    /// Current position in the iteration (-1 means not positioned yet)
    position: isize,
    /// Sorted entries that match the query parameters
    entries: Vec<(B256, Account)>,
}

impl InMemoryAccountCursor {
    fn new(storage: &InMemoryStorageInner, max_block_number: u64) -> Self {
        // Collect latest accounts for each address
        let mut addr_to_latest: std::collections::BTreeMap<B256, (u64, Option<Account>)> =
            std::collections::BTreeMap::new();

        for ((block, address), account) in &storage.hashed_accounts {
            if *block <= max_block_number {
                if let Some((existing_block, _)) = addr_to_latest.get(address) {
                    if *block > *existing_block {
                        addr_to_latest.insert(*address, (*block, *account));
                    }
                } else {
                    addr_to_latest.insert(*address, (*block, *account));
                }
            }
        }

        let mut entries: Vec<(B256, Account)> = addr_to_latest
            .into_iter()
            .filter_map(|(address, (_, account))| account.map(|acc| (address, acc)))
            .collect();

        entries.sort_by_key(|(address, _)| *address);

        Self { position: -1, entries }
    }
}

impl HashedCursor for InMemoryAccountCursor {
    type Value = Account;

    fn seek(&mut self, key: B256) -> Result<Option<(B256, Self::Value)>, DatabaseError> {
        if let Some(pos) = self.entries.iter().position(|(k, _)| *k >= key) {
            self.position = pos as isize;
            Ok(Some(self.entries[pos]))
        } else {
            Ok(None)
        }
    }

    fn next(&mut self) -> Result<Option<(B256, Self::Value)>, DatabaseError> {
        self.position += 1;
        if self.position >= 0 && (self.position as usize) < self.entries.len() {
            Ok(Some(self.entries[self.position as usize]))
        } else {
            Ok(None)
        }
    }

    fn reset(&mut self) {
        // no reset needed
    }
}

impl OpProofsStore for InMemoryProofsStorage {
    type StorageTrieCursor<'tx> = InMemoryTrieCursor;
    type AccountTrieCursor<'tx> = InMemoryTrieCursor;
    type StorageCursor<'tx> = InMemoryStorageCursor;
    type AccountHashedCursor<'tx> = InMemoryAccountCursor;

    fn store_account_branches(
        &self,
        updates: Vec<(Nibbles, Option<BranchNodeCompact>)>,
    ) -> OpProofsStorageResult<()> {
        let mut inner = self.inner.write();

        for (path, branch) in updates {
            inner.account_branches.insert((0, path), branch);
        }

        Ok(())
    }

    fn store_storage_branches(
        &self,
        hashed_address: B256,
        items: Vec<(Nibbles, Option<BranchNodeCompact>)>,
    ) -> OpProofsStorageResult<()> {
        let mut inner = self.inner.write();

        for (path, branch) in items {
            inner.storage_branches.insert((0, hashed_address, path), branch);
        }

        Ok(())
    }

    fn store_hashed_accounts(
        &self,
        accounts: Vec<(B256, Option<Account>)>,
    ) -> OpProofsStorageResult<()> {
        let mut inner = self.inner.write();

        for (address, account) in accounts {
            inner.hashed_accounts.insert((0, address), account);
        }

        Ok(())
    }

    fn store_hashed_storages(
        &self,
        hashed_address: B256,
        storages: Vec<(B256, U256)>,
    ) -> OpProofsStorageResult<()> {
        let mut inner = self.inner.write();

        for (slot, value) in storages {
            inner.hashed_storages.insert((0, hashed_address, slot), value);
        }

        Ok(())
    }

    fn get_earliest_block_number(&self) -> OpProofsStorageResult<Option<(u64, B256)>> {
        let inner = self.inner.read();
        Ok(inner.earliest_block)
    }

    fn get_latest_block_number(&self) -> OpProofsStorageResult<Option<(u64, B256)>> {
        let inner = self.inner.read();
        // Find the latest block number from trie_updates
        let latest_block = inner.trie_updates.keys().max().copied();
        if let Some(block) = latest_block {
            // We don't have a hash stored, so return a default
            Ok(Some((block, B256::ZERO)))
        } else {
            // otherwise return earliest block if set
            Ok(inner.earliest_block)
        }
    }

    fn storage_trie_cursor<'tx>(
        &self,
        hashed_address: B256,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::StorageTrieCursor<'tx>> {
        Ok(InMemoryTrieCursor::new(self.inner.clone(), Some(hashed_address), max_block_number))
    }

    fn account_trie_cursor<'tx>(
        &self,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::AccountTrieCursor<'tx>> {
        Ok(InMemoryTrieCursor::new(self.inner.clone(), None, max_block_number))
    }

    fn storage_hashed_cursor<'tx>(
        &self,
        hashed_address: B256,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::StorageCursor<'tx>> {
        Ok(InMemoryStorageCursor::new(self.inner.clone(), hashed_address, max_block_number))
    }

    fn account_hashed_cursor<'tx>(
        &self,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::AccountHashedCursor<'tx>> {
        let inner = self.inner.try_read().ok_or(OpProofsStorageError::TryLockError)?;
        Ok(InMemoryAccountCursor::new(&inner, max_block_number))
    }

    fn store_trie_updates(
        &self,
        block_ref: BlockWithParent,
        block_state_diff: BlockStateDiff,
    ) -> OpProofsStorageResult<WriteCounts> {
        let mut inner = self.inner.write();

        Ok(inner.store_trie_updates(block_ref.block.number, block_state_diff))
    }

    fn fetch_trie_updates(&self, block_number: u64) -> OpProofsStorageResult<BlockStateDiff> {
        let inner = self.inner.read();

        let trie_updates = inner.trie_updates.get(&block_number).cloned().unwrap_or_default();
        let post_state = inner.post_states.get(&block_number).cloned().unwrap_or_default();

        Ok(BlockStateDiff { sorted_trie_updates: trie_updates, sorted_post_state: post_state })
    }

    fn prune_earliest_state(
        &self,
        new_earliest_block_ref: BlockWithParent,
    ) -> OpProofsStorageResult<WriteCounts> {
        let mut write_counts = WriteCounts::default();
        let mut inner = self.inner.write();
        let new_earliest = new_earliest_block_ref.block.number;

        // 1. Account Branches
        // Identify keys to move (blocks <= new_earliest but > 0)
        let account_keys: Vec<_> = inner
            .account_branches
            .keys()
            .filter(|(block, _)| *block > 0 && *block <= new_earliest)
            .copied()
            .collect();

        for key in account_keys {
            if let Some(branch) = inner.account_branches.remove(&key) {
                // Determine if we should update the base state (block 0)
                // In a simpler model without diff calculation, we just overwrite block 0
                // with the latest version found in the pruned range.
                // Since keys are sorted by block, this logic naturally keeps the latest if we
                // iterate in order, but here we are just grabbing all of them.
                // For correctness in BTreeMap iteration order (which is sorted):
                inner.account_branches.insert((0, key.1), branch);
                write_counts.account_trie_updates_written_total += 1;
            }
        }

        // 2. Storage Branches
        let storage_keys: Vec<_> = inner
            .storage_branches
            .keys()
            .filter(|(block, _, _)| *block > 0 && *block <= new_earliest)
            .copied()
            .collect();

        for key in storage_keys {
            if let Some(branch) = inner.storage_branches.remove(&key) {
                inner.storage_branches.insert((0, key.1, key.2), branch);
                write_counts.storage_trie_updates_written_total += 1;
            }
        }

        // 3. Hashed Accounts
        let acc_keys: Vec<_> = inner
            .hashed_accounts
            .keys()
            .filter(|(block, _)| *block > 0 && *block <= new_earliest)
            .copied()
            .collect();

        for key in acc_keys {
            if let Some(acc) = inner.hashed_accounts.remove(&key) {
                inner.hashed_accounts.insert((0, key.1), acc);
                write_counts.hashed_accounts_written_total += 1;
            }
        }

        // 4. Hashed Storages
        let stor_keys: Vec<_> = inner
            .hashed_storages
            .keys()
            .filter(|(block, _, _)| *block > 0 && *block <= new_earliest)
            .copied()
            .collect();

        for key in stor_keys {
            if let Some(val) = inner.hashed_storages.remove(&key) {
                inner.hashed_storages.insert((0, key.1, key.2), val);
                write_counts.hashed_storages_written_total += 1;
            }
        }

        // Update earliest block pointer
        if let Some((_, hash)) = inner.earliest_block {
            inner.earliest_block = Some((new_earliest, hash));
        }

        // 5. Cleanup Metadata
        inner.trie_updates.retain(|block, _| *block > new_earliest);
        inner.post_states.retain(|block, _| *block > new_earliest);

        Ok(write_counts)
    }

    fn unwind_history(&self, unwind_upto_block: BlockWithParent) -> OpProofsStorageResult<()> {
        let mut inner = self.inner.write();
        let unwind_upto_block_number = unwind_upto_block.block.number - 1;

        // Remove all updates after unwind_upto_block_number
        inner.trie_updates.retain(|block, _| *block <= unwind_upto_block_number);
        inner.post_states.retain(|block, _| *block <= unwind_upto_block_number);
        inner.account_branches.retain(|(block, _), _| *block <= unwind_upto_block_number);
        inner.storage_branches.retain(|(block, _, _), _| *block <= unwind_upto_block_number);
        inner.hashed_accounts.retain(|(block, _), _| *block <= unwind_upto_block_number);
        inner.hashed_storages.retain(|(block, _, _), _| *block <= unwind_upto_block_number);

        Ok(())
    }

    fn replace_updates(
        &self,
        latest_common_block: BlockNumHash,
        blocks_to_add: Vec<(BlockWithParent, BlockStateDiff)>,
    ) -> OpProofsStorageResult<()> {
        let mut inner = self.inner.write();
        let latest_common_block_number = latest_common_block.number;

        // Remove all updates after latest_common_block_number
        inner.trie_updates.retain(|block, _| *block <= latest_common_block_number);
        inner.post_states.retain(|block, _| *block <= latest_common_block_number);
        inner.account_branches.retain(|(block, _), _| *block <= latest_common_block_number);
        inner.storage_branches.retain(|(block, _, _), _| *block <= latest_common_block_number);
        inner.hashed_accounts.retain(|(block, _), _| *block <= latest_common_block_number);
        inner.hashed_storages.retain(|(block, _, _), _| *block <= latest_common_block_number);

        for (block, block_state_diff) in blocks_to_add {
            inner.store_trie_updates(block.block.number, block_state_diff);
        }

        Ok(())
    }

    fn set_earliest_block_number(
        &self,
        block_number: u64,
        hash: B256,
    ) -> OpProofsStorageResult<()> {
        let mut inner = self.inner.write();
        inner.earliest_block = Some((block_number, hash));
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::OpProofsStorageError;
    use alloy_eips::NumHash;
    use alloy_primitives::U256;
    use reth_primitives_traits::Account;

    #[test]
    fn test_in_memory_storage_basic_operations() -> Result<(), OpProofsStorageError> {
        let storage = InMemoryProofsStorage::new();

        // Test setting earliest block
        let block_hash = B256::random();
        storage.set_earliest_block_number(1, block_hash)?;
        let earliest = storage.get_earliest_block_number()?;
        assert_eq!(earliest, Some((1, block_hash)));

        // Test storing and retrieving accounts
        let account = Account { nonce: 1, balance: U256::from(100), bytecode_hash: None };
        let hashed_address = B256::random();

        storage.store_hashed_accounts(vec![(hashed_address, Some(account))])?;

        let _cursor = storage.account_hashed_cursor(10)?;
        // Note: cursor testing would require more complex setup with proper seek/next operations

        Ok(())
    }

    #[test]
    fn test_trie_updates_storage() -> Result<(), OpProofsStorageError> {
        let storage = InMemoryProofsStorage::new();

        let sorted_trie_updates = TrieUpdatesSorted::default();
        let sorted_post_state = HashedPostStateSorted::default();
        let block_state_diff = BlockStateDiff {
            sorted_trie_updates: sorted_trie_updates.clone(),
            sorted_post_state: sorted_post_state.clone(),
        };

        const BLOCK: BlockWithParent =
            BlockWithParent::new(B256::ZERO, NumHash::new(5, B256::ZERO));
        storage.store_trie_updates(BLOCK, block_state_diff)?;

        let retrieved_diff = storage.fetch_trie_updates(BLOCK.block.number)?;
        assert_eq!(retrieved_diff.sorted_trie_updates, sorted_trie_updates);
        assert_eq!(retrieved_diff.sorted_post_state, sorted_post_state);

        Ok(())
    }
}

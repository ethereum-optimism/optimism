//! In-memory implementation of [`OpProofsStorage`] for testing purposes

use crate::{
    BlockStateDiff, OpProofsHashedCursor, OpProofsStorage, OpProofsStorageError,
    OpProofsStorageResult, OpProofsTrieCursor,
};
use alloy_primitives::{map::HashMap, B256, U256};
use reth_primitives_traits::Account;
use reth_trie::{updates::TrieUpdates, BranchNodeCompact, HashedPostState, Nibbles};
use std::{collections::BTreeMap, sync::Arc};
use tokio::sync::RwLock;

/// In-memory implementation of [`OpProofsStorage`] for testing purposes
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
    trie_updates: BTreeMap<u64, TrieUpdates>,

    /// Post state by block number
    post_states: BTreeMap<u64, HashedPostState>,

    /// Earliest block number and hash
    earliest_block: Option<(u64, B256)>,
}

impl InMemoryStorageInner {
    fn store_trie_updates(&mut self, block_number: u64, block_state_diff: BlockStateDiff) {
        // Store account branch nodes
        for (path, branch) in block_state_diff.trie_updates.account_nodes_ref() {
            self.account_branches.insert((block_number, *path), Some(branch.clone()));
        }

        // Store removed account nodes
        let account_removals = block_state_diff
            .trie_updates
            .removed_nodes_ref()
            .iter()
            .filter_map(|n| {
                (!block_state_diff.trie_updates.account_nodes_ref().contains_key(n))
                    .then_some((n, None))
            })
            .collect::<Vec<_>>();

        for (path, branch) in account_removals {
            self.account_branches.insert((block_number, *path), branch);
        }

        // Store storage branch nodes and removals
        for (address, storage_trie_updates) in block_state_diff.trie_updates.storage_tries_ref() {
            // Store storage branch nodes
            for (path, branch) in storage_trie_updates.storage_nodes_ref() {
                self.storage_branches.insert((block_number, *address, *path), Some(branch.clone()));
            }

            // Store removed storage nodes
            let storage_removals = storage_trie_updates
                .removed_nodes_ref()
                .iter()
                .filter_map(|n| {
                    (!storage_trie_updates.storage_nodes_ref().contains_key(n)).then_some((n, None))
                })
                .collect::<Vec<_>>();

            for (path, branch) in storage_removals {
                self.storage_branches.insert((block_number, *address, *path), branch);
            }
        }

        for (address, account) in &block_state_diff.post_state.accounts {
            self.hashed_accounts.insert((block_number, *address), *account);
        }

        for (hashed_address, storage) in &block_state_diff.post_state.storages {
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
                    }
                }
            } else {
                for (slot, value) in &storage.storage {
                    self.hashed_storages.insert((block_number, *hashed_address, *slot), *value);
                }
            }
        }

        self.trie_updates.insert(block_number, block_state_diff.trie_updates.clone());
        self.post_states.insert(block_number, block_state_diff.post_state.clone());
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

/// In-memory implementation of `OpProofsTrieCursor`
#[derive(Debug)]
pub struct InMemoryTrieCursor {
    /// Current position in the iteration (-1 means not positioned yet)
    position: isize,
    /// Sorted entries that match the query parameters
    entries: Vec<(Nibbles, BranchNodeCompact)>,
}

impl InMemoryTrieCursor {
    fn new(
        storage: &InMemoryStorageInner,
        hashed_address: Option<B256>,
        max_block_number: u64,
    ) -> Self {
        // Common logic: collect latest values for each path
        let mut path_to_latest: std::collections::BTreeMap<
            Nibbles,
            (u64, Option<BranchNodeCompact>),
        > = std::collections::BTreeMap::new();

        let mut collected_entries: Vec<(Nibbles, BranchNodeCompact)> =
            if let Some(addr) = hashed_address {
                // Storage trie cursor
                for ((block, address, path), branch) in &storage.storage_branches {
                    if *block <= max_block_number && *address == addr {
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
                    if *block <= max_block_number {
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
        collected_entries.sort_by(|(a, _), (b, _)| a.cmp(b));

        Self { position: -1, entries: collected_entries }
    }
}

impl OpProofsTrieCursor for InMemoryTrieCursor {
    fn seek_exact(
        &mut self,
        path: Nibbles,
    ) -> OpProofsStorageResult<Option<(Nibbles, BranchNodeCompact)>> {
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
    ) -> OpProofsStorageResult<Option<(Nibbles, BranchNodeCompact)>> {
        if let Some(pos) = self.entries.iter().position(|(p, _)| *p >= path) {
            self.position = pos as isize;
            Ok(Some(self.entries[pos].clone()))
        } else {
            Ok(None)
        }
    }

    fn next(&mut self) -> OpProofsStorageResult<Option<(Nibbles, BranchNodeCompact)>> {
        self.position += 1;
        if self.position >= 0 && (self.position as usize) < self.entries.len() {
            Ok(Some(self.entries[self.position as usize].clone()))
        } else {
            Ok(None)
        }
    }

    fn current(&mut self) -> OpProofsStorageResult<Option<Nibbles>> {
        if self.position >= 0 && (self.position as usize) < self.entries.len() {
            Ok(Some(self.entries[self.position as usize].0))
        } else {
            Ok(None)
        }
    }
}

/// In-memory implementation of `OpProofsHashedCursor` for storage slots
#[derive(Debug)]
pub struct InMemoryStorageCursor {
    /// Current position in the iteration (-1 means not positioned yet)
    position: isize,
    /// Sorted entries that match the query parameters
    entries: Vec<(B256, U256)>,
}

impl InMemoryStorageCursor {
    fn new(storage: &InMemoryStorageInner, hashed_address: B256, max_block_number: u64) -> Self {
        // Collect latest values for each slot
        let mut slot_to_latest: std::collections::BTreeMap<B256, (u64, U256)> =
            std::collections::BTreeMap::new();

        for ((block, address, slot), value) in &storage.hashed_storages {
            if *block <= max_block_number && *address == hashed_address {
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

        Self { position: -1, entries }
    }
}

impl OpProofsHashedCursor for InMemoryStorageCursor {
    type Value = U256;

    fn seek(&mut self, key: B256) -> OpProofsStorageResult<Option<(B256, Self::Value)>> {
        if let Some(pos) = self.entries.iter().position(|(k, _)| *k >= key) {
            self.position = pos as isize;
            Ok(Some(self.entries[pos]))
        } else {
            Ok(None)
        }
    }

    fn next(&mut self) -> OpProofsStorageResult<Option<(B256, Self::Value)>> {
        self.position += 1;
        if self.position >= 0 && (self.position as usize) < self.entries.len() {
            Ok(Some(self.entries[self.position as usize]))
        } else {
            Ok(None)
        }
    }
}

/// In-memory implementation of [`OpProofsHashedCursor`] for accounts
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

impl OpProofsHashedCursor for InMemoryAccountCursor {
    type Value = Account;

    fn seek(&mut self, key: B256) -> OpProofsStorageResult<Option<(B256, Self::Value)>> {
        if let Some(pos) = self.entries.iter().position(|(k, _)| *k >= key) {
            self.position = pos as isize;
            Ok(Some(self.entries[pos]))
        } else {
            Ok(None)
        }
    }

    fn next(&mut self) -> OpProofsStorageResult<Option<(B256, Self::Value)>> {
        self.position += 1;
        if self.position >= 0 && (self.position as usize) < self.entries.len() {
            Ok(Some(self.entries[self.position as usize]))
        } else {
            Ok(None)
        }
    }
}

impl OpProofsStorage for InMemoryProofsStorage {
    type StorageTrieCursor<'tx> = InMemoryTrieCursor;
    type AccountTrieCursor<'tx> = InMemoryTrieCursor;
    type StorageCursor = InMemoryStorageCursor;
    type AccountHashedCursor = InMemoryAccountCursor;

    async fn store_account_branches(
        &self,
        updates: Vec<(Nibbles, Option<BranchNodeCompact>)>,
    ) -> OpProofsStorageResult<()> {
        let mut inner = self.inner.write().await;

        for (path, branch) in updates {
            inner.account_branches.insert((0, path), branch);
        }

        Ok(())
    }

    async fn store_storage_branches(
        &self,
        hashed_address: B256,
        items: Vec<(Nibbles, Option<BranchNodeCompact>)>,
    ) -> OpProofsStorageResult<()> {
        let mut inner = self.inner.write().await;

        for (path, branch) in items {
            inner.storage_branches.insert((0, hashed_address, path), branch);
        }

        Ok(())
    }

    async fn store_hashed_accounts(
        &self,
        accounts: Vec<(B256, Option<Account>)>,
    ) -> OpProofsStorageResult<()> {
        let mut inner = self.inner.write().await;

        for (address, account) in accounts {
            inner.hashed_accounts.insert((0, address), account);
        }

        Ok(())
    }

    async fn store_hashed_storages(
        &self,
        hashed_address: B256,
        storages: Vec<(B256, U256)>,
    ) -> OpProofsStorageResult<()> {
        let mut inner = self.inner.write().await;

        for (slot, value) in storages {
            inner.hashed_storages.insert((0, hashed_address, slot), value);
        }

        Ok(())
    }

    async fn get_earliest_block_number(&self) -> OpProofsStorageResult<Option<(u64, B256)>> {
        let inner = self.inner.read().await;
        Ok(inner.earliest_block)
    }

    async fn get_latest_block_number(&self) -> OpProofsStorageResult<Option<(u64, B256)>> {
        let inner = self.inner.read().await;
        // Find the latest block number from trie_updates
        let latest_block = inner.trie_updates.keys().max().copied();
        if let Some(block) = latest_block {
            // We don't have a hash stored, so return a default
            Ok(Some((block, B256::ZERO)))
        } else {
            Ok(None)
        }
    }

    fn storage_trie_cursor<'tx>(
        &self,
        hashed_address: B256,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::StorageTrieCursor<'tx>> {
        // For synchronous methods, we need to try_read() and handle potential blocking
        let inner = self
            .inner
            .try_read()
            .map_err(|_| OpProofsStorageError::Other(eyre::eyre!("Failed to acquire read lock")))?;
        Ok(InMemoryTrieCursor::new(&inner, Some(hashed_address), max_block_number))
    }

    fn account_trie_cursor<'tx>(
        &self,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::AccountTrieCursor<'tx>> {
        let inner = self
            .inner
            .try_read()
            .map_err(|_| OpProofsStorageError::Other(eyre::eyre!("Failed to acquire read lock")))?;
        Ok(InMemoryTrieCursor::new(&inner, None, max_block_number))
    }

    fn storage_hashed_cursor(
        &self,
        hashed_address: B256,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::StorageCursor> {
        let inner = self
            .inner
            .try_read()
            .map_err(|_| OpProofsStorageError::Other(eyre::eyre!("Failed to acquire read lock")))?;
        Ok(InMemoryStorageCursor::new(&inner, hashed_address, max_block_number))
    }

    fn account_hashed_cursor(
        &self,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::AccountHashedCursor> {
        let inner = self
            .inner
            .try_read()
            .map_err(|_| OpProofsStorageError::Other(eyre::eyre!("Failed to acquire read lock")))?;
        Ok(InMemoryAccountCursor::new(&inner, max_block_number))
    }

    async fn store_trie_updates(
        &self,
        block_number: u64,
        block_state_diff: BlockStateDiff,
    ) -> OpProofsStorageResult<()> {
        let mut inner = self.inner.write().await;

        inner.store_trie_updates(block_number, block_state_diff);

        Ok(())
    }

    async fn fetch_trie_updates(&self, block_number: u64) -> OpProofsStorageResult<BlockStateDiff> {
        let inner = self.inner.read().await;

        let trie_updates = inner.trie_updates.get(&block_number).cloned().unwrap_or_default();
        let post_state = inner.post_states.get(&block_number).cloned().unwrap_or_default();

        Ok(BlockStateDiff { trie_updates, post_state })
    }

    async fn prune_earliest_state(
        &self,
        new_earliest_block_number: u64,
        diff: BlockStateDiff,
    ) -> OpProofsStorageResult<()> {
        let mut inner = self.inner.write().await;

        let branches_diff = diff.trie_updates;
        let leaves_diff = diff.post_state;

        // Apply branch updates to the earliest state (block 0)
        for (path, branch) in &branches_diff.account_nodes {
            inner.account_branches.insert((0, *path), Some(branch.clone()));
        }

        // Remove pruned account branches
        for path in &branches_diff.removed_nodes {
            inner.account_branches.remove(&(0, *path));
        }

        // Apply storage trie updates
        for (hashed_address, storage_updates) in &branches_diff.storage_tries {
            for (path, branch) in &storage_updates.storage_nodes {
                inner.storage_branches.insert((0, *hashed_address, *path), Some(branch.clone()));
            }

            for path in &storage_updates.removed_nodes {
                inner.storage_branches.remove(&(0, *hashed_address, *path));
            }
        }

        // Apply account updates
        for (hashed_address, account) in &leaves_diff.accounts {
            inner.hashed_accounts.insert((0, *hashed_address), *account);
        }

        // Apply storage updates
        for (hashed_address, storage) in &leaves_diff.storages {
            for (slot, value) in &storage.storage {
                inner.hashed_storages.insert((0, *hashed_address, *slot), *value);
            }
        }

        // Update earliest block number if we have one
        if let Some((_, hash)) = inner.earliest_block {
            inner.earliest_block = Some((new_earliest_block_number, hash));
        }

        // Remove all data for blocks before new_earliest_block_number (except block 0)
        inner
            .account_branches
            .retain(|(block, _), _| *block == 0 || *block >= new_earliest_block_number);
        inner
            .storage_branches
            .retain(|(block, _, _), _| *block == 0 || *block >= new_earliest_block_number);
        inner
            .hashed_accounts
            .retain(|(block, _), _| *block == 0 || *block >= new_earliest_block_number);
        inner
            .hashed_storages
            .retain(|(block, _, _), _| *block == 0 || *block >= new_earliest_block_number);
        inner.trie_updates.retain(|block, _| *block >= new_earliest_block_number);
        inner.post_states.retain(|block, _| *block >= new_earliest_block_number);

        Ok(())
    }

    async fn replace_updates(
        &self,
        latest_common_block_number: u64,
        blocks_to_add: HashMap<u64, BlockStateDiff>,
    ) -> OpProofsStorageResult<()> {
        let mut inner = self.inner.write().await;

        // Remove all updates after latest_common_block_number
        inner.trie_updates.retain(|block, _| *block <= latest_common_block_number);
        inner.post_states.retain(|block, _| *block <= latest_common_block_number);
        inner.account_branches.retain(|(block, _), _| *block <= latest_common_block_number);
        inner.storage_branches.retain(|(block, _, _), _| *block <= latest_common_block_number);
        inner.hashed_accounts.retain(|(block, _), _| *block <= latest_common_block_number);
        inner.hashed_storages.retain(|(block, _, _), _| *block <= latest_common_block_number);

        for (block_number, block_state_diff) in blocks_to_add {
            inner.store_trie_updates(block_number, block_state_diff);
        }

        Ok(())
    }

    async fn set_earliest_block_number(
        &self,
        block_number: u64,
        hash: B256,
    ) -> OpProofsStorageResult<()> {
        let mut inner = self.inner.write().await;
        inner.earliest_block = Some((block_number, hash));
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::U256;
    use reth_primitives_traits::Account;

    #[tokio::test]
    async fn test_in_memory_storage_basic_operations() -> Result<(), OpProofsStorageError> {
        let storage = InMemoryProofsStorage::new();

        // Test setting earliest block
        let block_hash = B256::random();
        storage.set_earliest_block_number(1, block_hash).await?;
        let earliest = storage.get_earliest_block_number().await?;
        assert_eq!(earliest, Some((1, block_hash)));

        // Test storing and retrieving accounts
        let account = Account { nonce: 1, balance: U256::from(100), bytecode_hash: None };
        let hashed_address = B256::random();

        storage.store_hashed_accounts(vec![(hashed_address, Some(account))]).await?;

        let _cursor = storage.account_hashed_cursor(10)?;
        // Note: cursor testing would require more complex setup with proper seek/next operations

        Ok(())
    }

    #[tokio::test]
    async fn test_trie_updates_storage() -> Result<(), OpProofsStorageError> {
        let storage = InMemoryProofsStorage::new();

        let trie_updates = TrieUpdates::default();
        let post_state = HashedPostState::default();
        let block_state_diff =
            BlockStateDiff { trie_updates: trie_updates.clone(), post_state: post_state.clone() };

        storage.store_trie_updates(5, block_state_diff).await?;

        let retrieved_diff = storage.fetch_trie_updates(5).await?;
        assert_eq!(retrieved_diff.trie_updates, trie_updates);
        assert_eq!(retrieved_diff.post_state, post_state);

        Ok(())
    }
}

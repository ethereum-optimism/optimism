use super::{BlockNumberHash, ProofWindow, ProofWindowKey};
use crate::{
    db::{
        cursor::Dup,
        models::{
            AccountTrieHistory, BlockChangeSet, ChangeSet, HashedAccountHistory,
            HashedStorageHistory, HashedStorageKey, MaybeDeleted, StorageTrieHistory,
            StorageTrieKey, StorageValue, VersionedValue,
        },
        MdbxAccountCursor, MdbxStorageCursor, MdbxTrieCursor,
    },
    BlockStateDiff, OpProofsHashedCursorRO, OpProofsStorageError, OpProofsStorageResult,
    OpProofsStore, OpProofsTrieCursorRO,
};
use alloy_eips::eip1898::BlockWithParent;
use alloy_primitives::{map::HashMap, B256, U256};
use itertools::Itertools;
use reth_db::{
    cursor::{DbCursorRO, DbCursorRW, DbDupCursorRO, DbDupCursorRW},
    mdbx::{init_db_for, DatabaseArguments},
    transaction::{DbTx, DbTxMut},
    Database, DatabaseEnv, DatabaseError,
};
use reth_primitives_traits::Account;
use reth_trie::{
    updates::StorageTrieUpdates, BranchNodeCompact, HashedStorage, Nibbles, StoredNibbles,
};
use std::path::Path;

/// MDBX implementation of [`OpProofsStore`].
#[derive(Debug)]
pub struct MdbxProofsStorage {
    env: DatabaseEnv,
}

impl MdbxProofsStorage {
    /// Creates a new [`MdbxProofsStorage`] instance with the given path.
    pub fn new(path: &Path) -> Result<Self, OpProofsStorageError> {
        let env = init_db_for::<_, super::models::Tables>(path, DatabaseArguments::default())
            .map_err(OpProofsStorageError::Other)?;
        Ok(Self { env })
    }

    async fn get_block_number_hash(
        &self,
        key: ProofWindowKey,
    ) -> OpProofsStorageResult<Option<(u64, B256)>> {
        let result = self.env.view(|tx| {
            let mut cursor = tx.cursor_read::<ProofWindow>().ok()?;
            let value = cursor.seek_exact(key).ok()?;
            value.map(|(_, val)| (val.number(), *val.hash()))
        });
        Ok(result?)
    }

    async fn set_earliest_block_number_hash(
        &self,
        block_number: u64,
        hash: B256,
    ) -> OpProofsStorageResult<()> {
        let _ = self.env.update(|tx| {
            Self::inner_set_earliest_block_number(tx, block_number, hash)?;
            Ok::<(), DatabaseError>(())
        })?;
        Ok(())
    }

    /// Internal helper to set earliest block number hash within an existing transaction
    fn inner_set_earliest_block_number(
        tx: &(impl DbTxMut + DbTx),
        block_number: u64,
        hash: B256,
    ) -> OpProofsStorageResult<()> {
        let mut cursor = tx.cursor_write::<ProofWindow>()?;
        cursor.upsert(ProofWindowKey::EarliestBlock, &BlockNumberHash::new(block_number, hash))?;
        Ok(())
    }

    fn store_trie_updates_inner(
        &self,
        tx: &<DatabaseEnv as Database>::TXMut,
        block_ref: BlockWithParent,
        block_state_diff: BlockStateDiff,
    ) -> OpProofsStorageResult<()> {
        let block_number = block_ref.block.number;
        // TODO: refactor the code - remove sorting out of tx

        let sorted_trie_updates = block_state_diff.trie_updates.into_sorted();
        //  Sorted list of updated and removed account nodes
        let sorted_account_nodes = sorted_trie_updates.account_nodes;

        //  Sorted list of updated and removed storage nodes
        let sorted_storage_nodes = sorted_trie_updates
            .storage_tries
            .into_iter()
            .sorted_by_key(|(hashed_address, _)| *hashed_address)
            .collect::<Vec<_>>();

        let sorted_post_state = block_state_diff.post_state.into_sorted();
        let sorted_accounts = sorted_post_state.accounts().accounts_sorted();

        let sorted_storage = sorted_post_state
            .account_storages()
            .iter()
            .sorted_by_key(|(hashed_address, _)| *hashed_address)
            .collect::<Vec<_>>();

        // check latest stored block is the parent of incoming block
        let latest_block_hash =
            if let Some(bn_hash) = tx.get::<ProofWindow>(ProofWindowKey::LatestBlock)? {
                *bn_hash.hash()
            } else if let Some(bn_hash) = tx.get::<ProofWindow>(ProofWindowKey::EarliestBlock)? {
                *bn_hash.hash()
            } else {
                B256::ZERO
            };

        if latest_block_hash != block_ref.parent {
            return Err(OpProofsStorageError::OutOfOrder {
                block_number,
                parent_block_hash: block_ref.parent,
                latest_block_hash,
            });
        }

        let account_trie_len = sorted_account_nodes.len();
        let storage_trie_len = sorted_storage_nodes.len();
        let hashed_account_len = sorted_accounts.size_hint().0;
        let hashed_storage_len = sorted_storage.len();

        // Preparing the entries for the `BlockChangeSet` table
        let mut account_trie_keys = Vec::<StoredNibbles>::with_capacity(account_trie_len);
        let mut storage_trie_keys = Vec::<StorageTrieKey>::with_capacity(storage_trie_len);
        let mut hashed_account_keys = Vec::<B256>::with_capacity(hashed_account_len);
        let mut hashed_storage_keys = Vec::<HashedStorageKey>::with_capacity(hashed_storage_len);

        let mut account_trie_cursor = tx.new_cursor::<AccountTrieHistory>()?;
        for (path, node) in sorted_account_nodes {
            let key: StoredNibbles = path.into();
            let vv = VersionedValue { block_number, value: MaybeDeleted(node) };
            account_trie_cursor.append_dup(key.clone(), vv)?;

            account_trie_keys.push(key);
        }

        let mut storage_trie_cursor = tx.new_cursor::<StorageTrieHistory>()?;
        for (hashed_address, nodes) in sorted_storage_nodes {
            // Handle wiped - mark all storage trie as deleted at the current block number
            if nodes.is_deleted {
                // Yet to have any update for the current block number - So just using up to
                // previous block number
                let mut storage_trie_cursor_ro =
                    self.storage_trie_cursor(hashed_address, block_number - 1)?;
                while let Some((path, _vv)) = storage_trie_cursor_ro.next()? {
                    // Mark deleted at current block
                    let del = VersionedValue { block_number, value: MaybeDeleted(None) };
                    storage_trie_cursor
                        .append_dup(StorageTrieKey::new(hashed_address, path.into()), del)?;
                }
                // Skip any further processing for this hashed_address
                continue;
            }
            for (path, node) in nodes.storage_nodes {
                let key = StorageTrieKey::new(hashed_address, path.into());
                let vv = VersionedValue { block_number, value: MaybeDeleted(node) };
                storage_trie_cursor.append_dup(key.clone(), vv)?;

                storage_trie_keys.push(key);
            }
        }

        let mut account_cursor = tx.new_cursor::<HashedAccountHistory>()?;
        for (hashed_address, account) in sorted_accounts {
            let vv = VersionedValue { block_number, value: MaybeDeleted(account) };
            account_cursor.append_dup(hashed_address, vv)?;

            hashed_account_keys.push(hashed_address);
        }

        let mut storage_cursor = tx.new_cursor::<HashedStorageHistory>()?;
        for (hashed_address, storage) in sorted_storage {
            // Handle wiped - mark all storage slots as deleted at the current block number
            if storage.is_wiped() {
                // Yet to have any update for the current block number - So just using up to
                // previous block number
                let mut storage_hashed_cursor =
                    self.storage_hashed_cursor(*hashed_address, block_number - 1)?;
                while let Some((key, _vv)) = storage_hashed_cursor.next()? {
                    // Mark deleted at current block
                    let del = VersionedValue { block_number, value: MaybeDeleted(None) };
                    storage_cursor.append_dup(HashedStorageKey::new(*hashed_address, key), del)?;
                }
                // Skip any further processing for this hashed_address
                continue;
            }
            let storage_items = storage.storage_slots_sorted().collect::<Vec<_>>();
            for (storage_key, storage_value) in storage_items {
                let vv = VersionedValue {
                    block_number,
                    value: MaybeDeleted(Some(StorageValue(storage_value))),
                };
                let key = HashedStorageKey::new(*hashed_address, storage_key);
                storage_cursor.append_dup(key.clone(), vv)?;

                hashed_storage_keys.push(key);
            }
        }

        // Cursor for recording all changes made in this block for all history tables
        let mut change_set_cursor = tx.new_cursor::<BlockChangeSet>()?;
        change_set_cursor.append(
            block_number,
            &ChangeSet {
                account_trie_keys,
                storage_trie_keys,
                hashed_account_keys,
                hashed_storage_keys,
            },
        )?;

        // update proof window latest block
        let mut proof_window_cursor = tx.new_cursor::<ProofWindow>()?;
        proof_window_cursor.append(
            ProofWindowKey::LatestBlock,
            &BlockNumberHash::new(block_number, block_ref.block.hash),
        )?;
        Ok(())
    }
}

impl OpProofsStore for MdbxProofsStorage {
    type StorageTrieCursor<'tx>
        = MdbxTrieCursor<StorageTrieHistory, Dup<'tx, StorageTrieHistory>>
    where
        Self: 'tx;
    type AccountTrieCursor<'tx>
        = MdbxTrieCursor<AccountTrieHistory, Dup<'tx, AccountTrieHistory>>
    where
        Self: 'tx;
    type StorageCursor<'tx>
        = MdbxStorageCursor<Dup<'tx, HashedStorageHistory>>
    where
        Self: 'tx;
    type AccountHashedCursor<'tx>
        = MdbxAccountCursor<Dup<'tx, HashedAccountHistory>>
    where
        Self: 'tx;

    async fn store_account_branches(
        &self,
        account_nodes: Vec<(Nibbles, Option<BranchNodeCompact>)>,
    ) -> OpProofsStorageResult<()> {
        let mut account_nodes = account_nodes;
        if account_nodes.is_empty() {
            return Ok(());
        }

        account_nodes.sort_by_key(|(key, _)| *key);

        self.env.update(|tx| {
            let mut cursor = tx.new_cursor::<AccountTrieHistory>()?;
            for (nibble, branch_node) in account_nodes {
                let vv = VersionedValue { block_number: 0, value: MaybeDeleted(branch_node) };
                cursor.append_dup(StoredNibbles::from(nibble), vv)?;
            }
            Ok(())
        })?
    }

    async fn store_storage_branches(
        &self,
        hashed_address: B256,
        storage_nodes: Vec<(Nibbles, Option<BranchNodeCompact>)>,
    ) -> OpProofsStorageResult<()> {
        let mut storage_nodes = storage_nodes;
        if storage_nodes.is_empty() {
            return Ok(());
        }

        storage_nodes.sort_by_key(|(key, _)| *key);

        self.env.update(|tx| {
            let mut cursor = tx.new_cursor::<StorageTrieHistory>()?;
            for (nibble, branch_node) in storage_nodes {
                let key = StorageTrieKey::new(hashed_address, StoredNibbles::from(nibble));
                let vv = VersionedValue { block_number: 0, value: MaybeDeleted(branch_node) };
                cursor.append_dup(key, vv)?;
            }
            Ok(())
        })?
    }

    async fn store_hashed_accounts(
        &self,
        accounts: Vec<(B256, Option<Account>)>,
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
                let vv = VersionedValue { block_number: 0, value: MaybeDeleted(account) };
                cursor.append_dup(key, vv)?;
            }
            Ok(())
        })?
    }

    async fn store_hashed_storages(
        &self,
        hashed_address: B256,
        storages: Vec<(B256, U256)>,
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
                let vv = VersionedValue {
                    block_number: 0,
                    value: MaybeDeleted(Some(StorageValue(value))),
                };
                let storage_key = HashedStorageKey::new(hashed_address, key);
                cursor.append_dup(storage_key, vv)?;
            }
            Ok(())
        })?
    }

    async fn get_earliest_block_number(&self) -> OpProofsStorageResult<Option<(u64, B256)>> {
        self.get_block_number_hash(ProofWindowKey::EarliestBlock).await
    }

    async fn get_latest_block_number(&self) -> OpProofsStorageResult<Option<(u64, B256)>> {
        let latest_block = self.get_block_number_hash(ProofWindowKey::LatestBlock).await?;
        if latest_block.is_some() {
            return Ok(latest_block);
        }

        self.get_block_number_hash(ProofWindowKey::EarliestBlock).await
    }

    fn storage_trie_cursor<'tx>(
        &self,
        hashed_address: B256,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::StorageTrieCursor<'tx>> {
        let tx = self.env.tx().map_err(|e| OpProofsStorageError::Other(e.into()))?;
        let cursor = tx
            .cursor_dup_read::<StorageTrieHistory>()
            .map_err(|e| OpProofsStorageError::Other(e.into()))?;

        Ok(MdbxTrieCursor::new(cursor, max_block_number, Some(hashed_address)))
    }

    fn account_trie_cursor<'tx>(
        &self,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::AccountTrieCursor<'tx>> {
        let tx = self.env.tx().map_err(|e| OpProofsStorageError::Other(e.into()))?;
        let cursor = tx
            .cursor_dup_read::<AccountTrieHistory>()
            .map_err(|e| OpProofsStorageError::Other(e.into()))?;

        Ok(MdbxTrieCursor::new(cursor, max_block_number, None))
    }

    fn storage_hashed_cursor<'tx>(
        &self,
        hashed_address: B256,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::StorageCursor<'tx>> {
        let tx = self.env.tx().map_err(|e| OpProofsStorageError::Other(e.into()))?;
        let cursor = tx
            .cursor_dup_read::<HashedStorageHistory>()
            .map_err(|e| OpProofsStorageError::Other(e.into()))?;

        Ok(MdbxStorageCursor::new(cursor, max_block_number, hashed_address))
    }

    fn account_hashed_cursor<'tx>(
        &self,
        max_block_number: u64,
    ) -> OpProofsStorageResult<Self::AccountHashedCursor<'tx>> {
        let tx = self.env.tx().map_err(|e| OpProofsStorageError::Other(e.into()))?;
        let cursor = tx
            .cursor_dup_read::<HashedAccountHistory>()
            .map_err(|e| OpProofsStorageError::Other(e.into()))?;

        Ok(MdbxAccountCursor::new(cursor, max_block_number))
    }

    async fn store_trie_updates(
        &self,
        block_ref: BlockWithParent,
        block_state_diff: BlockStateDiff,
    ) -> OpProofsStorageResult<()> {
        self.env.update(|tx| {
            self.store_trie_updates_inner(tx, block_ref, block_state_diff)?;
            Ok(())
        })?
    }

    async fn fetch_trie_updates(&self, block_number: u64) -> OpProofsStorageResult<BlockStateDiff> {
        self.env.view(|tx| {
            let mut change_set_cursor = tx.cursor_read::<BlockChangeSet>()?;
            let (_, change_set) = change_set_cursor
                .seek_exact(block_number)?
                .ok_or(OpProofsStorageError::NoChangeSetForBlock(block_number))?;

            let mut account_trie_cursor = tx.new_cursor::<AccountTrieHistory>()?;
            let mut storage_trie_cursor = tx.new_cursor::<StorageTrieHistory>()?;
            let mut hashed_account_cursor = tx.new_cursor::<HashedAccountHistory>()?;
            let mut hashed_storage_cursor = tx.new_cursor::<HashedStorageHistory>()?;

            let mut block_state_diff = BlockStateDiff::default();
            for key in change_set.account_trie_keys {
                let entry =
                    match account_trie_cursor.seek_by_key_subkey(key.clone(), block_number)? {
                        Some(v) if v.block_number == block_number => v.value.0,
                        _ => {
                            return Err(OpProofsStorageError::MissingAccountTrieHistory(
                                key.0,
                                block_number,
                            ))
                        }
                    };

                if let Some(value) = entry {
                    block_state_diff.trie_updates.account_nodes.insert(key.0, value);
                } else {
                    block_state_diff.trie_updates.removed_nodes.insert(key.0);
                }
            }

            for key in change_set.storage_trie_keys {
                let entry =
                    match storage_trie_cursor.seek_by_key_subkey(key.clone(), block_number)? {
                        Some(v) if v.block_number == block_number => v.value.0,
                        _ => {
                            return Err(OpProofsStorageError::MissingStorageTrieHistory(
                                key.hashed_address,
                                key.path.0,
                                block_number,
                            ))
                        }
                    };

                let stu = block_state_diff
                    .trie_updates
                    .storage_tries
                    .entry(key.hashed_address)
                    .or_insert_with(StorageTrieUpdates::default);

                // handle is_deleted scenario
                // Issue: https://github.com/op-rs/op-reth/issues/323
                if let Some(value) = entry {
                    stu.storage_nodes.insert(key.path.0, value);
                } else {
                    stu.removed_nodes.insert(key.path.0);
                }
            }

            for key in change_set.hashed_account_keys {
                let entry = match hashed_account_cursor.seek_by_key_subkey(key, block_number)? {
                    Some(v) if v.block_number == block_number => v.value.0,
                    _ => {
                        return Err(OpProofsStorageError::MissingHashedAccountHistory(
                            key,
                            block_number,
                        ))
                    }
                };

                block_state_diff.post_state.accounts.insert(key, entry);
            }

            for key in change_set.hashed_storage_keys {
                let entry =
                    match hashed_storage_cursor.seek_by_key_subkey(key.clone(), block_number)? {
                        Some(v) if v.block_number == block_number => v.value.0,
                        _ => {
                            return Err(OpProofsStorageError::MissingHashedStorageHistory {
                                hashed_address: key.hashed_address,
                                hashed_storage_key: key.hashed_storage_key,
                                block_number,
                            })
                        }
                    };

                let hs = block_state_diff
                    .post_state
                    .storages
                    .entry(key.hashed_address)
                    .or_insert_with(HashedStorage::default);

                // handle wiped storage scenario
                // Issue: https://github.com/op-rs/op-reth/issues/323
                if let Some(value) = entry {
                    hs.storage.insert(key.hashed_storage_key, value.0);
                } else {
                    hs.storage.insert(key.hashed_storage_key, U256::ZERO);
                }
            }

            Ok(block_state_diff)
        })?
    }

    /// Update the initial state with the provided diff.
    /// Prune all historical trie data prior to `new_earliest_block_number` using
    /// the [`BlockChangeSet`] index.
    async fn prune_earliest_state(
        &self,
        new_earliest_block_ref: BlockWithParent,
        diff: BlockStateDiff,
    ) -> OpProofsStorageResult<()> {
        let new_earliest_block_number = new_earliest_block_ref.block.number;
        let Some((start_block, _)) = self.get_earliest_block_number().await? else {
            return Ok(()); // Nothing to prune
        };

        if start_block >= new_earliest_block_number {
            return Ok(()); // Nothing to prune
        }

        let branches_diff = diff.trie_updates;
        let leaves_diff = diff.post_state;

        let _ = self.env.update(|tx| {
            let mut account_trie_cursor = tx.new_cursor::<AccountTrieHistory>()?;
            let mut storage_trie_cursor = tx.new_cursor::<StorageTrieHistory>()?;
            let mut hashed_account_cursor = tx.new_cursor::<HashedAccountHistory>()?;
            let mut hashed_storage_cursor = tx.new_cursor::<HashedStorageHistory>()?;

            // START PRUNING TABLES FIRST
            // We must delete old entries before inserting new ones at block 0
            // to maintain sorted order for append_dup operations.

            // First collect keys to prune to avoid borrow checker issues with cursors.
            let mut change_set_cursor = tx.new_cursor::<BlockChangeSet>()?;
            let keys_to_prune: Vec<(u64, ChangeSet)> = {
                change_set_cursor
                    .walk_range(start_block..new_earliest_block_number)?
                    .collect::<Result<Vec<_>, _>>()?
            };

            if !keys_to_prune.is_empty() {
                // Process already sorted entries directly
                for (block_number, change_set) in &keys_to_prune {
                    // Process account trie entries
                    for key in &change_set.account_trie_keys {
                        if let Some(vv) =
                            account_trie_cursor.seek_by_key_subkey(key.clone(), *block_number)? &&
                            vv.block_number == *block_number
                        {
                            account_trie_cursor.delete_current()?;
                        }
                    }

                    // Process storage trie entries
                    for key in &change_set.storage_trie_keys {
                        if let Some(vv) =
                            storage_trie_cursor.seek_by_key_subkey(key.clone(), *block_number)? &&
                            vv.block_number == *block_number
                        {
                            storage_trie_cursor.delete_current()?;
                        }
                    }

                    // Process hashed account entries
                    for key in &change_set.hashed_account_keys {
                        if let Some(vv) =
                            hashed_account_cursor.seek_by_key_subkey(*key, *block_number)? &&
                            vv.block_number == *block_number
                        {
                            hashed_account_cursor.delete_current()?;
                        }
                    }

                    // Process hashed storage entries
                    for key in &change_set.hashed_storage_keys {
                        if let Some(vv) =
                            hashed_storage_cursor.seek_by_key_subkey(key.clone(), *block_number)? &&
                            vv.block_number == *block_number
                        {
                            hashed_storage_cursor.delete_current()?;
                        }
                    }

                    // Delete the change set immediately
                    change_set_cursor.seek_exact(*block_number)?;
                    change_set_cursor.delete_current()?;
                }
            }

            // NOW INSERT THE NEW INITIAL STATE AT BLOCK 0
            // After deleting old entries, we can safely insert at block 0 with append_dup

            // Insert account trie nodes at block 0
            for (path, branch) in branches_diff.into_sorted_ref().account_nodes {
                let key: StoredNibbles = StoredNibbles::from(*path);
                let vv =
                    VersionedValue { block_number: 0, value: MaybeDeleted(Some(branch.clone())) };
                account_trie_cursor.append_dup(key, vv)?;
            }

            // Remove old entries for removed nodes prior to new_earliest_block_number
            // This ensures that any node that was removed but got added later which is still within
            // the new proof window is not deleted.
            for path in branches_diff.removed_nodes {
                let key: StoredNibbles = path.into();
                let mut walker = account_trie_cursor.walk_dup(Some(key.clone()), Some(0))?;
                while let Some(Ok((_, vv))) = walker.next() {
                    if vv.block_number < new_earliest_block_number {
                        walker.delete_current()?;
                    }
                }
            }

            // Same for storage trie nodes
            for (hashed_address, storage_updates) in branches_diff.storage_tries {
                for (path, branch) in storage_updates.into_sorted_ref().storage_nodes {
                    let key = StorageTrieKey::new(hashed_address, StoredNibbles::from(*path));
                    let vv = VersionedValue {
                        block_number: 0,
                        value: MaybeDeleted(Some(branch.clone())),
                    };
                    storage_trie_cursor.append_dup(key, vv)?;
                }

                for path in &storage_updates.removed_nodes {
                    let key = StorageTrieKey::new(hashed_address, StoredNibbles::from(*path));
                    let mut walker = storage_trie_cursor.walk_dup(Some(key.clone()), Some(0))?;
                    while let Some(Ok((_, vv))) = walker.next() {
                        if vv.block_number < new_earliest_block_number {
                            walker.delete_current()?;
                        }
                    }
                }
            }

            // Insert hashed account diff entries at block 0
            for (hashed_address, account) in leaves_diff.accounts {
                let vv = VersionedValue { block_number: 0, value: MaybeDeleted(account) };
                hashed_account_cursor.append_dup(hashed_address, vv)?;
            }

            // Hashed storage leaves
            for (hashed_address, storage) in leaves_diff.storages {
                for (slot, value) in storage.storage {
                    let key = HashedStorageKey::new(hashed_address, slot);
                    let vv = VersionedValue {
                        block_number: 0,
                        value: MaybeDeleted(Some(StorageValue(value))),
                    };
                    hashed_storage_cursor.append_dup(key, vv)?;
                }
            }

            // Set the earliest block number to the new value
            Self::inner_set_earliest_block_number(
                tx,
                new_earliest_block_number,
                new_earliest_block_ref.block.hash,
            )?;

            Ok::<(), DatabaseError>(())
        })?;

        Ok(())
    }

    async fn replace_updates(
        &self,
        latest_common_block_number: u64,
        blocks_to_add: HashMap<BlockWithParent, BlockStateDiff>,
    ) -> OpProofsStorageResult<()> {
        self.env.update(|tx| {
            let keys_to_rollback: Vec<(u64, ChangeSet)> = {
                let mut change_set_cursor = tx.new_cursor::<BlockChangeSet>()?;
                change_set_cursor
                    .walk(Some(latest_common_block_number + 1))?
                    .collect::<Result<Vec<_>, _>>()?
            };

            if !keys_to_rollback.is_empty() {
                let mut account_trie_cursor = tx.new_cursor::<AccountTrieHistory>()?;
                let mut storage_trie_cursor = tx.new_cursor::<StorageTrieHistory>()?;
                let mut hashed_account_cursor = tx.new_cursor::<HashedAccountHistory>()?;
                let mut hashed_storage_cursor = tx.new_cursor::<HashedStorageHistory>()?;
                let mut change_set_cursor = tx.new_cursor::<BlockChangeSet>()?;

                // Process already sorted entries directly
                for (block_number, change_set) in &keys_to_rollback {
                    // Process account trie entries
                    for key in &change_set.account_trie_keys {
                        if let Some(vv) =
                            account_trie_cursor.seek_by_key_subkey(key.clone(), *block_number)? &&
                            vv.block_number == *block_number
                        {
                            account_trie_cursor.delete_current()?;
                        }
                    }

                    // Process storage trie entries
                    for key in &change_set.storage_trie_keys {
                        if let Some(vv) =
                            storage_trie_cursor.seek_by_key_subkey(key.clone(), *block_number)? &&
                            vv.block_number == *block_number
                        {
                            storage_trie_cursor.delete_current()?;
                        }
                    }

                    // Process hashed account entries
                    for key in &change_set.hashed_account_keys {
                        if let Some(vv) =
                            hashed_account_cursor.seek_by_key_subkey(*key, *block_number)? &&
                            vv.block_number == *block_number
                        {
                            hashed_account_cursor.delete_current()?;
                        }
                    }

                    // Process hashed storage entries
                    for key in &change_set.hashed_storage_keys {
                        if let Some(vv) =
                            hashed_storage_cursor.seek_by_key_subkey(key.clone(), *block_number)? &&
                            vv.block_number == *block_number
                        {
                            hashed_storage_cursor.delete_current()?;
                        }
                    }

                    // Delete the change set immediately
                    change_set_cursor.seek_exact(*block_number)?;
                    change_set_cursor.delete_current()?;
                }
            }

            // Sort by block number: Hashmap does not guarantee order
            // todo: use a sorted vec instead
            let mut blocks_to_add_vec: Vec<(BlockWithParent, BlockStateDiff)> =
                blocks_to_add.into_iter().collect();

            blocks_to_add_vec.sort_unstable_by_key(|(bwp, _)| bwp.block.number);

            // update the proof window
            // todo: refactor to use block hash from the block to add. We need to pass the
            // BlockNumHash type for the latest_common_block_number
            let mut proof_window_cursor = tx.new_cursor::<ProofWindow>()?;
            proof_window_cursor.append(
                ProofWindowKey::LatestBlock,
                &BlockNumberHash::new(
                    latest_common_block_number,
                    blocks_to_add_vec.first().unwrap().0.parent,
                ),
            )?;

            for (block_with_parent, diff) in blocks_to_add_vec {
                self.store_trie_updates_inner(tx, block_with_parent, diff)?;
            }
            Ok(())
        })?
    }

    async fn set_earliest_block_number(
        &self,
        block_number: u64,
        hash: B256,
    ) -> OpProofsStorageResult<()> {
        self.set_earliest_block_number_hash(block_number, hash).await
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::db::{
        models::{AccountTrieHistory, StorageTrieHistory},
        StorageTrieKey,
    };
    use alloy_eips::NumHash;
    use alloy_primitives::B256;
    use reth_db::{
        cursor::DbDupCursorRO,
        transaction::{DbTx, DbTxMut},
        DatabaseError,
    };
    use reth_trie::{
        updates::StorageTrieUpdates, BranchNodeCompact, HashedStorage, Nibbles, StoredNibbles,
    };
    use tempfile::TempDir;

    const B0: u64 = 0;

    #[tokio::test]
    async fn store_hashed_accounts_writes_versioned_values() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        let addr = B256::from([0xAA; 32]);
        let account = Account::default();
        store.store_hashed_accounts(vec![(addr, Some(account))]).await.expect("write accounts");

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
            .store_hashed_accounts(vec![(a2, None), (a1, Some(acc1)), (a3, Some(acc3))])
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
                .store_hashed_accounts(vec![(a2, None), (a1, Some(acc1)), (a4, Some(acc4))])
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
                .store_hashed_accounts(vec![(a5, Some(acc5)), (a3, Some(acc3))])
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

        store.store_hashed_storages(addr, vec![(slot, val)]).await.expect("write storage");

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

        store.store_hashed_storages(addr, vec![(s2, v2), (s1, v1), (s3, v3)]).await.expect("write");

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
                .store_hashed_storages(addr, vec![(s2, v2), (s1, v1), (s5, v5)])
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
            store.store_hashed_storages(addr, vec![(s4, v4), (s3, v3)]).await.expect("write");

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

        store.store_account_branches(updates).await.expect("write");

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
        store.store_account_branches(updates.clone()).await.expect("write");

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
            store.store_account_branches(updates1.clone()).await.expect("write");

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
            store.store_account_branches(updates2.clone()).await.expect("write");

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

        store.store_storage_branches(hashed_address, items).await.expect("write");

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
        store.store_storage_branches(hashed_address, items.clone()).await.expect("write");

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
            store.store_storage_branches(hashed_address, items1.clone()).await.expect("write");

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
            store.store_storage_branches(hashed_address, items2.clone()).await.expect("write");

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

    #[tokio::test]
    async fn test_store_trie_updates_comprehensive() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        // Sample block number
        const BLOCK: BlockWithParent =
            BlockWithParent::new(B256::ZERO, NumHash::new(42, B256::ZERO));

        // Sample addresses and keys
        let addr1 = B256::from([0x11; 32]);
        let addr2 = B256::from([0x22; 32]);
        let slot1 = B256::from([0xA1; 32]);
        let slot2 = B256::from([0xA2; 32]);

        // Sample accounts
        let acc1 = Account { nonce: 1, balance: U256::from(100), ..Default::default() };

        // Sample storage values
        let val1 = U256::from(1234u64);
        let val2 = U256::from(5678u64);

        // Sample trie paths
        let account_path1 = Nibbles::from_nibbles_unchecked(vec![0, 1, 2, 3]);
        let account_path2 = Nibbles::from_nibbles_unchecked(vec![4, 5, 6, 7]);
        let removed_account_path = Nibbles::from_nibbles_unchecked(vec![7, 8, 9]);

        let account_node1 = BranchNodeCompact::default();
        let account_node2 = BranchNodeCompact::default();

        let storage_path1 = Nibbles::from_nibbles_unchecked(vec![1, 2, 3, 4]);
        let storage_path2 = Nibbles::from_nibbles_unchecked(vec![8, 9, 0, 1]);

        let storage_node1 = BranchNodeCompact::default();
        let storage_node2 = BranchNodeCompact::default();

        // Construct test BlockStateDiff
        let mut block_state_diff = BlockStateDiff::default();

        // Add account trie nodes
        block_state_diff.trie_updates.account_nodes.insert(account_path1, account_node1.clone());
        block_state_diff.trie_updates.account_nodes.insert(account_path2, account_node2.clone());
        block_state_diff.trie_updates.removed_nodes.insert(removed_account_path);

        // Add storage trie nodes for two addresses
        let mut storage_nodes1 = StorageTrieUpdates::default();
        storage_nodes1.storage_nodes.insert(storage_path1, storage_node1.clone());
        block_state_diff.trie_updates.storage_tries.insert(addr1, storage_nodes1);

        let mut storage_nodes2 = StorageTrieUpdates::default();
        storage_nodes2.storage_nodes.insert(storage_path2, storage_node2.clone());
        block_state_diff.trie_updates.storage_tries.insert(addr2, storage_nodes2);

        // Add hashed accounts (one Some, one None)
        block_state_diff.post_state.accounts.insert(addr1, Some(acc1));
        block_state_diff.post_state.accounts.insert(addr2, None); // Deletion

        // Add storage slots for both addresses
        let mut storage1 = HashedStorage::default();
        storage1.storage.insert(slot1, val1);
        block_state_diff.post_state.storages.insert(addr1, storage1);

        let mut storage2 = HashedStorage::default();
        storage2.storage.insert(slot2, val2);
        block_state_diff.post_state.storages.insert(addr2, storage2);

        // Store everything
        store.store_trie_updates(BLOCK, block_state_diff).await.expect("store");

        // Verify account trie nodes
        {
            let tx = store.env.tx().expect("tx");
            let mut cur = tx.new_cursor::<AccountTrieHistory>().expect("cursor");

            // Check first node
            let vv1 = cur
                .seek_by_key_subkey(account_path1.into(), BLOCK.block.number)
                .expect("seek")
                .expect("exists");
            assert_eq!(vv1.block_number, BLOCK.block.number);
            assert!(vv1.value.0.is_some());

            // Check second node
            let vv2 = cur
                .seek_by_key_subkey(account_path2.into(), BLOCK.block.number)
                .expect("seek")
                .expect("exists");
            assert_eq!(vv2.block_number, BLOCK.block.number);
            assert!(vv2.value.0.is_some());

            // Check removed node
            let vv3 = cur
                .seek_by_key_subkey(removed_account_path.into(), BLOCK.block.number)
                .expect("seek")
                .expect("exists");
            assert_eq!(vv3.block_number, BLOCK.block.number);
            assert!(vv3.value.0.is_none(), "Expected node deletion");
        }

        // Verify storage trie nodes
        {
            let tx = store.env.tx().expect("tx");
            let mut cur = tx.new_cursor::<StorageTrieHistory>().expect("cursor");

            // Check node for addr1
            let key1 = StorageTrieKey::new(addr1, storage_path1.into());
            let vv1 =
                cur.seek_by_key_subkey(key1, BLOCK.block.number).expect("seek").expect("exists");
            assert_eq!(vv1.block_number, BLOCK.block.number);
            assert!(vv1.value.0.is_some());

            // Check node for addr2
            let key2 = StorageTrieKey::new(addr2, storage_path2.into());
            let vv2 =
                cur.seek_by_key_subkey(key2, BLOCK.block.number).expect("seek").expect("exists");
            assert_eq!(vv2.block_number, BLOCK.block.number);
            assert!(vv2.value.0.is_some());
        }

        // Verify hashed accounts
        {
            let tx = store.env.tx().expect("tx");
            let mut cur = tx.new_cursor::<HashedAccountHistory>().expect("cursor");

            // Check account1 (exists)
            let vv1 =
                cur.seek_by_key_subkey(addr1, BLOCK.block.number).expect("seek").expect("exists");
            assert_eq!(vv1.block_number, BLOCK.block.number);
            assert_eq!(vv1.value.0, Some(acc1));

            // Check account2 (deletion)
            let vv2 =
                cur.seek_by_key_subkey(addr2, BLOCK.block.number).expect("seek").expect("exists");
            assert_eq!(vv2.block_number, BLOCK.block.number);
            assert!(vv2.value.0.is_none(), "Expected account deletion");
        }

        // Verify hashed storages
        {
            let tx = store.env.tx().expect("tx");
            let mut cur = tx.new_cursor::<HashedStorageHistory>().expect("cursor");

            // Check storage for addr1
            let key1 = HashedStorageKey::new(addr1, slot1);
            let vv1 =
                cur.seek_by_key_subkey(key1, BLOCK.block.number).expect("seek").expect("exists");
            assert_eq!(vv1.block_number, BLOCK.block.number);
            let inner1 = vv1.value.0.as_ref().expect("Some(StorageValue)");
            assert_eq!(inner1.0, val1);

            // Check storage for addr2
            let key2 = HashedStorageKey::new(addr2, slot2);
            let vv2 =
                cur.seek_by_key_subkey(key2, BLOCK.block.number).expect("seek").expect("exists");
            assert_eq!(vv2.block_number, BLOCK.block.number);
            let inner2 = vv2.value.0.as_ref().expect("Some(StorageValue)");
            assert_eq!(inner2.0, val2);
        }

        // Verify BlockChangeSet entries
        {
            let tx = store.env.tx().expect("tx");
            let mut cur = tx.new_cursor::<BlockChangeSet>().expect("cursor");
            let entries: Vec<_> = cur.walk(Some(BLOCK.block.number)).expect("walk").collect();
            assert_eq!(entries.len(), 1, "Expected 1 BlockChangeSet entry");
        }

        // check the latest block number in proof window
        {
            let tx = store.env.tx().expect("tx");
            let mut proof_window_cursor = tx.new_cursor::<ProofWindow>().expect("cursor");
            let latest_block = proof_window_cursor
                .seek(ProofWindowKey::LatestBlock)
                .expect("seek")
                .expect("exists");
            assert_eq!(latest_block.1.number(), BLOCK.block.number);
            assert_eq!(*latest_block.1.hash(), BLOCK.block.hash);
        }
    }

    #[tokio::test]
    async fn store_trie_updates_out_of_order_rejects() {
        let dir = tempfile::TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        // set latest to some hash H1
        let existing_block = BlockWithParent::new(B256::random(), NumHash::new(1, B256::random()));
        store
            .set_earliest_block_number(existing_block.block.number, existing_block.block.hash)
            .await
            .expect("set");

        // incoming block whose parent != existing latest
        let bad_parent = B256::from([0xFF; 32]);
        let bad_block: BlockWithParent =
            BlockWithParent::new(bad_parent, NumHash::new(2, B256::ZERO));
        let diff = BlockStateDiff::default();

        let res = store.store_trie_updates(bad_block, diff).await;
        assert!(matches!(res, Err(OpProofsStorageError::OutOfOrder { .. })));
        // verify nothing written: proof window still unchanged
        let latest = store.get_latest_block_number().await.expect("get latest");
        assert_eq!(latest.unwrap().1, existing_block.block.hash);
    }

    #[tokio::test]
    async fn store_trie_updates_multiple_blocks_append_versions() {
        let dir = tempfile::TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        let addr = B256::from([0x21; 32]);
        // block A (parent = ZERO)
        let block_a = BlockWithParent::new(B256::ZERO, NumHash::new(1, B256::random()));
        let mut diff_a = BlockStateDiff::default();
        diff_a.post_state.accounts.insert(addr, Some(Account::default()));

        store.store_trie_updates(block_a, diff_a).await.expect("store A");

        // block B (parent = hash of A)
        let block_b = BlockWithParent::new(block_a.block.hash, NumHash::new(2, B256::random()));
        let mut diff_b = BlockStateDiff::default();
        diff_b.post_state.accounts.insert(addr, Some(Account { nonce: 5, ..Default::default() }));

        store.store_trie_updates(block_b, diff_b).await.expect("store B");

        // verify we can retrieve entries for both block numbers
        let tx = store.env.tx().expect("tx");
        let mut cur = tx.new_cursor::<HashedAccountHistory>().expect("cursor");
        let v_a =
            cur.seek_by_key_subkey(addr, block_a.block.number).expect("seek").expect("exists");
        let v_b =
            cur.seek_by_key_subkey(addr, block_b.block.number).expect("seek").expect("exists");
        assert_eq!(v_a.block_number, block_a.block.number);
        assert_eq!(v_b.block_number, block_b.block.number);
    }

    #[tokio::test]
    async fn test_store_trie_updates_empty_collections() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        const BLOCK: BlockWithParent =
            BlockWithParent::new(B256::ZERO, NumHash::new(42, B256::ZERO));

        // Create BlockStateDiff with empty collections
        let block_state_diff = BlockStateDiff::default();

        // This should work without errors
        store.store_trie_updates(BLOCK, block_state_diff).await.expect("store");

        // Verify nothing was written (should be empty)
        let tx = store.env.tx().expect("tx");

        let mut cur1 = tx.new_cursor::<AccountTrieHistory>().expect("cursor");
        assert!(cur1.next_dup_val().expect("first").is_none(), "Account trie should be empty");

        let mut cur2 = tx.new_cursor::<StorageTrieHistory>().expect("cursor");
        assert!(cur2.next_dup_val().expect("first").is_none(), "Storage trie should be empty");

        let mut cur3 = tx.new_cursor::<HashedAccountHistory>().expect("cursor");
        assert!(cur3.next_dup_val().expect("first").is_none(), "Hashed accounts should be empty");

        let mut cur4 = tx.new_cursor::<HashedStorageHistory>().expect("cursor");
        assert!(cur4.next_dup_val().expect("first").is_none(), "Hashed storage should be empty");

        let mut cur5 = tx.new_cursor::<BlockChangeSet>().expect("cursor");
        assert!(
            cur5.next().expect("first").is_some(),
            "Pruning index SHOULD populate the change set even for empty diffs"
        );
    }

    #[tokio::test]
    async fn fetch_trie_updates_missing_changeset_returns_error() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        let res = store.fetch_trie_updates(99).await;
        assert!(matches!(res, Err(OpProofsStorageError::NoChangeSetForBlock(99))));
    }

    #[tokio::test]
    async fn fetch_trie_updates_empty_changeset() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        let block = BlockWithParent::new(B256::ZERO, NumHash::new(1, B256::random()));
        let diff = BlockStateDiff::default();

        store.store_trie_updates(block, diff).await.expect("store");
        let got = store.fetch_trie_updates(1).await.expect("fetch");
        assert!(got.trie_updates.account_nodes.is_empty());
        assert!(got.trie_updates.removed_nodes.is_empty());
        assert!(got.trie_updates.storage_tries.is_empty());
        assert!(got.post_state.accounts.is_empty());
        assert!(got.post_state.storages.is_empty());
    }

    #[tokio::test]
    async fn fetch_trie_updates_missing_account_history_entry_returns_error() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        // prepare ChangeSet that references StoredNibbles for account key
        // (insert ChangeSet into BlockChangeSet directly using tx)
        {
            let tx = store.env.tx_mut().unwrap();
            let mut cur = tx.cursor_write::<BlockChangeSet>().unwrap();
            cur.insert(
                1,
                &ChangeSet {
                    account_trie_keys: vec![StoredNibbles::default()],
                    ..Default::default()
                },
            )
            .unwrap();
            tx.commit().unwrap();
        }

        let res = store.fetch_trie_updates(1).await;
        assert!(matches!(res, Err(OpProofsStorageError::MissingAccountTrieHistory(..))));
    }

    #[tokio::test]
    async fn fetch_trie_updates_account_history_seek_returns_later_block_treated_as_missing() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        // manually insert account history and ChangeSet for block 1 referencing same key
        {
            let tx = store.env.tx_mut().unwrap();
            let mut acc_cur = tx.cursor_write::<AccountTrieHistory>().unwrap();
            acc_cur
                .insert(
                    StoredNibbles::from(Nibbles::from_nibbles_unchecked([0x1])),
                    &VersionedValue::new(2, MaybeDeleted(Some(BranchNodeCompact::default()))),
                )
                .unwrap();

            let mut cur = tx.cursor_write::<BlockChangeSet>().unwrap();
            cur.insert(
                1,
                &ChangeSet {
                    account_trie_keys: vec![StoredNibbles::from(Nibbles::from_nibbles_unchecked(
                        [0x1],
                    ))],
                    ..Default::default()
                },
            )
            .unwrap();
            tx.commit().unwrap();
        }

        // fetch block 1 -> seek will find block 2 but block_number != 1 so expect
        // MissingAccountTrieHistory
        let res = store.fetch_trie_updates(1).await;
        assert!(matches!(res, Err(OpProofsStorageError::MissingAccountTrieHistory(..))));
    }

    #[tokio::test]
    async fn fetch_trie_updates_missing_storage_history_entry_returns_error() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        // prepare ChangeSet that references StorageTrieKey for storage trie
        // (insert ChangeSet into BlockChangeSet directly using tx)
        {
            let tx = store.env.tx_mut().unwrap();
            let mut cur = tx.cursor_write::<BlockChangeSet>().unwrap();
            cur.insert(
                1,
                &ChangeSet {
                    storage_trie_keys: vec![StorageTrieKey::new(
                        B256::from([0u8; 32]),
                        StoredNibbles::default(),
                    )],
                    ..Default::default()
                },
            )
            .unwrap();
            tx.commit().unwrap();
        }

        let res = store.fetch_trie_updates(1).await;
        assert!(matches!(res, Err(OpProofsStorageError::MissingStorageTrieHistory(..))));
    }

    #[tokio::test]
    async fn fetch_trie_updates_storage_history_seek_returns_later_block_treated_as_missing() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        // manually insert storage history and ChangeSet for block 1 referencing same key
        {
            let tx = store.env.tx_mut().unwrap();
            let mut stor_cur = tx.cursor_write::<StorageTrieHistory>().unwrap();
            stor_cur
                .insert(
                    StorageTrieKey::new(
                        B256::from([0u8; 32]),
                        StoredNibbles::from(Nibbles::from_nibbles_unchecked([0x1])),
                    ),
                    &VersionedValue::new(2, MaybeDeleted(Some(BranchNodeCompact::default()))),
                )
                .unwrap();

            let mut cur = tx.cursor_write::<BlockChangeSet>().unwrap();
            cur.insert(
                1,
                &ChangeSet {
                    storage_trie_keys: vec![StorageTrieKey::new(
                        B256::from([0u8; 32]),
                        StoredNibbles::from(Nibbles::from_nibbles_unchecked([0x1])),
                    )],
                    ..Default::default()
                },
            )
            .unwrap();
            tx.commit().unwrap();
        }

        // fetch block 1 -> seek will find block 2 but block_number != 1 so expect
        // MissingStorageTrieHistory
        let res = store.fetch_trie_updates(1).await;
        assert!(matches!(res, Err(OpProofsStorageError::MissingStorageTrieHistory(..))));
    }

    #[tokio::test]
    async fn fetch_trie_updates_missing_hashed_account_entry_returns_error() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        // prepare ChangeSet that references hashed account address
        // (insert ChangeSet into BlockChangeSet directly using tx)
        {
            let tx = store.env.tx_mut().unwrap();
            let mut cur = tx.cursor_write::<BlockChangeSet>().unwrap();
            cur.insert(
                1,
                &ChangeSet {
                    hashed_account_keys: vec![B256::from([0u8; 32])],
                    ..Default::default()
                },
            )
            .unwrap();
            tx.commit().unwrap();
        }

        let res = store.fetch_trie_updates(1).await;
        assert!(matches!(res, Err(OpProofsStorageError::MissingHashedAccountHistory(..))));
    }

    #[tokio::test]
    async fn fetch_trie_updates_hashed_account_seek_returns_later_block_treated_as_missing() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        // manually insert hashed account history and ChangeSet for block 1 referencing same key
        {
            let tx = store.env.tx_mut().unwrap();
            let mut acc_cur = tx.cursor_write::<HashedAccountHistory>().unwrap();
            acc_cur
                .insert(
                    B256::from([0u8; 32]),
                    &VersionedValue::new(2, MaybeDeleted(Some(Account::default()))),
                )
                .unwrap();

            let mut cur = tx.cursor_write::<BlockChangeSet>().unwrap();
            cur.insert(
                1,
                &ChangeSet {
                    hashed_account_keys: vec![B256::from([0u8; 32])],
                    ..Default::default()
                },
            )
            .unwrap();
            tx.commit().unwrap();
        }

        // fetch block 1 -> seek will find block 2 but block_number != 1 so expect
        // MissingHashedAccountHistory
        let res = store.fetch_trie_updates(1).await;
        assert!(matches!(res, Err(OpProofsStorageError::MissingHashedAccountHistory(..))));
    }

    #[tokio::test]
    async fn fetch_trie_updates_missing_hashed_storage_entry_returns_error() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        // prepare ChangeSet that references hashed storage key
        // (insert ChangeSet into BlockChangeSet directly using tx)
        {
            let tx = store.env.tx_mut().unwrap();
            let mut cur = tx.cursor_write::<BlockChangeSet>().unwrap();
            cur.insert(
                1,
                &ChangeSet {
                    hashed_storage_keys: vec![HashedStorageKey::new(
                        B256::from([0u8; 32]),
                        B256::from([0u8; 32]),
                    )],
                    ..Default::default()
                },
            )
            .unwrap();
            tx.commit().unwrap();
        }

        let res = store.fetch_trie_updates(1).await;
        assert!(matches!(res, Err(OpProofsStorageError::MissingHashedStorageHistory { .. })));
    }

    #[tokio::test]
    async fn fetch_trie_updates_hashed_storage_seek_returns_later_block_treated_as_missing() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        // manually insert hashed storage history and ChangeSet for block 1 referencing same key
        {
            let tx = store.env.tx_mut().unwrap();
            let mut stor_cur = tx.cursor_write::<HashedStorageHistory>().unwrap();
            stor_cur
                .insert(
                    HashedStorageKey::new(B256::from([0u8; 32]), B256::from([0u8; 32])),
                    &VersionedValue::new(2, MaybeDeleted(Some(StorageValue::new(U256::ZERO)))),
                )
                .unwrap();

            let mut cur = tx.cursor_write::<BlockChangeSet>().unwrap();
            cur.insert(
                1,
                &ChangeSet {
                    hashed_storage_keys: vec![HashedStorageKey::new(
                        B256::from([0u8; 32]),
                        B256::from([0u8; 32]),
                    )],
                    ..Default::default()
                },
            )
            .unwrap();
            tx.commit().unwrap();
        }

        // fetch block 1 -> seek will find block 2 but block_number != 1 so expect
        // MissingHashedStorageHistory
        let res = store.fetch_trie_updates(1).await;
        assert!(matches!(res, Err(OpProofsStorageError::MissingHashedStorageHistory { .. })));
    }

    #[tokio::test]
    async fn fetch_trie_updates_basic() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        // Build a block with mixed changes (accounts, trie nodes, hashed storages)
        let block = BlockWithParent::new(B256::ZERO, NumHash::new(1, B256::random()));

        // prepare data
        let addr1 = B256::from([0x11; 32]);
        let addr2 = B256::from([0x22; 32]);
        let slot1 = B256::from([0xA1; 32]);
        let slot2 = B256::from([0xA2; 32]);

        let acc1 = Account { nonce: 1, balance: U256::from(100), ..Default::default() };

        let val1 = U256::from(1234u64);
        let val2 = U256::from(5678u64);

        let account_path1 = Nibbles::from_nibbles_unchecked(vec![0, 1, 2, 3]);
        let account_path2 = Nibbles::from_nibbles_unchecked(vec![4, 5, 6, 7]);
        let account_node1 =
            BranchNodeCompact { root_hash: Some(B256::random()), ..Default::default() };
        let account_node2 =
            BranchNodeCompact { root_hash: Some(B256::random()), ..Default::default() };

        let storage_path1 = Nibbles::from_nibbles_unchecked(vec![1, 2, 3, 4]);
        let storage_node1 =
            BranchNodeCompact { root_hash: Some(B256::random()), ..Default::default() };

        // Construct BlockStateDiff
        let mut block_state_diff = BlockStateDiff::default();
        block_state_diff.trie_updates.account_nodes.insert(account_path1, account_node1.clone());
        block_state_diff.trie_updates.account_nodes.insert(account_path2, account_node2.clone());
        // storage trie for addr1
        let mut storage_nodes1 = StorageTrieUpdates::default();
        storage_nodes1.storage_nodes.insert(storage_path1, storage_node1.clone());
        block_state_diff.trie_updates.storage_tries.insert(addr1, storage_nodes1);

        // hashed accounts: addr1 -> Some, addr2 -> None
        block_state_diff.post_state.accounts.insert(addr1, Some(acc1));
        block_state_diff.post_state.accounts.insert(addr2, None);

        // hashed storages
        let mut storage1 = HashedStorage::default();
        storage1.storage.insert(slot1, val1);
        block_state_diff.post_state.storages.insert(addr1, storage1);

        let mut storage2 = HashedStorage::default();
        storage2.storage.insert(slot2, val2);
        block_state_diff.post_state.storages.insert(addr2, storage2);

        // store then fetch
        store.store_trie_updates(block, block_state_diff.clone()).await.expect("store");
        let got = store.fetch_trie_updates(1).await.expect("fetch");

        // verify trie updates
        assert_eq!(got.trie_updates.account_nodes, block_state_diff.trie_updates.account_nodes,);
        assert_eq!(got.trie_updates.removed_nodes, block_state_diff.trie_updates.removed_nodes,);
        assert_eq!(got.trie_updates.storage_tries, block_state_diff.trie_updates.storage_tries,);

        // verify post state
        assert_eq!(got.post_state.accounts, block_state_diff.post_state.accounts);
        assert_eq!(got.post_state.storages, block_state_diff.post_state.storages);
    }

    #[tokio::test]
    async fn test_prune_earliest_state_single_entry() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");
        let block = BlockWithParent::new(B256::ZERO, NumHash::new(1, B256::random()));
        store.set_earliest_block_number(0, B256::ZERO).await.unwrap();

        // Insert a single entry to be pruned
        let addr = B256::random();
        let mut state_diff = BlockStateDiff::default();
        state_diff.post_state.accounts.insert(addr, Some(Account::default()));
        store.store_trie_updates(block, state_diff).await.unwrap();

        // Prune the entry - pass empty diff since we're just removing data
        let next_block = BlockWithParent::new(block.block.hash, NumHash::new(2, B256::random()));
        let diff = BlockStateDiff::default();
        store.prune_earliest_state(next_block, diff).await.unwrap();

        // Verify the entry was pruned
        let tx = store.env.tx().unwrap();
        let mut cur = tx.new_cursor::<HashedAccountHistory>().unwrap();
        assert!(cur.seek_by_key_subkey(addr, block.block.number).unwrap().is_none());
        let mut pruning_cur = tx.new_cursor::<BlockChangeSet>().unwrap();
        assert!(pruning_cur.seek_exact(block.block.number).unwrap().is_none());

        // Verify earliest block was updated
        let earliest = store.get_earliest_block_number().await.unwrap();
        assert_eq!(earliest, Some((2, next_block.block.hash)));
    }

    #[tokio::test]
    async fn test_prune_earliest_state_multiple_entries_same_block() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");
        let block = BlockWithParent::new(B256::ZERO, NumHash::new(1, B256::random()));
        store.set_earliest_block_number(0, B256::ZERO).await.unwrap();

        // Insert multiple entries for the same block
        let addr1 = B256::random();
        let addr2 = B256::random();
        let mut state_diff = BlockStateDiff::default();
        state_diff.post_state.accounts.insert(addr1, Some(Account::default()));
        state_diff.post_state.accounts.insert(addr2, Some(Account::default()));
        store.store_trie_updates(block, state_diff).await.unwrap();

        // Prune the entries
        let next_block = BlockWithParent::new(block.block.hash, NumHash::new(2, B256::random()));
        let diff = BlockStateDiff::default();
        store.prune_earliest_state(next_block, diff).await.unwrap();

        // Verify the entries were pruned
        let tx = store.env.tx().unwrap();
        let mut cur = tx.new_cursor::<HashedAccountHistory>().unwrap();
        assert!(cur.seek_by_key_subkey(addr1, block.block.number).unwrap().is_none());
        assert!(cur.seek_by_key_subkey(addr2, block.block.number).unwrap().is_none());
        let mut pruning_cur = tx.new_cursor::<BlockChangeSet>().unwrap();
        assert!(pruning_cur.seek_exact(block.block.number).unwrap().is_none());
    }

    #[tokio::test]
    async fn test_prune_earliest_state_multiple_blocks() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");
        let block_1 = BlockWithParent::new(B256::ZERO, NumHash::new(1, B256::random()));
        let block_2 = BlockWithParent::new(block_1.block.hash, NumHash::new(2, B256::random()));
        let block_3 = BlockWithParent::new(block_2.block.hash, NumHash::new(3, B256::random()));
        store.set_earliest_block_number(0, B256::ZERO).await.unwrap();

        // Insert entries for multiple blocks
        let addr1 = B256::random();
        let addr2 = B256::random();
        let mut state_diff1 = BlockStateDiff::default();
        state_diff1.post_state.accounts.insert(addr1, Some(Account::default()));
        store.store_trie_updates(block_1, state_diff1).await.unwrap();

        let mut state_diff2 = BlockStateDiff::default();
        state_diff2.post_state.accounts.insert(addr2, Some(Account::default()));
        store.store_trie_updates(block_2, state_diff2).await.unwrap();

        // Prune up to block 3 (should remove blocks 1 and 2)
        let diff = BlockStateDiff::default();
        store.prune_earliest_state(block_3, diff).await.unwrap();

        // Verify the entries were pruned
        let tx = store.env.tx().unwrap();
        let mut cur = tx.new_cursor::<HashedAccountHistory>().unwrap();
        assert!(cur.seek_by_key_subkey(addr1, 1).unwrap().is_none());
        assert!(cur.seek_by_key_subkey(addr2, 2).unwrap().is_none());
        let mut pruning_cur = tx.new_cursor::<BlockChangeSet>().unwrap();
        assert!(pruning_cur.seek_exact(1).unwrap().is_none());
        assert!(pruning_cur.seek_exact(2).unwrap().is_none());
    }

    #[tokio::test]
    async fn test_prune_earliest_state_no_op() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");
        let diff = BlockStateDiff::default();
        store.set_earliest_block_number(1, B256::random()).await.unwrap();

        // Attempt to prune with a new earliest block that is not newer
        let block_1 = BlockWithParent::new(B256::ZERO, NumHash::new(1, B256::random()));
        let block_0 = BlockWithParent::new(B256::ZERO, NumHash::new(0, B256::random()));
        store.prune_earliest_state(block_1, diff.clone()).await.unwrap();
        store.prune_earliest_state(block_0, diff).await.unwrap();

        // Nothing should have been pruned, this call should not panic or error
    }

    #[tokio::test]
    async fn test_prune_earliest_state_no_entries_to_prune() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");
        let diff = BlockStateDiff::default();
        store.set_earliest_block_number(1, B256::random()).await.unwrap();

        // Prune a range where no entries exist
        let block_10 = BlockWithParent::new(B256::ZERO, NumHash::new(10, B256::random()));
        store.prune_earliest_state(block_10, diff).await.unwrap();

        // Nothing should have been pruned, this call should not panic or error
    }

    #[tokio::test]
    async fn test_prune_earliest_state_with_diff_insertion() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");
        store.set_earliest_block_number(0, B256::ZERO).await.unwrap();

        // Insert entries for blocks 1 and 2
        let addr1 = B256::random();
        let addr2 = B256::random();
        let acc1 = Account { nonce: 1, balance: U256::from(100), ..Default::default() };
        let acc2 = Account { nonce: 2, balance: U256::from(200), ..Default::default() };

        let block_1 = BlockWithParent::new(B256::ZERO, NumHash::new(1, B256::random()));
        let mut state_diff1 = BlockStateDiff::default();
        state_diff1.post_state.accounts.insert(addr1, Some(acc1));
        store.store_trie_updates(block_1, state_diff1).await.unwrap();

        let block_2 = BlockWithParent::new(block_1.block.hash, NumHash::new(2, B256::random()));
        let mut state_diff2 = BlockStateDiff::default();
        state_diff2.post_state.accounts.insert(addr2, Some(acc2));
        store.store_trie_updates(block_2, state_diff2).await.unwrap();

        // Now prune to block 3, passing a diff that represents the new initial state
        let new_initial_account =
            Account { nonce: 10, balance: U256::from(1000), ..Default::default() };
        let new_addr = B256::random();
        let mut prune_diff = BlockStateDiff::default();
        prune_diff.post_state.accounts.insert(addr1, Some(acc1));
        prune_diff.post_state.accounts.insert(addr2, Some(acc2));
        prune_diff.post_state.accounts.insert(new_addr, Some(new_initial_account));

        let block_3 = BlockWithParent::new(block_2.block.hash, NumHash::new(3, B256::random()));
        store.prune_earliest_state(block_3, prune_diff).await.unwrap();

        // Verify that blocks 1 and 2 entries were pruned
        let tx = store.env.tx().unwrap();
        let mut cur = tx.new_cursor::<HashedAccountHistory>().unwrap();
        assert!(
            cur.seek_by_key_subkey(addr1, 1).unwrap().is_none(),
            "Block 1 entry should be pruned"
        );
        assert!(
            cur.seek_by_key_subkey(addr2, 2).unwrap().is_none(),
            "Block 2 entry should be pruned"
        );

        // Verify that the new diff was inserted at block 0
        let vv = cur
            .seek_by_key_subkey(new_addr, 0)
            .unwrap()
            .expect("New initial state should exist at block 0");
        assert_eq!(vv.block_number, 0);
        assert_eq!(vv.value.0, Some(new_initial_account));

        // Verify change sets for blocks 1 and 2 were removed
        let mut pruning_cur = tx.new_cursor::<BlockChangeSet>().unwrap();
        assert!(pruning_cur.seek_exact(1).unwrap().is_none());
        assert!(pruning_cur.seek_exact(2).unwrap().is_none());

        // Verify earliest block was updated
        let earliest = store.get_earliest_block_number().await.unwrap();
        assert_eq!(earliest, Some((3, block_3.block.hash)));
    }

    #[tokio::test]
    async fn test_prune_earliest_state_with_removed_nodes() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");
        store.set_earliest_block_number(0, B256::ZERO).await.unwrap();

        // Create some trie nodes in blocks 1, 2, 3
        let path1 = Nibbles::from_nibbles_unchecked([0x01, 0x02]);
        let path2 = Nibbles::from_nibbles_unchecked([0x03, 0x04]);
        let node1 = BranchNodeCompact::new(0b1, 0, 0, vec![], Some(B256::random()));
        let node2 = BranchNodeCompact::new(0b10, 0, 0, vec![], Some(B256::random()));

        let block_1 = BlockWithParent::new(B256::ZERO, NumHash::new(1, B256::random()));
        let mut diff1 = BlockStateDiff::default();
        diff1.trie_updates.account_nodes.insert(path1, node1.clone());
        store.store_trie_updates(block_1, diff1).await.unwrap();

        let block_2 = BlockWithParent::new(block_1.block.hash, NumHash::new(2, B256::random()));
        let mut diff2 = BlockStateDiff::default();
        diff2.trie_updates.account_nodes.insert(path2, node2.clone());
        store.store_trie_updates(block_2, diff2).await.unwrap();

        // In block 3, path1 is deleted (stored as None in the database)
        // This happens when we store trie updates with path1 mapped to None
        let block_3 = BlockWithParent::new(block_2.block.hash, NumHash::new(3, B256::random()));
        // Simulate storing a deletion by directly writing to DB
        store
            .env
            .update(|tx| {
                let mut cursor = tx.new_cursor::<AccountTrieHistory>()?;
                let vv = VersionedValue { block_number: 3, value: MaybeDeleted(None) };
                cursor.upsert(StoredNibbles::from(path1), &vv)?;

                // Record in change set
                let mut change_set_cursor = tx.new_cursor::<BlockChangeSet>()?;
                change_set_cursor.upsert(
                    3,
                    &ChangeSet {
                        account_trie_keys: vec![StoredNibbles::from(path1)],
                        storage_trie_keys: vec![],
                        hashed_account_keys: vec![],
                        hashed_storage_keys: vec![],
                    },
                )?;

                // Update proof window
                let mut proof_window_cursor = tx.new_cursor::<ProofWindow>()?;
                proof_window_cursor.upsert(
                    ProofWindowKey::LatestBlock,
                    &BlockNumberHash::new(3, block_3.block.hash),
                )?;
                Ok::<(), DatabaseError>(())
            })
            .unwrap()
            .unwrap();

        // Now prune to block 5, with the new initial state:
        // - path1 should be in removed_nodes (it was deleted in block 3)
        // - path2 should be included with its value (it still exists from block 2)
        let block_5 = BlockWithParent::new(B256::random(), NumHash::new(5, B256::random()));
        let mut prune_diff = BlockStateDiff::default();
        prune_diff.trie_updates.removed_nodes.insert(path1);
        prune_diff.trie_updates.account_nodes.insert(path2, node2.clone());
        store.prune_earliest_state(block_5, prune_diff).await.unwrap();

        // Verify that all entries for path1 before block 5 were removed
        let tx = store.env.tx().unwrap();
        let mut cur = tx.cursor_dup_read::<AccountTrieHistory>().unwrap();

        // path1 at block 1 should be gone
        assert!(
            cur.seek_by_key_subkey(StoredNibbles::from(path1), 1).unwrap().is_none(),
            "path1 at block 1 should be pruned"
        );
        // path1 at block 3 (deletion) should also be gone
        assert!(
            cur.seek_by_key_subkey(StoredNibbles::from(path1), 3).unwrap().is_none(),
            "path1 at block 3 should be pruned"
        );

        // path2 entries should be pruned (blocks < 5)
        assert!(
            cur.seek_by_key_subkey(StoredNibbles::from(path2), 2).unwrap().is_none(),
            "path2 at block 2 should be pruned"
        );

        // path2 should exist at block 0 as part of the new initial state
        let vv = cur
            .seek_by_key_subkey(StoredNibbles::from(path2), 0)
            .unwrap()
            .expect("path2 should exist at block 0");
        assert_eq!(vv.block_number, 0);
        assert_eq!(vv.value.0, Some(node2));
    }

    #[tokio::test]
    async fn test_prune_earliest_state_overlapping_keys() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");
        store.set_earliest_block_number(0, B256::ZERO).await.unwrap();

        // Use different addresses - addr1 in old history, addr2 in new initial state
        // This reflects the real-world use case where pruning replaces old account history
        // with a new set of accounts as the initial state
        let addr1 = B256::random();
        let addr2 = B256::random();
        let acc1 = Account { nonce: 1, balance: U256::from(100), ..Default::default() };
        let acc2 = Account { nonce: 2, balance: U256::from(200), ..Default::default() };
        let new_acc = Account { nonce: 10, balance: U256::from(500), ..Default::default() };

        let block_1 = BlockWithParent::new(B256::ZERO, NumHash::new(1, B256::random()));
        let mut diff1 = BlockStateDiff::default();
        diff1.post_state.accounts.insert(addr1, Some(acc1));
        store.store_trie_updates(block_1, diff1).await.unwrap();

        let block_2 = BlockWithParent::new(block_1.block.hash, NumHash::new(2, B256::random()));
        let mut diff2 = BlockStateDiff::default();
        diff2.post_state.accounts.insert(addr1, Some(acc2));
        store.store_trie_updates(block_2, diff2).await.unwrap();

        // Prune to block 3, with new initial state including:
        // - addr1 with its final value (acc2) from block 2
        // - addr2 as a new account
        let block_3 = BlockWithParent::new(block_2.block.hash, NumHash::new(3, B256::random()));
        let mut prune_diff = BlockStateDiff::default();
        prune_diff.post_state.accounts.insert(addr1, Some(acc2));
        prune_diff.post_state.accounts.insert(addr2, Some(new_acc));
        store.prune_earliest_state(block_3, prune_diff).await.unwrap();

        // Verify old versions of addr1 were pruned
        let tx = store.env.tx().unwrap();
        let mut cur = tx.new_cursor::<HashedAccountHistory>().unwrap();
        assert!(
            cur.seek_by_key_subkey(addr1, 1).unwrap().is_none(),
            "Block 1 entry should be pruned"
        );
        assert!(
            cur.seek_by_key_subkey(addr1, 2).unwrap().is_none(),
            "Block 2 entry should be pruned"
        );

        // Verify new initial state at block 0 for addr1 (with final value acc2)
        let vv1 = cur
            .seek_by_key_subkey(addr1, 0)
            .unwrap()
            .expect("addr1 initial state should exist at block 0");
        assert_eq!(vv1.block_number, 0);
        assert_eq!(vv1.value.0, Some(acc2));

        // Verify new initial state at block 0 for addr2
        let vv2 = cur
            .seek_by_key_subkey(addr2, 0)
            .unwrap()
            .expect("New initial state should exist at block 0");
        assert_eq!(vv2.block_number, 0);
        assert_eq!(vv2.value.0, Some(new_acc));
    }

    #[tokio::test]
    async fn test_prune_earliest_state_comprehensive() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");
        store.set_earliest_block_number(0, B256::ZERO).await.unwrap();

        // Setup complex scenario with accounts, storage, and trie nodes
        // Use addr1 for old history, addr2 for new initial state
        let addr1 = B256::random();
        let addr2 = B256::random();
        let slot1 = B256::random();
        let slot2 = B256::random();
        let path1 = Nibbles::from_nibbles_unchecked([0x01]);
        let path2 = Nibbles::from_nibbles_unchecked([0x02]);
        let storage_path1 = Nibbles::from_nibbles_unchecked([0x03]);
        let storage_path2 = Nibbles::from_nibbles_unchecked([0x04]);

        let acc1 = Account { nonce: 1, balance: U256::from(100), ..Default::default() };
        let node1 = BranchNodeCompact::new(0b1, 0, 0, vec![], Some(B256::random()));
        let storage_node1 = BranchNodeCompact::new(0b10, 0, 0, vec![], Some(B256::random()));

        // Block 1: Insert account, trie node, and storage for addr1
        let block_1 = BlockWithParent::new(B256::ZERO, NumHash::new(1, B256::random()));
        let mut diff1 = BlockStateDiff::default();
        diff1.post_state.accounts.insert(addr1, Some(acc1));
        diff1.trie_updates.account_nodes.insert(path1, node1.clone());
        let mut storage1 = HashedStorage::default();
        storage1.storage.insert(slot1, U256::from(1234));
        diff1.post_state.storages.insert(addr1, storage1.clone());
        let mut storage_updates1 = StorageTrieUpdates::default();
        storage_updates1.storage_nodes.insert(storage_path1, storage_node1.clone());
        diff1.trie_updates.storage_tries.insert(addr1, storage_updates1.clone());
        store.store_trie_updates(block_1, diff1).await.unwrap();

        // Block 2: Update account
        let acc2 = Account { nonce: 2, balance: U256::from(200), ..Default::default() };
        let block_2 = BlockWithParent::new(block_1.block.hash, NumHash::new(2, B256::random()));
        let mut diff2 = BlockStateDiff::default();
        diff2.post_state.accounts.insert(addr1, Some(acc2));
        store.store_trie_updates(block_2, diff2).await.unwrap();

        // Prune to block 3 with new initial state for DIFFERENT keys (addr2, path2, etc.)
        let new_acc = Account { nonce: 10, balance: U256::from(1000), ..Default::default() };
        let new_node = BranchNodeCompact::new(0b11, 0, 0, vec![], Some(B256::random()));
        let new_storage_node = BranchNodeCompact::new(0b100, 0, 0, vec![], Some(B256::random()));

        let block_3 = BlockWithParent::new(block_2.block.hash, NumHash::new(3, B256::random()));
        let mut prune_diff = BlockStateDiff::default();
        prune_diff.post_state.accounts.insert(addr1, Some(acc2));
        prune_diff.post_state.accounts.insert(addr2, Some(new_acc));
        prune_diff.trie_updates.account_nodes.insert(path1, node1.clone());
        prune_diff.trie_updates.account_nodes.insert(path2, new_node.clone());
        prune_diff.post_state.storages.insert(addr1, storage1);
        prune_diff.trie_updates.storage_tries.insert(addr1, storage_updates1);

        let mut new_storage = HashedStorage::default();
        new_storage.storage.insert(slot2, U256::from(9999));
        prune_diff.post_state.storages.insert(addr2, new_storage);
        let mut new_storage_updates = StorageTrieUpdates::default();
        new_storage_updates.storage_nodes.insert(storage_path2, new_storage_node.clone());
        prune_diff.trie_updates.storage_tries.insert(addr2, new_storage_updates);

        store.prune_earliest_state(block_3, prune_diff).await.unwrap();

        let tx = store.env.tx().unwrap();

        // Verify account history - old addr1 entries pruned
        let mut acc_cur = tx.new_cursor::<HashedAccountHistory>().unwrap();
        assert!(
            acc_cur.seek_by_key_subkey(addr1, 1).unwrap().is_none(),
            "Old account entries should be pruned"
        );
        assert!(
            acc_cur.seek_by_key_subkey(addr1, 2).unwrap().is_none(),
            "Old account entries should be pruned"
        );
        // New addr2 entry at block 0
        let new_acc_vv =
            acc_cur.seek_by_key_subkey(addr2, 0).unwrap().expect("New account at block 0");
        assert_eq!(new_acc_vv.value.0, Some(new_acc));

        // Verify account trie history - old path1 pruned, new path2 at block 0
        let mut trie_cur = tx.cursor_dup_read::<AccountTrieHistory>().unwrap();
        assert!(
            trie_cur.seek_by_key_subkey(StoredNibbles::from(path1), 1).unwrap().is_none(),
            "Old trie entry should be pruned"
        );
        let new_trie_vv = trie_cur
            .seek_by_key_subkey(StoredNibbles::from(path2), 0)
            .unwrap()
            .expect("New trie at block 0");
        assert_eq!(new_trie_vv.value.0, Some(new_node));

        // Verify storage history - old addr1/slot1 pruned, new addr2/slot2 at block 0
        let mut storage_cur = tx.new_cursor::<HashedStorageHistory>().unwrap();
        let old_storage_key = HashedStorageKey::new(addr1, slot1);
        assert!(
            storage_cur.seek_by_key_subkey(old_storage_key, 1).unwrap().is_none(),
            "Old storage should be pruned"
        );
        let new_storage_key = HashedStorageKey::new(addr2, slot2);
        let new_storage_vv = storage_cur
            .seek_by_key_subkey(new_storage_key, 0)
            .unwrap()
            .expect("New storage at block 0");
        assert_eq!(new_storage_vv.value.0.as_ref().unwrap().0, U256::from(9999));

        // Verify storage trie history - old addr1/storage_path1 pruned, new addr2/storage_path2 at
        // block 0
        let mut storage_trie_cur = tx.cursor_dup_read::<StorageTrieHistory>().unwrap();
        let old_storage_trie_key = StorageTrieKey::new(addr1, StoredNibbles::from(storage_path1));
        assert!(
            storage_trie_cur.seek_by_key_subkey(old_storage_trie_key, 1).unwrap().is_none(),
            "Old storage trie should be pruned"
        );
        let new_storage_trie_key = StorageTrieKey::new(addr2, StoredNibbles::from(storage_path2));
        let new_storage_trie_vv = storage_trie_cur
            .seek_by_key_subkey(new_storage_trie_key, 0)
            .unwrap()
            .expect("New storage trie at block 0");
        assert_eq!(new_storage_trie_vv.value.0, Some(new_storage_node));

        // Verify change sets pruned
        let mut change_cur = tx.new_cursor::<BlockChangeSet>().unwrap();
        assert!(change_cur.seek_exact(1).unwrap().is_none());
        assert!(change_cur.seek_exact(2).unwrap().is_none());
    }

    #[test]
    fn test_block_change_set_crud_operations() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");
        let tx = store.env.tx_mut().expect("rw tx");
        let mut cursor = tx.cursor_write::<BlockChangeSet>().expect("cursor");

        let block_1 = 42u64;
        let block_2 = 43u64;

        let entry1 = ChangeSet {
            account_trie_keys: vec![StoredNibbles::default()],
            storage_trie_keys: vec![],
            hashed_account_keys: vec![B256::ZERO],
            hashed_storage_keys: vec![],
        };
        let entry2 = ChangeSet {
            account_trie_keys: vec![],
            storage_trie_keys: vec![StorageTrieKey::new(B256::ZERO, StoredNibbles::default())],
            hashed_account_keys: vec![],
            hashed_storage_keys: vec![HashedStorageKey::new(B256::ZERO, B256::ZERO)],
        };

        // Insert entries
        cursor.insert(block_1, &entry1).unwrap();
        cursor.insert(block_2, &entry2).unwrap();

        // Read entries
        let mut walker = cursor.walk(Some(block_1)).unwrap();
        let mut entries = vec![walker.next().unwrap().unwrap().1];
        if let Some(Ok((_, val))) = walker.next() {
            entries.push(val);
        }
        entries.sort();
        let mut expected = vec![entry1.clone(), entry2.clone()];
        expected.sort();
        assert_eq!(entries, expected);

        // Delete entry1
        let mut walker = cursor.walk(Some(block_1)).unwrap();
        while let Some(Ok((_, val))) = walker.next() {
            if val == entry1 {
                walker.delete_current().unwrap();
                break;
            }
        }

        // Verify delete
        let mut walker = cursor.walk(Some(block_1)).unwrap();
        assert_eq!(walker.next().unwrap().unwrap().1, entry2);
        assert!(walker.next().is_none());
    }

    #[tokio::test]
    async fn store_trie_updates_deleted_account_trie() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        const BLOCK: BlockWithParent =
            BlockWithParent::new(B256::ZERO, NumHash::new(7, B256::ZERO));

        // Prepare a BlockStateDiff that removes an account trie node at `acc_path`
        let acc_path = Nibbles::from_nibbles_unchecked([0x0A, 0x0B, 0x0C]);
        let mut diff = BlockStateDiff::default();
        diff.trie_updates.removed_nodes.insert(acc_path);

        store.store_trie_updates(BLOCK, diff).await.expect("store");

        // Verify deletion was written at BLOCK
        let tx = store.env.tx().expect("tx");
        let mut cur = tx.new_cursor::<AccountTrieHistory>().expect("cursor");
        let vv = cur
            .seek_by_key_subkey(StoredNibbles::from(acc_path), BLOCK.block.number)
            .expect("seek")
            .expect("exists");
        assert_eq!(vv.block_number, BLOCK.block.number);
        assert!(vv.value.0.is_none(), "expected account trie deletion");
    }

    #[tokio::test]
    async fn store_trie_updates_deleted_storage_trie() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        const BLOCK: BlockWithParent =
            BlockWithParent::new(B256::ZERO, NumHash::new(8, B256::ZERO));

        // Prepare a BlockStateDiff that removes a storage trie node for `addr` at `st_path`
        let addr = B256::from([0xAB; 32]);
        let st_path = Nibbles::from_nibbles_unchecked([0x01, 0x02, 0x03]);

        let mut diff = BlockStateDiff::default();
        let mut st_updates = reth_trie::updates::StorageTrieUpdates::default();
        // mark this storage trie node as removed
        st_updates.removed_nodes.insert(st_path);
        diff.trie_updates.storage_tries.insert(addr, st_updates);

        store.store_trie_updates(BLOCK, diff).await.expect("store");

        // Verify deletion was written at BLOCK
        let tx = store.env.tx().expect("tx");
        let mut cur = tx.new_cursor::<StorageTrieHistory>().expect("cursor");
        let key = StorageTrieKey::new(addr, StoredNibbles::from(st_path));
        let vv = cur.seek_by_key_subkey(key, BLOCK.block.number).expect("seek").expect("exists");
        assert_eq!(vv.block_number, BLOCK.block.number);
        assert!(vv.value.0.is_none(), "expected storage trie deletion");
    }

    #[tokio::test]
    async fn store_trie_updates_wiped_storage_trie_nodes() {
        use reth_trie::updates::StorageTrieUpdates;

        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        let addr_wiped = B256::from([0x10; 32]);
        let addr_live = B256::from([0xF0; 32]);

        // Seed some storage-trie nodes at block 0 for the address that will be wiped.
        let p1 = Nibbles::from_nibbles_unchecked([0x01, 0x02]);
        let p2 = Nibbles::from_nibbles_unchecked([0x0A, 0x0B, 0x0C]);
        let n1 = BranchNodeCompact::default();
        let n2 = BranchNodeCompact::default();

        store
            .store_storage_branches(
                addr_wiped,
                vec![(p1, Some(n1.clone())), (p2, Some(n2.clone()))],
            )
            .await
            .expect("seed wiped addr trie nodes");

        // Build a BlockStateDiff that wipes addr_wiped's storage trie, and
        // also adds a normal storage-trie node for addr_live.
        const BLOCK: BlockWithParent =
            BlockWithParent::new(B256::ZERO, NumHash::new(123, B256::ZERO));
        let mut diff = BlockStateDiff::default();

        // Wipe for addr_wiped
        let mut wiped_updates = StorageTrieUpdates::default();
        wiped_updates.set_deleted(true);
        diff.trie_updates.storage_tries.insert(addr_wiped, wiped_updates);

        // Normal update for addr_live
        let live_path = Nibbles::from_nibbles_unchecked([0xEE, 0xFF]);
        let live_node = BranchNodeCompact::default();
        let mut live_updates = StorageTrieUpdates::default();
        live_updates.storage_nodes.insert(live_path, live_node.clone());
        diff.trie_updates.storage_tries.insert(addr_live, live_updates);

        // Execute the store
        store.store_trie_updates(BLOCK, diff).await.expect("store");

        // Verify: for addr_wiped, each previously existing path now has a deletion tombstone at
        // BLOCK.
        {
            let tx = store.env.tx().expect("tx");
            let mut cur = tx.new_cursor::<StorageTrieHistory>().expect("cursor");

            for path in [p1, p2] {
                let key = StorageTrieKey::new(addr_wiped, StoredNibbles::from(path));
                let vv =
                    cur.seek_by_key_subkey(key, BLOCK.block.number).expect("seek").expect("exists");
                assert_eq!(vv.block_number, BLOCK.block.number);
                assert!(
                    vv.value.0.is_none(),
                    "expected tombstone at wipe block for path {:?}",
                    path
                );
            }
        }

        // Verify: addr_live got its normal node written at BLOCK (not a deletion).
        {
            let tx = store.env.tx().expect("tx");
            let mut cur = tx.new_cursor::<StorageTrieHistory>().expect("cursor");

            let key = StorageTrieKey::new(addr_live, StoredNibbles::from(live_path));
            let vv =
                cur.seek_by_key_subkey(key, BLOCK.block.number).expect("seek").expect("exists");
            assert_eq!(vv.block_number, BLOCK.block.number);
            assert!(vv.value.0.is_some(), "expected normal node for non-wiped address at BLOCK");
        }
    }

    #[tokio::test]
    async fn store_trie_updates_wiped_storage() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        // We'll pre-seed storage at block 0, then issue a wipe at BLOCK.
        const BLOCK: BlockWithParent =
            BlockWithParent::new(B256::ZERO, NumHash::new(42, B256::ZERO));

        let addr = B256::from([0x55; 32]);
        let s1 = B256::from([0x01; 32]);
        let s2 = B256::from([0x02; 32]);
        let v1 = U256::from(111u64);
        let v2 = U256::from(222u64);

        // Seed prior storage (block_number = 0 in store_hashed_storages)
        store.store_hashed_storages(addr, vec![(s1, v1), (s2, v2)]).await.expect("seed");

        // Build BlockStateDiff that marks this address as wiped at BLOCK
        let mut diff = BlockStateDiff::default();

        let wiped = reth_trie::HashedStorage::new(true);

        diff.post_state.storages.insert(addr, wiped);

        // Execute
        store.store_trie_updates(BLOCK, diff).await.expect("store");

        // Verify: for each pre-existing slot, there should be a tombstone (MaybeDeleted(None)) at
        // BLOCK.
        let tx = store.env.tx().expect("tx");
        let mut cur = tx.new_cursor::<HashedStorageHistory>().expect("cursor");

        for slot in [s1, s2] {
            let key = HashedStorageKey::new(addr, slot);
            let vv =
                cur.seek_by_key_subkey(key, BLOCK.block.number).expect("seek").expect("exists");
            assert_eq!(vv.block_number, BLOCK.block.number);
            assert!(
                vv.value.0.is_none(),
                "expected deletion tombstone for slot {:?} at block {}",
                slot,
                BLOCK.block.number,
            );
        }
    }

    #[tokio::test]
    async fn store_trie_updates_wiped_and_non_wiped_mixed_order() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        // Choose addresses so that wiped < non_wiped in sort order (your impl sorts by address)
        let addr_wiped = B256::from([0x01; 32]); // will sort first
        let addr_live = B256::from([0xF0; 32]); // will sort later

        // Slots & values
        let ws1 = B256::from([0xA1; 32]);
        let ws2 = B256::from([0xA2; 32]);
        let wv1 = U256::from(111u64);
        let wv2 = U256::from(222u64);

        let ls1 = B256::from([0xB1; 32]);
        let lv1_old = U256::from(333u64);
        let lv1_new = U256::from(999u64); // will be written at BLOCK

        // Seed prior storage at block 0 for BOTH addresses
        store
            .store_hashed_storages(addr_wiped, vec![(ws1, wv1), (ws2, wv2)])
            .await
            .expect("seed wiped addr");
        store.store_hashed_storages(addr_live, vec![(ls1, lv1_old)]).await.expect("seed live addr");

        // Build diff: wiped first (by address sort), then non-wiped with a write
        const BLOCK: BlockWithParent =
            BlockWithParent::new(B256::ZERO, NumHash::new(77, B256::ZERO));
        let mut diff = BlockStateDiff::default();

        // Wiped storage for addr_wiped
        let wiped = reth_trie::HashedStorage::new(true);
        diff.post_state.storages.insert(addr_wiped, wiped);

        // Non-wiped storage for addr_live (append new value)
        let mut live = reth_trie::HashedStorage::default();
        live.storage.insert(ls1, lv1_new);
        diff.post_state.storages.insert(addr_live, live);

        // Execute
        store.store_trie_updates(BLOCK, diff).await.expect("store");

        // Verify: wiped address got tombstones at BLOCK for each pre-existing slot
        {
            let tx = store.env.tx().expect("tx");
            let mut cur = tx.new_cursor::<HashedStorageHistory>().expect("cursor");
            for slot in [ws1, ws2] {
                let key = HashedStorageKey::new(addr_wiped, slot);
                let vv =
                    cur.seek_by_key_subkey(key, BLOCK.block.number).expect("seek").expect("exists");
                assert_eq!(vv.block_number, BLOCK.block.number);
                assert!(
                    vv.value.0.is_none(),
                    "expected deletion tombstone for wiped slot {:?} at block {}",
                    slot,
                    BLOCK.block.number,
                );
            }
        }

        // Verify: non-wiped address got the new value at BLOCK (not a deletion)
        {
            let tx = store.env.tx().expect("tx");
            let mut cur = tx.new_cursor::<HashedStorageHistory>().expect("cursor");
            let key = HashedStorageKey::new(addr_live, ls1);
            let vv =
                cur.seek_by_key_subkey(key, BLOCK.block.number).expect("seek").expect("exists");
            assert_eq!(vv.block_number, BLOCK.block.number);
            let inner = vv.value.0.as_ref().expect("Some(StorageValue)");
            assert_eq!(inner.0, lv1_new, "expected updated value for non-wiped address");
        }
    }

    #[tokio::test]
    async fn test_proof_window() {
        let dir = TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        // Test initial state (no values set)
        let initial_value = store.get_earliest_block_number().await.expect("get earliest");
        assert_eq!(initial_value, None);

        // Test setting the value
        let block_number = 42u64;
        let hash = B256::random();
        store.set_earliest_block_number(block_number, hash).await.expect("set earliest");

        // Verify value was stored correctly
        let retrieved = store.get_earliest_block_number().await.expect("get earliest");
        assert_eq!(retrieved, Some((block_number, hash)));

        // Test updating with new values
        let new_block_number = 100u64;
        let new_hash = B256::random();
        store.set_earliest_block_number(new_block_number, new_hash).await.expect("update earliest");

        // Verify update worked
        let updated = store.get_earliest_block_number().await.expect("get updated earliest");
        assert_eq!(updated, Some((new_block_number, new_hash)));

        // Verify that latest_block falls back to earliest when not set
        let latest = store.get_latest_block_number().await.expect("get latest");
        assert_eq!(
            latest,
            Some((new_block_number, new_hash)),
            "Latest block should fall back to earliest when not explicitly set"
        );
    }

    #[tokio::test]
    async fn replace_updates_prunes_and_adds_new_chain() {
        let dir = tempfile::TempDir::new().unwrap();
        let store = MdbxProofsStorage::new(dir.path()).expect("env");

        // Test address and helper to make diffs with distinct nonces.
        let addr = B256::from([0xAB; 32]);
        let make_diff = |nonce: u64| {
            let mut d = BlockStateDiff::default();
            d.post_state.accounts.insert(addr, Some(Account { nonce, ..Default::default() }));
            d
        };

        // --- Build initial canonical chain: 1 -> 2 -> 3 ---
        let b1 = BlockWithParent::new(B256::ZERO, NumHash::new(1, B256::random()));
        let b2 = BlockWithParent::new(b1.block.hash, NumHash::new(2, B256::random()));
        let b3 = BlockWithParent::new(b2.block.hash, NumHash::new(3, B256::random()));

        store.store_trie_updates(b1, make_diff(10)).await.expect("store b1");
        store.store_trie_updates(b2, make_diff(20)).await.expect("store b2");
        store.store_trie_updates(b3, make_diff(30)).await.expect("store b3");

        // Sanity: entries for 1,2,3 exist with expected nonces.
        {
            let tx = store.env.tx().expect("tx");
            let mut cur = tx.new_cursor::<HashedAccountHistory>().expect("cursor");
            let v1 = cur.seek_by_key_subkey(addr, 1).expect("seek").expect("exists");
            let v2 = cur.seek_by_key_subkey(addr, 2).expect("seek").expect("exists");
            let v3 = cur.seek_by_key_subkey(addr, 3).expect("seek").expect("exists");
            assert_eq!(v1.value.0.unwrap().nonce, 10);
            assert_eq!(v2.value.0.unwrap().nonce, 20);
            assert_eq!(v3.value.0.unwrap().nonce, 30);
        }

        // --- Reorg at LCA = 2: prune >2, then add 3' and 4' ---
        let b3p = BlockWithParent::new(b2.block.hash, NumHash::new(3, B256::random())); // 3'
        let b4p = BlockWithParent::new(b3p.block.hash, NumHash::new(4, B256::random())); // 4'

        // Build blocks_to_add (HashMap). Order is not guaranteed.
        let mut blocks_to_add = HashMap::default();
        blocks_to_add.insert(b3p, make_diff(300)); // new value at height 3
        blocks_to_add.insert(b4p, make_diff(400)); // new value at height 4

        store.replace_updates(2, blocks_to_add).await.expect("replace_updates succeeds");

        // --- Verify post-conditions ---

        // 1) HashedAccountHistory: blocks 1,2 remain; block 3 was replaced (nonce=300); block 4
        //    exists (nonce=400).
        {
            let tx = store.env.tx().expect("tx");
            let mut cur = tx.new_cursor::<HashedAccountHistory>().expect("cursor");

            let v1 = cur.seek_by_key_subkey(addr, 1).expect("seek").expect("exists");
            assert_eq!(v1.value.0.unwrap().nonce, 10);

            let v2 = cur.seek_by_key_subkey(addr, 2).expect("seek").expect("exists");
            assert_eq!(v2.value.0.unwrap().nonce, 20);

            // Old block 3 (nonce=30) should have been pruned; we should now have a fresh entry at 3
            // with nonce=300.
            let v3_new = cur.seek_by_key_subkey(addr, 3).expect("seek").expect("replaced exists");
            assert_eq!(
                v3_new.value.0.unwrap().nonce,
                300,
                "block 3 should be replaced by new chain"
            );

            // New block 4 should exist.
            let v4 = cur.seek_by_key_subkey(addr, 4).expect("seek").expect("exists");
            assert_eq!(v4.value.0.unwrap().nonce, 400);
        }

        // 2) BlockChangeSet contains exactly entries for block numbers {1,2,3,4}.
        {
            let tx = store.env.tx().expect("tx");
            let mut cur = tx.new_cursor::<BlockChangeSet>().expect("cursor");
            let mut seen = std::collections::BTreeSet::new();
            let mut it = cur.walk(Some(1)).expect("walk");
            while let Some(Ok((bn, _))) = it.next() {
                seen.insert(bn);
            }
            let expected: std::collections::BTreeSet<u64> = [1u64, 2, 3, 4].into_iter().collect();
            assert_eq!(seen, expected, "BlockChangeSet should reflect pruned+new chain");
        }
    }
}

//! Read-only helpers for [`MdbxProofsProviderV2`].

use super::MdbxProofsProviderV2;
use crate::{
    OpProofsStorageError, OpProofsStorageResult,
    api::ProofWindowRange,
    db::{
        ProofWindowKey, V2ProofWindow,
        models::{
            BlockNumberHashedAddress, V2AccountTrieChangeSets, V2AccountsTrie,
            V2HashedAccountChangeSets, V2HashedAccounts, V2HashedStorageChangeSets,
            V2HashedStorages, V2StorageTrieChangeSets, V2StoragesTrie,
        },
    },
};
use alloy_eips::{BlockNumHash, NumHash};
use alloy_primitives::{B256, U256};
use reth_db::{
    cursor::{DbCursorRO, DbDupCursorRO},
    transaction::DbTx,
};
use reth_trie::{HashedPostState, StoredNibbles, StoredNibblesSubKey, updates::TrieUpdates};

impl<TX: DbTx> MdbxProofsProviderV2<TX> {
    pub(super) fn get_block_number_hash_inner(
        &self,
        key: ProofWindowKey,
    ) -> OpProofsStorageResult<NumHash> {
        let mut cursor = self.tx.cursor_read::<V2ProofWindow>()?;
        cursor
            .seek_exact(key)?
            .map(|(_, val)| NumHash::new(val.number(), *val.hash()))
            .ok_or(OpProofsStorageError::NoBlocksFound)
    }

    /// Returns `true` when `max_block_number` is >= the latest stored block,
    /// meaning the current-state tables are authoritative and history/changeset
    /// lookups can be skipped entirely.
    pub(super) fn is_latest_block(&self, max_block_number: u64) -> OpProofsStorageResult<bool> {
        match self.get_block_number_hash_inner(ProofWindowKey::LatestBlock) {
            Ok(latest) => Ok(max_block_number >= latest.number),
            // No blocks stored yet → current state is empty but correct.
            Err(OpProofsStorageError::NoBlocksFound) => Ok(true),
            Err(err) => Err(err),
        }
    }

    pub(super) fn get_proof_window_inner(&self) -> OpProofsStorageResult<ProofWindowRange> {
        let mut cursor = self.tx.cursor_read::<V2ProofWindow>()?;
        let earliest = match cursor.seek_exact(ProofWindowKey::EarliestBlock)? {
            Some((_, val)) => NumHash::new(val.number(), *val.hash()),
            None => return Err(OpProofsStorageError::NoBlocksFound),
        };
        let latest = match cursor.seek_exact(ProofWindowKey::LatestBlock)? {
            Some((_, val)) => NumHash::new(val.number(), *val.hash()),
            None => return Err(OpProofsStorageError::NoBlocksFound),
        };
        Ok(ProofWindowRange { earliest, latest })
    }

    pub(super) fn get_initial_state_anchor_inner(
        &self,
    ) -> OpProofsStorageResult<Option<BlockNumHash>> {
        let mut cur = self.tx.cursor_read::<V2ProofWindow>()?;
        Ok(cur.seek_exact(ProofWindowKey::InitialStateAnchor)?.map(|(_k, v)| v.into()))
    }

    /// Walk `V2AccountTrieChangeSets` for `block_number` and populate `updates`
    /// with the current node (or mark as removed) for each changed path.
    fn populate_account_trie_updates(
        &self,
        block_number: u64,
        updates: &mut TrieUpdates,
    ) -> OpProofsStorageResult<()> {
        let mut acct_state = self.tx.cursor_read::<V2AccountsTrie>()?;
        let mut cs = self.tx.cursor_read::<V2AccountTrieChangeSets>()?;
        let mut walker = cs.walk(Some(block_number))?;
        while let Some(Ok((bn, entry))) = walker.next() &&
            bn == block_number
        {
            let path = entry.nibbles.0;
            match acct_state.seek_exact(StoredNibbles(path))?.map(|(_, n)| n) {
                Some(node) => {
                    updates.account_nodes.insert(path, node);
                }
                None => {
                    updates.removed_nodes.insert(path);
                }
            }
        }
        Ok(())
    }

    /// Walk `V2StorageTrieChangeSets` for `block_number` and populate `updates`
    /// with the current node (or mark as removed) for each changed (address, nibble) pair.
    fn populate_storage_trie_updates(
        &self,
        block_number: u64,
        updates: &mut TrieUpdates,
    ) -> OpProofsStorageResult<()> {
        let mut stor_state = self.tx.cursor_dup_read::<V2StoragesTrie>()?;
        let mut cs = self.tx.cursor_read::<V2StorageTrieChangeSets>()?;
        let blk_range = BlockNumberHashedAddress((block_number, B256::ZERO))..=
            BlockNumberHashedAddress((block_number, B256::repeat_byte(0xff)));
        let mut walker = cs.walk_range(blk_range)?;
        while let Some(Ok((key, entry))) = walker.next() {
            let hashed_address = key.0.1;
            let subkey = StoredNibblesSubKey(entry.nibbles.0);
            let current_node = stor_state
                .seek_by_key_subkey(hashed_address, subkey.clone())?
                .filter(|e| e.nibbles == subkey)
                .map(|e| e.node);
            let stu = updates.storage_tries.entry(hashed_address).or_default();
            match current_node {
                Some(node) => {
                    stu.storage_nodes.insert(entry.nibbles.0, node);
                }
                None => {
                    stu.removed_nodes.insert(entry.nibbles.0);
                }
            }
        }
        Ok(())
    }

    /// Reconstruct [`TrieUpdates`] for a block by reading changeset + current state tables.
    pub(super) fn fetch_block_trie_updates(
        &self,
        block_number: u64,
    ) -> OpProofsStorageResult<TrieUpdates> {
        let mut updates = TrieUpdates::default();
        self.populate_account_trie_updates(block_number, &mut updates)?;
        self.populate_storage_trie_updates(block_number, &mut updates)?;
        Ok(updates)
    }

    /// Walk `V2HashedAccountChangeSets` for `block_number` and populate `post_state`
    /// with the current account (or `None` if deleted) for each changed address.
    fn populate_hashed_accounts(
        &self,
        block_number: u64,
        post_state: &mut HashedPostState,
    ) -> OpProofsStorageResult<()> {
        let mut acct_state = self.tx.cursor_read::<V2HashedAccounts>()?;
        let mut cs = self.tx.cursor_read::<V2HashedAccountChangeSets>()?;
        let mut walker = cs.walk(Some(block_number))?;
        while let Some(Ok((bn, entry))) = walker.next() &&
            bn == block_number
        {
            let current = acct_state.seek_exact(entry.hashed_address)?.map(|(_, a)| a);
            post_state.accounts.insert(entry.hashed_address, current);
        }
        Ok(())
    }

    /// Walk `V2HashedStorageChangeSets` for `block_number` and populate `post_state`
    /// with the current slot value for each changed (address, slot) pair.
    fn populate_hashed_storages(
        &self,
        block_number: u64,
        post_state: &mut HashedPostState,
    ) -> OpProofsStorageResult<()> {
        let mut stor_state = self.tx.cursor_dup_read::<V2HashedStorages>()?;
        let mut cs = self.tx.cursor_read::<V2HashedStorageChangeSets>()?;
        let blk_range = BlockNumberHashedAddress((block_number, B256::ZERO))..=
            BlockNumberHashedAddress((block_number, B256::repeat_byte(0xff)));
        let mut walker = cs.walk_range(blk_range)?;
        while let Some(Ok((key, entry))) = walker.next() {
            let hashed_address = key.0.1;
            let current_value = stor_state
                .seek_by_key_subkey(hashed_address, entry.key)?
                .filter(|e| e.key == entry.key)
                .map(|e| e.value)
                .unwrap_or(U256::ZERO);
            post_state
                .storages
                .entry(hashed_address)
                .or_default()
                .storage
                .insert(entry.key, current_value);
        }
        Ok(())
    }

    /// Reconstruct [`HashedPostState`] for a block by reading changeset + current state tables.
    pub(super) fn fetch_block_post_state(
        &self,
        block_number: u64,
    ) -> OpProofsStorageResult<HashedPostState> {
        let mut post_state = HashedPostState::default();
        self.populate_hashed_accounts(block_number, &mut post_state)?;
        self.populate_hashed_storages(block_number, &mut post_state)?;
        Ok(post_state)
    }
}

//! [`OpProofsSnapshotProviderRW`] implementation for [`MdbxProofsProviderV2`].
//!
//! Per-iteration backfill writer surface. The init-time writer
//! ([`OpProofsSnapshotInitProvider`](crate::api::OpProofsSnapshotInitProvider))
//! lives in [`super::snapshot_init`].

use super::MdbxProofsProviderV2;
use crate::{
    BlockStateDiff, OpProofsStorageError, OpProofsStorageResult,
    api::{OpProofsSnapshotProviderRW, WriteCounts},
    db::{
        SnapshotMeta, SnapshotMetaKey, SnapshotStatus,
        models::{
            V2AccountsTrieSnapshot, V2HashedAccountsSnapshot, V2HashedStoragesSnapshot,
            V2SnapshotMeta, V2StoragesTrieSnapshot,
        },
    },
};
use alloy_eips::BlockNumHash;
use reth_db::{
    cursor::{DbCursorRO, DbCursorRW, DbDupCursorRO, DbDupCursorRW},
    transaction::{DbTx, DbTxMut},
};
use reth_primitives_traits::StorageEntry;
use reth_trie::{
    HashedPostStateSorted, StorageTrieEntry, StoredNibbles, StoredNibblesSubKey,
    updates::TrieUpdatesSorted,
};
use std::fmt::Debug;

impl<TX: DbTxMut> MdbxProofsProviderV2<TX> {
    /// Upsert the singleton row in [`V2SnapshotMeta`].
    ///
    /// Shared helper used by both
    /// [`OpProofsSnapshotInitProvider`](crate::api::OpProofsSnapshotInitProvider)
    /// (init lifecycle transitions) and
    /// [`OpProofsSnapshotProviderRW::update_snapshot`] (per-iteration anchor
    /// updates).
    pub(super) fn write_snapshot_meta(&self, meta: SnapshotMeta) -> OpProofsStorageResult<()> {
        let mut cur = self.tx.cursor_write::<V2SnapshotMeta>()?;
        cur.upsert(SnapshotMetaKey::Singleton, &meta)?;
        Ok(())
    }
}

/// Per-phase projection helpers behind
/// [`OpProofsSnapshotProviderRW::update_snapshot`]. Each method projects one
/// table's worth of diff onto the snapshot and returns the row count it
/// applied (`upsert + delete`).
impl<TX: DbTxMut> MdbxProofsProviderV2<TX> {
    /// Project the account-trie portion of `trie_updates` onto
    /// [`V2AccountsTrieSnapshot`]. `(path, Some)` upserts, `(path, None)`
    /// deletes if present.
    fn write_account_trie_snapshot(
        &self,
        trie_updates: &TrieUpdatesSorted,
    ) -> OpProofsStorageResult<u64> {
        let mut count = 0u64;
        let mut cur = self.tx.cursor_write::<V2AccountsTrieSnapshot>()?;
        for (nibbles, maybe_node) in trie_updates.account_nodes_ref() {
            let key = StoredNibbles(*nibbles);
            match maybe_node {
                Some(node) => cur.upsert(key, node)?,
                None => {
                    if cur.seek_exact(key)?.is_some() {
                        cur.delete_current()?;
                    }
                }
            }
            count += 1;
        }
        Ok(count)
    }

    /// Project the storage-trie portion of `trie_updates` onto
    /// [`V2StoragesTrieSnapshot`].
    fn write_storage_trie_snapshot(
        &self,
        trie_updates: &TrieUpdatesSorted,
    ) -> OpProofsStorageResult<u64> {
        let mut count = 0u64;
        let mut cur = self.tx.cursor_dup_write::<V2StoragesTrieSnapshot>()?;
        for (hashed_address, nodes) in trie_updates.storage_tries_ref() {
            if nodes.is_deleted && cur.seek_exact(*hashed_address)?.is_some() {
                cur.delete_current_duplicates()?;
                count += 1;
            }
            for (nibbles, maybe_node) in nodes.storage_nodes_ref() {
                let subkey = StoredNibblesSubKey(*nibbles);
                let existing = cur
                    .seek_by_key_subkey(*hashed_address, subkey.clone())?
                    .filter(|e| e.nibbles == subkey)
                    .is_some();
                if existing {
                    cur.delete_current()?;
                }
                if let Some(node) = maybe_node {
                    cur.upsert(
                        *hashed_address,
                        &StorageTrieEntry { nibbles: subkey, node: node.clone() },
                    )?;
                }
                count += 1;
            }
        }
        Ok(count)
    }

    /// Project the account-leaf portion of `hashed_post_state` onto
    /// [`V2HashedAccountsSnapshot`].
    fn write_hashed_accounts_snapshot(
        &self,
        hashed_post_state: &HashedPostStateSorted,
    ) -> OpProofsStorageResult<u64> {
        let mut count = 0u64;
        let mut cur = self.tx.cursor_write::<V2HashedAccountsSnapshot>()?;
        for (hashed_addr, maybe_acct) in hashed_post_state.accounts() {
            match maybe_acct {
                Some(acct) => cur.upsert(*hashed_addr, acct)?,
                None => {
                    if cur.seek_exact(*hashed_addr)?.is_some() {
                        cur.delete_current()?;
                    }
                }
            }
            count += 1;
        }
        Ok(count)
    }

    /// Project the storage-leaf portion of `hashed_post_state` onto
    /// [`V2HashedStoragesSnapshot`].
    fn write_hashed_storages_snapshot(
        &self,
        hashed_post_state: &HashedPostStateSorted,
    ) -> OpProofsStorageResult<u64> {
        let mut count = 0u64;
        let mut cur = self.tx.cursor_dup_write::<V2HashedStoragesSnapshot>()?;
        for (hashed_addr, storage) in hashed_post_state.account_storages() {
            if storage.wiped && cur.seek_exact(*hashed_addr)?.is_some() {
                cur.delete_current_duplicates()?;
                count += 1;
            }
            for (slot, value) in &storage.storage_slots {
                let existing = cur
                    .seek_by_key_subkey(*hashed_addr, *slot)?
                    .filter(|e| e.key == *slot)
                    .is_some();
                if existing {
                    cur.delete_current()?;
                }
                if !value.is_zero() {
                    cur.upsert(*hashed_addr, &StorageEntry { key: *slot, value: *value })?;
                }
                count += 1;
            }
        }
        Ok(count)
    }
}

impl<TX: DbTxMut + DbTx + Send + Sync + Debug + 'static> OpProofsSnapshotProviderRW
    for MdbxProofsProviderV2<TX>
{
    fn clear_snapshot(&self) -> OpProofsStorageResult<()> {
        self.tx.clear::<V2AccountsTrieSnapshot>()?;
        self.tx.clear::<V2StoragesTrieSnapshot>()?;
        self.tx.clear::<V2HashedAccountsSnapshot>()?;
        self.tx.clear::<V2HashedStoragesSnapshot>()?;
        self.tx.clear::<V2SnapshotMeta>()?;
        Ok(())
    }

    fn update_snapshot(
        &self,
        new_anchor: BlockNumHash,
        diff: &BlockStateDiff,
    ) -> OpProofsStorageResult<WriteCounts> {
        // Refuse to mutate a Building snapshot: its rows are still being
        // populated, so applying a diff against it would corrupt the result.
        let SnapshotMeta { status, .. } = self.read_snapshot_meta()?;
        if status != SnapshotStatus::Ready {
            return Err(OpProofsStorageError::SnapshotUpdateNotReady { status });
        }

        let counts = WriteCounts {
            account_trie_updates_written_total: self
                .write_account_trie_snapshot(&diff.sorted_trie_updates)?,
            storage_trie_updates_written_total: self
                .write_storage_trie_snapshot(&diff.sorted_trie_updates)?,
            hashed_accounts_written_total: self
                .write_hashed_accounts_snapshot(&diff.sorted_post_state)?,
            hashed_storages_written_total: self
                .write_hashed_storages_snapshot(&diff.sorted_post_state)?,
        };

        // Advance the anchor atomically with the diff (status stays Ready).
        self.write_snapshot_meta(SnapshotMeta::new(new_anchor, SnapshotStatus::Ready))?;

        Ok(counts)
    }

    fn commit(self) -> OpProofsStorageResult<()> {
        self.tx.commit()?;
        Ok(())
    }
}

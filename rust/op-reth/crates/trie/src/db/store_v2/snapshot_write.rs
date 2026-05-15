//! [`OpProofsSnapshotProviderRw`] implementation for [`MdbxProofsProviderV2`].
//!
//! Per-iteration backfill writer surface. The init-time writer
//! ([`OpProofsSnapshotInitProvider`](crate::api::OpProofsSnapshotInitProvider))
//! lives in [`super::snapshot_init`].

use super::MdbxProofsProviderV2;
use crate::{
    OpProofsStorageError, OpProofsStorageResult,
    api::OpProofsSnapshotProviderRW,
    db::{
        SnapshotMeta, SnapshotMetaKey, SnapshotStatus,
        models::{V2AccountsTrieSnapshot, V2SnapshotMeta, V2StoragesTrieSnapshot},
    },
};
use alloy_eips::BlockNumHash;
use reth_db::{
    cursor::{DbCursorRO, DbCursorRW, DbDupCursorRO},
    transaction::{DbTx, DbTxMut},
};
use reth_trie::{StorageTrieEntry, StoredNibbles, StoredNibblesSubKey, updates::TrieUpdatesSorted};
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

impl<TX: DbTxMut + DbTx + Send + Sync + Debug + 'static> OpProofsSnapshotProviderRW
    for MdbxProofsProviderV2<TX>
{
    fn clear_snapshot(&self) -> OpProofsStorageResult<()> {
        self.tx.clear::<V2AccountsTrieSnapshot>()?;
        self.tx.clear::<V2StoragesTrieSnapshot>()?;
        self.tx.clear::<V2SnapshotMeta>()?;
        Ok(())
    }

    fn update_snapshot(
        &self,
        new_anchor: BlockNumHash,
        trie_updates: &TrieUpdatesSorted,
    ) -> OpProofsStorageResult<u64> {
        // Refuse to mutate a Building snapshot: its rows are still being
        // populated, so applying a diff against it would corrupt the result.
        let SnapshotMeta { status, .. } = self.read_snapshot_meta()?;
        if status != SnapshotStatus::Ready {
            return Err(OpProofsStorageError::SnapshotUpdateNotReady { status });
        }

        let mut count = 0u64;

        // Account trie diff.
        let mut acc = self.tx.cursor_write::<V2AccountsTrieSnapshot>()?;
        for (nibbles, maybe_node) in trie_updates.account_nodes_ref() {
            let key = StoredNibbles(*nibbles);
            match maybe_node {
                Some(node) => acc.upsert(key, node)?,
                None => {
                    if acc.seek_exact(key)?.is_some() {
                        acc.delete_current()?;
                    }
                }
            }
            count += 1;
        }

        // Storage trie diff.
        let mut stor = self.tx.cursor_dup_write::<V2StoragesTrieSnapshot>()?;
        for (hashed_address, nodes) in trie_updates.storage_tries_ref() {
            for (nibbles, maybe_node) in nodes.storage_nodes_ref() {
                let subkey = StoredNibblesSubKey(*nibbles);
                let existing = stor
                    .seek_by_key_subkey(*hashed_address, subkey.clone())?
                    .filter(|e| e.nibbles == subkey)
                    .is_some();
                if existing {
                    stor.delete_current()?;
                }
                if let Some(node) = maybe_node {
                    stor.upsert(
                        *hashed_address,
                        &StorageTrieEntry { nibbles: subkey, node: node.clone() },
                    )?;
                }
                count += 1;
            }
        }

        // Advance the anchor atomically with the diff (status stays Ready).
        self.write_snapshot_meta(SnapshotMeta::new(new_anchor, SnapshotStatus::Ready))?;

        Ok(count)
    }

    fn commit(self) -> OpProofsStorageResult<()> {
        self.tx.commit()?;
        Ok(())
    }
}

//! [`OpProofsSnapshotInitProvider`] implementation for [`MdbxProofsProviderV2`].
//!
//! Mirrors `init.rs`'s role for [`OpProofsInitProvider`](crate::api::OpProofsInitProvider):
//! this module is the one place where all init-time operations on the snapshot
//! tables live — bulk source reads, append-only destination writes, anchor
//! recovery, meta transitions, and `clear`.

use super::MdbxProofsProviderV2;
use crate::{
    OpProofsStorageError, OpProofsStorageResult,
    api::{OpProofsSnapshotInitProvider, SnapshotInitAnchor, SnapshotInitStatus},
    db::{
        SnapshotMeta, SnapshotMetaKey, SnapshotStatus, StorageTrieKey,
        models::{V2AccountsTrieSnapshot, V2SnapshotMeta, V2StoragesTrieSnapshot},
    },
};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use reth_db::{
    cursor::{DbCursorRO, DbCursorRW, DbDupCursorRW},
    transaction::{DbTx, DbTxMut},
};
use reth_trie::{BranchNodeCompact, Nibbles, StorageTrieEntry, StoredNibbles, StoredNibblesSubKey};
use std::fmt::Debug;

impl<TX: DbTxMut + DbTx + Send + Sync + Debug + 'static> OpProofsSnapshotInitProvider
    for MdbxProofsProviderV2<TX>
{
    fn snapshot_init_anchor(&self) -> OpProofsStorageResult<SnapshotInitAnchor> {
        let (block, status) = match self.read_snapshot_meta() {
            Ok(SnapshotMeta { anchor, status: SnapshotStatus::Building }) => {
                (Some(anchor), SnapshotInitStatus::InProgress)
            }
            Ok(SnapshotMeta { anchor, status: SnapshotStatus::Ready }) => {
                (Some(anchor), SnapshotInitStatus::Completed)
            }
            Err(OpProofsStorageError::SnapshotNotInitialized) => {
                (None, SnapshotInitStatus::NotStarted)
            }
            Err(e) => return Err(e),
        };

        let last_account_trie_key =
            self.tx.cursor_read::<V2AccountsTrieSnapshot>()?.last()?.map(|(k, _)| k);

        let last_storage_trie_key = self
            .tx
            .cursor_dup_read::<V2StoragesTrieSnapshot>()?
            .last()?
            .map(|(addr, entry)| StorageTrieKey::new(addr, StoredNibbles(entry.nibbles.0)));

        Ok(SnapshotInitAnchor { block, status, last_account_trie_key, last_storage_trie_key })
    }

    fn set_snapshot_init_anchor(&self, anchor: BlockNumHash) -> OpProofsStorageResult<()> {
        let mut cur = self.tx.cursor_write::<V2SnapshotMeta>()?;
        cur.insert(
            SnapshotMetaKey::Singleton,
            &SnapshotMeta::new(anchor, SnapshotStatus::Building),
        )?;
        Ok(())
    }

    fn store_account_trie_snapshot_branches(
        &self,
        entries: Vec<(StoredNibbles, BranchNodeCompact)>,
    ) -> OpProofsStorageResult<()> {
        if entries.is_empty() {
            return Ok(());
        }
        let mut cur = self.tx.cursor_write::<V2AccountsTrieSnapshot>()?;
        for (key, node) in entries {
            cur.append(key, &node)?;
        }
        Ok(())
    }

    fn store_storage_trie_snapshot_branches(
        &self,
        hashed_address: B256,
        storage_nodes: Vec<(Nibbles, Option<BranchNodeCompact>)>,
    ) -> OpProofsStorageResult<()> {
        if storage_nodes.is_empty() {
            return Ok(());
        }
        let mut cur = self.tx.cursor_dup_write::<V2StoragesTrieSnapshot>()?;
        for (nibbles, maybe_node) in storage_nodes {
            if let Some(node) = maybe_node {
                cur.append_dup(
                    hashed_address,
                    StorageTrieEntry { nibbles: StoredNibblesSubKey(nibbles), node },
                )?;
            }
        }
        Ok(())
    }

    fn commit_snapshot(&self) -> OpProofsStorageResult<()> {
        let SnapshotMeta { anchor, status } = self.read_snapshot_meta()?;
        if status != SnapshotStatus::Building {
            return Err(OpProofsStorageError::SnapshotCommitInvalidStatus { status });
        }
        self.write_snapshot_meta(SnapshotMeta::new(anchor, SnapshotStatus::Ready))
    }

    fn commit(self) -> OpProofsStorageResult<()> {
        self.tx.commit()?;
        Ok(())
    }
}

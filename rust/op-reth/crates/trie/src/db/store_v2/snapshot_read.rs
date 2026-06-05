//! [`OpProofsSnapshotProviderRO`] implementation for [`MdbxProofsProviderV2`].

use super::{
    MdbxProofsProviderV2,
    cursor::{V2AccountTrieSnapshotCursor, V2StorageTrieSnapshotCursor},
};
use crate::{
    OpProofsStorageError, OpProofsStorageResult,
    api::{OpProofsSnapshotProviderRO, SnapshotInitStatus},
    db::{
        SnapshotMeta, SnapshotMetaKey, SnapshotStatus,
        models::{V2AccountsTrieSnapshot, V2SnapshotMeta, V2StoragesTrieSnapshot},
    },
};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use reth_db::{cursor::DbCursorRO, transaction::DbTx};
use std::fmt::Debug;

impl<TX: DbTx> MdbxProofsProviderV2<TX> {
    /// Read the singleton row from [`V2SnapshotMeta`], or
    /// [`OpProofsStorageError::SnapshotNotInitialized`] if absent.
    ///
    /// Internal helper for V2 write paths that need to verify or mutate the
    /// current lifecycle state. External reads go through
    /// [`OpProofsSnapshotProviderRO::snapshot_anchor`].
    pub(super) fn read_snapshot_meta(&self) -> OpProofsStorageResult<SnapshotMeta> {
        let mut cursor = self.tx.cursor_read::<V2SnapshotMeta>()?;
        cursor
            .seek_exact(SnapshotMetaKey::Singleton)?
            .map(|(_, meta)| meta)
            .ok_or(OpProofsStorageError::SnapshotNotInitialized)
    }
}

impl<TX: DbTx + Send + Sync + Debug + 'static> OpProofsSnapshotProviderRO
    for MdbxProofsProviderV2<TX>
{
    type SnapshotAccountTrieCursor<'tx>
        = V2AccountTrieSnapshotCursor<TX::Cursor<V2AccountsTrieSnapshot>>
    where
        Self: 'tx,
        TX: 'tx;

    type SnapshotStorageTrieCursor<'tx>
        = V2StorageTrieSnapshotCursor<TX::DupCursor<V2StoragesTrieSnapshot>>
    where
        Self: 'tx,
        TX: 'tx;

    fn snapshot_anchor(&self) -> OpProofsStorageResult<BlockNumHash> {
        match self.read_snapshot_meta() {
            Ok(SnapshotMeta { anchor, status: SnapshotStatus::Ready }) => Ok(anchor),
            Ok(SnapshotMeta { status: SnapshotStatus::Building, .. }) => {
                Err(OpProofsStorageError::SnapshotNotReady {
                    status: SnapshotInitStatus::InProgress,
                })
            }
            Err(OpProofsStorageError::SnapshotNotInitialized) => {
                Err(OpProofsStorageError::SnapshotNotReady {
                    status: SnapshotInitStatus::NotStarted,
                })
            }
            Err(e) => Err(e),
        }
    }

    fn snapshot_account_trie_cursor<'tx>(
        &self,
    ) -> OpProofsStorageResult<Self::SnapshotAccountTrieCursor<'tx>> {
        Ok(V2AccountTrieSnapshotCursor::new(self.tx.cursor_read::<V2AccountsTrieSnapshot>()?))
    }

    fn snapshot_storage_trie_cursor<'tx>(
        &self,
        hashed_address: B256,
    ) -> OpProofsStorageResult<Self::SnapshotStorageTrieCursor<'tx>> {
        Ok(V2StorageTrieSnapshotCursor::new(
            self.tx.cursor_dup_read::<V2StoragesTrieSnapshot>()?,
            hashed_address,
        ))
    }
}

//! [`OpProofsSnapshotProviderRO`] implementation for [`MdbxProofsProvider`].

use super::{
    MdbxProofsProvider,
    cursor::{
        MdbxAccountTrieSnapshotCursor, MdbxHashedAccountSnapshotCursor,
        MdbxHashedStorageSnapshotCursor, MdbxStorageTrieSnapshotCursor,
    },
};
use crate::{
    OpProofsStorageError, OpProofsStorageResult,
    api::{OpProofsSnapshotProviderRO, SnapshotInitStatus},
    db::{
        SnapshotMeta, SnapshotMetaKey, SnapshotStatus,
        models::{
            V2AccountsTrieSnapshot, V2HashedAccountsSnapshot, V2HashedStoragesSnapshot,
            V2SnapshotMeta, V2StoragesTrieSnapshot,
        },
    },
};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use reth_db::{cursor::DbCursorRO, transaction::DbTx};
use std::fmt::Debug;

impl<TX: DbTx> MdbxProofsProvider<TX> {
    /// Read the singleton row from [`V2SnapshotMeta`], or
    /// [`OpProofsStorageError::SnapshotNotInitialized`] if absent.
    ///
    /// Internal helper for write paths that need to verify or mutate the
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
    for MdbxProofsProvider<TX>
{
    type SnapshotAccountTrieCursor<'tx>
        = MdbxAccountTrieSnapshotCursor<TX::Cursor<V2AccountsTrieSnapshot>>
    where
        Self: 'tx,
        TX: 'tx;

    type SnapshotStorageTrieCursor<'tx>
        = MdbxStorageTrieSnapshotCursor<TX::DupCursor<V2StoragesTrieSnapshot>>
    where
        Self: 'tx,
        TX: 'tx;

    type SnapshotHashedAccountCursor<'tx>
        = MdbxHashedAccountSnapshotCursor<TX::Cursor<V2HashedAccountsSnapshot>>
    where
        Self: 'tx,
        TX: 'tx;

    type SnapshotHashedStorageCursor<'tx>
        = MdbxHashedStorageSnapshotCursor<TX::DupCursor<V2HashedStoragesSnapshot>>
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
        Ok(MdbxAccountTrieSnapshotCursor::new(self.tx.cursor_read::<V2AccountsTrieSnapshot>()?))
    }

    fn snapshot_storage_trie_cursor<'tx>(
        &self,
        hashed_address: B256,
    ) -> OpProofsStorageResult<Self::SnapshotStorageTrieCursor<'tx>> {
        Ok(MdbxStorageTrieSnapshotCursor::new(
            self.tx.cursor_dup_read::<V2StoragesTrieSnapshot>()?,
            hashed_address,
        ))
    }

    fn snapshot_hashed_account_cursor<'tx>(
        &self,
    ) -> OpProofsStorageResult<Self::SnapshotHashedAccountCursor<'tx>> {
        Ok(MdbxHashedAccountSnapshotCursor::new(self.tx.cursor_read::<V2HashedAccountsSnapshot>()?))
    }

    fn snapshot_hashed_storage_cursor<'tx>(
        &self,
        hashed_address: B256,
    ) -> OpProofsStorageResult<Self::SnapshotHashedStorageCursor<'tx>> {
        Ok(MdbxHashedStorageSnapshotCursor::new(
            self.tx.cursor_dup_read::<V2HashedStoragesSnapshot>()?,
            hashed_address,
        ))
    }
}

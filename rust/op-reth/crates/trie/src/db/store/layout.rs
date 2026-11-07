//! Version of the history-index layout recorded in the store.
//!
//! The version lets the store carry one-time data migrations: [`MdbxProofsStorage::new`] brings
//! a store up to [`HISTORY_LAYOUT_VERSION`] before anything reads or writes it, and refuses a
//! store written by a newer release.

use super::{MdbxProofsProvider, MdbxProofsStorage};
use crate::{
    OpProofsStorageError, OpProofsStorageResult,
    db::models::{StoreMetaKey, V2StoreMeta},
};
use reth_db::{
    Database,
    transaction::{DbTx, DbTxMut},
};
use std::{cmp::Ordering, path::Path};

/// History-index layout version this release writes and expects.
///
/// - 0: no version recorded; written by a release whose unwind could leave stale shard keys.
/// - 1: stale shard keys re-keyed by the `history_migration` module.
pub const HISTORY_LAYOUT_VERSION: u64 = 1;

impl MdbxProofsStorage {
    /// Brings the store to [`HISTORY_LAYOUT_VERSION`], refusing a store from a newer release.
    pub(super) fn ensure_history_layout(&self, path: &Path) -> OpProofsStorageResult<()> {
        // A read-only check first, so opening a current store never takes the write lock.
        let found = MdbxProofsProvider::new(self.env.tx()?).history_layout_version()?;
        match found.cmp(&HISTORY_LAYOUT_VERSION) {
            Ordering::Equal => Ok(()),
            Ordering::Greater => Err(OpProofsStorageError::NewerHistoryLayout {
                found,
                supported: HISTORY_LAYOUT_VERSION,
                path: path.to_path_buf(),
            }),
            Ordering::Less => self.migrate_history_layout(path).map(drop),
        }
    }
}

impl<TX: DbTx> MdbxProofsProvider<TX> {
    /// Layout version recorded in the store; a store that predates the version records none.
    pub(super) fn history_layout_version(&self) -> OpProofsStorageResult<u64> {
        Ok(self.tx.get::<V2StoreMeta>(StoreMetaKey::HistoryLayoutVersion)?.unwrap_or(0))
    }
}

impl<TX: DbTxMut + DbTx> MdbxProofsProvider<TX> {
    /// Records `version` as the store's layout version.
    pub(super) fn set_history_layout_version(&self, version: u64) -> OpProofsStorageResult<()> {
        Ok(self.tx.put::<V2StoreMeta>(StoreMetaKey::HistoryLayoutVersion, version)?)
    }
}

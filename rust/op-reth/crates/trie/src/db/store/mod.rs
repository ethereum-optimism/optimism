//! MDBX implementation of [`OpProofsStore`].
//!
//! This module uses a **3-table-per-data-type** schema:
//!
//! | Domain | Current State | ChangeSet | History Bitmap |
//! |--------|--------------|-----------|----------------|
//! | Hashed Accounts | `V2HashedAccounts` | `V2HashedAccountChangeSets` | `V2HashedAccountsHistory` |
//! | Hashed Storages | `V2HashedStorages` | `V2HashedStorageChangeSets` | `V2HashedStoragesHistory` |
//! | Account Trie | `V2AccountsTrie` | `V2AccountTrieChangeSets` | `V2AccountsTrieHistory` |
//! | Storage Trie | `V2StoragesTrie` | `V2StorageTrieChangeSets` | `V2StoragesTrieHistory` |

mod backfill;
pub(crate) mod cursor;
mod init;
#[cfg(feature = "metrics")]
mod metrics;
mod provider_ro;
mod provider_rw;
mod read;
mod snapshot_init;
mod snapshot_read;
mod write;

pub use cursor::{
    MdbxAccountCursor, MdbxAccountTrieCursor, MdbxAccountTrieSnapshotCursor,
    MdbxHashedAccountSnapshotCursor, MdbxHashedStorageSnapshotCursor, MdbxStorageCursor,
    MdbxStorageTrieCursor, MdbxStorageTrieSnapshotCursor,
};

#[cfg(test)]
mod snapshot_tests;
#[cfg(test)]
mod tests;

use super::{Tables, V2ProofWindow};
use crate::{
    OpProofsStorageError, OpProofsStorageResult,
    api::{OpProofsBackfillStore, OpProofsStore},
};
use reth_db::{
    Database, DatabaseEnv, DatabaseError,
    mdbx::{self, DatabaseArguments, init_db_for},
    transaction::DbTx,
};
use std::{path::Path, sync::Arc};
use tracing::warn;

/// Maximum number of block indices per shard in history bitmap tables.
pub(super) const NUM_OF_INDICES_IN_SHARD: usize = 2_000;

/// Tables of the removed v1 proofs storage.
///
/// Older releases created these in every proofs database, so they may still exist on disk. They
/// are never read or deleted.
const LEGACY_V1_TABLES: &[&str] = &[
    "AccountTrieHistory",
    "StorageTrieHistory",
    "HashedAccountHistory",
    "HashedStorageHistory",
    "ProofWindow",
    "BlockChangeSet",
];

/// MDBX implementation of [`OpProofsStore`].
///
/// Uses a 3-table-per-data-type schema. Each data domain (accounts, storages,
/// account trie, storage trie) has a current-state table, a changeset table,
/// and a sharded history bitmap table.
#[derive(Debug)]
pub struct MdbxProofsStorage {
    env: DatabaseEnv,
}

impl MdbxProofsStorage {
    /// Creates a new [`MdbxProofsStorage`] instance with the given path.
    pub fn new(path: &Path) -> Result<Self, OpProofsStorageError> {
        let env = init_db_for::<_, Tables>(path, DatabaseArguments::default())
            .map_err(|e| DatabaseError::Other(format!("Failed to open database: {e}")))?;
        match LegacyV1Usage::read(&env) {
            Ok(usage) => usage.warn(path),
            Err(error) => {
                warn!(target: "trie::db", %error, "Failed to check proofs storage for legacy v1 data")
            }
        }
        Ok(Self { env })
    }

    /// Returns whether the database holds legacy v1 data but no v2 data.
    ///
    /// Such a database must be deleted before running `op-reth proofs init`, or the v1 data stays
    /// on disk forever.
    pub fn holds_only_legacy_v1_data(&self) -> Result<bool, DatabaseError> {
        Ok(LegacyV1Usage::read(&self.env)?.holds_only_legacy_v1_data())
    }
}

/// Size of the [`LEGACY_V1_TABLES`] in a proofs database.
#[derive(Debug, Default, PartialEq, Eq)]
struct LegacyV1Usage {
    /// Total number of entries across the legacy tables.
    entries: usize,
    /// Total size of the legacy tables, in bytes.
    size_bytes: usize,
    /// Whether the v2 proof window has data, including an in-progress initialization anchor.
    has_v2_data: bool,
}

impl LegacyV1Usage {
    /// Reads the legacy table usage from `env` in a single read transaction.
    fn read(env: &DatabaseEnv) -> Result<Self, DatabaseError> {
        env.view(|tx| {
            let mut usage =
                Self { has_v2_data: tx.entries::<V2ProofWindow>()? > 0, ..Default::default() };
            for &name in LEGACY_V1_TABLES {
                let table = match tx.inner().open_db(Some(name)) {
                    Ok(table) => table,
                    Err(mdbx::Error::NotFound) => continue,
                    Err(e) => return Err(DatabaseError::Open(e.into())),
                };
                let stat =
                    tx.inner().db_stat(table.dbi()).map_err(|e| DatabaseError::Stats(e.into()))?;
                usage.entries += stat.entries();
                usage.size_bytes += table_size_bytes(&stat);
            }
            Ok(usage)
        })?
    }

    /// Returns whether the legacy tables hold data while the v2 tables do not.
    const fn holds_only_legacy_v1_data(&self) -> bool {
        self.entries > 0 && !self.has_v2_data
    }

    /// Logs a warning if the legacy tables hold data.
    ///
    /// v1 data is ignored, but it still occupies disk space and is not covered by the table-size
    /// metrics.
    fn warn(&self, path: &Path) {
        let Self { entries, size_bytes, has_v2_data } = *self;
        if entries == 0 {
            return;
        }
        if has_v2_data {
            warn!(
                target: "trie::db",
                path = %path.display(),
                entries,
                size_bytes,
                "Proofs storage holds unused legacy v1 data, which still occupies disk space"
            );
        } else {
            warn!(
                target: "trie::db",
                path = %path.display(),
                entries,
                size_bytes,
                "Proofs storage only holds legacy v1 data, which is ignored and still occupies \
                 disk space. To reclaim it, delete the proofs storage directory (or use a fresh \
                 --proofs-history.storage-path) before running `op-reth proofs init`"
            );
        }
    }
}

/// Returns the on-disk size of a table, in bytes, from its MDBX stats.
const fn table_size_bytes(stat: &mdbx::Stat) -> usize {
    stat.page_size() as usize * (stat.leaf_pages() + stat.branch_pages() + stat.overflow_pages())
}

impl OpProofsStore for MdbxProofsStorage {
    type ProviderRO<'a> = Arc<MdbxProofsProvider<<DatabaseEnv as Database>::TX>>;
    type ProviderRw<'a> = MdbxProofsProvider<<DatabaseEnv as Database>::TXMut>;
    type Initializer<'a> = MdbxProofsProvider<<DatabaseEnv as Database>::TXMut>;

    fn provider_ro<'a>(&'a self) -> OpProofsStorageResult<Self::ProviderRO<'a>> {
        Ok(Arc::new(MdbxProofsProvider::new(self.env.tx()?)))
    }

    fn provider_rw<'a>(&'a self) -> OpProofsStorageResult<Self::ProviderRw<'a>> {
        Ok(MdbxProofsProvider::new(self.env.tx_mut()?))
    }

    fn initialization_provider<'a>(&'a self) -> OpProofsStorageResult<Self::Initializer<'a>> {
        Ok(MdbxProofsProvider::new(self.env.tx_mut()?))
    }
}

impl OpProofsBackfillStore for MdbxProofsStorage {
    type BackfillProvider<'a> = MdbxProofsProvider<<DatabaseEnv as Database>::TXMut>;
    type SnapshotProviderRO<'a> = Arc<MdbxProofsProvider<<DatabaseEnv as Database>::TX>>;
    type SnapshotInitializer<'a> = MdbxProofsProvider<<DatabaseEnv as Database>::TXMut>;

    fn backfill_provider<'a>(&'a self) -> OpProofsStorageResult<Self::BackfillProvider<'a>> {
        Ok(MdbxProofsProvider::new(self.env.tx_mut()?))
    }

    fn snapshot_provider_ro<'a>(&'a self) -> OpProofsStorageResult<Self::SnapshotProviderRO<'a>> {
        Ok(Arc::new(MdbxProofsProvider::new(self.env.tx()?)))
    }

    fn snapshot_initialization_provider<'a>(
        &'a self,
    ) -> OpProofsStorageResult<Self::SnapshotInitializer<'a>> {
        Ok(MdbxProofsProvider::new(self.env.tx_mut()?))
    }
}

// =============================================================================
// Provider (Transaction wrapper)
// =============================================================================

/// MDBX provider for proof storage, wrapping a database transaction.
#[derive(Debug)]
pub struct MdbxProofsProvider<TX> {
    pub(super) tx: TX,
}

impl<TX> MdbxProofsProvider<TX> {
    /// Creates a new [`MdbxProofsProvider`].
    pub const fn new(tx: TX) -> Self {
        Self { tx }
    }
}

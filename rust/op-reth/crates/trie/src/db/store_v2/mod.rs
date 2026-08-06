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

use super::Tables;
use crate::{
    OpProofsStorageError, OpProofsStorageResult,
    api::{OpProofsBackfillStore, OpProofsStore},
};
use reth_db::{
    Database, DatabaseEnv, DatabaseError,
    mdbx::{DatabaseArguments, init_db_for},
};
use std::{path::Path, sync::Arc};

/// Maximum number of block indices per shard in history bitmap tables.
pub(super) const NUM_OF_INDICES_IN_SHARD: usize = 2_000;

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
        Ok(Self { env })
    }
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

//! V2 MDBX implementation of [`OpProofsStore`].
//!
//! This module implements the v2 table schema using **3-table-per-data-type** pattern:
//!
//! | Domain | Current State | ChangeSet | History Bitmap |
//! |--------|--------------|-----------|----------------|
//! | Hashed Accounts | `V2HashedAccounts` | `V2HashedAccountChangeSets` | `V2HashedAccountsHistory` |
//! | Hashed Storages | `V2HashedStorages` | `V2HashedStorageChangeSets` | `V2HashedStoragesHistory` |
//! | Account Trie | `V2AccountsTrie` | `V2AccountTrieChangeSets` | `V2AccountsTrieHistory` |
//! | Storage Trie | `V2StoragesTrie` | `V2StorageTrieChangeSets` | `V2StoragesTrieHistory` |

pub(crate) mod cursor;
mod init;
#[cfg(feature = "metrics")]
mod metrics;
mod provider_ro;
mod provider_rw;
mod read;
mod write;

pub use cursor::{V2AccountCursor, V2AccountTrieCursor, V2StorageCursor, V2StorageTrieCursor};

#[cfg(test)]
mod tests;

use super::Tables;
use crate::{OpProofsStorageError, OpProofsStorageResult, api::OpProofsStore};
use reth_db::{
    Database, DatabaseEnv, DatabaseError,
    mdbx::{DatabaseArguments, init_db_for},
};
use std::{path::Path, sync::Arc};

/// Maximum number of block indices per shard in history bitmap tables.
pub(super) const NUM_OF_INDICES_IN_SHARD: usize = 2_000;

/// V2 MDBX implementation of [`OpProofsStore`].
///
/// Uses the v2 3-table-per-data-type schema. Each data domain (accounts, storages,
/// account trie, storage trie) has a current-state table, a changeset table,
/// and a sharded history bitmap table.
#[derive(Debug)]
pub struct MdbxProofsStorageV2 {
    env: DatabaseEnv,
}

impl MdbxProofsStorageV2 {
    /// Creates a new [`MdbxProofsStorageV2`] instance with the given path.
    pub fn new(path: &Path) -> Result<Self, OpProofsStorageError> {
        let env = init_db_for::<_, Tables>(path, DatabaseArguments::default())
            .map_err(|e| DatabaseError::Other(format!("Failed to open database: {e}")))?;
        Ok(Self { env })
    }
}

impl OpProofsStore for MdbxProofsStorageV2 {
    type ProviderRO<'a> = Arc<MdbxProofsProviderV2<<DatabaseEnv as Database>::TX>>;
    type ProviderRw<'a> = MdbxProofsProviderV2<<DatabaseEnv as Database>::TXMut>;
    type Initializer<'a> = MdbxProofsProviderV2<<DatabaseEnv as Database>::TXMut>;

    fn provider_ro<'a>(&'a self) -> OpProofsStorageResult<Self::ProviderRO<'a>> {
        Ok(Arc::new(MdbxProofsProviderV2::new(self.env.tx()?)))
    }

    fn provider_rw<'a>(&'a self) -> OpProofsStorageResult<Self::ProviderRw<'a>> {
        Ok(MdbxProofsProviderV2::new(self.env.tx_mut()?))
    }

    fn initialization_provider<'a>(&'a self) -> OpProofsStorageResult<Self::Initializer<'a>> {
        Ok(MdbxProofsProviderV2::new(self.env.tx_mut()?))
    }
}

// =============================================================================
// Provider (Transaction wrapper)
// =============================================================================

/// V2 MDBX provider for proof storage, wrapping a database transaction.
#[derive(Debug)]
pub struct MdbxProofsProviderV2<TX> {
    pub(super) tx: TX,
}

impl<TX> MdbxProofsProviderV2<TX> {
    /// Creates a new [`MdbxProofsProviderV2`].
    pub const fn new(tx: TX) -> Self {
        Self { tx }
    }
}

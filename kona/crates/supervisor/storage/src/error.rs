use alloy_eips::BlockNumHash;
use reth_db::DatabaseError;
use thiserror::Error;

/// Errors that may occur while interacting with supervisor log storage.
///
/// This enum is used across all implementations of the Storage traits.
#[derive(Debug, Error)]
pub enum StorageError {
    /// Represents a database error that occurred while interacting with storage.
    #[error(transparent)]
    Database(#[from] DatabaseError),

    /// Represents an error that occurred while initializing the database.
    #[error(transparent)]
    DatabaseInit(#[from] eyre::Report),

    /// Represents an error that occurred while writing to the database.
    #[error("lock poisoned")]
    LockPoisoned,

    /// The expected entry was not found in the database.
    #[error(transparent)]
    EntryNotFound(#[from] EntryNotFoundError),

    /// Represents an error that occurred while getting data that is not yet available.
    #[error("data not yet available")]
    FutureData,

    /// Represents an error that occurred when database is not initialized.
    #[error("database not initialized")]
    DatabaseNotInitialised,

    /// Represents a conflict occurred while attempting to write to the database.
    #[error("conflicting data")]
    ConflictError,

    /// Represents an error that occurred while writing to log database.
    #[error("latest stored block is not parent of the incoming block")]
    BlockOutOfOrder,

    /// Represents an error that occurred when there is inconsistency in log storage
    #[error("reorg required due to inconsistent storage state")]
    ReorgRequired,

    /// Represents an error that occurred when attempting to rewind log storage beyond the local
    /// safe head.
    #[error("rewinding log storage beyond local safe head. to: {to}, local_safe: {local_safe}")]
    RewindBeyondLocalSafeHead {
        /// The target block number to rewind to.
        to: u64,
        /// The local safe head block number.
        local_safe: u64,
    },
}

impl PartialEq for StorageError {
    fn eq(&self, other: &Self) -> bool {
        use StorageError::*;
        match (self, other) {
            (Database(a), Database(b)) => a == b,
            (DatabaseInit(a), DatabaseInit(b)) => format!("{a}") == format!("{b}"),
            (EntryNotFound(a), EntryNotFound(b)) => a == b,
            (DatabaseNotInitialised, DatabaseNotInitialised) | (ConflictError, ConflictError) => {
                true
            }
            _ => false,
        }
    }
}

impl Eq for StorageError {}

/// Entry not found error.
#[derive(Debug, Error, PartialEq, Eq)]
pub enum EntryNotFoundError {
    /// No derived blocks found for given source block.
    #[error("no derived blocks for source block, number: {}, hash: {}", .0.number, .0.hash)]
    MissingDerivedBlocks(BlockNumHash),

    /// Expected source block not found.
    #[error("source block not found, number: {0}")]
    SourceBlockNotFound(u64),

    /// Expected derived block not found.
    #[error("derived block not found, number: {0}")]
    DerivedBlockNotFound(u64),

    /// Expected log not found.
    #[error("log not found at block {block_number} index {log_index}")]
    LogNotFound {
        /// Block number.
        block_number: u64,
        /// Log index within the block.
        log_index: u32,
    },
}

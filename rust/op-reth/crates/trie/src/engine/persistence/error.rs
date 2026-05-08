//! Error types for the persistence service.

use crate::{OpProofsStorageError, PrunerError};
use thiserror::Error;

/// Errors returned by the persistence layer.
#[derive(Debug, Error)]
pub enum PersistenceError {
    /// The channel to the persistence service has been closed.
    #[error("persistence service disconnected")]
    Disconnected,
    /// An error from the underlying storage layer.
    #[error(transparent)]
    Storage(#[from] OpProofsStorageError),
    /// An error from the pruner.
    #[error(transparent)]
    Prune(#[from] PrunerError),
}

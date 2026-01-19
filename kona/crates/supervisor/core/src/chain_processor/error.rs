use crate::logindexer::LogIndexerError;
use kona_supervisor_storage::StorageError;
use thiserror::Error;

/// Errors that may occur while processing chains in the supervisor core.
#[derive(Debug, Error, PartialEq, Eq)]
pub enum ChainProcessorError {
    /// Represents an error that occurred while interacting with the storage layer.
    #[error(transparent)]
    StorageError(#[from] StorageError),

    /// Represents an error that occurred while indexing logs.
    #[error(transparent)]
    LogIndexerError(#[from] LogIndexerError),

    /// Represents an error that occurred while sending an event to the channel.
    #[error("failed to send event to channel: {0}")]
    ChannelSendFailed(String),
}

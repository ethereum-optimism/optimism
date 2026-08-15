//! Engine service errors.

use thiserror::Error;

/// Result returned by semantic engine operations.
pub type EngineResult<T> = Result<T, EngineError>;

/// An error returned by the semantic engine service.
#[derive(Debug, Error, Clone, PartialEq, Eq)]
pub enum EngineError {
    /// The underlying engine driver failed to apply an operation.
    #[error("engine driver failed: {0}")]
    Driver(String),
    /// Local payload construction was requested from a follower-only engine driver.
    #[error("local sequencing is disabled")]
    SequencingDisabled,
    /// The engine service is no longer accepting requests.
    #[error("engine service is unavailable")]
    Unavailable,
    /// The engine service stopped before returning a response.
    #[error("engine service dropped its response")]
    ResponseDropped,
}

impl EngineError {
    /// Wraps an implementation-specific driver error.
    pub fn driver(error: impl core::fmt::Display) -> Self {
        Self::Driver(error.to_string())
    }
}

/// A terminal error returned by the engine service task.
#[derive(Debug, Error, Clone, Copy, PartialEq, Eq)]
pub enum EngineServiceError {
    /// Every semantic engine client was dropped before node shutdown.
    #[error("all engine request senders were dropped")]
    RequestChannelClosed,
}

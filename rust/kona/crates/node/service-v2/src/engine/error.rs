//! Errors returned by the execution-engine service.

use thiserror::Error;

/// Result returned by semantic engine operations.
pub type EngineResult<T> = Result<T, EngineError>;

/// An error returned by the semantic engine service.
#[derive(Debug, Error, Clone, PartialEq, Eq)]
pub enum EngineError {
    /// A transient Engine API failure. Retrying the same semantic operation is safe.
    #[error("temporary engine failure: {0}")]
    Temporary(String),
    /// A critical engine invariant or payload conversion failed.
    #[error("critical engine failure: {0}")]
    Critical(String),
    /// A complete unsafe payload was rejected as invalid by the execution engine.
    #[error("invalid unsafe payload: {0}")]
    InvalidUnsafePayload(String),
    /// Safe-chain processing must reset its derivation pipeline before retrying.
    #[error("engine requested derivation reset: {0}")]
    ResetRequired(String),
    /// Safe-chain processing must flush its active derivation channel before retrying.
    #[error("engine requested derivation channel flush: {0}")]
    FlushRequired(String),
    /// A build became stale because the unsafe head changed before it was sealed.
    #[error("unsafe payload build became stale")]
    StaleBuild,
    /// Destructive engine recovery was requested while local sequencing was active.
    #[error("engine recovery is forbidden while local sequencing is active")]
    RecoveryWhileSequencing,
    /// The engine service is no longer accepting requests.
    #[error("engine service is unavailable")]
    Unavailable,
    /// The engine service stopped before returning a response.
    #[error("engine service dropped its response")]
    ResponseDropped,
}

/// A terminal error returned by the engine service task.
#[derive(Debug, Error)]
pub enum EngineServiceError {
    /// Startup forkchoice discovery failed.
    #[error("failed to discover startup forkchoice: {0}")]
    Startup(String),
    /// Every semantic engine client was dropped before node shutdown.
    #[error("all engine request senders were dropped")]
    RequestChannelClosed,
}

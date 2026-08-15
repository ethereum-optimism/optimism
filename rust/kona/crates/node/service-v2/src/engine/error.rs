//! Errors returned by the execution-engine domain.

use thiserror::Error;

/// Result returned by semantic engine operations.
pub type EngineResult<T> = Result<T, EngineError>;

/// An error returned by the semantic Engine service.
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
    /// Derivation must reset its local pipeline before retrying.
    #[error("engine requested derivation reset: {reason}")]
    ResetRequired {
        /// Reason reported by the Engine task.
        reason: String,
        /// Authoritative safe head after Engine-owned recovery, when recovery occurred.
        safe_head: Option<kona_protocol::L2BlockInfo>,
    },
    /// Derivation must flush its active channel before retrying.
    #[error("engine requested derivation channel flush: {reason}")]
    FlushRequired {
        /// Reason reported by the Engine task.
        reason: String,
        /// Current authoritative safe head after fallback processing.
        safe_head: Option<kona_protocol::L2BlockInfo>,
    },
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

/// A terminal error returned by the top-level Engine task.
#[derive(Debug, Error)]
pub enum EngineServiceError {
    /// Startup forkchoice discovery failed.
    #[error("failed to discover startup forkchoice: {0}")]
    Startup(String),
    /// Node stopped waiting for the startup capability handshake.
    #[error("Engine startup receiver was dropped")]
    StartupReceiverDropped,
    /// Every sender for a required Engine request lane was dropped unexpectedly.
    #[error("Engine request channel closed: {0}")]
    RequestChannelClosed(&'static str),
    /// An Engine-private critical child stopped.
    #[error("Engine child {name} failed: {error}")]
    Child {
        /// Private child name.
        name: &'static str,
        /// Child failure description.
        error: String,
    },
    /// An Engine-private task panicked.
    #[error("Engine child {name} panicked: {error}")]
    ChildPanic {
        /// Private child name.
        name: &'static str,
        /// Join failure description.
        error: String,
    },
    /// A lifecycle transition could not be acknowledged.
    #[error("Engine lifecycle transition failed: {0}")]
    Lifecycle(String),
}

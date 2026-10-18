/// The result of an acknowledged engine operation.
pub type EngineClientResult<T> = Result<T, EngineClientError>;

/// Caller actions decided by the engine owner, independent of Engine API error details.
#[derive(Debug, thiserror::Error)]
pub enum EngineClientError {
    /// The engine service stopped.
    #[error("engine service stopped")]
    Closed,
    /// The epoch or parent on which this work was prepared changed.
    #[error("engine context changed")]
    Stale,
    /// Retry the same work while its context remains current.
    #[error("engine dependency unavailable: {0}")]
    Temporary(String),
    /// A sequenced payload was rejected.
    #[error("payload rejected: {0}")]
    Rejected(String),
    /// The execution client no longer has this build job.
    #[error("payload build expired")]
    Expired,
    /// An invariant failed; the owner shuts the node down.
    #[error("engine invariant failed: {0}")]
    Fatal(String),
}

/// Fatal errors leave the actor and stop its peers.
#[derive(Debug, thiserror::Error)]
pub enum EngineError {
    /// A required service channel closed.
    #[error("engine channel closed")]
    ChannelClosed,
    /// An operation violated an invariant.
    #[error("engine invariant failed: {0}")]
    Invariant(String),
}

//! Optimism flashblock errors.

/// Flashblock related errors.
#[derive(Debug, thiserror::Error)]
pub enum OpFlashblockError {
    /// The base payload is required for the initial flashblock (index 0) but was not provided.
    #[error("Missing base payload for initial flashblock")]
    MissingBasePayload,
    /// A base payload was provided for a non-initial flashblock, but only the first flashblock
    /// should contain a base payload.
    #[error("Unexpected base payload for non-initial flashblock")]
    UnexpectedBasePayload,
    /// The delta field is required for flashblocks but was not provided.
    #[error("Missing delta for flashblock")]
    MissingDelta,
    /// The flashblock index is invalid or out of expected range.
    #[error("Invalid index for flashblock")]
    InvalidIndex,
    /// The execution payload is missing from the flashblock.
    #[error("Missing payload")]
    MissingPayload,
}

use kona_protocol::BlockInfo;
use thiserror::Error;

/// The result of an Engine client call.
pub type DerivationClientResult<T> = Result<T, DerivationClientError>;

/// Error making requests to the [`crate::DerivationActor`].
#[derive(Debug, Error)]
pub enum DerivationClientError {
    /// Error making a request to the [`crate::DerivationActor`]. The request never made it there.
    #[error("Error making a request to the derivation actor: {0}.")]
    RequestError(String),

    /// Error receiving response from the [`crate::DerivationActor`].
    /// This means the request may or may not have succeeded.
    #[error("Error receiving response from the derivation actor: {0}..")]
    ResponseError(String),
}

/// Inbound requests that the [`crate::DerivationActor`] can process.
#[derive(Debug)]
pub enum DerivationActorRequest {
    /// A request to process the provided finalized L1 [`BlockInfo`].
    ProcessFinalizedL1Block(Box<BlockInfo>),
    /// Request to process the provided L1 head block update.
    ProcessL1HeadUpdateRequest(Box<BlockInfo>),
}

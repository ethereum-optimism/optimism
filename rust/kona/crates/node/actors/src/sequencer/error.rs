use crate::engine::{BuildTaskError, EngineClientError};
use kona_derive::PipelineErrorKind;

/// An error returned by the actor.
#[derive(Debug, thiserror::Error)]
pub enum ActorError {
    /// An error occurred while building payload attributes.
    #[error(transparent)]
    AttributesBuilder(#[from] PipelineErrorKind),
    /// The origin actor stopped.
    #[error("the origin actor stopped")]
    OriginStopped,
    /// An error occurred communicating with the engine.
    #[error(transparent)]
    EngineError(#[from] EngineClientError),
    /// An error occurred while attempting to build a payload.
    #[error(transparent)]
    BuildError(#[from] BuildTaskError),
    /// An error occurred sending a payload to the signer.
    #[error("{0}")]
    Signer(String),
}

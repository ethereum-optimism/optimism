//! Error type for the derivation actors.

use super::DerivationStateTransitionError;
use crate::EngineClientError;
use kona_derive::PipelineErrorKind;
use thiserror::Error;

/// An error from a derivation actor. Every variant except [`ActorError::Yield`] stops the actor.
#[derive(Error, Debug)]
pub enum ActorError {
    /// An error originating from the derivation pipeline.
    #[error(transparent)]
    Pipeline(#[from] PipelineErrorKind),
    /// Waiting for more data to be available. The actor handles this itself.
    #[error("Waiting for more data to be available")]
    Yield,
    /// A request to the engine actor failed.
    #[error("Failed to send a request to the engine actor: {0}")]
    Sender(#[from] EngineClientError),
    /// Failed to receive inbound request
    #[error("Failed to receive inbound request")]
    RequestReceiveFailed,
    /// An L1 observation publisher closed.
    #[error("l1 observation channel closed")]
    L1ReceiveFailed,
    /// An invalid state transition occurred.
    #[error(transparent)]
    StateTransitionError(#[from] DerivationStateTransitionError),
}

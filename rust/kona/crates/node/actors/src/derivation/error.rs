//! Error type for the derivation actors.

use crate::EngineClientError;
use kona_derive::PipelineErrorKind;
use thiserror::Error;

/// An error that stops a derivation actor.
#[derive(Error, Debug)]
pub enum ActorError {
    /// An error originating from the derivation pipeline.
    #[error(transparent)]
    Pipeline(#[from] PipelineErrorKind),
    /// A request to the engine actor failed.
    #[error("Failed to send a request to the engine actor: {0}")]
    Sender(#[from] EngineClientError),
    /// The L1 actor stopped.
    #[error("the L1 actor stopped")]
    L1Stopped,
}

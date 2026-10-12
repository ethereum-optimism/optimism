//! Errors from requests to the node engine actor.

use crate::{BuildTaskError, SealTaskError};
use thiserror::Error;

/// The result of a request to the node engine actor.
pub type EngineRequestResult<T> = Result<T, EngineRequestError>;

/// Error making requests to the node engine actor.
#[derive(Debug, Error)]
pub enum EngineRequestError {
    /// Error making a request to the engine. The request never made it there.
    #[error("Error making a request to the engine: {0}.")]
    RequestError(String),

    /// Error receiving response from the engine.
    /// This means the request may or may not have succeeded.
    #[error("Error receiving response from the engine: {0}.")]
    ResponseError(String),

    /// An error occurred starting to build a block.
    #[error(transparent)]
    StartBuildError(#[from] BuildTaskError),

    /// An error occurred sealing a block.
    #[error(transparent)]
    SealError(#[from] SealTaskError),

    /// An error occurred performing the reset.
    #[error("An error occurred performing the reset: {0}.")]
    ResetForkchoiceError(String),
}

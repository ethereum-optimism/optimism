//! Error type for the [`EngineActor`].
//!
//! [`EngineActor`]: super::EngineActor

use kona_engine::{EngineClientBuilderError, EngineResetError, EngineTaskErrors};

/// An error from the [`EngineActor`].
///
/// [`EngineActor`]: super::EngineActor
#[derive(thiserror::Error, Debug)]
pub enum EngineError {
    /// Closed channel error.
    #[error("a channel has been closed unexpectedly")]
    ChannelClosed,
    /// Engine reset error.
    #[error(transparent)]
    EngineReset(#[from] EngineResetError),
    /// Engine client builder error.
    #[error(transparent)]
    EngineClientBuilder(#[from] EngineClientBuilderError),
    /// Engine task error.
    #[error(transparent)]
    EngineTask(#[from] EngineTaskErrors),
}

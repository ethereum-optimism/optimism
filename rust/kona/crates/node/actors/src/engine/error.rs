//! Error type for the engine actor.

use kona_engine::{EngineResetError, EngineTaskErrors};

/// A fatal error from the engine actor.
#[derive(thiserror::Error, Debug)]
pub enum ActorError {
    /// Closed channel error.
    #[error("a channel has been closed unexpectedly")]
    ChannelClosed,
    /// Engine reset error.
    #[error(transparent)]
    EngineReset(#[from] EngineResetError),
    /// Engine task error.
    #[error(transparent)]
    EngineTask(#[from] EngineTaskErrors),
}

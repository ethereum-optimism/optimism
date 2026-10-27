//! Error type for the [`EngineActor`].
//!
//! [`EngineActor`]: super::EngineActor

use kona_engine::{EngineResetError, EngineTaskErrors, SynchronizeTaskError};

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
    /// A pending forkchoice retry failed critically.
    #[error(transparent)]
    Forkchoice(#[from] SynchronizeTaskError),
    /// Engine task error.
    #[error(transparent)]
    EngineTask(#[from] EngineTaskErrors),
}

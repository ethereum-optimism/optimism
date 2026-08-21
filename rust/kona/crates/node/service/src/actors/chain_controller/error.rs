//! Error type for the [`ChainController`].
//!
//! [`ChainController`]: super::ChainController

use kona_engine::{EngineResetError, EngineTaskErrors};

/// An error from the [`ChainController`].
///
/// [`ChainController`]: super::ChainController
#[derive(thiserror::Error, Debug)]
pub enum ChainControllerError {
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

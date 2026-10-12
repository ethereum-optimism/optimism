//! Error type for the engine actor.

use kona_engine::{ConsolidateTaskError, EngineResetError, FinalizeTaskError, InsertTaskError};

/// A fatal error from the engine actor.
#[derive(thiserror::Error, Debug)]
pub enum ActorError {
    /// Closed channel error.
    #[error("a channel has been closed unexpectedly")]
    ChannelClosed,
    /// Engine reset error.
    #[error(transparent)]
    EngineReset(#[from] EngineResetError),
    /// A critical error inserting an unsafe block.
    #[error(transparent)]
    Insert(#[from] InsertTaskError),
    /// A critical error consolidating the safe head.
    #[error(transparent)]
    Consolidate(#[from] ConsolidateTaskError),
    /// A critical error finalizing a block.
    #[error(transparent)]
    Finalize(#[from] FinalizeTaskError),
}

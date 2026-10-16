//! Errors from the engine actor and the operations it performs.

mod severity;
pub use severity::{EngineTaskError, EngineTaskErrorSeverity};

mod synchronize;
pub use synchronize::SynchronizeTaskError;

mod insert;
pub use insert::InsertTaskError;

mod build;
pub use build::{BuildTaskError, EngineBuildError};

mod seal;
pub use seal::SealTaskError;

mod consolidate;
pub use consolidate::ConsolidateTaskError;

mod finalize;
pub use finalize::FinalizeTaskError;

mod reset;
pub use reset::EngineResetError;

mod client;
pub use client::{EngineClientError, EngineClientResult};

/// A fatal error from the engine actor.
#[derive(thiserror::Error, Debug)]
pub enum ActorError {
    /// Closed channel error.
    #[error("a channel has been closed unexpectedly")]
    ChannelClosed,
    /// The L1 actor stopped.
    #[error("the L1 actor stopped")]
    L1Stopped,
    /// Safety stopped.
    #[error("safety stopped")]
    SafetyStopped,
    /// Finality stopped.
    #[error("finality stopped")]
    FinalityStopped,
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

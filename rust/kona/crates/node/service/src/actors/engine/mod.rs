//! The [`EngineActor`] and its components.

mod actor;
pub use actor::EngineActor;

mod client;
pub use client::{EngineDerivationClient, QueuedEngineDerivationClient};

mod config;
pub use config::EngineConfig;

mod error;
pub use error::EngineError;

pub use kona_engine::{
    BuildRequest, EngineActorRequest, EngineRequestError as EngineClientError,
    EngineRequestResult as EngineClientResult, ResetRequest, SealRequest,
};

#[cfg(test)]
mod tests;

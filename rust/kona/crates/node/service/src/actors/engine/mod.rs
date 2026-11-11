//! The [`EngineActor`] and its components.

mod actor;
pub use actor::EngineActor;
#[cfg(test)]
pub(crate) use actor::{INITIAL_RETRY_DELAY, MAX_RETRY_DELAY, WAIT_RETRY_INTERVAL};

mod client;
pub use client::{EngineDerivationClient, QueuedEngineDerivationClient};

mod config;
pub use config::EngineConfig;

mod error;
pub use error::EngineError;

pub use kona_engine::{
    BuildRequest, CanonicalizeRequest, EngineActorRequest, EngineRequestError as EngineClientError,
    EngineRequestResult as EngineClientResult, ResetRequest, SealRequest,
};

#[cfg(test)]
mod tests;

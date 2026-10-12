//! Engine actor and handle.

mod actor;
pub use actor::Builder;

mod client;
pub use client::{EngineDerivationClient, QueuedEngineDerivationClient};

mod config;
pub use config::EngineConfig;

mod error;
pub use error::ActorError;

mod handle;
pub use crate::capacity::{Capacity, InvalidCapacity};
pub use handle::Handle;

pub use kona_engine::{
    EngineRequestError as EngineClientError, EngineRequestResult as EngineClientResult,
};

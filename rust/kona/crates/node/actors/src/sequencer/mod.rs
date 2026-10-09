//! Sequencer actor and handle.

mod config;
pub use config::SequencerConfig;

mod origin_selector;
pub use origin_selector::{
    DelayedL1OriginSelectorProvider, L1OriginSelector, L1OriginSelectorError,
    L1OriginSelectorProvider, OriginSelector,
};

mod actor;
pub use actor::Builder;

mod metrics;

mod error;
pub use error::ActorError;

mod conductor;

pub use conductor::{Conductor, ConductorClient, ConductorError};

mod engine_client;
pub use engine_client::{QueuedSequencerEngineClient, SequencerEngineClient};

#[cfg(test)]
pub use conductor::MockConductor;

#[cfg(test)]
pub use engine_client::MockSequencerEngineClient;

#[cfg(test)]
pub use origin_selector::MockOriginSelector;

mod handle;
pub use crate::capacity::{Capacity, InvalidCapacity};
pub use handle::{Handle, HandleError};

mod signer;
pub use signer::Signer;

#[cfg(test)]
pub use signer::MockSigner;

//! Sequencer actor and handle.

mod config;
pub use config::SequencerConfig;

mod origin;
pub use origin::Origin;

mod actor;
pub use actor::Builder;

mod metrics;

mod error;
pub use error::ActorError;

mod conductor;

pub use conductor::{Conductor, ConductorClient, ConductorError};

mod engine_client;
pub use engine_client::SequencerEngineClient;

#[cfg(test)]
pub use conductor::MockConductor;

#[cfg(test)]
pub use engine_client::MockSequencerEngineClient;

#[cfg(test)]
pub use origin::MockOrigin;

mod handle;
pub use handle::{Handle, HandleError};

mod signer;
pub use signer::Signer;

#[cfg(test)]
pub use signer::MockSigner;

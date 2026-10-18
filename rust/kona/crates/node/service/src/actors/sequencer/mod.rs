//! The `SequencerActor` and its components.

mod config;
pub use config::SequencerConfig;

mod origin_selector;
pub use origin_selector::{
    DelayedL1OriginSelectorProvider, L1OriginSelector, L1OriginSelectorError,
    L1OriginSelectorProvider, OriginSelector,
};

mod actor;
pub use actor::SequencerActor;

mod admin_api_impl;

pub(crate) mod handle;
pub use handle::{SequencerHandle, SequencerState};

mod metrics;

mod error;
pub use error::SequencerActorError;

mod conductor;

pub use conductor::{Conductor, ConductorClient, ConductorError};

#[cfg(test)]
pub use conductor::MockConductor;

#[cfg(test)]
pub use origin_selector::MockOriginSelector;

#[cfg(test)]
mod tests;

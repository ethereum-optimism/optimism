//! Engine-private unsafe-chain acquisition and local production.

#![allow(unreachable_pub)]

mod conductor;
#[cfg(test)]
pub(super) use conductor::ConductorError;
pub(super) use conductor::{Conductor, ConductorClient};

mod config;
pub use config::SequencerConfig;

mod control;
pub(super) use control::{SequencerHandle, SequencerStatus, UnsafeLifecycleCommand};

mod origin;
#[cfg(test)]
pub(super) use origin::L1OriginSelectorError;
pub(super) use origin::{DelayedL1OriginSelectorProvider, L1OriginSelector, OriginSelector};

mod sequencer;
pub(super) use sequencer::{SequencingWorkflow, SequencingWorkflowFactory};

mod service;
pub(super) use service::{UnsafeChainService, UnsafeChainServiceError};

#[cfg(test)]
mod tests;

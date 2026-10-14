#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]

#[macro_use]
extern crate tracing;

mod capacity;

pub mod engine;
pub use engine::{EngineClientError, EngineClientResult};

pub mod derivation;
pub use derivation::{DerivationDelegateClientError, DerivationDelegateProvider};

pub mod discovery;

pub mod finality;

pub mod l1_signer_updater;

pub mod l1;

pub mod signer;

pub mod network;
pub use network::NetworkConfig;

pub mod sequencer;

pub use sequencer::{
    Conductor, ConductorClient, ConductorError, DelayedL1OriginSelectorProvider, L1OriginSelector,
    L1OriginSelectorError, L1OriginSelectorProvider, OriginSelector, SequencerConfig,
    SequencerEngineClient,
};

#[cfg(test)]
pub use sequencer::{MockConductor, MockOriginSelector, MockSequencerEngineClient};

mod mode;
pub use mode::NodeMode;

mod metrics;
pub use metrics::Metrics;

pub use derivation::DerivationStatus;

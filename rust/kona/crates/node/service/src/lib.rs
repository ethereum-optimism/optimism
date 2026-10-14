#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]

#[macro_use]
extern crate tracing;

mod service;
pub use service::{
    ChainSpec, ListenProtocol, NodeMode, RollupNode, RollupNodeClients, RollupNodeClientsError,
    RollupNodeConfig, RollupNodeEndpoints, RollupNodeError,
};

pub use kona_node_actors::{
    Conductor, ConductorClient, ConductorError, DelayedL1OriginSelectorProvider,
    DerivationDelegateClientError, DerivationDelegateProvider, EngineClientError,
    EngineClientResult, L1OriginSelector, L1OriginSelectorError, L1OriginSelectorProvider,
    NetworkConfig, OriginSelector, SequencerConfig, SequencerEngineClient, derivation, discovery,
    engine, l1, l1_signer_updater, network, sequencer, signer,
};

pub use kona_node_actors::Metrics;
pub use kona_rpc::DerivationDelegateClient;

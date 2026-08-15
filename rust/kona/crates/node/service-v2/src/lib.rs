#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]

#[macro_use]
extern crate tracing;

/// Semantic execution-engine ownership and reconciliation.
pub mod engine;
/// Shared canonical L1 access.
pub mod l1;
/// Service metrics.
pub mod metrics;
/// P2P discovery and gossip transport.
pub mod network;
/// Node composition and supervision.
pub mod node;
/// JSON-RPC transport and administration routing.
pub mod rpc;
/// Safe-chain derivation and finality.
pub mod safe_chain;
/// Unsafe-chain following and local production.
pub mod unsafe_chain;

pub use engine::EngineConfig;
pub use metrics::Metrics;
pub use network::{NetworkBuilder, NetworkConfig};
pub use node::{
    DerivationDelegateConfig, InteropMode, L1Config, L1ConfigBuilder, NodeMode, RollupNode,
    RollupNodeBuilder,
};
pub use unsafe_chain::SequencerConfig;

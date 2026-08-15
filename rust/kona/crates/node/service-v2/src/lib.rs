#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]

#[macro_use]
extern crate tracing;

/// Safe-chain derivation, finality, and L1 reorg recovery.
pub mod derivation;
/// Execution, unsafe-chain acquisition, and authoritative forkchoice ownership.
pub mod engine;
/// Node-owned canonical L1 access infrastructure.
pub mod l1;
/// Service metrics.
pub mod metrics;
/// P2P transport construction primitives. The running network is owned privately by Engine.
pub mod network;
/// Node composition and explicit lifecycle ownership.
pub mod node;
/// JSON-RPC transport and administration routing.
pub mod rpc;

pub use engine::{EngineConfig, SequencerConfig};
pub use metrics::Metrics;
pub use network::{NetworkBuilder, NetworkConfig};
pub use node::{
    DerivationDelegateConfig, L1Config, L1ConfigBuilder, NodeMode, RollupNode, RollupNodeBuilder,
};

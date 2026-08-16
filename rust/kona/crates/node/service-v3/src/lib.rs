#![doc = include_str!("../DESIGN.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]

mod control;
pub use control::ControlError;

pub mod engine;
pub use engine::Engine;

pub mod safe_chain;
pub use safe_chain::{SafeChainBuilder, SafeChainBuilderError, SafeChainBuilderHandle};

pub mod unsafe_chain;
pub use unsafe_chain::{
    UnsafeChainBuilder, UnsafeChainBuilderError, UnsafeChainBuilderHandle, UnsafeMode,
};

pub mod rpc;
pub use rpc::{Rpc, RpcError, RpcHandle};

pub mod node;
pub use node::{NodeError, RollupNode};

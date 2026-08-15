#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]

/// Semantic execution-engine access and forkchoice reconciliation.
pub mod engine;
/// Shared L1 data access.
pub mod l1;
/// Network transport integration.
pub mod network;
/// Node composition and task supervision.
pub mod node;
/// RPC transport and subsystem control integration.
pub mod rpc;
/// Safe and finalized chain derivation from L1.
pub mod safe_chain;
/// Unsafe chain acquisition through local sequencing or network following.
pub mod unsafe_chain;

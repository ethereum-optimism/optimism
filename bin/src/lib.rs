//! Rust Optimism (op-reth) binary executable.
//!
//! ## Feature Flags
//!
//! - `jemalloc`: Uses [jemallocator](https://github.com/tikv/jemallocator) as the global allocator.
//!   This is **not recommended on Windows**. See [here](https://rust-lang.github.io/rfcs/1974-global-allocators.html#jemalloc)
//!   for more info.
//! - `jemalloc-prof`: Enables [jemallocator's](https://github.com/tikv/jemallocator) heap profiling
//!   and leak detection functionality. See [jemalloc's opt.prof](https://jemalloc.net/jemalloc.3.html#opt.prof)
//!   documentation for usage details. This is **not recommended on Windows**. See [here](https://rust-lang.github.io/rfcs/1974-global-allocators.html#jemalloc)
//!   for more info.
//! - `asm-keccak`: replaces the default, pure-Rust implementation of Keccak256 with one implemented
//!   in assembly; see [the `keccak-asm` crate](https://github.com/DaniPopes/keccak-asm) for more
//!   details and supported targets
//! - `min-error-logs`: Disables all logs below `error` level.
//! - `min-warn-logs`: Disables all logs below `warn` level.
//! - `min-info-logs`: Disables all logs below `info` level. This can speed up the node, since fewer
//!   calls to the logging component are made.
//! - `min-debug-logs`: Disables all logs below `debug` level.
//! - `min-trace-logs`: Disables all logs below `trace` level.
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]

/// Re-exported from `reth_optimism_cli`.
pub mod cli {
    pub use reth_optimism_cli::*;
}

/// Re-exported from `reth_optimism_chainspec`.
pub mod chainspec {
    pub use reth_optimism_chainspec::*;
}

/// Re-exported from `reth_optimism_consensus`.
pub mod consensus {
    pub use reth_optimism_consensus::*;
}

/// Re-exported from `reth_optimism_evm`.
pub mod evm {
    pub use reth_optimism_evm::*;
}

/// Re-exported from `reth_optimism_forks`.
pub mod forks {
    pub use reth_optimism_forks::*;
}

/// Re-exported from `reth_optimism_node`.
pub mod node {
    pub use reth_optimism_node::*;
}

/// Re-exported from `reth_optimism_payload_builder`.
pub mod payload {
    pub use reth_optimism_payload_builder::*;
}

/// Re-exported from `reth_optimism_primitives`.
pub mod primitives {
    pub use reth_optimism_primitives::*;
}

/// Re-exported from `reth_optimism_rpc`.
pub mod rpc {
    pub use reth_optimism_rpc::*;
}

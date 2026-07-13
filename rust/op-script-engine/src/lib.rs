//! Rust forge-script executor for the OP Stack.
//!
//! This crate re-implements the execution semantics of `op-chain-ops/script` (a Foundry-style
//! in-process forge-script runner) on top of revm, so the monorepo can drop its op-geth fork.
//! The cheatcode + console precompiles and prank/broadcast caller-overrides — which in op-geth
//! rely on fork-only `vm.Config` hooks — are re-implemented as a single revm `Inspector`.

pub mod addresses;
pub mod allocs;
pub mod artifacts;
pub mod cheatcodes;
pub mod host;
pub mod precompiles;
pub mod rpc;

pub use allocs::ForgeAllocs;
pub use cheatcodes::Broadcast;
pub use host::{HostConfig, HostError, ScriptContext, ScriptHost};

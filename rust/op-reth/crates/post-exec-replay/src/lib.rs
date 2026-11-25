//! Structural post-exec replay support for op-reth.
//!
//! Replay re-executes a historical block's non-`0x7D` transactions under standard gas accounting to
//! recover each transaction's raw (pre-rebate) gas, then checks the block's embedded post-exec
//! payload against that. It deliberately does **not** recompute refunds: no refund policy ships in
//! the public tree, so there is nothing to recompute against. Producers that run a real policy own
//! the policy-aware replay that reports embedded-versus-recomputed agreement.

#![cfg_attr(not(test), warn(unused_crate_dependencies))]

// Pulls in `reth-optimism-primitives` with `reth-codec` so the transitively-required
// `Compact` impls for `OpPrimitives` exist: this crate's provider/DB dependencies activate
// `reth-primitives-traits/reth-codec`, which demands them.
use reth_optimism_primitives as _;

mod replay;
mod types;

pub use replay::{PostExecReplayError, replay_block, strip_post_exec_tx_for_replay};
pub use types::{
    PostExecReplayBlock, PostExecReplayConfig, PostExecReplayMismatch, PostExecReplayMismatchKind,
    PostExecReplayPayload, PostExecReplayPayloadEntry, PostExecReplaySummary, PostExecReplayTx,
    ReplayPostExecBlockOptions, ReplayPostExecBlockRequest,
};

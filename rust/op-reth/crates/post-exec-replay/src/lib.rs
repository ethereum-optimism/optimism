//! Counterfactual post-exec replay support for op-reth.

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
    PostExecReplayPayload, PostExecReplayPayloadEntry, PostExecReplayRefundEvent,
    PostExecReplayRefundKind, PostExecReplaySummary, PostExecReplayTx, ReplayPostExecBlockOptions,
    ReplayPostExecBlockRequest,
};

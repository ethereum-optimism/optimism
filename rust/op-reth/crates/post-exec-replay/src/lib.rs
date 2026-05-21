//! Counterfactual post-exec replay support for op-reth.

#![cfg_attr(not(test), warn(unused_crate_dependencies))]

mod replay;
mod types;

pub use replay::{PostExecReplayError, replay_block, strip_post_exec_tx_for_replay};
pub use types::{
    PostExecReplayBlock, PostExecReplayConfig, PostExecReplayMismatch, PostExecReplayMismatchKind,
    PostExecReplayPayload, PostExecReplayPayloadEntry, PostExecReplayRefundEvent,
    PostExecReplayRefundKind, PostExecReplaySummary, PostExecReplayTx, ReplayPostExecBlockOptions,
    ReplayPostExecBlockRequest,
};

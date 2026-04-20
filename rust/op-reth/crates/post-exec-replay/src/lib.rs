//! Counterfactual post-exec replay support for op-reth.

#![cfg_attr(not(test), warn(unused_crate_dependencies))]

mod jsonl;
pub mod metrics;
mod replay;
mod types;

pub use jsonl::{PostExecReplayJsonlRecord, write_jsonl_record};
pub use metrics::SDMReplayMetrics;
pub use replay::{PostExecReplayError, replay_block, strip_post_exec_tx_for_replay};
pub use types::{
    PostExecReplayBlock, PostExecReplayConfig, PostExecReplayMismatch, PostExecReplayMismatchKind,
    PostExecReplayMode, PostExecReplayPayload, PostExecReplayPayloadEntry,
    PostExecReplayRefundEvent, PostExecReplayRefundKind, PostExecReplayRunConfig,
    PostExecReplaySummary, PostExecReplayTx, ReplayPostExecBlockOptions,
    ReplayPostExecBlockRequest,
};

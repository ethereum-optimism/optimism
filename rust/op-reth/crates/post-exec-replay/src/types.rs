#![allow(missing_docs)]

use alloy_eips::BlockNumberOrTag;
use alloy_primitives::{Address, B256, Bytes};
use serde::{Deserialize, Serialize};

/// Single-block replay request, accepting either a block tag/number or a block hash.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(untagged)]
pub enum ReplayPostExecBlockRequest {
    /// A block number or tag like `latest`.
    Number(BlockNumberOrTag),
    /// A block hash.
    Hash(B256),
}

/// Options for `debug_replaySDMBlock`.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default)]
pub struct ReplayPostExecBlockOptions {
    /// Compare replay refunds against any embedded post-exec payload in the source block.
    pub compare_payload: bool,
}

impl Default for ReplayPostExecBlockOptions {
    fn default() -> Self {
        Self { compare_payload: true }
    }
}

/// Replay configuration.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct PostExecReplayConfig {
    /// Compare replay refunds against an embedded payload when present.
    pub compare_payload: bool,
}

impl Default for PostExecReplayConfig {
    fn default() -> Self {
        Self { compare_payload: true }
    }
}

impl From<ReplayPostExecBlockOptions> for PostExecReplayConfig {
    fn from(options: ReplayPostExecBlockOptions) -> Self {
        Self { compare_payload: options.compare_payload }
    }
}

/// Exact refund categories emitted by the replay engine.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum PostExecReplayRefundKind {
    /// Warm account rebate (+2500).
    WarmAccount,
    /// Warm storage read rebate (+2000).
    WarmSload,
    /// Warm storage write rebate (+2100).
    WarmSstore,
}

/// Exact refund attribution event for one replayed transaction.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct PostExecReplayRefundEvent {
    /// Replay-local transaction index that claimed the rebate.
    pub claiming_replay_tx_index: u64,
    /// Original transaction index in the source block that claimed the rebate.
    pub claiming_tx_index: u64,
    /// Refund kind.
    pub kind: PostExecReplayRefundKind,
    /// Refund amount in gas.
    pub amount: u64,
    /// Account touched by the rebate.
    pub address: Address,
    /// Storage slot touched by the rebate, when applicable.
    pub slot: Option<B256>,
    /// Replay-local transaction index that first warmed the account or slot.
    pub first_warmed_by_replay_tx_index: u64,
    /// Original transaction index in the source block that first warmed the account or slot.
    pub first_warmed_by_tx_index: u64,
}

/// Per-transaction replay row.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct PostExecReplayTx {
    pub tx_index: u64,
    pub replay_tx_index: u64,
    pub tx_hash: B256,
    pub tx_type: u8,
    pub is_deposit_tx: bool,
    pub raw_gas_used: u64,
    pub canonical_gas_used: u64,
    pub op_gas_refund_replay: u64,
    pub op_gas_refund_payload: Option<u64>,
    pub refund_breakdown: Vec<PostExecReplayRefundEvent>,
    pub mismatch: bool,
}

/// Replay mismatch category.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum PostExecReplayMismatchKind {
    DuplicatePayloadIndex,
    PayloadIndexOutOfRange,
    PayloadTargetsDeposit,
    PayloadTargetsPostExec,
    PayloadRefundMismatch,
    PayloadRefundExceedsRawGas,
}

/// Replay mismatch row.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct PostExecReplayMismatch {
    pub category: PostExecReplayMismatchKind,
    pub block_num: u64,
    pub tx_index: Option<u64>,
    pub expected: Option<u64>,
    pub actual: Option<u64>,
    pub message: String,
}

/// Block-level summary.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct PostExecReplaySummary {
    pub block_num: u64,
    pub block_hash: B256,
    pub tx_count_total: usize,
    pub tx_count_user: usize,
    pub post_exec_tx_present: bool,
    pub post_exec_payload_entry_count: usize,
    pub block_gas_used: u64,
    pub block_raw_gas_used: u64,
    pub replay_refund_total: u64,
    pub payload_refund_total: u64,
    pub mismatch_count: usize,
}

/// Single-block replay response.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct PostExecReplayBlock {
    pub config: PostExecReplayConfig,
    pub block_num: u64,
    pub block_hash: B256,
    pub parent_hash: B256,
    pub post_exec_tx_present: bool,
    pub post_exec_tx_index: Option<u64>,
    pub embedded_payload: Option<PostExecReplayPayload>,
    pub synthesized_payload: PostExecReplayPayload,
    pub synthesized_payload_bytes: Bytes,
    pub txs: Vec<PostExecReplayTx>,
    pub mismatches: Vec<PostExecReplayMismatch>,
    pub summary: PostExecReplaySummary,
}

/// Serializable replay payload entry.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct PostExecReplayPayloadEntry {
    pub index: u64,
    pub gas_refund: u64,
}

/// Serializable replay payload.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct PostExecReplayPayload {
    pub version: u64,
    pub block_number: u64,
    pub gas_refund_entries: Vec<PostExecReplayPayloadEntry>,
}

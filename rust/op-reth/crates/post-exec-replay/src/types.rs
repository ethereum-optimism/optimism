#![allow(missing_docs)]

use alloy_eips::BlockNumberOrTag;
use alloy_primitives::B256;
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
    /// Check any embedded post-exec payload in the source block against replayed raw gas.
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
    /// Check an embedded payload against replayed raw gas when present.
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

/// Per-transaction replay row.
///
/// `raw_gas_used` comes from policy-free re-execution, so it is the transaction's gas cost before
/// any rebate. `canonical_gas_used` is what the block's embedded payload implies the producer
/// charged (`raw_gas_used` less the claimed refund); with no embedded claim the two are equal.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct PostExecReplayTx {
    pub tx_index: u64,
    pub replay_tx_index: u64,
    pub tx_hash: B256,
    pub tx_type: u8,
    pub is_deposit_tx: bool,
    pub raw_gas_used: u64,
    pub canonical_gas_used: u64,
    pub op_gas_refund_payload: Option<u64>,
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
    pub selected_base_fee_per_gas: u64,
    pub gas_refund_entries: Vec<PostExecReplayPayloadEntry>,
}

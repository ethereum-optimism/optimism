//! Observability for SDM replay mismatch detection.
//!
//! Every call into [`replay_block`](crate::replay_block) records one tick against
//! [`SDMReplayMetrics`] — whether or not a mismatch was found — so operators running a
//! follower-style replay loop (e.g. `sdmreplay --fail-on-mismatch`) can alert on both
//! sudden mismatch spikes and silent replay stoppage.
//!
//! The producer-side counterpart lives in `reth-optimism-payload-builder::metrics`.

use crate::types::PostExecReplayMismatchKind;
use metrics::Counter;
use reth_metrics::Metrics;

/// Follower-/replay-side SDM metrics.
///
/// Construct via [`Default::default`] at the point of use.
#[derive(Metrics, Clone)]
#[metrics(scope = "optimism_sdm.replay")]
pub struct SDMReplayMetrics {
    /// Total blocks replayed.
    pub blocks_total: Counter,
    /// Blocks for which replay produced at least one mismatch.
    pub blocks_with_mismatch_total: Counter,
    /// Payload carried two entries targeting the same tx index.
    pub mismatch_duplicate_payload_index: Counter,
    /// Payload entry targeted a tx index outside the block.
    pub mismatch_payload_index_out_of_range: Counter,
    /// Payload entry targeted a deposit tx.
    pub mismatch_payload_targets_deposit: Counter,
    /// Payload entry targeted the synthetic 0x7D tx itself.
    pub mismatch_payload_targets_post_exec: Counter,
    /// Payload refund disagreed with the replay refund for a tx.
    pub mismatch_payload_refund: Counter,
    /// Receipt-level `opGasRefund` disagreed with the replay refund for a tx.
    pub mismatch_receipt_refund: Counter,
    /// Payload refund exceeded the tx's raw gas used.
    pub mismatch_payload_refund_exceeds_raw_gas: Counter,
    /// Replay was invoked in an unsupported mode.
    pub mismatch_unsupported_mode: Counter,
}

impl SDMReplayMetrics {
    /// Record the outcome of a single [`replay_block`](crate::replay_block) call by category.
    ///
    /// The per-category counters add up to a rate higher than `blocks_with_mismatch_total`
    /// when a single block surfaces multiple mismatches — that's intentional, since an
    /// operator wants both "which block stopped the pipeline" and "which validation rule
    /// fired" signals.
    #[inline]
    pub fn record_block(&self, mismatches: &[PostExecReplayMismatchKind]) {
        self.blocks_total.increment(1);
        if mismatches.is_empty() {
            return;
        }
        self.blocks_with_mismatch_total.increment(1);
        for kind in mismatches {
            self.counter_for(kind).increment(1);
        }
    }

    const fn counter_for(&self, kind: &PostExecReplayMismatchKind) -> &Counter {
        match kind {
            PostExecReplayMismatchKind::DuplicatePayloadIndex => {
                &self.mismatch_duplicate_payload_index
            }
            PostExecReplayMismatchKind::PayloadIndexOutOfRange => {
                &self.mismatch_payload_index_out_of_range
            }
            PostExecReplayMismatchKind::PayloadTargetsDeposit => {
                &self.mismatch_payload_targets_deposit
            }
            PostExecReplayMismatchKind::PayloadTargetsPostExec => {
                &self.mismatch_payload_targets_post_exec
            }
            PostExecReplayMismatchKind::PayloadRefundMismatch => &self.mismatch_payload_refund,
            PostExecReplayMismatchKind::ReceiptRefundMismatch => &self.mismatch_receipt_refund,
            PostExecReplayMismatchKind::PayloadRefundExceedsRawGas => {
                &self.mismatch_payload_refund_exceeds_raw_gas
            }
            PostExecReplayMismatchKind::UnsupportedMode => &self.mismatch_unsupported_mode,
        }
    }
}

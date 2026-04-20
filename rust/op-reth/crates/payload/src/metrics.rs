//! Observability for the SDM post-exec transaction in the OP payload builder.
//!
//! Recorded once per SDM-enabled block the sequencer produces:
//! - a counter split on whether a synthetic 0x7D tx ended up in the block,
//! - a histogram of the total refund gas carried by the 0x7D payload,
//! - a histogram of how many refund entries the payload carried.
//!
//! The follower-side counterpart lives in `reth-optimism-post-exec-replay::metrics`
//! (not linked here because that crate is not a direct dependency).

use metrics::{Counter, Histogram};
use reth_metrics::Metrics;

/// Producer-side SDM payload-builder metrics.
///
/// Construct via [`Default::default`] at the point of use — the handles are cheap to mint and
/// backed by the global `metrics` registry, so per-block instantiation is fine.
#[derive(Metrics, Clone)]
#[metrics(scope = "optimism_sdm.payload_builder")]
pub struct SDMPayloadMetrics {
    /// Blocks the sequencer produced with a synthetic post-exec (0x7D) tx included.
    pub blocks_with_post_exec_tx: Counter,
    /// Blocks the sequencer produced with SDM enabled but no refunds to settle, so no
    /// synthetic tx was included.
    pub blocks_without_post_exec_tx: Counter,
    /// Total refund gas carried by the 0x7D payload, per block. Only recorded when a
    /// synthetic tx was included.
    pub block_refund_gas: Histogram,
    /// Number of refund entries in the 0x7D payload, per block. Only recorded when a
    /// synthetic tx was included.
    pub block_refund_entry_count: Histogram,
}

impl SDMPayloadMetrics {
    /// Record the outcome of a single SDM-enabled block build.
    ///
    /// `included` reflects whether a synthetic 0x7D tx was actually appended to the block;
    /// `refund_gas_total` and `entry_count` are the aggregate values from the refund entries
    /// (ignored when `!included`).
    #[inline]
    pub fn record(&self, included: bool, refund_gas_total: u64, entry_count: usize) {
        if included {
            self.blocks_with_post_exec_tx.increment(1);
            self.block_refund_gas.record(refund_gas_total as f64);
            self.block_refund_entry_count.record(entry_count as f64);
        } else {
            self.blocks_without_post_exec_tx.increment(1);
        }
    }
}

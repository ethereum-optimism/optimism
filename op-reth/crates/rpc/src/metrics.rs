//! RPC metrics unique for OP-stack.

use core::time::Duration;
use metrics::Histogram;
use reth_metrics::Metrics;

/// Optimism sequencer metrics
#[derive(Metrics, Clone)]
#[metrics(scope = "optimism_rpc.sequencer")]
pub struct SequencerMetrics {
    /// How long it takes to forward a transaction to the sequencer
    pub(crate) sequencer_forward_latency: Histogram,
}

impl SequencerMetrics {
    /// Records the duration it took to forward a transaction
    #[inline]
    pub fn record_forward_latency(&self, duration: Duration) {
        self.sequencer_forward_latency.record(duration.as_secs_f64());
    }
}

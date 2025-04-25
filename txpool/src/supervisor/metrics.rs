//! Optimism supervisor and sequencer metrics

use reth_metrics::{metrics::Histogram, Metrics};
use std::time::Duration;

/// Optimism supervisor metrics
#[derive(Metrics, Clone)]
#[metrics(scope = "optimism_transaction_pool.supervisor")]
pub struct SupervisorMetrics {
    /// How long it takes to query the supervisor in the Optimism transaction pool
    pub(crate) supervisor_query_latency: Histogram,
}

impl SupervisorMetrics {
    /// Records the duration of supervisor queries
    #[inline]
    pub fn record_supervisor_query(&self, duration: Duration) {
        self.supervisor_query_latency.record(duration.as_secs_f64());
    }
}

/// Optimism sequencer metrics
#[derive(Metrics, Clone)]
#[metrics(scope = "optimism_transaction_pool.sequencer")]
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

//! RPC metrics unique for OP-stack.

use core::time::Duration;
use metrics::{Counter, Histogram};
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

/// Optimism ETH API extension metrics
#[derive(Metrics, Clone)]
#[metrics(scope = "optimism_rpc.eth_api_ext")]
pub struct EthApiExtMetrics {
    /// How long it takes to handle a `eth_getProof` request successfully
    pub(crate) get_proof_latency: Histogram,

    /// Total number of `eth_getProof` requests
    pub(crate) get_proof_requests: Counter,

    /// Total number of successful `eth_getProof` responses
    pub(crate) get_proof_successful_responses: Counter,

    /// Total number of failures handling `eth_getProof` requests
    pub(crate) get_proof_failures: Counter,
}

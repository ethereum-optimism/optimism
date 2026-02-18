//! RPC metrics unique for OP-stack.

use alloy_primitives::map::HashMap;
use core::time::Duration;
use metrics::{Counter, Histogram};
use reth_metrics::Metrics;
use std::time::Instant;
use strum::{EnumCount, EnumIter, IntoEnumIterator};

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

/// Types of debug apis
#[derive(Debug, Clone, Copy, Eq, PartialEq, Hash, EnumCount, EnumIter)]
pub enum DebugApis {
    /// `DebugExecutePayload` Api
    DebugExecutePayload,
    /// `DebugExecutionWitness` Api
    DebugExecutionWitness,
}

impl DebugApis {
    /// Returns the operation as a string for metrics labels.
    pub const fn as_str(&self) -> &'static str {
        match self {
            Self::DebugExecutePayload => "debug_execute_payload",
            Self::DebugExecutionWitness => "debug_execution_witness",
        }
    }
}

/// Metrics for Debug API extension calls.
#[derive(Debug)]
pub struct DebugApiExtMetrics {
    /// Per-api metrics handles
    apis: HashMap<DebugApis, DebugApiExtRpcMetrics>,
}

impl DebugApiExtMetrics {
    /// Initializes new `DebugApiExtMetrics`
    pub fn new() -> Self {
        let mut apis = HashMap::default();
        for api in DebugApis::iter() {
            apis.insert(api, DebugApiExtRpcMetrics::new_with_labels(&[("api", api.as_str())]));
        }
        Self { apis }
    }

    /// Record a Debug API call async (tracks latency, requests, success, failures).
    pub async fn record_operation_async<F, T, E>(&self, api: DebugApis, f: F) -> Result<T, E>
    where
        F: Future<Output = Result<T, E>>,
    {
        if let Some(metrics) = self.apis.get(&api) {
            metrics.record_async(f).await
        } else {
            f.await
        }
    }
}

impl Default for DebugApiExtMetrics {
    fn default() -> Self {
        Self::new()
    }
}

/// Optimism Debug API extension metrics
#[derive(Metrics, Clone)]
#[metrics(scope = "optimism_rpc.debug_api_ext")]
pub struct DebugApiExtRpcMetrics {
    /// End-to-end time to handle this API call
    pub(crate) latency: Histogram,

    /// Total number of requests for this API
    pub(crate) requests: Counter,

    /// Total number of successful responses for this API
    pub(crate) successful_responses: Counter,

    /// Total number of failures for this API
    pub(crate) failures: Counter,
}

impl DebugApiExtRpcMetrics {
    /// Record rpc api call async.
    async fn record_async<T, E, F>(&self, f: F) -> Result<T, E>
    where
        F: Future<Output = Result<T, E>>,
    {
        let start = Instant::now();
        let result = f.await;

        self.latency.record(start.elapsed().as_secs_f64());
        self.requests.increment(1);

        if result.is_ok() {
            self.successful_responses.increment(1);
        } else {
            self.failures.increment(1);
        }

        result
    }
}

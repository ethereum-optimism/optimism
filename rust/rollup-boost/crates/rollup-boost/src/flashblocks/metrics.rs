use metrics::{Counter, Gauge, Histogram};
use metrics_derive::Metrics;

#[derive(Metrics, Clone)]
#[metrics(scope = "flashblocks.ws_inbound")]
pub struct FlashblocksWsInboundMetrics {
    /// Total number of WebSocket reconnection attempts
    #[metric(describe = "Total number of WebSocket reconnection attempts")]
    pub reconnect_attempts: Counter,

    /// Current WebSocket connection status (1 = connected, 0 = disconnected)
    #[metric(describe = "Current WebSocket connection status")]
    pub connection_status: Gauge,

    #[metric(describe = "Number of flashblock messages received from builder")]
    pub messages_received: Counter,
}

#[derive(Metrics, Clone)]
#[metrics(scope = "flashblocks.service")]
pub struct FlashblocksServiceMetrics {
    #[metric(describe = "Number of errors when extending payload")]
    pub extend_payload_errors: Counter,

    #[metric(describe = "Number of times the current payload ID has been set")]
    pub current_payload_id_mismatch: Counter,

    #[metric(describe = "Number of messages processed by the service")]
    pub messages_processed: Counter,

    #[metric(describe = "Total number of used flashblocks")]
    pub flashblocks_gauge: Gauge,

    #[metric(describe = "Total number of used flashblocks")]
    pub flashblocks_counter: Counter,

    #[metric(describe = "Reduction in flashblocks issued.")]
    pub flashblocks_missing_histogram: Histogram,

    #[metric(describe = "Reduction in flashblocks issued.")]
    pub flashblocks_missing_gauge: Gauge,

    #[metric(describe = "Reduction in flashblocks issued.")]
    pub flashblocks_missing_counter: Counter,
}

impl FlashblocksServiceMetrics {
    pub fn record_flashblocks(&self, flashblocks_count: u64, max_flashblocks: u64) {
        let reduced_flashblocks = max_flashblocks.saturating_sub(flashblocks_count);
        self.flashblocks_gauge.set(flashblocks_count as f64);
        self.flashblocks_counter.increment(flashblocks_count);
        self.flashblocks_missing_histogram
            .record(reduced_flashblocks as f64);
        self.flashblocks_missing_gauge
            .set(reduced_flashblocks as f64);
        self.flashblocks_missing_counter
            .increment(reduced_flashblocks);
    }
}

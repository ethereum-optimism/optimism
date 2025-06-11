use metrics::{Counter, Gauge};
use metrics_derive::Metrics;

#[derive(Metrics)]
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
}

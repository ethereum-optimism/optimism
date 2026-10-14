//! Metrics for the kona-node's data sources.

/// Identifiers of the metrics the data sources record.
#[derive(Debug, Clone)]
pub struct Metrics;

impl Metrics {
    /// Identifier for the histogram that tracks the duration of requests to the L2 execution
    /// layer, labeled by JSON-RPC method.
    pub const ENGINE_METHOD_REQUEST_DURATION: &str = "kona_node_engine_method_request_duration";

    /// Initializes the metrics of the data sources.
    pub fn init() {
        Self::describe();
    }

    /// Describes the metrics of the data sources.
    pub fn describe() {
        metrics::describe_histogram!(
            Self::ENGINE_METHOD_REQUEST_DURATION,
            metrics::Unit::Seconds,
            "Duration of requests to the L2 execution layer, by JSON-RPC method"
        );
    }
}

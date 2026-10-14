//! Metrics for the kona-node's data sources.

/// Identifiers of the metrics the data sources record.
#[derive(Debug, Clone)]
pub struct Metrics;

impl Metrics {
    /// Counter of the requests a JSON-RPC client sent, by client and method.
    pub const RPC_CLIENT_REQUESTS: &str = "kona_node_rpc_client_requests_total";

    /// Histogram of how long a JSON-RPC client's requests took, by client and method, including
    /// requests that failed.
    pub const RPC_CLIENT_REQUEST_DURATION: &str = "kona_node_rpc_client_request_duration_seconds";

    /// Counter of the responses a JSON-RPC client's requests ended with, by client, method and
    /// error.
    pub const RPC_CLIENT_RESPONSES: &str = "kona_node_rpc_client_responses_total";

    /// Counter of the bytes of params a JSON-RPC client sent, by client and method.
    pub const RPC_CLIENT_PARAMS_SIZE: &str = "kona_node_rpc_client_params_size_total";

    /// Counter of the bytes of results a JSON-RPC client received, by client and method.
    pub const RPC_CLIENT_RESULTS_SIZE: &str = "kona_node_rpc_client_results_size_total";

    /// Initializes the metrics of the data sources.
    pub fn init() {
        Self::describe();
    }

    /// Describes the metrics of the data sources.
    pub fn describe() {
        metrics::describe_counter!(
            Self::RPC_CLIENT_REQUESTS,
            metrics::Unit::Count,
            "Requests sent by JSON-RPC clients"
        );
        metrics::describe_histogram!(
            Self::RPC_CLIENT_REQUEST_DURATION,
            metrics::Unit::Seconds,
            "Duration of requests sent by JSON-RPC clients, including failed requests"
        );
        metrics::describe_counter!(
            Self::RPC_CLIENT_RESPONSES,
            metrics::Unit::Count,
            "Responses to requests sent by JSON-RPC clients, by error"
        );
        metrics::describe_counter!(
            Self::RPC_CLIENT_PARAMS_SIZE,
            metrics::Unit::Bytes,
            "Bytes of params sent by JSON-RPC clients"
        );
        metrics::describe_counter!(
            Self::RPC_CLIENT_RESULTS_SIZE,
            metrics::Unit::Bytes,
            "Bytes of results received by JSON-RPC clients"
        );
    }
}

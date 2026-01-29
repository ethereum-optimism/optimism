//! Metrics for the Managed Mode RPC client.

/// Container for metrics.
#[derive(Debug, Clone)]
pub(super) struct Metrics;

impl Metrics {
    // --- Metric Names ---
    /// Identifier for the counter of successful RPC requests. Labels: `method`.
    pub(crate) const MANAGED_NODE_RPC_REQUESTS_SUCCESS_TOTAL: &'static str =
        "managed_node_rpc_requests_success_total";
    /// Identifier for the counter of failed RPC requests. Labels: `method`.
    pub(crate) const MANAGED_NODE_RPC_REQUESTS_ERROR_TOTAL: &'static str =
        "managed_node_rpc_requests_error_total";
    /// Identifier for the histogram of RPC request durations. Labels: `method`.
    pub(crate) const MANAGED_NODE_RPC_REQUEST_DURATION_SECONDS: &'static str =
        "managed_node_rpc_request_duration_seconds";

    pub(crate) const RPC_METHOD_CHAIN_ID: &'static str = "chain_id";
    pub(crate) const RPC_METHOD_SUBSCRIBE_EVENTS: &'static str = "subscribe_events";
    pub(crate) const RPC_METHOD_FETCH_RECEIPTS: &'static str = "fetch_receipts";
    pub(crate) const RPC_METHOD_OUTPUT_V0_AT_TIMESTAMP: &'static str = "output_v0_at_timestamp";
    pub(crate) const RPC_METHOD_PENDING_OUTPUT_V0_AT_TIMESTAMP: &'static str =
        "pending_output_v0_at_timestamp";
    pub(crate) const RPC_METHOD_L2_BLOCK_REF_BY_TIMESTAMP: &'static str =
        "l2_block_ref_by_timestamp";
    pub(crate) const RPC_METHOD_BLOCK_REF_BY_NUMBER: &'static str = "block_ref_by_number";
    pub(crate) const RPC_METHOD_RESET_PRE_INTEROP: &'static str = "reset_pre_interop";
    pub(crate) const RPC_METHOD_RESET: &'static str = "reset";
    pub(crate) const RPC_METHOD_INVALIDATE_BLOCK: &'static str = "invalidate_block";
    pub(crate) const RPC_METHOD_PROVIDE_L1: &'static str = "provide_l1";
    pub(crate) const RPC_METHOD_UPDATE_FINALIZED: &'static str = "update_finalized";
    pub(crate) const RPC_METHOD_UPDATE_CROSS_UNSAFE: &'static str = "update_cross_unsafe";
    pub(crate) const RPC_METHOD_UPDATE_CROSS_SAFE: &'static str = "update_cross_safe";

    /// Initializes metrics for the Supervisor RPC service.
    ///
    /// This does two things:
    /// * Describes various metrics.
    /// * Initializes metrics with their labels to 0 so they can be queried immediately.
    pub(crate) fn init(node: &str) {
        Self::describe();
        Self::zero(node);
    }

    /// Describes metrics used in the Supervisor RPC service.
    fn describe() {
        metrics::describe_counter!(
            Self::MANAGED_NODE_RPC_REQUESTS_SUCCESS_TOTAL,
            metrics::Unit::Count,
            "Total number of successful RPC requests processed by the managed mode client"
        );
        metrics::describe_counter!(
            Self::MANAGED_NODE_RPC_REQUESTS_ERROR_TOTAL,
            metrics::Unit::Count,
            "Total number of failed RPC requests processed by the managed mode client"
        );
        metrics::describe_histogram!(
            Self::MANAGED_NODE_RPC_REQUEST_DURATION_SECONDS,
            metrics::Unit::Seconds,
            "Duration of RPC requests processed by the managed mode client"
        );
    }

    fn zero_rpc_method(method: &str, node: &str) {
        metrics::counter!(
            Self::MANAGED_NODE_RPC_REQUESTS_SUCCESS_TOTAL,
            "method" => method.to_string(),
            "node" => node.to_string()
        )
        .increment(0);
        metrics::counter!(
            Self::MANAGED_NODE_RPC_REQUESTS_ERROR_TOTAL,
            "method" => method.to_string(),
            "node" => node.to_string()
        )
        .increment(0);
        metrics::histogram!(
            Self::MANAGED_NODE_RPC_REQUEST_DURATION_SECONDS,
            "method" => method.to_string(),
            "node" => node.to_string()
        )
        .record(0.0);
    }

    /// Initializes metrics with their labels to `0` so they appear in Prometheus from the start.
    fn zero(node: &str) {
        Self::zero_rpc_method(Self::RPC_METHOD_CHAIN_ID, node);
        Self::zero_rpc_method(Self::RPC_METHOD_SUBSCRIBE_EVENTS, node);
        Self::zero_rpc_method(Self::RPC_METHOD_FETCH_RECEIPTS, node);
        Self::zero_rpc_method(Self::RPC_METHOD_OUTPUT_V0_AT_TIMESTAMP, node);
        Self::zero_rpc_method(Self::RPC_METHOD_PENDING_OUTPUT_V0_AT_TIMESTAMP, node);
        Self::zero_rpc_method(Self::RPC_METHOD_L2_BLOCK_REF_BY_TIMESTAMP, node);
        Self::zero_rpc_method(Self::RPC_METHOD_BLOCK_REF_BY_NUMBER, node);
        Self::zero_rpc_method(Self::RPC_METHOD_RESET_PRE_INTEROP, node);
        Self::zero_rpc_method(Self::RPC_METHOD_RESET, node);
        Self::zero_rpc_method(Self::RPC_METHOD_INVALIDATE_BLOCK, node);
        Self::zero_rpc_method(Self::RPC_METHOD_PROVIDE_L1, node);
        Self::zero_rpc_method(Self::RPC_METHOD_UPDATE_FINALIZED, node);
        Self::zero_rpc_method(Self::RPC_METHOD_UPDATE_CROSS_UNSAFE, node);
        Self::zero_rpc_method(Self::RPC_METHOD_UPDATE_CROSS_SAFE, node);
    }
}

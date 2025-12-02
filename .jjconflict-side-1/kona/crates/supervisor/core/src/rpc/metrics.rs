//! Metrics for the Supervisor RPC service.

/// Container for metrics.
#[derive(Debug, Clone)]
pub(crate) struct Metrics;

impl Metrics {
    // --- Metric Names ---
    /// Identifier for the counter of successful RPC requests. Labels: `method`.
    pub(crate) const SUPERVISOR_RPC_REQUESTS_SUCCESS_TOTAL: &'static str =
        "supervisor_rpc_requests_success_total";
    /// Identifier for the counter of failed RPC requests. Labels: `method`.
    pub(crate) const SUPERVISOR_RPC_REQUESTS_ERROR_TOTAL: &'static str =
        "supervisor_rpc_requests_error_total";
    /// Identifier for the histogram of RPC request durations. Labels: `method`.
    pub(crate) const SUPERVISOR_RPC_REQUEST_DURATION_SECONDS: &'static str =
        "supervisor_rpc_request_duration_seconds";

    pub(crate) const SUPERVISOR_RPC_METHOD_CROSS_DERIVED_TO_SOURCE: &'static str =
        "cross_derived_to_source";
    pub(crate) const SUPERVISOR_RPC_METHOD_DEPENDENCY_SET: &'static str = "dependency_set";
    pub(crate) const SUPERVISOR_RPC_METHOD_LOCAL_UNSAFE: &'static str = "local_unsafe";
    pub(crate) const SUPERVISOR_RPC_METHOD_LOCAL_SAFE: &'static str = "local_safe";
    pub(crate) const SUPERVISOR_RPC_METHOD_CROSS_SAFE: &'static str = "cross_safe";
    pub(crate) const SUPERVISOR_RPC_METHOD_FINALIZED: &'static str = "finalized";
    pub(crate) const SUPERVISOR_RPC_METHOD_FINALIZED_L1: &'static str = "finalized_l1";
    pub(crate) const SUPERVISOR_RPC_METHOD_SUPER_ROOT_AT_TIMESTAMP: &'static str =
        "super_root_at_timestamp";
    pub(crate) const SUPERVISOR_RPC_METHOD_SYNC_STATUS: &'static str = "sync_status";
    pub(crate) const SUPERVISOR_RPC_METHOD_ALL_SAFE_DERIVED_AT: &'static str =
        "all_safe_derived_at";
    pub(crate) const SUPERVISOR_RPC_METHOD_CHECK_ACCESS_LIST: &'static str = "check_access_list";

    /// Initializes metrics for the Supervisor RPC service.
    ///
    /// This does two things:
    /// * Describes various metrics.
    /// * Initializes metrics with their labels to 0 so they can be queried immediately.
    pub(crate) fn init() {
        Self::describe();
        Self::zero();
    }

    /// Describes metrics used in the Supervisor RPC service.
    fn describe() {
        metrics::describe_counter!(
            Self::SUPERVISOR_RPC_REQUESTS_SUCCESS_TOTAL,
            metrics::Unit::Count,
            "Total number of successful RPC requests processed by the supervisor"
        );
        metrics::describe_counter!(
            Self::SUPERVISOR_RPC_REQUESTS_ERROR_TOTAL,
            metrics::Unit::Count,
            "Total number of failed RPC requests processed by the supervisor"
        );
        metrics::describe_histogram!(
            Self::SUPERVISOR_RPC_REQUEST_DURATION_SECONDS,
            metrics::Unit::Seconds,
            "Duration of RPC requests processed by the supervisor"
        );
    }

    fn zero_rpc_method(method: &str) {
        metrics::counter!(
            Self::SUPERVISOR_RPC_REQUESTS_SUCCESS_TOTAL,
            "method" => method.to_string()
        )
        .increment(0);

        metrics::counter!(
            Self::SUPERVISOR_RPC_REQUESTS_ERROR_TOTAL,
            "method" => method.to_string()
        )
        .increment(0);

        metrics::histogram!(
            Self::SUPERVISOR_RPC_REQUEST_DURATION_SECONDS,
            "method" => method.to_string()
        )
        .record(0.0); // Record a zero value to ensure the label combination is present
    }

    /// Initializes metrics with their labels to `0` so they appear in Prometheus from the start.
    fn zero() {
        Self::zero_rpc_method(Self::SUPERVISOR_RPC_METHOD_CROSS_DERIVED_TO_SOURCE);
        Self::zero_rpc_method(Self::SUPERVISOR_RPC_METHOD_LOCAL_UNSAFE);
        Self::zero_rpc_method(Self::SUPERVISOR_RPC_METHOD_LOCAL_SAFE);
        Self::zero_rpc_method(Self::SUPERVISOR_RPC_METHOD_CROSS_SAFE);
        Self::zero_rpc_method(Self::SUPERVISOR_RPC_METHOD_FINALIZED);
        Self::zero_rpc_method(Self::SUPERVISOR_RPC_METHOD_FINALIZED_L1);
        Self::zero_rpc_method(Self::SUPERVISOR_RPC_METHOD_SUPER_ROOT_AT_TIMESTAMP);
        Self::zero_rpc_method(Self::SUPERVISOR_RPC_METHOD_SYNC_STATUS);
        Self::zero_rpc_method(Self::SUPERVISOR_RPC_METHOD_ALL_SAFE_DERIVED_AT);
        Self::zero_rpc_method(Self::SUPERVISOR_RPC_METHOD_CHECK_ACCESS_LIST);
    }
}

/// Observes an RPC call, recording its duration and outcome.
///
/// # Usage
/// ```ignore
/// async fn my_rpc_method(&self, arg: u32) -> RpcResult<String> {
///     observe_rpc_call!("my_rpc_method_name", {
///         // todo: add actual RPC logic
///         if arg == 0 { Ok("success".to_string()) } else { Err(ErrorObject::owned(1, "failure", None::<()>)) }
///     })
/// }
/// ```
#[macro_export]
macro_rules! observe_rpc_call {
    ($method_name:expr, $block:expr) => {{
        let start_time = std::time::Instant::now();
        let result = $block; // Execute the provided code block
        let duration = start_time.elapsed().as_secs_f64();

        if result.is_ok() {
            metrics::counter!($crate::rpc::metrics::Metrics::SUPERVISOR_RPC_REQUESTS_SUCCESS_TOTAL, "method" => $method_name).increment(1);
        } else {
            metrics::counter!($crate::rpc::metrics::Metrics::SUPERVISOR_RPC_REQUESTS_ERROR_TOTAL, "method" => $method_name).increment(1);
        }

        metrics::histogram!($crate::rpc::metrics::Metrics::SUPERVISOR_RPC_REQUEST_DURATION_SECONDS, "method" => $method_name).record(duration);
        result
    }};
}

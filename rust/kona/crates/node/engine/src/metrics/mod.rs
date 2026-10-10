//! Prometheus metrics collection for engine operations.
//!
//! Provides metric identifiers and labels for monitoring engine performance,
//! task execution, and block progression through safety levels.

/// Metrics container with constants for Prometheus metric collection.
///
/// Contains identifiers for gauges, counters, and histograms used to monitor
/// engine operations. Metrics instrumentation is always enabled and tracks:
///
/// - Block progression through safety levels (unsafe → finalized)
/// - Task execution success/failure rates by type
/// - Engine API method call latencies
///
/// # Usage
///
/// ```rust,ignore
/// use metrics::{counter, gauge, histogram};
/// use kona_engine::Metrics;
///
/// // Track successful task execution
/// counter!(Metrics::ENGINE_TASK_SUCCESS, "task" => Metrics::INSERT_TASK_LABEL);
///
/// // Record block height at safety level
/// gauge!(Metrics::BLOCK_LABELS, block_num as f64, "level" => Metrics::SAFE_BLOCK_LABEL);
///
/// // Time Engine API calls
/// histogram!(Metrics::ENGINE_METHOD_REQUEST_DURATION, duration.as_secs_f64());
/// ```
#[derive(Debug, Clone)]
pub struct Metrics;

impl Metrics {
    /// Identifier for the gauge that tracks block labels.
    pub const BLOCK_LABELS: &str = "kona_node_block_labels";
    /// Unsafe block label.
    pub const UNSAFE_BLOCK_LABEL: &str = "unsafe";
    /// Local-safe block label.
    pub const LOCAL_SAFE_BLOCK_LABEL: &str = "local-safe";
    /// Safe block label.
    pub const SAFE_BLOCK_LABEL: &str = "safe";
    /// Finalized block label.
    pub const FINALIZED_BLOCK_LABEL: &str = "finalized";

    /// Identifier for the counter that records engine task counts.
    pub const ENGINE_TASK_SUCCESS: &str = "kona_node_engine_task_count";
    /// Identifier for the counter that records engine task counts.
    pub const ENGINE_TASK_FAILURE: &str = "kona_node_engine_task_failure";

    /// Insert task label.
    pub const INSERT_TASK_LABEL: &str = "insert";
    /// Consolidate task label.
    pub const CONSOLIDATE_TASK_LABEL: &str = "consolidate";
    /// Forkchoice task label.
    pub const FORKCHOICE_TASK_LABEL: &str = "forkchoice-update";
    /// Build task label.
    pub const BUILD_TASK_LABEL: &str = "build";
    /// Seal task label.
    pub const SEAL_TASK_LABEL: &str = "seal";
    /// Finalize task label.
    pub const FINALIZE_TASK_LABEL: &str = "finalize";

    /// Identifier for the histogram that tracks the duration of requests to the L2 execution
    /// layer, labeled by JSON-RPC method.
    pub const ENGINE_METHOD_REQUEST_DURATION: &str = "kona_node_engine_method_request_duration";

    /// Identifier for the counter that tracks the number of times the engine has been reset.
    pub const ENGINE_RESET_COUNT: &str = "kona_node_engine_reset_count";

    /// Initializes metrics for the engine.
    ///
    /// This does two things:
    /// * Describes various metrics.
    /// * Initializes metrics to 0 so they can be queried immediately.
    pub fn init() {
        Self::describe();
        Self::zero();
    }

    /// Describes metrics used in [`kona_engine`][crate].
    pub fn describe() {
        // Block labels
        metrics::describe_gauge!(Self::BLOCK_LABELS, "Blockchain head labels");

        // Engine task counts
        metrics::describe_counter!(Self::ENGINE_TASK_SUCCESS, "Engine tasks successfully executed");
        metrics::describe_counter!(Self::ENGINE_TASK_FAILURE, "Engine tasks failed");

        // L2 execution layer request duration histogram
        metrics::describe_histogram!(
            Self::ENGINE_METHOD_REQUEST_DURATION,
            metrics::Unit::Seconds,
            "Duration of requests to the L2 execution layer, by JSON-RPC method"
        );

        // Engine reset counter
        metrics::describe_counter!(
            Self::ENGINE_RESET_COUNT,
            metrics::Unit::Count,
            "Engine reset count"
        );
    }

    /// Initializes metrics to `0` so they can be queried immediately by consumers of prometheus
    /// metrics.
    pub fn zero() {
        // Engine task counts
        metrics::counter!(Self::ENGINE_TASK_SUCCESS, "type" => Self::INSERT_TASK_LABEL).absolute(0);
        metrics::counter!(Self::ENGINE_TASK_SUCCESS, "type" => Self::CONSOLIDATE_TASK_LABEL)
            .absolute(0);
        metrics::counter!(Self::ENGINE_TASK_SUCCESS, "type" => Self::BUILD_TASK_LABEL).absolute(0);
        metrics::counter!(Self::ENGINE_TASK_SUCCESS, "type" => Self::FINALIZE_TASK_LABEL)
            .absolute(0);

        metrics::counter!(Self::ENGINE_TASK_FAILURE, "type" => Self::INSERT_TASK_LABEL).absolute(0);
        metrics::counter!(Self::ENGINE_TASK_FAILURE, "type" => Self::CONSOLIDATE_TASK_LABEL)
            .absolute(0);
        metrics::counter!(Self::ENGINE_TASK_FAILURE, "type" => Self::BUILD_TASK_LABEL).absolute(0);
        metrics::counter!(Self::ENGINE_TASK_FAILURE, "type" => Self::FINALIZE_TASK_LABEL)
            .absolute(0);

        // Engine reset count
        metrics::counter!(Self::ENGINE_RESET_COUNT).absolute(0);
    }
}

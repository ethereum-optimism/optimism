//! Prometheus metrics collection for engine operations.
//!
//! Provides metric identifiers and labels for monitoring engine performance,
//! task execution, and block progression through safety levels.

#[cfg(feature = "metrics")]
use crate::EngineTaskErrorSeverity;

/// Metrics container with constants for Prometheus metric collection.
///
/// Contains identifiers for gauges, counters, and histograms used to monitor
/// engine operations when the `metrics` feature is enabled. Metrics track:
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
    /// Label key carrying the L2 chain ID, present on every metric emitted by the engine.
    pub const CHAIN_ID_LABEL: &str = kona_macros::CHAIN_ID_LABEL;

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

    /// Identifier for the histogram that tracks engine method call time.
    pub const ENGINE_METHOD_REQUEST_DURATION: &str = "kona_node_engine_method_request_duration";
    /// `engine_forkchoiceUpdatedV<N>` label
    pub const FORKCHOICE_UPDATE_METHOD: &str = "engine_forkchoiceUpdated";
    /// `engine_newPayloadV<N>` label.
    pub const NEW_PAYLOAD_METHOD: &str = "engine_newPayload";
    /// `engine_getPayloadV<N>` label.
    pub const GET_PAYLOAD_METHOD: &str = "engine_getPayload";

    /// Identifier for the counter that tracks the number of times the engine has been reset.
    pub const ENGINE_RESET_COUNT: &str = "kona_node_engine_reset_count";

    /// Initializes metrics for the engine.
    ///
    /// This does two things:
    /// * Describes various metrics.
    /// * Initializes metrics to 0 so they can be queried immediately.
    #[cfg(feature = "metrics")]
    pub fn init(chain_id: u64) {
        Self::describe();
        Self::zero(chain_id);
    }

    /// Describes metrics used in [`kona_engine`][crate].
    #[cfg(feature = "metrics")]
    pub fn describe() {
        // Block labels
        metrics::describe_gauge!(Self::BLOCK_LABELS, "Blockchain head labels");

        // Engine task counts
        metrics::describe_counter!(Self::ENGINE_TASK_SUCCESS, "Engine tasks successfully executed");
        metrics::describe_counter!(Self::ENGINE_TASK_FAILURE, "Engine tasks failed");

        // Engine method request duration histogram
        metrics::describe_histogram!(
            Self::ENGINE_METHOD_REQUEST_DURATION,
            metrics::Unit::Seconds,
            "Engine method request duration"
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
    #[cfg(feature = "metrics")]
    pub fn zero(chain_id: u64) {
        let chain_id = kona_macros::chain_id_label(chain_id);

        // Engine task counts
        // Every arm of `EngineTask::task_metrics_label`, so the pre-created set matches the
        // emit exactly.
        for task in [
            Self::INSERT_TASK_LABEL,
            Self::CONSOLIDATE_TASK_LABEL,
            Self::BUILD_TASK_LABEL,
            Self::SEAL_TASK_LABEL,
            Self::FINALIZE_TASK_LABEL,
        ] {
            kona_macros::set!(
                counter,
                Self::ENGINE_TASK_SUCCESS,
                0,
                "type" => task,
                Self::CHAIN_ID_LABEL => chain_id.clone()
            );
            // The runtime failure emit carries a `severity` dimension too, so pre-create the
            // full task x severity cross product. The severity strings come from the enum's
            // own `Display` so the two sites cannot drift apart.
            for severity in [
                EngineTaskErrorSeverity::Temporary,
                EngineTaskErrorSeverity::Critical,
                EngineTaskErrorSeverity::Reset,
                EngineTaskErrorSeverity::Flush,
            ] {
                kona_macros::set!(
                    counter,
                    Self::ENGINE_TASK_FAILURE,
                    0,
                    "type" => task,
                    "severity" => severity.to_string(),
                    Self::CHAIN_ID_LABEL => chain_id.clone()
                );
            }
        }

        // Engine reset count
        kona_macros::set!(counter, Self::ENGINE_RESET_COUNT, 0, Self::CHAIN_ID_LABEL => chain_id);
    }
}

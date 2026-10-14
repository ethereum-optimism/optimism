//! Metrics for the node

/// Container for metrics.
#[derive(Debug, Clone)]
pub struct Metrics;

impl Metrics {
    /// Parsed server RPC calls and notifications, by registered method name and request kind.
    pub const RPC_REQUESTS: &str = "kona_node_rpc_requests_total";

    /// Failed server RPC calls, by registered method name and JSON-RPC error code.
    pub const RPC_ERRORS: &str = "kona_node_rpc_errors_total";

    /// Server RPC call duration in seconds, including failed calls.
    pub const RPC_REQUEST_DURATION: &str = "kona_node_rpc_request_duration_seconds";

    /// Identifier for the counter that tracks the number of times the L1 has reorganized.
    pub const L1_REORG_COUNT: &str = "kona_node_l1_reorg_count";

    /// Identifier for the counter that tracks the L1 origin of the derivation pipeline.
    pub const DERIVATION_L1_ORIGIN: &str = "kona_node_derivation_l1_origin";

    /// Identifier for the counter of critical derivation errors (strictly for alerting.)
    pub const DERIVATION_CRITICAL_ERROR: &str = "kona_node_derivation_critical_errors";

    /// Identifier for the counter that tracks sequencer state flags.
    pub const SEQUENCER_STATE: &str = "kona_node_sequencer_state";

    /// Gauge for the sequencer's attributes builder duration.
    pub const SEQUENCER_ATTRIBUTES_BUILDER_DURATION: &str =
        "kona_node_sequencer_attributes_build_duration";

    /// Gauge for the sequencer's block building start task duration.
    pub const SEQUENCER_BLOCK_BUILDING_START_TASK_DURATION: &str =
        "kona_node_sequencer_block_building_start_task_duration";

    /// Gauge for the sequencer's block building seal task duration.
    pub const SEQUENCER_BLOCK_BUILDING_SEAL_TASK_DURATION: &str =
        "kona_node_sequencer_block_building_seal_task_duration";

    /// Gauge for the sequencer's conductor commitment duration.
    pub const SEQUENCER_CONDUCTOR_COMMITMENT_DURATION: &str =
        "kona_node_sequencer_conductor_commitment_duration";

    /// Total number of transactions of sequenced by sequencer.
    pub const SEQUENCER_TOTAL_TRANSACTIONS_SEQUENCED: &str =
        "kona_node_sequencer_total_transactions_sequenced";

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

    /// Identifier for the counter that tracks the number of times the engine has been reset.
    pub const ENGINE_RESET_COUNT: &str = "kona_node_engine_reset_count";

    /// Initializes metrics for the node service.
    ///
    /// This does two things:
    /// * Describes various metrics.
    /// * Initializes metrics to 0 so they can be queried immediately.
    pub fn init() {
        Self::describe();
        Self::zero();
    }

    /// Describes node metrics.
    pub fn describe() {
        metrics::describe_counter!(Self::RPC_REQUESTS, "Parsed JSON-RPC calls and notifications");
        metrics::describe_counter!(Self::RPC_ERRORS, "Failed JSON-RPC calls");
        metrics::describe_histogram!(
            Self::RPC_REQUEST_DURATION,
            metrics::Unit::Seconds,
            "JSON-RPC call duration, including failed calls"
        );

        // L1 reorg count
        metrics::describe_counter!(Self::L1_REORG_COUNT, metrics::Unit::Count, "L1 reorg count");

        // Derivation L1 origin
        metrics::describe_counter!(Self::DERIVATION_L1_ORIGIN, "Derivation pipeline L1 origin");

        // Derivation critical error
        metrics::describe_counter!(
            Self::DERIVATION_CRITICAL_ERROR,
            "Critical errors in the derivation pipeline"
        );

        // Sequencer state
        metrics::describe_counter!(Self::SEQUENCER_STATE, "Tracks sequencer state flags");

        // Sequencer attributes builder duration
        metrics::describe_gauge!(
            Self::SEQUENCER_ATTRIBUTES_BUILDER_DURATION,
            "Duration of the sequencer attributes builder"
        );

        // Sequencer block building job duration
        metrics::describe_gauge!(
            Self::SEQUENCER_BLOCK_BUILDING_START_TASK_DURATION,
            "Duration of the sequencer block building start task"
        );

        // Sequencer block building job duration
        metrics::describe_gauge!(
            Self::SEQUENCER_BLOCK_BUILDING_SEAL_TASK_DURATION,
            "Duration of the sequencer block building seal task"
        );

        // Sequencer conductor commitment duration
        metrics::describe_gauge!(
            Self::SEQUENCER_CONDUCTOR_COMMITMENT_DURATION,
            "Duration of the sequencer conductor commitment"
        );

        // Sequencer total transactions sequenced
        metrics::describe_counter!(
            Self::SEQUENCER_TOTAL_TRANSACTIONS_SEQUENCED,
            metrics::Unit::Count,
            "Total count of sequenced transactions"
        );

        // Block labels
        metrics::describe_gauge!(Self::BLOCK_LABELS, "Blockchain head labels");

        // Engine task counts
        metrics::describe_counter!(Self::ENGINE_TASK_SUCCESS, "Engine tasks successfully executed");
        metrics::describe_counter!(Self::ENGINE_TASK_FAILURE, "Engine tasks failed");

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
        // L1 reorg reset count
        metrics::counter!(Self::L1_REORG_COUNT).absolute(0);

        // Derivation critical error
        metrics::counter!(Self::DERIVATION_CRITICAL_ERROR).absolute(0);

        // Sequencer: reset total transactions sequenced
        metrics::counter!(Self::SEQUENCER_TOTAL_TRANSACTIONS_SEQUENCED).absolute(0);

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

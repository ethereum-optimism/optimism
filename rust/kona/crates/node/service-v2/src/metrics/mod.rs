//! Metrics for the node service

/// Container for metrics.
#[derive(Debug, Clone)]
pub struct Metrics;

impl Metrics {
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

    /// Initializes metrics for the node service.
    ///
    /// This does two things:
    /// * Describes various metrics.
    /// * Initializes metrics to 0 so they can be queried immediately.
    #[cfg(feature = "metrics")]
    pub fn init() {
        Self::describe();
        Self::zero();
    }

    /// Describes metrics used in [`kona-node-service`][crate].
    #[cfg(feature = "metrics")]
    pub fn describe() {
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
    }

    /// Initializes metrics to `0` so they can be queried immediately by consumers of prometheus
    /// metrics.
    #[cfg(feature = "metrics")]
    pub fn zero() {
        // L1 reorg reset count
        kona_macros::set!(counter, Self::L1_REORG_COUNT, 0);

        // Derivation critical error
        kona_macros::set!(counter, Self::DERIVATION_CRITICAL_ERROR, 0);

        // Sequencer: reset total transactions sequenced
        kona_macros::set!(counter, Self::SEQUENCER_TOTAL_TRANSACTIONS_SEQUENCED, 0);
    }
}

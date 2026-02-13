//! Metrics for the derivation pipeline.

/// Container for metrics.
#[derive(Debug, Clone)]
pub struct Metrics;

impl Metrics {
    /// Identifier for the pipeline origin gauge.
    pub const PIPELINE_ORIGIN: &str = "kona_derive_pipeline_origin";

    /// Identifier for the latest l2 block the pipeline stepped on.
    pub const PIPELINE_STEP_BLOCK: &str = "kona_derive_pipeline_step_block";

    /// Identifier for if the batch reader is set.
    pub const PIPELINE_BATCH_READER_SET: &str = "kona_derive_batch_reader_set";

    /// Identifier to track the amount of time it takes to advance the pipeline origin.
    pub const PIPELINE_ORIGIN_ADVANCE: &str = "kona_derive_pipeline_origin_advance";

    /// Identifier for the histogram that tracks when the system config is updated.
    pub const SYSTEM_CONFIG_UPDATE: &str = "kona_derive_system_config_update";

    /// Identifier for the number of frames in the frame queue pipeline stage.
    pub const PIPELINE_FRAME_QUEUE_BUFFER: &str = "kona_derive_frame_queue_buffer";

    /// Identifier for the frame queue buffer memory overhead gauge.
    pub const PIPELINE_FRAME_QUEUE_MEM: &str = "kona_derive_frame_queue_mem";

    /// Identifier for the number of channels held in the pipeline.
    pub const PIPELINE_CHANNEL_BUFFER: &str = "kona_derive_channel_buffer";

    /// Identifier for the channel buffer memory overhead gauge.
    pub const PIPELINE_CHANNEL_MEM: &str = "kona_derive_channel_mem";

    /// Identifier for a gauge that tracks the number of blocks until the next channel times out.
    pub const PIPELINE_CHANNEL_TIMEOUT: &str = "kona_derive_blocks_until_channel_timeout";

    /// Identifier for the gauge that tracks the maximum rlp byte size per channel.
    pub const PIPELINE_MAX_RLP_BYTES: &str = "kona_derive_max_rlp_bytes";

    /// Identifier for the batch stream stage singular batch buffer size.
    pub const PIPELINE_BATCH_BUFFER: &str = "kona_derive_batch_buffer";

    /// Identifier for the batch stream stage batch memory overhead gauge.
    pub const PIPELINE_BATCH_MEM: &str = "kona_derive_batch_mem";

    /// Identifier for the size of batches read by the channel reader.
    pub const PIPELINE_READ_BATCHES: &str = "kona_derive_read_batches";

    /// Identifier for the gauge that tracks the number of pipeline steps.
    pub const PIPELINE_STEPS: &str = "kona_derive_pipeline_steps";

    /// Identifier for the gauge that tracks the number of prepared attributes.
    pub const PIPELINE_PREPARED_ATTRIBUTES: &str = "kona_derive_prepared_attributes";

    /// Identifier tracking the number of pipeline signals.
    pub const PIPELINE_SIGNALS: &str = "kona_derive_pipeline_signals";

    /// Identifier that tracks the batch validator l1 blocks start.
    pub const PIPELINE_L1_BLOCKS_START: &str = "kona_derive_l1_blocks_start";

    /// Identifier that tracks the batch validator l1 blocks end.
    pub const PIPELINE_L1_BLOCKS_END: &str = "kona_derive_l1_blocks_end";

    /// Identifier to track the size of the current derived span batch.
    pub const PIPELINE_DERIVED_SPAN_SIZE: &str = "kona_derive_span_size";

    /// Identifier to track the number of transactions in the latest derived payload attributes.
    pub const PIPELINE_LATEST_PAYLOAD_TX_COUNT: &str = "kona_derive_payload_tx_count";

    /// Identifier for the data availability provider data.
    pub const PIPELINE_DATA_AVAILABILITY_PROVIDER: &str = "kona_derive_dap_sources";

    /// Identifier for a gauge that tracks batch validity.
    pub const PIPELINE_BATCH_VALIDITY: &str = "kona_derive_batch_validity";

    /// Identifier for the histogram that tracks the amount of time it takes to validate a
    /// span batch.
    pub const PIPELINE_CHECK_BATCH_PREFIX: &str = "kona_derive_check_batch_prefix_duration";

    /// Identifier for the histogram that tracks the amount of time it takes to build payload
    /// attributes.
    pub const PIPELINE_ATTRIBUTES_BUILD_DURATION: &str = "kona_derive_attributes_build_duration";

    /// Identifier for the gauge that tracks the number of payload attributes buffered in the
    /// pipeline.
    pub const PIPELINE_PAYLOAD_ATTRIBUTES_BUFFER: &str = "kona_derive_payload_attributes_buffer";

    /// Identifier for a gauge that tracks the latest block number for a system config update.
    pub const PIPELINE_LATEST_SYS_CONFIG_UPDATE: &'static str =
        "kona_genesis_latest_system_config_update";

    /// Identifier for a gauge that tracks the block height at which a system config update errored.
    pub const PIPELINE_SYS_CONFIG_UPDATE_ERROR: &'static str =
        "kona_genesis_sys_config_update_error";

    /// Gauge that tracks the latest decompressed batch size.
    pub const PIPELINE_LATEST_DECOMPRESSED_BATCH_SIZE: &str =
        "kona_derive_latest_decompressed_batch_size";

    /// Gauge that tracks the latest decompressed batch type.
    pub const PIPELINE_LATEST_DECOMPRESSED_BATCH_TYPE: &str =
        "kona_derive_latest_decompressed_batch_type";
}

impl Metrics {
    /// Initializes metrics.
    ///
    /// This does two things:
    /// * Describes various metrics.
    /// * Initializes metrics to 0 so they can be queried immediately.
    #[cfg(feature = "metrics")]
    pub fn init() {
        Self::describe();
        Self::zero();
    }

    /// Describes metrics.
    #[cfg(feature = "metrics")]
    pub fn describe() {
        metrics::describe_gauge!(
            Self::PIPELINE_SYS_CONFIG_UPDATE_ERROR,
            "The block height at which a system config update errored"
        );
        metrics::describe_gauge!(
            Self::PIPELINE_LATEST_SYS_CONFIG_UPDATE,
            "The latest block number for a system config update"
        );
        metrics::describe_gauge!(
            Self::PIPELINE_LATEST_DECOMPRESSED_BATCH_SIZE,
            "The latest decompressed batch size"
        );
        metrics::describe_gauge!(
            Self::PIPELINE_LATEST_DECOMPRESSED_BATCH_TYPE,
            "The latest decompressed batch type"
        );
        metrics::describe_gauge!(
            Self::PIPELINE_ORIGIN,
            "The block height of the pipeline l1 origin"
        );
        metrics::describe_gauge!(
            Self::PIPELINE_BATCH_VALIDITY,
            "The validity of the batch being processed",
        );
        metrics::describe_gauge!(
            Self::PIPELINE_DATA_AVAILABILITY_PROVIDER,
            "The source of pipeline data"
        );
        metrics::describe_gauge!(
            Self::PIPELINE_DERIVED_SPAN_SIZE,
            "The number of payload attributes in the current span"
        );
        metrics::describe_gauge!(
            Self::PIPELINE_LATEST_PAYLOAD_TX_COUNT,
            "The number of transactions in the latest derived payload attributes"
        );
        metrics::describe_gauge!(Self::PIPELINE_READ_BATCHES, "The read batches");
        metrics::describe_gauge!(Self::PIPELINE_BATCH_READER_SET, "If the batch reader is set");
        metrics::describe_gauge!(Self::PIPELINE_L1_BLOCKS_START, "Earliest l1 blocks height");
        metrics::describe_gauge!(Self::PIPELINE_L1_BLOCKS_END, "Latest l1 blocks height");
        metrics::describe_gauge!(
            Self::PIPELINE_STEP_BLOCK,
            "The latest L2 block height that the pipeline stepped on"
        );
        metrics::describe_histogram!(
            Self::PIPELINE_CHECK_BATCH_PREFIX,
            "The time it takes to validate a span batch"
        );
        metrics::describe_histogram!(
            Self::PIPELINE_ORIGIN_ADVANCE,
            "The amount of time it takes to advance the pipeline origin"
        );
        metrics::describe_histogram!(
            Self::SYSTEM_CONFIG_UPDATE,
            "The time it takes to update the system config"
        );
        metrics::describe_gauge!(
            Self::PIPELINE_FRAME_QUEUE_BUFFER,
            "The number of frames in the frame queue"
        );
        metrics::describe_gauge!(
            Self::PIPELINE_FRAME_QUEUE_MEM,
            "The memory size of frames held in the frame queue"
        );
        metrics::describe_gauge!(
            Self::PIPELINE_CHANNEL_BUFFER,
            "The number of channels in the channel assembler stage"
        );
        metrics::describe_gauge!(
            Self::PIPELINE_CHANNEL_MEM,
            "The memory size of channels held in the channel assembler stage"
        );
        metrics::describe_gauge!(
            Self::PIPELINE_CHANNEL_TIMEOUT,
            "The number of blocks until the next channel times out"
        );
        metrics::describe_gauge!(
            Self::PIPELINE_MAX_RLP_BYTES,
            "The maximum rlp byte size of a channel"
        );
        metrics::describe_gauge!(
            Self::PIPELINE_BATCH_BUFFER,
            "The number of batches held in the batch stream stage"
        );
        metrics::describe_gauge!(
            Self::PIPELINE_BATCH_MEM,
            "The memory size of batches held in the batch stream stage"
        );
        metrics::describe_gauge!(
            Self::PIPELINE_STEPS,
            "The total number of pipeline steps on the derivation pipeline"
        );
        metrics::describe_gauge!(
            Self::PIPELINE_PREPARED_ATTRIBUTES,
            "The total number of prepared attributes generated by the derivation pipeline"
        );
        metrics::describe_gauge!(
            Self::PIPELINE_SIGNALS,
            "Number of times the pipeline has been signalled"
        );
        metrics::describe_histogram!(
            Self::PIPELINE_ATTRIBUTES_BUILD_DURATION,
            "The time it takes to build payload attributes"
        );
        metrics::describe_gauge!(
            Self::PIPELINE_PAYLOAD_ATTRIBUTES_BUFFER,
            "The number of payload attributes buffered in the pipeline"
        );
    }

    /// Initializes metrics to 0 so they can be queried immediately.
    #[cfg(feature = "metrics")]
    pub fn zero() {
        // The batch reader is by default not set.
        kona_macros::set!(gauge, Self::PIPELINE_BATCH_READER_SET, 0);

        // No source data is initially read.
        kona_macros::set!(gauge, Self::PIPELINE_DATA_AVAILABILITY_PROVIDER, "source", "blobs", 0);
        kona_macros::set!(
            gauge,
            Self::PIPELINE_DATA_AVAILABILITY_PROVIDER,
            "source",
            "calldata",
            0
        );

        // Manually translate a value of `0` for sys config update as no update yet.
        kona_macros::set!(gauge, Self::PIPELINE_LATEST_SYS_CONFIG_UPDATE, 0);
        kona_macros::set!(gauge, Self::PIPELINE_SYS_CONFIG_UPDATE_ERROR, 0);

        // Pipeline signals start at zero.
        kona_macros::set!(gauge, Self::PIPELINE_SIGNALS, "type", "reset", 0);
        kona_macros::set!(gauge, Self::PIPELINE_SIGNALS, "type", "activation", 0);
        kona_macros::set!(gauge, Self::PIPELINE_SIGNALS, "type", "flush_channel", 0);

        // No batches are initially read.
        kona_macros::set!(gauge, Self::PIPELINE_READ_BATCHES, "type", "single", 0);
        kona_macros::set!(gauge, Self::PIPELINE_READ_BATCHES, "type", "span", 0);

        // Cumulative counters start at zero.
        kona_macros::set!(gauge, Self::PIPELINE_STEPS, 0);
        kona_macros::set!(gauge, Self::PIPELINE_PREPARED_ATTRIBUTES, 0);

        // All buffers can be zeroed out since they are expected to return to zero.
        kona_macros::set!(gauge, Self::PIPELINE_BATCH_BUFFER, 0);
        kona_macros::set!(gauge, Self::PIPELINE_CHANNEL_BUFFER, 0);
        kona_macros::set!(gauge, Self::PIPELINE_FRAME_QUEUE_BUFFER, 0);
        kona_macros::set!(gauge, Self::PIPELINE_PAYLOAD_ATTRIBUTES_BUFFER, 0);
    }
}

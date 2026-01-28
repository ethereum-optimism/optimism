use crate::ChainProcessorError;
use alloy_primitives::ChainId;
use kona_protocol::BlockInfo;
use std::time::SystemTime;
use tracing::error;

#[derive(Debug)]
pub(crate) struct Metrics;

impl Metrics {
    // --- Metric Names ---
    /// Identifier for block processing success.
    /// Labels: `chain_id`, `type`
    pub(crate) const BLOCK_PROCESSING_SUCCESS_TOTAL: &'static str =
        "supervisor_block_processing_success_total";

    /// Identifier for block processing errors.
    /// Labels: `chain_id`, `type`
    pub(crate) const BLOCK_PROCESSING_ERROR_TOTAL: &'static str =
        "supervisor_block_processing_error_total";

    /// Identifier for block processing latency.
    /// Labels: `chain_id`, `type`
    pub(crate) const BLOCK_PROCESSING_LATENCY_SECONDS: &'static str =
        "supervisor_block_processing_latency_seconds";

    pub(crate) const BLOCK_TYPE_LOCAL_UNSAFE: &'static str = "local_unsafe";
    pub(crate) const BLOCK_TYPE_CROSS_UNSAFE: &'static str = "cross_unsafe";
    pub(crate) const BLOCK_TYPE_LOCAL_SAFE: &'static str = "local_safe";
    pub(crate) const BLOCK_TYPE_CROSS_SAFE: &'static str = "cross_safe";
    pub(crate) const BLOCK_TYPE_FINALIZED: &'static str = "finalized";

    // --- Block Invalidation Metric Names ---
    /// Identifier for block invalidation success.
    /// Labels: `chain_id`
    pub(crate) const BLOCK_INVALIDATION_SUCCESS_TOTAL: &'static str =
        "supervisor_block_invalidation_success_total";

    /// Identifier for block invalidation errors.
    /// Labels: `chain_id`
    pub(crate) const BLOCK_INVALIDATION_ERROR_TOTAL: &'static str =
        "supervisor_block_invalidation_error_total";

    /// Identifier for block invalidation latency.
    /// Labels: `chain_id`
    pub(crate) const BLOCK_INVALIDATION_LATENCY_SECONDS: &'static str =
        "supervisor_block_invalidation_latency_seconds";

    pub(crate) const BLOCK_INVALIDATION_METHOD_INVALIDATE_BLOCK: &'static str = "invalidate_block";

    // --- Block Replacement Metric Names ---
    /// Identifier for block replacement success.
    /// Labels: `chain_id`
    pub(crate) const BLOCK_REPLACEMENT_SUCCESS_TOTAL: &'static str =
        "supervisor_block_replacement_success_total";

    /// Identifier for block replacement errors.
    /// Labels: `chain_id`
    pub(crate) const BLOCK_REPLACEMENT_ERROR_TOTAL: &'static str =
        "supervisor_block_replacement_error_total";

    /// Identifier for block replacement latency.
    /// Labels: `chain_id`
    pub(crate) const BLOCK_REPLACEMENT_LATENCY_SECONDS: &'static str =
        "supervisor_block_replacement_latency_seconds";

    pub(crate) const BLOCK_REPLACEMENT_METHOD_REPLACE_BLOCK: &'static str = "replace_block";

    // --- Safety Head Ref Metric Names ---
    /// Identifier for safety head ref.
    /// Labels: `chain_id`, `type`
    pub(crate) const SAFETY_HEAD_REF_LABELS: &'static str = "supervisor_safety_head_ref_labels";

    pub(crate) fn init(chain_id: ChainId) {
        Self::describe();
        Self::zero(chain_id);
    }

    fn describe() {
        metrics::describe_counter!(
            Self::BLOCK_PROCESSING_SUCCESS_TOTAL,
            metrics::Unit::Count,
            "Total number of successfully processed blocks in the supervisor",
        );

        metrics::describe_counter!(
            Self::BLOCK_PROCESSING_ERROR_TOTAL,
            metrics::Unit::Count,
            "Total number of errors encountered while processing blocks in the supervisor",
        );

        metrics::describe_histogram!(
            Self::BLOCK_PROCESSING_LATENCY_SECONDS,
            metrics::Unit::Seconds,
            "Latency for processing in the supervisor",
        );

        metrics::describe_counter!(
            Self::BLOCK_INVALIDATION_SUCCESS_TOTAL,
            metrics::Unit::Count,
            "Total number of successfully invalidated blocks in the supervisor",
        );

        metrics::describe_counter!(
            Self::BLOCK_INVALIDATION_ERROR_TOTAL,
            metrics::Unit::Count,
            "Total number of errors encountered while invalidating blocks in the supervisor",
        );

        metrics::describe_histogram!(
            Self::BLOCK_INVALIDATION_LATENCY_SECONDS,
            metrics::Unit::Seconds,
            "Latency for invalidating blocks in the supervisor",
        );

        metrics::describe_counter!(
            Self::BLOCK_REPLACEMENT_SUCCESS_TOTAL,
            metrics::Unit::Count,
            "Total number of successfully replaced blocks in the supervisor",
        );

        metrics::describe_counter!(
            Self::BLOCK_REPLACEMENT_ERROR_TOTAL,
            metrics::Unit::Count,
            "Total number of errors encountered while replacing blocks in the supervisor",
        );

        metrics::describe_histogram!(
            Self::BLOCK_REPLACEMENT_LATENCY_SECONDS,
            metrics::Unit::Seconds,
            "Latency for replacing blocks in the supervisor",
        );

        metrics::describe_gauge!(Self::SAFETY_HEAD_REF_LABELS, "Supervisor safety head ref",);
    }

    fn zero_block_processing(chain_id: ChainId, block_type: &'static str) {
        metrics::counter!(
            Self::BLOCK_PROCESSING_SUCCESS_TOTAL,
            "type" => block_type,
            "chain_id" => chain_id.to_string()
        )
        .increment(0);

        metrics::counter!(
            Self::BLOCK_PROCESSING_ERROR_TOTAL,
            "type" => block_type,
            "chain_id" => chain_id.to_string()
        )
        .increment(0);

        metrics::histogram!(
            Self::BLOCK_PROCESSING_LATENCY_SECONDS,
            "type" => block_type,
            "chain_id" => chain_id.to_string()
        )
        .record(0.0);
    }

    fn zero_safety_head_ref(chain_id: ChainId, head_type: &'static str) {
        metrics::gauge!(
            Self::SAFETY_HEAD_REF_LABELS,
            "type" => head_type,
            "chain_id" => chain_id.to_string(),
        )
        .set(0.0);
    }

    fn zero_block_invalidation(chain_id: ChainId) {
        metrics::counter!(
            Self::BLOCK_INVALIDATION_SUCCESS_TOTAL,
            "method" => Self::BLOCK_INVALIDATION_METHOD_INVALIDATE_BLOCK,
            "chain_id" => chain_id.to_string()
        )
        .increment(0);

        metrics::counter!(
            Self::BLOCK_INVALIDATION_ERROR_TOTAL,
            "method" => Self::BLOCK_INVALIDATION_METHOD_INVALIDATE_BLOCK,
            "chain_id" => chain_id.to_string()
        )
        .increment(0);

        metrics::histogram!(
            Self::BLOCK_INVALIDATION_LATENCY_SECONDS,
            "method" => Self::BLOCK_INVALIDATION_METHOD_INVALIDATE_BLOCK,
            "chain_id" => chain_id.to_string()
        )
        .record(0.0);
    }

    fn zero_block_replacement(chain_id: ChainId) {
        metrics::counter!(
            Self::BLOCK_REPLACEMENT_SUCCESS_TOTAL,
            "method" => Self::BLOCK_REPLACEMENT_METHOD_REPLACE_BLOCK,
            "chain_id" => chain_id.to_string()
        )
        .increment(0);

        metrics::counter!(
            Self::BLOCK_REPLACEMENT_ERROR_TOTAL,
            "method" => Self::BLOCK_REPLACEMENT_METHOD_REPLACE_BLOCK,
            "chain_id" => chain_id.to_string()
        )
        .increment(0);

        metrics::histogram!(
            Self::BLOCK_REPLACEMENT_LATENCY_SECONDS,
            "method" => Self::BLOCK_REPLACEMENT_METHOD_REPLACE_BLOCK,
            "chain_id" => chain_id.to_string()
        )
        .record(0.0);
    }

    fn zero(chain_id: ChainId) {
        Self::zero_block_processing(chain_id, Self::BLOCK_TYPE_LOCAL_UNSAFE);
        Self::zero_block_processing(chain_id, Self::BLOCK_TYPE_CROSS_UNSAFE);
        Self::zero_block_processing(chain_id, Self::BLOCK_TYPE_LOCAL_SAFE);
        Self::zero_block_processing(chain_id, Self::BLOCK_TYPE_CROSS_SAFE);
        Self::zero_block_processing(chain_id, Self::BLOCK_TYPE_FINALIZED);

        Self::zero_block_invalidation(chain_id);
        Self::zero_block_replacement(chain_id);

        Self::zero_safety_head_ref(chain_id, Self::BLOCK_TYPE_LOCAL_UNSAFE);
        Self::zero_safety_head_ref(chain_id, Self::BLOCK_TYPE_CROSS_UNSAFE);
        Self::zero_safety_head_ref(chain_id, Self::BLOCK_TYPE_LOCAL_SAFE);
        Self::zero_safety_head_ref(chain_id, Self::BLOCK_TYPE_CROSS_SAFE);
        Self::zero_safety_head_ref(chain_id, Self::BLOCK_TYPE_FINALIZED);
    }

    /// Records metrics for a block processing operation.
    /// Takes the result of the processing and extracts the block info if successful.
    pub(crate) fn record_block_processing(
        chain_id: ChainId,
        block_type: &'static str,
        result: &Result<BlockInfo, ChainProcessorError>,
    ) {
        match result {
            Ok(block) => {
                metrics::counter!(
                    Self::BLOCK_PROCESSING_SUCCESS_TOTAL,
                    "type" => block_type,
                    "chain_id" => chain_id.to_string()
                )
                .increment(1);

                metrics::gauge!(
                    Self::SAFETY_HEAD_REF_LABELS,
                    "type" => block_type,
                    "chain_id" => chain_id.to_string(),
                )
                .set(block.number as f64);

                // Calculate latency
                match SystemTime::now().duration_since(std::time::UNIX_EPOCH) {
                    Ok(duration) => {
                        let now = duration.as_secs_f64();
                        let latency = now - block.timestamp as f64;

                        metrics::histogram!(
                            Self::BLOCK_PROCESSING_LATENCY_SECONDS,
                            "type" => block_type,
                            "chain_id" => chain_id.to_string()
                        )
                        .record(latency);
                    }
                    Err(err) => {
                        error!(
                            target: "chain_processor",
                            chain_id = chain_id,
                            %err,
                            "SystemTime error when recording block processing latency"
                        );
                    }
                }
            }
            Err(_) => {
                metrics::counter!(
                    Self::BLOCK_PROCESSING_ERROR_TOTAL,
                    "type" => block_type,
                    "chain_id" => chain_id.to_string()
                )
                .increment(1);
            }
        }
    }
}

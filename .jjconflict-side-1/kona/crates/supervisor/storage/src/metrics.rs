use alloy_primitives::ChainId;

/// Container for ChainDb metrics.
#[derive(Debug, Clone)]
pub(crate) struct Metrics;

// todo: implement this using the reth metrics for tables
impl Metrics {
    pub(crate) const STORAGE_REQUESTS_SUCCESS_TOTAL: &'static str =
        "kona_supervisor_storage_success_total";
    pub(crate) const STORAGE_REQUESTS_ERROR_TOTAL: &'static str =
        "kona_supervisor_storage_error_total";
    pub(crate) const STORAGE_REQUEST_DURATION_SECONDS: &'static str =
        "kona_supervisor_storage_duration_seconds";

    pub(crate) const STORAGE_METHOD_DERIVED_TO_SOURCE: &'static str = "derived_to_source";
    pub(crate) const STORAGE_METHOD_LATEST_DERIVED_BLOCK_AT_SOURCE: &'static str =
        "latest_derived_block_at_source";
    pub(crate) const STORAGE_METHOD_LATEST_DERIVATION_STATE: &'static str =
        "latest_derivation_state";
    pub(crate) const STORAGE_METHOD_GET_SOURCE_BLOCK: &'static str = "get_source_block";
    pub(crate) const STORAGE_METHOD_GET_ACTIVATION_BLOCK: &'static str = "get_activation_block";
    pub(crate) const STORAGE_METHOD_INITIALISE_DERIVATION_STORAGE: &'static str =
        "initialise_derivation_storage";
    pub(crate) const STORAGE_METHOD_SAVE_DERIVED_BLOCK: &'static str = "save_derived_block";
    pub(crate) const STORAGE_METHOD_SAVE_SOURCE_BLOCK: &'static str = "save_source_block";
    pub(crate) const STORAGE_METHOD_GET_LATEST_BLOCK: &'static str = "get_latest_block";
    pub(crate) const STORAGE_METHOD_GET_BLOCK: &'static str = "get_block";
    pub(crate) const STORAGE_METHOD_GET_LOG: &'static str = "get_log";
    pub(crate) const STORAGE_METHOD_GET_LOGS: &'static str = "get_logs";
    pub(crate) const STORAGE_METHOD_INITIALISE_LOG_STORAGE: &'static str = "initialise_log_storage";
    pub(crate) const STORAGE_METHOD_STORE_BLOCK_LOGS: &'static str = "store_block_logs";
    pub(crate) const STORAGE_METHOD_GET_SAFETY_HEAD_REF: &'static str = "get_safety_head_ref";
    pub(crate) const STORAGE_METHOD_GET_SUPER_HEAD: &'static str = "get_super_head";
    pub(crate) const STORAGE_METHOD_UPDATE_FINALIZED_USING_SOURCE: &'static str =
        "update_finalized_using_source";
    pub(crate) const STORAGE_METHOD_UPDATE_CURRENT_CROSS_UNSAFE: &'static str =
        "update_current_cross_unsafe";
    pub(crate) const STORAGE_METHOD_UPDATE_CURRENT_CROSS_SAFE: &'static str =
        "update_current_cross_safe";
    pub(crate) const STORAGE_METHOD_UPDATE_FINALIZED_L1: &'static str = "update_finalized_l1";
    pub(crate) const STORAGE_METHOD_GET_FINALIZED_L1: &'static str = "get_finalized_l1";
    pub(crate) const STORAGE_METHOD_REWIND_LOG_STORAGE: &'static str = "rewind_log_storage";
    pub(crate) const STORAGE_METHOD_REWIND: &'static str = "rewind";
    pub(crate) const STORAGE_METHOD_REWIND_TO_SOURCE: &'static str = "rewind_to_source";

    pub(crate) fn init(chain_id: ChainId) {
        Self::describe();
        Self::zero(chain_id);
    }

    fn describe() {
        metrics::describe_counter!(
            Self::STORAGE_REQUESTS_SUCCESS_TOTAL,
            metrics::Unit::Count,
            "Total number of successful Kona Supervisor Storage requests"
        );
        metrics::describe_counter!(
            Self::STORAGE_REQUESTS_ERROR_TOTAL,
            metrics::Unit::Count,
            "Total number of failed Kona Supervisor Storage requests"
        );
        metrics::describe_histogram!(
            Self::STORAGE_REQUEST_DURATION_SECONDS,
            metrics::Unit::Seconds,
            "Duration of Kona Supervisor Storage requests"
        );
    }

    fn zero_storage_methods(chain_id: ChainId, method_name: &'static str) {
        metrics::counter!(
            Self::STORAGE_REQUESTS_SUCCESS_TOTAL,
            "method" => method_name,
            "chain_id" => chain_id.to_string()
        )
        .increment(0);

        metrics::counter!(
            Self::STORAGE_REQUESTS_ERROR_TOTAL,
            "method" => method_name,
            "chain_id" => chain_id.to_string()
        )
        .increment(0);

        metrics::histogram!(
            Self::STORAGE_REQUEST_DURATION_SECONDS,
            "method" => method_name,
            "chain_id" => chain_id.to_string()
        )
        .record(0.0);
    }

    fn zero(chain_id: ChainId) {
        Self::zero_storage_methods(chain_id, Self::STORAGE_METHOD_DERIVED_TO_SOURCE);
        Self::zero_storage_methods(chain_id, Self::STORAGE_METHOD_LATEST_DERIVED_BLOCK_AT_SOURCE);
        Self::zero_storage_methods(chain_id, Self::STORAGE_METHOD_LATEST_DERIVATION_STATE);
        Self::zero_storage_methods(chain_id, Self::STORAGE_METHOD_GET_SOURCE_BLOCK);
        Self::zero_storage_methods(chain_id, Self::STORAGE_METHOD_INITIALISE_DERIVATION_STORAGE);
        Self::zero_storage_methods(chain_id, Self::STORAGE_METHOD_SAVE_DERIVED_BLOCK);
        Self::zero_storage_methods(chain_id, Self::STORAGE_METHOD_SAVE_SOURCE_BLOCK);
        Self::zero_storage_methods(chain_id, Self::STORAGE_METHOD_GET_LATEST_BLOCK);
        Self::zero_storage_methods(chain_id, Self::STORAGE_METHOD_GET_BLOCK);
        Self::zero_storage_methods(chain_id, Self::STORAGE_METHOD_GET_LOG);
        Self::zero_storage_methods(chain_id, Self::STORAGE_METHOD_GET_LOGS);
        Self::zero_storage_methods(chain_id, Self::STORAGE_METHOD_INITIALISE_LOG_STORAGE);
        Self::zero_storage_methods(chain_id, Self::STORAGE_METHOD_STORE_BLOCK_LOGS);
        Self::zero_storage_methods(chain_id, Self::STORAGE_METHOD_GET_SAFETY_HEAD_REF);
        Self::zero_storage_methods(chain_id, Self::STORAGE_METHOD_GET_SUPER_HEAD);
        Self::zero_storage_methods(chain_id, Self::STORAGE_METHOD_UPDATE_FINALIZED_USING_SOURCE);
        Self::zero_storage_methods(chain_id, Self::STORAGE_METHOD_UPDATE_CURRENT_CROSS_UNSAFE);
        Self::zero_storage_methods(chain_id, Self::STORAGE_METHOD_UPDATE_CURRENT_CROSS_SAFE);
        Self::zero_storage_methods(chain_id, Self::STORAGE_METHOD_UPDATE_FINALIZED_L1);
        Self::zero_storage_methods(chain_id, Self::STORAGE_METHOD_GET_FINALIZED_L1);
        Self::zero_storage_methods(chain_id, Self::STORAGE_METHOD_REWIND_LOG_STORAGE);
        Self::zero_storage_methods(chain_id, Self::STORAGE_METHOD_REWIND);
        Self::zero_storage_methods(chain_id, Self::STORAGE_METHOD_REWIND_TO_SOURCE);
    }
}

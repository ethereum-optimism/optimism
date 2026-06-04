//! Optimism interop txpool metrics.

use crate::interop_filter::InteropTxValidatorError;
use op_alloy_rpc_types::SuperchainDAError;
use reth_metrics::{
    Metrics,
    metrics::{Gauge, Histogram, counter},
};
use std::time::Duration;

/// Fully-qualified name of the per-decision filter outcome counter.
const FILTER_DECISIONS: &str = "optimism_transaction_pool.interop.filter_decisions_total";
/// Fully-qualified name of the legitimate-rejection reason breakdown counter.
const VALIDATION_ERRORS: &str = "optimism_transaction_pool.interop.validation_errors_total";
/// Fully-qualified name of the infra-failure breakdown counter (filter unreachable / no
/// trustworthy answer), kept separate from legitimate rejections so an outage does not look
/// like a flood of rejected transactions.
const FILTER_ERRORS: &str = "optimism_transaction_pool.interop.filter_errors_total";

/// `result` label values for [`InteropMetrics::record_decision`]. Every transaction that reaches
/// the interop filter records exactly one of these.
pub(crate) const RESULT_ALLOWED: &str = "allowed";
pub(crate) const RESULT_REJECTED_FAILSAFE: &str = "rejected_failsafe";
/// Rejected because the interop hardfork is not yet active at this block (a cross-chain tx cannot
/// exist pre-activation). A local gate — the filter is not contacted.
pub(crate) const RESULT_REJECTED_PRE_INTEROP_HARDFORK: &str = "rejected_pre_interop_hardfork";
/// The filter answered and rejected the tx (a genuine invalid-entry verdict).
pub(crate) const RESULT_REJECTED_VALIDATION: &str = "rejected_validation";
/// The filter could not be reached or did not return a trustworthy answer (timeout / transport /
/// server error). The tx is still rejected, but this is an infra failure, not the chain's fault.
pub(crate) const RESULT_ERRORED: &str = "errored";

/// Optimism interop txpool metrics.
#[derive(Metrics, Clone)]
#[metrics(scope = "optimism_transaction_pool.interop")]
pub struct InteropMetrics {
    /// How long it takes to query the interop filter in the Optimism transaction pool.
    pub(crate) interop_query_latency: Histogram,

    /// Current interop failsafe state: `1` if failsafe is enabled (all interop txs are
    /// rejected/evicted), `0` if disabled. Refreshed on every failsafe poll.
    pub(crate) failsafe_enabled: Gauge,
}

impl InteropMetrics {
    /// Records the duration of interop filter queries.
    #[inline]
    pub fn record_interop_query(&self, duration: Duration) {
        self.interop_query_latency.record(duration.as_secs_f64());
    }

    /// Records the current interop failsafe state (`1` = enabled, `0` = disabled).
    #[inline]
    pub fn set_failsafe_enabled(&self, enabled: bool) {
        self.failsafe_enabled.set(if enabled { 1.0 } else { 0.0 });
    }

    /// Records a single interop filter decision under the given `result` label.
    ///
    /// `result` must be one of the `RESULT_*` constants in this module so the label set stays
    /// low-cardinality and stable.
    #[inline]
    pub fn record_decision(&self, result: &'static str) {
        counter!(FILTER_DECISIONS, "result" => result).increment(1);
    }

    /// Records a legitimate validation rejection (the filter answered and rejected the tx), broken
    /// down by DA `reason`. Always increments exactly once, so `sum(validation_errors_total)`
    /// equals `filter_decisions_total{result="rejected_validation"}`.
    pub fn record_validation_error(&self, error: &SuperchainDAError) {
        counter!(VALIDATION_ERRORS, "reason" => validation_reason(error)).increment(1);
    }

    /// Records an infra failure where the filter could not return a trustworthy answer, broken
    /// down by `kind`. Kept separate from [`Self::record_validation_error`] so a filter outage
    /// does not corrupt the legitimate-rejection signal. Always increments exactly once, so
    /// `sum(filter_errors_total)` equals `filter_decisions_total{result="errored"}`.
    pub fn record_filter_error(&self, error: &InteropTxValidatorError) {
        counter!(FILTER_ERRORS, "kind" => filter_error_kind(error)).increment(1);
    }
}

/// Maps a DA validation error to its stable `reason` label for `validation_errors_total`.
pub(crate) const fn validation_reason(error: &SuperchainDAError) -> &'static str {
    match error {
        SuperchainDAError::SkippedData => "skipped_data",
        SuperchainDAError::UnknownChain => "unknown_chain",
        SuperchainDAError::ConflictingData => "conflicting_data",
        SuperchainDAError::IneffectiveData => "ineffective_data",
        SuperchainDAError::OutOfOrder => "out_of_order",
        SuperchainDAError::AwaitingReplacement => "awaiting_replacement",
        SuperchainDAError::OutOfScope => "out_of_scope",
        SuperchainDAError::NoParentForFirstBlock => "no_parent_for_first_block",
        SuperchainDAError::FutureData => "future_data",
        SuperchainDAError::MissedData => "missed_data",
        SuperchainDAError::DataCorruption => "data_corruption",
        _ => "other",
    }
}

/// Maps an infra failure to its stable `kind` label for `filter_errors_total`. `InvalidEntry` is a
/// legitimate rejection, not an infra failure, and is never routed here; it (and any transport /
/// server error) is classified as `other` defensively.
pub(crate) const fn filter_error_kind(error: &InteropTxValidatorError) -> &'static str {
    match error {
        InteropTxValidatorError::Timeout(_) => "timeout",
        _ => "other",
    }
}

/// Maps a filter validation failure to its decision `result` label, separating a legitimate
/// rejection (the filter answered "no") from an infra failure (no trustworthy answer). This is the
/// single source of truth for which `result` the error path records.
pub(crate) const fn validation_failure_result(error: &InteropTxValidatorError) -> &'static str {
    match error {
        InteropTxValidatorError::InvalidEntry(_) => RESULT_REJECTED_VALIDATION,
        _ => RESULT_ERRORED,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn server_error() -> InteropTxValidatorError {
        InteropTxValidatorError::other(std::io::Error::other("filter unreachable"))
    }

    #[test]
    fn validation_reason_maps_every_da_error() {
        assert_eq!(validation_reason(&SuperchainDAError::SkippedData), "skipped_data");
        assert_eq!(validation_reason(&SuperchainDAError::UnknownChain), "unknown_chain");
        assert_eq!(validation_reason(&SuperchainDAError::ConflictingData), "conflicting_data");
        assert_eq!(validation_reason(&SuperchainDAError::IneffectiveData), "ineffective_data");
        assert_eq!(validation_reason(&SuperchainDAError::OutOfOrder), "out_of_order");
        assert_eq!(
            validation_reason(&SuperchainDAError::AwaitingReplacement),
            "awaiting_replacement"
        );
        assert_eq!(validation_reason(&SuperchainDAError::OutOfScope), "out_of_scope");
        assert_eq!(
            validation_reason(&SuperchainDAError::NoParentForFirstBlock),
            "no_parent_for_first_block"
        );
        assert_eq!(validation_reason(&SuperchainDAError::FutureData), "future_data");
        assert_eq!(validation_reason(&SuperchainDAError::MissedData), "missed_data");
        assert_eq!(validation_reason(&SuperchainDAError::DataCorruption), "data_corruption");
        // Unmapped DA variants fall back to a stable catch-all.
        assert_eq!(validation_reason(&SuperchainDAError::InvalidatedRead), "other");
    }

    #[test]
    fn filter_error_kind_distinguishes_timeout_from_other() {
        assert_eq!(filter_error_kind(&InteropTxValidatorError::Timeout(2)), "timeout");
        assert_eq!(filter_error_kind(&server_error()), "other");
        // A legitimate rejection is never routed here, but is defensively classified as `other`.
        assert_eq!(
            filter_error_kind(&InteropTxValidatorError::InvalidEntry(
                SuperchainDAError::SkippedData
            )),
            "other"
        );
    }

    #[test]
    fn validation_failure_result_separates_infra_failure_from_rejection() {
        // The filter answered "no" -> legitimate rejection.
        assert_eq!(
            validation_failure_result(&InteropTxValidatorError::InvalidEntry(
                SuperchainDAError::ConflictingData
            )),
            RESULT_REJECTED_VALIDATION
        );
        // No trustworthy answer -> infra failure, must not look like a rejection.
        assert_eq!(validation_failure_result(&InteropTxValidatorError::Timeout(2)), RESULT_ERRORED);
        assert_eq!(validation_failure_result(&server_error()), RESULT_ERRORED);
    }
}

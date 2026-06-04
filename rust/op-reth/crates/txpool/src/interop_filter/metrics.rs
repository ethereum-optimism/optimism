//! Optimism interop txpool metrics.

use crate::interop_filter::InteropTxValidatorError;
use op_alloy_rpc_types::SuperchainDAError;
use reth_metrics::{
    Metrics,
    metrics::{Counter, Gauge, Histogram, counter, histogram},
};
use std::time::Duration;

/// Fully-qualified name of the single per-tx interop filter decision counter. Every interop tx that
/// reaches the filter increments this exactly once, labeled by `result` (and `reason`, where a
/// result carries a sub-classification). It is the one source of truth for the decision taxonomy:
/// `sum by (result)` is the outcome breakdown (including the former `quorum_*` counters), and
/// `sum by (reason) (filter_decisions_total{result="rejected_invalid"})` is the DA-reason breakdown
/// (the former per-reason `*_count` counters).
const FILTER_DECISIONS: &str = "optimism_transaction_pool.interop.filter_decisions_total";

/// Fully-qualified name of the per-endpoint verdict counter, labeled by `endpoint` index and
/// `verdict`. Answers *which* endpoint is returning invalids or going unavailable. Labeled by index
/// (not the raw URL), since interop-http URLs can carry basic-auth credentials.
const ENDPOINT_VERDICTS: &str = "optimism_transaction_pool.interop.endpoint.verdicts_total";

/// Fully-qualified name of the per-endpoint query-latency histogram. Matches the metric the derived
/// `EndpointMetrics` used to emit, so existing dashboards keep working.
const ENDPOINT_QUERY_LATENCY: &str = "optimism_transaction_pool.interop.endpoint.query_latency";

/// `result` label values for [`InteropMetrics::record_decision`]. Every transaction that reaches
/// the interop filter records exactly one of these. Low-cardinality and stable.
pub(crate) const RESULT_ALLOWED: &str = "allowed";
/// Rejected because the interop hardfork is not yet active at this block (a cross-chain tx cannot
/// exist pre-activation). A local gate — the filter is not contacted.
pub(crate) const RESULT_REJECTED_PRE_INTEROP: &str = "rejected_pre_interop";
/// Rejected because failsafe is active (fast-path cached gate, or an endpoint reported failsafe
/// during the check). All interop txs are rejected while failsafe is on.
pub(crate) const RESULT_REJECTED_FAILSAFE: &str = "rejected_failsafe";
/// The filter reached quorum and the verdict was invalid (a genuine invalid-entry verdict). The
/// `reason` label carries the DA reason.
pub(crate) const RESULT_REJECTED_INVALID: &str = "rejected_invalid";
/// The endpoints that responded definitively disagreed (a mix of valid and invalid).
pub(crate) const RESULT_REJECTED_DISAGREEMENT: &str = "rejected_disagreement";
/// Too few definitive verdicts were collected to satisfy the configured quorum (fail closed).
pub(crate) const RESULT_REJECTED_NO_QUORUM: &str = "rejected_no_quorum";
/// No trustworthy answer was produced for this tx (a non-response surfaced as the decision). The
/// quorum aggregator never returns this — it folds non-responses into `rejected_no_quorum` — so
/// this is a defensive catch-all that keeps every decision counted.
pub(crate) const RESULT_ERRORED: &str = "errored";

/// `reason` label value when a `result` has no sub-classification.
pub(crate) const REASON_NONE: &str = "none";
/// `reason` label value for a definitive rejection that carries no recognized DA reason (a generic
/// filter rejection, or a DA variant without a dedicated label), and for an unclassified error.
pub(crate) const REASON_OTHER: &str = "other";
/// `reason` label value for a query that timed out.
pub(crate) const REASON_TIMEOUT: &str = "timeout";

/// `verdict` label values for [`EndpointMetrics`].
const VERDICT_VALID: &str = "valid";
const VERDICT_INVALID: &str = "invalid";
const VERDICT_UNAVAILABLE: &str = "unavailable";

/// Optimism interop txpool metrics.
#[derive(Metrics, Clone)]
#[metrics(scope = "optimism_transaction_pool.interop")]
pub struct InteropMetrics {
    /// How long it takes to query the interop filter in the Optimism transaction pool (fleet-wide
    /// across endpoints).
    pub(crate) interop_query_latency: Histogram,

    /// Current interop failsafe state: `1` if failsafe is enabled (all interop txs are
    /// rejected/evicted), `0` if disabled. Refreshed on every failsafe poll.
    pub(crate) failsafe_enabled: Gauge,

    /// Distribution of definitive verdicts (valid + invalid) collected per check. Watch the low
    /// percentile against [`quorum_min_responses`](Self::quorum_min_responses): when it approaches
    /// the threshold, the filter is one endpoint blip away from failing closed on every interop tx
    /// — a leading indicator the outcome counts alone cannot give.
    pub(crate) quorum_verdicts_collected: Histogram,

    /// The configured quorum threshold (minimum definitive verdicts required to decide a check).
    /// Set once at startup so a dashboard can draw the fail-closed line without hardcoding config.
    pub(crate) quorum_min_responses: Gauge,
}

/// Per-endpoint interop metrics, labeled by endpoint index so each configured interop filter can be
/// observed independently. With the fleet-wide counters alone you can tell *an* endpoint is slow or
/// down; these labeled metrics answer *which* one — the first question on call for a fan-out
/// client.
///
/// Hand-rolled rather than `#[derive(Metrics)]` so the three verdicts collapse into a single
/// `verdicts_total{verdict=…}` counter instead of three separate metrics.
#[derive(Clone, Debug)]
pub struct EndpointMetrics {
    /// How long this endpoint took to answer an `interop_checkAccessList` query. Lets an endpoint
    /// creeping toward the request timeout be spotted before it starts failing.
    query_latency: Histogram,
    /// Definitive valid verdicts returned by this endpoint.
    valid: Counter,
    /// Definitive invalid verdicts returned by this endpoint.
    invalid: Counter,
    /// Non-responses from this endpoint (timeout, transport error, soft out-of-sync,
    /// cancellation).
    unavailable: Counter,
}

impl EndpointMetrics {
    /// Builds per-endpoint metrics labeled with the endpoint's index. Counter and histogram handles
    /// are resolved once here (not per-call) since each endpoint is built once at startup.
    pub fn for_endpoint(index: usize) -> Self {
        let endpoint = index.to_string();
        Self {
            query_latency: histogram!(ENDPOINT_QUERY_LATENCY, "endpoint" => endpoint.clone()),
            valid: counter!(ENDPOINT_VERDICTS, "endpoint" => endpoint.clone(), "verdict" => VERDICT_VALID),
            invalid: counter!(ENDPOINT_VERDICTS, "endpoint" => endpoint.clone(), "verdict" => VERDICT_INVALID),
            unavailable: counter!(ENDPOINT_VERDICTS, "endpoint" => endpoint, "verdict" => VERDICT_UNAVAILABLE),
        }
    }

    /// Records the duration of this endpoint's interop filter query.
    #[inline]
    pub fn record_query(&self, duration: Duration) {
        self.query_latency.record(duration.as_secs_f64());
    }

    /// Records this endpoint's definitive valid verdict.
    #[inline]
    pub fn record_valid(&self) {
        self.valid.increment(1);
    }

    /// Records this endpoint's definitive invalid verdict.
    #[inline]
    pub fn record_invalid(&self) {
        self.invalid.increment(1);
    }

    /// Records this endpoint's non-response (timeout, transport error, soft out-of-sync).
    #[inline]
    pub fn record_unavailable(&self) {
        self.unavailable.increment(1);
    }
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

    /// Records the configured quorum threshold. Called once at startup.
    #[inline]
    pub fn set_quorum_min_responses(&self, min_responses: usize) {
        self.quorum_min_responses.set(min_responses as f64);
    }

    /// Records how many definitive verdicts (valid + invalid) a check collected, for the
    /// fail-closed margin signal. Called once per check.
    #[inline]
    pub fn record_quorum_verdicts_collected(&self, collected: usize) {
        self.quorum_verdicts_collected.record(collected as f64);
    }

    /// Records a single interop filter decision under the given `result` and `reason` labels.
    ///
    /// Both must be one of the `RESULT_*` / `REASON_*` constants (or a DA reason from
    /// [`validation_reason`]) so the label set stays low-cardinality and stable. Use
    /// [`decision_for_error`] to derive the pair from a validation error.
    #[inline]
    pub fn record_decision(&self, result: &'static str, reason: &'static str) {
        counter!(FILTER_DECISIONS, "result" => result, "reason" => reason).increment(1);
    }
}

/// Maps a DA validation error to its stable `reason` label.
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
        _ => REASON_OTHER,
    }
}

/// Maps a validation failure to its `(result, reason)` decision labels. This is the single source
/// of truth for how an [`InteropTxValidatorError`] is classified on the `filter_decisions_total`
/// counter. Failsafe is handled before this is reached (it rejects on the fast-path and on
/// detection), but is mapped here too for completeness.
///
/// The quorum aggregator only ever returns `InvalidEntry`, `Rejected`, `Disagreement`,
/// `QuorumNotReached`, or `FailsafeEnabled`; the remaining arms are defensive so every decision is
/// counted regardless of future changes to the aggregator.
pub(crate) const fn decision_for_error(
    error: &InteropTxValidatorError,
) -> (&'static str, &'static str) {
    match error {
        // The filter reached quorum and answered "invalid".
        InteropTxValidatorError::InvalidEntry(reason) => {
            (RESULT_REJECTED_INVALID, validation_reason(reason))
        }
        // A definitive rejection that is not a recognized DA code (e.g. a generic `-32602`).
        InteropTxValidatorError::Rejected { .. } => (RESULT_REJECTED_INVALID, REASON_OTHER),
        InteropTxValidatorError::Disagreement { .. } => (RESULT_REJECTED_DISAGREEMENT, REASON_NONE),
        InteropTxValidatorError::QuorumNotReached { .. } => {
            (RESULT_REJECTED_NO_QUORUM, REASON_NONE)
        }
        InteropTxValidatorError::FailsafeEnabled => (RESULT_REJECTED_FAILSAFE, REASON_NONE),
        InteropTxValidatorError::Timeout(_) => (RESULT_ERRORED, REASON_TIMEOUT),
        // Non-responses that never surface as the aggregate's decision; counted defensively.
        InteropTxValidatorError::DataUnavailable { .. } | InteropTxValidatorError::Other(_) => {
            (RESULT_ERRORED, REASON_OTHER)
        }
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
        assert_eq!(validation_reason(&SuperchainDAError::InvalidatedRead), REASON_OTHER);
    }

    #[test]
    fn decision_for_error_maps_invalid_to_da_reason() {
        assert_eq!(
            decision_for_error(&InteropTxValidatorError::InvalidEntry(
                SuperchainDAError::ConflictingData
            )),
            (RESULT_REJECTED_INVALID, "conflicting_data")
        );
        // A generic (non-DA) rejection is still a genuine invalid verdict, reason `other`.
        assert_eq!(
            decision_for_error(&InteropTxValidatorError::Rejected {
                code: -32602,
                message: "failed to parse access entry".to_string(),
            }),
            (RESULT_REJECTED_INVALID, REASON_OTHER)
        );
    }

    #[test]
    fn decision_for_error_separates_quorum_outcomes() {
        assert_eq!(
            decision_for_error(&InteropTxValidatorError::Disagreement { valid: 1, invalid: 1 }),
            (RESULT_REJECTED_DISAGREEMENT, REASON_NONE)
        );
        assert_eq!(
            decision_for_error(&InteropTxValidatorError::QuorumNotReached {
                received: 1,
                required: 2
            }),
            (RESULT_REJECTED_NO_QUORUM, REASON_NONE)
        );
        assert_eq!(
            decision_for_error(&InteropTxValidatorError::FailsafeEnabled),
            (RESULT_REJECTED_FAILSAFE, REASON_NONE)
        );
    }

    #[test]
    fn decision_for_error_classifies_non_responses_as_errored() {
        assert_eq!(
            decision_for_error(&InteropTxValidatorError::Timeout(2)),
            (RESULT_ERRORED, REASON_TIMEOUT)
        );
        assert_eq!(
            decision_for_error(&InteropTxValidatorError::DataUnavailable { code: -321401 }),
            (RESULT_ERRORED, REASON_OTHER)
        );
        assert_eq!(decision_for_error(&server_error()), (RESULT_ERRORED, REASON_OTHER));
    }
}

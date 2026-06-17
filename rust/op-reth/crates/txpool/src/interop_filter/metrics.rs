//! Optimism interop txpool metrics.

use crate::interop_filter::InteropTxValidatorError;
use op_alloy_rpc_types::SuperchainDAError;
use reth_metrics::{
    Metrics,
    metrics::{
        Counter, Gauge, Histogram, Unit, counter, describe_counter, describe_gauge,
        describe_histogram, gauge, histogram,
    },
};
use std::time::Duration;

/// Fully-qualified name of the single per-tx interop filter decision counter. Every interop tx that
/// reaches the filter increments this exactly once, labeled by `result` (and `reason`, where a
/// result carries a sub-classification). It is the one source of truth for the decision taxonomy:
/// `sum by (result)` is the outcome breakdown (including the former `quorum_*` counters), and
/// `sum by (reason) (filter_decisions{result="rejected_invalid"})` is the DA-reason breakdown
/// (the former per-reason `*_count` counters). No `_total` suffix: the Prometheus recorder does not
/// append one (unit suffixes are disabled), matching the rest of the reth counter naming.
const FILTER_DECISIONS: &str = "optimism_transaction_pool.interop.filter_decisions";

/// Fully-qualified name of the per-endpoint verdict counter, labeled by `endpoint` index and
/// `verdict`. Answers *which* endpoint is returning invalids or going unavailable. Labeled by index
/// (not the raw URL), since interop-http URLs can carry basic-auth credentials.
const ENDPOINT_VERDICTS: &str = "optimism_transaction_pool.interop.endpoint.verdicts";

/// Fully-qualified name of the per-endpoint query-latency histogram. Matches the metric the derived
/// `EndpointMetrics` used to emit, so existing dashboards keep working.
const ENDPOINT_QUERY_LATENCY: &str = "optimism_transaction_pool.interop.endpoint.query_latency";

/// Fully-qualified name of the per-endpoint health gauge: `1` if the endpoint answered its last
/// *completed* check (valid or invalid — both mean it responded), `0` if that check was a
/// non-response. Left untouched when a check is cancelled by fast-accept, so it reflects the last
/// real outcome. `sum(endpoint.up)` against
/// [`quorum_min_responses`](InteropMetrics::quorum_min_responses) is the fail-closed headroom: it
/// drops as endpoints fall away, *before* quorum is lost.
const ENDPOINT_UP: &str = "optimism_transaction_pool.interop.endpoint.up";

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

    /// The configured quorum threshold (minimum definitive verdicts required to decide a check).
    /// Set once at startup so a dashboard can draw the fail-closed line for `sum(endpoint.up)`
    /// without hardcoding config.
    pub(crate) quorum_min_responses: Gauge,
}

/// Per-endpoint interop metrics, labeled by endpoint index so each configured interop filter can be
/// observed independently. With the fleet-wide counters alone you can tell *an* endpoint is slow or
/// down; these labeled metrics answer *which* one — the first question on call for a fan-out
/// client.
///
/// Hand-rolled rather than `#[derive(Metrics)]` so the three verdicts collapse into a single
/// `verdicts{verdict=…}` counter instead of three separate metrics.
#[derive(Clone, Debug)]
pub struct EndpointMetrics {
    /// How long this endpoint took to answer an `interop_checkAccessList` query. Lets an endpoint
    /// creeping toward the request timeout be spotted before it starts failing.
    query_latency: Histogram,
    /// Definitive valid verdicts returned by this endpoint.
    valid: Counter,
    /// Definitive invalid verdicts returned by this endpoint.
    invalid: Counter,
    /// Non-responses from this endpoint (timeout, transport error, soft out-of-sync).
    unavailable: Counter,
    /// `1` if this endpoint's last completed check produced a verdict, `0` if it was a
    /// non-response. See [`ENDPOINT_UP`].
    up: Gauge,
}

impl EndpointMetrics {
    /// Builds per-endpoint metrics labeled with the endpoint's index. Counter and histogram handles
    /// are resolved once here (not per-call) since each endpoint is built once at startup.
    pub fn for_endpoint(index: usize) -> Self {
        let endpoint = index.to_string();
        let up = gauge!(ENDPOINT_UP, "endpoint" => endpoint.clone());
        // Assume healthy until a check proves otherwise, so `sum(endpoint.up)` reads at full quorum
        // headroom from boot instead of the series being absent until the first failure.
        up.set(1.0);
        Self {
            query_latency: histogram!(ENDPOINT_QUERY_LATENCY, "endpoint" => endpoint.clone()),
            valid: counter!(ENDPOINT_VERDICTS, "endpoint" => endpoint.clone(), "verdict" => VERDICT_VALID),
            invalid: counter!(ENDPOINT_VERDICTS, "endpoint" => endpoint.clone(), "verdict" => VERDICT_INVALID),
            unavailable: counter!(ENDPOINT_VERDICTS, "endpoint" => endpoint, "verdict" => VERDICT_UNAVAILABLE),
            up,
        }
    }

    /// Records the duration of this endpoint's interop filter query.
    #[inline]
    pub fn record_query(&self, duration: Duration) {
        self.query_latency.record(duration.as_secs_f64());
    }

    /// Records this endpoint's definitive valid verdict (a valid verdict still means it responded).
    #[inline]
    pub fn record_valid(&self) {
        self.valid.increment(1);
        self.up.set(1.0);
    }

    /// Records this endpoint's definitive invalid verdict (an invalid verdict still means it
    /// responded).
    #[inline]
    pub fn record_invalid(&self) {
        self.invalid.increment(1);
        self.up.set(1.0);
    }

    /// Records this endpoint's non-response (timeout, transport error, soft out-of-sync).
    #[inline]
    pub fn record_unavailable(&self) {
        self.unavailable.increment(1);
        self.up.set(0.0);
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

    /// Registers HELP descriptions for the hand-rolled metrics and pre-creates every decision
    /// series at `0`. Called once at startup.
    ///
    /// Both are needed because `counter!`/`gauge!`/`histogram!` (unlike `#[derive(Metrics)]`) carry
    /// no description and create a series only on first use — so an outcome that has never happened
    /// would otherwise be *absent*, making `rate(filter_decisions{result="rejected_no_quorum"})`
    /// return empty instead of `0` and silently breaking fail-closed alerting.
    pub fn init(&self) {
        describe_metrics();
        for (result, reason) in canonical_decisions() {
            counter!(FILTER_DECISIONS, "result" => result, "reason" => reason).increment(0);
        }
    }

    /// Records a single interop filter decision under the given `result` and `reason` labels.
    ///
    /// Both must be one of the `RESULT_*` / `REASON_*` constants (or a DA reason from
    /// `validation_reason`) so the label set stays low-cardinality and stable. Use
    /// `decision_for_error` to derive the pair from a validation error.
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

/// Decision pairs that are not sub-classified by a DA reason. Combined with one
/// `(rejected_invalid, <reason>)` pair per [`CLASSIFIED_DA_REASONS`] entry, this is every series
/// the filter can emit — the set [`InteropMetrics::init`] pre-creates at `0`. Kept beside
/// [`decision_for_error`] so the two stay in lockstep.
const STRUCTURAL_DECISIONS: &[(&str, &str)] = &[
    (RESULT_ALLOWED, REASON_NONE),
    (RESULT_REJECTED_PRE_INTEROP, REASON_NONE),
    (RESULT_REJECTED_FAILSAFE, REASON_NONE),
    (RESULT_REJECTED_DISAGREEMENT, REASON_NONE),
    (RESULT_REJECTED_NO_QUORUM, REASON_NONE),
    (RESULT_ERRORED, REASON_TIMEOUT),
    (RESULT_ERRORED, REASON_OTHER),
    // A definitive rejection that carries no recognized DA reason (e.g. a generic `-32602`).
    (RESULT_REJECTED_INVALID, REASON_OTHER),
];

/// Every DA error variant [`validation_reason`] maps to a dedicated `reason` label. Drives the
/// `rejected_invalid` zero-init through `validation_reason` so the reason strings have a single
/// source of truth. A new, unmapped DA variant falls through to [`REASON_OTHER`] (already covered
/// by [`STRUCTURAL_DECISIONS`]), so it is never left uncounted.
const CLASSIFIED_DA_REASONS: &[SuperchainDAError] = &[
    SuperchainDAError::SkippedData,
    SuperchainDAError::UnknownChain,
    SuperchainDAError::ConflictingData,
    SuperchainDAError::IneffectiveData,
    SuperchainDAError::OutOfOrder,
    SuperchainDAError::AwaitingReplacement,
    SuperchainDAError::OutOfScope,
    SuperchainDAError::NoParentForFirstBlock,
    SuperchainDAError::FutureData,
    SuperchainDAError::MissedData,
    SuperchainDAError::DataCorruption,
];

/// Every `(result, reason)` series the filter can emit — the single source of truth for the
/// zero-init in [`InteropMetrics::init`].
fn canonical_decisions() -> impl Iterator<Item = (&'static str, &'static str)> {
    STRUCTURAL_DECISIONS.iter().copied().chain(
        CLASSIFIED_DA_REASONS
            .iter()
            .map(|reason| (RESULT_REJECTED_INVALID, validation_reason(reason))),
    )
}

/// Registers HELP text for the metrics created via the `counter!`/`gauge!`/`histogram!` macros (the
/// `#[derive(Metrics)]` fields get theirs from doc comments). Idempotent; called from
/// [`InteropMetrics::init`].
fn describe_metrics() {
    describe_counter!(
        FILTER_DECISIONS,
        "Interop filter decisions, one per interop tx reaching the filter (labels: result, reason)"
    );
    describe_counter!(
        ENDPOINT_VERDICTS,
        "Per-endpoint interop verdicts (labels: endpoint, verdict)"
    );
    describe_histogram!(
        ENDPOINT_QUERY_LATENCY,
        Unit::Seconds,
        "Per-endpoint interop_checkAccessList query latency"
    );
    describe_gauge!(
        ENDPOINT_UP,
        "1 if an interop endpoint answered its last completed check, else 0; sum vs \
         quorum_min_responses is the fail-closed headroom (labels: endpoint)"
    );
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

    /// Guards [`InteropMetrics::init`] against drift: every decision the filter can emit must be in
    /// the pre-created set, or its series goes absent until the first occurrence (breaking
    /// `rate()`-based alerts on never-yet-seen outcomes).
    #[test]
    fn canonical_decisions_covers_every_emitted_pair() {
        let canonical: std::collections::HashSet<_> = canonical_decisions().collect();

        // Pre-filter gates, recorded directly in `is_valid_cross_tx` (not via
        // `decision_for_error`).
        for pair in [
            (RESULT_ALLOWED, REASON_NONE),
            (RESULT_REJECTED_PRE_INTEROP, REASON_NONE),
            (RESULT_REJECTED_FAILSAFE, REASON_NONE),
        ] {
            assert!(canonical.contains(&pair), "gate decision {pair:?} not pre-created");
        }

        // Every non-`InvalidEntry` outcome `decision_for_error` can return.
        let errors = [
            InteropTxValidatorError::Rejected { code: -32602, message: String::new() },
            InteropTxValidatorError::Disagreement { valid: 1, invalid: 1 },
            InteropTxValidatorError::QuorumNotReached { received: 1, required: 2 },
            InteropTxValidatorError::FailsafeEnabled,
            InteropTxValidatorError::Timeout(1),
            InteropTxValidatorError::DataUnavailable { code: -1 },
            server_error(),
        ];
        for err in &errors {
            let pair = decision_for_error(err);
            assert!(canonical.contains(&pair), "decision {pair:?} not pre-created");
        }

        // Every classified DA reason, routed through `decision_for_error` like production does.
        for reason in CLASSIFIED_DA_REASONS {
            let pair = decision_for_error(&InteropTxValidatorError::InvalidEntry(*reason));
            assert!(canonical.contains(&pair), "invalid decision {pair:?} not pre-created");
        }
    }
}

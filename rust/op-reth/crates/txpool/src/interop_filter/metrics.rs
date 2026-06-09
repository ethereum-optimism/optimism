//! Optimism interop txpool metrics.

use crate::interop_filter::InteropTxValidatorError;
use op_alloy_rpc_types::SuperchainDAError;
use reth_metrics::{
    Metrics,
    metrics::{Counter, Gauge, Histogram, Label},
};
use std::time::Duration;

/// Optimism interop txpool metrics.
#[derive(Metrics, Clone)]
#[metrics(scope = "optimism_transaction_pool.interop")]
pub struct InteropMetrics {
    /// How long it takes to query the interop filter in the Optimism transaction pool.
    pub(crate) interop_query_latency: Histogram,

    /// Counter for the number of times data was skipped
    pub(crate) skipped_data_count: Counter,
    /// Counter for the number of times an unknown chain was encountered
    pub(crate) unknown_chain_count: Counter,
    /// Counter for the number of times conflicting data was encountered
    pub(crate) conflicting_data_count: Counter,
    /// Counter for the number of times ineffective data was encountered
    pub(crate) ineffective_data_count: Counter,
    /// Counter for the number of times data was out of order
    pub(crate) out_of_order_count: Counter,
    /// Counter for the number of times data was awaiting replacement
    pub(crate) awaiting_replacement_count: Counter,
    /// Counter for the number of times data was out of scope
    pub(crate) out_of_scope_count: Counter,
    /// Counter for the number of times there was no parent for the first block
    pub(crate) no_parent_for_first_block_count: Counter,
    /// Counter for the number of times future data was encountered
    pub(crate) future_data_count: Counter,
    /// Counter for the number of times data was missed
    pub(crate) missed_data_count: Counter,
    /// Counter for the number of times data corruption was encountered
    pub(crate) data_corruption_count: Counter,

    /// Current interop failsafe state: `1` if failsafe is enabled (all interop txs are
    /// rejected/evicted), `0` if disabled. Refreshed on every failsafe poll.
    pub(crate) failsafe_enabled: Gauge,

    /// Quorum accepted: enough definitive verdicts collected and all agreed valid.
    pub(crate) quorum_accept: Counter,
    /// Quorum rejected: enough definitive verdicts collected and all agreed invalid.
    pub(crate) quorum_reject_agreed: Counter,
    /// Quorum rejected: collected verdicts disagreed (mix of valid and invalid).
    pub(crate) quorum_reject_disagreement: Counter,
    /// Quorum rejected: fewer definitive verdicts than required were collected (fail closed).
    pub(crate) quorum_reject_not_reached: Counter,
    /// Quorum rejected: an endpoint reported its failsafe active (hard short-circuit rejection).
    pub(crate) quorum_reject_failsafe: Counter,
}

/// Per-endpoint interop metrics, labeled by endpoint index so each configured interop filter can be
/// observed independently. With the fleet-wide counters alone you can tell *an* endpoint is slow or
/// down; these labeled metrics answer *which* one — the first question on call for a fan-out
/// client. Labeled by index (not the raw URL), since interop-http URLs can carry basic-auth
/// credentials.
#[derive(Metrics, Clone)]
#[metrics(scope = "optimism_transaction_pool.interop.endpoint")]
pub struct EndpointMetrics {
    /// How long this endpoint took to answer an `interop_checkAccessList` query. Lets an endpoint
    /// creeping toward the request timeout be spotted before it starts failing.
    pub(crate) query_latency: Histogram,
    /// Definitive valid verdicts returned by this endpoint.
    pub(crate) valid: Counter,
    /// Definitive invalid verdicts returned by this endpoint.
    pub(crate) invalid: Counter,
    /// Non-responses from this endpoint (timeout, transport error, soft out-of-sync,
    /// cancellation).
    pub(crate) unavailable: Counter,
}

impl EndpointMetrics {
    /// Builds per-endpoint metrics labeled with the endpoint's index.
    pub fn for_endpoint(index: usize) -> Self {
        Self::new_with_labels(vec![Label::new("endpoint", index.to_string())])
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

    /// Records the outcome of a multi-endpoint quorum decision. Increments exactly one of the
    /// `quorum_*` counters based on the collected verdicts. This is the sole signal for the
    /// [`Disagreement`](InteropTxValidatorError::Disagreement) and
    /// [`QuorumNotReached`](InteropTxValidatorError::QuorumNotReached) outcomes, which
    /// [`increment_metrics_for_error`](Self::increment_metrics_for_error) does not record.
    pub fn record_quorum_outcome(&self, valid: usize, invalid: usize, min_responses: usize) {
        if valid + invalid < min_responses {
            self.quorum_reject_not_reached.increment(1);
        } else if invalid == 0 {
            self.quorum_accept.increment(1);
        } else if valid == 0 {
            self.quorum_reject_agreed.increment(1);
        } else {
            self.quorum_reject_disagreement.increment(1);
        }
    }

    /// Records a hard failsafe rejection: an endpoint reported its failsafe active and the check
    /// was short-circuited.
    #[inline]
    pub fn record_failsafe_reject(&self) {
        self.quorum_reject_failsafe.increment(1);
    }

    /// Increments the metrics for the given error
    pub fn increment_metrics_for_error(&self, error: &InteropTxValidatorError) {
        if let InteropTxValidatorError::InvalidEntry(inner) = error {
            match inner {
                SuperchainDAError::SkippedData => self.skipped_data_count.increment(1),
                SuperchainDAError::UnknownChain => self.unknown_chain_count.increment(1),
                SuperchainDAError::ConflictingData => self.conflicting_data_count.increment(1),
                SuperchainDAError::IneffectiveData => self.ineffective_data_count.increment(1),
                SuperchainDAError::OutOfOrder => self.out_of_order_count.increment(1),
                SuperchainDAError::AwaitingReplacement => {
                    self.awaiting_replacement_count.increment(1)
                }
                SuperchainDAError::OutOfScope => self.out_of_scope_count.increment(1),
                SuperchainDAError::NoParentForFirstBlock => {
                    self.no_parent_for_first_block_count.increment(1)
                }
                SuperchainDAError::FutureData => self.future_data_count.increment(1),
                SuperchainDAError::MissedData => self.missed_data_count.increment(1),
                SuperchainDAError::DataCorruption => self.data_corruption_count.increment(1),
                _ => {}
            }
        }
    }
}

//! Optimism supervisor metrics

use crate::supervisor::InteropTxValidatorError;
use op_alloy_rpc_types::SuperchainDAError;
use reth_metrics::{
    Metrics,
    metrics::{Counter, Histogram},
};
use std::time::Duration;

/// Optimism supervisor metrics
#[derive(Metrics, Clone)]
#[metrics(scope = "optimism_transaction_pool.supervisor")]
pub struct SupervisorMetrics {
    /// How long it takes to query the supervisor in the Optimism transaction pool
    pub(crate) supervisor_query_latency: Histogram,

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
}

impl SupervisorMetrics {
    /// Records the duration of supervisor queries
    #[inline]
    pub fn record_supervisor_query(&self, duration: Duration) {
        self.supervisor_query_latency.record(duration.as_secs_f64());
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

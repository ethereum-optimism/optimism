//! Metrics for the persistence service.

use crate::api::WriteCounts;
use metrics::{Counter, Histogram};
use reth_metrics::Metrics;

/// Metrics tracking items written to persistent storage.
#[derive(Metrics, Clone)]
#[metrics(scope = "optimism_trie.engine.persistence")]
pub struct PersistenceMetrics {
    /// Time spent opening a read-write transaction in seconds
    pub open_tx_duration_seconds: Histogram,
    /// Time spent writing trie updates to storage in seconds
    pub write_duration_seconds: Histogram,
    /// Time spent pruning old state in seconds
    pub prune_duration_seconds: Histogram,
    /// Time spent committing the transaction in seconds
    pub commit_duration_seconds: Histogram,
    /// Number of trie updates written
    pub account_trie_updates_written_total: Counter,
    /// Number of storage trie updates written
    pub storage_trie_updates_written_total: Counter,
    /// Number of hashed accounts written
    pub hashed_accounts_written_total: Counter,
    /// Number of hashed storages written
    pub hashed_storages_written_total: Counter,
}

impl PersistenceMetrics {
    /// Increment write counts of historical trie updates for a single block.
    pub fn increment_write_counts(&self, counts: &WriteCounts) {
        self.account_trie_updates_written_total
            .increment(counts.account_trie_updates_written_total);
        self.storage_trie_updates_written_total
            .increment(counts.storage_trie_updates_written_total);
        self.hashed_accounts_written_total.increment(counts.hashed_accounts_written_total);
        self.hashed_storages_written_total.increment(counts.hashed_storages_written_total);
    }

    /// Record timing metrics for a persistence operation.
    pub fn record_metrics(
        &self,
        write_counts: &WriteCounts,
        open_tx_duration: std::time::Duration,
        write_duration: std::time::Duration,
        prune_duration: std::time::Duration,
        commit_duration: std::time::Duration,
    ) {
        self.increment_write_counts(write_counts);
        self.open_tx_duration_seconds.record(open_tx_duration);
        self.write_duration_seconds.record(write_duration);
        self.prune_duration_seconds.record(prune_duration);
        self.commit_duration_seconds.record(commit_duration);
    }
}

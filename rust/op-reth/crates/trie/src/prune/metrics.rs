// Counters are cast into `f64` space for the metrics sink; loss past 2^53 is not
// a concern. The `PrunerOutput` stat blob is a tiny Copy struct, so the
// `needless_pass_by_value` lint doesn't reflect a real cost here.
#![allow(
    clippy::cast_precision_loss,
    clippy::cast_possible_truncation,
    clippy::needless_pass_by_value
)]

use crate::PrunerOutput;
use reth_metrics::{
    Metrics,
    metrics::{Gauge, Histogram},
};

#[derive(Metrics)]
#[metrics(scope = "optimism_trie.pruner")]
pub(super) struct Metrics {
    /// Pruning duration
    pub(crate) total_duration_seconds: Histogram,
    /// Number of pruned blocks
    pub(crate) pruned_blocks: Gauge,
    /// Number of account trie updates written in the prune run
    pub(crate) account_trie_updates_written: Gauge,
    /// Number of storage trie updates written in the prune run
    pub(crate) storage_trie_updates_written: Gauge,
    /// Number of hashed accounts written in the prune run
    pub(crate) hashed_accounts_written: Gauge,
    /// Number of hashed storages written in the prune run
    pub(crate) hashed_storages_written: Gauge,
}

impl Metrics {
    pub(crate) fn record_prune_result(&self, result: PrunerOutput) {
        let blocks_pruned = result.end_block - result.start_block;
        if blocks_pruned > 0 {
            self.total_duration_seconds.record(result.duration.as_secs_f64());
            self.pruned_blocks.set(blocks_pruned as f64);

            // Consume write counts
            let wc = &result.write_counts;
            self.account_trie_updates_written.set(wc.account_trie_updates_written_total as f64);
            self.storage_trie_updates_written.set(wc.storage_trie_updates_written_total as f64);
            self.hashed_accounts_written.set(wc.hashed_accounts_written_total as f64);
            self.hashed_storages_written.set(wc.hashed_storages_written_total as f64);
        }
    }
}

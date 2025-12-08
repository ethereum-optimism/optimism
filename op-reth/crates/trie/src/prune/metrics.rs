use crate::PrunerOutput;
use reth_metrics::{
    metrics::{Gauge, Histogram},
    Metrics,
};

#[derive(Metrics)]
#[metrics(scope = "optimism_trie.pruner")]
pub(crate) struct Metrics {
    /// Pruning duration
    pub(crate) total_duration_seconds: Histogram,
    /// Duration spent fetching state diffs (non-blocking)
    pub(crate) state_diff_fetch_duration_seconds: Histogram,
    /// Duration spent pruning
    pub(crate) prune_duration_seconds: Histogram,
    /// Number of pruned blocks
    pub(crate) pruned_blocks: Gauge,
}

impl Metrics {
    pub(crate) fn record_prune_result(&self, result: PrunerOutput) {
        let blocks_pruned = result.end_block - result.start_block;
        if blocks_pruned > 0 {
            self.total_duration_seconds.record(result.duration.as_secs_f64());
            self.state_diff_fetch_duration_seconds.record(result.fetch_duration.as_secs_f64());
            self.prune_duration_seconds.record(result.prune_duration.as_secs_f64());
            self.pruned_blocks.set(blocks_pruned as f64);
        }
    }
}

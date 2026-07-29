//! Metrics for the live trie engine.

use metrics::Histogram;
use reth_metrics::Metrics;

/// High-level engine metrics.
#[derive(Metrics, Clone)]
#[metrics(scope = "optimism_trie.engine")]
pub(super) struct EngineMetrics {
    /// Time to execute a block end-to-end (EVM + state root) in seconds.
    pub execute_block_duration_seconds: Histogram,
    /// Time spent executing the block (EVM only) in seconds.
    pub execution_duration_seconds: Histogram,
    /// Time spent calculating the state root in seconds.
    pub state_root_duration_seconds: Histogram,
    /// Time to index pre-computed trie updates for a block in seconds.
    pub index_block_duration_seconds: Histogram,
    /// Time to handle a reorg (unwind + re-index) in seconds.
    pub reorg_duration_seconds: Histogram,
    /// Time spent unwinding persistence and memory in seconds.
    pub unwind_duration_seconds: Histogram,
}

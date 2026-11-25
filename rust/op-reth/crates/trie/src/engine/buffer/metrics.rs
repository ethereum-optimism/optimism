//! Metrics for the trie buffer.

use metrics::Gauge;
use reth_metrics::Metrics;

/// Metrics for the in-memory trie buffer.
#[derive(Metrics, Clone)]
#[metrics(scope = "optimism_trie.engine.buffer")]
pub(super) struct BufferMetrics {
    /// Current number of blocks buffered in memory (between persistence flushes)
    pub buffer_size: Gauge,
}

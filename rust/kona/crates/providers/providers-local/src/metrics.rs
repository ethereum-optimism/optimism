//! Metrics for the local buffered provider.

/// Container for metrics.
#[derive(Debug, Clone)]
pub struct Metrics;

impl Metrics {
    /// Label key carrying the L2 chain ID, present on every metric emitted by the buffered
    /// provider.
    pub const CHAIN_ID_LABEL: &str = kona_macros::CHAIN_ID_LABEL;

    /// Identifier for the gauge that tracks buffered provider cache hits.
    pub const BUFFERED_PROVIDER_CACHE_HITS: &str = "kona_providers_local_cache_hits";

    /// Identifier for the gauge that tracks buffered provider cache misses.
    pub const BUFFERED_PROVIDER_CACHE_MISSES: &str = "kona_providers_local_cache_misses";

    /// Identifier for the gauge that tracks chain events processed.
    pub const CHAIN_EVENTS_PROCESSED: &str = "kona_providers_local_chain_events";

    /// Identifier for the gauge that tracks chain event errors.
    pub const CHAIN_EVENT_ERRORS: &str = "kona_providers_local_chain_event_errors";

    /// Identifier for the gauge that tracks blocks added to cache.
    pub const BLOCKS_ADDED: &str = "kona_providers_local_blocks_added";

    /// Identifier for the gauge that tracks active cache entries.
    pub const CACHE_ENTRIES: &str = "kona_providers_local_cache_entries";

    /// Identifier for the gauge that tracks reorg depth.
    pub const REORG_DEPTH: &str = "kona_providers_local_reorg_depth";

    /// Identifier for the gauge that tracks cache clears.
    pub const CACHE_CLEARS: &str = "kona_providers_local_cache_clears";

    /// Initializes metrics for the local buffered provider.
    ///
    /// This does two things:
    /// * Describes various metrics.
    /// * Initializes metrics to 0 so they can be queried immediately.
    #[cfg(feature = "metrics")]
    pub fn init(chain_id: u64) {
        Self::describe();
        Self::zero(chain_id);
    }

    /// Describes metrics used in [`kona_providers_local`][crate].
    #[cfg(feature = "metrics")]
    pub fn describe() {
        metrics::describe_gauge!(
            Self::BUFFERED_PROVIDER_CACHE_HITS,
            "Number of cache hits in buffered provider"
        );
        metrics::describe_gauge!(
            Self::BUFFERED_PROVIDER_CACHE_MISSES,
            "Number of cache misses in buffered provider"
        );
        metrics::describe_gauge!(Self::CHAIN_EVENTS_PROCESSED, "Number of chain events processed");
        metrics::describe_gauge!(
            Self::CHAIN_EVENT_ERRORS,
            "Number of chain event processing errors"
        );
        metrics::describe_gauge!(Self::BLOCKS_ADDED, "Number of blocks added to cache");
        metrics::describe_gauge!(Self::CACHE_ENTRIES, "Number of active entries in cache");
        metrics::describe_gauge!(Self::REORG_DEPTH, "Maximum depth of reorganization observed");
        metrics::describe_gauge!(Self::CACHE_CLEARS, "Number of times cache was cleared");
    }

    /// Initializes metrics to `0` so they can be queried immediately.
    #[cfg(feature = "metrics")]
    pub fn zero(chain_id: u64) {
        let chain_id = kona_macros::chain_id_label(chain_id);

        // Cache hit/miss metrics
        for metric in [Self::BUFFERED_PROVIDER_CACHE_HITS, Self::BUFFERED_PROVIDER_CACHE_MISSES] {
            for method in ["block_by_number", "l2_block_info", "system_config"] {
                kona_macros::set!(
                    gauge,
                    metric,
                    0,
                    "method" => method,
                    Self::CHAIN_ID_LABEL => chain_id.clone()
                );
            }
        }

        // Chain event metrics
        for metric in [Self::CHAIN_EVENTS_PROCESSED, Self::CHAIN_EVENT_ERRORS] {
            for event in ["committed", "reorged", "reverted"] {
                kona_macros::set!(
                    gauge,
                    metric,
                    0,
                    "event" => event,
                    Self::CHAIN_ID_LABEL => chain_id.clone()
                );
            }
        }

        // General metrics
        for cache in ["blocks_by_hash", "blocks_by_number"] {
            kona_macros::set!(
                gauge,
                Self::CACHE_ENTRIES,
                0,
                "cache" => cache,
                Self::CHAIN_ID_LABEL => chain_id.clone()
            );
        }
        for metric in [Self::BLOCKS_ADDED, Self::REORG_DEPTH, Self::CACHE_CLEARS] {
            kona_macros::set!(gauge, metric, 0, Self::CHAIN_ID_LABEL => chain_id.clone());
        }
    }
}

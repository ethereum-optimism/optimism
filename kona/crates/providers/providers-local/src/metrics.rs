//! Metrics for the local buffered provider.

/// Container for metrics.
#[derive(Debug, Clone)]
pub struct Metrics;

impl Metrics {
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

    /// Identifier for the gauge that tracks cache capacity.
    pub const CACHE_CAPACITY: &str = "kona_providers_local_cache_capacity";

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
    pub fn init() {
        Self::describe();
        Self::zero();
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
        metrics::describe_gauge!(Self::CACHE_CAPACITY, "Total capacity of cache");
        metrics::describe_gauge!(Self::REORG_DEPTH, "Maximum depth of reorganization observed");
        metrics::describe_gauge!(Self::CACHE_CLEARS, "Number of times cache was cleared");
    }

    /// Initializes metrics to `0` so they can be queried immediately.
    #[cfg(feature = "metrics")]
    pub fn zero() {
        // Cache hit/miss metrics
        kona_macros::set!(
            gauge,
            Self::BUFFERED_PROVIDER_CACHE_HITS,
            "method",
            "block_by_number",
            0
        );
        kona_macros::set!(gauge, Self::BUFFERED_PROVIDER_CACHE_HITS, "method", "block_by_hash", 0);
        kona_macros::set!(gauge, Self::BUFFERED_PROVIDER_CACHE_HITS, "method", "l2_block_info", 0);
        kona_macros::set!(gauge, Self::BUFFERED_PROVIDER_CACHE_HITS, "method", "system_config", 0);

        kona_macros::set!(
            gauge,
            Self::BUFFERED_PROVIDER_CACHE_MISSES,
            "method",
            "block_by_number",
            0
        );
        kona_macros::set!(
            gauge,
            Self::BUFFERED_PROVIDER_CACHE_MISSES,
            "method",
            "block_by_hash",
            0
        );
        kona_macros::set!(
            gauge,
            Self::BUFFERED_PROVIDER_CACHE_MISSES,
            "method",
            "l2_block_info",
            0
        );
        kona_macros::set!(
            gauge,
            Self::BUFFERED_PROVIDER_CACHE_MISSES,
            "method",
            "system_config",
            0
        );

        // Chain event metrics
        kona_macros::set!(gauge, Self::CHAIN_EVENTS_PROCESSED, "event", "committed", 0);
        kona_macros::set!(gauge, Self::CHAIN_EVENTS_PROCESSED, "event", "reorged", 0);
        kona_macros::set!(gauge, Self::CHAIN_EVENTS_PROCESSED, "event", "reverted", 0);

        kona_macros::set!(gauge, Self::CHAIN_EVENT_ERRORS, "event", "committed", 0);
        kona_macros::set!(gauge, Self::CHAIN_EVENT_ERRORS, "event", "reorged", 0);
        kona_macros::set!(gauge, Self::CHAIN_EVENT_ERRORS, "event", "reverted", 0);

        // General metrics
        kona_macros::set!(gauge, Self::BLOCKS_ADDED, 0);
        kona_macros::set!(gauge, Self::CACHE_ENTRIES, "cache", "blocks_by_hash", 0);
        kona_macros::set!(gauge, Self::CACHE_ENTRIES, "cache", "blocks_by_number", 0);
        kona_macros::set!(gauge, Self::CACHE_CAPACITY, 0);
        kona_macros::set!(gauge, Self::REORG_DEPTH, 0);
        kona_macros::set!(gauge, Self::CACHE_CLEARS, 0);
    }
}

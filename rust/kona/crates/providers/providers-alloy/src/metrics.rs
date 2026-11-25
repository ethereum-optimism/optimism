//! Metrics for the Alloy providers.

/// Container for metrics.
#[derive(Debug, Clone)]
pub struct Metrics;

impl Metrics {
    /// Identifier for the gauge that tracks chain provider cache hits.
    pub const CHAIN_PROVIDER_CACHE_HITS: &str = "kona_providers_chain_cache_hits";

    /// Identifier for the gauge that tracks chain provider cache misses.
    pub const CHAIN_PROVIDER_CACHE_MISSES: &str = "kona_providers_chain_cache_misses";

    /// Identifier for the gauge that tracks beacon client requests.
    pub const BEACON_CLIENT_REQUESTS: &str = "kona_providers_beacon_requests";

    /// Identifier for the gauge that tracks beacon client errors.
    pub const BEACON_CLIENT_ERRORS: &str = "kona_providers_beacon_errors";

    /// Identifier for the gauge that tracks L2 chain provider requests.
    pub const L2_CHAIN_PROVIDER_REQUESTS: &str = "kona_providers_l2_chain_requests";

    /// Identifier for the gauge that tracks L2 chain provider errors.
    pub const L2_CHAIN_PROVIDER_ERRORS: &str = "kona_providers_l2_chain_errors";

    /// Identifier for the gauge that tracks blob fetches.
    pub const BLOB_FETCHES: &str = "kona_providers_blob_fetches";

    /// Identifier for the gauge that tracks blob fetch errors.
    pub const BLOB_FETCH_ERRORS: &str = "kona_providers_blob_fetch_errors";

    /// Identifier for the gauge that tracks active cache entries.
    pub const CHAIN_PROVIDER_CACHE_ENTRIES: &str = "kona_providers_cache_entries";

    /// Initializes metrics for the Alloy providers.
    ///
    /// This does two things:
    /// * Describes various metrics.
    /// * Initializes metrics to 0 so they can be queried immediately.
    #[cfg(feature = "metrics")]
    pub fn init() {
        Self::describe();
        Self::zero();
    }

    /// Describes metrics used in [`kona_providers_alloy`][crate].
    #[cfg(feature = "metrics")]
    pub fn describe() {
        metrics::describe_gauge!(
            Self::CHAIN_PROVIDER_CACHE_HITS,
            "Number of cache hits in chain provider"
        );
        metrics::describe_gauge!(
            Self::CHAIN_PROVIDER_CACHE_MISSES,
            "Number of cache misses in chain provider"
        );
        metrics::describe_gauge!(
            Self::BEACON_CLIENT_REQUESTS,
            "Number of requests made to beacon client"
        );
        metrics::describe_gauge!(
            Self::BEACON_CLIENT_ERRORS,
            "Number of errors in beacon client requests"
        );
        metrics::describe_gauge!(
            Self::L2_CHAIN_PROVIDER_REQUESTS,
            "Number of requests made to L2 chain provider"
        );
        metrics::describe_gauge!(
            Self::L2_CHAIN_PROVIDER_ERRORS,
            "Number of errors in L2 chain provider requests"
        );
        metrics::describe_gauge!(Self::BLOB_FETCHES, "Number of blob sidecar fetches");
        metrics::describe_gauge!(Self::BLOB_FETCH_ERRORS, "Number of blob sidecar fetch errors");
        metrics::describe_gauge!(
            Self::CHAIN_PROVIDER_CACHE_ENTRIES,
            "Number of active entries in the chain provider cache"
        );
    }

    /// Initializes metrics to `0` so they can be queried immediately by consumers of prometheus
    /// metrics.
    #[cfg(feature = "metrics")]
    pub fn zero() {
        // Chain provider cache metrics
        for cache in ["headers", "receipts", "transactions"] {
            kona_macros::set!(gauge, Self::CHAIN_PROVIDER_CACHE_HITS, "cache", cache, 0);
            kona_macros::set!(gauge, Self::CHAIN_PROVIDER_CACHE_MISSES, "cache", cache, 0);
            kona_macros::set!(gauge, Self::CHAIN_PROVIDER_CACHE_ENTRIES, "cache", cache, 0);
        }

        // Beacon client metrics
        kona_macros::set!(gauge, Self::BEACON_CLIENT_REQUESTS, "method", "spec", 0);
        kona_macros::set!(gauge, Self::BEACON_CLIENT_REQUESTS, "method", "genesis", 0);
        kona_macros::set!(gauge, Self::BEACON_CLIENT_REQUESTS, "method", "blob_sidecars", 0);

        kona_macros::set!(gauge, Self::BEACON_CLIENT_ERRORS, "method", "spec", 0);
        kona_macros::set!(gauge, Self::BEACON_CLIENT_ERRORS, "method", "genesis", 0);
        kona_macros::set!(gauge, Self::BEACON_CLIENT_ERRORS, "method", "blob_sidecars", 0);

        // L2 chain provider metrics
        kona_macros::set!(
            gauge,
            Self::L2_CHAIN_PROVIDER_REQUESTS,
            "method",
            "l2_block_ref_by_label",
            0
        );
        kona_macros::set!(
            gauge,
            Self::L2_CHAIN_PROVIDER_REQUESTS,
            "method",
            "l2_block_ref_by_hash",
            0
        );
        kona_macros::set!(
            gauge,
            Self::L2_CHAIN_PROVIDER_REQUESTS,
            "method",
            "l2_block_ref_by_number",
            0
        );

        kona_macros::set!(
            gauge,
            Self::L2_CHAIN_PROVIDER_ERRORS,
            "method",
            "l2_block_ref_by_label",
            0
        );
        kona_macros::set!(
            gauge,
            Self::L2_CHAIN_PROVIDER_ERRORS,
            "method",
            "l2_block_ref_by_hash",
            0
        );
        kona_macros::set!(
            gauge,
            Self::L2_CHAIN_PROVIDER_ERRORS,
            "method",
            "l2_block_ref_by_number",
            0
        );

        // Blob sidecar metrics
        kona_macros::set!(gauge, Self::BLOB_FETCHES, 0);
        kona_macros::set!(gauge, Self::BLOB_FETCH_ERRORS, 0);
    }
}

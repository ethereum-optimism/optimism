//! Metrics for the Alloy providers.

/// Container for metrics.
#[derive(Debug, Clone)]
pub struct Metrics;

impl Metrics {
    /// Identifier for the gauge that tracks chain provider cache hits.
    pub const CHAIN_PROVIDER_CACHE_HITS: &str = "kona_providers_chain_cache_hits";

    /// Identifier for the gauge that tracks chain provider cache misses.
    pub const CHAIN_PROVIDER_CACHE_MISSES: &str = "kona_providers_chain_cache_misses";

    /// Identifier for the gauge that tracks chain provider RPC calls.
    pub const CHAIN_PROVIDER_RPC_CALLS: &str = "kona_providers_chain_rpc_calls";

    /// Identifier for the gauge that tracks chain provider RPC errors.
    pub const CHAIN_PROVIDER_RPC_ERRORS: &str = "kona_providers_chain_rpc_errors";

    /// Identifier for the gauge that tracks beacon client requests.
    pub const BEACON_CLIENT_REQUESTS: &str = "kona_providers_beacon_requests";

    /// Identifier for the gauge that tracks beacon client errors.
    pub const BEACON_CLIENT_ERRORS: &str = "kona_providers_beacon_errors";

    /// Identifier for the gauge that tracks L2 chain provider requests.
    pub const L2_CHAIN_PROVIDER_REQUESTS: &str = "kona_providers_l2_chain_requests";

    /// Identifier for the gauge that tracks L2 chain provider errors.
    pub const L2_CHAIN_PROVIDER_ERRORS: &str = "kona_providers_l2_chain_errors";

    /// Identifier for the gauge that tracks blob sidecar fetches.
    pub const BLOB_SIDECAR_FETCHES: &str = "kona_providers_blob_sidecar_fetches";

    /// Identifier for the gauge that tracks blob sidecar fetch errors.
    pub const BLOB_SIDECAR_FETCH_ERRORS: &str = "kona_providers_blob_sidecar_errors";

    /// Identifier for the histogram that tracks provider request duration.
    pub const PROVIDER_REQUEST_DURATION: &str = "kona_providers_request_duration";

    /// Identifier for the gauge that tracks active cache entries.
    pub const CACHE_ENTRIES: &str = "kona_providers_cache_entries";

    /// Identifier for the gauge that tracks cache memory usage.
    pub const CACHE_MEMORY_USAGE: &str = "kona_providers_cache_memory_bytes";

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
            Self::CHAIN_PROVIDER_RPC_CALLS,
            "Number of RPC calls made by chain provider"
        );
        metrics::describe_gauge!(
            Self::CHAIN_PROVIDER_RPC_ERRORS,
            "Number of RPC errors in chain provider"
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
        metrics::describe_gauge!(Self::BLOB_SIDECAR_FETCHES, "Number of blob sidecar fetches");
        metrics::describe_gauge!(
            Self::BLOB_SIDECAR_FETCH_ERRORS,
            "Number of blob sidecar fetch errors"
        );
        metrics::describe_histogram!(
            Self::PROVIDER_REQUEST_DURATION,
            "Duration of provider requests in seconds"
        );
        metrics::describe_gauge!(
            Self::CACHE_ENTRIES,
            "Number of active entries in provider caches"
        );
        metrics::describe_gauge!(
            Self::CACHE_MEMORY_USAGE,
            "Memory usage of provider caches in bytes"
        );
    }

    /// Initializes metrics to `0` so they can be queried immediately by consumers of prometheus
    /// metrics.
    #[cfg(feature = "metrics")]
    pub fn zero() {
        // Chain provider cache metrics
        kona_macros::set!(gauge, Self::CHAIN_PROVIDER_CACHE_HITS, "cache", "header_by_hash", 0);
        kona_macros::set!(gauge, Self::CHAIN_PROVIDER_CACHE_HITS, "cache", "receipts_by_hash", 0);
        kona_macros::set!(gauge, Self::CHAIN_PROVIDER_CACHE_HITS, "cache", "block_info_and_tx", 0);

        kona_macros::set!(gauge, Self::CHAIN_PROVIDER_CACHE_MISSES, "cache", "header_by_hash", 0);
        kona_macros::set!(gauge, Self::CHAIN_PROVIDER_CACHE_MISSES, "cache", "receipts_by_hash", 0);
        kona_macros::set!(
            gauge,
            Self::CHAIN_PROVIDER_CACHE_MISSES,
            "cache",
            "block_info_and_tx",
            0
        );

        // RPC call metrics
        kona_macros::set!(gauge, Self::CHAIN_PROVIDER_RPC_CALLS, "method", "header_by_hash", 0);
        kona_macros::set!(gauge, Self::CHAIN_PROVIDER_RPC_CALLS, "method", "receipts_by_hash", 0);
        kona_macros::set!(gauge, Self::CHAIN_PROVIDER_RPC_CALLS, "method", "block_by_hash", 0);
        kona_macros::set!(gauge, Self::CHAIN_PROVIDER_RPC_CALLS, "method", "block_number", 0);

        // RPC error metrics
        kona_macros::set!(gauge, Self::CHAIN_PROVIDER_RPC_ERRORS, "method", "header_by_hash", 0);
        kona_macros::set!(gauge, Self::CHAIN_PROVIDER_RPC_ERRORS, "method", "receipts_by_hash", 0);
        kona_macros::set!(gauge, Self::CHAIN_PROVIDER_RPC_ERRORS, "method", "block_by_hash", 0);
        kona_macros::set!(gauge, Self::CHAIN_PROVIDER_RPC_ERRORS, "method", "block_number", 0);

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
        kona_macros::set!(gauge, Self::BLOB_SIDECAR_FETCHES, 0);
        kona_macros::set!(gauge, Self::BLOB_SIDECAR_FETCH_ERRORS, 0);

        // Cache metrics
        kona_macros::set!(gauge, Self::CACHE_ENTRIES, "cache", "header_by_hash", 0);
        kona_macros::set!(gauge, Self::CACHE_ENTRIES, "cache", "receipts_by_hash", 0);
        kona_macros::set!(gauge, Self::CACHE_ENTRIES, "cache", "block_info_and_tx", 0);

        kona_macros::set!(gauge, Self::CACHE_MEMORY_USAGE, "cache", "header_by_hash", 0);
        kona_macros::set!(gauge, Self::CACHE_MEMORY_USAGE, "cache", "receipts_by_hash", 0);
        kona_macros::set!(gauge, Self::CACHE_MEMORY_USAGE, "cache", "block_info_and_tx", 0);
    }
}

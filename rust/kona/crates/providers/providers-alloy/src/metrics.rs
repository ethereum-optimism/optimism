//! Metrics for the Alloy providers.

/// Container for metrics.
#[derive(Debug, Clone)]
pub struct Metrics;

impl Metrics {
    /// Label key carrying the L2 chain ID, present on every metric emitted by the alloy
    /// providers.
    pub const CHAIN_ID_LABEL: &str = "chain_id";

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

    /// Identifier for the gauge that tracks blob fetches.
    pub const BLOB_FETCHES: &str = "kona_providers_blob_fetches";

    /// Identifier for the gauge that tracks blob fetch errors.
    pub const BLOB_FETCH_ERRORS: &str = "kona_providers_blob_fetch_errors";

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
    pub fn init(chain_id: u64) {
        Self::describe();
        Self::zero(chain_id);
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
        metrics::describe_gauge!(Self::BLOB_FETCHES, "Number of blob sidecar fetches");
        metrics::describe_gauge!(Self::BLOB_FETCH_ERRORS, "Number of blob sidecar fetch errors");
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
    pub fn zero(chain_id: u64) {
        let chain_id: std::sync::Arc<str> = std::sync::Arc::from(chain_id.to_string());

        // Chain provider cache metrics
        for metric in [Self::CHAIN_PROVIDER_CACHE_HITS, Self::CHAIN_PROVIDER_CACHE_MISSES] {
            for cache in ["header_by_hash", "receipts_by_hash", "block_info_and_tx"] {
                kona_macros::set!(
                    gauge,
                    metric,
                    0,
                    "cache" => cache,
                    Self::CHAIN_ID_LABEL => chain_id.clone()
                );
            }
        }

        // RPC call and error metrics
        for metric in [Self::CHAIN_PROVIDER_RPC_CALLS, Self::CHAIN_PROVIDER_RPC_ERRORS] {
            for method in ["header_by_hash", "receipts_by_hash", "block_by_hash", "block_number"] {
                kona_macros::set!(
                    gauge,
                    metric,
                    0,
                    "method" => method,
                    Self::CHAIN_ID_LABEL => chain_id.clone()
                );
            }
        }

        // Beacon client metrics.
        //
        // Deliberately unlabelled: `OnlineBeaconClient` is an L1 beacon-API client that the
        // interop host shares across every L2 chain it serves, so there is no single chain ID
        // to attribute these to. Keep the zeroed series identical to the runtime emits.
        for metric in [Self::BEACON_CLIENT_REQUESTS, Self::BEACON_CLIENT_ERRORS] {
            for method in ["spec", "genesis", "blob_sidecars"] {
                kona_macros::set!(gauge, metric, 0, "method" => method);
            }
        }

        // L2 chain provider metrics
        for metric in [Self::L2_CHAIN_PROVIDER_REQUESTS, Self::L2_CHAIN_PROVIDER_ERRORS] {
            for method in
                ["l2_block_ref_by_label", "l2_block_ref_by_hash", "l2_block_ref_by_number"]
            {
                kona_macros::set!(
                    gauge,
                    metric,
                    0,
                    "method" => method,
                    Self::CHAIN_ID_LABEL => chain_id.clone()
                );
            }
        }

        // Blob sidecar metrics. Unlabelled for the same reason as the beacon client metrics
        // above: `OnlineBlobProvider` wraps a beacon client that is shared across chains.
        for metric in [Self::BLOB_FETCHES, Self::BLOB_FETCH_ERRORS] {
            kona_macros::set!(gauge, metric, 0);
        }

        // Cache metrics
        for metric in [Self::CACHE_ENTRIES, Self::CACHE_MEMORY_USAGE] {
            for cache in ["header_by_hash", "receipts_by_hash", "block_info_and_tx"] {
                kona_macros::set!(
                    gauge,
                    metric,
                    0,
                    "cache" => cache,
                    Self::CHAIN_ID_LABEL => chain_id.clone()
                );
            }
        }
    }
}

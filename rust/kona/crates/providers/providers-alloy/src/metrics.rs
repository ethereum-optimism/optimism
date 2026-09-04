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

    /// Identifier for the gauge that tracks blob fetches.
    pub const BLOB_FETCHES: &str = "kona_providers_blob_fetches";

    /// Identifier for the gauge that tracks blob fetch errors.
    pub const BLOB_FETCH_ERRORS: &str = "kona_providers_blob_fetch_errors";

    /// Identifier for the gauge that tracks active cache entries.
    pub const CACHE_ENTRIES: &str = "kona_providers_cache_entries";

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
        metrics::describe_gauge!(Self::BLOB_FETCHES, "Number of blob sidecar fetches");
        metrics::describe_gauge!(Self::BLOB_FETCH_ERRORS, "Number of blob sidecar fetch errors");
        metrics::describe_gauge!(
            Self::CACHE_ENTRIES,
            "Number of active entries in provider caches"
        );
    }

    /// Initializes metrics to `0` so they can be queried immediately by consumers of prometheus
    /// metrics.
    #[cfg(feature = "metrics")]
    pub fn zero() {
        for cache in ["header_by_hash", "receipts_by_hash", "block_info_and_tx"] {
            kona_macros::set!(gauge, Self::CHAIN_PROVIDER_CACHE_HITS, "cache", cache, 0);
            kona_macros::set!(gauge, Self::CHAIN_PROVIDER_CACHE_MISSES, "cache", cache, 0);
            kona_macros::set!(gauge, Self::CACHE_ENTRIES, "cache", cache, 0);
        }

        // Per method emitted by `chain_provider.rs`.
        for method in [
            "header_by_hash",
            "receipts_by_hash",
            "block_by_hash",
            "block_by_number",
            "block_number",
        ] {
            kona_macros::set!(gauge, Self::CHAIN_PROVIDER_RPC_CALLS, "method", method, 0);
            kona_macros::set!(gauge, Self::CHAIN_PROVIDER_RPC_ERRORS, "method", method, 0);
        }

        // Per method emitted by `beacon_client.rs`.
        for method in ["spec", "genesis", "blobs"] {
            kona_macros::set!(gauge, Self::BEACON_CLIENT_REQUESTS, "method", method, 0);
            kona_macros::set!(gauge, Self::BEACON_CLIENT_ERRORS, "method", method, 0);
        }

        // Per method emitted by `l2_chain_provider.rs`.
        for method in ["l2_block_by_hash", "l2_block_ref_by_number"] {
            kona_macros::set!(gauge, Self::L2_CHAIN_PROVIDER_REQUESTS, "method", method, 0);
            kona_macros::set!(gauge, Self::L2_CHAIN_PROVIDER_ERRORS, "method", method, 0);
        }

        kona_macros::set!(gauge, Self::BLOB_FETCHES, 0);
        kona_macros::set!(gauge, Self::BLOB_FETCH_ERRORS, 0);
    }
}

#[cfg(all(test, feature = "metrics"))]
mod tests {
    use super::Metrics;
    use metrics_util::debugging::DebuggingRecorder;
    use std::collections::BTreeSet;

    fn zeroed(metric: &str, label: &str) -> BTreeSet<Option<String>> {
        let recorder = DebuggingRecorder::new();
        let snapshotter = recorder.snapshotter();
        metrics::with_local_recorder(&recorder, Metrics::zero);

        snapshotter
            .snapshot()
            .into_vec()
            .into_iter()
            .map(|(ckey, ..)| ckey)
            .filter(|ckey| ckey.key().name() == metric)
            // `map`, not `filter_map`: a label-free series must show up as `None` and break
            // equality.
            .map(|ckey| {
                ckey.key().labels().find(|l| l.key() == label).map(|l| l.value().to_string())
            })
            .collect()
    }

    fn expect(metric: &str, label: &str, values: &[&str]) {
        let expected: BTreeSet<_> = values.iter().map(|v| Some(v.to_string())).collect();
        assert_eq!(zeroed(metric, label), expected, "{metric} pre-created the wrong {label} set");
    }

    #[test]
    fn chain_provider_methods_match_the_emits() {
        let methods = [
            "header_by_hash",
            "receipts_by_hash",
            "block_by_hash",
            "block_by_number",
            "block_number",
        ];
        expect(Metrics::CHAIN_PROVIDER_RPC_CALLS, "method", &methods);
        expect(Metrics::CHAIN_PROVIDER_RPC_ERRORS, "method", &methods);
    }

    #[test]
    fn beacon_client_methods_match_the_emits() {
        expect(Metrics::BEACON_CLIENT_REQUESTS, "method", &["spec", "genesis", "blobs"]);
        expect(Metrics::BEACON_CLIENT_ERRORS, "method", &["spec", "genesis", "blobs"]);
    }

    #[test]
    fn l2_chain_provider_methods_match_the_emits() {
        let methods = ["l2_block_by_hash", "l2_block_ref_by_number"];
        expect(Metrics::L2_CHAIN_PROVIDER_REQUESTS, "method", &methods);
        expect(Metrics::L2_CHAIN_PROVIDER_ERRORS, "method", &methods);
    }

    #[test]
    fn no_series_is_pre_created_without_an_emit() {
        assert!(zeroed("kona_providers_cache_memory_bytes", "cache").is_empty());
    }
}

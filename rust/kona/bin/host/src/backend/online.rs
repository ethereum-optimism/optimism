//! Contains the [`OnlineHostBackend`] definition.

use crate::SharedKeyValueStore;
use anyhow::Result;
use async_trait::async_trait;
use kona_preimage::{
    HintRouter, PreimageFetcher, PreimageKey,
    errors::{PreimageOracleError, PreimageOracleResult},
};
use kona_proof::{Hint, errors::HintParsingError};
use std::{collections::HashSet, hash::Hash, str::FromStr, sync::Arc};
use tokio::sync::RwLock;
use tracing::{debug, error, trace};

/// The [`OnlineHostBackendCfg`] trait is used to define the type configuration for the
/// [`OnlineHostBackend`].
pub trait OnlineHostBackendCfg {
    /// The hint type describing the range of hints that can be received.
    type HintType: FromStr<Err = HintParsingError> + Hash + Eq + PartialEq + Clone + Send + Sync;

    /// The providers that are used to fetch data in response to hints.
    type Providers: Send + Sync;
}

/// A [`HintHandler`] is an interface for receiving hints, fetching remote data, and storing it in
/// the key-value store.
#[async_trait]
pub trait HintHandler {
    /// The type configuration for the [`HintHandler`].
    type Cfg: OnlineHostBackendCfg;

    /// Fetches data in response to a hint.
    async fn fetch_hint(
        hint: Hint<<Self::Cfg as OnlineHostBackendCfg>::HintType>,
        cfg: &Self::Cfg,
        providers: &<Self::Cfg as OnlineHostBackendCfg>::Providers,
        kv: SharedKeyValueStore,
    ) -> Result<()>;
}

/// The [`OnlineHostBackend`] is a [`HintRouter`] and [`PreimageFetcher`] that is used to fetch data
/// from remote sources in response to hints.
///
/// [PreimageKey]: kona_preimage::PreimageKey
#[allow(missing_debug_implementations)]
pub struct OnlineHostBackend<C, H>
where
    C: OnlineHostBackendCfg,
    H: HintHandler,
{
    /// The configuration that is used to route hints.
    cfg: C,
    /// The key-value store that is used to store preimages.
    kv: SharedKeyValueStore,
    /// The providers that are used to fetch data in response to hints.
    providers: C::Providers,
    /// Hint types treated as "high-level" (see [`Self::last_high_level_hint`]).
    high_level_hint_types: HashSet<C::HintType>,
    /// The latest high-level hint. High-level hints bulk-populate the key-value store, so rather
    /// than being fetched once on receipt the latest is retained and tried before
    /// [`Self::last_hint`] on every [`Self::get_preimage`] attempt: cleared once it succeeds (its
    /// response is deterministic) and kept while it fails, so a transient error is retried
    /// indefinitely instead of falling back to an unavailable `debug_dbGet`.
    last_high_level_hint: Arc<RwLock<Option<Hint<C::HintType>>>>,
    /// The latest fine-grained hint.
    last_hint: Arc<RwLock<Option<Hint<C::HintType>>>>,
    /// Phantom marker for the [`HintHandler`].
    _hint_handler: std::marker::PhantomData<H>,
}

impl<C, H> OnlineHostBackend<C, H>
where
    C: OnlineHostBackendCfg,
    H: HintHandler,
{
    /// Creates a new [`HintHandler`] with the given configuration, key-value store, providers, and
    /// external configuration.
    pub fn new(cfg: C, kv: SharedKeyValueStore, providers: C::Providers, _: H) -> Self {
        Self {
            cfg,
            kv,
            providers,
            high_level_hint_types: HashSet::default(),
            last_high_level_hint: Arc::new(RwLock::new(None)),
            last_hint: Arc::new(RwLock::new(None)),
            _hint_handler: std::marker::PhantomData,
        }
    }

    /// Registers a hint type as "high-level": rather than being fetched once and discarded, the
    /// latest hint of this type is retained and tried first by the `get_preimage` retry loop.
    pub fn with_high_level_hint(mut self, hint_type: C::HintType) -> Self {
        self.high_level_hint_types.insert(hint_type);
        self
    }
}

#[async_trait]
impl<C, H> HintRouter for OnlineHostBackend<C, H>
where
    C: OnlineHostBackendCfg + Send + Sync,
    H: HintHandler<Cfg = C> + Send + Sync,
{
    /// Set the last hint to be received.
    async fn route_hint(&self, hint: String) -> PreimageOracleResult<()> {
        trace!(target: "host_backend", "Received hint: {hint}");

        let parsed_hint = hint
            .parse::<Hint<C::HintType>>()
            .map_err(|e| PreimageOracleError::HintParseFailed(e.to_string()))?;
        if self.high_level_hint_types.contains(&parsed_hint.ty) {
            debug!(target: "host_backend", "High-level hint received; retaining {hint}");
            self.last_high_level_hint.write().await.replace(parsed_hint);
        } else {
            self.last_hint.write().await.replace(parsed_hint);
        }

        Ok(())
    }
}

#[async_trait]
impl<C, H> PreimageFetcher for OnlineHostBackend<C, H>
where
    C: OnlineHostBackendCfg + Send + Sync,
    H: HintHandler<Cfg = C> + Send + Sync,
{
    /// Get the preimage for the given key.
    async fn get_preimage(&self, key: PreimageKey) -> PreimageOracleResult<Vec<u8>> {
        trace!(target: "host_backend", "Pre-image requested. Key: {key}");

        // Acquire a read lock on the key-value store.
        let kv_lock = self.kv.read().await;
        let mut preimage = kv_lock.get(key.into());

        // Drop the read lock before beginning the retry loop.
        drop(kv_lock);

        // Use a loop to keep retrying the prefetch as long as the key is not found
        while preimage.is_none() {
            // Try the retained high-level hint first (see `last_high_level_hint`).
            let high_level_hint = self.last_high_level_hint.read().await.clone();
            if let Some(hint) = high_level_hint {
                match H::fetch_hint(hint, &self.cfg, &self.providers, self.kv.clone()).await {
                    Ok(()) => {
                        // Clearing unconditionally is safe: the client blocks on this request and
                        // issues hints and preimage requests sequentially, so no newer high-level
                        // hint can arrive mid-fetch.
                        self.last_high_level_hint.write().await.take();
                        preimage = self.kv.read().await.get(key.into());
                        if preimage.is_some() {
                            break;
                        }
                    }
                    Err(e) => {
                        error!(target: "host_backend", "Failed to prefetch high-level hint: {e}");
                    }
                }
            }

            // Fall back to the fine-grained hint.
            if let Some(hint) = self.last_hint.read().await.clone() {
                if let Err(e) =
                    H::fetch_hint(hint, &self.cfg, &self.providers, self.kv.clone()).await
                {
                    error!(target: "host_backend", "Failed to prefetch hint: {e}");
                    continue;
                }

                preimage = self.kv.read().await.get(key.into());
            }
        }

        preimage.ok_or(PreimageOracleError::KeyNotFound)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::kv::MemoryKeyValueStore;
    use alloy_primitives::B256;
    use kona_preimage::PreimageKey;
    use std::sync::atomic::{AtomicUsize, Ordering};

    #[derive(Clone, PartialEq, Eq, Hash, Debug)]
    enum TestHint {
        HighLevel,
        LowLevel,
    }

    impl FromStr for TestHint {
        type Err = HintParsingError;

        fn from_str(s: &str) -> Result<Self, Self::Err> {
            match s {
                "high" => Ok(Self::HighLevel),
                "low" => Ok(Self::LowLevel),
                other => Err(HintParsingError(format!("unknown test hint: {other}"))),
            }
        }
    }

    struct TestCfg;

    impl OnlineHostBackendCfg for TestCfg {
        type HintType = TestHint;
        type Providers = TestProviders;
    }

    /// Configures the mock [`HintHandler`]. The attempt counters are behind `Arc`s so a test can
    /// inspect them after `providers` has been moved into the backend.
    #[derive(Clone)]
    struct TestProviders {
        /// The keccak key both hints populate; equal to what `get_preimage` looks up.
        target: B256,
        /// The value stored under `target`.
        value: Vec<u8>,
        /// Number of times the high-level fetch fails before it succeeds.
        high_level_fail_until: usize,
        /// Whether the high-level fetch stores `target` once it succeeds (false models a
        /// method-not-found, where the witness call returns `Ok` without populating anything).
        high_level_stores_target: bool,
        high_level_attempts: Arc<AtomicUsize>,
        /// Whether the fine-grained fetch stores `target`.
        low_level_stores_target: bool,
        low_level_attempts: Arc<AtomicUsize>,
    }

    struct TestHintHandler;

    #[async_trait]
    impl HintHandler for TestHintHandler {
        type Cfg = TestCfg;

        async fn fetch_hint(
            hint: Hint<TestHint>,
            _cfg: &TestCfg,
            providers: &TestProviders,
            kv: SharedKeyValueStore,
        ) -> Result<()> {
            match hint.ty {
                TestHint::HighLevel => {
                    let attempt = providers.high_level_attempts.fetch_add(1, Ordering::SeqCst);
                    if attempt < providers.high_level_fail_until {
                        anyhow::bail!("transient high-level failure");
                    }
                    if providers.high_level_stores_target {
                        kv.write().await.set(providers.target, providers.value.clone())?;
                    }
                    Ok(())
                }
                TestHint::LowLevel => {
                    providers.low_level_attempts.fetch_add(1, Ordering::SeqCst);
                    if providers.low_level_stores_target {
                        kv.write().await.set(providers.target, providers.value.clone())?;
                    }
                    Ok(())
                }
            }
        }
    }

    /// A keccak preimage key and the `B256` it maps to in the key-value store.
    fn target_key() -> (PreimageKey, B256) {
        let key = PreimageKey::new_keccak256(*B256::repeat_byte(0x11));
        (key, key.into())
    }

    fn new_backend(providers: TestProviders) -> OnlineHostBackend<TestCfg, TestHintHandler> {
        let kv: SharedKeyValueStore = Arc::new(RwLock::new(MemoryKeyValueStore::new()));
        OnlineHostBackend::new(TestCfg, kv, providers, TestHintHandler)
            .with_high_level_hint(TestHint::HighLevel)
    }

    /// The core regression: a transient high-level (witness) fetch failure is retried by the
    /// `get_preimage` loop until it succeeds, rather than being surfaced as a permanent error that
    /// forces a fall-through to the unsupported `debug_dbGet`.
    #[tokio::test]
    async fn high_level_hint_retried_until_success_then_cleared() {
        let (key, target) = target_key();
        let high_level_attempts = Arc::new(AtomicUsize::new(0));
        let backend = new_backend(TestProviders {
            target,
            value: b"witness".to_vec(),
            high_level_fail_until: 2,
            high_level_stores_target: true,
            high_level_attempts: high_level_attempts.clone(),
            low_level_stores_target: false,
            low_level_attempts: Arc::new(AtomicUsize::new(0)),
        });
        backend.route_hint("high 00".to_string()).await.unwrap();

        let preimage = backend.get_preimage(key).await.unwrap();

        assert_eq!(preimage, b"witness".to_vec());
        assert_eq!(
            high_level_attempts.load(Ordering::SeqCst),
            3,
            "should fail twice then succeed on the third attempt"
        );
        assert!(
            backend.last_high_level_hint.read().await.is_none(),
            "high-level hint should be cleared once it succeeds"
        );
    }

    /// The high-level hint is tried before the fine-grained hint, so when it satisfies the key the
    /// fine-grained hint is never fetched.
    #[tokio::test]
    async fn high_level_hint_tried_before_fine_grained() {
        let (key, target) = target_key();
        let high_level_attempts = Arc::new(AtomicUsize::new(0));
        let low_level_attempts = Arc::new(AtomicUsize::new(0));
        let backend = new_backend(TestProviders {
            target,
            value: b"witness".to_vec(),
            high_level_fail_until: 0,
            high_level_stores_target: true,
            high_level_attempts: high_level_attempts.clone(),
            low_level_stores_target: true,
            low_level_attempts: low_level_attempts.clone(),
        });
        backend.route_hint("high 00".to_string()).await.unwrap();
        backend.route_hint("low 00".to_string()).await.unwrap();

        let preimage = backend.get_preimage(key).await.unwrap();

        assert_eq!(preimage, b"witness".to_vec());
        assert_eq!(high_level_attempts.load(Ordering::SeqCst), 1);
        assert_eq!(
            low_level_attempts.load(Ordering::SeqCst),
            0,
            "fine-grained hint should not run once the high-level hint satisfies the key"
        );
    }

    /// When the high-level hint succeeds but doesn't populate this key (a node outside the
    /// witness), it is cleared and the loop falls through to the fine-grained `debug_dbGet` hint.
    #[tokio::test]
    async fn falls_back_to_fine_grained_when_high_level_does_not_populate() {
        let (key, target) = target_key();
        let high_level_attempts = Arc::new(AtomicUsize::new(0));
        let low_level_attempts = Arc::new(AtomicUsize::new(0));
        let backend = new_backend(TestProviders {
            target,
            value: b"node".to_vec(),
            high_level_fail_until: 0,
            high_level_stores_target: false,
            high_level_attempts: high_level_attempts.clone(),
            low_level_stores_target: true,
            low_level_attempts: low_level_attempts.clone(),
        });
        backend.route_hint("high 00".to_string()).await.unwrap();
        backend.route_hint("low 00".to_string()).await.unwrap();

        let preimage = backend.get_preimage(key).await.unwrap();

        assert_eq!(preimage, b"node".to_vec());
        assert_eq!(
            high_level_attempts.load(Ordering::SeqCst),
            1,
            "high-level hint should be cleared after one non-populating success"
        );
        assert!(low_level_attempts.load(Ordering::SeqCst) >= 1);
        assert!(backend.last_high_level_hint.read().await.is_none());
    }

    /// A high-level hint that keeps failing (e.g. an unimplemented `debug_executePayload`) is
    /// retried and kept, but never blocks the fine-grained fallback that still serves the key.
    #[tokio::test]
    async fn fine_grained_fallback_survives_failing_high_level_hint() {
        let (key, target) = target_key();
        let high_level_attempts = Arc::new(AtomicUsize::new(0));
        let low_level_attempts = Arc::new(AtomicUsize::new(0));
        let backend = new_backend(TestProviders {
            target,
            value: b"node".to_vec(),
            high_level_fail_until: usize::MAX,
            high_level_stores_target: false,
            high_level_attempts: high_level_attempts.clone(),
            low_level_stores_target: true,
            low_level_attempts: low_level_attempts.clone(),
        });
        backend.route_hint("high 00".to_string()).await.unwrap();
        backend.route_hint("low 00".to_string()).await.unwrap();

        let preimage = backend.get_preimage(key).await.unwrap();

        assert_eq!(preimage, b"node".to_vec());
        assert!(high_level_attempts.load(Ordering::SeqCst) >= 1);
        assert!(low_level_attempts.load(Ordering::SeqCst) >= 1);
        assert!(
            backend.last_high_level_hint.read().await.is_some(),
            "a failing high-level hint is kept for retry, not cleared"
        );
    }

    /// `route_hint` routes a registered high-level type to its retained slot and everything else to
    /// the fine-grained slot.
    #[tokio::test]
    async fn route_hint_separates_high_level_from_fine_grained() {
        let (_key, target) = target_key();
        let backend = new_backend(TestProviders {
            target,
            value: Vec::new(),
            high_level_fail_until: 0,
            high_level_stores_target: false,
            high_level_attempts: Arc::new(AtomicUsize::new(0)),
            low_level_stores_target: false,
            low_level_attempts: Arc::new(AtomicUsize::new(0)),
        });

        backend.route_hint("high 00".to_string()).await.unwrap();
        backend.route_hint("low 00".to_string()).await.unwrap();

        assert_eq!(
            backend.last_high_level_hint.read().await.as_ref().map(|h| h.ty.clone()),
            Some(TestHint::HighLevel)
        );
        assert_eq!(
            backend.last_hint.read().await.as_ref().map(|h| h.ty.clone()),
            Some(TestHint::LowLevel)
        );
    }
}

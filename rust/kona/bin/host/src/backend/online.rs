//! Contains the [`OnlineHostBackend`] definition.

use crate::SharedKeyValueStore;
use alloy_primitives::hex;
use anyhow::Result;
use async_trait::async_trait;
use kona_preimage::{
    HintRouter, PreimageFetcher, PreimageKey,
    errors::{PreimageOracleError, PreimageOracleResult},
};
use kona_proof::{Hint, errors::HintParsingError};
use std::{fmt::Debug, str::FromStr, sync::Arc};
use tokio::sync::RwLock;
use tracing::trace;

/// The [`OnlineHostBackendCfg`] trait is used to define the type configuration for the
/// [`OnlineHostBackend`].
pub trait OnlineHostBackendCfg {
    /// The hint type describing the range of hints that can be received.
    type HintType: FromStr<Err = HintParsingError> + Clone + Debug + Send + Sync;

    /// The providers that are used to fetch data in response to hints.
    type Providers: Send + Sync;
}

/// A [`HintHandler`] is an interface for receiving hints, fetching remote data, and storing it in
/// the key-value store.
#[async_trait]
pub trait HintHandler {
    /// The type configuration for the [`HintHandler`].
    type Cfg: OnlineHostBackendCfg;

    /// Optionally fetches data immediately when a hint is routed.
    ///
    /// Returns `true` if the hint was handled eagerly and should not replace the latest lazy hint.
    async fn fetch_hint_eager(
        _hint: &Hint<<Self::Cfg as OnlineHostBackendCfg>::HintType>,
        _cfg: &Self::Cfg,
        _providers: &<Self::Cfg as OnlineHostBackendCfg>::Providers,
        _kv: SharedKeyValueStore,
    ) -> Result<bool> {
        Ok(false)
    }

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
    /// The latest hint received from the client, used for diagnostics even when fetched eagerly.
    last_routed_hint: Arc<RwLock<Option<Hint<C::HintType>>>>,
    /// The latest hint received from the client.
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
            last_routed_hint: Arc::new(RwLock::new(None)),
            last_hint: Arc::new(RwLock::new(None)),
            _hint_handler: std::marker::PhantomData,
        }
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
        self.last_routed_hint.write().await.replace(parsed_hint.clone());
        let fetched_eagerly =
            H::fetch_hint_eager(&parsed_hint, &self.cfg, &self.providers, self.kv.clone())
                .await
                .map_err(|e| {
                    PreimageOracleError::Other(format!("failed to eagerly fetch hint: {e}"))
                })?;

        if !fetched_eagerly {
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

        if let Some(preimage) = self.kv.read().await.get(key.into()) {
            return Ok(preimage);
        }

        let Some(hint) = self.last_hint.read().await.clone() else {
            if let Some(last_routed) = self.last_routed_hint.read().await.clone() {
                let hint_type = format!("{:?}", last_routed.ty);
                let hint_data_len = last_routed.data.len();
                let hint_data_prefix = hex::encode(&last_routed.data[..hint_data_len.min(32)]);
                return Err(PreimageOracleError::Other(format!(
                    "no lazy hint available for requested preimage key {key}; last routed hint was {hint_type} ({hint_data_len} bytes, prefix 0x{hint_data_prefix})"
                )));
            }
            return Err(PreimageOracleError::KeyNotFound);
        };

        let hint_type = format!("{:?}", hint.ty);
        let hint_data_len = hint.data.len();
        let hint_data_prefix = hex::encode(&hint.data[..hint_data_len.min(32)]);

        H::fetch_hint(hint, &self.cfg, &self.providers, self.kv.clone()).await.map_err(|e| {
            PreimageOracleError::Other(format!("failed to fetch latest hint for key {key}: {e}"))
        })?;

        self.kv.read().await.get(key.into()).ok_or_else(|| {
            PreimageOracleError::Other(format!(
                "latest hint {hint_type} ({hint_data_len} bytes, prefix 0x{hint_data_prefix}) did not populate requested preimage key {key}"
            ))
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::kv::MemoryKeyValueStore;
    use alloy_primitives::B256;
    use kona_preimage::PreimageKey;
    use std::sync::atomic::{AtomicUsize, Ordering};

    #[derive(Clone, Debug)]
    enum TestHint {
        Fill,
        Other,
        EagerFill,
    }

    impl FromStr for TestHint {
        type Err = HintParsingError;

        fn from_str(s: &str) -> Result<Self, Self::Err> {
            match s {
                "fill" => Ok(Self::Fill),
                "other" => Ok(Self::Other),
                "eager-fill" => Ok(Self::EagerFill),
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
        /// Whether fetching the fill hint fails.
        fill_fails: bool,
        /// Whether the fill hint stores `target` once it succeeds.
        fill_stores_target: bool,
        fill_attempts: Arc<AtomicUsize>,
        other_attempts: Arc<AtomicUsize>,
    }

    struct TestHintHandler;

    #[async_trait]
    impl HintHandler for TestHintHandler {
        type Cfg = TestCfg;

        async fn fetch_hint_eager(
            hint: &Hint<TestHint>,
            cfg: &TestCfg,
            providers: &TestProviders,
            kv: SharedKeyValueStore,
        ) -> Result<bool> {
            if matches!(hint.ty, TestHint::EagerFill) {
                Self::fetch_hint(hint.clone(), cfg, providers, kv).await?;
                return Ok(true);
            }
            Ok(false)
        }

        async fn fetch_hint(
            hint: Hint<TestHint>,
            _cfg: &TestCfg,
            providers: &TestProviders,
            kv: SharedKeyValueStore,
        ) -> Result<()> {
            match hint.ty {
                TestHint::Fill | TestHint::EagerFill => {
                    providers.fill_attempts.fetch_add(1, Ordering::SeqCst);
                    if providers.fill_fails {
                        anyhow::bail!("fill hint failed");
                    }
                    if providers.fill_stores_target {
                        kv.write().await.set(providers.target, providers.value.clone())?;
                    }
                    Ok(())
                }
                TestHint::Other => {
                    providers.other_attempts.fetch_add(1, Ordering::SeqCst);
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
    }

    #[tokio::test]
    async fn latest_hint_populates_missing_preimage() {
        let (key, target) = target_key();
        let fill_attempts = Arc::new(AtomicUsize::new(0));
        let backend = new_backend(TestProviders {
            target,
            value: b"witness".to_vec(),
            fill_fails: false,
            fill_stores_target: true,
            fill_attempts: fill_attempts.clone(),
            other_attempts: Arc::new(AtomicUsize::new(0)),
        });
        backend.route_hint("fill 00".to_string()).await.unwrap();

        let preimage = backend.get_preimage(key).await.unwrap();

        assert_eq!(preimage, b"witness".to_vec());
        assert_eq!(fill_attempts.load(Ordering::SeqCst), 1);
    }

    #[tokio::test]
    async fn latest_hint_replaces_previous_hint() {
        let (key, target) = target_key();
        let fill_attempts = Arc::new(AtomicUsize::new(0));
        let other_attempts = Arc::new(AtomicUsize::new(0));
        let backend = new_backend(TestProviders {
            target,
            value: b"witness".to_vec(),
            fill_fails: false,
            fill_stores_target: true,
            fill_attempts: fill_attempts.clone(),
            other_attempts: other_attempts.clone(),
        });
        backend.route_hint("fill 00".to_string()).await.unwrap();
        backend.route_hint("other 00".to_string()).await.unwrap();

        let err = backend.get_preimage(key).await.unwrap_err();

        assert!(matches!(err, PreimageOracleError::Other(_)));
        assert_eq!(fill_attempts.load(Ordering::SeqCst), 0);
        assert_eq!(other_attempts.load(Ordering::SeqCst), 1);
    }

    #[tokio::test]
    async fn eager_hint_populates_before_latest_hint_is_replaced() {
        let (key, target) = target_key();
        let fill_attempts = Arc::new(AtomicUsize::new(0));
        let other_attempts = Arc::new(AtomicUsize::new(0));
        let backend = new_backend(TestProviders {
            target,
            value: b"witness".to_vec(),
            fill_fails: false,
            fill_stores_target: true,
            fill_attempts: fill_attempts.clone(),
            other_attempts: other_attempts.clone(),
        });
        backend.route_hint("eager-fill 00".to_string()).await.unwrap();
        backend.route_hint("other 00".to_string()).await.unwrap();

        let preimage = backend.get_preimage(key).await.unwrap();

        assert_eq!(preimage, b"witness".to_vec());
        assert_eq!(fill_attempts.load(Ordering::SeqCst), 1);
        assert_eq!(other_attempts.load(Ordering::SeqCst), 0);
    }

    #[tokio::test]
    async fn eager_hint_failure_is_returned_on_route() {
        let (_, target) = target_key();
        let fill_attempts = Arc::new(AtomicUsize::new(0));
        let backend = new_backend(TestProviders {
            target,
            value: Vec::new(),
            fill_fails: true,
            fill_stores_target: false,
            fill_attempts: fill_attempts.clone(),
            other_attempts: Arc::new(AtomicUsize::new(0)),
        });

        let err = backend.route_hint("eager-fill 00".to_string()).await.unwrap_err();

        assert!(matches!(err, PreimageOracleError::Other(_)));
        assert_eq!(fill_attempts.load(Ordering::SeqCst), 1);
    }

    #[tokio::test]
    async fn latest_hint_failure_is_returned() {
        let (key, target) = target_key();
        let fill_attempts = Arc::new(AtomicUsize::new(0));
        let backend = new_backend(TestProviders {
            target,
            value: Vec::new(),
            fill_fails: true,
            fill_stores_target: false,
            fill_attempts: fill_attempts.clone(),
            other_attempts: Arc::new(AtomicUsize::new(0)),
        });
        backend.route_hint("fill 00".to_string()).await.unwrap();

        let err = backend.get_preimage(key).await.unwrap_err();

        assert!(matches!(err, PreimageOracleError::Other(_)));
        assert_eq!(fill_attempts.load(Ordering::SeqCst), 1);
    }

    #[tokio::test]
    async fn successful_hint_that_does_not_populate_key_errors() {
        let (key, target) = target_key();
        let fill_attempts = Arc::new(AtomicUsize::new(0));
        let backend = new_backend(TestProviders {
            target,
            value: Vec::new(),
            fill_fails: false,
            fill_stores_target: false,
            fill_attempts: fill_attempts.clone(),
            other_attempts: Arc::new(AtomicUsize::new(0)),
        });
        backend.route_hint("fill 00".to_string()).await.unwrap();

        let err = backend.get_preimage(key).await.unwrap_err();

        assert!(matches!(err, PreimageOracleError::Other(_)));
        assert_eq!(fill_attempts.load(Ordering::SeqCst), 1);
    }

    #[tokio::test]
    async fn missing_key_without_hint_returns_key_not_found() {
        let (_key, target) = target_key();
        let backend = new_backend(TestProviders {
            target,
            value: Vec::new(),
            fill_fails: false,
            fill_stores_target: false,
            fill_attempts: Arc::new(AtomicUsize::new(0)),
            other_attempts: Arc::new(AtomicUsize::new(0)),
        });

        let err = backend.get_preimage(PreimageKey::new_keccak256(*target)).await.unwrap_err();

        assert!(matches!(err, PreimageOracleError::KeyNotFound));
    }
}

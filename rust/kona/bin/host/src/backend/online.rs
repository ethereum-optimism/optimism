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
use tokio::{sync::RwLock, time::Duration};
use tracing::{debug, error, trace};

/// Bounded retry policy for `OnlineHostBackend::get_preimage`. Private to the `online` module.
#[allow(dead_code)]
struct RetryPolicy {
    /// Maximum number of `fetch_hint` invocations per `get_preimage` call.
    max_attempts: usize,
    /// Initial backoff between retry attempts.
    initial_backoff: Duration,
    /// Maximum backoff between retry attempts.
    max_backoff: Duration,
}

impl Default for RetryPolicy {
    fn default() -> Self {
        Self {
            max_attempts: 5,
            initial_backoff: Duration::from_millis(50),
            max_backoff: Duration::from_secs(2),
        }
    }
}

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
    /// Hints that should be immediately executed by the host.
    proactive_hints: HashSet<C::HintType>,
    /// The last hint that was received.
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
            proactive_hints: HashSet::default(),
            last_hint: Arc::new(RwLock::new(None)),
            _hint_handler: std::marker::PhantomData,
        }
    }

    /// Adds a new proactive hint to the [`OnlineHostBackend`].
    pub fn with_proactive_hint(mut self, hint_type: C::HintType) -> Self {
        self.proactive_hints.insert(hint_type);
        self
    }

    /// Override the default retry policy. Test-only seam.
    #[cfg(test)]
    #[allow(clippy::missing_const_for_fn)]
    fn with_retry_policy(self, _policy: RetryPolicy) -> Self {
        // Commit 1 (red): no-op stub. Commit 2 replaces the body with the real assignment.
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
        if self.proactive_hints.contains(&parsed_hint.ty) {
            debug!(target: "host_backend", "Proactive hint received; Immediately fetching {hint}");
            H::fetch_hint(parsed_hint, &self.cfg, &self.providers, self.kv.clone())
                .await
                .map_err(|e| PreimageOracleError::Other(e.to_string()))?;
        } else {
            let mut hint_lock = self.last_hint.write().await;
            hint_lock.replace(parsed_hint);
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
            if let Some(hint) = self.last_hint.read().await.as_ref() {
                let value =
                    H::fetch_hint(hint.clone(), &self.cfg, &self.providers, self.kv.clone()).await;

                if let Err(e) = value {
                    error!(target: "host_backend", "Failed to prefetch hint: {e}");
                    continue;
                }

                let kv_lock = self.kv.read().await;
                preimage = kv_lock.get(key.into());
            }
        }

        preimage.ok_or(PreimageOracleError::KeyNotFound)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::kv::MemoryKeyValueStore;
    use alloy_primitives::{B256, Bytes};
    use kona_preimage::{PreimageKey, PreimageKeyType};
    use std::sync::atomic::{AtomicUsize, Ordering};
    use tokio::sync::Mutex;

    #[derive(Clone, Hash, Eq, PartialEq, Debug)]
    struct TestHintType;

    impl FromStr for TestHintType {
        type Err = HintParsingError;

        fn from_str(_: &str) -> Result<Self, Self::Err> {
            Ok(Self)
        }
    }

    enum TestBehavior {
        AlwaysErr,
        AlwaysOkNoWrite,
        FlakyThenWrite { failures_remaining: usize, kv_key: B256, value: Vec<u8> },
    }

    #[derive(Clone)]
    struct TestCfg {
        behavior: Arc<Mutex<TestBehavior>>,
        fetch_count: Arc<AtomicUsize>,
    }

    impl OnlineHostBackendCfg for TestCfg {
        type HintType = TestHintType;
        type Providers = ();
    }

    struct TestHintHandler;

    #[async_trait]
    impl HintHandler for TestHintHandler {
        type Cfg = TestCfg;

        async fn fetch_hint(
            _hint: Hint<TestHintType>,
            cfg: &Self::Cfg,
            _providers: &(),
            kv: SharedKeyValueStore,
        ) -> Result<()> {
            cfg.fetch_count.fetch_add(1, Ordering::SeqCst);
            let mut behavior = cfg.behavior.lock().await;
            match &mut *behavior {
                TestBehavior::AlwaysErr => Err(anyhow::anyhow!("simulated rpc failure")),
                TestBehavior::AlwaysOkNoWrite => Ok(()),
                TestBehavior::FlakyThenWrite { failures_remaining, kv_key, value } => {
                    if *failures_remaining > 0 {
                        *failures_remaining -= 1;
                        Err(anyhow::anyhow!("flaky"))
                    } else {
                        kv.write().await.set(*kv_key, value.clone())?;
                        Ok(())
                    }
                }
            }
        }
    }

    fn tiny_policy(max_attempts: usize) -> RetryPolicy {
        RetryPolicy {
            max_attempts,
            initial_backoff: Duration::from_millis(1),
            max_backoff: Duration::from_millis(2),
        }
    }

    fn shared_mem_kv() -> SharedKeyValueStore {
        Arc::new(RwLock::new(MemoryKeyValueStore::new()))
    }

    fn make_backend(
        behavior: TestBehavior,
        max_attempts: usize,
    ) -> (OnlineHostBackend<TestCfg, TestHintHandler>, TestCfg) {
        let cfg = TestCfg {
            behavior: Arc::new(Mutex::new(behavior)),
            fetch_count: Arc::new(AtomicUsize::new(0)),
        };
        let kv = shared_mem_kv();
        let backend = OnlineHostBackend::<TestCfg, TestHintHandler>::new(
            cfg.clone(),
            kv,
            (),
            TestHintHandler,
        )
        .with_retry_policy(tiny_policy(max_attempts));
        (backend, cfg)
    }

    async fn set_hint(backend: &OnlineHostBackend<TestCfg, TestHintHandler>) {
        *backend.last_hint.write().await = Some(Hint { ty: TestHintType, data: Bytes::new() });
    }

    #[tokio::test]
    async fn no_hint_returns_key_not_found_immediately() {
        let (backend, cfg) = make_backend(TestBehavior::AlwaysOkNoWrite, 3);
        // Intentionally leave last_hint as None.
        let preimage_key = PreimageKey::new([0xFFu8; 32], PreimageKeyType::Keccak256);

        let result =
            tokio::time::timeout(Duration::from_millis(500), backend.get_preimage(preimage_key))
                .await;

        assert!(
            result.is_ok(),
            "call should return immediately when last_hint is None; baseline spins forever in the no-hint case"
        );
        let inner = result.unwrap();
        assert!(
            matches!(inner, Err(PreimageOracleError::KeyNotFound)),
            "expected KeyNotFound, got {inner:?}"
        );
        assert_eq!(
            cfg.fetch_count.load(Ordering::SeqCst),
            0,
            "fetch_hint must not be invoked when last_hint is None"
        );
    }

    #[tokio::test]
    async fn permanent_fetch_failure_returns_other_after_budget() {
        let (backend, cfg) = make_backend(TestBehavior::AlwaysErr, 3);
        set_hint(&backend).await;
        let preimage_key = PreimageKey::new([0xFFu8; 32], PreimageKeyType::Keccak256);

        let result =
            tokio::time::timeout(Duration::from_millis(500), backend.get_preimage(preimage_key))
                .await;

        assert!(
            result.is_ok(),
            "call should return Other after exhausting budget; baseline spins forever on permanent fetch failure"
        );
        let inner = result.unwrap();
        assert!(
            matches!(inner, Err(PreimageOracleError::Other(_))),
            "expected Other(_), got {inner:?}"
        );
        assert_eq!(cfg.fetch_count.load(Ordering::SeqCst), 3, "budget must be consumed exactly");
    }

    #[tokio::test]
    async fn fetch_ok_but_key_missing_counts_against_budget() {
        let (backend, cfg) = make_backend(TestBehavior::AlwaysOkNoWrite, 3);
        set_hint(&backend).await;
        let preimage_key = PreimageKey::new([0xFFu8; 32], PreimageKeyType::Keccak256);

        let result =
            tokio::time::timeout(Duration::from_millis(500), backend.get_preimage(preimage_key))
                .await;

        assert!(
            result.is_ok(),
            "Ok-but-missing case should be retry-bounded; baseline spins forever"
        );
        let inner = result.unwrap();
        assert!(
            matches!(inner, Err(PreimageOracleError::Other(_))),
            "expected Other(_), got {inner:?}"
        );
        assert_eq!(
            cfg.fetch_count.load(Ordering::SeqCst),
            3,
            "Ok-but-missing must count one attempt per call"
        );
    }

    #[tokio::test]
    async fn transient_failure_then_success_returns_preimage() {
        let preimage_key = PreimageKey::new([0xFFu8; 32], PreimageKeyType::Keccak256);
        let kv_key: B256 = preimage_key.into();
        let expected_preimage: Vec<u8> = b"the preimage bytes".to_vec();

        let (backend, cfg) = make_backend(
            TestBehavior::FlakyThenWrite {
                failures_remaining: 2,
                kv_key,
                value: expected_preimage.clone(),
            },
            5,
        );
        set_hint(&backend).await;

        let result =
            tokio::time::timeout(Duration::from_millis(500), backend.get_preimage(preimage_key))
                .await
                .expect("timeout: transient-then-success should resolve quickly");

        assert_eq!(result.expect("expected Ok(preimage), got error"), expected_preimage);
        assert_eq!(
            cfg.fetch_count.load(Ordering::SeqCst),
            3,
            "two failures + one success = three attempts"
        );
    }
}

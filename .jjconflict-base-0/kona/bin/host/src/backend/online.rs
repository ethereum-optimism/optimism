//! Contains the [OnlineHostBackend] definition.

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

/// The [OnlineHostBackendCfg] trait is used to define the type configuration for the
/// [OnlineHostBackend].
pub trait OnlineHostBackendCfg {
    /// The hint type describing the range of hints that can be received.
    type HintType: FromStr<Err = HintParsingError> + Hash + Eq + PartialEq + Clone + Send + Sync;

    /// The providers that are used to fetch data in response to hints.
    type Providers: Send + Sync;
}

/// A [HintHandler] is an interface for receiving hints, fetching remote data, and storing it in the
/// key-value store.
#[async_trait]
pub trait HintHandler {
    /// The type configuration for the [HintHandler].
    type Cfg: OnlineHostBackendCfg;

    /// Fetches data in response to a hint.
    async fn fetch_hint(
        hint: Hint<<Self::Cfg as OnlineHostBackendCfg>::HintType>,
        cfg: &Self::Cfg,
        providers: &<Self::Cfg as OnlineHostBackendCfg>::Providers,
        kv: SharedKeyValueStore,
    ) -> Result<()>;
}

/// The [OnlineHostBackend] is a [HintRouter] and [PreimageFetcher] that is used to fetch data from
/// remote sources in response to hints.
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
    /// Phantom marker for the [HintHandler].
    _hint_handler: std::marker::PhantomData<H>,
}

impl<C, H> OnlineHostBackend<C, H>
where
    C: OnlineHostBackendCfg,
    H: HintHandler,
{
    /// Creates a new [HintHandler] with the given configuration, key-value store, providers, and
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

    /// Adds a new proactive hint to the [OnlineHostBackend].
    pub fn with_proactive_hint(mut self, hint_type: C::HintType) -> Self {
        self.proactive_hints.insert(hint_type);
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

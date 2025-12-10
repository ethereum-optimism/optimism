//! Contains the implementations of the [HintRouter] and [PreimageFetcher] traits.

use crate::kv::KeyValueStore;
use async_trait::async_trait;
use kona_preimage::{
    HintRouter, PreimageFetcher, PreimageKey,
    errors::{PreimageOracleError, PreimageOracleResult},
};
use std::sync::Arc;
use tokio::sync::RwLock;

/// A [KeyValueStore]-backed implementation of the [PreimageFetcher] trait.
#[derive(Debug)]
pub struct OfflineHostBackend<KV>
where
    KV: KeyValueStore + ?Sized,
{
    inner: Arc<RwLock<KV>>,
}

impl<KV> OfflineHostBackend<KV>
where
    KV: KeyValueStore + ?Sized,
{
    /// Create a new [OfflineHostBackend] from the given [KeyValueStore].
    pub const fn new(kv_store: Arc<RwLock<KV>>) -> Self {
        Self { inner: kv_store }
    }
}

#[async_trait]
impl<KV> PreimageFetcher for OfflineHostBackend<KV>
where
    KV: KeyValueStore + Send + Sync + ?Sized,
{
    async fn get_preimage(&self, key: PreimageKey) -> PreimageOracleResult<Vec<u8>> {
        let kv_store = self.inner.read().await;
        kv_store.get(key.into()).ok_or(PreimageOracleError::KeyNotFound)
    }
}

#[async_trait]
impl<KV> HintRouter for OfflineHostBackend<KV>
where
    KV: KeyValueStore + Send + Sync + ?Sized,
{
    async fn route_hint(&self, _hint: String) -> PreimageOracleResult<()> {
        Ok(())
    }
}

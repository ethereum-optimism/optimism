//! The [`AlloyChainProvider`](super::AlloyChainProvider)'s cache of verified L1 responses.

#[cfg(feature = "metrics")]
use crate::Metrics;
use alloy_consensus::{Header, Receipt, TxEnvelope};
use alloy_primitives::B256;
use lru::LruCache;
use std::{
    num::NonZeroUsize,
    sync::{Mutex, MutexGuard, PoisonError},
};

/// An LRU cache keyed by block hash that records its own hit, miss and entry metrics.
#[derive(Debug)]
struct HashCache<V> {
    /// The cached values.
    inner: LruCache<B256, V>,
    /// The `cache` label on this cache's metrics.
    #[cfg_attr(not(feature = "metrics"), allow(dead_code))]
    metrics_label: &'static str,
}

impl<V> HashCache<V> {
    /// Creates an empty cache that holds at most `size` entries.
    fn new(size: NonZeroUsize, metrics_label: &'static str) -> Self {
        Self { inner: LruCache::new(size), metrics_label }
    }

    /// Inserts `value` under `hash`, evicting the least recently used entry if the cache is full.
    fn put(&mut self, hash: B256, value: V) {
        self.inner.put(hash, value);
        kona_macros::set!(
            gauge,
            Metrics::CHAIN_PROVIDER_CACHE_ENTRIES,
            "cache",
            self.metrics_label,
            self.inner.len() as f64
        );
    }

    /// Returns the value under `hash` and marks it most recently used. Every call counts as a hit
    /// or a miss.
    fn get(&mut self, hash: B256) -> Option<&V> {
        let value = self.inner.get(&hash);
        if value.is_some() {
            kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_CACHE_HITS, "cache" => self.metrics_label);
        } else {
            kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_CACHE_MISSES, "cache" => self.metrics_label);
        }
        value
    }
}

/// Responses cached by block hash.
///
/// Every entry was fetched under the same trust setting, so sharing never mixes verified and
/// unverified data. Each lock is held for a single lookup or insert, never across an RPC call.
#[derive(Debug)]
pub(super) struct Cache {
    /// Block headers.
    headers: Mutex<HashCache<Header>>,
    /// Block receipts.
    receipts: Mutex<HashCache<Vec<Receipt>>>,
    /// Block transactions.
    transactions: Mutex<HashCache<Vec<TxEnvelope>>>,
}

/// Locks a cache. A panic while holding the lock cannot leave an LRU cache inconsistent, so a
/// poisoned lock is still usable.
fn lock<T>(mutex: &Mutex<T>) -> MutexGuard<'_, T> {
    mutex.lock().unwrap_or_else(PoisonError::into_inner)
}

impl Cache {
    /// Creates an empty cache in which each kind of response holds at most `size` entries.
    pub(super) fn new(size: NonZeroUsize) -> Self {
        Self {
            headers: Mutex::new(HashCache::new(size, "headers")),
            receipts: Mutex::new(HashCache::new(size, "receipts")),
            transactions: Mutex::new(HashCache::new(size, "transactions")),
        }
    }

    /// Returns the cached header of the block with the given hash.
    pub(super) fn header(&self, hash: B256) -> Option<Header> {
        lock(&self.headers).get(hash).cloned()
    }

    /// Caches the header of the block with the given hash.
    pub(super) fn insert_header(&self, hash: B256, header: Header) {
        lock(&self.headers).put(hash, header);
    }

    /// Returns the cached receipts of the block with the given hash.
    pub(super) fn receipts(&self, hash: B256) -> Option<Vec<Receipt>> {
        lock(&self.receipts).get(hash).cloned()
    }

    /// Caches the receipts of the block with the given hash.
    pub(super) fn insert_receipts(&self, hash: B256, receipts: Vec<Receipt>) {
        lock(&self.receipts).put(hash, receipts);
    }

    /// Returns the cached transactions of the block with the given hash.
    pub(super) fn transactions(&self, hash: B256) -> Option<Vec<TxEnvelope>> {
        lock(&self.transactions).get(hash).cloned()
    }

    /// Caches the transactions of the block with the given hash.
    pub(super) fn insert_transactions(&self, hash: B256, transactions: Vec<TxEnvelope>) {
        lock(&self.transactions).put(hash, transactions);
    }
}

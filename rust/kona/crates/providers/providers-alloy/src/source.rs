//! Consensus-data sources composed as fetching and caching layers.

#[cfg(feature = "metrics")]
use crate::Metrics;
use crate::SourceError;
use alloy_consensus::{Block, Header, ReceiptEnvelope, transaction::TxHashRef};
use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::B256;
use async_trait::async_trait;
use lru::LruCache;
use std::{fmt::Debug, num::NonZeroUsize, sync::Arc};

/// Block data before conversion into derivation's L1 or L2 interfaces.
#[async_trait]
pub trait BlockSource: Clone + Debug + Send + Sync {
    /// Consensus transaction encoding, including its reported hash.
    type Transaction: Clone + Debug + Encodable2718 + TxHashRef + Send + Sync;

    /// Fetches a header by hash.
    async fn header_by_hash(&mut self, hash: B256) -> Result<Header, SourceError>;

    /// Fetches a header by number, retaining the RPC's claimed hash.
    async fn header_by_number(&mut self, number: u64) -> Result<(B256, Header), SourceError>;

    /// Fetches a full block by hash.
    async fn block_by_hash(
        &mut self,
        hash: B256,
    ) -> Result<Arc<Block<Self::Transaction>>, SourceError>;

    /// Fetches a full block by number, retaining the RPC's claimed hash.
    async fn block_by_number(
        &mut self,
        number: u64,
    ) -> Result<(B256, Arc<Block<Self::Transaction>>), SourceError>;

    /// Returns the RPC's chain ID.
    async fn chain_id(&mut self) -> Result<u64, SourceError>;

    /// Returns the RPC's latest block number.
    async fn latest_block_number(&mut self) -> Result<u64, SourceError>;
}

/// Ethereum receipts with their consensus type and bloom intact.
#[async_trait]
pub trait ReceiptSource: BlockSource {
    /// Fetches the block's receipts in transaction order.
    async fn receipts_by_hash(&mut self, hash: B256) -> Result<Vec<ReceiptEnvelope>, SourceError>;
}

/// Hash-keyed caches over an already selected source.
///
/// Failed responses are never cached. Number-based lookups always reach the source because a
/// height can resolve to a different block after a reorg.
#[derive(Debug, Clone)]
pub struct Cached<S: BlockSource> {
    pub(crate) inner: S,
    pub(crate) headers: LruCache<B256, Header>,
    pub(crate) blocks: LruCache<B256, Arc<Block<S::Transaction>>>,
    pub(crate) receipts: LruCache<B256, Vec<ReceiptEnvelope>>,
}

impl<S: BlockSource> Cached<S> {
    /// Creates empty hash-keyed caches.
    ///
    /// # Panics
    /// Panics if `cache_size` is zero.
    pub fn new(inner: S, cache_size: usize) -> Self {
        let size = NonZeroUsize::new(cache_size).expect("cache size must be nonzero");
        Self {
            inner,
            headers: LruCache::new(size),
            blocks: LruCache::new(size),
            receipts: LruCache::new(size),
        }
    }
}

#[async_trait]
impl<S: BlockSource> BlockSource for Cached<S> {
    type Transaction = S::Transaction;

    async fn header_by_hash(&mut self, hash: B256) -> Result<Header, SourceError> {
        if let Some(header) = self.headers.get(&hash) {
            kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_CACHE_HITS, "cache" => "header_by_hash");
            return Ok(header.clone());
        }
        kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_CACHE_MISSES, "cache" => "header_by_hash");
        let header = self.inner.header_by_hash(hash).await?;
        self.headers.put(hash, header.clone());
        kona_macros::inc!(gauge, Metrics::CACHE_ENTRIES, "cache" => "header_by_hash");
        Ok(header)
    }

    async fn header_by_number(&mut self, number: u64) -> Result<(B256, Header), SourceError> {
        self.inner.header_by_number(number).await
    }

    async fn block_by_hash(
        &mut self,
        hash: B256,
    ) -> Result<Arc<Block<Self::Transaction>>, SourceError> {
        if let Some(block) = self.blocks.get(&hash) {
            kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_CACHE_HITS, "cache" => "block_info_and_tx");
            return Ok(Arc::clone(block));
        }
        kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_CACHE_MISSES, "cache" => "block_info_and_tx");
        let block = self.inner.block_by_hash(hash).await?;
        self.blocks.put(hash, Arc::clone(&block));
        kona_macros::inc!(gauge, Metrics::CACHE_ENTRIES, "cache" => "block_info_and_tx");
        Ok(block)
    }

    async fn block_by_number(
        &mut self,
        number: u64,
    ) -> Result<(B256, Arc<Block<Self::Transaction>>), SourceError> {
        let (hash, block) = self.inner.block_by_number(number).await?;
        self.blocks.put(block.header.hash_slow(), Arc::clone(&block));
        Ok((hash, block))
    }

    async fn chain_id(&mut self) -> Result<u64, SourceError> {
        self.inner.chain_id().await
    }

    async fn latest_block_number(&mut self) -> Result<u64, SourceError> {
        self.inner.latest_block_number().await
    }
}

#[async_trait]
impl<S: ReceiptSource> ReceiptSource for Cached<S> {
    async fn receipts_by_hash(&mut self, hash: B256) -> Result<Vec<ReceiptEnvelope>, SourceError> {
        if let Some(receipts) = self.receipts.get(&hash) {
            kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_CACHE_HITS, "cache" => "receipts_by_hash");
            return Ok(receipts.clone());
        }
        kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_CACHE_MISSES, "cache" => "receipts_by_hash");
        let receipts = self.inner.receipts_by_hash(hash).await?;
        self.receipts.put(hash, receipts.clone());
        kona_macros::inc!(gauge, Metrics::CACHE_ENTRIES, "cache" => "receipts_by_hash");
        Ok(receipts)
    }
}

//! Providers that use alloy provider types on the backend.

#[cfg(feature = "metrics")]
use crate::Metrics;
use alloy_consensus::{Header, Receipt, TxEnvelope};
use alloy_eips::BlockId;
use alloy_primitives::B256;
use alloy_provider::{Provider, RootProvider};
use alloy_transport::{RpcError, TransportErrorKind};
use async_trait::async_trait;
use kona_derive::{ChainProvider, PipelineError, PipelineErrorKind};
use kona_protocol::BlockInfo;
use lru::LruCache;
use std::{boxed::Box, num::NonZeroUsize, vec::Vec};

/// The [AlloyChainProvider] is a concrete implementation of the [ChainProvider] trait, providing
/// data over Ethereum JSON-RPC using an alloy provider as the backend.
#[derive(Debug, Clone)]
pub struct AlloyChainProvider {
    /// The inner Ethereum JSON-RPC provider.
    pub inner: RootProvider,
    /// Whether to trust the RPC without verification.
    pub trust_rpc: bool,
    /// `header_by_hash` LRU cache.
    header_by_hash_cache: LruCache<B256, Header>,
    /// `receipts_by_hash_cache` LRU cache.
    receipts_by_hash_cache: LruCache<B256, Vec<Receipt>>,
    /// `block_info_and_transactions_by_hash` LRU cache.
    block_info_and_transactions_by_hash_cache: LruCache<B256, (BlockInfo, Vec<TxEnvelope>)>,
}

impl AlloyChainProvider {
    /// Creates a new [AlloyChainProvider] with the given alloy provider.
    ///
    /// ## Panics
    /// - Panics if `cache_size` is zero.
    pub fn new(inner: RootProvider, cache_size: usize) -> Self {
        Self::new_with_trust(inner, cache_size, true)
    }

    /// Creates a new [AlloyChainProvider] with the given alloy provider and trust setting.
    ///
    /// ## Panics
    /// - Panics if `cache_size` is zero.
    pub fn new_with_trust(inner: RootProvider, cache_size: usize, trust_rpc: bool) -> Self {
        Self {
            inner,
            trust_rpc,
            header_by_hash_cache: LruCache::new(NonZeroUsize::new(cache_size).unwrap()),
            receipts_by_hash_cache: LruCache::new(NonZeroUsize::new(cache_size).unwrap()),
            block_info_and_transactions_by_hash_cache: LruCache::new(
                NonZeroUsize::new(cache_size).unwrap(),
            ),
        }
    }

    /// Creates a new [AlloyChainProvider] from the provided [reqwest::Url].
    pub fn new_http(url: reqwest::Url, cache_size: usize) -> Self {
        let inner = RootProvider::new_http(url);
        Self::new(inner, cache_size)
    }

    /// Returns the latest L2 block number.
    pub async fn latest_block_number(&mut self) -> Result<u64, RpcError<TransportErrorKind>> {
        kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_RPC_CALLS, "method" => "block_number");

        let result = self.inner.get_block_number().await;

        #[cfg(feature = "metrics")]
        if result.is_err() {
            kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_RPC_ERRORS, "method" => "block_number");
        }

        result
    }

    /// Returns the chain ID.
    pub async fn chain_id(&mut self) -> Result<u64, RpcError<TransportErrorKind>> {
        self.inner.get_chain_id().await
    }

    /// Verifies that a header's hash matches the expected hash when trust_rpc is false.
    fn verify_header_hash(
        &self,
        header: &Header,
        expected_hash: B256,
    ) -> Result<(), AlloyChainProviderError> {
        if self.trust_rpc {
            return Ok(());
        }

        let actual_hash = header.hash_slow();
        if actual_hash != expected_hash {
            return Err(AlloyChainProviderError::Transport(RpcError::Transport(
                TransportErrorKind::Custom(
                    format!(
                        "Header hash mismatch: expected {expected_hash:?}, got {actual_hash:?}"
                    )
                    .into(),
                ),
            )));
        }

        Ok(())
    }
}

/// An error for the [AlloyChainProvider].
#[allow(clippy::enum_variant_names)]
#[derive(Debug, thiserror::Error)]
pub enum AlloyChainProviderError {
    /// Transport error
    #[error(transparent)]
    Transport(#[from] RpcError<TransportErrorKind>),
    /// Block not found.
    #[error("Block not found: {0}")]
    BlockNotFound(BlockId),
    /// Failed to convert RPC receipts into consensus receipts.
    #[error("Failed to convert RPC receipts into consensus receipts: {0}")]
    ReceiptsConversion(B256),
}

impl From<AlloyChainProviderError> for PipelineErrorKind {
    fn from(e: AlloyChainProviderError) -> Self {
        match e {
            AlloyChainProviderError::Transport(e) => {
                Self::Temporary(PipelineError::Provider(format!("Transport error: {e}")))
            }
            AlloyChainProviderError::BlockNotFound(id) => {
                Self::Temporary(PipelineError::Provider(format!("L1 Block not found: {id}")))
            }
            AlloyChainProviderError::ReceiptsConversion(_) => {
                Self::Temporary(PipelineError::Provider(
                    "Failed to convert RPC receipts into consensus receipts".to_string(),
                ))
            }
        }
    }
}

#[async_trait]
impl ChainProvider for AlloyChainProvider {
    type Error = AlloyChainProviderError;

    async fn header_by_hash(&mut self, hash: B256) -> Result<Header, Self::Error> {
        if let Some(header) = self.header_by_hash_cache.get(&hash) {
            kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_CACHE_HITS, "cache" => "header_by_hash");
            return Ok(header.clone());
        }

        kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_CACHE_MISSES, "cache" => "header_by_hash");

        kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_RPC_CALLS, "method" => "header_by_hash");

        let block = self
            .inner
            .get_block_by_hash(hash)
            .await
            .inspect_err(|_e| {
                kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_RPC_ERRORS, "method" => "header_by_hash");
            })?
            .ok_or(AlloyChainProviderError::BlockNotFound(hash.into()))?;
        let header = block.header.into_consensus();

        // Verify the header hash matches what we requested
        self.verify_header_hash(&header, hash)?;

        self.header_by_hash_cache.put(hash, header.clone());

        kona_macros::inc!(gauge, Metrics::CACHE_ENTRIES, "cache" => "header_by_hash");

        Ok(header)
    }

    async fn block_info_by_number(&mut self, number: u64) -> Result<BlockInfo, Self::Error> {
        kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_RPC_CALLS, "method" => "block_by_number");

        let block = self
            .inner
            .get_block_by_number(number.into())
            .await
            .inspect_err(|_e| {
                kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_RPC_ERRORS, "method" => "block_by_number");
            })?
            .ok_or(AlloyChainProviderError::BlockNotFound(number.into()))?;
        let header = block.header.into_consensus();

        let block_info = BlockInfo {
            hash: header.hash_slow(),
            number,
            parent_hash: header.parent_hash,
            timestamp: header.timestamp,
        };
        Ok(block_info)
    }

    async fn receipts_by_hash(&mut self, hash: B256) -> Result<Vec<Receipt>, Self::Error> {
        if let Some(receipts) = self.receipts_by_hash_cache.get(&hash) {
            kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_CACHE_HITS, "cache" => "receipts_by_hash");
            return Ok(receipts.clone());
        }

        kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_CACHE_MISSES, "cache" => "receipts_by_hash");

        kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_RPC_CALLS, "method" => "receipts_by_hash");

        let receipts = self
            .inner
            .get_block_receipts(hash.into())
            .await
            .inspect_err(|_e| {
                kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_RPC_ERRORS, "method" => "receipts_by_hash");
            })?
            .ok_or(AlloyChainProviderError::BlockNotFound(hash.into()))?;
        let consensus_receipts = receipts
            .into_iter()
            .map(|r| r.inner.into_primitives_receipt().as_receipt().cloned())
            .collect::<Option<Vec<_>>>()
            .ok_or(AlloyChainProviderError::ReceiptsConversion(hash))?;

        self.receipts_by_hash_cache.put(hash, consensus_receipts.clone());

        kona_macros::inc!(gauge, Metrics::CACHE_ENTRIES, "cache" => "receipts_by_hash");

        Ok(consensus_receipts)
    }

    async fn block_info_and_transactions_by_hash(
        &mut self,
        hash: B256,
    ) -> Result<(BlockInfo, Vec<TxEnvelope>), Self::Error> {
        if let Some(block_info_and_txs) = self.block_info_and_transactions_by_hash_cache.get(&hash)
        {
            kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_CACHE_HITS, "cache" => "block_info_and_tx");
            return Ok(block_info_and_txs.clone());
        }

        kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_CACHE_MISSES, "cache" => "block_info_and_tx");

        kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_RPC_CALLS, "method" => "block_by_hash");

        let block = self
            .inner
            .get_block_by_hash(hash)
            .full()
            .await
            .inspect_err(|_e| {
                kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_RPC_ERRORS, "method" => "block_by_hash");
            })?
            .ok_or(AlloyChainProviderError::BlockNotFound(hash.into()))?
            .into_consensus()
            .map_transactions(|t| t.inner.into_inner());

        // Verify the block hash matches what we requested
        self.verify_header_hash(&block.header, hash)?;

        let block_info = BlockInfo {
            hash, // Use the already verified hash instead of recomputing
            number: block.header.number,
            parent_hash: block.header.parent_hash,
            timestamp: block.header.timestamp,
        };

        self.block_info_and_transactions_by_hash_cache
            .put(hash, (block_info, block.body.transactions.clone()));

        kona_macros::inc!(gauge, Metrics::CACHE_ENTRIES, "cache" => "block_info_and_tx");

        Ok((block_info, block.body.transactions))
    }
}

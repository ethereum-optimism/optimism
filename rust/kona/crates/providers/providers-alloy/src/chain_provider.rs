//! Providers that use alloy provider types on the backend.

#[cfg(feature = "metrics")]
use crate::Metrics;
use crate::verify;
use alloy_consensus::{Header, Receipt, ReceiptEnvelope, TxEnvelope};
use alloy_eips::{
    BlockId,
    eip2718::{Decodable2718, Eip2718Error},
};
use alloy_primitives::{B256, Bytes};
use alloy_provider::{Provider, RootProvider};
use alloy_transport::{RpcError, TransportErrorKind};
use alloy_transport_http::reqwest;
use async_trait::async_trait;
use kona_derive::{ChainProvider, PipelineError, PipelineErrorKind, ResetError};
use kona_protocol::BlockInfo;
use lru::LruCache;
use std::{
    boxed::Box,
    num::NonZeroUsize,
    sync::{
        Arc,
        atomic::{AtomicBool, Ordering},
    },
    vec::Vec,
};
use tokio::sync::watch;

/// Limits online derivation's forward traversal to sufficiently confirmed L1 blocks.
#[derive(Debug, Clone)]
struct ConfirmationDepth {
    depth: u64,
    head: watch::Receiver<Option<BlockInfo>>,
}

impl ConfirmationDepth {
    fn allows(&self, number: u64) -> bool {
        let head = *self.head.borrow();
        let Some(head) = head.filter(|head| *head != BlockInfo::default()) else {
            // Like op-node, do not restrict startup before the first L1 head signal.
            return true;
        };
        // Always expose L1 genesis, and avoid accepting future blocks through integer overflow.
        number == 0 ||
            self.depth == 0 ||
            number
                .checked_add(self.depth)
                .is_some_and(|confirmed_at| confirmed_at <= head.number)
    }
}

/// The [`AlloyChainProvider`] is a concrete implementation of the [`ChainProvider`] trait,
/// providing data over Ethereum JSON-RPC using an alloy provider as the backend.
#[derive(Debug, Clone)]
pub struct AlloyChainProvider {
    /// The inner Ethereum JSON-RPC provider.
    pub inner: RootProvider,
    /// Whether to trust the RPC without verification.
    pub trust_rpc: bool,
    /// Optional confirmation policy for online derivation's forward traversal only.
    confirmation_depth: Option<ConfirmationDepth>,
    /// `header_by_hash` LRU cache.
    header_by_hash_cache: LruCache<B256, Header>,
    /// `receipts_by_hash_cache` LRU cache.
    receipts_by_hash_cache: LruCache<B256, Vec<Receipt>>,
    /// Whether the L1 RPC supports `debug_getRawReceipts`.
    ///
    /// The flag is shared across clones because a pipeline contains multiple clones of this
    /// provider. Once any clone observes `MethodNotFound`, all clones must use the fallback.
    raw_receipts_supported: Arc<AtomicBool>,
    /// `block_info_and_transactions_by_hash` LRU cache.
    block_info_and_transactions_by_hash_cache: LruCache<B256, (BlockInfo, Vec<TxEnvelope>)>,
}

impl AlloyChainProvider {
    /// Creates a new [`AlloyChainProvider`] with the given alloy provider.
    ///
    /// ## Panics
    /// - Panics if `cache_size` is zero.
    pub fn new(inner: RootProvider, cache_size: usize) -> Self {
        Self::new_with_trust(inner, cache_size, true)
    }

    /// Creates a new [`AlloyChainProvider`] with the given alloy provider and trust setting.
    ///
    /// ## Panics
    /// - Panics if `cache_size` is zero.
    pub fn new_with_trust(inner: RootProvider, cache_size: usize, trust_rpc: bool) -> Self {
        Self {
            inner,
            trust_rpc,
            confirmation_depth: None,
            header_by_hash_cache: LruCache::new(NonZeroUsize::new(cache_size).unwrap()),
            receipts_by_hash_cache: LruCache::new(NonZeroUsize::new(cache_size).unwrap()),
            raw_receipts_supported: Arc::new(AtomicBool::new(true)),
            block_info_and_transactions_by_hash_cache: LruCache::new(
                NonZeroUsize::new(cache_size).unwrap(),
            ),
        }
    }

    /// Creates a new [`AlloyChainProvider`] from the provided [`reqwest::Url`].
    pub fn new_http(url: reqwest::Url, cache_size: usize) -> Self {
        let inner = RootProvider::new_http(url);
        Self::new(inner, cache_size)
    }

    /// Hides L1 blocks with fewer than `depth` confirmations from forward traversal.
    ///
    /// Reset and hash-based lookups are unaffected. Configure only the online derivation provider,
    /// so unsafe-head recovery and sequencing keep their own L1 views.
    pub fn with_confirmation_depth(
        mut self,
        depth: u64,
        head: watch::Receiver<Option<BlockInfo>>,
    ) -> Self {
        self.confirmation_depth = Some(ConfirmationDepth { depth, head });
        self
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

    /// Fetches raw consensus-encoded receipts and decodes their EIP-2718 envelopes.
    async fn raw_receipts_by_hash(
        &mut self,
        hash: B256,
    ) -> Result<Vec<ReceiptEnvelope>, AlloyChainProviderError> {
        let raw_receipts: Option<Vec<Bytes>> =
            self.inner.client().request("debug_getRawReceipts", [hash]).await?;
        let raw_receipts = raw_receipts.ok_or(AlloyChainProviderError::RawReceiptsNull(hash))?;

        // Geth returns an empty array both for an unknown hash and for an existing block with no
        // transactions. Resolve that ambiguity before accepting and caching an empty result.
        if raw_receipts.is_empty() {
            self.verify_empty_raw_receipts(hash).await?;
        }

        let receipts = raw_receipts
            .into_iter()
            .enumerate()
            .map(|(index, raw)| {
                ReceiptEnvelope::decode_2718_exact(raw.as_ref()).map_err(|source| {
                    AlloyChainProviderError::RawReceiptDecode { hash, index, source }
                })
            })
            .collect::<Result<Vec<_>, _>>()?;
        if !self.trust_rpc {
            self.verify_receipts(hash, &receipts).await?;
        }
        Ok(receipts)
    }

    /// Verifies that an empty raw receipt response belongs to an existing empty block.
    async fn verify_empty_raw_receipts(
        &mut self,
        hash: B256,
    ) -> Result<(), AlloyChainProviderError> {
        let header = self.header_by_hash(hash).await?;
        if !header.transaction_root_is_empty() {
            return Err(AlloyChainProviderError::EmptyRawReceiptsForNonEmptyBlock {
                hash,
                transactions_root: header.transactions_root,
            });
        }
        Ok(())
    }

    /// Fetches JSON receipts through the standard Ethereum RPC method.
    async fn json_receipts_by_hash(
        &mut self,
        hash: B256,
    ) -> Result<Vec<ReceiptEnvelope>, AlloyChainProviderError> {
        let receipts = self
            .inner
            .get_block_receipts(hash.into())
            .await?
            .ok_or(AlloyChainProviderError::BlockNotFound(hash.into()))?;
        let receipts = receipts
            .into_iter()
            .map(|receipt| receipt.inner.into_primitives_receipt())
            .collect::<Vec<_>>();
        if !self.trust_rpc {
            self.verify_receipts(hash, &receipts).await?;
        }
        Ok(receipts)
    }

    /// Falls back to `eth_getBlockReceipts` after `debug_getRawReceipts` fails.
    async fn fallback_receipts_by_hash(
        &mut self,
        hash: B256,
        raw_error: AlloyChainProviderError,
    ) -> Result<Vec<ReceiptEnvelope>, AlloyChainProviderError> {
        let method_not_found = raw_error.is_method_not_found();
        let newly_disabled =
            method_not_found && self.raw_receipts_supported.swap(false, Ordering::SeqCst);

        if newly_disabled {
            tracing::warn!(
                target: "l1_provider",
                %hash,
                error = %raw_error,
                "L1 RPC returned MethodNotFound for debug_getRawReceipts; switching to \
                 eth_getBlockReceipts and disabling debug_getRawReceipts for all future receipt \
                 requests"
            );
        } else if !method_not_found {
            tracing::warn!(
                target: "l1_provider",
                %hash,
                error = %raw_error,
                "debug_getRawReceipts failed; trying eth_getBlockReceipts for this L1 block while \
                 leaving debug_getRawReceipts enabled for future requests"
            );
        }

        match self.json_receipts_by_hash(hash).await {
            Ok(receipts) => {
                tracing::info!(
                    target: "l1_provider",
                    %hash,
                    raw_receipts_disabled = !self.raw_receipts_supported.load(Ordering::SeqCst),
                    "eth_getBlockReceipts fallback succeeded; derivation will continue"
                );
                Ok(receipts)
            }
            Err(fallback_error) => {
                let raw_receipts_disabled = !self.raw_receipts_supported.load(Ordering::SeqCst);
                if fallback_error.is_hash_block_not_found() {
                    tracing::warn!(
                        target: "l1_provider",
                        %hash,
                        raw_error = %raw_error,
                        fallback_error = %fallback_error,
                        raw_receipts_disabled,
                        "debug_getRawReceipts failed and eth_getBlockReceipts reported that the \
                         L1 block is missing; derivation will reset to recover from a possible L1 \
                         reorg"
                    );
                } else {
                    tracing::warn!(
                        target: "l1_provider",
                        %hash,
                        raw_error = %raw_error,
                        fallback_error = %fallback_error,
                        raw_receipts_disabled,
                        "Both debug_getRawReceipts and eth_getBlockReceipts failed; derivation \
                         will pause and retry receipt fetching"
                    );
                }
                Err(AlloyChainProviderError::ReceiptMethodsFailed {
                    raw: Box::new(raw_error),
                    fallback: Box::new(fallback_error),
                })
            }
        }
    }

    /// Fetches receipts directly through the sticky `eth_getBlockReceipts` fallback.
    async fn sticky_fallback_receipts_by_hash(
        &mut self,
        hash: B256,
    ) -> Result<Vec<ReceiptEnvelope>, AlloyChainProviderError> {
        self.json_receipts_by_hash(hash).await.inspect_err(|error| {
            if error.is_hash_block_not_found() {
                tracing::warn!(
                    target: "l1_provider",
                    %hash,
                    error = %error,
                    "eth_getBlockReceipts reported that the L1 block is missing while \
                     debug_getRawReceipts is disabled; derivation will reset to recover from a \
                     possible L1 reorg"
                );
            } else {
                tracing::warn!(
                    target: "l1_provider",
                    %hash,
                    error = %error,
                    "eth_getBlockReceipts failed while debug_getRawReceipts is disabled; \
                     derivation will pause and retry receipt fetching"
                );
            }
        })
    }

    /// Keep the EIP-2718 type and bloom until the consensus receipt root has been checked.
    async fn verify_receipts(
        &mut self,
        hash: B256,
        receipts: &[ReceiptEnvelope],
    ) -> Result<(), AlloyChainProviderError> {
        let header = self.header_by_hash(hash).await?;
        verify::receipts(&header, receipts).map_err(RpcError::local_usage)?;
        Ok(())
    }
}

/// An error for the [`AlloyChainProvider`].
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
    /// `debug_getRawReceipts` returned null for an existing or unknown block.
    #[error("debug_getRawReceipts returned null for L1 block {0}")]
    RawReceiptsNull(B256),
    /// A raw receipt failed EIP-2718 decoding.
    #[error("Failed to decode raw receipt {index} for L1 block {hash}: {source}")]
    RawReceiptDecode {
        /// The requested L1 block hash.
        hash: B256,
        /// The index of the invalid receipt in the RPC response.
        index: usize,
        /// The concrete EIP-2718 decoding error.
        #[source]
        source: Eip2718Error,
    },
    /// An empty raw receipt response was returned for a block that contains transactions.
    #[error(
        "debug_getRawReceipts returned no receipts for non-empty L1 block {hash} \
         (transactions root: {transactions_root})"
    )]
    EmptyRawReceiptsForNonEmptyBlock {
        /// The requested L1 block hash.
        hash: B256,
        /// The transaction root in the block header.
        transactions_root: B256,
    },
    /// Both receipt RPC methods failed.
    #[error("debug_getRawReceipts failed: {raw}; eth_getBlockReceipts fallback failed: {fallback}")]
    ReceiptMethodsFailed {
        /// The error returned by `debug_getRawReceipts`.
        raw: Box<Self>,
        /// The error returned by the `eth_getBlockReceipts` fallback.
        fallback: Box<Self>,
    },
}

impl AlloyChainProviderError {
    /// Returns whether this is the JSON-RPC `MethodNotFound` response.
    fn is_method_not_found(&self) -> bool {
        const METHOD_NOT_FOUND: i64 = -32601;

        let Self::Transport(error) = self else {
            return false;
        };
        let error_code =
            error.as_error_resp().map(|payload| payload.code).or_else(|| match error {
                // Some RPC services return JSON-RPC errors with a non-successful HTTP status, which
                // Alloy exposes as an HTTP transport error rather than an `ErrorResp`.
                RpcError::Transport(error) => {
                    error.as_http_error().and_then(|error| Self::error_code_from_body(&error.body))
                }
                // Also recognize an exact error code in a malformed JSON-RPC response. Alloy
                // normally promotes these to `ErrorResp`, but a response with an
                // invalid or missing ID may remain a deserialization error.
                RpcError::DeserError { text, .. } => Self::error_code_from_body(text),
                _ => None,
            });
        error_code == Some(METHOD_NOT_FOUND)
    }

    /// Extracts a JSON-RPC error code from either an error payload or a full response body.
    fn error_code_from_body(body: &str) -> Option<i64> {
        let value: serde_json::Value = serde_json::from_str(body).ok()?;
        value.get("error").unwrap_or(&value).get("code").and_then(serde_json::Value::as_i64)
    }

    /// Returns whether the standard receipt method confirmed that a hash is no longer available.
    const fn is_hash_block_not_found(&self) -> bool {
        matches!(self, Self::BlockNotFound(BlockId::Hash(_)))
    }
}

impl From<AlloyChainProviderError> for PipelineErrorKind {
    fn from(e: AlloyChainProviderError) -> Self {
        match e {
            AlloyChainProviderError::Transport(e) => {
                Self::Temporary(PipelineError::Provider(format!("Transport error: {e}")))
            }
            AlloyChainProviderError::BlockNotFound(id) => {
                // A hash-based lookup returning not-found means the block was reorged
                // out of the chain — retrying will never succeed, so reset.
                // A number-based lookup returning not-found means the next L1 block
                // hasn't been produced yet — this is transient, so Temporary.
                match id {
                    BlockId::Hash(_) => ResetError::BlockNotFound(id).reset(),
                    BlockId::Number(_) => Self::Temporary(PipelineError::Provider(format!(
                        "L1 Block not found: {id}"
                    ))),
                }
            }
            AlloyChainProviderError::ReceiptsConversion(hash) => {
                Self::Temporary(PipelineError::Provider(format!(
                    "Failed to convert RPC receipts into consensus receipts for L1 block {hash}"
                )))
            }
            error @ (AlloyChainProviderError::RawReceiptsNull(_) |
            AlloyChainProviderError::RawReceiptDecode { .. } |
            AlloyChainProviderError::EmptyRawReceiptsForNonEmptyBlock { .. }) => {
                Self::Temporary(PipelineError::Provider(error.to_string()))
            }
            AlloyChainProviderError::ReceiptMethodsFailed { raw, fallback } => {
                let message = format!(
                    "debug_getRawReceipts failed: {raw}; eth_getBlockReceipts fallback failed: \
                     {fallback}"
                );

                // The standard method determines the derivation impact. A missing hash confirms
                // the possible reorg signaled by the preferred method and requires a reset. Every
                // other failure remains temporary so that receipt fetching can be retried.
                match *fallback {
                    AlloyChainProviderError::BlockNotFound(id @ BlockId::Hash(_)) => {
                        ResetError::BlockNotFound(id).reset()
                    }
                    _ => Self::Temporary(PipelineError::Provider(message)),
                }
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

        if !self.trust_rpc {
            verify::header_hash(&header, hash).map_err(RpcError::local_usage)?;
        }

        self.header_by_hash_cache.put(hash, header.clone());

        kona_macros::inc!(gauge, Metrics::CACHE_ENTRIES, "cache" => "header_by_hash");

        Ok(header)
    }

    async fn block_info_by_number(&mut self, number: u64) -> Result<BlockInfo, Self::Error> {
        if self.confirmation_depth.as_ref().is_some_and(|depth| !depth.allows(number)) {
            // A number-based not-found is temporary: traversal waits for more confirmations.
            return Err(AlloyChainProviderError::BlockNotFound(number.into()));
        }
        self.block_info_by_number_for_reset(number).await
    }

    async fn block_info_by_number_for_reset(
        &mut self,
        number: u64,
    ) -> Result<BlockInfo, Self::Error> {
        kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_RPC_CALLS, "method" => "block_by_number");

        let block = self
            .inner
            .get_block_by_number(number.into())
            .await
            .inspect_err(|_e| {
                kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_RPC_ERRORS, "method" => "block_by_number");
            })?
            .ok_or(AlloyChainProviderError::BlockNotFound(number.into()))?;
        let claimed_hash = block.header.hash;
        let header = block.header.into_consensus();
        if !self.trust_rpc {
            verify::header_hash(&header, claimed_hash).map_err(RpcError::local_usage)?;
            verify::block_number(&header, number).map_err(RpcError::local_usage)?;
        }

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

        let receipts = if self.raw_receipts_supported.load(Ordering::SeqCst) {
            match self.raw_receipts_by_hash(hash).await {
                Ok(receipts) => Ok(receipts),
                Err(error) if error.is_hash_block_not_found() => {
                    tracing::warn!(
                        target: "l1_provider",
                        %hash,
                        error = %error,
                        "debug_getRawReceipts returned no receipts and the L1 block no longer \
                         exists; derivation will reset to recover from a possible L1 reorg"
                    );
                    Err(error)
                }
                Err(raw_error) => self.fallback_receipts_by_hash(hash, raw_error).await,
            }
        } else {
            self.sticky_fallback_receipts_by_hash(hash).await
        }
        .inspect_err(|_e| {
            kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_RPC_ERRORS, "method" => "receipts_by_hash");
        })?;

        let receipts = receipts.into_iter().map(ReceiptEnvelope::into_receipt).collect::<Vec<_>>();
        self.receipts_by_hash_cache.put(hash, receipts.clone());

        kona_macros::inc!(gauge, Metrics::CACHE_ENTRIES, "cache" => "receipts_by_hash");

        Ok(receipts)
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

        if !self.trust_rpc {
            verify::header_hash(&block.header, hash).map_err(RpcError::local_usage)?;
            verify::transactions(&block.header, &block.body.transactions)
                .map_err(RpcError::local_usage)?;
        }

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

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::{Eip658Value, ReceiptWithBloom};
    use alloy_eips::eip2718::Encodable2718;
    use alloy_primitives::{Address, Bloom, Log};
    use alloy_rpc_client::RpcClient;
    use alloy_rpc_types_eth::{Block as RpcBlock, BlockTransactions, Header as RpcHeader};
    use alloy_transport::mock::Asserter;
    use httpmock::prelude::*;

    type EthereumBlock =
        <alloy_provider::network::Ethereum as alloy_provider::Network>::BlockResponse;

    /// Builds an RPC block with the requested transaction metadata.
    fn rpc_block(
        hash: B256,
        transaction_hashes: Vec<B256>,
        transactions_root: B256,
    ) -> EthereumBlock {
        let header = Header { transactions_root, ..Default::default() };
        let mut rpc_header = RpcHeader::new(header);
        // Tests use a trusted provider, so the reported hash may be set independently of the test
        // header's computed hash.
        rpc_header.hash = hash;
        RpcBlock::new(rpc_header, BlockTransactions::Hashes(transaction_hashes))
    }

    /// Creates a chain provider backed by a FIFO mock RPC transport.
    fn mocked_provider(asserter: Asserter) -> AlloyChainProvider {
        AlloyChainProvider::new(RootProvider::new(RpcClient::mocked(asserter)), 8)
    }

    fn observed_head(number: u64) -> BlockInfo {
        BlockInfo { number, hash: B256::repeat_byte(0xaa), ..Default::default() }
    }

    #[test]
    fn confirmation_depth_boundaries() {
        let (_head_tx, head_rx) = watch::channel(Some(observed_head(10)));
        let policy = ConfirmationDepth { depth: 4, head: head_rx };
        assert!(policy.allows(0), "L1 genesis remains visible");
        assert!(policy.allows(6), "the exact confirmation boundary is visible");
        assert!(!policy.allows(7));
        assert!(!policy.allows(10));
        assert!(!policy.allows(u64::MAX), "addition must not overflow to an allowed height");
    }

    #[test]
    fn confirmation_depth_unknown_head_and_zero_depth() {
        let (head_tx, head_rx) = watch::channel(None);
        let policy = ConfirmationDepth { depth: 4, head: head_rx.clone() };
        assert!(policy.allows(100), "op-node bypasses the limit before its first head signal");
        head_tx.send_replace(Some(BlockInfo::default()));
        assert!(policy.allows(100), "an empty head matches op-node's zero block reference");
        head_tx.send_replace(Some(observed_head(10)));
        assert!(!policy.allows(100));
        let no_delay = ConfirmationDepth { depth: 0, head: head_rx };
        assert!(no_delay.allows(u64::MAX), "zero depth ignores the observed L1 head");
    }

    #[tokio::test]
    async fn confirmation_depth_tracks_head_regression_and_clones() {
        let asserter = Asserter::new();
        let block = rpc_block(B256::ZERO, Vec::new(), alloy_consensus::EMPTY_ROOT_HASH);
        asserter.push_success(&Some(block.clone()));
        asserter.push_success(&Some(block));
        let (head_tx, head_rx) = watch::channel(Some(observed_head(10)));
        let mut provider = mocked_provider(asserter.clone()).with_confirmation_depth(4, head_rx);
        let mut cloned = provider.clone();
        provider.block_info_by_number(6).await.unwrap();
        head_tx.send_replace(Some(observed_head(9)));
        let error = cloned.block_info_by_number(6).await.unwrap_err();
        assert!(matches!(PipelineErrorKind::from(error), PipelineErrorKind::Temporary(_)));
        assert_eq!(asserter.read_q().len(), 1, "withheld blocks must not reach the RPC");
        head_tx.send_replace(Some(observed_head(10)));
        cloned.block_info_by_number(6).await.unwrap();
        assert!(asserter.read_q().is_empty());
    }

    #[tokio::test]
    async fn confirmation_depth_leaves_hash_reads_unrestricted() {
        let asserter = Asserter::new();
        let block = rpc_block(B256::ZERO, Vec::new(), alloy_consensus::EMPTY_ROOT_HASH);
        asserter.push_success(&Some(block));
        let (_head_tx, head_rx) = watch::channel(Some(observed_head(10)));
        let mut provider = mocked_provider(asserter.clone()).with_confirmation_depth(4, head_rx);
        provider.header_by_hash(B256::repeat_byte(0xbb)).await.unwrap();
        assert!(asserter.read_q().is_empty());
    }

    #[tokio::test]
    async fn confirmation_depth_does_not_block_reset_within_the_window() {
        use kona_derive::{OriginAdvancer, OriginProvider, PollingTraversal, Stage};
        use kona_genesis::{RollupConfig, SystemConfig};

        let asserter = Asserter::new();
        let block: EthereumBlock = RpcBlock::new(
            RpcHeader::new(Header { number: 1, ..Default::default() }),
            BlockTransactions::Hashes(Vec::new()),
        );
        asserter.push_success(&Some(block.clone()));
        let (_head_tx, head_rx) = watch::channel(Some(observed_head(2)));
        let provider = mocked_provider(asserter.clone()).with_confirmation_depth(4, head_rx);
        let mut traversal = PollingTraversal::new(provider, Arc::new(RollupConfig::default()));
        // Reset re-resolves the canonical block by number, even when its old hash is stale.
        traversal
            .reset(alloy_eips::BlockNumHash::new(1, B256::ZERO), SystemConfig::default())
            .await
            .unwrap();
        assert_eq!(traversal.origin().unwrap().hash, block.header.hash);
        assert!(matches!(traversal.advance_origin().await, Err(PipelineErrorKind::Temporary(_))));
        assert!(asserter.read_q().is_empty());
    }

    /// A one-block tip replacement is invisible to delayed traversal. At zero depth, consuming
    /// the old tip and then its replacement's child produces the ordinary reorg reset.
    #[tokio::test]
    async fn confirmation_depth_shields_traversal_from_shallow_reorg() {
        use kona_derive::{OriginAdvancer, OriginProvider, PollingTraversal};
        use kona_genesis::RollupConfig;

        let parent = BlockInfo { number: 13, hash: B256::repeat_byte(0x13), ..Default::default() };
        let make_block = |number, parent_hash, timestamp| -> EthereumBlock {
            RpcBlock::new(
                RpcHeader::new(Header { number, parent_hash, timestamp, ..Default::default() }),
                BlockTransactions::Hashes(Vec::new()),
            )
        };
        let old_tip = make_block(14, parent.hash, 14);
        let replacement = make_block(14, parent.hash, 15);
        let child = make_block(15, replacement.header.hash, 16);
        let receipt =
            ReceiptEnvelope::Legacy(ReceiptWithBloom::new(Receipt::default(), Bloom::ZERO));
        let raw_receipts = vec![Bytes::from(receipt.encoded_2718())];

        for depth in [0, 1] {
            let asserter = Asserter::new();
            let (head_tx, head_rx) = watch::channel(Some(observed_head(14)));
            if depth == 0 {
                asserter.push_success(&Some(old_tip.clone()));
                asserter.push_success(&raw_receipts);
                asserter.push_success(&Some(child.clone()));
            } else {
                asserter.push_success(&Some(replacement.clone()));
                asserter.push_success(&raw_receipts);
            }
            let provider =
                mocked_provider(asserter.clone()).with_confirmation_depth(depth, head_rx);
            let mut traversal = PollingTraversal::new(provider, Arc::new(RollupConfig::default()));
            traversal.block = Some(parent);
            if depth == 0 {
                traversal.advance_origin().await.unwrap();
                assert_eq!(traversal.origin().unwrap().hash, old_tip.header.hash);
            } else {
                assert!(matches!(
                    traversal.advance_origin().await,
                    Err(PipelineErrorKind::Temporary(_))
                ));
                assert_eq!(traversal.origin(), Some(parent));
            }
            head_tx.send_replace(Some(observed_head(15)));
            let result = traversal.advance_origin().await;
            if depth == 0 {
                assert!(matches!(
                    result,
                    Err(PipelineErrorKind::Reset(ResetError::ReorgDetected(..)))
                ));
            } else {
                result.unwrap();
                assert_eq!(traversal.origin().unwrap().hash, replacement.header.hash);
            }
            assert!(asserter.read_q().is_empty());
        }
    }

    #[test]
    fn test_from_alloy_chain_provider_error() {
        // Transport errors are transient — retry makes sense.
        let transport_err =
            AlloyChainProviderError::Transport(alloy_transport::RpcError::Transport(
                alloy_transport::TransportErrorKind::Custom("timeout".into()),
            ));
        let kind: PipelineErrorKind = transport_err.into();
        assert!(matches!(kind, PipelineErrorKind::Temporary(_)));

        // ReceiptsConversion is a transient decode failure.
        let kind: PipelineErrorKind =
            AlloyChainProviderError::ReceiptsConversion(Default::default()).into();
        assert!(matches!(kind, PipelineErrorKind::Temporary(_)));

        // Hash-based BlockNotFound: the block was reorged out. Retrying will never succeed
        // — the pipeline must reset. Without this, the safe head stalls on L1 reorgs.
        let kind: PipelineErrorKind =
            AlloyChainProviderError::BlockNotFound(B256::default().into()).into();
        assert!(
            matches!(kind, PipelineErrorKind::Reset(_)),
            "hash-based BlockNotFound must map to Reset (block reorged out)"
        );

        // Number-based BlockNotFound: the next L1 block hasn't been mined yet. This is
        // transient — the pipeline must wait, not reset.
        let kind: PipelineErrorKind = AlloyChainProviderError::BlockNotFound(0u64.into()).into();
        assert!(
            matches!(kind, PipelineErrorKind::Temporary(_)),
            "number-based BlockNotFound must stay Temporary (block not yet produced)"
        );
    }

    #[test]
    fn receipt_fallback_determines_derivation_impact() {
        let missing_hash = B256::repeat_byte(0xaa);
        let temporary = AlloyChainProviderError::ReceiptMethodsFailed {
            raw: Box::new(AlloyChainProviderError::BlockNotFound(missing_hash.into())),
            fallback: Box::new(AlloyChainProviderError::Transport(RpcError::Transport(
                TransportErrorKind::Custom("fallback timeout".into()),
            ))),
        };
        let message = temporary.to_string();
        assert!(message.contains("Block not found"));
        assert!(message.contains("fallback timeout"));
        assert!(matches!(PipelineErrorKind::from(temporary), PipelineErrorKind::Temporary(_)));

        let reset = AlloyChainProviderError::ReceiptMethodsFailed {
            raw: Box::new(AlloyChainProviderError::Transport(RpcError::Transport(
                TransportErrorKind::Custom("raw timeout".into()),
            ))),
            fallback: Box::new(AlloyChainProviderError::BlockNotFound(missing_hash.into())),
        };
        assert!(matches!(
            PipelineErrorKind::from(reset),
            PipelineErrorKind::Reset(ResetError::BlockNotFound(id)) if id == missing_hash.into()
        ));
    }

    #[tokio::test]
    async fn receipts_by_hash_prefers_raw_receipts() {
        let expected = Receipt {
            status: Eip658Value::Eip658(true),
            cumulative_gas_used: 21_000,
            logs: Vec::new(),
        };
        let envelope =
            ReceiptEnvelope::Eip1559(ReceiptWithBloom::new(expected.clone(), Bloom::ZERO));
        let raw_receipt = Bytes::from(envelope.encoded_2718());
        let response = serde_json::json!({
            "jsonrpc": "2.0",
            "id": 0,
            "result": [raw_receipt],
        });

        let server = MockServer::start();
        let raw_mock = server.mock(|when, then| {
            when.method(POST).body_includes("debug_getRawReceipts");
            then.status(200).header("content-type", "application/json").json_body(response);
        });
        let fallback_mock = server.mock(|when, then| {
            when.method(POST).body_includes("eth_getBlockReceipts");
            then.status(500);
        });
        let mut provider = AlloyChainProvider::new_http(server.base_url().parse().unwrap(), 8);

        assert_eq!(provider.receipts_by_hash(B256::repeat_byte(0xaa)).await.unwrap(), [expected]);
        raw_mock.assert();
        assert_eq!(fallback_mock.calls(), 0);
    }

    #[tokio::test]
    async fn empty_raw_receipts_for_unknown_hash_reset_without_fallback() {
        let asserter = Asserter::new();
        asserter.push_success(&Vec::<Bytes>::new());
        asserter.push_success(&Option::<EthereumBlock>::None);
        let mut provider = mocked_provider(asserter.clone());
        let hash = B256::repeat_byte(0xab);

        let error = provider.receipts_by_hash(hash).await.unwrap_err();
        assert!(matches!(
            &error,
            AlloyChainProviderError::BlockNotFound(id) if *id == hash.into()
        ));
        assert!(matches!(
            PipelineErrorKind::from(error),
            PipelineErrorKind::Reset(ResetError::BlockNotFound(id)) if id == hash.into()
        ));
        assert!(
            asserter.read_q().is_empty(),
            "a confirmed missing block must reset without requesting fallback receipts"
        );
    }

    #[tokio::test]
    async fn empty_raw_receipts_for_existing_empty_block_are_accepted() {
        let hash = B256::repeat_byte(0xac);
        let asserter = Asserter::new();
        asserter.push_success(&Vec::<Bytes>::new());
        asserter.push_success(&Some(rpc_block(hash, Vec::new(), alloy_consensus::EMPTY_ROOT_HASH)));
        let mut provider = mocked_provider(asserter.clone());

        assert!(provider.receipts_by_hash(hash).await.unwrap().is_empty());
        assert!(asserter.read_q().is_empty());
    }

    #[tokio::test]
    async fn empty_raw_receipts_for_nonempty_block_are_rejected() {
        let hash = B256::repeat_byte(0xad);
        let transactions_root = B256::repeat_byte(0x11);
        let asserter = Asserter::new();
        asserter.push_success(&Vec::<Bytes>::new());
        asserter.push_success(&Some(rpc_block(
            hash,
            vec![B256::repeat_byte(0x22)],
            transactions_root,
        )));
        let mut provider = mocked_provider(asserter.clone());

        let error = provider.raw_receipts_by_hash(hash).await.unwrap_err();
        assert!(matches!(
            error,
            AlloyChainProviderError::EmptyRawReceiptsForNonEmptyBlock {
                hash: error_hash,
                transactions_root: error_root,
            } if error_hash == hash && error_root == transactions_root
        ));
        assert!(asserter.read_q().is_empty());
    }

    #[tokio::test]
    async fn raw_receipt_decode_error_preserves_source_and_index() {
        let asserter = Asserter::new();
        asserter.push_success(&vec![Bytes::from_static(&[0x05])]);
        let mut provider = mocked_provider(asserter.clone());
        let hash = B256::repeat_byte(0xae);

        let error = provider.raw_receipts_by_hash(hash).await.unwrap_err();
        match error {
            AlloyChainProviderError::RawReceiptDecode { hash: error_hash, index, source } => {
                assert_eq!(error_hash, hash);
                assert_eq!(index, 0);
                assert!(matches!(source, Eip2718Error::RlpError(_)));
            }
            other => panic!("expected typed raw receipt decode error, got {other:?}"),
        }
        assert!(asserter.read_q().is_empty());
    }

    #[tokio::test]
    async fn method_not_found_makes_fallback_sticky_across_clones() {
        let server = MockServer::start();
        let raw_mock = server.mock(|when, then| {
            when.method(POST).body_includes("debug_getRawReceipts");
            then.status(200).header("content-type", "application/json").json_body(
                serde_json::json!({
                    "jsonrpc": "2.0",
                    "id": 0,
                    "error": { "code": -32601, "message": "Method not found" },
                }),
            );
        });
        let fallback_mock = server.mock(|when, then| {
            when.method(POST).body_includes("eth_getBlockReceipts");
            then.status(200)
                .header("content-type", "application/json")
                .json_body(serde_json::json!({ "jsonrpc": "2.0", "id": 1, "result": [] }));
        });
        let mut provider = AlloyChainProvider::new_http(server.base_url().parse().unwrap(), 8);
        let mut provider_clone = provider.clone();

        assert!(provider.receipts_by_hash(B256::repeat_byte(0xba)).await.unwrap().is_empty());
        assert!(provider_clone.receipts_by_hash(B256::repeat_byte(0xbb)).await.unwrap().is_empty());
        assert_eq!(raw_mock.calls(), 1, "the clone must share the sticky fallback state");
        assert_eq!(fallback_mock.calls(), 2);
        assert!(!provider.raw_receipts_supported.load(Ordering::SeqCst));
        assert!(!provider_clone.raw_receipts_supported.load(Ordering::SeqCst));
    }

    #[tokio::test]
    async fn method_not_found_in_http_error_body_makes_fallback_sticky() {
        let server = MockServer::start();
        let raw_mock = server.mock(|when, then| {
            when.method(POST).body_includes("debug_getRawReceipts");
            then.status(403).header("content-type", "application/json").body(
                r#"{"jsonrpc":"2.0","id":0,"error":{"code":-32601,"message":"Method not found"}}"#,
            );
        });
        let fallback_mock = server.mock(|when, then| {
            when.method(POST).body_includes("eth_getBlockReceipts");
            then.status(200)
                .header("content-type", "application/json")
                .json_body(serde_json::json!({ "jsonrpc": "2.0", "id": 1, "result": [] }));
        });
        let mut provider = AlloyChainProvider::new_http(server.base_url().parse().unwrap(), 8);

        assert!(provider.receipts_by_hash(B256::repeat_byte(0xbc)).await.unwrap().is_empty());
        assert!(provider.receipts_by_hash(B256::repeat_byte(0xbd)).await.unwrap().is_empty());
        assert_eq!(raw_mock.calls(), 1);
        assert_eq!(fallback_mock.calls(), 2);
        assert!(!provider.raw_receipts_supported.load(Ordering::SeqCst));
    }

    #[tokio::test]
    async fn method_not_found_stays_sticky_when_fallback_fails() {
        let server = MockServer::start();
        let raw_mock = server.mock(|when, then| {
            when.method(POST).body_includes("debug_getRawReceipts");
            then.status(200).header("content-type", "application/json").json_body(
                serde_json::json!({
                    "jsonrpc": "2.0",
                    "id": 0,
                    "error": { "code": -32601, "message": "Method not found" },
                }),
            );
        });
        let fallback_mock = server.mock(|when, then| {
            when.method(POST).body_includes("eth_getBlockReceipts");
            then.status(200).header("content-type", "application/json").json_body(
                serde_json::json!({
                    "jsonrpc": "2.0",
                    "id": 1,
                    "error": { "code": -32000, "message": "fallback unavailable" },
                }),
            );
        });
        let mut provider = AlloyChainProvider::new_http(server.base_url().parse().unwrap(), 8);

        let first_error = provider.receipts_by_hash(B256::repeat_byte(0xbc)).await.unwrap_err();
        assert!(matches!(first_error, AlloyChainProviderError::ReceiptMethodsFailed { .. }));
        assert!(matches!(PipelineErrorKind::from(first_error), PipelineErrorKind::Temporary(_)));

        let second_error = provider.receipts_by_hash(B256::repeat_byte(0xbd)).await.unwrap_err();
        assert!(matches!(second_error, AlloyChainProviderError::Transport(_)));
        assert!(matches!(PipelineErrorKind::from(second_error), PipelineErrorKind::Temporary(_)));
        assert_eq!(raw_mock.calls(), 1);
        assert_eq!(fallback_mock.calls(), 2);
        assert!(!provider.raw_receipts_supported.load(Ordering::SeqCst));
    }

    #[tokio::test]
    async fn other_raw_errors_fall_back_without_becoming_sticky() {
        let server = MockServer::start();
        let raw_mock = server.mock(|when, then| {
            when.method(POST).body_includes("debug_getRawReceipts");
            then.status(200).header("content-type", "application/json").json_body(
                serde_json::json!({
                    "jsonrpc": "2.0",
                    "id": 0,
                    "error": { "code": -32000, "message": "temporary raw receipt failure" },
                }),
            );
        });
        let fallback_mock = server.mock(|when, then| {
            when.method(POST).body_includes("eth_getBlockReceipts");
            then.status(200)
                .header("content-type", "application/json")
                .json_body(serde_json::json!({ "jsonrpc": "2.0", "id": 1, "result": [] }));
        });
        let mut provider = AlloyChainProvider::new_http(server.base_url().parse().unwrap(), 8);

        assert!(provider.receipts_by_hash(B256::repeat_byte(0xca)).await.unwrap().is_empty());
        assert!(provider.receipts_by_hash(B256::repeat_byte(0xcb)).await.unwrap().is_empty());
        assert_eq!(raw_mock.calls(), 2);
        assert_eq!(fallback_mock.calls(), 2);
        assert!(provider.raw_receipts_supported.load(Ordering::SeqCst));
    }

    #[tokio::test]
    async fn raw_not_found_uses_successful_fallback() {
        let server = MockServer::start();
        let raw_mock = server.mock(|when, then| {
            when.method(POST).body_includes("debug_getRawReceipts");
            then.status(200)
                .header("content-type", "application/json")
                .json_body(serde_json::json!({ "jsonrpc": "2.0", "id": 0, "result": null }));
        });
        let fallback_mock = server.mock(|when, then| {
            when.method(POST).body_includes("eth_getBlockReceipts");
            then.status(200)
                .header("content-type", "application/json")
                .json_body(serde_json::json!({ "jsonrpc": "2.0", "id": 1, "result": [] }));
        });
        let mut provider = AlloyChainProvider::new_http(server.base_url().parse().unwrap(), 8);

        assert!(provider.receipts_by_hash(B256::repeat_byte(0xda)).await.unwrap().is_empty());
        raw_mock.assert();
        fallback_mock.assert();
        assert!(provider.raw_receipts_supported.load(Ordering::SeqCst));
    }

    #[tokio::test]
    async fn both_method_errors_are_temporary_and_preserve_both_causes() {
        let server = MockServer::start();
        let raw_mock = server.mock(|when, then| {
            when.method(POST).body_includes("debug_getRawReceipts");
            then.status(200).header("content-type", "application/json").json_body(
                serde_json::json!({
                    "jsonrpc": "2.0",
                    "id": 0,
                    "error": { "code": -32000, "message": "raw receipts unavailable" },
                }),
            );
        });
        let fallback_mock = server.mock(|when, then| {
            when.method(POST).body_includes("eth_getBlockReceipts");
            then.status(200).header("content-type", "application/json").json_body(
                serde_json::json!({
                    "jsonrpc": "2.0",
                    "id": 1,
                    "error": { "code": -32001, "message": "JSON receipts unavailable" },
                }),
            );
        });
        let mut provider = AlloyChainProvider::new_http(server.base_url().parse().unwrap(), 8);

        let error = provider.receipts_by_hash(B256::repeat_byte(0xea)).await.unwrap_err();
        assert!(matches!(error, AlloyChainProviderError::ReceiptMethodsFailed { .. }));
        assert!(error.to_string().contains("raw receipts unavailable"));
        assert!(error.to_string().contains("JSON receipts unavailable"));

        let pipeline_error = PipelineErrorKind::from(error);
        assert!(matches!(pipeline_error, PipelineErrorKind::Temporary(_)));
        assert!(pipeline_error.to_string().contains("raw receipts unavailable"));
        assert!(pipeline_error.to_string().contains("JSON receipts unavailable"));
        raw_mock.assert();
        fallback_mock.assert();
    }

    #[tokio::test]
    async fn fallback_not_found_preserves_reorg_signal() {
        let server = MockServer::start();
        let raw_mock = server.mock(|when, then| {
            when.method(POST).body_includes("debug_getRawReceipts");
            then.status(200).header("content-type", "application/json").json_body(
                serde_json::json!({
                    "jsonrpc": "2.0",
                    "id": 0,
                    "error": { "code": -32000, "message": "raw receipts unavailable" },
                }),
            );
        });
        let fallback_mock = server.mock(|when, then| {
            when.method(POST).body_includes("eth_getBlockReceipts");
            then.status(200)
                .header("content-type", "application/json")
                .json_body(serde_json::json!({ "jsonrpc": "2.0", "id": 1, "result": null }));
        });
        let mut provider = AlloyChainProvider::new_http(server.base_url().parse().unwrap(), 8);
        let hash = B256::repeat_byte(0xfa);

        let error = provider.receipts_by_hash(hash).await.unwrap_err();
        assert!(matches!(error, AlloyChainProviderError::ReceiptMethodsFailed { .. }));
        assert!(matches!(
            PipelineErrorKind::from(error),
            PipelineErrorKind::Reset(ResetError::BlockNotFound(id)) if id == hash.into()
        ));
        raw_mock.assert();
        fallback_mock.assert();
    }

    #[tokio::test]
    async fn fallback_converts_json_receipts_in_order() {
        let block_hash = B256::repeat_byte(0xdb);
        let tx_hashes = [B256::repeat_byte(0x01), B256::repeat_byte(0x02), B256::repeat_byte(0x03)];
        let logs = [
            Log::new_unchecked(
                Address::repeat_byte(0x10),
                vec![B256::repeat_byte(0x20), B256::repeat_byte(0x21)],
                Bytes::from_static(&[0x01, 0x02]),
            ),
            Log::new_unchecked(
                Address::repeat_byte(0x11),
                vec![B256::repeat_byte(0x22)],
                Bytes::new(),
            ),
            Log::new_unchecked(Address::repeat_byte(0x12), Vec::new(), Bytes::from_static(&[0xff])),
        ];
        let expected = vec![
            Receipt {
                status: Eip658Value::Eip658(true),
                cumulative_gas_used: 46_000,
                logs: logs[..2].to_vec(),
            },
            Receipt {
                status: Eip658Value::Eip658(false),
                cumulative_gas_used: 67_000,
                logs: Vec::new(),
            },
            Receipt {
                status: Eip658Value::Eip658(true),
                cumulative_gas_used: 112_000,
                logs: logs[2..].to_vec(),
            },
        ];

        let mut log_index = 0;
        let json_receipts: Vec<_> = expected
            .iter()
            .zip(["0x2", "0x0", "0x2"])
            .enumerate()
            .map(|(tx_index, (receipt, tx_type))| {
                let json_logs: Vec<_> = receipt
                    .logs
                    .iter()
                    .map(|log| {
                        log_index += 1;
                        serde_json::json!({
                            "address": log.address,
                            "topics": log.topics(),
                            "data": log.data.data,
                            "blockHash": block_hash,
                            "blockNumber": "0x10",
                            "transactionHash": tx_hashes[tx_index],
                            "transactionIndex": format!("{tx_index:#x}"),
                            "logIndex": format!("{:#x}", log_index - 1),
                            "removed": false,
                        })
                    })
                    .collect();
                serde_json::json!({
                    "type": tx_type,
                    "status": if receipt.status.coerce_status() { "0x1" } else { "0x0" },
                    "cumulativeGasUsed": format!("{:#x}", receipt.cumulative_gas_used),
                    "logs": json_logs,
                    "logsBloom": Bloom::ZERO,
                    "transactionHash": tx_hashes[tx_index],
                    "transactionIndex": format!("{tx_index:#x}"),
                    "blockHash": block_hash,
                    "blockNumber": "0x10",
                    "gasUsed": "0x5208",
                    "effectiveGasPrice": "0x3b9aca00",
                    "from": Address::repeat_byte(0x30),
                    "to": Address::repeat_byte(0x31),
                    "contractAddress": null,
                })
            })
            .collect();

        let server = MockServer::start();
        let raw_mock = server.mock(|when, then| {
            when.method(POST).body_includes("debug_getRawReceipts");
            then.status(200).header("content-type", "application/json").json_body(
                serde_json::json!({
                    "jsonrpc": "2.0",
                    "id": 0,
                    "error": { "code": -32000, "message": "raw receipts unavailable" },
                }),
            );
        });
        let fallback_mock = server.mock(|when, then| {
            when.method(POST).body_includes("eth_getBlockReceipts");
            then.status(200).header("content-type", "application/json").json_body(
                serde_json::json!({ "jsonrpc": "2.0", "id": 1, "result": json_receipts }),
            );
        });
        let mut provider = AlloyChainProvider::new_http(server.base_url().parse().unwrap(), 8);

        assert_eq!(provider.receipts_by_hash(block_hash).await.unwrap(), expected);
        raw_mock.assert();
        fallback_mock.assert();
    }
    #[tokio::test]
    async fn untrusted_l1_transactions_are_verified_before_caching() {
        use crate::test_utils::{ethereum_transactions, rpc_block};
        use alloy_consensus::proofs::calculate_transaction_root;
        let transactions = ethereum_transactions();
        let header = Header {
            transactions_root: calculate_transaction_root(&transactions),
            ..Default::default()
        };
        let hash = header.hash_slow();
        let valid = rpc_block(header, &transactions);
        let mut missing = valid.clone();
        missing["transactions"].as_array_mut().unwrap().pop();
        let mut reordered = valid.clone();
        reordered["transactions"].as_array_mut().unwrap().swap(0, 1);
        let mut altered = valid.clone();
        altered["transactions"][0]["nonce"] = serde_json::json!("0x99");
        let mut forged_hash = valid.clone();
        forged_hash["transactions"][1]["hash"] =
            serde_json::to_value(B256::repeat_byte(9)).unwrap();
        for invalid in [&missing, &reordered, &altered, &forged_hash] {
            let asserter = Asserter::new();
            asserter.push_success(invalid);
            asserter.push_success(&valid);
            let mut provider = AlloyChainProvider::new_with_trust(
                RootProvider::new(RpcClient::mocked(asserter)),
                8,
                false,
            );
            let error = provider.block_info_and_transactions_by_hash(hash).await.unwrap_err();
            assert!(matches!(error, AlloyChainProviderError::Transport(_)), "{error:?}");
            assert!(
                error.to_string().contains("root mismatch") ||
                    error.to_string().contains("hash mismatch"),
                "{error:?}"
            );
            assert!(provider.block_info_and_transactions_by_hash_cache.is_empty());
            assert!(matches!(PipelineErrorKind::from(error), PipelineErrorKind::Temporary(_)));
            assert_eq!(
                provider.block_info_and_transactions_by_hash(hash).await.unwrap().1,
                transactions
            );
            assert_eq!(
                provider.block_info_and_transactions_by_hash(hash).await.unwrap().1,
                transactions
            );
        }
    }

    #[tokio::test]
    async fn untrusted_receipts_preserve_types_blooms_and_logs() {
        use crate::test_utils::{json_receipts, receipts, rpc_block};
        use alloy_consensus::proofs::calculate_receipt_root;
        let receipts = receipts();
        let header =
            Header { receipts_root: calculate_receipt_root(&receipts), ..Default::default() };
        let hash = header.hash_slow();
        let expected: Vec<_> =
            receipts.iter().cloned().map(ReceiptEnvelope::into_receipt).collect();
        // Exercise both raw and standard JSON methods; the JSON path stays active after
        // MethodNotFound, and both paths must authenticate the typed consensus encoding.
        for raw_supported in [true, false] {
            let asserter = Asserter::new();
            if raw_supported {
                let raw: Vec<Bytes> = receipts.iter().map(|r| r.encoded_2718().into()).collect();
                asserter.push_success(&raw);
            } else {
                push_method_not_found(&asserter);
                asserter.push_success(&json_receipts(&receipts));
            }
            asserter.push_success(&rpc_block::<TxEnvelope>(header.clone(), &[]));
            let mut provider = AlloyChainProvider::new_with_trust(
                RootProvider::new(RpcClient::mocked(asserter)),
                8,
                false,
            );
            assert_eq!(provider.receipts_by_hash(hash).await.unwrap(), expected);
            assert_eq!(provider.receipts_by_hash(hash).await.unwrap(), expected);
            assert_eq!(provider.raw_receipts_supported.load(Ordering::SeqCst), raw_supported);
        }
    }

    /// A method-not-found response causes subsequent receipt requests to use JSON directly.
    fn push_method_not_found(asserter: &Asserter) {
        asserter.push_failure(
            serde_json::from_value(serde_json::json!({
                "code": -32601, "message": "method not found",
            }))
            .unwrap(),
        );
    }

    #[tokio::test]
    async fn untrusted_json_receipts_reject_tampering_without_caching() {
        use crate::test_utils::{json_receipts, receipts, rpc_block};
        use alloy_consensus::proofs::calculate_receipt_root;
        let receipts = receipts();
        let header =
            Header { receipts_root: calculate_receipt_root(&receipts), ..Default::default() };
        let hash = header.hash_slow();
        let valid = json_receipts(&receipts);
        let mut missing = valid.clone();
        missing.pop();
        let mut reordered = valid.clone();
        reordered.swap(0, 1);
        let mut wrong_type = valid.clone();
        wrong_type[0]["type"] = serde_json::json!("0x2");
        let mut wrong_bloom = valid.clone();
        wrong_bloom[0]["logsBloom"] = serde_json::to_value(Bloom::ZERO).unwrap();
        let mut wrong_logs = valid.clone();
        wrong_logs[0]["logs"][0]["data"] = serde_json::json!("0xdeadbeef");
        let mut wrong_status = valid.clone();
        wrong_status[0]["status"] = serde_json::json!("0x0");
        for invalid in
            [missing, reordered, wrong_type, wrong_bloom, wrong_logs, wrong_status, vec![]]
        {
            let asserter = Asserter::new();
            push_method_not_found(&asserter);
            asserter.push_success(&invalid);
            asserter.push_success(&rpc_block::<TxEnvelope>(header.clone(), &[]));
            asserter.push_success(&valid);
            let mut provider = AlloyChainProvider::new_with_trust(
                RootProvider::new(RpcClient::mocked(asserter)),
                8,
                false,
            );
            let error = provider.receipts_by_hash(hash).await.unwrap_err();
            assert!(error.to_string().contains("Receipts root mismatch"), "{error:?}");
            assert!(provider.receipts_by_hash_cache.is_empty());
            assert!(matches!(PipelineErrorKind::from(error), PipelineErrorKind::Temporary(_)));
            assert_eq!(provider.receipts_by_hash(hash).await.unwrap().len(), receipts.len());
        }
    }

    #[tokio::test]
    async fn invalid_raw_receipts_can_fall_back_to_verified_json() {
        use crate::test_utils::{json_receipts, receipts, rpc_block};
        use alloy_consensus::proofs::calculate_receipt_root;
        let receipts = receipts();
        let header =
            Header { receipts_root: calculate_receipt_root(&receipts), ..Default::default() };
        let hash = header.hash_slow();
        let raw: Vec<Bytes> = receipts.iter().rev().map(|r| r.encoded_2718().into()).collect();
        let asserter = Asserter::new();
        asserter.push_success(&raw);
        asserter.push_success(&rpc_block::<TxEnvelope>(header, &[]));
        asserter.push_success(&json_receipts(&receipts));
        let mut provider = AlloyChainProvider::new_with_trust(
            RootProvider::new(RpcClient::mocked(asserter)),
            8,
            false,
        );
        assert_eq!(provider.receipts_by_hash(hash).await.unwrap().len(), receipts.len());
        assert!(provider.raw_receipts_supported.load(Ordering::SeqCst));
    }

    #[tokio::test]
    async fn invalid_raw_and_json_receipts_are_not_cached() {
        use crate::test_utils::{json_receipts, receipts, rpc_block};
        use alloy_consensus::proofs::calculate_receipt_root;
        let receipts = receipts();
        let header =
            Header { receipts_root: calculate_receipt_root(&receipts), ..Default::default() };
        let hash = header.hash_slow();
        let raw: Vec<Bytes> = receipts.iter().rev().map(|r| r.encoded_2718().into()).collect();
        let mut invalid_json = json_receipts(&receipts);
        invalid_json.reverse();
        let asserter = Asserter::new();
        asserter.push_success(&raw);
        asserter.push_success(&rpc_block::<TxEnvelope>(header, &[]));
        asserter.push_success(&invalid_json);
        let valid_raw: Vec<Bytes> = receipts.iter().map(|r| r.encoded_2718().into()).collect();
        asserter.push_success(&valid_raw);
        let mut provider = AlloyChainProvider::new_with_trust(
            RootProvider::new(RpcClient::mocked(asserter)),
            8,
            false,
        );
        let error = provider.receipts_by_hash(hash).await.unwrap_err();
        let AlloyChainProviderError::ReceiptMethodsFailed { raw, fallback } = error else {
            panic!("expected both receipt methods to fail: {error:?}")
        };
        assert!(raw.to_string().contains("Receipts root mismatch"), "{raw:?}");
        assert!(fallback.to_string().contains("Receipts root mismatch"), "{fallback:?}");
        assert!(provider.receipts_by_hash_cache.is_empty());
        assert_eq!(provider.receipts_by_hash(hash).await.unwrap().len(), receipts.len());
    }

    #[tokio::test]
    async fn untrusted_empty_receipts_must_match_the_header() {
        use crate::test_utils::rpc_block;
        for receipts_root in [alloy_consensus::EMPTY_ROOT_HASH, B256::repeat_byte(1)] {
            let header = Header { receipts_root, ..Default::default() };
            let hash = header.hash_slow();
            let asserter = Asserter::new();
            asserter.push_success(&Vec::<Bytes>::new());
            asserter.push_success(&rpc_block::<TxEnvelope>(header, &[]));
            // If the raw root is invalid, the JSON fallback must reject the empty set as well.
            if receipts_root != alloy_consensus::EMPTY_ROOT_HASH {
                asserter.push_success(&Vec::<serde_json::Value>::new());
            }
            let mut provider = AlloyChainProvider::new_with_trust(
                RootProvider::new(RpcClient::mocked(asserter)),
                8,
                false,
            );
            let result = provider.receipts_by_hash(hash).await;
            assert_eq!(result.is_ok(), receipts_root == alloy_consensus::EMPTY_ROOT_HASH);
        }
    }

    #[tokio::test]
    async fn untrusted_l1_number_lookup_checks_header_hash_and_number() {
        use crate::test_utils::rpc_block;
        let valid = rpc_block::<TxEnvelope>(Header { number: 42, ..Default::default() }, &[]);
        let mut forged = valid.clone();
        forged["hash"] = serde_json::to_value(B256::repeat_byte(1)).unwrap();
        for (response, number, expected_ok) in
            [(&valid, 42, true), (&forged, 42, false), (&valid, 43, false)]
        {
            let asserter = Asserter::new();
            asserter.push_success(response);
            let mut provider = AlloyChainProvider::new_with_trust(
                RootProvider::new(RpcClient::mocked(asserter)),
                8,
                false,
            );
            assert_eq!(provider.block_info_by_number(number).await.is_ok(), expected_ok);
        }
    }

    #[tokio::test]
    async fn receipt_verification_authenticates_the_header() {
        use crate::test_utils::{json_receipts, receipts, rpc_block};
        use alloy_consensus::proofs::calculate_receipt_root;
        let receipts = receipts();
        let header =
            Header { receipts_root: calculate_receipt_root(&receipts), ..Default::default() };
        let requested = B256::repeat_byte(0xab);
        let mut block = rpc_block::<TxEnvelope>(header, &[]);
        block["hash"] = serde_json::to_value(requested).unwrap();
        let asserter = Asserter::new();
        push_method_not_found(&asserter);
        asserter.push_success(&json_receipts(&receipts));
        asserter.push_success(&block);
        let mut provider = AlloyChainProvider::new_with_trust(
            RootProvider::new(RpcClient::mocked(asserter)),
            8,
            false,
        );
        let error = provider.receipts_by_hash(requested).await.unwrap_err();
        assert!(error.to_string().contains("Header hash mismatch"), "{error:?}");
        assert!(provider.header_by_hash_cache.is_empty());
        assert!(provider.receipts_by_hash_cache.is_empty());
    }

    #[tokio::test]
    async fn empty_l1_body_is_verified_and_trusted_bodies_keep_existing_behavior() {
        use crate::test_utils::rpc_block;
        for (trust_rpc, transactions_root, expected_ok) in [
            (false, alloy_consensus::EMPTY_ROOT_HASH, true),
            (false, B256::repeat_byte(1), false),
            (true, B256::repeat_byte(1), true),
        ] {
            let header = Header { transactions_root, ..Default::default() };
            let hash = header.hash_slow();
            let asserter = Asserter::new();
            asserter.push_success(&rpc_block::<TxEnvelope>(header, &[]));
            let mut provider = AlloyChainProvider::new_with_trust(
                RootProvider::new(RpcClient::mocked(asserter)),
                8,
                trust_rpc,
            );
            assert_eq!(
                provider.block_info_and_transactions_by_hash(hash).await.is_ok(),
                expected_ok
            );
        }
    }
    #[tokio::test]
    async fn header_and_empty_receipt_lookups_share_the_cache_in_both_trust_modes() {
        use crate::test_utils::rpc_block;
        let header = Header::default();
        let hash = header.hash_slow();
        let block = rpc_block::<TxEnvelope>(header.clone(), &[]);
        for trust_rpc in [false, true] {
            for header_first in [false, true] {
                let asserter = Asserter::new();
                if header_first {
                    asserter.push_success(&block);
                    asserter.push_success(&Vec::<Bytes>::new());
                } else {
                    asserter.push_success(&Vec::<Bytes>::new());
                    asserter.push_success(&block);
                }
                let mut provider = AlloyChainProvider::new_with_trust(
                    RootProvider::new(RpcClient::mocked(asserter.clone())),
                    8,
                    trust_rpc,
                );
                if header_first {
                    assert_eq!(provider.header_by_hash(hash).await.unwrap(), header);
                }
                assert!(provider.receipts_by_hash(hash).await.unwrap().is_empty());
                // No second block response is available: this must reuse the same header.
                assert_eq!(provider.header_by_hash(hash).await.unwrap(), header);
                assert!(asserter.read_q().is_empty());
            }
        }
    }

    #[tokio::test]
    async fn cached_headers_use_the_transaction_root_to_check_empty_receipts() {
        for trust_rpc in [false, true] {
            for transactions_root in [alloy_consensus::EMPTY_ROOT_HASH, B256::repeat_byte(1)] {
                let header = Header { transactions_root, ..Default::default() };
                let hash = header.hash_slow();
                let asserter = Asserter::new();
                // RPC transaction-list metadata does not determine whether the block is empty.
                asserter.push_success(&rpc_block(
                    hash,
                    vec![B256::repeat_byte(2)],
                    transactions_root,
                ));
                asserter.push_success(&Vec::<Bytes>::new());
                let mut provider = AlloyChainProvider::new_with_trust(
                    RootProvider::new(RpcClient::mocked(asserter.clone())),
                    8,
                    trust_rpc,
                );
                assert_eq!(provider.header_by_hash(hash).await.unwrap(), header);
                let result = provider.raw_receipts_by_hash(hash).await;
                if header.transaction_root_is_empty() {
                    assert!(result.unwrap().is_empty());
                } else {
                    assert!(matches!(
                        result.unwrap_err(),
                        AlloyChainProviderError::EmptyRawReceiptsForNonEmptyBlock {
                            transactions_root: actual_root, ..
                        } if actual_root == transactions_root
                    ));
                }
                assert!(asserter.read_q().is_empty());
            }
        }
    }

    #[tokio::test]
    async fn shared_header_fetch_rejects_invalid_headers_before_caching() {
        use crate::test_utils::rpc_block;
        let header = Header::default();
        let hash = header.hash_slow();
        let valid = rpc_block::<TxEnvelope>(header.clone(), &[]);
        let mut invalid = valid.clone();
        invalid["number"] = serde_json::json!("0x1");
        for empty_receipts_first in [false, true] {
            let asserter = Asserter::new();
            if empty_receipts_first {
                asserter.push_success(&Vec::<Bytes>::new());
            }
            asserter.push_success(&invalid);
            asserter.push_success(&valid);
            let mut provider = AlloyChainProvider::new_with_trust(
                RootProvider::new(RpcClient::mocked(asserter.clone())),
                8,
                false,
            );
            let error = if empty_receipts_first {
                provider.raw_receipts_by_hash(hash).await.unwrap_err()
            } else {
                provider.header_by_hash(hash).await.unwrap_err()
            };
            assert!(error.to_string().contains("Header hash mismatch"), "{error:?}");
            assert!(provider.header_by_hash_cache.is_empty());
            assert_eq!(provider.header_by_hash(hash).await.unwrap(), header);
            assert!(asserter.read_q().is_empty());
        }
    }
}

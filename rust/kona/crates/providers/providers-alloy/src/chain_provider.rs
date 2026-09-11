//! Providers that use alloy provider types on the backend.

#[cfg(feature = "metrics")]
use crate::Metrics;
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

/// The [`AlloyChainProvider`] is a concrete implementation of the [`ChainProvider`] trait,
/// providing data over Ethereum JSON-RPC using an alloy provider as the backend.
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
        &self,
        hash: B256,
    ) -> Result<Vec<Receipt>, AlloyChainProviderError> {
        let raw_receipts: Option<Vec<Bytes>> =
            self.inner.client().request("debug_getRawReceipts", [hash]).await?;
        let raw_receipts = raw_receipts.ok_or(AlloyChainProviderError::RawReceiptsNull(hash))?;

        // Geth returns an empty array both for an unknown hash and for an existing block with no
        // transactions. Resolve that ambiguity before accepting and caching an empty result.
        if raw_receipts.is_empty() {
            self.verify_empty_raw_receipts(hash).await?;
        }

        raw_receipts
            .into_iter()
            .enumerate()
            .map(|(index, raw)| {
                ReceiptEnvelope::decode_2718_exact(raw.as_ref())
                    .map(ReceiptEnvelope::into_receipt)
                    .map_err(|source| AlloyChainProviderError::RawReceiptDecode {
                        hash,
                        index,
                        source,
                    })
            })
            .collect()
    }

    /// Verifies that an empty raw receipt response belongs to an existing empty block.
    async fn verify_empty_raw_receipts(&self, hash: B256) -> Result<(), AlloyChainProviderError> {
        let block = self
            .inner
            .get_block_by_hash(hash)
            .await?
            .ok_or(AlloyChainProviderError::BlockNotFound(hash.into()))?;
        let transaction_count = block.transactions.len();
        let header = block.header.into_consensus();
        self.verify_header_hash(&header, hash)?;

        if transaction_count != 0 || !header.transaction_root_is_empty() {
            return Err(AlloyChainProviderError::EmptyRawReceiptsForNonEmptyBlock {
                hash,
                transaction_count,
                transactions_root: header.transactions_root,
            });
        }
        Ok(())
    }

    /// Fetches JSON receipts through the standard Ethereum RPC method.
    async fn json_receipts_by_hash(
        &self,
        hash: B256,
    ) -> Result<Vec<Receipt>, AlloyChainProviderError> {
        let receipts = self
            .inner
            .get_block_receipts(hash.into())
            .await?
            .ok_or(AlloyChainProviderError::BlockNotFound(hash.into()))?;
        receipts
            .into_iter()
            .map(|receipt| receipt.inner.into_primitives_receipt().as_receipt().cloned())
            .collect::<Option<Vec<_>>>()
            .ok_or(AlloyChainProviderError::ReceiptsConversion(hash))
    }

    /// Falls back to `eth_getBlockReceipts` after `debug_getRawReceipts` fails.
    async fn fallback_receipts_by_hash(
        &self,
        hash: B256,
        raw_error: AlloyChainProviderError,
    ) -> Result<Vec<Receipt>, AlloyChainProviderError> {
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
        &self,
        hash: B256,
    ) -> Result<Vec<Receipt>, AlloyChainProviderError> {
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

    /// Verifies that a header's hash matches the expected hash when `trust_rpc` is false.
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
         (transaction count: {transaction_count}, transactions root: {transactions_root})"
    )]
    EmptyRawReceiptsForNonEmptyBlock {
        /// The requested L1 block hash.
        hash: B256,
        /// The transaction count reported by `eth_getBlockByHash`.
        transaction_count: usize,
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

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::{Eip658Value, ReceiptWithBloom};
    use alloy_eips::eip2718::Encodable2718;
    use alloy_primitives::Bloom;
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
        let provider = mocked_provider(asserter.clone());

        let error = provider.raw_receipts_by_hash(hash).await.unwrap_err();
        assert!(matches!(
            error,
            AlloyChainProviderError::EmptyRawReceiptsForNonEmptyBlock {
                hash: error_hash,
                transaction_count: 1,
                transactions_root: error_root,
            } if error_hash == hash && error_root == transactions_root
        ));
        assert!(asserter.read_q().is_empty());
    }

    #[tokio::test]
    async fn raw_receipt_decode_error_preserves_source_and_index() {
        let asserter = Asserter::new();
        asserter.push_success(&vec![Bytes::from_static(&[0x05])]);
        let provider = mocked_provider(asserter.clone());
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
}

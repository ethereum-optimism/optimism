//! Providers that use alloy provider types on the backend.

mod cache;
use alloy_consensus::{
    Header, Receipt, ReceiptEnvelope, TxEnvelope,
    proofs::{calculate_receipt_root, ordered_trie_root_encoded},
};
use alloy_eips::{
    BlockId, BlockNumberOrTag,
    eip2718::{Decodable2718, Eip2718Error},
};
use alloy_primitives::{Address, B256, Bytes, b256};
use alloy_provider::{Provider, RootProvider};
use alloy_transport::{RpcError, TransportErrorKind, TransportResult};
use async_trait::async_trait;
use cache::Cache;
use kona_derive::{ChainProvider, PipelineError, PipelineErrorKind, ResetError};
use kona_protocol::BlockInfo;
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

/// An L1 block identified by its position relative to the chain head.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum BlockLabel {
    /// The chain head.
    Latest,
    /// The most recent block that the consensus layer considers safe.
    Safe,
    /// The most recent finalized block.
    Finalized,
}

impl From<BlockLabel> for BlockNumberOrTag {
    fn from(label: BlockLabel) -> Self {
        match label {
            BlockLabel::Latest => Self::Latest,
            BlockLabel::Safe => Self::Safe,
            BlockLabel::Finalized => Self::Finalized,
        }
    }
}

/// The [`AlloyChainProvider`] is a concrete implementation of the [`ChainProvider`] trait,
/// providing data over Ethereum JSON-RPC using an alloy provider as the backend.
///
/// Clones share their caches and RPC capabilities, so every component holding a clone benefits
/// from what any other clone has fetched.
#[derive(Debug, Clone)]
pub struct AlloyChainProvider {
    /// The inner Ethereum JSON-RPC provider.
    inner: RootProvider,
    /// Whether to verify the RPC's responses rather than trust them.
    verify: bool,
    /// Optional confirmation policy for online derivation's forward traversal only.
    confirmation_depth: Option<ConfirmationDepth>,
    /// State shared by all clones.
    shared: Arc<Shared>,
}

/// State shared by all clones of an [`AlloyChainProvider`].
#[derive(Debug)]
struct Shared {
    /// Responses cached by block hash.
    cache: Cache,
    /// Whether the L1 RPC supports `debug_getRawReceipts`. Once any clone observes
    /// `MethodNotFound`, all clones use the fallback.
    raw_receipts_supported: AtomicBool,
}

impl Shared {
    fn new(cache_size: NonZeroUsize) -> Self {
        Self { cache: Cache::new(cache_size), raw_receipts_supported: AtomicBool::new(true) }
    }
}

impl AlloyChainProvider {
    /// Creates a new [`AlloyChainProvider`] with the given alloy provider and trust setting.
    pub fn new(inner: RootProvider, cache_size: NonZeroUsize, trust_rpc: bool) -> Self {
        Self {
            inner,
            verify: !trust_rpc,
            confirmation_depth: None,
            shared: Arc::new(Shared::new(cache_size)),
        }
    }

    /// Reads the unsafe block signer from `SystemConfig` at `block_hash`.
    ///
    /// Reading state rather than individual update logs also recovers rotations missed between
    /// polls and restores the previous signer when a rotation is reorged out. The value is not yet
    /// verified against the block's state root, so the read trusts the RPC.
    pub async fn unsafe_block_signer(
        &self,
        system_config: Address,
        block_hash: B256,
    ) -> TransportResult<Address> {
        // bytes32(uint256(keccak256("systemconfig.unsafeblocksigner")) - 1)
        const SLOT: B256 =
            b256!("65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c08");
        let value = self
            .inner
            .get_storage_at(system_config, SLOT.into())
            .hash_canonical(block_hash)
            .await?;
        Ok(Address::from_slice(&value.to_be_bytes::<32>()[12..]))
    }

    /// Hides L1 blocks with fewer than `depth` confirmations from forward traversal.
    ///
    /// Hash lookups, which pipeline resets use, are unaffected. Configure only the online
    /// derivation provider, so unsafe-head recovery and sequencing keep their own L1 views.
    pub fn with_confirmation_depth(
        mut self,
        depth: u64,
        head: watch::Receiver<Option<BlockInfo>>,
    ) -> Self {
        self.confirmation_depth = Some(ConfirmationDepth { depth, head });
        self
    }

    /// Returns the latest L2 block number.
    pub async fn latest_block_number(&self) -> Result<u64, RpcError<TransportErrorKind>> {
        self.inner.get_block_number().await
    }

    /// Returns the chain ID.
    pub async fn chain_id(&self) -> Result<u64, RpcError<TransportErrorKind>> {
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
        // transactions. Resolve that ambiguity before accepting and caching an empty result. An
        // existing block without transactions commits to no receipts, so it needs no root check.
        if raw_receipts.is_empty() {
            self.verify_empty_raw_receipts(hash).await?;
        } else if self.verify {
            // The raw receipts are already EIP-2718 encoded, as the receipts trie stores them.
            self.verify_receipts_root(hash, ordered_trie_root_encoded(&raw_receipts)).await?;
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
        if self.verify {
            verify_header(&header, hash)?;
        }

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
        let receipts: Vec<_> = self
            .inner
            .get_block_receipts(hash.into())
            .await?
            .ok_or(AlloyChainProviderError::BlockNotFound(hash.into()))?
            .into_iter()
            .map(|receipt| receipt.inner.into_primitives_receipt())
            .collect();
        if self.verify {
            // Verify the typed envelopes, before the conversion below discards their types.
            self.verify_receipts_root(hash, calculate_receipt_root(&receipts)).await?;
        }
        receipts
            .iter()
            .map(|receipt| receipt.as_receipt().cloned())
            .collect::<Option<Vec<_>>>()
            .ok_or(AlloyChainProviderError::ReceiptsConversion(hash))
    }

    /// Verifies that the block with the given hash commits to receipts whose root is `actual`.
    async fn verify_receipts_root(
        &self,
        hash: B256,
        actual: B256,
    ) -> Result<(), AlloyChainProviderError> {
        let expected = self.header_by_hash(hash).await?.receipts_root;
        if actual != expected {
            return Err(AlloyChainProviderError::ReceiptsRootMismatch { hash, expected, actual });
        }
        Ok(())
    }

    /// Falls back to `eth_getBlockReceipts` after `debug_getRawReceipts` fails.
    async fn fallback_receipts_by_hash(
        &self,
        hash: B256,
        raw_error: AlloyChainProviderError,
    ) -> Result<Vec<Receipt>, AlloyChainProviderError> {
        let method_not_found = raw_error.is_method_not_found();
        let newly_disabled =
            method_not_found && self.shared.raw_receipts_supported.swap(false, Ordering::SeqCst);

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
                    raw_receipts_disabled = !self.shared.raw_receipts_supported.load(Ordering::SeqCst),
                    "eth_getBlockReceipts fallback succeeded; derivation will continue"
                );
                Ok(receipts)
            }
            Err(fallback_error) => {
                let raw_receipts_disabled =
                    !self.shared.raw_receipts_supported.load(Ordering::SeqCst);
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
}

/// Verifies that `header` hashes to `expected`.
fn verify_header(header: &Header, expected: B256) -> Result<(), AlloyChainProviderError> {
    let actual = header.hash_slow();
    if actual != expected {
        return Err(AlloyChainProviderError::HeaderHashMismatch { expected, actual });
    }
    Ok(())
}

/// An error for the [`AlloyChainProvider`].
#[allow(clippy::enum_variant_names)]
#[derive(Debug, thiserror::Error)]
pub enum AlloyChainProviderError {
    /// Transport error
    #[error(transparent)]
    Transport(#[from] RpcError<TransportErrorKind>),
    /// The RPC returned a header that does not hash to the requested hash.
    #[error("Header hash mismatch: expected {expected}, got {actual}")]
    HeaderHashMismatch {
        /// The requested hash.
        expected: B256,
        /// The hash of the returned header.
        actual: B256,
    },
    /// The RPC returned receipts that do not match the block's receipts root.
    #[error("Receipts root mismatch for L1 block {hash}: expected {expected}, got {actual}")]
    ReceiptsRootMismatch {
        /// The block's hash.
        hash: B256,
        /// The receipts root in the block's header.
        expected: B256,
        /// The root of the returned receipts.
        actual: B256,
    },
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
            error @ (AlloyChainProviderError::HeaderHashMismatch { .. } |
            AlloyChainProviderError::ReceiptsRootMismatch { .. } |
            AlloyChainProviderError::RawReceiptsNull(_) |
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

impl AlloyChainProvider {
    /// Returns the header with the given hash.
    pub async fn header_by_hash(&self, hash: B256) -> Result<Header, AlloyChainProviderError> {
        if let Some(header) = self.shared.cache.header(hash) {
            return Ok(header);
        }
        let block = self
            .inner
            .get_block_by_hash(hash)
            .await?
            .ok_or(AlloyChainProviderError::BlockNotFound(hash.into()))?;
        let header = block.header.into_consensus();
        if self.verify {
            verify_header(&header, hash)?;
        }
        self.shared.cache.insert_header(hash, header.clone());
        Ok(header)
    }

    /// Returns the [`BlockInfo`] of the block with the given number, subject to the confirmation
    /// depth.
    pub async fn block_info_by_number(
        &self,
        number: u64,
    ) -> Result<BlockInfo, AlloyChainProviderError> {
        if self.confirmation_depth.as_ref().is_some_and(|depth| !depth.allows(number)) {
            // A number-based not-found is temporary: traversal waits for more confirmations.
            return Err(AlloyChainProviderError::BlockNotFound(number.into()));
        }
        self.block_info(number.into()).await
    }

    /// Returns the [`BlockInfo`] of the block currently at `label`.
    ///
    /// Which block holds a label is the RPC's claim. Without trusting the RPC, only the returned
    /// header's hash is verified. The header is cached by that hash.
    pub async fn block_info_by_label(
        &self,
        label: BlockLabel,
    ) -> Result<BlockInfo, AlloyChainProviderError> {
        self.block_info(BlockNumberOrTag::from(label).into()).await
    }

    async fn block_info(&self, block_id: BlockId) -> Result<BlockInfo, AlloyChainProviderError> {
        let rpc_header = self
            .inner
            .get_block(block_id)
            .await?
            .ok_or(AlloyChainProviderError::BlockNotFound(block_id))?
            .header;
        let hash = rpc_header.hash;
        let header = rpc_header.into_consensus();
        if self.verify {
            verify_header(&header, hash)?;
        }
        let block_info = BlockInfo {
            hash,
            number: header.number,
            parent_hash: header.parent_hash,
            timestamp: header.timestamp,
        };
        self.shared.cache.insert_header(hash, header);
        Ok(block_info)
    }

    /// Returns the receipts of the block with the given hash.
    pub async fn receipts_by_hash(
        &self,
        hash: B256,
    ) -> Result<Vec<Receipt>, AlloyChainProviderError> {
        if let Some(receipts) = self.shared.cache.receipts(hash) {
            return Ok(receipts);
        }

        let receipts = if self.shared.raw_receipts_supported.load(Ordering::SeqCst) {
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
        }?;

        self.shared.cache.insert_receipts(hash, receipts.clone());

        Ok(receipts)
    }

    /// Returns the [`BlockInfo`] and transactions of the block with the given hash.
    pub async fn block_info_and_transactions_by_hash(
        &self,
        hash: B256,
    ) -> Result<(BlockInfo, Vec<TxEnvelope>), AlloyChainProviderError> {
        if let (Some(header), Some(transactions)) =
            (self.shared.cache.header(hash), self.shared.cache.transactions(hash))
        {
            let block_info = BlockInfo {
                hash,
                number: header.number,
                parent_hash: header.parent_hash,
                timestamp: header.timestamp,
            };
            return Ok((block_info, transactions));
        }
        let block = self
            .inner
            .get_block_by_hash(hash)
            .full()
            .await?
            .ok_or(AlloyChainProviderError::BlockNotFound(hash.into()))?
            .into_consensus()
            .map_transactions(|t| t.inner.into_inner());
        if self.verify {
            verify_header(&block.header, hash)?;
        }
        let block_info = BlockInfo {
            hash,
            number: block.header.number,
            parent_hash: block.header.parent_hash,
            timestamp: block.header.timestamp,
        };
        self.shared.cache.insert_header(hash, block.header);
        self.shared.cache.insert_transactions(hash, block.body.transactions.clone());
        Ok((block_info, block.body.transactions))
    }
}

#[async_trait]
impl ChainProvider for AlloyChainProvider {
    type Error = AlloyChainProviderError;

    async fn header_by_hash(&mut self, hash: B256) -> Result<Header, Self::Error> {
        Self::header_by_hash(self, hash).await
    }

    async fn block_info_by_number(&mut self, number: u64) -> Result<BlockInfo, Self::Error> {
        Self::block_info_by_number(self, number).await
    }

    async fn receipts_by_hash(&mut self, hash: B256) -> Result<Vec<Receipt>, Self::Error> {
        Self::receipts_by_hash(self, hash).await
    }

    async fn block_info_and_transactions_by_hash(
        &mut self,
        hash: B256,
    ) -> Result<(BlockInfo, Vec<TxEnvelope>), Self::Error> {
        Self::block_info_and_transactions_by_hash(self, hash).await
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

    const CACHE_SIZE: NonZeroUsize = NonZeroUsize::new(8).unwrap();

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
        AlloyChainProvider::new(RootProvider::new(RpcClient::mocked(asserter)), CACHE_SIZE, true)
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
        let provider = mocked_provider(asserter.clone()).with_confirmation_depth(4, head_rx);
        let cloned = provider.clone();
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
        let provider = mocked_provider(asserter.clone()).with_confirmation_depth(4, head_rx);
        provider.header_by_hash(B256::repeat_byte(0xbb)).await.unwrap();
        assert!(asserter.read_q().is_empty());
    }

    #[tokio::test]
    async fn unsafe_block_signer_snapshots_are_pinned_to_the_requested_contract_and_branch() {
        let server = MockServer::start();
        let provider = AlloyChainProvider::new(
            RootProvider::new_http(server.url("/").parse().unwrap()),
            CACHE_SIZE,
            true,
        );
        let contract = Address::repeat_byte(1);
        // Distinct branches can have different signers, including a return to the old signer.
        for (request_id, index) in [2, 3, 2].into_iter().enumerate() {
            let hash = B256::repeat_byte(index);
            let signer = Address::repeat_byte(index + 10);
            let mut mock = server.mock(|when, then| {
                when.method("POST").json_body(serde_json::json!({
                    "jsonrpc": "2.0", "id": request_id,
                    "method": "eth_getStorageAt",
                    "params": [contract, "0x65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c08", {"blockHash": hash, "requireCanonical": true}]
                }));
                then.json_body(serde_json::json!({"jsonrpc":"2.0", "id":request_id, "result": signer.into_word()}));
            });
            let result = provider.unsafe_block_signer(contract, hash).await;
            mock.assert();
            assert_eq!(result.unwrap(), signer);
            mock.delete();
        }
    }

    #[tokio::test]
    async fn clones_share_cached_lookups() {
        // The mock serves one response; any second RPC call would fail.
        let asserter = Asserter::new();
        let hash = B256::repeat_byte(0xbb);
        asserter.push_success(&Some(rpc_block(hash, Vec::new(), alloy_consensus::EMPTY_ROOT_HASH)));
        let provider = mocked_provider(asserter.clone());
        let clone = provider.clone();

        let fetched = provider.header_by_hash(hash).await.unwrap();
        assert_eq!(clone.header_by_hash(hash).await.unwrap(), fetched);
        assert!(asserter.read_q().is_empty());
    }

    #[tokio::test]
    async fn label_lookups_fill_the_header_cache() {
        let asserter = Asserter::new();
        let header = Header { number: 7, ..Default::default() };
        let hash = header.hash_slow();
        asserter.push_success(&Some(RpcBlock::<TxEnvelope>::new(
            RpcHeader::new(header.clone()),
            BlockTransactions::Hashes(Vec::new()),
        )));
        let provider = AlloyChainProvider::new(
            RootProvider::new(RpcClient::mocked(asserter.clone())),
            CACHE_SIZE,
            false,
        );

        let block_info = provider.block_info_by_label(BlockLabel::Finalized).await.unwrap();
        assert_eq!(block_info.hash, hash);
        assert_eq!(block_info.number, 7);
        assert_eq!(provider.header_by_hash(hash).await.unwrap(), header);
        assert!(asserter.read_q().is_empty());
    }

    #[tokio::test]
    async fn untrusted_label_lookups_reject_a_mismatched_hash() {
        let asserter = Asserter::new();
        let mut rpc_header = RpcHeader::new(Header::default());
        rpc_header.hash = B256::repeat_byte(0xbb);
        asserter.push_success(&Some(RpcBlock::<TxEnvelope>::new(
            rpc_header,
            BlockTransactions::Hashes(Vec::new()),
        )));
        let provider = AlloyChainProvider::new(
            RootProvider::new(RpcClient::mocked(asserter)),
            CACHE_SIZE,
            false,
        );

        assert!(provider.block_info_by_label(BlockLabel::Latest).await.is_err());
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
        // Reset resolves the base by hash, which the confirmation depth never delays.
        traversal
            .reset(alloy_eips::BlockNumHash::new(1, block.header.hash), SystemConfig::default())
            .await
            .unwrap();
        assert_eq!(traversal.origin().unwrap().hash, block.header.hash);
        assert!(matches!(traversal.advance_origin().await, Err(PipelineErrorKind::Temporary(_))));
        assert!(asserter.read_q().is_empty());
    }

    #[tokio::test]
    async fn reset_to_a_reorged_out_base_resets_again() {
        use kona_derive::{PollingTraversal, Stage};
        use kona_genesis::{RollupConfig, SystemConfig};

        let asserter = Asserter::new();
        asserter.push_success(&Option::<EthereumBlock>::None);
        let mut traversal =
            PollingTraversal::new(mocked_provider(asserter), Arc::new(RollupConfig::default()));
        let base = alloy_eips::BlockNumHash::new(1, B256::repeat_byte(0xbb));
        assert!(matches!(
            traversal.reset(base, SystemConfig::default()).await,
            Err(PipelineErrorKind::Reset(ResetError::BlockNotFound(_)))
        ));
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
        let provider = AlloyChainProvider::new(
            RootProvider::new_http(server.base_url().parse().unwrap()),
            CACHE_SIZE,
            true,
        );

        assert_eq!(provider.receipts_by_hash(B256::repeat_byte(0xaa)).await.unwrap(), [expected]);
        raw_mock.assert();
        assert_eq!(fallback_mock.calls(), 0);
    }

    #[tokio::test]
    async fn empty_raw_receipts_for_unknown_hash_reset_without_fallback() {
        let asserter = Asserter::new();
        asserter.push_success(&Vec::<Bytes>::new());
        asserter.push_success(&Option::<EthereumBlock>::None);
        let provider = mocked_provider(asserter.clone());
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
        let provider = mocked_provider(asserter.clone());

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
        let provider = AlloyChainProvider::new(
            RootProvider::new_http(server.base_url().parse().unwrap()),
            CACHE_SIZE,
            true,
        );
        let provider_clone = provider.clone();

        assert!(provider.receipts_by_hash(B256::repeat_byte(0xba)).await.unwrap().is_empty());
        assert!(provider_clone.receipts_by_hash(B256::repeat_byte(0xbb)).await.unwrap().is_empty());
        assert_eq!(raw_mock.calls(), 1, "the clone must share the sticky fallback state");
        assert_eq!(fallback_mock.calls(), 2);
        assert!(!provider.shared.raw_receipts_supported.load(Ordering::SeqCst));
        assert!(!provider_clone.shared.raw_receipts_supported.load(Ordering::SeqCst));
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
        let provider = AlloyChainProvider::new(
            RootProvider::new_http(server.base_url().parse().unwrap()),
            CACHE_SIZE,
            true,
        );

        assert!(provider.receipts_by_hash(B256::repeat_byte(0xbc)).await.unwrap().is_empty());
        assert!(provider.receipts_by_hash(B256::repeat_byte(0xbd)).await.unwrap().is_empty());
        assert_eq!(raw_mock.calls(), 1);
        assert_eq!(fallback_mock.calls(), 2);
        assert!(!provider.shared.raw_receipts_supported.load(Ordering::SeqCst));
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
        let provider = AlloyChainProvider::new(
            RootProvider::new_http(server.base_url().parse().unwrap()),
            CACHE_SIZE,
            true,
        );

        let first_error = provider.receipts_by_hash(B256::repeat_byte(0xbc)).await.unwrap_err();
        assert!(matches!(first_error, AlloyChainProviderError::ReceiptMethodsFailed { .. }));
        assert!(matches!(PipelineErrorKind::from(first_error), PipelineErrorKind::Temporary(_)));

        let second_error = provider.receipts_by_hash(B256::repeat_byte(0xbd)).await.unwrap_err();
        assert!(matches!(second_error, AlloyChainProviderError::Transport(_)));
        assert!(matches!(PipelineErrorKind::from(second_error), PipelineErrorKind::Temporary(_)));
        assert_eq!(raw_mock.calls(), 1);
        assert_eq!(fallback_mock.calls(), 2);
        assert!(!provider.shared.raw_receipts_supported.load(Ordering::SeqCst));
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
        let provider = AlloyChainProvider::new(
            RootProvider::new_http(server.base_url().parse().unwrap()),
            CACHE_SIZE,
            true,
        );

        assert!(provider.receipts_by_hash(B256::repeat_byte(0xca)).await.unwrap().is_empty());
        assert!(provider.receipts_by_hash(B256::repeat_byte(0xcb)).await.unwrap().is_empty());
        assert_eq!(raw_mock.calls(), 2);
        assert_eq!(fallback_mock.calls(), 2);
        assert!(provider.shared.raw_receipts_supported.load(Ordering::SeqCst));
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
        let provider = AlloyChainProvider::new(
            RootProvider::new_http(server.base_url().parse().unwrap()),
            CACHE_SIZE,
            true,
        );

        assert!(provider.receipts_by_hash(B256::repeat_byte(0xda)).await.unwrap().is_empty());
        raw_mock.assert();
        fallback_mock.assert();
        assert!(provider.shared.raw_receipts_supported.load(Ordering::SeqCst));
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
        let provider = AlloyChainProvider::new(
            RootProvider::new_http(server.base_url().parse().unwrap()),
            CACHE_SIZE,
            true,
        );

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
        let provider = AlloyChainProvider::new(
            RootProvider::new_http(server.base_url().parse().unwrap()),
            CACHE_SIZE,
            true,
        );
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
        let provider = AlloyChainProvider::new(
            RootProvider::new_http(server.base_url().parse().unwrap()),
            CACHE_SIZE,
            true,
        );

        assert_eq!(provider.receipts_by_hash(block_hash).await.unwrap(), expected);
        raw_mock.assert();
        fallback_mock.assert();
    }

    /// Two receipts of different types, one with a log, as `(JSON type, receipt, envelope)`.
    fn verified_receipts() -> Vec<(&'static str, Receipt, ReceiptEnvelope)> {
        let first = Receipt {
            status: Eip658Value::Eip658(true),
            cumulative_gas_used: 21_000,
            logs: vec![Log::new_unchecked(
                Address::repeat_byte(0x10),
                vec![B256::repeat_byte(0x20)],
                Bytes::from_static(&[0x01]),
            )],
        };
        let second = Receipt {
            status: Eip658Value::Eip658(false),
            cumulative_gas_used: 42_000,
            logs: Vec::new(),
        };
        vec![
            ("0x2", first.clone(), ReceiptEnvelope::Eip1559(first.with_bloom())),
            ("0x0", second.clone(), ReceiptEnvelope::Legacy(second.with_bloom())),
        ]
    }

    /// The `eth_getBlockReceipts` JSON for `receipts`, each given with its JSON type.
    fn json_receipts(block_hash: B256, receipts: &[(&str, Receipt)]) -> serde_json::Value {
        receipts
            .iter()
            .enumerate()
            .map(|(index, (tx_type, receipt))| {
                let logs: Vec<_> = receipt
                    .logs
                    .iter()
                    .enumerate()
                    .map(|(log_index, log)| {
                        serde_json::json!({
                            "address": log.address,
                            "topics": log.topics(),
                            "data": log.data.data,
                            "blockHash": block_hash,
                            "blockNumber": "0x10",
                            "transactionHash": B256::with_last_byte(index as u8),
                            "transactionIndex": format!("{index:#x}"),
                            "logIndex": format!("{log_index:#x}"),
                            "removed": false,
                        })
                    })
                    .collect();
                serde_json::json!({
                    "type": tx_type,
                    "status": if receipt.status.coerce_status() { "0x1" } else { "0x0" },
                    "cumulativeGasUsed": format!("{:#x}", receipt.cumulative_gas_used),
                    "logs": logs,
                    "logsBloom": receipt.bloom_slow(),
                    "transactionHash": B256::with_last_byte(index as u8),
                    "transactionIndex": format!("{index:#x}"),
                    "blockHash": block_hash,
                    "blockNumber": "0x10",
                    "gasUsed": "0x5208",
                    "effectiveGasPrice": "0x3b9aca00",
                    "from": Address::repeat_byte(0x30),
                    "to": Address::repeat_byte(0x31),
                    "contractAddress": null,
                })
            })
            .collect()
    }

    /// An untrusted provider over a server whose block commits to `verified_receipts`, answering
    /// `debug_getRawReceipts` with `raw` and `eth_getBlockReceipts` with `json`.
    fn receipts_root_server(
        raw: serde_json::Value,
        json: serde_json::Value,
    ) -> (MockServer, AlloyChainProvider, B256) {
        let envelopes: Vec<_> =
            verified_receipts().into_iter().map(|(_, _, envelope)| envelope).collect();
        let header = Header {
            number: 16,
            receipts_root: calculate_receipt_root(&envelopes),
            ..Default::default()
        };
        let hash = header.hash_slow();
        let block = RpcBlock::<TxEnvelope>::new(
            RpcHeader::new(header),
            BlockTransactions::Hashes(Vec::new()),
        );

        let server = MockServer::start();
        server.mock(|when, then| {
            when.method(POST).body_includes("eth_getBlockByHash");
            then.status(200)
                .header("content-type", "application/json")
                .json_body(serde_json::json!({ "jsonrpc": "2.0", "id": 0, "result": block }));
        });
        server.mock(|when, then| {
            when.method(POST).body_includes("debug_getRawReceipts");
            then.status(200)
                .header("content-type", "application/json")
                .json_body(serde_json::json!({ "jsonrpc": "2.0", "id": 0, "result": raw }));
        });
        server.mock(|when, then| {
            when.method(POST).body_includes("eth_getBlockReceipts");
            then.status(200)
                .header("content-type", "application/json")
                .json_body(serde_json::json!({ "jsonrpc": "2.0", "id": 0, "result": json }));
        });
        let provider = AlloyChainProvider::new(
            RootProvider::new_http(server.base_url().parse().unwrap()),
            CACHE_SIZE,
            false,
        );
        (server, provider, hash)
    }

    /// The raw EIP-2718 encodings of `envelopes`.
    fn raw_receipts(envelopes: &[ReceiptEnvelope]) -> serde_json::Value {
        serde_json::json!(
            envelopes
                .iter()
                .map(|envelope| Bytes::from(envelope.encoded_2718()))
                .collect::<Vec<_>>()
        )
    }

    #[tokio::test]
    async fn untrusted_raw_receipts_matching_the_receipts_root_are_accepted() {
        let receipts = verified_receipts();
        let envelopes: Vec<_> = receipts.iter().map(|(_, _, envelope)| envelope.clone()).collect();
        let (_server, provider, hash) =
            receipts_root_server(raw_receipts(&envelopes), serde_json::Value::Null);

        let fetched = provider.receipts_by_hash(hash).await.unwrap();

        assert_eq!(
            fetched,
            receipts.into_iter().map(|(_, receipt, _)| receipt).collect::<Vec<_>>()
        );
    }

    #[tokio::test]
    async fn untrusted_raw_receipts_failing_the_root_fall_back_to_verified_json_receipts() {
        let receipts = verified_receipts();
        // The raw method drops the second receipt; the JSON method answers in full.
        let raw = raw_receipts(&[receipts[0].2.clone()]);
        let json = json_receipts(
            B256::ZERO,
            &receipts.iter().map(|(ty, receipt, _)| (*ty, receipt.clone())).collect::<Vec<_>>(),
        );
        let (_server, provider, hash) = receipts_root_server(raw, json);

        let fetched = provider.receipts_by_hash(hash).await.unwrap();

        assert_eq!(
            fetched,
            receipts.into_iter().map(|(_, receipt, _)| receipt).collect::<Vec<_>>()
        );
        // A bad answer is not a missing method, so the raw method stays enabled.
        assert!(provider.shared.raw_receipts_supported.load(Ordering::SeqCst));
    }

    #[tokio::test]
    async fn receipts_failing_the_root_on_both_methods_are_temporary() {
        let receipts = verified_receipts();
        let raw = raw_receipts(&[receipts[0].2.clone()]);
        // The receipts root commits to each receipt's type, so mislabelling the legacy receipt
        // as EIP-1559 is caught even though every other field is correct.
        let json = json_receipts(
            B256::ZERO,
            &receipts.iter().map(|(_, receipt, _)| ("0x2", receipt.clone())).collect::<Vec<_>>(),
        );
        let (_server, provider, hash) = receipts_root_server(raw, json);

        let error = provider.receipts_by_hash(hash).await.unwrap_err();

        let AlloyChainProviderError::ReceiptMethodsFailed { raw, fallback } = &error else {
            panic!("expected both methods to fail, got {error}");
        };
        assert!(matches!(**raw, AlloyChainProviderError::ReceiptsRootMismatch { .. }), "{raw}");
        assert!(
            matches!(**fallback, AlloyChainProviderError::ReceiptsRootMismatch { .. }),
            "{fallback}"
        );
        assert!(matches!(PipelineErrorKind::from(error), PipelineErrorKind::Temporary(_)));
    }
}

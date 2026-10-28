//! Network-generic RPC fetching and Ethereum receipt-method fallback.
#[cfg(feature = "metrics")]
use crate::Metrics;
use crate::{BlockSource, ReceiptSource, SourceError};
use alloy_consensus::{Block, Header, ReceiptEnvelope, transaction::TxHashRef};
use alloy_eips::eip2718::Decodable2718;
use alloy_primitives::{B256, Bytes};
use alloy_provider::{Network, Provider, RootProvider, network::Ethereum};
use alloy_rpc_types_eth::{Block as RpcBlock, Header as RpcHeader};
use async_trait::async_trait;
use std::sync::{
    Arc,
    atomic::{AtomicBool, Ordering},
};

/// Fetches consensus data without authenticating the RPC's claims.
#[derive(Debug, Clone)]
pub struct RpcSource<N: Network = Ethereum> {
    inner: RootProvider<N>,
    // Shared across clones; only Ethereum's ReceiptSource implementation uses this capability.
    pub(crate) raw_receipts_supported: Arc<AtomicBool>,
}

impl<N: Network> RpcSource<N> {
    /// Wraps an Alloy RPC client. No trust decision is made here.
    pub fn new(inner: RootProvider<N>) -> Self {
        Self { inner, raw_receipts_supported: Arc::new(AtomicBool::new(true)) }
    }
}

#[async_trait]
impl<N> BlockSource for RpcSource<N>
where
    N: Network<
            Header = Header,
            HeaderResponse = RpcHeader,
            BlockResponse = RpcBlock<<N as Network>::TransactionResponse>,
        >,
    N::TxEnvelope: TxHashRef,
{
    type Transaction = N::TxEnvelope;

    async fn header_by_hash(&mut self, hash: B256) -> Result<Header, SourceError> {
        #[cfg(feature = "metrics")]
        Metrics::rpc_call::<N>("header_by_hash", Some("l2_block_by_hash"));
        let block = self
            .inner
            .get_block_by_hash(hash)
            .await
            .inspect_err(|_| {
                #[cfg(feature = "metrics")]
                Metrics::rpc_error::<N>("header_by_hash", Some("l2_block_by_hash"));
            })?
            .ok_or(SourceError::BlockNotFound(hash.into()))?;
        Ok(block.header.into_consensus())
    }

    async fn header_by_number(&mut self, number: u64) -> Result<(B256, Header), SourceError> {
        #[cfg(feature = "metrics")]
        Metrics::rpc_call::<N>("block_by_number", Some("l2_block_ref_by_number"));
        let block = self
            .inner
            .get_block_by_number(number.into())
            .await
            .inspect_err(|_| {
                #[cfg(feature = "metrics")]
                Metrics::rpc_error::<N>("block_by_number", Some("l2_block_ref_by_number"));
            })?
            .ok_or(SourceError::BlockNotFound(number.into()))?;
        Ok((block.header.hash, block.header.into_consensus()))
    }

    async fn block_by_hash(
        &mut self,
        hash: B256,
    ) -> Result<Arc<Block<Self::Transaction>>, SourceError> {
        #[cfg(feature = "metrics")]
        Metrics::rpc_call::<N>("block_by_hash", Some("l2_block_by_hash"));
        let block = self
            .inner
            .get_block_by_hash(hash)
            .full()
            .await
            .inspect_err(|_| {
                #[cfg(feature = "metrics")]
                Metrics::rpc_error::<N>("block_by_hash", Some("l2_block_by_hash"));
            })?
            .ok_or(SourceError::BlockNotFound(hash.into()))?;
        Ok(Arc::new(block.into_consensus().map_transactions(|tx| tx.as_ref().clone())))
    }

    async fn block_by_number(
        &mut self,
        number: u64,
    ) -> Result<(B256, Arc<Block<Self::Transaction>>), SourceError> {
        #[cfg(feature = "metrics")]
        Metrics::rpc_call::<N>("block_by_number", Some("l2_block_ref_by_number"));
        let block = self
            .inner
            .get_block_by_number(number.into())
            .full()
            .await
            .inspect_err(|_| {
                #[cfg(feature = "metrics")]
                Metrics::rpc_error::<N>("block_by_number", Some("l2_block_ref_by_number"));
            })?
            .ok_or(SourceError::BlockNotFound(number.into()))?;
        let hash = block.header.hash;
        Ok((hash, Arc::new(block.into_consensus().map_transactions(|tx| tx.as_ref().clone()))))
    }

    async fn chain_id(&mut self) -> Result<u64, SourceError> {
        Ok(self.inner.get_chain_id().await?)
    }

    async fn latest_block_number(&mut self) -> Result<u64, SourceError> {
        #[cfg(feature = "metrics")]
        Metrics::rpc_call::<N>("block_number", None);
        Ok(self.inner.get_block_number().await.inspect_err(|_| {
            #[cfg(feature = "metrics")]
            Metrics::rpc_error::<N>("block_number", None);
        })?)
    }
}

impl RpcSource<Ethereum> {
    /// Fetches raw consensus-encoded receipts and decodes their EIP-2718 envelopes.
    pub(crate) async fn raw_receipts_by_hash(
        &mut self,
        hash: B256,
    ) -> Result<Vec<ReceiptEnvelope>, SourceError> {
        let raw_receipts: Option<Vec<Bytes>> =
            self.inner.client().request("debug_getRawReceipts", [hash]).await?;
        let raw_receipts = raw_receipts.ok_or(SourceError::RawReceiptsNull(hash))?;

        // Geth returns an empty array both for an unknown hash and for an existing block with no
        // transactions. Resolve that ambiguity before accepting and caching an empty result.
        if raw_receipts.is_empty() {
            self.check_empty_raw_receipts(hash).await?;
        }

        let receipts = raw_receipts
            .into_iter()
            .enumerate()
            .map(|(index, raw)| {
                ReceiptEnvelope::decode_2718_exact(raw.as_ref())
                    .map_err(|source| SourceError::RawReceiptDecode { hash, index, source })
            })
            .collect::<Result<Vec<_>, _>>()?;
        Ok(receipts)
    }

    /// Verifies that an empty raw receipt response belongs to an existing empty block.
    async fn check_empty_raw_receipts(&mut self, hash: B256) -> Result<(), SourceError> {
        let header = self.header_by_hash(hash).await?;
        if !header.transaction_root_is_empty() {
            return Err(SourceError::EmptyRawReceiptsForNonEmptyBlock {
                hash,
                transactions_root: header.transactions_root,
            });
        }
        Ok(())
    }

    /// Fetches JSON receipts through the standard Ethereum RPC method.
    async fn json_receipts_by_hash(&self, hash: B256) -> Result<Vec<ReceiptEnvelope>, SourceError> {
        let receipts = self
            .inner
            .get_block_receipts(hash.into())
            .await?
            .ok_or(SourceError::BlockNotFound(hash.into()))?;
        let receipts = receipts
            .into_iter()
            .map(|receipt| receipt.inner.into_primitives_receipt())
            .collect::<Vec<_>>();
        Ok(receipts)
    }

    /// Falls back to `eth_getBlockReceipts` after `debug_getRawReceipts` fails.
    async fn fallback_receipts_by_hash(
        &self,
        hash: B256,
        raw_error: SourceError,
    ) -> Result<Vec<ReceiptEnvelope>, SourceError> {
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
                    "eth_getBlockReceipts fallback returned receipts"
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
                Err(SourceError::ReceiptMethodsFailed {
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
    ) -> Result<Vec<ReceiptEnvelope>, SourceError> {
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

#[async_trait]
impl ReceiptSource for RpcSource<Ethereum> {
    async fn receipts_by_hash(&mut self, hash: B256) -> Result<Vec<ReceiptEnvelope>, SourceError> {
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
                Err(error @ SourceError::EmptyRawReceiptsForNonEmptyBlock { .. }) => Err(error),
                Err(raw_error) => self.fallback_receipts_by_hash(hash, raw_error).await,
            }
        } else {
            self.sticky_fallback_receipts_by_hash(hash).await
        }
        .inspect_err(|_e| {
            kona_macros::inc!(gauge, Metrics::CHAIN_PROVIDER_RPC_ERRORS, "method" => "receipts_by_hash");
        })?;

        Ok(receipts)
    }
}

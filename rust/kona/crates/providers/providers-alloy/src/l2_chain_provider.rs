//! Providers that use alloy provider types on the backend.

#[cfg(feature = "metrics")]
use crate::Metrics;
use alloy_primitives::B256;
use alloy_provider::{Network, Provider, RootProvider};
use alloy_transport::{RpcError, TransportErrorKind};
use async_trait::async_trait;
use kona_derive::{L2ChainProvider, PipelineError, PipelineErrorKind, ResetError};
use kona_genesis::{RollupConfig, SystemConfig};
use kona_protocol::{BatchValidationProvider, L2BlockInfo, to_system_config};
use lru::LruCache;
use op_alloy_consensus::OpBlock;
use op_alloy_network::Optimism;
use std::{num::NonZeroUsize, sync::Arc};

/// The [`AlloyL2ChainProvider`] is a concrete implementation of the [`L2ChainProvider`] trait,
/// providing data over Ethereum JSON-RPC using an alloy provider as the backend.
///
/// A returned block must have the requested hash or number and carry its transactions, but its
/// transactions are not checked against the header's transactions root, so point it only at an
/// execution client you control, as op-node does with its engine client.
#[derive(Debug, Clone)]
pub struct AlloyL2ChainProvider {
    /// The inner Ethereum JSON-RPC provider.
    inner: RootProvider<Optimism>,
    /// The rollup configuration.
    rollup_config: Arc<RollupConfig>,
    /// The `block_by_hash` LRU cache. Blocks are cached by hash only: a height does not identify
    /// a block across a reorg, so `block_by_number` always asks the RPC.
    block_by_hash_cache: LruCache<B256, Arc<OpBlock>>,
}

impl AlloyL2ChainProvider {
    /// Creates a new [`AlloyL2ChainProvider`] with the given alloy provider and [`RollupConfig`].
    ///
    /// ## Panics
    /// - Panics if `cache_size` is zero.
    pub fn new(
        inner: RootProvider<Optimism>,
        rollup_config: Arc<RollupConfig>,
        cache_size: usize,
    ) -> Self {
        Self {
            inner,
            rollup_config,
            block_by_hash_cache: LruCache::new(NonZeroUsize::new(cache_size).unwrap()),
        }
    }

    /// Fetches the [`OpBlock`] with the given hash, caching the result.
    async fn block_by_hash(
        &mut self,
        hash: B256,
    ) -> Result<Arc<OpBlock>, AlloyL2ChainProviderError> {
        if let Some(block) = self.block_by_hash_cache.get(&hash) {
            return Ok(Arc::clone(block));
        }

        kona_macros::inc!(gauge, Metrics::L2_CHAIN_PROVIDER_REQUESTS, "method" => "l2_block_by_hash");

        let block = self
            .inner
            .get_block_by_hash(hash)
            .full()
            .await
            .map_err(|e| {
                kona_macros::inc!(gauge, Metrics::L2_CHAIN_PROVIDER_ERRORS, "method" => "l2_block_by_hash");
                AlloyL2ChainProviderError::Transport(e)
            })?
            .ok_or(AlloyL2ChainProviderError::BlockHashNotFound(hash))?;

        let block = Arc::new(into_op_block(block)?);
        let actual = block.header.hash_slow();
        if actual != hash {
            return Err(AlloyL2ChainProviderError::BlockHashMismatch { expected: hash, actual });
        }
        self.block_by_hash_cache.put(hash, Arc::clone(&block));
        Ok(block)
    }

    /// Returns the chain ID.
    pub async fn chain_id(&mut self) -> Result<u64, RpcError<TransportErrorKind>> {
        self.inner.get_chain_id().await
    }

    /// Returns the latest L2 block number.
    pub async fn latest_block_number(&mut self) -> Result<u64, RpcError<TransportErrorKind>> {
        self.inner.get_block_number().await
    }
}

/// Converts a block fetched with full transactions into an [`OpBlock`], failing if the RPC
/// returned only the transaction hashes.
fn into_op_block(
    block: <Optimism as Network>::BlockResponse,
) -> Result<OpBlock, AlloyL2ChainProviderError> {
    if !block.transactions.is_full() {
        return Err(AlloyL2ChainProviderError::TransactionHashesOnly(block.header.number));
    }
    Ok(block.into_consensus().map_transactions(|t| t.inner.inner.into_inner()))
}

/// An error for the [`AlloyL2ChainProvider`].
#[derive(Debug, thiserror::Error)]
pub enum AlloyL2ChainProviderError {
    /// Transport error
    #[error(transparent)]
    Transport(#[from] RpcError<TransportErrorKind>),
    /// Failed to find a block.
    #[error("Failed to fetch block {0}")]
    BlockNotFound(u64),
    /// Failed to find a block by hash.
    #[error("Failed to fetch block {0}")]
    BlockHashNotFound(B256),
    /// The RPC answered a lookup by hash with a block of another hash.
    #[error("L2 execution layer returned block {actual} for requested block {expected}")]
    BlockHashMismatch {
        /// The requested hash.
        expected: B256,
        /// The hash of the returned block's header.
        actual: B256,
    },
    /// The RPC answered a lookup by number with a block at another number.
    #[error(
        "L2 execution layer returned block number {actual} for requested block number {expected}"
    )]
    BlockNumberMismatch {
        /// The requested number.
        expected: u64,
        /// The number of the returned block.
        actual: u64,
    },
    /// The RPC returned the block's transaction hashes instead of its transactions.
    #[error("L2 execution layer returned only the transaction hashes of block {0}")]
    TransactionHashesOnly(u64),
    /// Failed to construct [`L2BlockInfo`] from the block and genesis.
    #[error("Failed to construct L2BlockInfo from block {0} and genesis")]
    L2BlockInfoConstruction(u64),
    /// Failed to construct [`L2BlockInfo`] from the block with the given hash and genesis.
    #[error("Failed to construct L2BlockInfo from block {0} and genesis")]
    L2BlockInfoConstructionByHash(B256),
    /// Failed to convert the block into a [`SystemConfig`].
    #[error("Failed to convert block {0} into SystemConfig")]
    SystemConfigConversion(B256),
}

impl From<AlloyL2ChainProviderError> for PipelineErrorKind {
    fn from(e: AlloyL2ChainProviderError) -> Self {
        match e {
            AlloyL2ChainProviderError::Transport(e) => {
                Self::Temporary(PipelineError::Provider(format!("Transport error: {e}")))
            }
            AlloyL2ChainProviderError::BlockNotFound(_) => {
                Self::Temporary(PipelineError::Provider("Block not found".to_string()))
            }
            // A hash that resolves to nothing was reorged out; retrying will never succeed.
            AlloyL2ChainProviderError::BlockHashNotFound(hash) => {
                ResetError::BlockNotFound(hash.into()).reset()
            }
            AlloyL2ChainProviderError::L2BlockInfoConstruction(_) |
            AlloyL2ChainProviderError::L2BlockInfoConstructionByHash(_) => Self::Temporary(
                PipelineError::Provider("L2 block info construction failed".to_string()),
            ),
            AlloyL2ChainProviderError::SystemConfigConversion(_) => Self::Temporary(
                PipelineError::Provider("system config conversion failed".to_string()),
            ),
            // An answer that cannot be right for the request stops the node, so the operator
            // fixes the execution layer instead of the node running on its data.
            error @ (AlloyL2ChainProviderError::BlockHashMismatch { .. } |
            AlloyL2ChainProviderError::BlockNumberMismatch { .. } |
            AlloyL2ChainProviderError::TransactionHashesOnly(_)) => {
                Self::Critical(PipelineError::Provider(error.to_string()))
            }
        }
    }
}

#[async_trait]
impl BatchValidationProvider for AlloyL2ChainProvider {
    type Error = AlloyL2ChainProviderError;

    async fn l2_block_info_by_number(&mut self, number: u64) -> Result<L2BlockInfo, Self::Error> {
        let block = self.block_by_number(number).await?;
        L2BlockInfo::from_block_and_genesis(&block, &self.rollup_config.genesis)
            .map_err(|_| AlloyL2ChainProviderError::L2BlockInfoConstruction(number))
    }

    async fn l2_block_info_by_hash(&mut self, hash: B256) -> Result<L2BlockInfo, Self::Error> {
        let block = self.block_by_hash(hash).await?;
        L2BlockInfo::from_block_and_genesis(&block, &self.rollup_config.genesis)
            .map_err(|_| AlloyL2ChainProviderError::L2BlockInfoConstructionByHash(hash))
    }

    async fn block_by_number(&mut self, number: u64) -> Result<Arc<OpBlock>, Self::Error> {
        kona_macros::inc!(gauge, Metrics::L2_CHAIN_PROVIDER_REQUESTS, "method" => "l2_block_ref_by_number");

        let block = self
            .inner
            .get_block_by_number(number.into())
            .full()
            .await
            .map_err(|e| {
                kona_macros::inc!(gauge, Metrics::L2_CHAIN_PROVIDER_ERRORS, "method" => "l2_block_ref_by_number");
                AlloyL2ChainProviderError::Transport(e)
            })?
            .ok_or(AlloyL2ChainProviderError::BlockNotFound(number))?;
        let block = Arc::new(into_op_block(block)?);
        if block.header.number != number {
            return Err(AlloyL2ChainProviderError::BlockNumberMismatch {
                expected: number,
                actual: block.header.number,
            });
        }

        self.block_by_hash_cache.put(block.header.hash_slow(), Arc::clone(&block));
        Ok(block)
    }
}

#[async_trait]
impl L2ChainProvider for AlloyL2ChainProvider {
    type Error = AlloyL2ChainProviderError;

    async fn system_config_by_l2_hash(
        &mut self,
        hash: B256,
        rollup_config: Arc<RollupConfig>,
    ) -> Result<SystemConfig, <Self as BatchValidationProvider>::Error> {
        let block = self.block_by_hash(hash).await?;
        to_system_config(&block, &rollup_config)
            .map_err(|_| AlloyL2ChainProviderError::SystemConfigConversion(hash))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_rpc_client::RpcClient;
    use alloy_rpc_types_eth::{Block as RpcBlock, BlockTransactions, Header as RpcHeader};
    use alloy_transport::mock::Asserter;
    use httpmock::prelude::*;

    /// A JSON-RPC response serving `header` as a full block with no transactions.
    fn block_response(header: alloy_consensus::Header) -> String {
        let rpc_header = RpcHeader::new(header);
        let block: <Optimism as alloy_provider::Network>::BlockResponse =
            RpcBlock::new(rpc_header, BlockTransactions::Full(vec![]));
        serde_json::json!({
            "jsonrpc": "2.0",
            "id": 0,
            "result": serde_json::to_value(block).unwrap(),
        })
        .to_string()
    }

    fn mocked_provider(asserter: Asserter) -> AlloyL2ChainProvider {
        AlloyL2ChainProvider::new(
            RootProvider::<Optimism>::new(RpcClient::mocked(asserter)),
            Arc::new(RollupConfig::default()),
            8,
        )
    }

    fn rpc_block(
        header: alloy_consensus::Header,
        transactions: BlockTransactions<<Optimism as alloy_provider::Network>::TransactionResponse>,
    ) -> <Optimism as alloy_provider::Network>::BlockResponse {
        RpcBlock::new(RpcHeader::new(header), transactions)
    }

    #[tokio::test]
    async fn block_at_another_number_is_critical() {
        let asserter = Asserter::new();
        let header = alloy_consensus::Header { number: 7, ..Default::default() };
        asserter.push_success(&Some(rpc_block(header, BlockTransactions::Full(vec![]))));
        let mut provider = mocked_provider(asserter);

        let error = provider.block_by_number(5).await.unwrap_err();

        assert!(
            matches!(
                error,
                AlloyL2ChainProviderError::BlockNumberMismatch { expected: 5, actual: 7 }
            ),
            "{error}"
        );
        assert!(matches!(PipelineErrorKind::from(error), PipelineErrorKind::Critical(_)));
    }

    #[tokio::test]
    async fn block_of_another_hash_is_critical_and_not_cached() {
        let requested = B256::repeat_byte(0xab);
        let header = alloy_consensus::Header::default();
        let actual = header.hash_slow();
        let asserter = Asserter::new();
        let mut block = rpc_block(header, BlockTransactions::Full(vec![]));
        // An RPC can claim any hash for the header it serves.
        block.header.hash = requested;
        asserter.push_success(&Some(block));
        let mut provider = mocked_provider(asserter);

        let error = provider.l2_block_info_by_hash(requested).await.unwrap_err();

        assert!(
            matches!(
                error,
                AlloyL2ChainProviderError::BlockHashMismatch { expected, actual: got }
                    if expected == requested && got == actual
            ),
            "{error}"
        );
        assert!(matches!(PipelineErrorKind::from(error), PipelineErrorKind::Critical(_)));
        assert!(provider.block_by_hash_cache.is_empty());
    }

    /// Span-batch validation compares a batch's transactions with those of the L2 blocks it
    /// overlaps, so a block served with transaction hashes must not read as a block without any.
    #[tokio::test]
    async fn block_with_transaction_hashes_is_critical() {
        let header = alloy_consensus::Header::default();
        let hash = header.hash_slow();
        for by_hash in [false, true] {
            let asserter = Asserter::new();
            asserter.push_success(&Some(rpc_block(
                header.clone(),
                BlockTransactions::Hashes(vec![B256::repeat_byte(1)]),
            )));
            let mut provider = mocked_provider(asserter);

            let error = if by_hash {
                provider.l2_block_info_by_hash(hash).await.unwrap_err()
            } else {
                provider.block_by_number(0).await.unwrap_err()
            };

            assert!(
                matches!(error, AlloyL2ChainProviderError::TransactionHashesOnly(0)),
                "by hash {by_hash}: {error}"
            );
            assert!(matches!(PipelineErrorKind::from(error), PipelineErrorKind::Critical(_)));
            assert!(provider.block_by_hash_cache.is_empty());
        }
    }

    #[tokio::test]
    async fn block_by_number_follows_a_replaced_block() {
        // A height does not identify a block across a reorg. Once the block at a height is
        // replaced, by-number lookups must return the replacement, as op-node's uncached
        // PayloadByNumber does, or span-batch overlap checks compare against the stale block.
        let old = alloy_consensus::Header { gas_limit: 1, ..Default::default() };
        let new = alloy_consensus::Header { gas_limit: 2, ..Default::default() };
        assert_ne!(old.hash_slow(), new.hash_slow());

        let server = MockServer::start();
        let mut old_mock = server.mock(|when, then| {
            when.method(POST).body_includes("eth_getBlockByNumber");
            then.status(200)
                .header("content-type", "application/json")
                .body(block_response(old.clone()));
        });
        let mut provider = AlloyL2ChainProvider::new(
            RootProvider::<Optimism>::new(RpcClient::new_http(server.base_url().parse().unwrap())),
            Arc::new(RollupConfig::default()),
            8,
        );
        assert_eq!(provider.block_by_number(0).await.unwrap().header.hash_slow(), old.hash_slow());

        old_mock.delete();
        server.mock(|when, then| {
            when.method(POST).body_includes("eth_getBlockByNumber");
            then.status(200)
                .header("content-type", "application/json")
                .body(block_response(new.clone()));
        });
        assert_eq!(provider.block_by_number(0).await.unwrap().header.hash_slow(), new.hash_slow());
    }
}

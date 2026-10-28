//! L2 derivation adapter over a cached consensus-data source.
use crate::{BlockSource, Cached, RpcSource, SourceError};
use alloy_primitives::{B256, Bytes};
use alloy_provider::RootProvider;
use alloy_rpc_client::RpcClient;
use alloy_rpc_types_engine::JwtSecret;
use alloy_transport::{RpcError, TransportErrorKind};
use alloy_transport_http::{
    AuthLayer, Http, HyperClient,
    hyper_util::{client::legacy::Client, rt::TokioExecutor},
    reqwest,
};
use async_trait::async_trait;
use http_body_util::Full;
use kona_derive::{L2ChainProvider, PipelineError, PipelineErrorKind, ResetError};
use kona_genesis::{RollupConfig, SystemConfig};
use kona_protocol::{BatchValidationProvider, L2BlockInfo, to_system_config};
use op_alloy_consensus::{OpBlock, OpTxEnvelope};
use op_alloy_network::Optimism;
use std::sync::Arc;
use tower::ServiceBuilder;

/// Converts accepted L2 blocks into derivation's block info and system configuration.
#[derive(Debug, Clone)]
pub struct AlloyL2ChainProvider<S: BlockSource = Cached<RpcSource<Optimism>>> {
    source: S,
    rollup_config: Arc<RollupConfig>,
}

impl<S: BlockSource<Transaction = OpTxEnvelope>> AlloyL2ChainProvider<S> {
    /// Creates an L2 provider over the selected source and wrappers.
    pub const fn new(source: S, rollup_config: Arc<RollupConfig>) -> Self {
        Self { source, rollup_config }
    }

    async fn block_by_hash(
        &mut self,
        hash: B256,
    ) -> Result<Arc<OpBlock>, AlloyL2ChainProviderError> {
        Ok(self.source.block_by_hash(hash).await?)
    }

    /// Returns the chain ID.
    pub async fn chain_id(&mut self) -> Result<u64, SourceError> {
        self.source.chain_id().await
    }

    /// Returns the latest L2 block number.
    pub async fn latest_block_number(&mut self) -> Result<u64, SourceError> {
        self.source.latest_block_number().await
    }
}

impl AlloyL2ChainProvider {
    /// Creates a trusting provider with an authenticated HTTP client.
    ///
    /// # Panics
    /// Panics if `cache_size` is zero.
    pub fn new_http(
        url: reqwest::Url,
        rollup_config: Arc<RollupConfig>,
        cache_size: usize,
        jwt: JwtSecret,
    ) -> Self {
        let hyper_client = Client::builder(TokioExecutor::new()).build_http::<Full<Bytes>>();
        let service = ServiceBuilder::new().layer(AuthLayer::new(jwt)).service(hyper_client);
        let transport = HyperClient::with_service(service);
        let client = RpcClient::new(Http::with_client(transport, url), false);
        Self::new(
            Cached::new(RpcSource::new(RootProvider::<Optimism>::new(client)), cache_size),
            rollup_config,
        )
    }
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
        }
    }
}

impl From<SourceError> for AlloyL2ChainProviderError {
    fn from(error: SourceError) -> Self {
        match error {
            SourceError::Transport(error) => Self::Transport(error),
            SourceError::BlockNotFound(alloy_eips::BlockId::Hash(hash)) => {
                Self::BlockHashNotFound(hash.block_hash)
            }
            SourceError::BlockNotFound(alloy_eips::BlockId::Number(number)) => Self::BlockNotFound(
                number.as_number().expect("sources request explicit block numbers"),
            ),
            error => Self::Transport(RpcError::local_usage(error)),
        }
    }
}

#[async_trait]
impl<S: BlockSource<Transaction = OpTxEnvelope>> BatchValidationProvider
    for AlloyL2ChainProvider<S>
{
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
        Ok(self.source.block_by_number(number).await?.1)
    }
}

#[async_trait]
impl<S: BlockSource<Transaction = OpTxEnvelope>> L2ChainProvider for AlloyL2ChainProvider<S> {
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
    use crate::Verifying;

    fn verifying_provider(
        inner: RootProvider<Optimism>,
        config: Arc<RollupConfig>,
        cache_size: usize,
    ) -> AlloyL2ChainProvider<Cached<Verifying<RpcSource<Optimism>>>> {
        AlloyL2ChainProvider::new(
            Cached::new(Verifying::new(RpcSource::new(inner)), cache_size),
            config,
        )
    }

    use alloy_rpc_client::RpcClient;
    use alloy_rpc_types_eth::{Block as RpcBlock, BlockTransactions, Header as RpcHeader};
    use httpmock::prelude::*;
    use kona_genesis::ChainGenesis;

    /// An endpoint claiming `claimed_hash` for a block whose header hashes to something else.
    fn mismatched_block(claimed_hash: B256) -> (B256, String) {
        let header = alloy_consensus::Header::default();
        let true_hash = header.hash_slow();
        let mut rpc_header = RpcHeader::new(header);
        rpc_header.hash = claimed_hash;
        let block: <Optimism as alloy_provider::Network>::BlockResponse =
            RpcBlock::new(rpc_header, BlockTransactions::Full(vec![]));
        let body = serde_json::json!({
            "jsonrpc": "2.0",
            "id": 0,
            "result": serde_json::to_value(block).unwrap(),
        });
        (true_hash, body.to_string())
    }

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
            Cached::new(
                RpcSource::new(RootProvider::<Optimism>::new(RpcClient::new_http(
                    server.base_url().parse().unwrap(),
                ))),
                8,
            ),
            Arc::new(RollupConfig::default()),
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

    #[tokio::test]
    async fn untrusted_rpc_cannot_echo_the_requested_block_hash() {
        // The hash an endpoint reports alongside a block is its own claim. Taken at face value, a
        // faulty or malicious endpoint can echo back whatever hash was asked for while serving
        // different header fields, and the block is then cached and used under a hash it does not
        // have.
        let requested = B256::repeat_byte(0xbb);
        let (true_hash, response) = mismatched_block(requested);
        assert_ne!(true_hash, requested, "the header must not actually hash to the request");

        let server = MockServer::start();
        server.mock(|when, then| {
            when.method(POST).body_includes("eth_getBlockByHash");
            then.status(200).header("content-type", "application/json").body(response);
        });

        // Genesis is pinned to the header the endpoint really returned, so the lookup would
        // otherwise succeed and the mismatch would go unnoticed.
        let config = Arc::new(RollupConfig {
            genesis: ChainGenesis {
                l2: alloy_eips::BlockNumHash { hash: true_hash, number: 0 },
                system_config: Some(SystemConfig::default()),
                ..Default::default()
            },
            ..Default::default()
        });
        let mut provider = verifying_provider(
            RootProvider::<Optimism>::new(RpcClient::new_http(server.base_url().parse().unwrap())),
            config.clone(),
            8,
        );

        let err = provider.system_config_by_l2_hash(requested, config).await.unwrap_err();
        assert!(
            matches!(err, AlloyL2ChainProviderError::Transport(_)),
            "expected the hash mismatch to be rejected, got {err:?}"
        );
    }
    #[tokio::test]
    async fn untrusted_block_bodies_are_verified_before_caching() {
        use crate::test_utils::{optimism_transactions, rpc_block};
        use alloy_consensus::{Header, proofs::calculate_transaction_root};
        use alloy_transport::mock::Asserter;

        let transactions = optimism_transactions();
        let header = Header {
            number: 42,
            transactions_root: calculate_transaction_root(&transactions),
            ..Default::default()
        };
        let hash = header.hash_slow();
        let valid = rpc_block(header.clone(), &transactions);
        let mut missing = valid.clone();
        missing["transactions"].as_array_mut().unwrap().pop();
        let mut reordered = valid.clone();
        reordered["transactions"].as_array_mut().unwrap().swap(0, 1);
        let mut forged_hash = valid.clone();
        // PostExec's RPC decoder deliberately stores the supplied hash without recomputing it.
        forged_hash["transactions"][5]["hash"] =
            serde_json::to_value(B256::repeat_byte(0xab)).unwrap();
        let mut altered = valid.clone();
        altered["transactions"][4]["input"] = serde_json::json!("0xdeadbeef");

        for by_number in [false, true] {
            for invalid in [&missing, &reordered, &forged_hash, &altered] {
                let asserter = Asserter::new();
                asserter.push_success(invalid);
                asserter.push_success(&valid);
                let mut provider = verifying_provider(
                    RootProvider::<Optimism>::new(RpcClient::mocked(asserter)),
                    Arc::new(RollupConfig::default()),
                    8,
                );
                let result = if by_number {
                    provider.block_by_number(42).await
                } else {
                    provider.block_by_hash(hash).await
                };
                let error = result.unwrap_err();
                assert!(matches!(error, AlloyL2ChainProviderError::Transport(_)), "{error:?}");
                assert!(
                    error.to_string().contains("root mismatch") ||
                        error.to_string().contains("hash mismatch"),
                    "{error:?}"
                );
                assert!(provider.source.blocks.is_empty());
                // A rejected by-number response must not poison a later by-hash lookup either.
                let accepted = if by_number {
                    provider.block_by_number(42).await.unwrap()
                } else {
                    provider.block_by_hash(hash).await.unwrap()
                };
                assert_eq!(accepted.body.transactions, transactions);
                assert_eq!(
                    provider.block_by_hash(hash).await.unwrap().body.transactions,
                    transactions
                );
            }
        }
    }

    #[tokio::test]
    async fn untrusted_number_lookup_checks_header_hash_and_number() {
        use crate::test_utils::rpc_block;
        use alloy_consensus::Header;
        use alloy_transport::mock::Asserter;
        let valid = rpc_block::<op_alloy_consensus::OpTxEnvelope>(
            Header { number: 42, ..Default::default() },
            &[],
        );
        let mut forged = valid.clone();
        forged["hash"] = serde_json::to_value(B256::repeat_byte(0xab)).unwrap();
        for (response, number) in [(&forged, 42), (&valid, 43)] {
            let asserter = Asserter::new();
            asserter.push_success(response);
            let mut provider = verifying_provider(
                RootProvider::<Optimism>::new(RpcClient::mocked(asserter)),
                Arc::new(RollupConfig::default()),
                8,
            );
            assert!(matches!(
                provider.block_by_number(number).await,
                Err(AlloyL2ChainProviderError::Transport(_))
            ));
            assert!(provider.source.blocks.is_empty());
        }
    }

    #[tokio::test]
    async fn trusted_l2_bodies_keep_the_existing_behavior() {
        use crate::test_utils::rpc_block;
        use alloy_consensus::Header;
        use alloy_transport::mock::Asserter;

        async fn check<S: BlockSource<Transaction = OpTxEnvelope>>(source: S, expected_ok: bool) {
            let mut provider = AlloyL2ChainProvider::new(
                Cached::new(source, 8),
                Arc::new(RollupConfig::default()),
            );
            assert_eq!(provider.block_by_number(0).await.is_ok(), expected_ok);
        }

        for transactions_root in [alloy_consensus::EMPTY_ROOT_HASH, B256::repeat_byte(1)] {
            let response =
                rpc_block::<OpTxEnvelope>(Header { transactions_root, ..Default::default() }, &[]);
            let asserter = Asserter::new();
            asserter.push_success(&response);
            asserter.push_success(&response);
            let rpc = RpcSource::new(RootProvider::<Optimism>::new(RpcClient::mocked(asserter)));
            check(rpc.clone(), true).await;
            check(Verifying::new(rpc), transactions_root == alloy_consensus::EMPTY_ROOT_HASH).await;
        }
    }
}

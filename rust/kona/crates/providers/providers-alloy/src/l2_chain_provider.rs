//! Providers that use alloy provider types on the backend.

#[cfg(feature = "metrics")]
use crate::Metrics;
use alloy_consensus::Header;
use alloy_primitives::{B256, Bytes};
use alloy_provider::{Provider, RootProvider};
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
use lru::LruCache;
use op_alloy_consensus::OpBlock;
use op_alloy_network::Optimism;
use std::{num::NonZeroUsize, sync::Arc};
use tower::ServiceBuilder;

/// The [`AlloyL2ChainProvider`] is a concrete implementation of the [`L2ChainProvider`] trait,
/// providing data over Ethereum JSON-RPC using an alloy provider as the backend.
#[derive(Debug, Clone)]
pub struct AlloyL2ChainProvider {
    /// The inner Ethereum JSON-RPC provider.
    inner: RootProvider<Optimism>,
    /// Whether to trust the RPC without verification.
    trust_rpc: bool,
    /// The rollup configuration.
    rollup_config: Arc<RollupConfig>,
    /// The `block_by_number` LRU cache. Shares its blocks with `block_by_hash_cache`.
    block_by_number_cache: LruCache<u64, Arc<OpBlock>>,
    /// The `block_by_hash` LRU cache. Shares its blocks with `block_by_number_cache`.
    block_by_hash_cache: LruCache<B256, Arc<OpBlock>>,
    /// The L2 chain ID, pre-rendered as the `chain_id` metric label value.
    chain_id_label: Arc<str>,
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
        Self::new_with_trust(inner, rollup_config, cache_size, true)
    }

    /// Creates a new [`AlloyL2ChainProvider`] with the given alloy provider, [`RollupConfig`], and
    /// trust setting.
    ///
    /// ## Panics
    /// - Panics if `cache_size` is zero.
    pub fn new_with_trust(
        inner: RootProvider<Optimism>,
        rollup_config: Arc<RollupConfig>,
        cache_size: usize,
        trust_rpc: bool,
    ) -> Self {
        let chain_id_label = kona_macros::chain_id_label(rollup_config.l2_chain_id.id());
        Self {
            inner,
            trust_rpc,
            rollup_config,
            block_by_number_cache: LruCache::new(NonZeroUsize::new(cache_size).unwrap()),
            block_by_hash_cache: LruCache::new(NonZeroUsize::new(cache_size).unwrap()),
            chain_id_label,
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

        kona_macros::inc!(
            gauge,
            Metrics::L2_CHAIN_PROVIDER_REQUESTS,
            "method" => "l2_block_by_hash",
            Metrics::CHAIN_ID_LABEL => self.chain_id_label.clone()
        );

        let block = self
            .inner
            .get_block_by_hash(hash)
            .full()
            .await
            .map_err(|e| {
                kona_macros::inc!(
                    gauge,
                    Metrics::L2_CHAIN_PROVIDER_ERRORS,
                    "method" => "l2_block_by_hash",
                    Metrics::CHAIN_ID_LABEL => self.chain_id_label.clone()
                );
                AlloyL2ChainProviderError::Transport(e)
            })?
            .ok_or(AlloyL2ChainProviderError::BlockHashNotFound(hash))?;

        let block = block.into_consensus().map_transactions(|t| t.inner.inner.into_inner());
        self.verify_block_hash(&block.header, hash)?;

        let block = Arc::new(block);
        // Not also cached by number: a block found by hash carries no claim to being the
        // canonical one at its own height.
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

    /// Verifies that a header hashes to the expected hash when `trust_rpc` is false.
    ///
    /// The hash an endpoint reports alongside a block is its own claim, so it is recomputed from
    /// the header rather than taken at face value: otherwise an endpoint could echo back the hash
    /// it was asked for while serving different header fields.
    fn verify_block_hash(
        &self,
        header: &Header,
        expected_hash: B256,
    ) -> Result<(), RpcError<TransportErrorKind>> {
        if self.trust_rpc {
            return Ok(());
        }

        let actual_hash = header.hash_slow();
        if actual_hash != expected_hash {
            return Err(RpcError::local_usage_str(&format!(
                "Block hash mismatch: expected {expected_hash:?}, got {actual_hash:?}"
            )));
        }

        Ok(())
    }

    /// Creates a new [`AlloyL2ChainProvider`] from the provided [`reqwest::Url`].
    pub fn new_http(
        url: reqwest::Url,
        rollup_config: Arc<RollupConfig>,
        cache_size: usize,
        jwt: JwtSecret,
    ) -> Self {
        let hyper_client = Client::builder(TokioExecutor::new()).build_http::<Full<Bytes>>();

        let auth_layer = AuthLayer::new(jwt);
        let service = ServiceBuilder::new().layer(auth_layer).service(hyper_client);

        let layer_transport = HyperClient::with_service(service);
        let http_hyper = Http::with_client(layer_transport, url);
        let rpc_client = RpcClient::new(http_hyper, false);

        let rpc = RootProvider::<Optimism>::new(rpc_client);
        Self::new(rpc, rollup_config, cache_size)
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
            AlloyL2ChainProviderError::L2BlockInfoConstruction(_) => Self::Temporary(
                PipelineError::Provider("L2 block info construction failed".to_string()),
            ),
            AlloyL2ChainProviderError::SystemConfigConversion(_) => Self::Temporary(
                PipelineError::Provider("system config conversion failed".to_string()),
            ),
        }
    }
}

#[async_trait]
impl BatchValidationProvider for AlloyL2ChainProvider {
    type Error = AlloyL2ChainProviderError;

    async fn l2_block_info_by_number(&mut self, number: u64) -> Result<L2BlockInfo, Self::Error> {
        let block = self
            .block_by_number(number)
            .await
            .map_err(|_| AlloyL2ChainProviderError::BlockNotFound(number))?;
        L2BlockInfo::from_block_and_genesis(&block, &self.rollup_config.genesis)
            .map_err(|_| AlloyL2ChainProviderError::L2BlockInfoConstruction(number))
    }

    async fn block_by_number(&mut self, number: u64) -> Result<OpBlock, Self::Error> {
        if let Some(block) = self.block_by_number_cache.get(&number) {
            return Ok((**block).clone());
        }

        kona_macros::inc!(
            gauge,
            Metrics::L2_CHAIN_PROVIDER_REQUESTS,
            "method" => "l2_block_ref_by_number",
            Metrics::CHAIN_ID_LABEL => self.chain_id_label.clone()
        );

        let block = Arc::new(
            self.inner
                .get_block_by_number(number.into())
                .full()
                .await
                .map_err(|e| {
                    kona_macros::inc!(
                        gauge,
                        Metrics::L2_CHAIN_PROVIDER_ERRORS,
                        "method" => "l2_block_ref_by_number",
                        Metrics::CHAIN_ID_LABEL => self.chain_id_label.clone()
                    );
                    AlloyL2ChainProviderError::Transport(e)
                })?
                .ok_or(AlloyL2ChainProviderError::BlockNotFound(number))?
                .into_consensus()
                .map_transactions(|t| t.inner.inner.into_inner()),
        );

        self.block_by_hash_cache.put(block.header.hash_slow(), Arc::clone(&block));
        self.block_by_number_cache.put(number, Arc::clone(&block));
        Ok((*block).clone())
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
        let mut provider = AlloyL2ChainProvider::new_with_trust(
            RootProvider::<Optimism>::new(RpcClient::new_http(server.base_url().parse().unwrap())),
            config.clone(),
            8,
            false, // l2.trust-rpc=false
        );

        let err = provider.system_config_by_l2_hash(requested, config).await.unwrap_err();
        assert!(
            matches!(err, AlloyL2ChainProviderError::Transport(_)),
            "expected the hash mismatch to be rejected, got {err:?}"
        );
    }
}

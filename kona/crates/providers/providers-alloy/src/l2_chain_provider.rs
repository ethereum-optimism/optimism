//! Providers that use alloy provider types on the backend.

#[cfg(feature = "metrics")]
use crate::Metrics;
use alloy_eips::BlockId;
use alloy_primitives::{B256, Bytes};
use alloy_provider::{Provider, RootProvider};
use alloy_rpc_client::RpcClient;
use alloy_rpc_types_engine::JwtSecret;
use alloy_transport::{RpcError, TransportErrorKind};
use alloy_transport_http::{
    AuthLayer, Http, HyperClient,
    hyper_util::{client::legacy::Client, rt::TokioExecutor},
};
use async_trait::async_trait;
use http_body_util::Full;
use kona_derive::{L2ChainProvider, PipelineError, PipelineErrorKind};
use kona_genesis::{RollupConfig, SystemConfig};
use kona_protocol::{BatchValidationProvider, L2BlockInfo, to_system_config};
use lru::LruCache;
use op_alloy_consensus::OpBlock;
use op_alloy_network::Optimism;
use std::{num::NonZeroUsize, sync::Arc};
use tower::ServiceBuilder;

/// The [AlloyL2ChainProvider] is a concrete implementation of the [L2ChainProvider] trait,
/// providing data over Ethereum JSON-RPC using an alloy provider as the backend.
#[derive(Debug, Clone)]
pub struct AlloyL2ChainProvider {
    /// The inner Ethereum JSON-RPC provider.
    inner: RootProvider<Optimism>,
    /// Whether to trust the RPC without verification.
    trust_rpc: bool,
    /// The rollup configuration.
    rollup_config: Arc<RollupConfig>,
    /// The `block_by_number` LRU cache.
    block_by_number_cache: LruCache<u64, OpBlock>,
}

impl AlloyL2ChainProvider {
    /// Creates a new [AlloyL2ChainProvider] with the given alloy provider and [RollupConfig].
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

    /// Creates a new [AlloyL2ChainProvider] with the given alloy provider, [RollupConfig], and
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
        Self {
            inner,
            trust_rpc,
            rollup_config,
            block_by_number_cache: LruCache::new(NonZeroUsize::new(cache_size).unwrap()),
        }
    }

    /// Returns the chain ID.
    pub async fn chain_id(&mut self) -> Result<u64, RpcError<TransportErrorKind>> {
        self.inner.get_chain_id().await
    }

    /// Returns the latest L2 block number.
    pub async fn latest_block_number(&mut self) -> Result<u64, RpcError<TransportErrorKind>> {
        self.inner.get_block_number().await
    }

    /// Verifies that a block's hash matches the expected hash when trust_rpc is false.
    fn verify_block_hash(
        &self,
        block_hash: B256,
        expected_hash: B256,
    ) -> Result<(), RpcError<TransportErrorKind>> {
        if self.trust_rpc {
            return Ok(());
        }

        if block_hash != expected_hash {
            return Err(RpcError::local_usage_str(&format!(
                "Block hash mismatch: expected {expected_hash:?}, got {block_hash:?}"
            )));
        }

        Ok(())
    }

    /// Returns the [L2BlockInfo] for the given [BlockId]. [None] is returned if the block
    /// does not exist.
    pub async fn block_info_by_id(
        &mut self,
        id: BlockId,
    ) -> Result<Option<L2BlockInfo>, RpcError<TransportErrorKind>> {
        #[cfg(feature = "metrics")]
        let method_name = match id {
            BlockId::Number(_) => "l2_block_ref_by_number",
            BlockId::Hash(_) => "l2_block_ref_by_hash",
        };

        kona_macros::inc!(gauge, Metrics::L2_CHAIN_PROVIDER_REQUESTS, "method" => method_name);

        let result = async {
            let block = match id {
                BlockId::Number(num) => self.inner.get_block_by_number(num).full().await?,
                BlockId::Hash(hash) => {
                    let block = self.inner.get_block_by_hash(hash.block_hash).full().await?;

                    // Verify block hash matches if we fetched by hash
                    if let Some(ref b) = block {
                        self.verify_block_hash(b.header.hash, hash.block_hash)?;
                    }

                    block
                }
            };

            match block {
                Some(block) => {
                    let consensus_block =
                        block.into_consensus().map_transactions(|t| t.inner.inner);

                    let l2_block = L2BlockInfo::from_block_and_genesis(
                        &consensus_block,
                        &self.rollup_config.genesis,
                    )
                    .map_err(|_| {
                        RpcError::local_usage_str(
                            "failed to construct L2BlockInfo from block and genesis",
                        )
                    })?;
                    Ok(Some(l2_block))
                }
                None => Ok(None),
            }
        }
        .await;

        #[cfg(feature = "metrics")]
        if result.is_err() {
            kona_macros::inc!(gauge, Metrics::L2_CHAIN_PROVIDER_ERRORS, "method" => method_name);
        }

        result
    }

    /// Creates a new [AlloyL2ChainProvider] from the provided [reqwest::Url].
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

/// An error for the [AlloyL2ChainProvider].
#[derive(Debug, thiserror::Error)]
pub enum AlloyL2ChainProviderError {
    /// Transport error
    #[error(transparent)]
    Transport(#[from] RpcError<TransportErrorKind>),
    /// Failed to find a block.
    #[error("Failed to fetch block {0}")]
    BlockNotFound(u64),
    /// Failed to construct [L2BlockInfo] from the block and genesis.
    #[error("Failed to construct L2BlockInfo from block {0} and genesis")]
    L2BlockInfoConstruction(u64),
    /// Failed to convert the block into a [SystemConfig].
    #[error("Failed to convert block {0} into SystemConfig")]
    SystemConfigConversion(u64),
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
            return Ok(block.clone());
        }

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
            .ok_or(AlloyL2ChainProviderError::BlockNotFound(number))?
            .into_consensus()
            .map_transactions(|t| t.inner.inner.into_inner());

        self.block_by_number_cache.put(number, block.clone());
        Ok(block)
    }
}

#[async_trait]
impl L2ChainProvider for AlloyL2ChainProvider {
    type Error = AlloyL2ChainProviderError;

    async fn system_config_by_number(
        &mut self,
        number: u64,
        rollup_config: Arc<RollupConfig>,
    ) -> Result<SystemConfig, <Self as BatchValidationProvider>::Error> {
        let block = self
            .block_by_number(number)
            .await
            .map_err(|_| AlloyL2ChainProviderError::BlockNotFound(number))?;
        to_system_config(&block, &rollup_config)
            .map_err(|_| AlloyL2ChainProviderError::SystemConfigConversion(number))
    }
}

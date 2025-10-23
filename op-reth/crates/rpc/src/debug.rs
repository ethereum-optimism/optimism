//! Historical proofs RPC server implementation for `debug_` namespace.

use crate::state::OpStateProviderFactory;
use alloy_eips::{BlockId, BlockNumberOrTag};
use alloy_primitives::B256;
use alloy_rpc_types_debug::ExecutionWitness;
use async_trait::async_trait;
use jsonrpsee::proc_macros::rpc;
use jsonrpsee_core::RpcResult;
use jsonrpsee_types::error::ErrorObject;
use reth_basic_payload_builder::PayloadConfig;
use reth_evm::ConfigureEvm;
use reth_node_api::{BuildNextEnv, NodePrimitives, PayloadBuilderError};
use reth_optimism_forks::OpHardforks;
use reth_optimism_payload_builder::{
    builder::{OpBuilder, OpPayloadBuilderCtx},
    OpAttributes, OpPayloadPrimitives,
};
use reth_optimism_trie::{OpProofsStorage, OpProofsStore};
use reth_optimism_txpool::OpPooledTransaction as OpPooledTx2;
use reth_payload_util::NoopPayloadTransactions;
use reth_primitives_traits::{SealedHeader, TxTy};
use reth_provider::{
    BlockReaderIdExt, ChainSpecProvider, HeaderProvider, NodePrimitivesProvider, ProviderError,
    ProviderResult, StateProviderFactory,
};
use reth_rpc_api::eth::helpers::FullEthApi;
use reth_rpc_server_types::{result::internal_rpc_err, ToRpcResult};
use reth_tasks::TaskSpawner;
use std::{marker::PhantomData, sync::Arc};
use tokio::sync::{oneshot, Semaphore};

#[cfg_attr(not(test), rpc(server, namespace = "debug"))]
#[cfg_attr(test, rpc(server, client, namespace = "debug"))]
pub trait DebugApiOverride<Attributes> {
    /// Executes a payload and returns the execution witness.
    #[method(name = "executePayload")]
    async fn execute_payload(
        &self,
        parent_block_hash: B256,
        attributes: Attributes,
    ) -> RpcResult<ExecutionWitness>;

    /// Returns the execution witness for a given block.
    #[method(name = "executionWitness")]
    async fn execution_witness(&self, block: BlockNumberOrTag) -> RpcResult<ExecutionWitness>;
}

#[derive(Debug)]
/// Overrides applied to the `debug_` namespace of the RPC API for the OP Proofs ExEx.
pub struct DebugApiExt<Eth: FullEthApi, Storage, Provider, EvmConfig, Attrs> {
    inner: Arc<DebugApiExtInner<Eth, Storage, Provider, EvmConfig, Attrs>>,
}

impl<Eth, Storage, Provider, EvmConfig, Attrs> DebugApiExt<Eth, Storage, Provider, EvmConfig, Attrs>
where
    Eth: FullEthApi + Send + Sync + 'static,
    ErrorObject<'static>: From<Eth::Error>,
    Storage: OpProofsStore + Clone + 'static,
    Provider: BlockReaderIdExt + NodePrimitivesProvider<Primitives: OpPayloadPrimitives>,
    EvmConfig: ConfigureEvm<Primitives = Provider::Primitives> + 'static,
{
    /// Creates a new instance of the `DebugApiExt`.
    pub fn new(
        provider: Provider,
        eth_api: Eth,
        preimage_store: OpProofsStorage<Storage>,
        task_spawner: Box<dyn TaskSpawner>,
        evm_config: EvmConfig,
    ) -> Self {
        Self {
            inner: Arc::new(DebugApiExtInner::new(
                provider,
                eth_api,
                preimage_store,
                task_spawner,
                evm_config,
            )),
        }
    }
}

#[derive(Debug)]
/// Overrides applied to the `debug_` namespace of the RPC API for historical proofs ExEx.
pub struct DebugApiExtInner<Eth: FullEthApi, Storage, Provider, EvmConfig, Attrs> {
    provider: Provider,
    state_provider_factory: OpStateProviderFactory<Eth, Storage>,
    evm_config: EvmConfig,
    task_spawner: Box<dyn TaskSpawner>,
    semaphore: Semaphore,
    _attrs: PhantomData<Attrs>,
}

impl<Eth, P, Provider, EvmConfig, Attrs> DebugApiExtInner<Eth, P, Provider, EvmConfig, Attrs>
where
    Eth: FullEthApi + Send + Sync + 'static,
    ErrorObject<'static>: From<Eth::Error>,
    P: OpProofsStore + Clone + 'static,
    Provider: NodePrimitivesProvider<Primitives: OpPayloadPrimitives>,
{
    fn new(
        provider: Provider,
        eth_api: Eth,
        preimage_store: OpProofsStorage<P>,
        task_spawner: Box<dyn TaskSpawner>,
        evm_config: EvmConfig,
    ) -> Self {
        Self {
            provider,
            state_provider_factory: OpStateProviderFactory::new(eth_api, preimage_store),
            evm_config,
            task_spawner,
            semaphore: Semaphore::new(3),
            _attrs: PhantomData,
        }
    }
}

impl<Eth, P, Provider, EvmConfig, Attrs> DebugApiExt<Eth, P, Provider, EvmConfig, Attrs>
where
    Eth: FullEthApi + Send + Sync + 'static,
    ErrorObject<'static>: From<Eth::Error>,
    P: OpProofsStore + Clone + 'static,
    Provider: BlockReaderIdExt
        + NodePrimitivesProvider<Primitives: OpPayloadPrimitives>
        + HeaderProvider<Header = <Provider::Primitives as NodePrimitives>::BlockHeader>,
{
    fn parent_header(
        &self,
        parent_block_hash: B256,
    ) -> ProviderResult<SealedHeader<Provider::Header>> {
        self.inner
            .provider
            .sealed_header_by_hash(parent_block_hash)?
            .ok_or_else(|| ProviderError::HeaderNotFound(parent_block_hash.into()))
    }
}

#[async_trait]
impl<Eth, P, Provider, EvmConfig, Attrs, N> DebugApiOverrideServer<Attrs::RpcPayloadAttributes>
    for DebugApiExt<Eth, P, Provider, EvmConfig, Attrs>
where
    Eth: FullEthApi + Send + Sync + 'static,
    ErrorObject<'static>: From<Eth::Error>,
    P: OpProofsStore + Clone + 'static,
    Attrs: OpAttributes<Transaction = TxTy<EvmConfig::Primitives>>,
    N: OpPayloadPrimitives,
    EvmConfig: ConfigureEvm<
            Primitives = N,
            NextBlockEnvCtx: BuildNextEnv<Attrs, N::BlockHeader, Provider::ChainSpec>,
        > + 'static,
    Provider: BlockReaderIdExt<Header = N::BlockHeader>
        + StateProviderFactory
        + ChainSpecProvider<ChainSpec: OpHardforks>
        + NodePrimitivesProvider<Primitives = N>
        + HeaderProvider<Header = N::BlockHeader>
        + Clone
        + 'static,
    op_alloy_consensus::OpPooledTransaction:
        TryFrom<<N as OpPayloadPrimitives>::_TX, Error: core::error::Error>,
    <N as OpPayloadPrimitives>::_TX: From<op_alloy_consensus::OpPooledTransaction>,
{
    async fn execute_payload(
        &self,
        parent_block_hash: B256,
        attributes: Attrs::RpcPayloadAttributes,
    ) -> RpcResult<ExecutionWitness> {
        let _permit = self.inner.semaphore.acquire().await;

        let parent_header = self.parent_header(parent_block_hash).to_rpc_result()?;

        let (tx, rx) = oneshot::channel();
        let this = self.inner.clone();
        self.inner.task_spawner.spawn_blocking(Box::pin(async move {
            let result = async {
                let parent_hash = parent_header.hash();
                let attributes = Attrs::try_new(parent_hash, attributes, 3)
                    .map_err(PayloadBuilderError::other)?;

                let config = PayloadConfig { parent_header: Arc::new(parent_header), attributes };
                let ctx = OpPayloadBuilderCtx {
                    evm_config: this.evm_config.clone(),
                    da_config: Default::default(), // doesn't matter if no txpool
                    chain_spec: this.provider.chain_spec(),
                    config,
                    cancel: Default::default(),
                    best_payload: Default::default(),
                };

                let state_provider = this
                    .state_provider_factory
                    .state_provider(Some(BlockId::Hash(parent_hash.into())))
                    .await
                    .map_err(PayloadBuilderError::other)?;

                let builder = OpBuilder::new(|_| {
                    NoopPayloadTransactions::<
                        OpPooledTx2<
                            <N as OpPayloadPrimitives>::_TX,
                            op_alloy_consensus::OpPooledTransaction,
                        >,
                    >::default()
                });

                builder.witness(state_provider, &ctx).map_err(PayloadBuilderError::other)
            };

            let _ = tx.send(result.await);
        }));

        rx.await
            .map_err(|err| internal_rpc_err(err.to_string()))?
            .map_err(|err| internal_rpc_err(err.to_string()))
    }

    async fn execution_witness(&self, _block: BlockNumberOrTag) -> RpcResult<ExecutionWitness> {
        unimplemented!()
    }
}

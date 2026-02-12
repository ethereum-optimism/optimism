//! Support for optimism specific witness RPCs.

use alloy_primitives::B256;
use alloy_rpc_types_debug::ExecutionWitness;
use jsonrpsee_core::{RpcResult, async_trait};
use reth_chainspec::ChainSpecProvider;
use reth_evm::ConfigureEvm;
use reth_node_api::{BuildNextEnv, NodePrimitives};
use reth_optimism_forks::OpHardforks;
use reth_optimism_payload_builder::{OpAttributes, OpPayloadBuilder, OpPayloadPrimitives};
use reth_optimism_txpool::OpPooledTx;
use reth_primitives_traits::{SealedHeader, TxTy};
pub use reth_rpc_api::DebugExecutionWitnessApiServer;
use reth_rpc_server_types::{ToRpcResult, result::internal_rpc_err};
use reth_storage_api::{
    BlockReaderIdExt, NodePrimitivesProvider, StateProviderFactory,
    errors::{ProviderError, ProviderResult},
};
use reth_tasks::TaskSpawner;
use reth_transaction_pool::TransactionPool;
use std::{fmt::Debug, sync::Arc};
use tokio::sync::{Semaphore, oneshot};

/// An extension to the `debug_` namespace of the RPC API.
pub struct OpDebugWitnessApi<Pool, Provider, EvmConfig, Attrs> {
    inner: Arc<OpDebugWitnessApiInner<Pool, Provider, EvmConfig, Attrs>>,
}

impl<Pool, Provider, EvmConfig, Attrs> OpDebugWitnessApi<Pool, Provider, EvmConfig, Attrs> {
    /// Creates a new instance of the `OpDebugWitnessApi`.
    pub fn new(
        provider: Provider,
        task_spawner: Box<dyn TaskSpawner>,
        builder: OpPayloadBuilder<Pool, Provider, EvmConfig, (), Attrs>,
    ) -> Self {
        let semaphore = Arc::new(Semaphore::new(3));
        let inner = OpDebugWitnessApiInner { provider, builder, task_spawner, semaphore };
        Self { inner: Arc::new(inner) }
    }
}

impl<Pool, Provider, EvmConfig, Attrs> OpDebugWitnessApi<Pool, Provider, EvmConfig, Attrs>
where
    EvmConfig: ConfigureEvm,
    Provider: NodePrimitivesProvider<Primitives: NodePrimitives<BlockHeader = Provider::Header>>
        + BlockReaderIdExt,
{
    /// Fetches the parent header by hash.
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
impl<Pool, Provider, EvmConfig, Attrs> DebugExecutionWitnessApiServer<Attrs::RpcPayloadAttributes>
    for OpDebugWitnessApi<Pool, Provider, EvmConfig, Attrs>
where
    Pool: TransactionPool<
            Transaction: OpPooledTx<Consensus = <Provider::Primitives as NodePrimitives>::SignedTx>,
        > + 'static,
    Provider: BlockReaderIdExt<Header = <Provider::Primitives as NodePrimitives>::BlockHeader>
        + NodePrimitivesProvider<Primitives: OpPayloadPrimitives>
        + StateProviderFactory
        + ChainSpecProvider<ChainSpec: OpHardforks>
        + Clone
        + 'static,
    EvmConfig: ConfigureEvm<
            Primitives = Provider::Primitives,
            NextBlockEnvCtx: BuildNextEnv<Attrs, Provider::Header, Provider::ChainSpec>,
        > + 'static,
    Attrs: OpAttributes<Transaction = TxTy<EvmConfig::Primitives>>,
{
    async fn execute_payload(
        &self,
        parent_block_hash: B256,
        attributes: Attrs::RpcPayloadAttributes,
    ) -> RpcResult<ExecutionWitness> {
        let _permit = self.inner.semaphore.acquire().await;

        let parent_header = self.parent_header(parent_block_hash).to_rpc_result()?;

        let (tx, rx) = oneshot::channel();
        let this = self.clone();
        self.inner.task_spawner.spawn_blocking(Box::pin(async move {
            let res = this.inner.builder.payload_witness(parent_header, attributes);
            let _ = tx.send(res);
        }));

        rx.await
            .map_err(|err| internal_rpc_err(err.to_string()))?
            .map_err(|err| internal_rpc_err(err.to_string()))
    }
}

impl<Pool, Provider, EvmConfig, Attrs> Clone
    for OpDebugWitnessApi<Pool, Provider, EvmConfig, Attrs>
{
    fn clone(&self) -> Self {
        Self { inner: Arc::clone(&self.inner) }
    }
}
impl<Pool, Provider, EvmConfig, Attrs> Debug
    for OpDebugWitnessApi<Pool, Provider, EvmConfig, Attrs>
{
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("OpDebugWitnessApi").finish_non_exhaustive()
    }
}

struct OpDebugWitnessApiInner<Pool, Provider, EvmConfig, Attrs> {
    provider: Provider,
    builder: OpPayloadBuilder<Pool, Provider, EvmConfig, (), Attrs>,
    task_spawner: Box<dyn TaskSpawner>,
    semaphore: Arc<Semaphore>,
}

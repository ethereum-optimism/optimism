//! Support for optimism specific witness RPCs.

use alloy_consensus::{Block as AlloyBlock, BlockHeader};
use alloy_eips::BlockId;
use alloy_primitives::{B256, Sealed};
use alloy_rpc_types_debug::ExecutionWitness;
use jsonrpsee::proc_macros::rpc;
use jsonrpsee_core::{RpcResult, async_trait};
use op_alloy_consensus::TxPostExec;
use reth_chainspec::ChainSpecProvider;
use reth_node_api::{BuildNextEnv, NodePrimitives};
use reth_optimism_evm::ConfigurePostExecEvm;
use reth_optimism_forks::OpHardforks;
use reth_optimism_payload_builder::{OpAttributes, OpPayloadBuilder, OpPayloadPrimitives};
use reth_optimism_post_exec_replay::{
    PostExecReplayBlock, PostExecReplayConfig, ReplayPostExecBlockOptions,
    ReplayPostExecBlockRequest, replay_block,
};
use reth_optimism_txpool::OpPooledTx;
use reth_primitives_traits::{RecoveredBlock, SealedHeader, TxTy};
use reth_revm::database::StateProviderDatabase;
use reth_rpc_server_types::{ToRpcResult, result::internal_rpc_err};

/// Trait for the `debug_executePayload` endpoint, which re-executes a payload and returns the
/// resulting execution witness.
///
/// Vendored from `reth_rpc_api::DebugExecutionWitnessApi`, which was removed upstream in
/// paradigmxyz/reth#24284.
#[cfg_attr(not(feature = "client"), rpc(server, namespace = "debug"))]
#[cfg_attr(feature = "client", rpc(server, client, namespace = "debug"))]
pub trait DebugExecutionWitnessApi<Attributes> {
    /// Re-executes a payload built on top of the given parent block and returns the resulting
    /// execution witness.
    #[method(name = "executePayload")]
    async fn execute_payload(
        &self,
        parent_block_hash: B256,
        attributes: Attributes,
    ) -> RpcResult<ExecutionWitness>;
}
use reth_storage_api::{
    BlockReaderIdExt, NodePrimitivesProvider, StateProviderFactory, TransactionVariant,
    errors::{ProviderError, ProviderResult},
};
use reth_tasks::Runtime;
use reth_transaction_pool::TransactionPool;
use std::{fmt::Debug, sync::Arc};
use tokio::sync::{Semaphore, oneshot};

/// An extension to the `debug_` namespace for post-exec replay.
///
/// This trait is registered under the `debug` namespace and is intended for operator and
/// research tooling only. Do not expose the `debug` namespace on public RPC endpoints: each
/// call replays an entire historical block against live state and is unbounded in cost, so
/// an unauthenticated caller can trivially saturate the node.
#[cfg_attr(not(test), rpc(server, namespace = "debug"))]
#[cfg_attr(test, rpc(server, client, namespace = "debug"))]
pub trait OpDebugPostExecApi {
    /// Counterfactually replay a historical block with post-exec enabled.
    ///
    /// Replays one block per call; callers driving a block range are responsible for
    /// their own pacing and cancellation. Requires historical state for the target block
    /// (full/archive node); on a pruned node this will fail at state lookup.
    #[method(name = "replaySDMBlock")]
    async fn replay_post_exec_block(
        &self,
        block: ReplayPostExecBlockRequest,
        options: Option<ReplayPostExecBlockOptions>,
    ) -> RpcResult<PostExecReplayBlock>;
}

/// An extension to the `debug_` namespace of the RPC API.
pub struct OpDebugWitnessApi<Pool, Provider, EvmConfig, Attrs> {
    inner: Arc<OpDebugWitnessApiInner<Pool, Provider, EvmConfig, Attrs>>,
}

impl<Pool, Provider, EvmConfig, Attrs> OpDebugWitnessApi<Pool, Provider, EvmConfig, Attrs> {
    /// Creates a new instance of the `OpDebugWitnessApi`.
    pub fn new(
        provider: Provider,
        task_spawner: Runtime,
        builder: OpPayloadBuilder<Pool, Provider, EvmConfig, (), Attrs>,
        evm_config: EvmConfig,
    ) -> Self {
        let semaphore = Arc::new(Semaphore::new(3));
        let inner =
            OpDebugWitnessApiInner { provider, builder, evm_config, task_spawner, semaphore };
        Self { inner: Arc::new(inner) }
    }
}

impl<Pool, Provider, EvmConfig, Attrs> OpDebugWitnessApi<Pool, Provider, EvmConfig, Attrs>
where
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

impl<Pool, Provider, EvmConfig, Attrs> OpDebugWitnessApi<Pool, Provider, EvmConfig, Attrs>
where
    Provider: BlockReaderIdExt<
            Block = AlloyBlock<
                <Provider::Primitives as OpPayloadPrimitives>::_TX,
                <Provider::Primitives as OpPayloadPrimitives>::_Header,
            >,
            Header = <Provider::Primitives as NodePrimitives>::BlockHeader,
        > + NodePrimitivesProvider<Primitives: OpPayloadPrimitives>,
{
    fn replay_block_by_request(
        &self,
        request: ReplayPostExecBlockRequest,
    ) -> ProviderResult<RecoveredBlock<Provider::Block>> {
        let provider = &self.inner.provider;
        match request {
            ReplayPostExecBlockRequest::Hash(hash) => provider
                .recovered_block(hash.into(), TransactionVariant::NoHash)?
                .ok_or_else(|| ProviderError::HeaderNotFound(hash.into())),
            ReplayPostExecBlockRequest::Number(number_or_tag) => provider
                .block_with_senders_by_id(
                    BlockId::Number(number_or_tag),
                    TransactionVariant::NoHash,
                )?
                .ok_or_else(|| {
                    ProviderError::HeaderNotFound(
                        number_or_tag.as_number().unwrap_or_default().into(),
                    )
                }),
        }
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
    <Provider::Primitives as NodePrimitives>::SignedTx: From<Sealed<TxPostExec>>,
    EvmConfig: ConfigurePostExecEvm<
            Primitives = Provider::Primitives,
            NextBlockEnvCtx: BuildNextEnv<Attrs, Provider::Header, Provider::ChainSpec>,
        > + 'static,
    Attrs: OpAttributes<Transaction = TxTy<EvmConfig::Primitives>, RpcPayloadAttributes: Send>,
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
        self.inner.task_spawner.spawn_blocking_task(async move {
            let res = this.inner.builder.payload_witness(parent_header, attributes);
            let _ = tx.send(res);
        });

        rx.await
            .map_err(|err| internal_rpc_err(err.to_string()))?
            .map_err(|err| internal_rpc_err(err.to_string()))
    }
}

#[async_trait]
impl<Pool, Provider, EvmConfig, Attrs> OpDebugPostExecApiServer
    for OpDebugWitnessApi<Pool, Provider, EvmConfig, Attrs>
where
    Provider::Primitives: OpPayloadPrimitives
        + NodePrimitives<
            Block = AlloyBlock<
                <Provider::Primitives as OpPayloadPrimitives>::_TX,
                <Provider::Primitives as OpPayloadPrimitives>::_Header,
            >,
            BlockBody = alloy_consensus::BlockBody<
                <Provider::Primitives as OpPayloadPrimitives>::_TX,
                <Provider::Primitives as OpPayloadPrimitives>::_Header,
            >,
            BlockHeader = <Provider::Primitives as OpPayloadPrimitives>::_Header,
            SignedTx = <Provider::Primitives as OpPayloadPrimitives>::_TX,
        >,
    Pool: TransactionPool<
            Transaction: OpPooledTx<Consensus = <Provider::Primitives as NodePrimitives>::SignedTx>,
        > + 'static,
    Provider: BlockReaderIdExt<
            Block = AlloyBlock<
                <Provider::Primitives as OpPayloadPrimitives>::_TX,
                <Provider::Primitives as OpPayloadPrimitives>::_Header,
            >,
            Header = <Provider::Primitives as NodePrimitives>::BlockHeader,
        > + NodePrimitivesProvider
        + StateProviderFactory
        + ChainSpecProvider<ChainSpec: OpHardforks>
        + Clone
        + 'static,
    EvmConfig: ConfigurePostExecEvm<
            Primitives = Provider::Primitives,
            NextBlockEnvCtx: BuildNextEnv<Attrs, Provider::Header, Provider::ChainSpec>,
        > + Clone
        + 'static,
    Attrs: OpAttributes<Transaction = TxTy<EvmConfig::Primitives>>,
{
    async fn replay_post_exec_block(
        &self,
        request: ReplayPostExecBlockRequest,
        options: Option<ReplayPostExecBlockOptions>,
    ) -> RpcResult<PostExecReplayBlock> {
        let _permit = self.inner.semaphore.acquire().await;
        let block = self.replay_block_by_request(request).to_rpc_result()?;
        let config: PostExecReplayConfig = options.unwrap_or_default().into();

        let (tx, rx) = oneshot::channel();
        let this = self.clone();
        self.inner.task_spawner.spawn_blocking_task(async move {
            let replay = || {
                let state_provider = this
                    .inner
                    .provider
                    .state_by_block_hash(block.header().parent_hash())
                    .to_rpc_result()?;
                let db = StateProviderDatabase::new(&state_provider);
                replay_block(&this.inner.evm_config, db, &block, config)
                    .map_err(|err| internal_rpc_err(err.to_string()))
            };
            let _ = tx.send(replay());
        });

        rx.await.map_err(|err| internal_rpc_err(err.to_string()))?
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
    evm_config: EvmConfig,
    task_spawner: Runtime,
    semaphore: Arc<Semaphore>,
}

//! Historical proofs RPC server implementation for `debug_` namespace.

use crate::{
    metrics::{DebugApiExtMetrics, DebugApis},
    state::OpStateProviderFactory,
};
use alloy_consensus::BlockHeader;
use alloy_eips::{BlockId, BlockNumberOrTag};
use alloy_primitives::B256;
use alloy_rlp::Encodable;
use alloy_rpc_types_debug::ExecutionWitness;
use async_trait::async_trait;
use jsonrpsee::proc_macros::rpc;
use jsonrpsee_core::RpcResult;
use jsonrpsee_types::error::ErrorObject;
use reth_basic_payload_builder::PayloadConfig;
use reth_evm::{execute::Executor, ConfigureEvm};
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
use reth_revm::{database::StateProviderDatabase, witness::ExecutionWitnessRecord, State};
use reth_rpc_api::eth::helpers::FullEthApi;
use reth_rpc_eth_types::EthApiError;
use reth_rpc_server_types::{result::internal_rpc_err, ToRpcResult};
use reth_tasks::TaskSpawner;
use serde::{Deserialize, Serialize};
use std::{marker::PhantomData, sync::Arc};
use tokio::sync::{oneshot, Semaphore};

/// Represents the current proofs sync status.
#[derive(Debug, Serialize, Deserialize, Clone, PartialEq, Eq)]
pub struct ProofsSyncStatus {
    /// The earliest block number for which proofs are available.
    earliest: Option<u64>,
    /// The latest block number for which proofs are available.
    latest: Option<u64>,
}

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

    /// Returns the current proofs sync status.
    #[method(name = "proofsSyncStatus")]
    async fn proofs_sync_status(&self) -> RpcResult<ProofsSyncStatus>;
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
    eth_api: Eth,
    storage: OpProofsStorage<Storage>,
    state_provider_factory: OpStateProviderFactory<Eth, Storage>,
    evm_config: EvmConfig,
    task_spawner: Box<dyn TaskSpawner>,
    semaphore: Semaphore,
    _attrs: PhantomData<Attrs>,
    metrics: DebugApiExtMetrics,
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
        storage: OpProofsStorage<P>,
        task_spawner: Box<dyn TaskSpawner>,
        evm_config: EvmConfig,
    ) -> Self {
        Self {
            provider,
            storage: storage.clone(),
            state_provider_factory: OpStateProviderFactory::new(eth_api.clone(), storage),
            eth_api,
            evm_config,
            task_spawner,
            semaphore: Semaphore::new(3),
            _attrs: PhantomData,
            metrics: DebugApiExtMetrics::new(),
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
        self.inner
            .metrics
            .record_operation_async(DebugApis::DebugExecutePayload, async {
                let _permit = self.inner.semaphore.acquire().await;

                let parent_header = self.parent_header(parent_block_hash).to_rpc_result()?;

                let (tx, rx) = oneshot::channel();
                let this = self.inner.clone();
                self.inner.task_spawner.spawn_blocking(Box::pin(async move {
                    let result = async {
                        let parent_hash = parent_header.hash();
                        let attributes = Attrs::try_new(parent_hash, attributes, 3)
                            .map_err(PayloadBuilderError::other)?;

                        let config =
                            PayloadConfig { parent_header: Arc::new(parent_header), attributes };
                        let ctx = OpPayloadBuilderCtx {
                            evm_config: this.evm_config.clone(),
                            chain_spec: this.provider.chain_spec(),
                            config,
                            cancel: Default::default(),
                            best_payload: Default::default(),
                            builder_config: Default::default(),
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
            })
            .await
    }

    async fn execution_witness(&self, block_id: BlockNumberOrTag) -> RpcResult<ExecutionWitness> {
        self.inner
            .metrics
            .record_operation_async(DebugApis::DebugExecutionWitness, async {
                let _permit = self.inner.semaphore.acquire().await;

                let block = self
                    .inner
                    .eth_api
                    .recovered_block(block_id.into())
                    .await?
                    .ok_or(EthApiError::HeaderNotFound(block_id.into()))?;

                let this = self.inner.clone();
                let block_number = block.header().number();

                let state_provider = this
                    .state_provider_factory
                    .state_provider(Some(BlockId::Number(block.parent_num_hash().number.into())))
                    .await
                    .map_err(EthApiError::from)?;
                let db = StateProviderDatabase::new(&state_provider);
                let block_executor = this.eth_api.evm_config().executor(db);

                let mut witness_record = ExecutionWitnessRecord::default();

                let _ = block_executor
                    .execute_with_state_closure(&block, |statedb: &State<_>| {
                        witness_record.record_executed_state(statedb);
                    })
                    .map_err(EthApiError::from)?;

                let ExecutionWitnessRecord { hashed_state, codes, keys, lowest_block_number } =
                    witness_record;

                let state = state_provider
                    .witness(Default::default(), hashed_state)
                    .map_err(EthApiError::from)?;
                let mut exec_witness =
                    ExecutionWitness { state, codes, keys, ..Default::default() };

                let smallest = match lowest_block_number {
                    Some(smallest) => smallest,
                    None => {
                        // Return only the parent header, if there were no calls to the
                        // BLOCKHASH opcode.
                        block_number.saturating_sub(1)
                    }
                };

                let range = smallest..block_number;
                exec_witness.headers = self
                    .inner
                    .provider
                    .headers_range(range)
                    .map_err(EthApiError::from)?
                    .into_iter()
                    .map(|header| {
                        let mut serialized_header = Vec::new();
                        header.encode(&mut serialized_header);
                        serialized_header.into()
                    })
                    .collect();

                Ok(exec_witness)
            })
            .await
    }

    async fn proofs_sync_status(&self) -> RpcResult<ProofsSyncStatus> {
        let earliest = self
            .inner
            .storage
            .get_earliest_block_number()
            .await
            .map_err(|err| internal_rpc_err(err.to_string()))?;
        let latest = self
            .inner
            .storage
            .get_latest_block_number()
            .await
            .map_err(|err| internal_rpc_err(err.to_string()))?;

        Ok(ProofsSyncStatus {
            earliest: earliest.map(|(block_number, _)| block_number),
            latest: latest.map(|(block_number, _)| block_number),
        })
    }
}

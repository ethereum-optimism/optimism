//! RPC component builder

use reth_node_api::AddOnsContext;
use reth_node_builder::rpc::{EngineApiBuilder, PayloadValidatorBuilder};
use reth_node_core::version::{CLIENT_CODE, version_metadata};
use reth_optimism_node::OpEngineTypes;
pub use reth_optimism_rpc::OpEngineApi;
use reth_optimism_rpc::engine::OP_ENGINE_CAPABILITIES;
use reth_payload_builder::PayloadStore;
use reth_rpc_engine_api::EngineCapabilities;

use crate::traits::NodeComponents;
use alloy_eips::eip7685::Requests;
use alloy_primitives::{B256, BlockHash, U64};
use alloy_rpc_types_engine::{
    ClientVersionV1, ExecutionPayloadBodiesV1, ExecutionPayloadInputV2, ExecutionPayloadV3,
    ForkchoiceState, ForkchoiceUpdated, PayloadId, PayloadStatus,
};
use jsonrpsee::proc_macros::rpc;
use jsonrpsee_core::{RpcResult, server::RpcModule};
use op_alloy_rpc_types_engine::{
    OpExecutionPayloadEnvelopeV3, OpExecutionPayloadEnvelopeV4, OpExecutionPayloadV4,
    OpPayloadAttributes, ProtocolVersion, SuperchainSignal,
};
use reth::builder::NodeTypes;
use reth_node_api::{EngineApiValidator, EngineTypes};
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_rpc::OpEngineApiServer;
use reth_rpc_api::IntoEngineApiRpcModule;
use reth_storage_api::{BlockReader, HeaderProvider, StateProviderFactory};
use reth_transaction_pool::TransactionPool;

/// Builder for basic [`OpEngineApi`] implementation.
#[derive(Debug, Clone)]
pub struct OpEngineApiBuilder<EV> {
    engine_validator_builder: EV,
}

impl<EV> Default for OpEngineApiBuilder<EV>
where
    EV: Default,
{
    fn default() -> Self {
        Self {
            engine_validator_builder: EV::default(),
        }
    }
}

impl<N, EV> EngineApiBuilder<N> for OpEngineApiBuilder<EV>
where
    N: NodeComponents,
    EV: PayloadValidatorBuilder<N>,
    EV::Validator: EngineApiValidator<<N::Types as NodeTypes>::Payload>,
{
    type EngineApi = OpEngineApiExt<N::Provider, N::Pool, EV::Validator>;

    async fn build_engine_api(self, ctx: &AddOnsContext<'_, N>) -> eyre::Result<Self::EngineApi> {
        let Self {
            engine_validator_builder,
        } = self;

        let engine_validator = engine_validator_builder.build(ctx).await?;
        let client = ClientVersionV1 {
            code: CLIENT_CODE,
            name: version_metadata().name_client.to_string(),
            version: version_metadata().cargo_pkg_version.to_string(),
            commit: version_metadata().vergen_git_sha.to_string(),
        };
        let inner = reth_rpc_engine_api::EngineApi::new(
            ctx.node.provider().clone(),
            ctx.config.chain.clone(),
            ctx.beacon_engine_handle.clone(),
            PayloadStore::new(ctx.node.payload_builder_handle().clone()),
            ctx.node.pool().clone(),
            Box::new(ctx.node.task_executor().clone()),
            client,
            EngineCapabilities::new(OP_ENGINE_CAPABILITIES.iter().copied()),
            engine_validator,
            ctx.config.engine.accept_execution_requests_hash,
        );

        Ok(OpEngineApiExt::new(OpEngineApi::new(inner)))
    }
}

pub struct OpEngineApiExt<Provider, Pool, Validator> {
    inner: OpEngineApi<Provider, OpEngineTypes, Pool, Validator, OpChainSpec>,
}

impl<Provider, Pool, Validator> OpEngineApiExt<Provider, Pool, Validator>
where
    Provider: HeaderProvider + BlockReader + StateProviderFactory + 'static,
    Pool: TransactionPool + 'static,
    Validator: EngineApiValidator<OpEngineTypes>,
{
    pub fn new(engine: OpEngineApi<Provider, OpEngineTypes, Pool, Validator, OpChainSpec>) -> Self {
        Self { inner: engine }
    }
}

#[async_trait::async_trait]
impl<Provider, Pool, Validator> OpRbuilderEngineApiServer<OpEngineTypes>
    for OpEngineApiExt<Provider, Pool, Validator>
where
    Provider: HeaderProvider + BlockReader + StateProviderFactory + 'static,
    Pool: TransactionPool + 'static,
    Validator: EngineApiValidator<OpEngineTypes>,
{
    async fn new_payload_v2(&self, payload: ExecutionPayloadInputV2) -> RpcResult<PayloadStatus> {
        self.inner.new_payload_v2(payload).await
    }

    async fn new_payload_v3(
        &self,
        payload: ExecutionPayloadV3,
        versioned_hashes: Vec<B256>,
        parent_beacon_block_root: B256,
    ) -> RpcResult<PayloadStatus> {
        self.inner
            .new_payload_v3(payload, versioned_hashes, parent_beacon_block_root)
            .await
    }

    async fn new_payload_v4(
        &self,
        payload: OpExecutionPayloadV4,
        versioned_hashes: Vec<B256>,
        parent_beacon_block_root: B256,
        execution_requests: Requests,
    ) -> RpcResult<PayloadStatus> {
        self.inner
            .new_payload_v4(
                payload,
                versioned_hashes,
                parent_beacon_block_root,
                execution_requests,
            )
            .await
    }

    async fn fork_choice_updated_v1(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<OpPayloadAttributes>,
    ) -> RpcResult<ForkchoiceUpdated> {
        self.inner
            .fork_choice_updated_v1(fork_choice_state, payload_attributes)
            .await
    }

    async fn fork_choice_updated_v2(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<OpPayloadAttributes>,
    ) -> RpcResult<ForkchoiceUpdated> {
        self.inner
            .fork_choice_updated_v2(fork_choice_state, payload_attributes)
            .await
    }

    async fn fork_choice_updated_v3(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<OpPayloadAttributes>,
    ) -> RpcResult<ForkchoiceUpdated> {
        self.inner
            .fork_choice_updated_v3(fork_choice_state, payload_attributes)
            .await
    }

    async fn get_payload_v2(
        &self,
        payload_id: PayloadId,
    ) -> RpcResult<<OpEngineTypes as EngineTypes>::ExecutionPayloadEnvelopeV2> {
        self.inner.get_payload_v2(payload_id).await
    }

    async fn get_payload_v3(
        &self,
        payload_id: PayloadId,
    ) -> RpcResult<OpExecutionPayloadEnvelopeV3> {
        self.inner.get_payload_v3(payload_id).await
    }

    async fn get_payload_v4(
        &self,
        payload_id: PayloadId,
    ) -> RpcResult<OpExecutionPayloadEnvelopeV4> {
        self.inner.get_payload_v4(payload_id).await
    }

    async fn get_payload_bodies_by_hash_v1(
        &self,
        block_hashes: Vec<BlockHash>,
    ) -> RpcResult<ExecutionPayloadBodiesV1> {
        self.inner.get_payload_bodies_by_hash_v1(block_hashes).await
    }

    async fn get_payload_bodies_by_range_v1(
        &self,
        start: U64,
        count: U64,
    ) -> RpcResult<ExecutionPayloadBodiesV1> {
        self.inner
            .get_payload_bodies_by_range_v1(start, count)
            .await
    }

    async fn signal_superchain_v1(&self, signal: SuperchainSignal) -> RpcResult<ProtocolVersion> {
        self.inner.signal_superchain_v1(signal).await
    }

    async fn get_client_version_v1(
        &self,
        client: ClientVersionV1,
    ) -> RpcResult<Vec<ClientVersionV1>> {
        self.inner.get_client_version_v1(client).await
    }

    async fn exchange_capabilities(&self, capabilities: Vec<String>) -> RpcResult<Vec<String>> {
        self.inner.exchange_capabilities(capabilities).await
    }
}

impl<Provider, Pool, Validator> IntoEngineApiRpcModule for OpEngineApiExt<Provider, Pool, Validator>
where
    Self: OpRbuilderEngineApiServer<OpEngineTypes>,
{
    fn into_rpc_module(self) -> RpcModule<()> {
        self.into_rpc().remove_context()
    }
}

/// Extension trait that gives access to Optimism engine API RPC methods.
///
/// Note:
/// > The provider should use a JWT authentication layer.
///
/// This follows the Optimism specs that can be found at:
/// <https://specs.optimism.io/protocol/exec-engine.html#engine-api>
#[rpc(server, namespace = "engine", server_bounds(Engine::PayloadAttributes: jsonrpsee::core::DeserializeOwned))]
pub trait OpRbuilderEngineApi<Engine: EngineTypes> {
    /// Sends the given payload to the execution layer client, as specified for the Shanghai fork.
    ///
    /// See also <https://github.com/ethereum/execution-apis/blob/584905270d8ad665718058060267061ecfd79ca5/src/engine/shanghai.md#engine_newpayloadv2>
    ///
    /// No modifications needed for OP compatibility.
    #[method(name = "newPayloadV2")]
    async fn new_payload_v2(&self, payload: ExecutionPayloadInputV2) -> RpcResult<PayloadStatus>;

    /// Sends the given payload to the execution layer client, as specified for the Cancun fork.
    ///
    /// See also <https://github.com/ethereum/execution-apis/blob/main/src/engine/cancun.md#engine_newpayloadv3>
    ///
    /// OP modifications:
    /// - expected versioned hashes MUST be an empty array: therefore the `versioned_hashes`
    ///   parameter is removed.
    /// - parent beacon block root MUST be the parent beacon block root from the L1 origin block of
    ///   the L2 block.
    /// - blob versioned hashes MUST be empty list.
    #[method(name = "newPayloadV3")]
    async fn new_payload_v3(
        &self,
        payload: ExecutionPayloadV3,
        versioned_hashes: Vec<B256>,
        parent_beacon_block_root: B256,
    ) -> RpcResult<PayloadStatus>;

    /// Sends the given payload to the execution layer client, as specified for the Prague fork.
    ///
    /// See also <https://github.com/ethereum/execution-apis/blob/03911ffc053b8b806123f1fc237184b0092a485a/src/engine/prague.md#engine_newpayloadv4>
    ///
    /// - blob versioned hashes MUST be empty list.
    /// - execution layer requests MUST be empty list.
    #[method(name = "newPayloadV4")]
    async fn new_payload_v4(
        &self,
        payload: OpExecutionPayloadV4,
        versioned_hashes: Vec<B256>,
        parent_beacon_block_root: B256,
        execution_requests: Requests,
    ) -> RpcResult<PayloadStatus>;

    /// See also <https://github.com/ethereum/execution-apis/blob/6709c2a795b707202e93c4f2867fa0bf2640a84f/src/engine/paris.md#engine_forkchoiceupdatedv1>
    ///
    /// This exists because it is used by op-node: <https://github.com/ethereum-optimism/optimism/blob/0bc5fe8d16155dc68bcdf1fa5733abc58689a618/op-node/rollup/types.go#L615-L617>
    ///
    /// Caution: This should not accept the `withdrawals` field in the payload attributes.
    #[method(name = "forkchoiceUpdatedV1")]
    async fn fork_choice_updated_v1(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<Engine::PayloadAttributes>,
    ) -> RpcResult<ForkchoiceUpdated>;

    /// Updates the execution layer client with the given fork choice, as specified for the Shanghai
    /// fork.
    ///
    /// Caution: This should not accept the `parentBeaconBlockRoot` field in the payload attributes.
    ///
    /// See also <https://github.com/ethereum/execution-apis/blob/6709c2a795b707202e93c4f2867fa0bf2640a84f/src/engine/shanghai.md#engine_forkchoiceupdatedv2>
    ///
    /// OP modifications:
    /// - The `payload_attributes` parameter is extended with the [`EngineTypes::PayloadAttributes`](EngineTypes) type as described in <https://specs.optimism.io/protocol/exec-engine.html#extended-payloadattributesv2>
    #[method(name = "forkchoiceUpdatedV2")]
    async fn fork_choice_updated_v2(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<Engine::PayloadAttributes>,
    ) -> RpcResult<ForkchoiceUpdated>;

    /// Updates the execution layer client with the given fork choice, as specified for the Cancun
    /// fork.
    ///
    /// See also <https://github.com/ethereum/execution-apis/blob/main/src/engine/cancun.md#engine_forkchoiceupdatedv3>
    ///
    /// OP modifications:
    /// - Must be called with an Ecotone payload
    /// - Attributes must contain the parent beacon block root field
    /// - The `payload_attributes` parameter is extended with the [`EngineTypes::PayloadAttributes`](EngineTypes) type as described in <https://specs.optimism.io/protocol/exec-engine.html#extended-payloadattributesv2>
    #[method(name = "forkchoiceUpdatedV3")]
    async fn fork_choice_updated_v3(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<Engine::PayloadAttributes>,
    ) -> RpcResult<ForkchoiceUpdated>;

    /// Retrieves an execution payload from a previously started build process, as specified for the
    /// Shanghai fork.
    ///
    /// See also <https://github.com/ethereum/execution-apis/blob/6709c2a795b707202e93c4f2867fa0bf2640a84f/src/engine/shanghai.md#engine_getpayloadv2>
    ///
    /// Note:
    /// > Provider software MAY stop the corresponding build process after serving this call.
    ///
    /// No modifications needed for OP compatibility.
    #[method(name = "getPayloadV2")]
    async fn get_payload_v2(
        &self,
        payload_id: PayloadId,
    ) -> RpcResult<Engine::ExecutionPayloadEnvelopeV2>;

    /// Retrieves an execution payload from a previously started build process, as specified for the
    /// Cancun fork.
    ///
    /// See also <https://github.com/ethereum/execution-apis/blob/main/src/engine/cancun.md#engine_getpayloadv3>
    ///
    /// Note:
    /// > Provider software MAY stop the corresponding build process after serving this call.
    ///
    /// OP modifications:
    /// - the response type is extended to [`EngineTypes::ExecutionPayloadEnvelopeV3`].
    #[method(name = "getPayloadV3")]
    async fn get_payload_v3(
        &self,
        payload_id: PayloadId,
    ) -> RpcResult<Engine::ExecutionPayloadEnvelopeV3>;

    /// Returns the most recent version of the payload that is available in the corresponding
    /// payload build process at the time of receiving this call.
    ///
    /// See also <https://github.com/ethereum/execution-apis/blob/main/src/engine/prague.md#engine_getpayloadv4>
    ///
    /// Note:
    /// > Provider software MAY stop the corresponding build process after serving this call.
    ///
    /// OP modifications:
    /// - the response type is extended to [`EngineTypes::ExecutionPayloadEnvelopeV4`].
    #[method(name = "getPayloadV4")]
    async fn get_payload_v4(
        &self,
        payload_id: PayloadId,
    ) -> RpcResult<Engine::ExecutionPayloadEnvelopeV4>;

    /// Returns the execution payload bodies by the given hash.
    ///
    /// See also <https://github.com/ethereum/execution-apis/blob/6452a6b194d7db269bf1dbd087a267251d3cc7f8/src/engine/shanghai.md#engine_getpayloadbodiesbyhashv1>
    #[method(name = "getPayloadBodiesByHashV1")]
    async fn get_payload_bodies_by_hash_v1(
        &self,
        block_hashes: Vec<BlockHash>,
    ) -> RpcResult<ExecutionPayloadBodiesV1>;

    /// Returns the execution payload bodies by the range starting at `start`, containing `count`
    /// blocks.
    ///
    /// WARNING: This method is associated with the BeaconBlocksByRange message in the consensus
    /// layer p2p specification, meaning the input should be treated as untrusted or potentially
    /// adversarial.
    ///
    /// Implementers should take care when acting on the input to this method, specifically
    /// ensuring that the range is limited properly, and that the range boundaries are computed
    /// correctly and without panics.
    ///
    /// See also <https://github.com/ethereum/execution-apis/blob/6452a6b194d7db269bf1dbd087a267251d3cc7f8/src/engine/shanghai.md#engine_getpayloadbodiesbyrangev1>
    #[method(name = "getPayloadBodiesByRangeV1")]
    async fn get_payload_bodies_by_range_v1(
        &self,
        start: U64,
        count: U64,
    ) -> RpcResult<ExecutionPayloadBodiesV1>;

    /// Signals superchain information to the Engine.
    /// Returns the latest supported OP-Stack protocol version of the execution engine.
    /// See also <https://specs.optimism.io/protocol/exec-engine.html#engine_signalsuperchainv1>
    #[method(name = "engine_signalSuperchainV1")]
    async fn signal_superchain_v1(&self, _signal: SuperchainSignal) -> RpcResult<ProtocolVersion>;

    /// Returns the execution client version information.
    ///
    /// Note:
    /// > The `client_version` parameter identifies the consensus client.
    ///
    /// See also <https://github.com/ethereum/execution-apis/blob/main/src/engine/identification.md#engine_getclientversionv1>
    #[method(name = "getClientVersionV1")]
    async fn get_client_version_v1(
        &self,
        client_version: ClientVersionV1,
    ) -> RpcResult<Vec<ClientVersionV1>>;

    /// Returns the list of Engine API methods supported by the execution layer client software.
    ///
    /// See also <https://github.com/ethereum/execution-apis/blob/6452a6b194d7db269bf1dbd087a267251d3cc7f8/src/engine/common.md#capabilities>
    #[method(name = "exchangeCapabilities")]
    async fn exchange_capabilities(&self, capabilities: Vec<String>) -> RpcResult<Vec<String>>;
}

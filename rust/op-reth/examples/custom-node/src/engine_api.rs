use crate::{
    engine::{CustomExecutionData, CustomPayloadAttributes, CustomPayloadTypes},
    primitives::CustomNodePrimitives,
    CustomNode,
};
use alloy_rpc_types_engine::{
    ExecutionPayloadV3, ForkchoiceState, ForkchoiceUpdated, PayloadId, PayloadStatus,
};
use async_trait::async_trait;
use jsonrpsee::{core::RpcResult, proc_macros::rpc, RpcModule};
use reth_ethereum::node::api::{
    AddOnsContext, ConsensusEngineHandle, EngineApiMessageVersion, FullNodeComponents,
};
use reth_node_builder::rpc::EngineApiBuilder;
use reth_op::node::OpBuiltPayload;
use reth_payload_builder::PayloadStore;
use reth_rpc_api::IntoEngineApiRpcModule;
use reth_rpc_engine_api::EngineApiError;
use std::sync::Arc;

#[derive(serde::Deserialize)]
pub struct CustomExecutionPayloadInput {}

#[derive(Clone, serde::Serialize)]
pub struct CustomExecutionPayloadEnvelope {
    execution_payload: ExecutionPayloadV3,
    extension: u64,
}

impl From<OpBuiltPayload<CustomNodePrimitives>> for CustomExecutionPayloadEnvelope {
    fn from(value: OpBuiltPayload<CustomNodePrimitives>) -> Self {
        let sealed_block = value.into_sealed_block();
        let hash = sealed_block.hash();
        let extension = sealed_block.header().extension;
        let block = sealed_block.into_block();

        Self {
            execution_payload: ExecutionPayloadV3::from_block_unchecked(hash, &block),
            extension,
        }
    }
}

#[rpc(server, namespace = "engine")]
pub trait CustomEngineApi {
    #[method(name = "newPayload")]
    async fn new_payload(&self, payload: CustomExecutionData) -> RpcResult<PayloadStatus>;

    #[method(name = "forkchoiceUpdated")]
    async fn fork_choice_updated(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<CustomPayloadAttributes>,
    ) -> RpcResult<ForkchoiceUpdated>;

    #[method(name = "getPayload")]
    async fn get_payload(&self, payload_id: PayloadId)
        -> RpcResult<CustomExecutionPayloadEnvelope>;
}

pub struct CustomEngineApi {
    inner: Arc<CustomEngineApiInner>,
}

struct CustomEngineApiInner {
    beacon_consensus: ConsensusEngineHandle<CustomPayloadTypes>,
    payload_store: PayloadStore<CustomPayloadTypes>,
}

impl CustomEngineApiInner {
    fn new(
        beacon_consensus: ConsensusEngineHandle<CustomPayloadTypes>,
        payload_store: PayloadStore<CustomPayloadTypes>,
    ) -> Self {
        Self { beacon_consensus, payload_store }
    }
}

#[async_trait]
impl CustomEngineApiServer for CustomEngineApi {
    async fn new_payload(&self, payload: CustomExecutionData) -> RpcResult<PayloadStatus> {
        Ok(self
            .inner
            .beacon_consensus
            .new_payload(payload)
            .await
            .map_err(EngineApiError::NewPayload)?)
    }

    async fn fork_choice_updated(
        &self,
        fork_choice_state: ForkchoiceState,
        payload_attributes: Option<CustomPayloadAttributes>,
    ) -> RpcResult<ForkchoiceUpdated> {
        Ok(self
            .inner
            .beacon_consensus
            .fork_choice_updated(fork_choice_state, payload_attributes, EngineApiMessageVersion::V3)
            .await
            .map_err(EngineApiError::ForkChoiceUpdate)?)
    }

    async fn get_payload(
        &self,
        payload_id: PayloadId,
    ) -> RpcResult<CustomExecutionPayloadEnvelope> {
        Ok(self
            .inner
            .payload_store
            .resolve(payload_id)
            .await
            .ok_or(EngineApiError::UnknownPayload)?
            .map_err(|_| EngineApiError::UnknownPayload)?
            .into())
    }
}

impl IntoEngineApiRpcModule for CustomEngineApi
where
    Self: CustomEngineApiServer,
{
    fn into_rpc_module(self) -> RpcModule<()> {
        self.into_rpc().remove_context()
    }
}

#[derive(Debug, Default, Clone)]
pub struct CustomEngineApiBuilder {}

impl<N> EngineApiBuilder<N> for CustomEngineApiBuilder
where
    N: FullNodeComponents<Types = CustomNode>,
{
    type EngineApi = CustomEngineApi;

    async fn build_engine_api(self, ctx: &AddOnsContext<'_, N>) -> eyre::Result<Self::EngineApi> {
        Ok(CustomEngineApi {
            inner: Arc::new(CustomEngineApiInner::new(
                ctx.beacon_engine_handle.clone(),
                PayloadStore::new(ctx.node.payload_builder_handle().clone()),
            )),
        })
    }
}

//! RPC component builder

pub use reth_optimism_rpc::OpEngineApi;

use crate::OP_NAME_CLIENT;
use alloy_rpc_types_engine::ClientVersionV1;
use op_alloy_rpc_types_engine::OpExecutionData;
use reth_chainspec::EthereumHardforks;
use reth_node_api::{
    AddOnsContext, EngineTypes, FullNodeComponents, NodeTypes, NodeTypesWithEngine,
};
use reth_node_builder::rpc::{EngineApiBuilder, EngineValidatorBuilder};
use reth_node_core::version::{CARGO_PKG_VERSION, CLIENT_CODE, VERGEN_GIT_SHA};
use reth_optimism_rpc::engine::OP_ENGINE_CAPABILITIES;
use reth_payload_builder::PayloadStore;
use reth_rpc_engine_api::{EngineApi, EngineCapabilities};

/// Builder for basic [`OpEngineApi`] implementation.
#[derive(Debug, Default)]
pub struct OpEngineApiBuilder<EV> {
    engine_validator_builder: EV,
}

impl<N, EV> EngineApiBuilder<N> for OpEngineApiBuilder<EV>
where
    N: FullNodeComponents<
        Types: NodeTypesWithEngine<
            ChainSpec: EthereumHardforks,
            Engine: EngineTypes<ExecutionData = OpExecutionData>,
        >,
    >,
    EV: EngineValidatorBuilder<N>,
{
    type EngineApi = OpEngineApi<
        N::Provider,
        <N::Types as NodeTypesWithEngine>::Engine,
        N::Pool,
        EV::Validator,
        <N::Types as NodeTypes>::ChainSpec,
    >;

    async fn build_engine_api(self, ctx: &AddOnsContext<'_, N>) -> eyre::Result<Self::EngineApi> {
        let Self { engine_validator_builder } = self;

        let engine_validator = engine_validator_builder.build(ctx).await?;
        let client = ClientVersionV1 {
            code: CLIENT_CODE,
            name: OP_NAME_CLIENT.to_string(),
            version: CARGO_PKG_VERSION.to_string(),
            commit: VERGEN_GIT_SHA.to_string(),
        };
        let inner = EngineApi::new(
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

        Ok(OpEngineApi::new(inner))
    }
}

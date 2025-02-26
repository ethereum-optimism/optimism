//! RPC component builder

use reth_optimism_rpc::engine::OP_CAPABILITIES;
pub use reth_optimism_rpc::OpEngineApi;

use alloy_rpc_types_engine::ExecutionData;
use reth_chainspec::EthereumHardforks;
use reth_node_api::{
    AddOnsContext, EngineTypes, FullNodeComponents, NodeTypes, NodeTypesWithEngine,
};
use reth_node_builder::rpc::{BasicEngineApiBuilder, EngineApiBuilder, EngineValidatorBuilder};

/// Builder for basic [`OpEngineApi`] implementation.
#[derive(Debug, Default)]
pub struct OpEngineApiBuilder<EV> {
    inner: BasicEngineApiBuilder<EV>,
}

impl<N, EV> EngineApiBuilder<N> for OpEngineApiBuilder<EV>
where
    N: FullNodeComponents<
        Types: NodeTypesWithEngine<
            ChainSpec: EthereumHardforks,
            Engine: EngineTypes<ExecutionData = ExecutionData>,
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
        let inner = self.inner.capabilities(OP_CAPABILITIES).build_engine_api(ctx).await?;

        Ok(OpEngineApi::new(inner))
    }
}

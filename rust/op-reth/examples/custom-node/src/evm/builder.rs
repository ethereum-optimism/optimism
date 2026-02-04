use crate::{chainspec::CustomChainSpec, evm::CustomEvmConfig, primitives::CustomNodePrimitives};
use reth_ethereum::node::api::FullNodeTypes;
use reth_node_builder::{components::ExecutorBuilder, BuilderContext, NodeTypes};
use std::{future, future::Future};

#[derive(Debug, Clone, Default)]
#[non_exhaustive]
pub struct CustomExecutorBuilder;

impl<Node: FullNodeTypes> ExecutorBuilder<Node> for CustomExecutorBuilder
where
    Node::Types: NodeTypes<ChainSpec = CustomChainSpec, Primitives = CustomNodePrimitives>,
{
    type EVM = CustomEvmConfig;

    fn build_evm(
        self,
        ctx: &BuilderContext<Node>,
    ) -> impl Future<Output = eyre::Result<Self::EVM>> + Send {
        future::ready(Ok(CustomEvmConfig::new(ctx.chain_spec())))
    }
}

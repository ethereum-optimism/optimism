//! This example shows how to implement a custom node.
//!
//! A node consists of:
//! - primitives: block,header,transactions
//! - components: network,pool,evm
//! - engine: advances the node

#![cfg_attr(not(test), warn(unused_crate_dependencies))]

use crate::{
    engine::{CustomEngineValidatorBuilder, CustomPayloadTypes},
    engine_api::CustomEngineApiBuilder,
    evm::CustomExecutorBuilder,
    pool::CustomPooledTransaction,
    primitives::CustomTransaction,
    rpc::CustomRpcTypes,
};
use chainspec::CustomChainSpec;
use primitives::CustomNodePrimitives;
use reth_ethereum::node::api::{FullNodeTypes, NodeTypes};
use reth_node_builder::{
    components::{BasicPayloadServiceBuilder, ComponentsBuilder},
    Node, NodeAdapter,
};
use reth_op::{
    node::{
        node::{OpConsensusBuilder, OpNetworkBuilder, OpPayloadBuilder, OpPoolBuilder},
        txpool, OpAddOns, OpNode,
    },
    rpc::OpEthApiBuilder,
};

pub mod chainspec;
pub mod engine;
pub mod engine_api;
pub mod evm;
pub mod pool;
pub mod primitives;
pub mod rpc;

#[derive(Debug, Clone)]
pub struct CustomNode {
    inner: OpNode,
}

impl NodeTypes for CustomNode {
    type Primitives = CustomNodePrimitives;
    type ChainSpec = CustomChainSpec;
    type Storage = <OpNode as NodeTypes>::Storage;
    type Payload = CustomPayloadTypes;
}

impl<N> Node<N> for CustomNode
where
    N: FullNodeTypes<Types = Self>,
{
    type ComponentsBuilder = ComponentsBuilder<
        N,
        OpPoolBuilder<txpool::OpPooledTransaction<CustomTransaction, CustomPooledTransaction>>,
        BasicPayloadServiceBuilder<OpPayloadBuilder>,
        OpNetworkBuilder,
        CustomExecutorBuilder,
        OpConsensusBuilder,
    >;

    type AddOns = OpAddOns<
        NodeAdapter<N>,
        OpEthApiBuilder<CustomRpcTypes>,
        CustomEngineValidatorBuilder,
        CustomEngineApiBuilder,
    >;

    fn components_builder(&self) -> Self::ComponentsBuilder {
        ComponentsBuilder::default()
            .node_types::<N>()
            .pool(OpPoolBuilder::default())
            .executor(CustomExecutorBuilder::default())
            .payload(BasicPayloadServiceBuilder::new(OpPayloadBuilder::new(false)))
            .network(OpNetworkBuilder::new(false, false))
            .consensus(OpConsensusBuilder::default())
    }

    fn add_ons(&self) -> Self::AddOns {
        self.inner.add_ons_builder().build()
    }
}

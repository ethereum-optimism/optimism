//! This example shows how to implement a custom node.
//!
//! A node consists of:
//! - primitives: block,header,transactions
//! - components: network,pool,evm
//! - engine: advances the node

#![cfg_attr(not(test), warn(unused_crate_dependencies))]

// Required for feature forwarding
use reth_ethereum as _;
use reth_payload_primitives as _;

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
use reth_node_api::FullNodeTypes;
use reth_node_builder::{
    Node, NodeAdapter, NodeTypes,
    components::{BasicPayloadServiceBuilder, ComponentsBuilder, NodeComponentsBuilder},
    rpc::BasicEngineValidatorBuilder,
};
use reth_op::{
    node::{
        OpAddOns, OpNode,
        node::{OpConsensusBuilder, OpNetworkBuilder, OpPayloadBuilder, OpPoolBuilder},
        txpool,
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

#[derive(Debug, Clone, Default)]
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
        NodeAdapter<N, <Self::ComponentsBuilder as NodeComponentsBuilder<N>>::Components>,
        OpEthApiBuilder<CustomRpcTypes>,
        CustomEngineValidatorBuilder,
        CustomEngineApiBuilder,
        BasicEngineValidatorBuilder<CustomEngineValidatorBuilder>,
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

//! OP payload-service configuration.
//!
//! Normal launches delegate to
//! [`BasicPayloadServiceBuilder`](reth_node_builder::components::BasicPayloadServiceBuilder) with
//! the stock EVM configuration.
//! The hidden, default-off SDM testing control replaces that configuration once, when the payload
//! service is constructed, so policy selection never occurs during transaction execution.

use crate::{
    node::{OpNodeTypes, OpPayloadBuilder},
    sdm_test_policy::{FixedRefundPolicy, configure_excessive_refund_target},
};
use alloy_primitives::Address;
use op_alloy_consensus::OpTxEnvelope;
use reth_node_api::{BuildNextEnv, NodeTypes, node::FullNodeTypes};
use reth_node_builder::{
    BuilderContext,
    components::{BasicPayloadServiceBuilder, PayloadBuilderBuilder, PayloadServiceBuilder},
};
use reth_optimism_evm::{
    ConfigurePostExecEvm, OpEvmConfig, OpEvmFactory, OpRethReceiptBuilder, OpTx,
    PostExecEvmFactoryAdapter,
};
use reth_optimism_payload_builder::OpPayloadBuilderAttributes;
use reth_optimism_primitives::OpPrimitives;
use reth_payload_builder::PayloadBuilderHandle;
use reth_transaction_pool::TransactionPool;
use std::sync::Arc;

/// EVM configuration used only by the fixed SDM test policy.
type FixedPolicyOpEvmConfig<ChainSpec, N> = OpEvmConfig<
    ChainSpec,
    N,
    OpRethReceiptBuilder,
    PostExecEvmFactoryAdapter<OpEvmFactory<OpTx, FixedRefundPolicy>>,
>;

fn fixed_policy_evm_config<ChainSpec, N>(
    chain_spec: Arc<ChainSpec>,
) -> FixedPolicyOpEvmConfig<ChainSpec, N>
where
    N: reth_node_api::NodePrimitives,
{
    OpEvmConfig::new_with_evm_factory(
        chain_spec,
        OpRethReceiptBuilder::default(),
        PostExecEvmFactoryAdapter::new(OpEvmFactory::default()),
    )
}

/// Builds the payload service for an OP node.
///
/// By default this delegates to the stock [`BasicPayloadServiceBuilder`] and EVM configuration. A
/// hidden test-only configuration may instead select the deterministic fixed-refund SDM policy.
/// The choice is made once while constructing the service.
#[derive(Debug, Clone)]
pub struct OpPayloadServiceBuilder {
    payload_builder: OpPayloadBuilder,
    /// `None` selects the stock policy; `Some(target)` selects the fixed test policy with an
    /// optional excessive-refund target.
    testing_sdm_fixed_policy: Option<Option<Address>>,
}

impl OpPayloadServiceBuilder {
    /// Creates an OP payload-service builder with an optional test-only fixed SDM policy.
    pub const fn new(
        payload_builder: OpPayloadBuilder,
        testing_sdm_fixed_policy: Option<Option<Address>>,
    ) -> Self {
        Self { payload_builder, testing_sdm_fixed_policy }
    }
}

impl<Node, Pool, EvmConfig> PayloadServiceBuilder<Node, Pool, EvmConfig> for OpPayloadServiceBuilder
where
    Node: FullNodeTypes<Types: OpNodeTypes>,
    Pool: TransactionPool + Clone + Send + Sync + Unpin + 'static,
    EvmConfig: Send,
    FixedPolicyOpEvmConfig<<Node::Types as NodeTypes>::ChainSpec, OpPrimitives>:
        ConfigurePostExecEvm<
                Primitives = OpPrimitives,
                NextBlockEnvCtx: BuildNextEnv<
                    OpPayloadBuilderAttributes<OpTxEnvelope>,
                    alloy_consensus::Header,
                    <Node::Types as NodeTypes>::ChainSpec,
                >,
            > + Clone
            + Send
            + Sync
            + Unpin
            + 'static,
    OpPayloadBuilder: PayloadBuilderBuilder<Node, Pool, EvmConfig>
        + PayloadBuilderBuilder<
            Node,
            Pool,
            FixedPolicyOpEvmConfig<<Node::Types as NodeTypes>::ChainSpec, OpPrimitives>,
        >,
{
    async fn spawn_payload_builder_service(
        self,
        ctx: &BuilderContext<Node>,
        pool: Pool,
        evm_config: EvmConfig,
    ) -> eyre::Result<PayloadBuilderHandle<<Node::Types as NodeTypes>::Payload>> {
        let Self { payload_builder, testing_sdm_fixed_policy } = self;
        let Some(excessive_refund_target) = testing_sdm_fixed_policy else {
            return BasicPayloadServiceBuilder::new(payload_builder)
                .spawn_payload_builder_service(ctx, pool, evm_config)
                .await;
        };

        configure_excessive_refund_target(excessive_refund_target)?;
        let evm_config = fixed_policy_evm_config::<
            <Node::Types as NodeTypes>::ChainSpec,
            OpPrimitives,
        >(ctx.chain_spec());
        BasicPayloadServiceBuilder::new(payload_builder)
            .spawn_payload_builder_service(ctx, pool, evm_config)
            .await
    }
}

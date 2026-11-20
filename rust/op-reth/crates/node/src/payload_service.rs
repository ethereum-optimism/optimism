//! OP payload-service configuration.
//!
//! Normal launches delegate to [`BasicPayloadServiceBuilder`] with the EVM configuration the node
//! hands the payload service: the stock one, or the one [`WithPayloadEvm`] builds. The hidden,
//! default-off SDM testing control replaces that configuration once, when the payload service is
//! constructed, so policy selection never occurs during transaction execution.

use crate::{
    node::{OpNodeTypes, OpPayloadBuilder},
    sdm_test_policy::FixedRefundPolicyFactory,
};
use alloy_primitives::Address;
use reth_node_api::{NodeTypes, node::FullNodeTypes};
use reth_node_builder::{
    BuilderContext,
    components::{BasicPayloadServiceBuilder, PayloadBuilderBuilder, PayloadServiceBuilder},
};
use reth_optimism_evm::{OpEvmConfig, OpEvmFactory, OpTx, PostExecEvmFactoryAdapter};
use reth_payload_builder::PayloadBuilderHandle;
use reth_transaction_pool::TransactionPool;
/// EVM factory used only by the fixed SDM test policy.
type FixedPolicyEvmFactory =
    PostExecEvmFactoryAdapter<OpEvmFactory<OpTx, FixedRefundPolicyFactory>>;

/// EVM configuration used only by the fixed SDM test policy.
type FixedPolicyOpEvmConfig = OpEvmConfig<FixedPolicyEvmFactory>;

/// The EVM factory of the fixed SDM test policy, with an optional excessive-refund target.
pub(crate) const fn fixed_policy_evm_factory(
    excessive_refund_target: Option<Address>,
) -> FixedPolicyEvmFactory {
    PostExecEvmFactoryAdapter::new(OpEvmFactory::new(FixedRefundPolicyFactory::new(
        excessive_refund_target,
    )))
}

/// Builds the payload service for an OP node.
///
/// By default this delegates to the stock [`BasicPayloadServiceBuilder`] with the EVM configuration
/// it is handed: the node's, or the one [`WithPayloadEvm`] builds. A hidden test-only
/// configuration may instead select the deterministic fixed-refund SDM policy, which replaces
/// either. The choice is made once while constructing the service.
///
/// The service must be wrapped in [`DerivedPayloadServiceBuilder`], as [`OpNode`] does.
///
/// [`DerivedPayloadServiceBuilder`]: crate::DerivedPayloadServiceBuilder
/// [`OpNode`]: crate::OpNode
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
    OpPayloadBuilder: PayloadBuilderBuilder<Node, Pool, EvmConfig>
        + PayloadBuilderBuilder<Node, Pool, FixedPolicyOpEvmConfig>,
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

        let evm_config: FixedPolicyOpEvmConfig = OpEvmConfig::new_with_evm_factory(
            ctx.chain_spec(),
            fixed_policy_evm_factory(excessive_refund_target),
        );
        BasicPayloadServiceBuilder::new(payload_builder)
            .spawn_payload_builder_service(ctx, pool, evm_config)
            .await
    }
}

/// A payload service that builds payloads with the EVM factory `F` instead of the node's.
///
/// The node imports blocks with its own EVM configuration and hands it to the payload service as
/// well. This hands the wrapped payload service `S` an [`OpEvmConfig`] with `F` instead, so a node
/// can build blocks with another EVM, for example one with a refund policy, without writing a
/// payload service. Derived payloads are still built with the node's EVM configuration, by the
/// [`DerivedPayloadServiceBuilder`] the node wraps this in. op-reth's [`OpPayloadServiceBuilder`]
/// replaces `F` with its fixed-refund policy under the hidden `--testing.sdm-fixed-policy`.
///
/// It takes only the node's EVM configuration, so it cannot wrap itself: one factory applies.
///
/// [`OpNode::with_payload_evm`] and [`CustomOpNode::with_payload_evm`] create it.
///
/// [`DerivedPayloadServiceBuilder`]: crate::DerivedPayloadServiceBuilder
/// [`OpNode::with_payload_evm`]: crate::OpNode::with_payload_evm
/// [`CustomOpNode::with_payload_evm`]: crate::CustomOpNode::with_payload_evm
#[derive(Debug, Clone)]
pub struct WithPayloadEvm<S, F> {
    inner: S,
    evm_factory: F,
}

impl<S, F> WithPayloadEvm<S, F> {
    /// Wraps `inner`, which then builds payloads with `evm_factory`.
    pub const fn new(inner: S, evm_factory: F) -> Self {
        Self { inner, evm_factory }
    }
}

impl<Node, Pool, S, F> PayloadServiceBuilder<Node, Pool, OpEvmConfig> for WithPayloadEvm<S, F>
where
    Node: FullNodeTypes<Types: OpNodeTypes>,
    Pool: TransactionPool,
    S: PayloadServiceBuilder<Node, Pool, OpEvmConfig<F>>,
    F: Send,
{
    async fn spawn_payload_builder_service(
        self,
        ctx: &BuilderContext<Node>,
        pool: Pool,
        _node_evm_config: OpEvmConfig,
    ) -> eyre::Result<PayloadBuilderHandle<<Node::Types as NodeTypes>::Payload>> {
        let evm_config = OpEvmConfig::new_with_evm_factory(ctx.chain_spec(), self.evm_factory);
        self.inner.spawn_payload_builder_service(ctx, pool, evm_config).await
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::OpNode;
    use reth_db::test_utils::TempDatabase;
    use reth_db_common::init::init_genesis;
    use reth_node_api::NodeTypesWithDBAdapter;
    use reth_node_builder::{FullNodeTypesAdapter, NodeConfig, common::WithConfigs};
    use reth_optimism_chainspec::OP_SEPOLIA;
    use reth_optimism_txpool::OpPooledTransaction;
    use reth_provider::{
        providers::BlockchainProvider, test_utils::create_test_provider_factory_with_node_types,
    };
    use reth_tasks::Runtime;
    use reth_transaction_pool::noop::NoopTransactionPool;
    use std::sync::{Arc, Mutex};

    type TestDb = Arc<TempDatabase<reth_db::DatabaseEnv>>;
    type TestNode = FullNodeTypesAdapter<
        OpNode,
        TestDb,
        BlockchainProvider<NodeTypesWithDBAdapter<OpNode, TestDb>>,
    >;
    type TestPool = NoopTransactionPool<OpPooledTransaction>;

    /// An EVM factory that only identifies itself.
    #[derive(Debug, Clone, PartialEq, Eq)]
    struct TestEvmFactory(u8);

    /// A payload service that records the EVM configuration it is handed, then fails.
    struct Recording<Evm>(Arc<Mutex<Option<Evm>>>);

    impl<Evm: Send> PayloadServiceBuilder<TestNode, TestPool, Evm> for Recording<Evm> {
        async fn spawn_payload_builder_service(
            self,
            _ctx: &BuilderContext<TestNode>,
            _pool: TestPool,
            evm_config: Evm,
        ) -> eyre::Result<PayloadBuilderHandle<<OpNode as NodeTypes>::Payload>> {
            *self.0.lock().unwrap() = Some(evm_config);
            Err(eyre::eyre!("recorded"))
        }
    }

    fn builder_context() -> BuilderContext<TestNode> {
        let factory = create_test_provider_factory_with_node_types::<OpNode>(OP_SEPOLIA.clone());
        init_genesis(&factory).unwrap();
        BuilderContext::new(
            Default::default(),
            BlockchainProvider::new(factory).unwrap(),
            Runtime::test(),
            WithConfigs {
                config: NodeConfig::new(OP_SEPOLIA.clone()),
                toml_config: Default::default(),
            },
        )
    }

    /// [`WithPayloadEvm`] hands the payload service an EVM configuration with its factory, not the
    /// node's, which the node keeps importing blocks with.
    #[tokio::test]
    async fn payload_service_builds_with_the_payload_evm() {
        let ctx = builder_context();
        let recorded = Arc::new(Mutex::new(None));
        let node_evm_config = OpEvmConfig::optimism(OP_SEPOLIA.clone());

        let err = WithPayloadEvm::new(Recording(recorded.clone()), TestEvmFactory(7))
            .spawn_payload_builder_service(&ctx, TestPool::new(), node_evm_config)
            .await
            .unwrap_err();
        assert_eq!(err.to_string(), "recorded");

        let evm_config: OpEvmConfig<TestEvmFactory> = recorded.lock().unwrap().take().unwrap();
        assert_eq!(evm_config.executor_factory.evm_factory(), &TestEvmFactory(7));
        assert!(Arc::ptr_eq(evm_config.chain_spec(), &ctx.chain_spec()));
    }
}

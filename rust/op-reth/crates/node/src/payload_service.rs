//! OP payload-service configuration.
//!
//! The service creates jobs with an [`OpPayloadJobGenerator`]. Normal launches use the stock EVM
//! configuration.
//! The hidden, default-off SDM testing control replaces that configuration once, when the payload
//! service is constructed, so policy selection never occurs during transaction execution.

use crate::{
    node::{OpNodeTypes, OpPayloadBuilder},
    payload_job::OpPayloadJobGenerator,
    sdm_test_policy::FixedRefundPolicyFactory,
};
use alloy_primitives::Address;
use op_alloy_consensus::OpTxEnvelope;
use reth_basic_payload_builder::{BasicPayloadJobGenerator, BasicPayloadJobGeneratorConfig};
use reth_chain_state::CanonStateSubscriptions;
use reth_node_api::{BuildNextEnv, NodeTypes, node::FullNodeTypes};
use reth_node_builder::{
    BuilderContext,
    components::{PayloadBuilderBuilder, PayloadServiceBuilder},
};
use reth_optimism_evm::{
    ConfigurePostExecEvm, OpEvmConfig, OpEvmFactory, OpRethReceiptBuilder, OpTx,
    PostExecEvmFactoryAdapter,
};
use reth_optimism_payload_builder::OpPayloadBuilderAttributes;
use reth_optimism_primitives::OpPrimitives;
use reth_payload_builder::{PayloadBuilderHandle, PayloadBuilderService};
use reth_transaction_pool::TransactionPool;
/// EVM configuration used only by the fixed SDM test policy.
type FixedPolicyOpEvmConfig<ChainSpec, N> = OpEvmConfig<
    ChainSpec,
    N,
    OpRethReceiptBuilder,
    PostExecEvmFactoryAdapter<OpEvmFactory<OpTx, FixedRefundPolicyFactory>>,
>;

/// Builds the payload service for an OP node.
///
/// By default the service uses the stock EVM configuration. A hidden test-only configuration may
/// instead select the deterministic fixed-refund SDM policy. The choice is made once while
/// constructing the service.
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
            return OpBasicPayloadServiceBuilder::new(payload_builder)
                .spawn_payload_builder_service(ctx, pool, evm_config)
                .await;
        };

        let evm_config: FixedPolicyOpEvmConfig<
            <Node::Types as NodeTypes>::ChainSpec,
            OpPrimitives,
        > = OpEvmConfig::new_with_evm_factory(
            ctx.chain_spec(),
            OpRethReceiptBuilder::default(),
            PostExecEvmFactoryAdapter::new(OpEvmFactory::new(FixedRefundPolicyFactory::new(
                excessive_refund_target,
            ))),
        );
        OpBasicPayloadServiceBuilder::new(payload_builder)
            .spawn_payload_builder_service(ctx, pool, evm_config)
            .await
    }
}

/// Spawns a payload service whose jobs come from an [`OpPayloadJobGenerator`] wrapping a
/// [`BasicPayloadJobGenerator`].
///
/// A node that replaces [`OpPayloadServiceBuilder`] uses this instead of
/// [`BasicPayloadServiceBuilder`](reth_node_builder::components::BasicPayloadServiceBuilder) to
/// keep rejecting invalid derived `PostExec` data at `engine_forkchoiceUpdated`.
#[derive(Debug, Clone)]
pub struct OpBasicPayloadServiceBuilder<PB> {
    payload_builder_builder: PB,
}

impl<PB> OpBasicPayloadServiceBuilder<PB> {
    /// Creates a service builder whose jobs use the payload builder `payload_builder_builder`
    /// builds.
    pub const fn new(payload_builder_builder: PB) -> Self {
        Self { payload_builder_builder }
    }
}

impl<Node, Pool, PB, EvmConfig> PayloadServiceBuilder<Node, Pool, EvmConfig>
    for OpBasicPayloadServiceBuilder<PB>
where
    Node: FullNodeTypes<Types: OpNodeTypes>,
    Pool: TransactionPool,
    EvmConfig: Send,
    PB: PayloadBuilderBuilder<Node, Pool, EvmConfig>,
{
    /// UPSTREAM-MIRROR(copy): reth@rev:0fbe428
    /// `reth_node_builder::components::BasicPayloadServiceBuilder::spawn_payload_builder_service`
    ///
    /// Wraps the basic job generator in an [`OpPayloadJobGenerator`], which builds derived payloads
    /// that carry a `0x7D` itself. Upstream's `pre_cache_state` option is left out, so state is
    /// always pre-cached, as upstream does by default; the rest is upstream's body.
    async fn spawn_payload_builder_service(
        self,
        ctx: &BuilderContext<Node>,
        pool: Pool,
        evm_config: EvmConfig,
    ) -> eyre::Result<PayloadBuilderHandle<<Node::Types as NodeTypes>::Payload>> {
        let payload_builder =
            self.payload_builder_builder.build_payload_builder(ctx, pool, evm_config).await?;

        let conf = &ctx.config().builder;
        let job_config = BasicPayloadJobGeneratorConfig::default()
            .interval(conf.interval)
            .deadline(conf.deadline)
            .max_payload_tasks(conf.max_payload_tasks);
        let generator = OpPayloadJobGenerator::new(
            BasicPayloadJobGenerator::with_builder(
                ctx.provider().clone(),
                ctx.task_executor().clone(),
                job_config,
                payload_builder.clone(),
            ),
            ctx.provider().clone(),
            payload_builder,
            conf.deadline,
        );
        let (payload_service, payload_service_handle) =
            PayloadBuilderService::<_, _, <Node::Types as NodeTypes>::Payload>::new(
                generator,
                ctx.provider().canonical_state_stream(),
            );

        ctx.task_executor().spawn_critical_os_thread(
            "payload-service",
            "payload builder service",
            payload_service,
        );

        Ok(payload_service_handle)
    }
}

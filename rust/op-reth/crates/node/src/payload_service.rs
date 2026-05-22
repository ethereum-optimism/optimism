//! Op-reth payload service builder.
//!
//! Mirrors upstream
//! [`reth_node_builder::components::BasicPayloadServiceBuilder`] but spawns
//! the [`PayloadBuilderService`] via [`spawn_critical_task`] (a managed Tokio
//! task) rather than [`spawn_critical_os_thread`] (a raw OS thread parked in
//! `handle.block_on`).
//!
//! Upstream paradigmxyz/reth#24038 switched the payload builder to a named
//! OS thread, but `spawn_critical_os_thread` returns a `std::thread::JoinHandle`
//! that the upstream builder discards, and `reth_tasks::RuntimeInner` does not
//! track those handles or join them on drop. The OS thread can therefore
//! outlive its owning Tokio runtime and touch Tokio internals during teardown,
//! which manifests as a SIGSEGV in `peers_negotiate_eth_69` (issue #20973).
//!
//! Until upstream manages OS-thread lifecycles, op-reth keeps the payload
//! service on the Tokio runtime via this builder.
//!
//! [`spawn_critical_task`]: reth_tasks::TaskExecutor::spawn_critical_task
//! [`spawn_critical_os_thread`]: reth_tasks::TaskExecutor::spawn_critical_os_thread

use reth_basic_payload_builder::{BasicPayloadJobGenerator, BasicPayloadJobGeneratorConfig};
use reth_node_builder::{
    BuilderContext,
    components::{PayloadBuilderBuilder, PayloadServiceBuilder},
    node::{FullNodeTypes, NodeTypes},
};
use reth_payload_builder::{PayloadBuilderHandle, PayloadBuilderService};
use reth_provider::CanonStateSubscriptions;
use reth_transaction_pool::TransactionPool;

/// Op-reth replacement for
/// [`reth_node_builder::components::BasicPayloadServiceBuilder`] that spawns
/// the payload service as a Tokio task.
#[derive(Debug, Default, Clone)]
pub struct OpPayloadServiceBuilder<PB>(PB);

impl<PB> OpPayloadServiceBuilder<PB> {
    /// Wrap `payload_builder_builder` in a new [`OpPayloadServiceBuilder`].
    pub const fn new(payload_builder_builder: PB) -> Self {
        Self(payload_builder_builder)
    }
}

impl<Node, Pool, PB, EvmConfig> PayloadServiceBuilder<Node, Pool, EvmConfig>
    for OpPayloadServiceBuilder<PB>
where
    Node: FullNodeTypes,
    Pool: TransactionPool,
    EvmConfig: Send,
    PB: PayloadBuilderBuilder<Node, Pool, EvmConfig>,
{
    async fn spawn_payload_builder_service(
        self,
        ctx: &BuilderContext<Node>,
        pool: Pool,
        evm_config: EvmConfig,
    ) -> eyre::Result<PayloadBuilderHandle<<Node::Types as NodeTypes>::Payload>> {
        let payload_builder = self.0.build_payload_builder(ctx, pool, evm_config).await?;

        let conf = ctx.config().builder.clone();

        let payload_job_config = BasicPayloadJobGeneratorConfig::default()
            .interval(conf.interval)
            .deadline(conf.deadline)
            .max_payload_tasks(conf.max_payload_tasks);

        let payload_generator = BasicPayloadJobGenerator::with_builder(
            ctx.provider().clone(),
            ctx.task_executor().clone(),
            payload_job_config,
            payload_builder,
        );
        let (payload_service, payload_service_handle) =
            PayloadBuilderService::<_, _, <Node::Types as NodeTypes>::Payload>::new(
                payload_generator,
                ctx.provider().canonical_state_stream(),
            );

        ctx.task_executor().spawn_critical_task("payload builder service", payload_service);

        Ok(payload_service_handle)
    }
}

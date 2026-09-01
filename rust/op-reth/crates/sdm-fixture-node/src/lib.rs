//! Test-only op-reth fixture producer for public SDM acceptance tests.
//!
//! This crate is not published or included in production images. It replaces only the stock
//! payload service's EVM configuration and assigns one gas of refund to every committed normal
//! transaction. It is deliberately not an implementation of a production SDM policy.

use std::{borrow::Cow, sync::Arc};

use alloy_op_evm::{
    OpEvmFactory, OpTx,
    post_exec::{
        PostExecEvmFactoryAdapter, PostExecExecutedTx, PostExecRefundInspector, PostExecTxContext,
        PostExecTxKind,
    },
};
use alloy_primitives::{Address, U256};
use clap::Parser;
use futures_util::FutureExt;
use op_alloy_consensus::OpTxEnvelope;
use reth_db::DatabaseEnv;
use reth_db_api::database_metrics::DatabaseMetrics;
use reth_node_api::{FullNodeComponents, NodeTypes};
use reth_node_builder::{
    BuilderContext, FullNodeTypes, Node, NodeAdapter, NodeBuilder, NodeComponentsBuilder,
    WithLaunchContext,
    components::{
        BasicPayloadServiceBuilder, ComponentsBuilder, PayloadBuilderBuilder, PayloadServiceBuilder,
    },
    rpc::BasicEngineValidatorBuilder,
};
use reth_node_core::version::{RethCliVersionConsts, try_init_version_metadata};
use reth_optimism_chainspec::{OpChainSpec, project_genesis_from};
use reth_optimism_cli::{Cli, chainspec::OpChainSpecParser};
use reth_optimism_evm::{ConfigurePostExecEvm, OpEvmConfig, OpRethReceiptBuilder};
use reth_optimism_exex::OpProofsExEx;
use reth_optimism_node::{
    OpAddOns, OpConsensusBuilder, OpEngineApiBuilder, OpEngineValidatorBuilder, OpExecutorBuilder,
    OpNetworkBuilder, OpNode, OpNodeTypes, OpPoolBuilder,
    args::{ProofsStorageVersion, RollupArgs},
    node::{OpFullNodeTypes, OpPayloadBuilder},
    proof_history::spawn_proofs_db_metrics,
    rpc::OpEthApiBuilder,
};
use reth_optimism_payload_builder::OpPayloadBuilderAttributes;
use reth_optimism_primitives::OpPrimitives;
use reth_optimism_rpc::{
    debug::{DebugApiExt, DebugApiOverrideServer},
    eth::proofs::{EthApiExt, EthApiOverrideServer},
};
use reth_optimism_trie::{
    OpProofsStorage, OpProofsStore,
    db::{MdbxProofsStorage, MdbxProofsStorageV2},
};
use reth_payload_builder::PayloadBuilderHandle;
use reth_payload_primitives::BuildNextEnv;
use reth_transaction_pool::TransactionPool;
use revm::{
    context_interface::ContextTr,
    inspector::JournalExt,
    interpreter::{CallInputs, CallOutcome, CreateInputs, CreateOutcome, Interpreter},
};
use tracing::{info, warn};

/// A deterministic fixture policy that refunds one gas per committed normal transaction.
///
/// Deposits and the synthetic post-exec transaction receive no refund. The policy is stateless,
/// observes no opcode or account activity, and exposes no configurable refund amount.
#[derive(Debug, Clone, Copy, Default)]
pub struct FixedRefundPolicy {
    current_kind: Option<PostExecTxKind>,
}

impl PostExecRefundInspector for FixedRefundPolicy {
    type Snapshot = ();

    fn begin_tx(&mut self, ctx: PostExecTxContext) {
        self.current_kind = Some(ctx.kind);
    }

    fn note_account_touch(&mut self, _address: Address) {}

    fn finish_tx(&mut self) -> PostExecExecutedTx {
        let refund_total = u64::from(self.current_kind.take() == Some(PostExecTxKind::Normal));
        PostExecExecutedTx { refund_total, refund_events: Vec::new() }
    }

    fn inspect_step<CTX>(&mut self, _interp: &mut Interpreter, _context: &mut CTX)
    where
        CTX: ContextTr<Journal: JournalExt>,
    {
    }

    fn inspect_call<CTX>(&mut self, _context: &mut CTX, _inputs: &mut CallInputs)
    where
        CTX: ContextTr<Journal: JournalExt>,
    {
    }

    fn inspect_call_end<CTX>(
        &mut self,
        _context: &mut CTX,
        _inputs: &CallInputs,
        _outcome: &CallOutcome,
    ) where
        CTX: ContextTr<Journal: JournalExt>,
    {
    }

    fn inspect_create<CTX>(&mut self, _context: &mut CTX, _inputs: &mut CreateInputs)
    where
        CTX: ContextTr<Journal: JournalExt>,
    {
    }

    fn inspect_create_end<CTX>(
        &mut self,
        _context: &mut CTX,
        _inputs: &CreateInputs,
        _outcome: &CreateOutcome,
    ) where
        CTX: ContextTr<Journal: JournalExt>,
    {
    }

    fn inspect_selfdestruct(&mut self, _contract: Address, _target: Address, _value: U256) {}

    fn snapshot(&self) -> Self::Snapshot {}

    fn restore(&mut self, _snapshot: Self::Snapshot) {}
}

/// EVM configuration used only by the SDM fixture payload service.
pub type FixtureOpEvmConfig<ChainSpec, N> = OpEvmConfig<
    ChainSpec,
    N,
    OpRethReceiptBuilder,
    PostExecEvmFactoryAdapter<OpEvmFactory<OpTx, FixedRefundPolicy>>,
>;

fn fixture_evm_config<ChainSpec, N>(chain_spec: Arc<ChainSpec>) -> FixtureOpEvmConfig<ChainSpec, N>
where
    N: reth_node_api::NodePrimitives,
{
    OpEvmConfig::new_with_evm_factory(
        chain_spec,
        OpRethReceiptBuilder::default(),
        PostExecEvmFactoryAdapter::new(OpEvmFactory::default()),
    )
}

/// Runs a configured [`OpPayloadBuilder`] with the fixed fixture policy.
#[derive(Debug, Clone)]
pub struct FixturePayloadServiceBuilder {
    inner: OpPayloadBuilder,
}

impl FixturePayloadServiceBuilder {
    /// Creates a fixture payload service builder.
    pub const fn new(inner: OpPayloadBuilder) -> Self {
        Self { inner }
    }
}

impl<Node, Pool, EvmConfig> PayloadServiceBuilder<Node, Pool, EvmConfig>
    for FixturePayloadServiceBuilder
where
    Node: FullNodeTypes<Types: OpNodeTypes>,
    Pool: TransactionPool + Clone + Send + Sync + Unpin + 'static,
    EvmConfig: Send,
    FixtureOpEvmConfig<<Node::Types as NodeTypes>::ChainSpec, OpPrimitives>: ConfigurePostExecEvm<
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
    OpPayloadBuilder: PayloadBuilderBuilder<
            Node,
            Pool,
            FixtureOpEvmConfig<<Node::Types as NodeTypes>::ChainSpec, OpPrimitives>,
        >,
{
    async fn spawn_payload_builder_service(
        self,
        ctx: &BuilderContext<Node>,
        pool: Pool,
        _evm_config: EvmConfig,
    ) -> eyre::Result<PayloadBuilderHandle<<Node::Types as NodeTypes>::Payload>> {
        let evm_config = fixture_evm_config::<<Node::Types as NodeTypes>::ChainSpec, OpPrimitives>(
            ctx.chain_spec(),
        );

        BasicPayloadServiceBuilder::new(self.inner)
            .spawn_payload_builder_service(ctx, pool, evm_config)
            .await
    }
}

/// An [`OpNode`] wrapper that replaces only the payload service and EVM configuration.
#[derive(Debug, Clone)]
pub struct FixtureOpNode {
    inner: OpNode,
}

impl FixtureOpNode {
    /// Creates a test-only fixture node.
    pub fn new(rollup_args: RollupArgs) -> Self {
        Self { inner: OpNode::new(rollup_args) }
    }
}

impl reth_node_builder::node::NodeTypes for FixtureOpNode {
    type Primitives = <OpNode as reth_node_builder::node::NodeTypes>::Primitives;
    type ChainSpec = <OpNode as reth_node_builder::node::NodeTypes>::ChainSpec;
    type Storage = <OpNode as reth_node_builder::node::NodeTypes>::Storage;
    type Payload = <OpNode as reth_node_builder::node::NodeTypes>::Payload;
}

impl<N> Node<N> for FixtureOpNode
where
    N: FullNodeTypes<Types: OpFullNodeTypes + OpNodeTypes>,
{
    type ComponentsBuilder = ComponentsBuilder<
        N,
        OpPoolBuilder,
        FixturePayloadServiceBuilder,
        OpNetworkBuilder,
        OpExecutorBuilder,
        OpConsensusBuilder,
    >;

    type AddOns = OpAddOns<
        NodeAdapter<N, <Self::ComponentsBuilder as NodeComponentsBuilder<N>>::Components>,
        OpEthApiBuilder,
        OpEngineValidatorBuilder,
        OpEngineApiBuilder<OpEngineValidatorBuilder>,
        BasicEngineValidatorBuilder<OpEngineValidatorBuilder>,
    >;

    fn components_builder(&self) -> Self::ComponentsBuilder {
        OpNode::components::<N>(&self.inner)
            .payload(FixturePayloadServiceBuilder::new(self.inner.payload_builder()))
    }

    fn add_ons(&self) -> Self::AddOns {
        self.inner.add_ons()
    }
}

// Mirror of `reth_optimism_node::proof_history::launch_node` for `FixtureOpNode`.
//
// Keep in sync with that function. It cannot be shared generically: reth parameterizes add-ons and
// the whole RPC stack by a node's component set, so a launcher generic over the node type has to
// restate reth's entire `EthApi`/`RpcNodeCore` bound chain — which is a worse artifact than this
// duplication. The pieces that *can* be shared are shared: `spawn_proofs_db_metrics` and
// `OpNode::payload_builder`.
//
// One deliberate difference: stock launches with debug capabilities, this launches plain, because
// `FixtureOpNode` does not implement reth's `DebugNode` and acceptance tests never use reth's debug
// launch features.
async fn launch_fixture_node(
    mut builder: WithLaunchContext<NodeBuilder<DatabaseEnv, OpChainSpec>>,
    args: RollupArgs,
) -> eyre::Result<()> {
    if args.private {
        let source = builder.config().chain.genesis.clone();
        let projected = project_genesis_from(&source)
            .map_err(|err| eyre::eyre!("failed to project private-chain genesis: {err}"))?;
        builder.config_mut().chain = Arc::new(OpChainSpec::from_genesis(projected));
        info!(target: "reth::cli", "Using deterministic public-projection genesis");
    }

    if !args.proofs_history {
        let handle = builder.node(FixtureOpNode::new(args)).launch().await?;
        return handle.node_exit_future.await;
    }

    let path = args.history.resolve_storage_path(builder.config().datadir().as_ref());
    match args.history.storage_version {
        ProofsStorageVersion::V1 => {
            info!(target: "reth::cli", "Using on-disk storage for proofs history (v1)");
            let storage = Arc::new(
                MdbxProofsStorage::new(&path)
                    .map_err(|err| eyre::eyre!("Failed to create MdbxProofsStorage: {err}"))?,
            );
            launch_fixture_with_proof_history(builder, args, storage).await
        }
        ProofsStorageVersion::V2 => {
            info!(target: "reth::cli", "Using on-disk storage for proofs history (v2)");
            let storage = Arc::new(
                MdbxProofsStorageV2::new(&path)
                    .map_err(|err| eyre::eyre!("Failed to create MdbxProofsStorageV2: {err}"))?,
            );
            launch_fixture_with_proof_history(builder, args, storage).await
        }
    }
}

async fn launch_fixture_with_proof_history<S>(
    builder: WithLaunchContext<NodeBuilder<DatabaseEnv, OpChainSpec>>,
    args: RollupArgs,
    mdbx: Arc<S>,
) -> eyre::Result<()>
where
    S: OpProofsStore + DatabaseMetrics + Send + Sync + 'static,
{
    let storage: OpProofsStorage<Arc<S>> = mdbx.clone().into();
    let storage_exec = storage.clone();
    let RollupArgs { proofs_history_window, proofs_history_verification_interval, .. } =
        args.clone();

    let handle = builder
        .node(FixtureOpNode::new(args))
        .on_node_started(move |node| {
            spawn_proofs_db_metrics(
                node.task_executor,
                mdbx,
                node.config.metrics.push_gateway_interval,
            );
            Ok(())
        })
        .install_exex("proofs-history", async move |exex_context| {
            Ok(OpProofsExEx::builder(exex_context, storage_exec)
                .with_proofs_history_window(proofs_history_window.window)
                .with_verification_interval(proofs_history_verification_interval)
                .build()
                .run()
                .boxed())
        })
        .extend_rpc_modules(move |ctx| {
            info!(target: "reth::cli", "Installing proofs-history RPC overrides (eth_getProof, debug_executePayload)");
            let api_ext = EthApiExt::new(ctx.registry.eth_api().clone(), storage.clone());
            let auth_api_ext = EthApiExt::new(ctx.registry.eth_api().clone(), storage.clone());
            let debug_ext = DebugApiExt::new(
                ctx.node().provider().clone(),
                ctx.registry.eth_api().clone(),
                storage,
                ctx.node().task_executor().clone(),
                ctx.node().evm_config().clone(),
            );
            let eth_replaced = ctx.modules.replace_configured(api_ext.into_rpc())?;
            let auth_eth_replaced = ctx
                .auth_module
                .replace_auth_methods(auth_api_ext.into_rpc())?;
            let debug_replaced = ctx.modules.replace_configured(debug_ext.into_rpc())?;
            info!(target: "reth::cli", eth_replaced, auth_eth_replaced, debug_replaced, "Proofs-history RPC overrides installed");
            Ok(())
        })
        .launch()
        .await?;

    handle.node_exit_future.await
}

/// Launches the test-only SDM fixture binary.
pub fn run() -> ! {
    const CLIENT_NAME: &str = "op-reth-sdm-fixture";
    let build = op_version::build_info!();
    let version_metadata = try_init_version_metadata(RethCliVersionConsts {
        name_client: Cow::Borrowed(CLIENT_NAME),
        cargo_pkg_version: Cow::Owned(build.version().to_string()),
        vergen_git_sha_long: Cow::Owned(build.commit_sha().to_string()),
        vergen_git_sha: Cow::Owned(build.short_sha().to_string()),
        vergen_build_timestamp: Cow::Owned(build.build_timestamp().to_string()),
        vergen_cargo_target_triple: Cow::Owned(build.target_triple().to_string()),
        vergen_cargo_features: Cow::Owned(build.cargo_features().to_string()),
        short_version: Cow::Owned(build.short_version()),
        long_version: Cow::Owned(build.long_version()),
        build_profile_name: Cow::Owned(build.build_profile().to_string()),
        p2p_client_version: Cow::Owned(format!(
            "{CLIENT_NAME}/v{}-{}/{}",
            build.version(),
            build.short_sha(),
            build.target_triple()
        )),
        extra_data: Cow::Owned(String::new()),
    });
    if version_metadata.is_err() {
        eprintln!("Error: build info is already embedded. This is a bug.")
    }

    let result = Cli::<OpChainSpecParser, RollupArgs>::parse().run(async move |builder, args| {
        warn!(
            target: "op-reth-sdm-fixture::cli",
            "TEST-ONLY FIXTURE POLICY: fixed one-gas refunds; never use this binary in production"
        );
        info!(target: "op-reth-sdm-fixture::cli", "Launching SDM fixture node");
        launch_fixture_node(builder, args).await
    });

    match result {
        Ok(()) => std::process::exit(0),
        Err(err) => {
            eprintln!("Error: {err:?}");
            std::process::exit(1);
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fixed_policy_refunds_only_normal_transactions() {
        let mut policy = FixedRefundPolicy::default();
        for (kind, expected) in [
            (PostExecTxKind::Normal, 1),
            (PostExecTxKind::Deposit, 0),
            (PostExecTxKind::PostExec, 0),
        ] {
            policy.begin_tx(PostExecTxContext { tx_index: 3, kind });
            policy.note_account_touch(Address::ZERO);
            assert_eq!(policy.finish_tx().refund_total, expected);
        }
    }

    #[test]
    fn fixture_policy_has_no_refund_configuration() {
        assert_eq!(std::mem::size_of::<FixedRefundPolicy>(), 1);
    }

    #[test]
    fn fixture_factory_uses_unit_snapshots() {
        fn assert_unit_snapshot<
            F: alloy_op_evm::post_exec::PostExecEvmFactoryHooks<Snapshot = ()>,
        >() {
        }
        assert_unit_snapshot::<OpEvmFactory<OpTx, FixedRefundPolicy>>();
        let _ = OpEvmFactory::<OpTx, FixedRefundPolicy>::default();
    }
}

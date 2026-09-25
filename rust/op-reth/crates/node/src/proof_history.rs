//! Node launcher with proof history support.

use crate::{
    OpNode,
    args::{ProofsStorageVersion, RollupArgs},
};
use alloy_primitives::Address;
use eyre::ErrReport;
use futures_util::FutureExt;
use reth_db::DatabaseEnv;
use reth_db_api::database_metrics::DatabaseMetrics;
use reth_node_builder::{
    FullNodeComponents, Node, NodeBuilder, NodeBuilderWithComponents, RethFullAdapter,
    WithLaunchContext,
};
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_exex::OpProofsExEx;
use reth_optimism_rpc::{
    debug::{DebugApiExt, DebugApiOverrideServer},
    eth::proofs::{EthApiExt, EthApiOverrideServer},
};
use reth_optimism_trie::{
    OpProofsStorage, OpProofsStore,
    db::{MdbxProofsStorage, MdbxProofsStorageV2},
};
use reth_tasks::TaskExecutor;
use std::{sync::Arc, time::Duration};
use tokio::time::sleep;
use tracing::info;

type ConfiguredOpNodeBuilder = WithLaunchContext<
    NodeBuilderWithComponents<
        RethFullAdapter<DatabaseEnv, OpNode>,
        <OpNode as Node<RethFullAdapter<DatabaseEnv, OpNode>>>::ComponentsBuilder,
        <OpNode as Node<RethFullAdapter<DatabaseEnv, OpNode>>>::AddOns,
    >,
>;

/// Declarative configuration for the shared OP node launcher.
#[derive(Debug)]
pub struct OpNodeLaunchConfig {
    node: OpNode,
}

impl OpNodeLaunchConfig {
    /// Configures a production op-reth node.
    pub fn production(args: RollupArgs) -> Self {
        Self { node: OpNode::new(args) }
    }

    /// Selects the deterministic fixed-refund policy used by SDM acceptance tests.
    #[doc(hidden)]
    #[must_use]
    pub fn with_test_sdm_fixed_refund(mut self, excessive_refund_target: Option<Address>) -> Self {
        self.node = self.node.with_test_sdm_fixed_refund(excessive_refund_target);
        self
    }
}

/// Launches an OP node, optionally installing proof history, then waits for it to exit.
pub async fn launch_node(
    builder: WithLaunchContext<NodeBuilder<DatabaseEnv, OpChainSpec>>,
    config: OpNodeLaunchConfig,
) -> eyre::Result<(), ErrReport> {
    let OpNodeLaunchConfig { node } = config;
    let args = &node.args;
    let proof_history = args.proofs_history.then(|| {
        (
            // Defaults to `<reth-data-dir>/historical-proofs` when not supplied — see
            // [`ProofsHistoryStorageArgs::resolve_storage_path`].
            args.history.resolve_storage_path(builder.config().datadir().as_ref()),
            args.history.storage_version,
            args.proofs_history_window.window,
            args.proofs_history_verification_interval,
        )
    });

    let builder = builder.node(node);
    let builder = match proof_history {
        None => builder,
        Some((path, ProofsStorageVersion::V1, window, verification_interval)) => {
            info!(target: "reth::cli", "Using on-disk storage for proofs history (v1)");
            let mdbx = Arc::new(
                MdbxProofsStorage::new(&path)
                    .map_err(|e| eyre::eyre!("Failed to create MdbxProofsStorage: {e}"))?,
            );
            configure_proof_history(builder, mdbx, window, verification_interval)
        }
        Some((path, ProofsStorageVersion::V2, window, verification_interval)) => {
            info!(target: "reth::cli", "Using on-disk storage for proofs history (v2)");
            let mdbx = Arc::new(
                MdbxProofsStorageV2::new(&path)
                    .map_err(|e| eyre::eyre!("Failed to create MdbxProofsStorageV2: {e}"))?,
            );
            configure_proof_history(builder, mdbx, window, verification_interval)
        }
    };

    builder.launch_with_debug_capabilities().await?.node_exit_future.await
}

/// Installs the ExEx, RPC overrides, and metrics hook for proof history.
fn configure_proof_history<S>(
    builder: ConfiguredOpNodeBuilder,
    mdbx: Arc<S>,
    proofs_history_window: u64,
    proofs_history_verification_interval: u64,
) -> ConfiguredOpNodeBuilder
where
    S: OpProofsStore + DatabaseMetrics + Send + Sync + 'static,
{
    let storage: OpProofsStorage<Arc<S>> = mdbx.clone().into();
    let storage_exec = storage.clone();

    builder
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
                .with_proofs_history_window(proofs_history_window)
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
            let auth_eth_replaced = ctx.auth_module.replace_auth_methods(auth_api_ext.into_rpc())?;
            let debug_replaced = ctx.modules.replace_configured(debug_ext.into_rpc())?;
            info!(target: "reth::cli", eth_replaced, auth_eth_replaced, debug_replaced, "Proofs-history RPC overrides installed");
            Ok(())
        })
}

/// Spawns a task that periodically reports metrics for the proofs DB.
fn spawn_proofs_db_metrics<S>(
    executor: TaskExecutor,
    storage: Arc<S>,
    metrics_report_interval: Duration,
) where
    S: DatabaseMetrics + Send + Sync + 'static,
{
    executor.spawn_critical_task("op-proofs-storage-metrics", async move {
        info!(
            target: "reth::cli",
            ?metrics_report_interval,
            "Starting op-proofs-storage metrics task"
        );

        loop {
            sleep(metrics_report_interval).await;
            storage.report_metrics();
        }
    });
}

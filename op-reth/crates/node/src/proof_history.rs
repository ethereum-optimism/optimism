//! Node luncher with proof history support.

use crate::{args::RollupArgs, OpNode};
use eyre::ErrReport;
use futures_util::FutureExt;
use reth_db::DatabaseEnv;
use reth_db_api::database_metrics::DatabaseMetrics;
use reth_node_builder::{FullNodeComponents, NodeBuilder, WithLaunchContext};
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_exex::OpProofsExEx;
use reth_optimism_rpc::{
    debug::{DebugApiExt, DebugApiOverrideServer},
    eth::proofs::{EthApiExt, EthApiOverrideServer},
};
use reth_optimism_trie::{db::MdbxProofsStorage, OpProofsStorage};
use reth_tasks::TaskExecutor;
use std::{sync::Arc, time::Duration};
use tokio::time::sleep;
use tracing::info;

/// - no proofs history (plain node),
/// - in-mem proofs storage,
/// - MDBX proofs storage.
pub async fn launch_node_with_proof_history(
    builder: WithLaunchContext<NodeBuilder<Arc<DatabaseEnv>, OpChainSpec>>,
    args: RollupArgs,
) -> eyre::Result<(), ErrReport> {
    let RollupArgs {
        proofs_history,
        proofs_history_window,
        proofs_history_prune_interval,
        proofs_history_verification_interval,
        ..
    } = args;

    // Start from a plain OpNode builder
    let mut node_builder = builder.node(OpNode::new(args.clone()));

    if proofs_history {
        let path = args
            .proofs_history_storage_path
            .clone()
            .expect("Path must be provided if not using in-memory storage");
        info!(target: "reth::cli", "Using on-disk storage for proofs history");

        let mdbx = Arc::new(
            MdbxProofsStorage::new(&path)
                .map_err(|e| eyre::eyre!("Failed to create MdbxProofsStorage: {e}"))?,
        );
        let storage: OpProofsStorage<Arc<MdbxProofsStorage>> = mdbx.clone().into();

        let storage_exec = storage.clone();

        node_builder = node_builder
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
                    .with_proofs_history_prune_interval(proofs_history_prune_interval)
                    .with_verification_interval(proofs_history_verification_interval)
                    .build()
                    .run()
                    .boxed())
            })
            .extend_rpc_modules(move |ctx| {
                let api_ext = EthApiExt::new(ctx.registry.eth_api().clone(), storage.clone());
                let debug_ext = DebugApiExt::new(
                    ctx.node().provider().clone(),
                    ctx.registry.eth_api().clone(),
                    storage,
                    Box::new(ctx.node().task_executor().clone()),
                    ctx.node().evm_config().clone(),
                );
                ctx.modules.replace_configured(api_ext.into_rpc())?;
                ctx.modules.replace_configured(debug_ext.into_rpc())?;
                Ok(())
            });
    }

    // In all cases (with or without proofs), launch the node.
    let handle = node_builder.launch_with_debug_capabilities().await?;
    handle.node_exit_future.await
}
/// Spawns a task that periodically reports metrics for the proofs DB.
fn spawn_proofs_db_metrics(
    executor: TaskExecutor,
    storage: Arc<MdbxProofsStorage>,
    metrics_report_interval: Duration,
) {
    executor.spawn_critical("op-proofs-storage-metrics", async move {
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

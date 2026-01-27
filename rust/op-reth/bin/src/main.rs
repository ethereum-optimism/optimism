#![allow(missing_docs, rustdoc::missing_crate_level_docs)]

use clap::Parser;
use eyre::ErrReport;
use futures_util::FutureExt;
use reth_db::DatabaseEnv;
use reth_db_api::database_metrics::DatabaseMetrics;
use reth_node_builder::{FullNodeComponents, NodeBuilder, WithLaunchContext};
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_cli::{chainspec::OpChainSpecParser, Cli};
use reth_optimism_exex::OpProofsExEx;
use reth_optimism_node::{args::RollupArgs, OpNode};
use reth_optimism_rpc::{
    debug::{DebugApiExt, DebugApiOverrideServer},
    eth::proofs::{EthApiExt, EthApiOverrideServer},
};
use reth_optimism_trie::{db::MdbxProofsStorage, OpProofsStorage};
use reth_tasks::TaskExecutor;
use std::{sync::Arc, time::Duration};
use tokio::time::sleep;
use tracing::info;

#[global_allocator]
static ALLOC: reth_cli_util::allocator::Allocator = reth_cli_util::allocator::new_allocator();

#[cfg(all(feature = "jemalloc-prof", unix))]
#[unsafe(export_name = "_rjem_malloc_conf")]
static MALLOC_CONF: &[u8] = b"prof:true,prof_active:true,lg_prof_sample:19\0";

/// Single entry that handles:
/// - no proofs history (plain node),
/// - in-mem proofs storage,
/// - MDBX proofs storage.
async fn launch_node(
    builder: WithLaunchContext<NodeBuilder<Arc<DatabaseEnv>, OpChainSpec>>,
    args: RollupArgs,
) -> eyre::Result<(), ErrReport> {
    let proofs_history_enabled = args.proofs_history;
    let proofs_history_window = args.proofs_history_window;
    let proofs_history_prune_interval = args.proofs_history_prune_interval;
    let proofs_history_verification_interval = args.proofs_history_verification_interval;

    // Start from a plain OpNode builder
    let mut node_builder = builder.node(OpNode::new(args.clone()));

    if proofs_history_enabled {
        let path = args
            .proofs_history_storage_path
            .clone()
            .expect("Path must be provided if not using in-memory storage");
        info!(target: "reth::cli", "Using on-disk storage for proofs history");

        let mdbx = Arc::new(
            MdbxProofsStorage::new(&path)
                .map_err(|e| eyre::eyre!("Failed to create MdbxProofsStorage: {e}"))?,
        );
        let storage: OpProofsStorage<_> = mdbx.clone().into();

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

fn main() {
    reth_cli_util::sigsegv_handler::install();

    // Enable backtraces unless a RUST_BACKTRACE value has already been explicitly provided.
    if std::env::var_os("RUST_BACKTRACE").is_none() {
        unsafe {
            std::env::set_var("RUST_BACKTRACE", "1");
        }
    }

    if let Err(err) =
        Cli::<OpChainSpecParser, RollupArgs>::parse().run(async move |builder, args| {
            info!(target: "reth::cli", "Launching node");
            launch_node(builder, args.clone()).await?;
            Ok(())
        })
    {
        eprintln!("Error: {err:?}");
        std::process::exit(1);
    }
}

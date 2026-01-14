#![allow(missing_docs, rustdoc::missing_crate_level_docs)]

use clap::{builder::ArgPredicate, Parser};
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
use std::{path::PathBuf, sync::Arc, time::Duration};
use tokio::time::sleep;
use tracing::info;

#[global_allocator]
static ALLOC: reth_cli_util::allocator::Allocator = reth_cli_util::allocator::new_allocator();

#[cfg(all(feature = "jemalloc-prof", unix))]
#[unsafe(export_name = "_rjem_malloc_conf")]
static MALLOC_CONF: &[u8] = b"prof:true,prof_active:true,lg_prof_sample:19\0";

#[derive(Debug, Clone, PartialEq, Eq, clap::Args)]
#[command(next_help_heading = "Proofs History")]
struct Args {
    #[command(flatten)]
    pub rollup_args: RollupArgs,

    /// If true, initialize external-proofs exex to save and serve trie nodes to provide proofs
    /// faster.
    #[arg(
        long = "proofs-history",
        value_name = "PROOFS_HISTORY",
        default_value_ifs([
            ("proofs-history.storage-path", ArgPredicate::IsPresent, "true")
        ])
    )]
    pub proofs_history: bool,

    /// The path to the storage DB for proofs history.
    #[arg(long = "proofs-history.storage-path", value_name = "PROOFS_HISTORY_STORAGE_PATH")]
    pub proofs_history_storage_path: Option<PathBuf>,

    /// The window to span blocks for proofs history. Value is the number of blocks.
    /// Default is 1 month of blocks based on 2 seconds block time.
    /// 30 * 24 * 60 * 60 / 2 = `1_296_000`
    #[arg(
        long = "proofs-history.window",
        default_value_t = 1_296_000,
        value_name = "PROOFS_HISTORY_WINDOW"
    )]
    pub proofs_history_window: u64,

    /// Interval between proof-storage prune runs. Accepts human-friendly durations
    /// like "100s", "5m", "1h". Defaults to 15s.
    ///
    /// - Shorter intervals prune smaller batches more often, so each prune run tends to be faster
    ///   and the blocking pause for writes is shorter, at the cost of more frequent pauses.
    /// - Longer intervals prune larger batches less often, which reduces how often pruning runs,
    ///   but each run can take longer and block writes for longer.
    ///
    /// A shorter interval is preferred so that prune
    /// runs stay small and don’t stall writes for too long.
    ///
    /// CLI: `--proofs-history.prune-interval 10m`
    #[arg(
        long = "proofs-history.prune-interval",
        value_name = "PROOFS_HISTORY_PRUNE_INTERVAL",
        default_value = "15s",
        value_parser = humantime::parse_duration
    )]
    pub proofs_history_prune_interval: Duration,
    /// Verification interval: perform full block execution every N blocks for data integrity.
    /// - 0: Disabled (Default) (always use fast path with pre-computed data from notifications)
    /// - 1: Always verify (always execute blocks, slowest)
    /// - N: Verify every Nth block (e.g., 100 = every 100 blocks)
    ///
    /// Periodic verification helps catch data corruption or consensus bugs while maintaining
    /// good performance.
    ///
    /// CLI: `--proofs-history.verification-interval 100`
    #[arg(
        long = "proofs-history.verification-interval",
        value_name = "PROOFS_HISTORY_VERIFICATION_INTERVAL",
        default_value_t = 0
    )]
    pub proofs_history_verification_interval: u64,
}

/// Single entry that handles:
/// - no proofs history (plain node),
/// - in-mem proofs storage,
/// - MDBX proofs storage.
async fn launch_node(
    builder: WithLaunchContext<NodeBuilder<Arc<DatabaseEnv>, OpChainSpec>>,
    args: Args,
) -> eyre::Result<(), ErrReport> {
    let proofs_history_enabled = args.proofs_history;
    let rollup_args = args.rollup_args.clone();
    let proofs_history_window = args.proofs_history_window;
    let proofs_history_prune_interval = args.proofs_history_prune_interval;
    let proofs_history_verification_interval = args.proofs_history_verification_interval;

    // Start from a plain OpNode builder
    let mut node_builder = builder.node(OpNode::new(rollup_args));

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
                Ok(OpProofsExEx::new(
                    exex_context,
                    storage_exec,
                    proofs_history_window,
                    proofs_history_prune_interval,
                    proofs_history_verification_interval,
                )
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

    if let Err(err) = Cli::<OpChainSpecParser, Args>::parse().run(async move |builder, args| {
        info!(target: "reth::cli", "Launching node");
        launch_node(builder, args.clone()).await?;
        Ok(())
    }) {
        eprintln!("Error: {err:?}");
        std::process::exit(1);
    }
}

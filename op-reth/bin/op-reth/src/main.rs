#![allow(missing_docs, rustdoc::missing_crate_level_docs)]

use clap::{builder::ArgPredicate, Parser};
use eyre::ErrReport;
use futures_util::FutureExt;
use reth_db::DatabaseEnv;
use reth_node_builder::{NodeBuilder, WithLaunchContext};
use reth_optimism_chainspec::OpChainSpec;
use reth_optimism_cli::{chainspec::OpChainSpecParser, Cli};
use reth_optimism_exex::OpProofsExEx;
use reth_optimism_node::{args::RollupArgs, OpNode};
use reth_optimism_rpc::eth::proofs::{EthApiExt, EthApiOverrideServer};
use reth_optimism_trie::{
    db::MdbxProofsStorage, InMemoryProofsStorage, OpProofsStorage, OpProofsStore, StorageMetrics,
};
use tracing::info;

use std::{path::PathBuf, sync::Arc};

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
            ("proofs-history.in_mem", ArgPredicate::IsPresent, "true"),
            ("proofs-history.storage-path", ArgPredicate::IsPresent, "true")
        ])
    )]
    pub proofs_history: bool,

    /// The storage DB for proofs history.
    #[arg(
        long = "proofs-history.in_mem",
        value_name = "PROOFS_HISTORY_STORAGE_IN_MEM",
        conflicts_with = "proofs-history.storage-path",
        default_value_if("proofs-history", "true", Some("false"))
    )]
    pub proofs_history_storage_in_mem: bool,

    /// The path to the storage DB for proofs history.
    #[arg(
        long = "proofs-history.storage-path",
        value_name = "PROOFS_HISTORY_STORAGE_PATH",
        required_if_eq("proofs-history.in_mem", "false")
    )]
    pub proofs_history_storage_path: Option<PathBuf>,

    /// The window to span blocks for proofs history. Value is the number of blocks.
    /// Default is 1 month of blocks based on 2 seconds block time.
    /// 30 * 24 * 60 * 60 / 2 = `1_296_000`
    // TODO: Pass this arg to the ExEx or remove it if not needed.
    #[arg(
        long = "proofs-history.window",
        default_value_t = 1_296_000,
        value_name = "PROOFS_HISTORY_WINDOW"
    )]
    pub proofs_history_window: u64,
}

async fn launch_node_with_storage<S>(
    builder: WithLaunchContext<NodeBuilder<Arc<DatabaseEnv>, OpChainSpec>>,
    args: Args,
    storage: OpProofsStorage<S>,
) -> eyre::Result<(), ErrReport>
where
    S: OpProofsStore + Clone + 'static,
{
    let storage_clone = storage.clone();
    let proofs_history_enabled = args.proofs_history;
    let handle = builder
        .node(OpNode::new(args.rollup_args))
        .install_exex_if(proofs_history_enabled, "proofs-history", async move |exex_context| {
            Ok(OpProofsExEx::new(exex_context, storage, args.proofs_history_window).run().boxed())
        })
        .extend_rpc_modules(move |ctx| {
            if proofs_history_enabled {
                let api_ext = EthApiExt::new(ctx.registry.eth_api().clone(), storage_clone);
                ctx.modules.replace_configured(api_ext.into_rpc())?;
            }
            Ok(())
        })
        .launch_with_debug_capabilities()
        .await?;
    handle.node_exit_future.await
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

        if args.proofs_history_storage_in_mem {
            // todo: enable launch without metrics
            let storage = OpProofsStorage::new(
                Arc::new(InMemoryProofsStorage::new()),
                Arc::new(StorageMetrics::default()),
            );

            launch_node_with_storage(builder, args.clone(), storage).await?;
        } else {
            let path = args
                .proofs_history_storage_path
                .clone()
                .expect("Path must be provided if not using in-memory storage");
            info!(target: "reth::cli", "Using on-disk storage for proofs history");

            // todo: enable launch without metrics
            let storage = OpProofsStorage::new(
                Arc::new(
                    MdbxProofsStorage::new(&path)
                        .map_err(|e| eyre::eyre!("Failed to create MdbxProofsStorage: {e}"))?,
                ),
                Arc::new(StorageMetrics::default()),
            );

            launch_node_with_storage(builder, args.clone(), storage).await?;
        }

        Ok(())
    }) {
        eprintln!("Error: {err:?}");
        std::process::exit(1);
    }
}

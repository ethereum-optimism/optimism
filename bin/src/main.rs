#![allow(missing_docs, rustdoc::missing_crate_level_docs)]

use clap::Parser;
use reth_optimism_cli::{chainspec::OpChainSpecParser, Cli};
use reth_optimism_node::{args::RollupArgs, OpNode};
use tracing::info;

#[global_allocator]
static ALLOC: reth_cli_util::allocator::Allocator = reth_cli_util::allocator::new_allocator();

fn main() {
    reth_cli_util::sigsegv_handler::install();

    // Enable backtraces unless a RUST_BACKTRACE value has already been explicitly provided.
    if std::env::var_os("RUST_BACKTRACE").is_none() {
        std::env::set_var("RUST_BACKTRACE", "1");
    }

    if let Err(err) =
        Cli::<OpChainSpecParser, RollupArgs>::parse().run(async move |builder, rollup_args| {
            info!(target: "reth::cli", "Launching node");
            let handle =
                builder.node(OpNode::new(rollup_args)).launch_with_debug_capabilities().await?;
            handle.node_exit_future.await
        })
    {
        eprintln!("Error: {err:?}");
        std::process::exit(1);
    }
}

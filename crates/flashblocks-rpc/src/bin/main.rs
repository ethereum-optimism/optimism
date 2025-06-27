#![allow(missing_docs, rustdoc::missing_crate_level_docs)]

use clap::Parser;
use flashblocks_rpc::{EthApiOverrideServer, FlashblocksApiExt, FlashblocksOverlay};
use reth_optimism_cli::{Cli, chainspec::OpChainSpecParser};
use reth_optimism_node::{OpNode, args::RollupArgs};
use tracing::info;

#[derive(Debug, Clone, PartialEq, Eq, clap::Args)]
#[command(next_help_heading = "Rollup")]
struct FlashblocksRollupArgs {
    #[command(flatten)]
    rollup_args: RollupArgs,

    #[arg(long = "flashblocks.enabled", default_value = "false")]
    flashblocks_enabled: bool,

    #[arg(long = "flashblocks.websocket-url", value_name = "WEBSOCKET_URL")]
    websocket_url: url::Url,
}

fn main() {
    if let Err(err) =
        Cli::<OpChainSpecParser, FlashblocksRollupArgs>::parse().run(async move |builder, args| {
            let rollup_args = args.rollup_args;
            let chain_spec = builder.config().chain.clone();

            info!(target: "reth::cli", "Launching Flashblocks RPC overlay node");
            let handle = builder
                .node(OpNode::new(rollup_args))
                .extend_rpc_modules(move |ctx| {
                    if args.flashblocks_enabled {
                        let mut flashblocks_overlay =
                            FlashblocksOverlay::new(args.websocket_url, chain_spec);
                        flashblocks_overlay.start()?;

                        let eth_api = ctx.registry.eth_api().clone();
                        let api_ext = FlashblocksApiExt::new(eth_api.clone(), flashblocks_overlay);

                        ctx.modules.replace_configured(api_ext.into_rpc())?;
                    }
                    Ok(())
                })
                .launch_with_debug_capabilities()
                .await?;
            handle.node_exit_future.await
        })
    {
        tracing::error!("Error: {err:?}");
        std::process::exit(1);
    }
}

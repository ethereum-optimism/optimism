//! OP-Reth CLI implementation.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(docsrs, feature(doc_cfg, doc_auto_cfg))]

/// A configurable App on top of the cli parser.
pub mod app;
/// Optimism chain specification parser.
pub mod chainspec;
/// Optimism CLI commands.
pub mod commands;
/// Module with a codec for reading and encoding receipts in files.
///
/// Enables decoding and encoding `OpGethReceipt` type. See <https://github.com/testinprod-io/op-geth/pull/1>.
///
/// Currently configured to use codec [`OpGethReceipt`](receipt_file_codec::OpGethReceipt) based on
/// export of below Bedrock data using <https://github.com/testinprod-io/op-geth/pull/1>. Codec can
/// be replaced with regular encoding of receipts for export.
///
/// NOTE: receipts can be exported using regular op-geth encoding for `Receipt` type, to fit
/// reth's needs for importing. However, this would require patching the diff in <https://github.com/testinprod-io/op-geth/pull/1> to export the `Receipt` and not `OpGethReceipt` type (originally
/// made for op-erigon's import needs).
pub mod receipt_file_codec;

/// OVM block, same as EVM block at bedrock, except for signature of deposit transaction
/// not having a signature back then.
/// Enables decoding and encoding `Block` types within file contexts.
pub mod ovm_file_codec;

pub use app::CliApp;
pub use commands::{import::ImportOpCommand, import_receipts::ImportReceiptsOpCommand};
use reth_optimism_chainspec::OpChainSpec;

use std::{ffi::OsString, fmt, sync::Arc};

use chainspec::OpChainSpecParser;
use clap::{command, Parser};
use commands::Commands;
use futures_util::Future;
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_commands::launcher::FnLauncher;
use reth_cli_runner::CliRunner;
use reth_db::DatabaseEnv;
use reth_node_builder::{NodeBuilder, WithLaunchContext};
use reth_node_core::{
    args::LogArgs,
    version::{LONG_VERSION, SHORT_VERSION},
};
use reth_optimism_node::args::RollupArgs;

// This allows us to manually enable node metrics features, required for proper jemalloc metric
// reporting
use reth_node_metrics as _;

/// The main op-reth cli interface.
///
/// This is the entrypoint to the executable.
#[derive(Debug, Parser)]
#[command(author, version = SHORT_VERSION, long_version = LONG_VERSION, about = "Reth", long_about = None)]
pub struct Cli<Spec: ChainSpecParser = OpChainSpecParser, Ext: clap::Args + fmt::Debug = RollupArgs>
{
    /// The command to run
    #[command(subcommand)]
    pub command: Commands<Spec, Ext>,

    /// The logging configuration for the CLI.
    #[command(flatten)]
    pub logs: LogArgs,
}

impl Cli {
    /// Parsers only the default CLI arguments
    pub fn parse_args() -> Self {
        Self::parse()
    }

    /// Parsers only the default CLI arguments from the given iterator
    pub fn try_parse_args_from<I, T>(itr: I) -> Result<Self, clap::error::Error>
    where
        I: IntoIterator<Item = T>,
        T: Into<OsString> + Clone,
    {
        Self::try_parse_from(itr)
    }
}

impl<C, Ext> Cli<C, Ext>
where
    C: ChainSpecParser<ChainSpec = OpChainSpec>,
    Ext: clap::Args + fmt::Debug,
{
    /// Configures the CLI and returns a [`CliApp`] instance.
    ///
    /// This method is used to prepare the CLI for execution by wrapping it in a
    /// [`CliApp`] that can be further configured before running.
    pub fn configure(self) -> CliApp<C, Ext> {
        CliApp::new(self)
    }

    /// Execute the configured cli command.
    ///
    /// This accepts a closure that is used to launch the node via the
    /// [`NodeCommand`](reth_cli_commands::node::NodeCommand).
    pub fn run<L, Fut>(self, launcher: L) -> eyre::Result<()>
    where
        L: FnOnce(WithLaunchContext<NodeBuilder<Arc<DatabaseEnv>, C::ChainSpec>>, Ext) -> Fut,
        Fut: Future<Output = eyre::Result<()>>,
    {
        self.with_runner(CliRunner::try_default_runtime()?, launcher)
    }

    /// Execute the configured cli command with the provided [`CliRunner`].
    pub fn with_runner<L, Fut>(self, runner: CliRunner, launcher: L) -> eyre::Result<()>
    where
        L: FnOnce(WithLaunchContext<NodeBuilder<Arc<DatabaseEnv>, C::ChainSpec>>, Ext) -> Fut,
        Fut: Future<Output = eyre::Result<()>>,
    {
        let mut this = self.configure();
        this.set_runner(runner);
        this.run(FnLauncher::new::<C, Ext>(async move |builder, chain_spec| {
            launcher(builder, chain_spec).await
        }))
    }
}

#[cfg(test)]
mod test {
    use crate::{chainspec::OpChainSpecParser, commands::Commands, Cli};
    use clap::Parser;
    use reth_cli_commands::{node::NoArgs, NodeCommand};
    use reth_optimism_chainspec::{BASE_MAINNET, OP_DEV};
    use reth_optimism_node::args::RollupArgs;

    #[test]
    fn parse_dev() {
        let cmd = NodeCommand::<OpChainSpecParser, NoArgs>::parse_from(["op-reth", "--dev"]);
        let chain = OP_DEV.clone();
        assert_eq!(cmd.chain.chain, chain.chain);
        assert_eq!(cmd.chain.genesis_hash(), chain.genesis_hash());
        assert_eq!(
            cmd.chain.paris_block_and_final_difficulty,
            chain.paris_block_and_final_difficulty
        );
        assert_eq!(cmd.chain.hardforks, chain.hardforks);

        assert!(cmd.rpc.http);
        assert!(cmd.network.discovery.disable_discovery);

        assert!(cmd.dev.dev);
    }

    #[test]
    fn parse_node() {
        let cmd = Cli::<OpChainSpecParser, RollupArgs>::parse_from([
            "op-reth",
            "node",
            "--chain",
            "base",
            "--datadir",
            "/mnt/datadirs/base",
            "--instance",
            "2",
            "--http",
            "--http.addr",
            "0.0.0.0",
            "--ws",
            "--ws.addr",
            "0.0.0.0",
            "--http.api",
            "admin,debug,eth,net,trace,txpool,web3,rpc,reth,ots",
            "--rollup.sequencer-http",
            "https://mainnet-sequencer.base.org",
            "--rpc-max-tracing-requests",
            "1000000",
            "--rpc.gascap",
            "18446744073709551615",
            "--rpc.max-connections",
            "429496729",
            "--rpc.max-logs-per-response",
            "0",
            "--rpc.max-subscriptions-per-connection",
            "10000",
            "--metrics",
            "9003",
            "--log.file.max-size",
            "100",
        ]);

        match cmd.command {
            Commands::Node(command) => {
                assert_eq!(command.chain.as_ref(), BASE_MAINNET.as_ref());
            }
            _ => panic!("unexpected command"),
        }
    }
}

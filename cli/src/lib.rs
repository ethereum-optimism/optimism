//! OP-Reth CLI implementation.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(docsrs, feature(doc_cfg, doc_auto_cfg))]

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

pub use commands::{import::ImportOpCommand, import_receipts::ImportReceiptsOpCommand};
use reth_optimism_chainspec::OpChainSpec;

use std::{ffi::OsString, fmt, sync::Arc};

use chainspec::OpChainSpecParser;
use clap::{command, Parser};
use commands::Commands;
use futures_util::Future;
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_runner::CliRunner;
use reth_db::DatabaseEnv;
use reth_node_builder::{NodeBuilder, WithLaunchContext};
use reth_node_core::{
    args::LogArgs,
    version::{LONG_VERSION, SHORT_VERSION},
};
use reth_optimism_consensus::OpBeaconConsensus;
use reth_optimism_evm::OpExecutorProvider;
use reth_optimism_node::{args::RollupArgs, OpNetworkPrimitives, OpNode};
use reth_tracing::FileWorkerGuard;
use tracing::info;

// This allows us to manually enable node metrics features, required for proper jemalloc metric
// reporting
use reth_node_metrics as _;
use reth_node_metrics::recorder::install_prometheus_recorder;

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
    pub fn with_runner<L, Fut>(mut self, runner: CliRunner, launcher: L) -> eyre::Result<()>
    where
        L: FnOnce(WithLaunchContext<NodeBuilder<Arc<DatabaseEnv>, C::ChainSpec>>, Ext) -> Fut,
        Fut: Future<Output = eyre::Result<()>>,
    {
        // add network name to logs dir
        // Add network name if available to the logs dir
        if let Some(chain_spec) = self.command.chain_spec() {
            self.logs.log_file_directory =
                self.logs.log_file_directory.join(chain_spec.chain.to_string());
        }
        let _guard = self.init_tracing()?;
        info!(target: "reth::cli", "Initialized tracing, debug log directory: {}", self.logs.log_file_directory);

        // Install the prometheus recorder to be sure to record all metrics
        let _ = install_prometheus_recorder();

        match self.command {
            Commands::Node(command) => {
                runner.run_command_until_exit(|ctx| command.execute(ctx, launcher))
            }
            Commands::Init(command) => {
                runner.run_blocking_until_ctrl_c(command.execute::<OpNode>())
            }
            Commands::InitState(command) => {
                runner.run_blocking_until_ctrl_c(command.execute::<OpNode>())
            }
            Commands::ImportOp(command) => {
                runner.run_blocking_until_ctrl_c(command.execute::<OpNode>())
            }
            Commands::ImportReceiptsOp(command) => {
                runner.run_blocking_until_ctrl_c(command.execute::<OpNode>())
            }
            Commands::DumpGenesis(command) => runner.run_blocking_until_ctrl_c(command.execute()),
            Commands::Db(command) => runner.run_blocking_until_ctrl_c(command.execute::<OpNode>()),
            Commands::Stage(command) => runner.run_command_until_exit(|ctx| {
                command.execute::<OpNode, _, _, OpNetworkPrimitives>(ctx, |spec| {
                    (OpExecutorProvider::optimism(spec.clone()), OpBeaconConsensus::new(spec))
                })
            }),
            Commands::P2P(command) => {
                runner.run_until_ctrl_c(command.execute::<OpNetworkPrimitives>())
            }
            Commands::Config(command) => runner.run_until_ctrl_c(command.execute()),
            Commands::Recover(command) => {
                runner.run_command_until_exit(|ctx| command.execute::<OpNode>(ctx))
            }
            Commands::Prune(command) => runner.run_until_ctrl_c(command.execute::<OpNode>()),
            #[cfg(feature = "dev")]
            Commands::TestVectors(command) => runner.run_until_ctrl_c(command.execute()),
        }
    }

    /// Initializes tracing with the configured options.
    ///
    /// If file logging is enabled, this function returns a guard that must be kept alive to ensure
    /// that all logs are flushed to disk.
    pub fn init_tracing(&self) -> eyre::Result<Option<FileWorkerGuard>> {
        let guard = self.logs.init_tracing()?;
        Ok(guard)
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

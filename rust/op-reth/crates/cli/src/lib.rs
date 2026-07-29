//! OP-Reth CLI implementation.

#![doc(
    html_logo_url = "https://raw.githubusercontent.com/paradigmxyz/reth/main/assets/reth-docs.png",
    html_favicon_url = "https://avatars0.githubusercontent.com/u/97369466?s=256",
    issue_tracker_base_url = "https://github.com/paradigmxyz/reth/issues/"
)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(docsrs, feature(doc_cfg))]

/// A configurable App on top of the cli parser.
pub mod app;
/// Optimism chain specification parser.
pub mod chainspec;
/// Optimism CLI commands.
pub mod commands;
/// Optimism CLI metrics.
pub mod metrics;
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
use reth_rpc_server_types::{DefaultRpcModuleValidator, RpcModuleValidator};

use std::{ffi::OsString, fmt, marker::PhantomData};

use chainspec::OpChainSpecParser;
use clap::Parser;
use commands::Commands;
use futures_util::Future;
use reth_cli::chainspec::ChainSpecParser;
use reth_cli_commands::launcher::FnLauncher;
use reth_cli_runner::CliRunner;
use reth_db::DatabaseEnv;
use reth_node_builder::{NodeBuilder, WithLaunchContext};
use reth_node_core::{
    args::{LogArgs, TraceArgs},
    version::version_metadata,
};
use reth_optimism_node::args::RollupArgs;

// This allows us to manually enable node metrics features, required for proper jemalloc metric
// reporting
use reth_node_metrics as _;

/// The main op-reth cli interface.
///
/// This is the entrypoint to the executable.
#[derive(Debug, Parser)]
#[command(author, name = version_metadata().name_client.as_ref(), version = version_metadata().short_version.as_ref(), long_version = version_metadata().long_version.as_ref(), about = "Reth", long_about = None)]
pub struct Cli<
    Spec: ChainSpecParser = OpChainSpecParser,
    Ext: clap::Args + fmt::Debug = RollupArgs,
    Rpc: RpcModuleValidator = DefaultRpcModuleValidator,
> {
    /// The command to run
    #[command(subcommand)]
    pub command: Commands<Spec, Ext>,

    /// The logging configuration for the CLI.
    #[command(flatten)]
    pub logs: LogArgs,

    /// The metrics configuration for the CLI.
    #[command(flatten)]
    pub traces: TraceArgs,

    /// Type marker for the RPC module validator
    #[arg(skip)]
    _phantom: PhantomData<Rpc>,
}

/// Upstream reth CLI arguments that op-reth does not support, as
/// `(subcommand, argument id, error shown when used)` entries.
///
/// These arguments live in upstream arg structs that op-reth reuses, so they cannot be removed at
/// the type level. They are hidden from all help output and rejected with a hard error after
/// parsing. Add new entries here as further unsupported upstream options are identified
/// (tracked in [#21687](https://github.com/ethereum-optimism/optimism/issues/21687)).
///
/// Binaries that build their clap command themselves instead of going through the parse methods
/// on [`Cli`] (e.g. the vendored op-rbuilder) are covered by a backstop check in
/// [`CliApp::run`](app::CliApp::run).
const DENIED_ARGS: &[(&str, &str, &str)] = &[("node", "minimal", MINIMAL_REMOVED_HELP)];

/// Startup error for `--minimal`, which configures pruning (including block-body pruning) that
/// op-node derivation cannot tolerate.
pub(crate) const MINIMAL_REMOVED_HELP: &str = "--minimal is not supported by op-reth and has been removed.\n\nIt prunes block bodies to a fixed 10,064-block window, which breaks op-node derivation\n(op-node reads the L1-info deposit transaction from historical block bodies).\n\nFor a pruned (non-archive) node, use the supported pruning recipe instead:\n  --prune.minimum-distance <BLOCKS>\n  --prune.receipts.distance <BLOCKS>\n  --prune.account-history.distance <BLOCKS>\n  --prune.storage-history.distance <BLOCKS>\nDo NOT prune block bodies.\n\nSee https://docs.optimism.io/node-operators/guides/management/archive-node#pruning-op-reth";

impl Cli {
    /// Parsers only the default CLI arguments, rejecting `DENIED_ARGS`
    pub fn parse_args() -> Self {
        Self::parse_with_denied_args()
    }

    /// Parsers only the default CLI arguments from the given iterator, rejecting `DENIED_ARGS`
    pub fn try_parse_args_from<I, T>(itr: I) -> Result<Self, clap::error::Error>
    where
        I: IntoIterator<Item = T>,
        T: Into<OsString> + Clone,
    {
        Self::try_parse_with_denied_args_from(itr)
    }
}

impl<C, Ext, Rpc> Cli<C, Ext, Rpc>
where
    C: ChainSpecParser,
    Ext: clap::Args + fmt::Debug,
    Rpc: RpcModuleValidator,
{
    /// Returns the clap command with all `DENIED_ARGS` hidden from help output.
    pub fn command_with_denied_args_hidden() -> clap::Command {
        let mut cmd = <Self as clap::CommandFactory>::command();
        for (subcommand, arg_id, _) in DENIED_ARGS {
            cmd = cmd.mut_subcommand(*subcommand, |sc| sc.mut_arg(*arg_id, |arg| arg.hide(true)));
        }
        cmd
    }

    /// Parses the CLI from the process arguments, hiding `DENIED_ARGS` from help output and
    /// exiting with a hard error when one is used.
    ///
    /// This is the entrypoint used by the `op-reth` binary.
    pub fn parse_with_denied_args() -> Self {
        match Self::try_parse_with_denied_args_from(std::env::args_os()) {
            Ok(cli) => cli,
            Err(err) => err.exit(),
        }
    }

    /// Parses the CLI from the given iterator, hiding `DENIED_ARGS` from help output and
    /// returning a hard error when one is used.
    pub fn try_parse_with_denied_args_from<I, T>(itr: I) -> Result<Self, clap::error::Error>
    where
        I: IntoIterator<Item = T>,
        T: Into<OsString> + Clone,
    {
        let mut cmd = Self::command_with_denied_args_hidden();
        let mut matches = cmd.try_get_matches_from_mut(itr)?;
        // Clap still accepts hidden arguments, so denied ones are rejected post-parse.
        for (subcommand, arg_id, help) in DENIED_ARGS {
            if let Some(sub) = matches.subcommand_matches(subcommand) &&
                sub.value_source(arg_id) == Some(clap::parser::ValueSource::CommandLine)
            {
                return Err(cmd.error(clap::error::ErrorKind::ValueValidation, *help));
            }
        }
        <Self as clap::FromArgMatches>::from_arg_matches_mut(&mut matches)
    }
}

impl<C, Ext, Rpc> Cli<C, Ext, Rpc>
where
    C: ChainSpecParser<ChainSpec = OpChainSpec>,
    Ext: clap::Args + fmt::Debug,
    Rpc: RpcModuleValidator,
{
    /// Configures the CLI and returns a [`CliApp`] instance.
    ///
    /// This method is used to prepare the CLI for execution by wrapping it in a
    /// [`CliApp`] that can be further configured before running.
    pub fn configure(self) -> CliApp<C, Ext, Rpc> {
        CliApp::new(self)
    }

    /// Execute the configured cli command.
    ///
    /// This accepts a closure that is used to launch the node via the
    /// [`NodeCommand`](reth_cli_commands::node::NodeCommand).
    pub fn run<L, Fut>(self, launcher: L) -> eyre::Result<()>
    where
        L: FnOnce(WithLaunchContext<NodeBuilder<DatabaseEnv, C::ChainSpec>>, Ext) -> Fut,
        Fut: Future<Output = eyre::Result<()>>,
    {
        self.with_runner(CliRunner::try_default_runtime()?, launcher)
    }

    /// Execute the configured cli command with the provided [`CliRunner`].
    pub fn with_runner<L, Fut>(self, runner: CliRunner, launcher: L) -> eyre::Result<()>
    where
        L: FnOnce(WithLaunchContext<NodeBuilder<DatabaseEnv, C::ChainSpec>>, Ext) -> Fut,
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
    use crate::{Cli, chainspec::OpChainSpecParser, commands::Commands};
    use clap::Parser;
    use reth_cli_commands::{NodeCommand, node::NoArgs};
    use reth_optimism_chainspec::{OP_DEV, OP_MAINNET};
    use reth_optimism_node::args::RollupArgs;

    #[test]
    fn deny_minimal_on_parse() {
        let err = Cli::<OpChainSpecParser, RollupArgs>::try_parse_with_denied_args_from([
            "op-reth",
            "node",
            "--minimal",
        ])
        .unwrap_err();
        assert_eq!(err.kind(), clap::error::ErrorKind::ValueValidation);
        let msg = err.to_string();
        assert!(msg.contains("--minimal is not supported by op-reth"), "{msg}");
        assert!(msg.contains("--prune.minimum-distance"), "{msg}");
        assert!(msg.contains("--prune.receipts.distance"), "{msg}");
    }

    #[test]
    fn deny_minimal_hidden_from_help() {
        let mut cmd = Cli::<OpChainSpecParser, RollupArgs>::command_with_denied_args_hidden();
        let help = cmd
            .find_subcommand_mut("node")
            .expect("node subcommand")
            .render_long_help()
            .to_string();
        assert!(!help.contains("--minimal"), "--minimal must be hidden from node help:\n{help}");
        assert!(help.contains("--full"), "other pruning args must stay visible:\n{help}");
    }

    #[test]
    fn deny_list_does_not_affect_other_args() {
        let cli = Cli::<OpChainSpecParser, RollupArgs>::try_parse_with_denied_args_from([
            "op-reth", "node", "--full",
        ])
        .unwrap();
        match cli.command {
            Commands::Node(command) => {
                assert!(command.pruning.full);
                assert!(!command.pruning.minimal);
            }
            _ => panic!("unexpected command"),
        }
    }

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
            "op-mainnet",
            "--datadir",
            "/mnt/datadirs/optimism",
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
            "https://mainnet-sequencer.optimism.io",
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
            "--tracing-otlp=http://localhost:4318/v1/traces",
            "--log.file.max-size",
            "100",
        ]);

        match cmd.command {
            Commands::Node(command) => {
                assert_eq!(command.chain.as_ref(), OP_MAINNET.as_ref());
            }
            _ => panic!("unexpected command"),
        }
    }
}

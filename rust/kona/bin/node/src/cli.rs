//! Contains the node CLI.

use crate::{
    commands::NodeCommand,
    flags::{GlobalArgs, init_unified_metrics},
    version,
};
use anyhow::Result;
use clap::{Command, CommandFactory, FromArgMatches, Parser, Subcommand};
use kona_cli::cli_styles;

/// Subcommands for the CLI.
#[derive(Debug, Clone, Subcommand)]
pub enum Commands {
    /// Runs the consensus node.
    #[command(alias = "n")]
    Node(NodeCommand),
}

/// The node CLI.
#[derive(Parser, Clone, Debug)]
#[command(
    author,
    version = version::short_version(),
    long_version = version::long_version(),
    about,
    styles = cli_styles(),
    long_about = None
)]
pub struct Cli {
    /// The subcommand to run.
    #[command(subcommand)]
    pub subcommand: Commands,
    /// Global arguments for the CLI.
    #[command(flatten)]
    pub global: GlobalArgs,
}

impl Cli {
    /// Parses the CLI arguments and environment variables, exiting with clap's message if they're
    /// invalid.
    ///
    /// Unlike [`Parser::parse`], a list flag whose environment variable is set but empty gets no
    /// entries, see [`ignore_empty_list_envs`].
    pub fn parse_args() -> Self {
        let matches = ignore_empty_list_envs(Self::command()).get_matches();
        // Formatted with the command, as `Parser::parse` does, for the usage and styling.
        Self::from_arg_matches(&matches)
            .map_err(|e| e.format(&mut Self::command()))
            .unwrap_or_else(|e| e.exit())
    }

    /// Runs the CLI.
    pub fn run(self) -> Result<()> {
        let Commands::Node(node) = self.subcommand;
        node.init_logs(&self.global)?;
        init_unified_metrics(&self.global.metrics)?;
        node.init_cli_metrics(&self.global.metrics)?;
        Self::run_until_ctrl_c(node.run(&self.global))
    }

    /// Run until ctrl-c is pressed.
    pub fn run_until_ctrl_c<F>(fut: F) -> Result<()>
    where
        F: std::future::Future<Output = Result<()>>,
    {
        let rt = Self::tokio_runtime().map_err(|e| anyhow::anyhow!(e))?;
        rt.block_on(async move {
            tokio::select! {
                res = fut => res,
                _ = tokio::signal::ctrl_c() => {
                    tracing::info!(target: "cli", "Received Ctrl-C, shutting down...");
                    Ok(())
                }
            }
        })
    }

    /// Creates a new default tokio multi-thread [Runtime](tokio::runtime::Runtime) with all
    /// features enabled
    pub fn tokio_runtime() -> Result<tokio::runtime::Runtime, std::io::Error> {
        tokio::runtime::Builder::new_multi_thread().enable_all().build()
    }
}

/// Unbinds the environment variable of each list flag, here and in subcommands, whose variable is
/// set but empty.
///
/// clap reads an empty variable as one empty entry, which the entry's parser rejects. op-node reads
/// it as an empty list, so deployments can leave such a variable empty, and so can kona's.
fn ignore_empty_list_envs(cmd: Command) -> Command {
    cmd.mut_args(|arg| {
        let empty_env = arg.get_value_delimiter().is_some() &&
            arg.get_env().and_then(std::env::var_os).is_some_and(|value| value.is_empty());
        if empty_env { arg.env(None::<&'static str>) } else { arg }
    })
    .mut_subcommands(ignore_empty_list_envs)
}

#[cfg(test)]
mod tests {
    use super::*;
    use rstest::rstest;

    /// Parses the CLI with `global_args` before the `node` subcommand, invoked as `subcommand`,
    /// with its required flags.
    fn try_parse(global_args: &[&str], subcommand: &str) -> Result<Cli, clap::Error> {
        let node_args = [
            subcommand,
            "--l1-eth-rpc",
            "http://localhost:8545",
            "--l1-beacon",
            "http://localhost:5052",
            "--l2-engine-rpc",
            "http://localhost:8551",
        ];
        Cli::try_parse_from(
            std::iter::once("kona-node").chain(global_args.iter().copied()).chain(node_args),
        )
    }

    /// A list flag under a subcommand, as the p2p list flags are under `node`, bound to an
    /// environment variable no other test uses.
    #[derive(Parser, Debug)]
    struct ListCommand {
        #[command(subcommand)]
        subcommand: ListSubcommand,
    }

    #[derive(Subcommand, Debug)]
    enum ListSubcommand {
        Run {
            #[arg(long, value_delimiter = ',', env = "KONA_NODE_TEST_EMPTY_LIST_ENV", value_parser = parse_entry)]
            list: Vec<String>,
        },
    }

    /// Accepts any entry but an empty one, like the bootnode and static peer parsers.
    fn parse_entry(entry: &str) -> Result<String, String> {
        if entry.is_empty() {
            Err("Entry must not be empty".to_string())
        } else {
            Ok(entry.to_string())
        }
    }

    fn parse_list(env: &str) -> Result<Vec<String>, String> {
        // SAFETY: only this test reads or writes this variable.
        unsafe { std::env::set_var("KONA_NODE_TEST_EMPTY_LIST_ENV", env) };
        let matches = ignore_empty_list_envs(ListCommand::command())
            .try_get_matches_from(["test", "run"])
            .map_err(|e| e.to_string())?;
        let ListSubcommand::Run { list } =
            ListCommand::from_arg_matches(&matches).map_err(|e| e.to_string())?.subcommand;
        Ok(list)
    }

    /// An empty environment variable is an empty list. Other values are split on commas, and an
    /// empty entry among them is an error.
    #[test]
    fn test_empty_list_env_is_empty_list() {
        assert_eq!(parse_list(""), Ok(vec![]));
        assert_eq!(parse_list("foo,bar"), Ok(vec!["foo".to_string(), "bar".to_string()]));
        for env in [",", ",,foo,,bar"] {
            let err = parse_list(env).unwrap_err();
            assert!(err.contains("Entry must not be empty"), "{env:?}: {err}");
        }
    }

    #[rstest]
    #[case::node_subcommand_long("node")]
    #[case::node_subcommand_short("n")]
    fn test_parse_cli(#[case] subcommand: &str) {
        try_parse(&[], subcommand).unwrap();
    }

    #[rstest]
    #[case::numeric_optimism("--l2-chain-id", "10", 10)]
    #[case::numeric_base("--l2-chain-id", "8453", 8453)]
    #[case::numeric_ethereum("--l2-chain-id", "1", 1)]
    #[case::string_optimism("--l2-chain-id", "optimism", 10)]
    #[case::string_base("--l2-chain-id", "base", 8453)]
    #[case::string_mainnet("--l2-chain-id", "mainnet", 1)]
    #[case::short_flag_numeric("-c", "10", 10)]
    #[case::short_flag_string("-c", "optimism", 10)]
    fn test_cli_l2_chain_id_valid(
        #[case] flag: &str,
        #[case] value: &str,
        #[case] expected_id: u64,
    ) {
        let cli = try_parse(&[flag, value], "node").unwrap();
        assert_eq!(cli.global.l2_chain_id.id(), expected_id);
    }

    #[rstest]
    #[case::invalid_string("invalid_chain")]
    #[case::empty_string("")]
    fn test_cli_l2_chain_id_invalid(#[case] invalid_value: &str) {
        let result = try_parse(&["--l2-chain-id", invalid_value], "node");
        assert!(result.is_err());
    }

    #[test]
    fn test_cli_l2_chain_id_default() {
        // Test that the default chain ID is 10 (Optimism)
        let cli = try_parse(&[], "node").unwrap();
        assert_eq!(cli.global.l2_chain_id.id(), 10);
    }
}

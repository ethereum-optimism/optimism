//! Contains the node CLI.

use crate::{
    commands::NodeCommand,
    flags::{GlobalArgs, init_unified_metrics},
    version,
};
use anyhow::Result;
use clap::{Parser, Subcommand};
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
    /// Runs the CLI.
    pub fn run(self) -> Result<()> {
        let Commands::Node(node) = self.subcommand;
        node.init_logs(&self.global)?;
        init_unified_metrics(&self.global.metrics)?;
        node.init_cli_metrics(&self.global.metrics)?;
        Self::run_until_ctrl_c(node.run(&self.global))
    }

    /// Run until Ctrl-C or SIGTERM is received.
    pub fn run_until_ctrl_c<F>(fut: F) -> Result<()>
    where
        F: std::future::Future<Output = Result<()>>,
    {
        let rt = Self::tokio_runtime().map_err(|e| anyhow::anyhow!(e))?;
        rt.block_on(async move {
            #[cfg(unix)]
            let mut terminate =
                tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())?;
            #[cfg(unix)]
            let terminate = terminate.recv();

            #[cfg(not(unix))]
            let terminate = std::future::pending::<Option<()>>();

            tokio::select! {
                res = fut => res,
                res = tokio::signal::ctrl_c() => {
                    res?;
                    tracing::info!(target: "cli", "Received Ctrl-C, shutting down...");
                    Ok(())
                }
                _ = terminate => {
                    tracing::info!(target: "cli", "Received SIGTERM, shutting down...");
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

    #[test]
    fn run_until_ctrl_c_returns_when_future_completes() {
        assert!(Cli::run_until_ctrl_c(async { Ok(()) }).is_ok());
    }

    #[cfg(unix)]
    #[test]
    fn run_until_ctrl_c_handles_shutdown_signals() {
        let Ok(signal) = std::env::var("KONA_TEST_SHUTDOWN_SIGNAL") else {
            // Isolate real signals from any other tests running in this process.
            for signal in ["-INT", "-TERM"] {
                let output = std::process::Command::new(std::env::current_exe().unwrap())
                    .args(["--exact", "cli::tests::run_until_ctrl_c_handles_shutdown_signals"])
                    .env("KONA_TEST_SHUTDOWN_SIGNAL", signal)
                    .output()
                    .expect("failed to spawn signal test process");
                assert!(
                    output.status.success(),
                    "{signal} test failed:\n{}\n{}",
                    String::from_utf8_lossy(&output.stdout),
                    String::from_utf8_lossy(&output.stderr),
                );
            }
            return;
        };

        let result = Cli::run_until_ctrl_c(async {
            // Let the select poll the Ctrl-C handler before sending a signal to this test process.
            tokio::task::yield_now().await;
            let status = std::process::Command::new("kill")
                .args([&signal, &std::process::id().to_string()])
                .status()
                .expect("failed to signal test process");
            assert!(status.success());
            tokio::time::sleep(std::time::Duration::from_secs(10)).await;
            Err(anyhow::anyhow!("shutdown signal was not handled"))
        });

        assert!(result.is_ok());
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

//! Contains the node CLI.

use crate::{commands::NodeCommand, flags::init_unified_metrics, version};
use anyhow::Result;
use clap::{Parser, Subcommand};
use kona_cli::cli_styles;

/// Subcommands for the CLI.
#[derive(Debug, Clone, Subcommand)]
pub enum Commands {
    /// Runs the consensus node.
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
}

impl Cli {
    /// Runs the CLI.
    pub fn run(self) -> Result<()> {
        let Commands::Node(node) = self.subcommand;
        node.init_logs()?;
        init_unified_metrics(&node.metrics)?;
        node.init_cli_metrics()?;
        Self::run_until_ctrl_c(node.run())
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

    /// The node subcommand's required flags.
    const REQUIRED_NODE_FLAGS: [&str; 8] = [
        "--l1-eth-rpc",
        "http://localhost:8545",
        "--l1-beacon",
        "http://localhost:5052",
        "--l2-engine-rpc",
        "http://localhost:8551",
        "--l2-engine-jwt-encoded",
        "0xabababababababababababababababababababababababababababababababab",
    ];

    /// Parses `kona-node node` with the node's required flags followed by `args`.
    fn try_parse_node(args: &[&str]) -> Result<Cli, clap::Error> {
        Cli::try_parse_from(
            ["kona-node", "node"]
                .into_iter()
                .chain(REQUIRED_NODE_FLAGS)
                .chain(args.iter().copied()),
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

    #[test]
    fn test_parse_node_flags() {
        let flags = [
            "-vvvv",
            "--chain",
            "8453",
            "--karst-override",
            "123",
            "--logs.stdout.format",
            "json",
            "--metrics.enabled",
        ];
        let Commands::Node(node) = try_parse_node(&flags).unwrap().subcommand;
        assert_eq!(node.log_args.level, 4);
        assert_eq!(node.chain.l2_chain_id.unwrap().id(), 8453);
        assert_eq!(node.chain.overrides.karst_override, Some(123));
        assert!(node.metrics.enabled);

        // The flags belong to the node subcommand, so they cannot precede it.
        let err = Cli::try_parse_from(
            ["kona-node", "--chain", "8453", "node"].into_iter().chain(REQUIRED_NODE_FLAGS),
        )
        .unwrap_err();
        assert_eq!(err.kind(), clap::error::ErrorKind::UnknownArgument);
    }
}

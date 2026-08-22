//! The `lokahi` CLI.

use crate::{
    config::{LokahiConfig, ResolvedConfig},
    metrics::init_chain_metrics,
    supernode::Supernode,
    version,
};
use anyhow::{Context, Result, anyhow};
use clap::{Parser, Subcommand};
use kona_cli::{LogArgs, LogConfig, MetricsArgs};
use std::path::PathBuf;

/// The `lokahi` CLI.
#[derive(Parser, Clone, Debug)]
#[command(
    author,
    version = version::short_version(),
    long_version = version::long_version(),
    about,
    long_about = None
)]
pub(crate) struct Cli {
    /// The subcommand to run.
    #[command(subcommand)]
    command: Commands,
    /// Logging arguments.
    #[command(flatten)]
    log_args: LogArgs,
    /// Prometheus metrics arguments.
    #[command(flatten)]
    metrics: MetricsArgs,
}

/// The `lokahi` subcommands.
#[derive(Debug, Clone, Subcommand)]
pub(crate) enum Commands {
    /// Runs the supernode over the chains a configuration file lists.
    Node(NodeCommand),
}

/// Runs the supernode.
///
/// Everything about the chains comes from the configuration file rather than from flags: with N
/// chains, a flag per chain per setting is not a usable interface, and the file is also what the
/// per-chain overlay needs in order to state the common settings once.
#[derive(Parser, Clone, Debug)]
pub(crate) struct NodeCommand {
    /// The path to the supernode's configuration file.
    #[arg(long, short = 'c', env = "LOKAHI_CONFIG")]
    config: PathBuf,
}

impl Cli {
    /// Runs the CLI.
    pub(crate) fn run(self) -> Result<()> {
        LogConfig::new(self.log_args.clone()).init_tracing_subscriber(None)?;
        self.init_metrics()?;

        match self.command {
            Commands::Node(ref node) => {
                let node = node.clone();
                let metrics_enabled = self.metrics.enabled;
                Self::run_until_ctrl_c(async move { node.run(metrics_enabled).await })
            }
        }
    }

    /// Starts the Prometheus endpoint, if it is enabled.
    ///
    /// The metrics of the crates the chains run are registered per chain once the configuration
    /// has been read, since each registration is labelled with the chain it is for.
    fn init_metrics(&self) -> Result<()> {
        self.metrics.init_metrics().map_err(Into::into)
    }

    /// Runs `fut` on a multi-threaded runtime until it finishes or the process is interrupted.
    fn run_until_ctrl_c<F>(fut: F) -> Result<()>
    where
        F: std::future::Future<Output = Result<()>>,
    {
        tokio::runtime::Builder::new_multi_thread().enable_all().build()?.block_on(async move {
            tokio::select! {
                result = fut => result,
                _ = tokio::signal::ctrl_c() => {
                    tracing::info!(target: "lokahi", "Received Ctrl-C, shutting down");
                    Ok(())
                }
            }
        })
    }
}

impl NodeCommand {
    /// Reads the configuration file and runs the supernode it describes.
    async fn run(self, metrics_enabled: bool) -> Result<()> {
        let config = self.read_config()?;

        if metrics_enabled {
            // One registration per chain: the crates label their series with the chain id they
            // were registered for, so this is what makes a supernode's metrics per chain.
            for chain in &config.chains {
                init_chain_metrics(chain.l2_chain_id);
            }
        }

        Supernode::load(config)?.run().await
    }

    /// Reads and resolves the configuration file.
    fn read_config(&self) -> Result<ResolvedConfig> {
        let toml = std::fs::read_to_string(&self.config)
            .with_context(|| format!("failed to read {}", self.config.display()))?;

        LokahiConfig::parse(&toml)
            .map_err(|e| anyhow!("failed to parse {}: {e}", self.config.display()))?
            .resolve()
            .with_context(|| format!("invalid configuration in {}", self.config.display()))
    }
}

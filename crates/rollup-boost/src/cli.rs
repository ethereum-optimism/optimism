use clap::Parser;
use jsonrpsee::{RpcModule, server::Server};
use std::{net::SocketAddr, path::PathBuf};
use tokio::signal::unix::{SignalKind, signal as unix_signal};
use tracing::{Level, info};

use crate::payload::PayloadSource;
use crate::{
    BlockSelectionPolicy, ClientArgs, DebugServer, FlashblocksArgs, ProxyLayer, RollupBoostServer,
    client::rpc::{BuilderArgs, L2ClientArgs},
    debug_api::ExecutionMode,
    get_version, init_metrics,
    probe::ProbeLayer,
};
use crate::{FlashblocksService, RpcClient};

#[derive(Clone, Debug, clap::Args)]
pub struct RollupBoostLibArgs {
    #[clap(flatten)]
    pub builder: BuilderArgs,

    #[clap(flatten)]
    pub l2_client: L2ClientArgs,

    /// Execution mode to start rollup boost with
    #[arg(long, env, default_value = "enabled")]
    pub execution_mode: ExecutionMode,

    #[arg(long, env)]
    pub block_selection_policy: Option<BlockSelectionPolicy>,

    /// Should we use the l2 client for computing state root
    #[arg(long, env, default_value = "false")]
    pub external_state_root: bool,

    /// Allow all engine API calls to builder even when marked as unhealthy
    /// This is default true assuming no builder CL set up
    #[arg(long, env, default_value = "false")]
    pub ignore_unhealthy_builders: bool,

    #[clap(flatten)]
    pub flashblocks: FlashblocksArgs,

    /// Duration in seconds between async health checks on the builder
    #[arg(long, env, default_value = "60")]
    pub health_check_interval: u64,

    /// Max duration in seconds between the unsafe head block of the builder and the current time
    #[arg(long, env, default_value = "10")]
    pub max_unsafe_interval: u64,
}

#[derive(Clone, Parser, Debug)]
#[clap(author, version = get_version(), about)]
pub struct RollupBoostServiceArgs {
    #[clap(flatten)]
    pub lib: RollupBoostLibArgs,

    /// Host to run the server on
    #[arg(long, env, default_value = "127.0.0.1")]
    pub rpc_host: String,

    /// Port to run the server on
    #[arg(long, env, default_value = "8081")]
    pub rpc_port: u16,

    // Enable tracing
    #[arg(long, env, default_value = "false")]
    pub tracing: bool,

    // Enable Prometheus metrics
    #[arg(long, env, default_value = "false")]
    pub metrics: bool,

    /// Host to run the metrics server on
    #[arg(long, env, default_value = "127.0.0.1")]
    pub metrics_host: String,

    /// Port to run the metrics server on
    #[arg(long, env, default_value = "9090")]
    pub metrics_port: u16,

    /// OTLP endpoint
    #[arg(long, env, default_value = "http://localhost:4317")]
    pub otlp_endpoint: String,

    /// Log level
    #[arg(long, env, default_value = "info")]
    pub log_level: Level,

    /// Log format
    #[arg(long, env, default_value = "text")]
    pub log_format: LogFormat,

    /// Redirect logs to a file
    #[arg(long, env)]
    pub log_file: Option<PathBuf>,

    /// Host to run the debug server on
    #[arg(long, env, default_value = "127.0.0.1")]
    pub debug_host: String,

    /// Debug server port
    #[arg(long, env, default_value = "5555")]
    pub debug_server_port: u16,
}

impl RollupBoostServiceArgs {
    pub async fn run(self) -> eyre::Result<()> {
        let _ = rustls::crypto::ring::default_provider().install_default();
        init_metrics(&self)?;

        let debug_addr = format!("{}:{}", self.debug_host, self.debug_server_port);
        let l2_client_args: ClientArgs = self.lib.l2_client.clone().into();
        let l2_http_client = l2_client_args.new_http_client(PayloadSource::L2)?;

        let builder_client_args: ClientArgs = self.lib.builder.clone().into();
        let builder_http_client = builder_client_args.new_http_client(PayloadSource::Builder)?;

        let (probe_layer, probes) = ProbeLayer::new();

        let (health_handle, rpc_module) = if self.lib.flashblocks.flashblocks {
            let rollup_boost = RollupBoostServer::<FlashblocksService>::new_from_args(
                self.lib.clone(),
                probes.clone(),
            )?;
            let health_handle = rollup_boost
                .spawn_health_check(self.lib.health_check_interval, self.lib.max_unsafe_interval);
            let debug_server = DebugServer::new(rollup_boost.execution_mode.clone());
            debug_server.run(&debug_addr).await?;
            let rpc_module: RpcModule<()> = rollup_boost.try_into()?;
            (health_handle, rpc_module)
        } else {
            let rollup_boost =
                RollupBoostServer::<RpcClient>::new_from_args(self.lib.clone(), probes.clone())?;
            let health_handle = rollup_boost
                .spawn_health_check(self.lib.health_check_interval, self.lib.max_unsafe_interval);
            let debug_server = DebugServer::new(rollup_boost.execution_mode.clone());
            debug_server.run(&debug_addr).await?;
            let rpc_module: RpcModule<()> = rollup_boost.try_into()?;
            (health_handle, rpc_module)
        };

        // Build and start the server
        let http_middleware =
            tower::ServiceBuilder::new()
                .layer(probe_layer)
                .layer(ProxyLayer::new(
                    l2_http_client.clone(),
                    builder_http_client.clone(),
                ));

        let server = Server::builder()
            .set_http_middleware(http_middleware)
            .build(format!("{}:{}", self.rpc_host, self.rpc_port).parse::<SocketAddr>()?)
            .await?;

        let local_addr = server.local_addr()?;
        info!("Starting server on {}", local_addr);

        let handle = server.start(rpc_module);

        let stop_handle = handle.clone();

        // Capture SIGINT and SIGTERM
        let mut sigint = unix_signal(SignalKind::interrupt())?;
        let mut sigterm = unix_signal(SignalKind::terminate())?;

        tokio::select! {
            _ = handle.stopped() => {
                // The server has already shut down by itself
                info!("Server stopped");
            }
            _ = health_handle => {
                info!("Health check task stopped");
            }
            _ = sigint.recv() => {
                info!("Received SIGINT, shutting down gracefully...");
                let _ = stop_handle.stop();
            }
            _ = sigterm.recv() => {
                info!("Received SIGTERM, shutting down gracefully...");
                let _ = stop_handle.stop();
            }
        }

        Ok(())
    }
}

impl Default for RollupBoostServiceArgs {
    fn default() -> Self {
        Self::parse_from::<_, &str>(std::iter::empty())
    }
}

#[derive(Clone, Debug)]
pub enum LogFormat {
    Json,
    Text,
}

impl std::str::FromStr for LogFormat {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "json" => Ok(LogFormat::Json),
            "text" => Ok(LogFormat::Text),
            _ => Err("Invalid log format".into()),
        }
    }
}

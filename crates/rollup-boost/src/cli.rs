use alloy_rpc_types_engine::JwtSecret;
use clap::{Parser, Subcommand};
use eyre::bail;
use jsonrpsee::{RpcModule, server::Server};
use parking_lot::Mutex;
use std::{
    net::{IpAddr, SocketAddr},
    path::PathBuf,
    str::FromStr,
    sync::Arc,
};
use tokio::signal::unix::{SignalKind, signal as unix_signal};
use tracing::{Level, info};

use crate::{
    BlockSelectionPolicy, DebugClient, EngineApiExt, Flashblocks, FlashblocksArgs, ProxyLayer,
    RollupBoostServer, RpcClient,
    client::rpc::{BuilderArgs, L2ClientArgs},
    debug_api::ExecutionMode,
    init_metrics,
    payload::PayloadSource,
    probe::ProbeLayer,
};

#[derive(Clone, Parser, Debug)]
#[clap(author, version, about)]
pub struct Args {
    #[command(subcommand)]
    pub command: Option<Commands>,

    #[clap(flatten)]
    pub builder: BuilderArgs,

    #[clap(flatten)]
    pub l2_client: L2ClientArgs,

    /// Duration in seconds between async health checks on the builder
    #[arg(long, env, default_value = "60")]
    pub health_check_interval: u64,

    /// Max duration in seconds between the unsafe head block of the builder and the current time
    #[arg(long, env, default_value = "10")]
    pub max_unsafe_interval: u64,

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

    /// Execution mode to start rollup boost with
    #[arg(long, env, default_value = "enabled")]
    pub execution_mode: ExecutionMode,

    #[arg(long, env)]
    pub block_selection_policy: Option<BlockSelectionPolicy>,

    #[clap(flatten)]
    pub flashblocks: FlashblocksArgs,
}

impl Args {
    pub async fn run(self) -> eyre::Result<()> {
        let _ = rustls::crypto::ring::default_provider().install_default();

        let debug_addr = format!("{}:{}", self.debug_host, self.debug_server_port);

        // Handle commands if present
        if let Some(cmd) = self.command {
            let debug_addr = format!("http://{debug_addr}");
            return match cmd {
                Commands::Debug { command } => match command {
                    DebugCommands::SetExecutionMode { execution_mode } => {
                        let client = DebugClient::new(debug_addr.as_str())?;
                        let result = client.set_execution_mode(execution_mode).await?;
                        println!("Response: {:?}", result.execution_mode);

                        Ok(())
                    }
                    DebugCommands::ExecutionMode {} => {
                        let client = DebugClient::new(debug_addr.as_str())?;
                        let result = client.get_execution_mode().await?;
                        println!("Execution mode: {:?}", result.execution_mode);

                        Ok(())
                    }
                },
            };
        }

        init_metrics(&self)?;

        let l2_client_args = self.l2_client;

        let l2_auth_jwt = if let Some(secret) = l2_client_args.l2_jwt_token {
            secret
        } else if let Some(path) = l2_client_args.l2_jwt_path.as_ref() {
            JwtSecret::from_file(path)?
        } else {
            bail!("Missing L2 Client JWT secret");
        };

        let l2_client = RpcClient::new(
            l2_client_args.l2_url.clone(),
            l2_auth_jwt,
            l2_client_args.l2_timeout,
            PayloadSource::L2,
        )?;

        let builder_args = self.builder;
        let builder_auth_jwt = if let Some(secret) = builder_args.builder_jwt_token {
            secret
        } else if let Some(path) = builder_args.builder_jwt_path.as_ref() {
            JwtSecret::from_file(path)?
        } else {
            bail!("Missing Builder JWT secret");
        };

        let builder_client = RpcClient::new(
            builder_args.builder_url.clone(),
            builder_auth_jwt,
            builder_args.builder_timeout,
            PayloadSource::Builder,
        )?;

        let (probe_layer, probes) = ProbeLayer::new();

        let builder_client: Arc<dyn EngineApiExt> = if self.flashblocks.flashblocks {
            let inbound_url = self.flashblocks.flashblocks_builder_url;
            let outbound_addr = SocketAddr::new(
                IpAddr::from_str(&self.flashblocks.flashblocks_host)?,
                self.flashblocks.flashblocks_port,
            );

            Arc::new(Flashblocks::run(
                builder_client.clone(),
                inbound_url,
                outbound_addr,
                self.flashblocks.flashblock_builder_ws_reconnect_ms,
            )?)
        } else {
            Arc::new(builder_client)
        };

        let execution_mode = Arc::new(Mutex::new(self.execution_mode));
        let rollup_boost = RollupBoostServer::new(
            l2_client,
            builder_client,
            execution_mode.clone(),
            self.block_selection_policy,
            probes.clone(),
        );

        let health_handle =
            rollup_boost.spawn_health_check(self.health_check_interval, self.max_unsafe_interval);

        // Spawn the debug server
        rollup_boost.start_debug_server(debug_addr.as_str()).await?;

        let module: RpcModule<()> = rollup_boost.try_into()?;

        // Build and start the server
        info!("Starting server on :{}", self.rpc_port);

        let http_middleware =
            tower::ServiceBuilder::new()
                .layer(probe_layer)
                .layer(ProxyLayer::new(
                    l2_client_args.l2_url,
                    l2_auth_jwt,
                    l2_client_args.l2_timeout,
                    builder_args.builder_url,
                    builder_auth_jwt,
                    builder_args.builder_timeout,
                ));

        let server = Server::builder()
            .set_http_middleware(http_middleware)
            .build(format!("{}:{}", self.rpc_host, self.rpc_port).parse::<SocketAddr>()?)
            .await?;
        let handle = server.start(module);

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

#[derive(Clone, Subcommand, Debug)]
pub enum Commands {
    /// Debug commands
    Debug {
        #[command(subcommand)]
        command: DebugCommands,
    },
}

#[derive(Clone, Subcommand, Debug)]
pub enum DebugCommands {
    /// Set the execution mode
    SetExecutionMode { execution_mode: ExecutionMode },

    /// Get the execution mode
    ExecutionMode {},
}

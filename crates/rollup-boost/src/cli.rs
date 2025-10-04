use alloy_rpc_types_engine::JwtSecret;
use clap::Parser;
use eyre::bail;
use jsonrpsee::{RpcModule, server::Server};
use jsonrpsee_core::middleware::RpcServiceBuilder;
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
    BlockSelectionPolicy, Flashblocks, FlashblocksArgs, ProxyLayer, RollupBoostServer, RpcClient,
    RpcProxyClient,
    client::rpc::{BuilderArgs, L2ClientArgs},
    debug_api::ExecutionMode,
    get_version, init_metrics,
    payload::PayloadSource,
    probe::ProbeLayer,
};

#[derive(Clone, Parser, Debug)]
#[clap(author, version = get_version(), about)]
pub struct RollupBoostArgs {
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

    /// Should we use the l2 client for computing state root
    #[arg(long, env, default_value = "false")]
    pub external_state_root: bool,

    /// Allow all engine API calls to builder even when marked as unhealthy
    /// This is default true assuming no builder CL set up
    #[arg(long, env, default_value = "false")]
    pub ignore_unhealthy_builders: bool,

    #[clap(flatten)]
    pub flashblocks: FlashblocksArgs,
}

impl RollupBoostArgs {
    pub async fn run(self) -> eyre::Result<()> {
        let _ = rustls::crypto::ring::default_provider().install_default();
        init_metrics(&self)?;

        let debug_addr = format!("{}:{}", self.debug_host, self.debug_server_port);
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
        let execution_mode = Arc::new(Mutex::new(self.execution_mode));

        let (rpc_module, health_handle): (RpcModule<()>, _) = if self.flashblocks.flashblocks {
            let flashblocks_args = self.flashblocks;
            let inbound_url = flashblocks_args.flashblocks_builder_url;
            let outbound_addr = SocketAddr::new(
                IpAddr::from_str(&flashblocks_args.flashblocks_host)?,
                flashblocks_args.flashblocks_port,
            );

            let builder_client = Arc::new(Flashblocks::run(
                builder_client.clone(),
                inbound_url,
                outbound_addr,
                flashblocks_args.flashblock_builder_ws_reconnect_ms,
            )?);

            let rollup_boost = RollupBoostServer::new(
                l2_client,
                builder_client,
                execution_mode.clone(),
                self.block_selection_policy,
                probes.clone(),
                self.external_state_root,
                self.ignore_unhealthy_builders,
            );

            let health_handle = rollup_boost
                .spawn_health_check(self.health_check_interval, self.max_unsafe_interval);

            // Spawn the debug server
            rollup_boost.start_debug_server(debug_addr.as_str()).await?;
            (rollup_boost.try_into()?, health_handle)
        } else {
            let rollup_boost = RollupBoostServer::new(
                l2_client,
                Arc::new(builder_client),
                execution_mode.clone(),
                self.block_selection_policy,
                probes.clone(),
                self.external_state_root,
                self.ignore_unhealthy_builders,
            );

            let health_handle = rollup_boost
                .spawn_health_check(self.health_check_interval, self.max_unsafe_interval);

            // Spawn the debug server
            rollup_boost.start_debug_server(debug_addr.as_str()).await?;
            (rollup_boost.try_into()?, health_handle)
        };

        // Build and start the server
        info!("Starting server on :{}", self.rpc_port);

        let http_middleware = tower::ServiceBuilder::new().layer(probe_layer);
        let l2_client = RpcProxyClient::new_l2(
            l2_client_args.l2_url.clone(),
            l2_auth_jwt,
            l2_client_args.l2_timeout,
        );
        let builder_client = RpcProxyClient::new_builder(
            builder_args.builder_url.clone(),
            builder_auth_jwt,
            builder_args.builder_timeout,
        );
        let rpc_middleware =
            RpcServiceBuilder::new().layer(ProxyLayer::new(l2_client, builder_client));

        let server = Server::builder()
            .set_http_middleware(http_middleware)
            .set_rpc_middleware(rpc_middleware)
            .build(format!("{}:{}", self.rpc_host, self.rpc_port).parse::<SocketAddr>()?)
            .await?;
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

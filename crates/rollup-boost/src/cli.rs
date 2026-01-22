use clap::Parser;
use jsonrpsee::{RpcModule, server::Server};
use std::{net::SocketAddr, path::PathBuf};
use tokio::signal::unix::{SignalKind, signal as unix_signal};
use tracing::{Level, info};

use crate::{
    BlockSelectionPolicy, ClientArgs, DebugServer, FlashblocksP2PArgs, FlashblocksWsArgs,
    ProxyLayer, RollupBoostServer,
    client::rpc::{BuilderArgs, L2ClientArgs},
    debug_api::ExecutionMode,
    get_version, init_metrics,
    probe::ProbeLayer,
};
use rollup_boost_types::payload::PayloadSource;

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
    pub flashblocks_ws: Option<FlashblocksWsArgs>,

    #[clap(flatten)]
    pub flashblocks_p2p: Option<FlashblocksP2PArgs>,

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

        let rollup_boost = RollupBoostServer::new_from_args(self.lib.clone(), probes.clone())?;
        let health_handle = rollup_boost
            .spawn_health_check(self.lib.health_check_interval, self.lib.max_unsafe_interval);
        let debug_server = DebugServer::new(rollup_boost.execution_mode.clone());
        debug_server.run(&debug_addr).await?;
        let rpc_module: RpcModule<()> = rollup_boost.try_into()?;

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

#[cfg(test)]
pub mod tests {
    use std::result::Result;

    use super::*;

    const SECRET: &str = "f79ae8046bc11c9927afe911db7143c51a806c4a537cc08e0d37140b0192f430";
    const FLASHBLOCKS_SK: &str = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef";
    const FLASHBLOCKS_VK: &str = "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210";

    #[test]
    fn test_parse_args_minimal() -> Result<(), Box<dyn std::error::Error>> {
        let args = RollupBoostServiceArgs::try_parse_from(["rollup-boost"])?;

        assert!(!args.tracing);
        assert!(!args.metrics);
        assert_eq!(args.rpc_host, "127.0.0.1");
        assert_eq!(args.rpc_port, 8081);
        assert!(args.lib.flashblocks_ws.is_none());
        assert!(args.lib.flashblocks_p2p.is_none());

        Ok(())
    }

    #[test]
    fn test_parse_args_missing_flashblocks_flag() -> Result<(), Box<dyn std::error::Error>> {
        let args = RollupBoostServiceArgs::try_parse_from([
            "rollup-boost",
            "--builder-jwt-token",
            SECRET,
            "--l2-jwt-token",
            SECRET,
            "--flashblocks-authorizer-sk",
            FLASHBLOCKS_SK,
            "--flashblocks-builder-vk",
            FLASHBLOCKS_VK,
        ]);

        assert!(
            args.is_err(),
            "flashblocks args should be invalid without --flashblocks-p2p flag"
        );

        Ok(())
    }

    #[test]
    fn test_parse_args_with_flashblocks_flag() -> Result<(), Box<dyn std::error::Error>> {
        let args = RollupBoostServiceArgs::try_parse_from([
            "rollup-boost",
            "--builder-jwt-token",
            SECRET,
            "--l2-jwt-token",
            SECRET,
            "--flashblocks-p2p",
            "--flashblocks-authorizer-sk",
            FLASHBLOCKS_SK,
            "--flashblocks-builder-vk",
            FLASHBLOCKS_VK,
        ])?;

        let flashblocks = args
            .lib
            .flashblocks_p2p
            .expect("flashblocks should be Some when flag is passed");
        assert!(flashblocks.flashblocks_p2p);

        Ok(())
    }

    #[test]
    fn test_parse_args_with_flashblocks_custom_values() -> Result<(), Box<dyn std::error::Error>> {
        let args = RollupBoostServiceArgs::try_parse_from([
            "rollup-boost",
            "--builder-jwt-token",
            SECRET,
            "--l2-jwt-token",
            SECRET,
            "--flashblocks-p2p",
            "--flashblocks-authorizer-sk",
            FLASHBLOCKS_SK,
            "--flashblocks-builder-vk",
            FLASHBLOCKS_VK,
        ])?;

        let flashblocks = args
            .lib
            .flashblocks_p2p
            .expect("flashblocks should be Some when flag is passed");
        assert!(flashblocks.flashblocks_p2p);

        Ok(())
    }

    #[test]
    fn test_parse_args_with_all_options() -> Result<(), Box<dyn std::error::Error>> {
        let args = RollupBoostServiceArgs::try_parse_from([
            "rollup-boost",
            "--builder-jwt-token",
            SECRET,
            "--l2-jwt-token",
            SECRET,
            "--health-check-interval",
            "120",
            "--max-unsafe-interval",
            "20",
            "--rpc-host",
            "0.0.0.0",
            "--rpc-port",
            "9090",
            "--tracing",
            "--metrics",
            "--metrics-host",
            "192.168.1.1",
            "--metrics-port",
            "8080",
            "--log-level",
            "debug",
            "--log-format",
            "json",
            "--debug-host",
            "localhost",
            "--debug-server-port",
            "6666",
            "--execution-mode",
            "disabled",
            "--flashblocks-p2p",
            "--flashblocks-authorizer-sk",
            FLASHBLOCKS_SK,
            "--flashblocks-builder-vk",
            FLASHBLOCKS_VK,
        ])?;

        assert_eq!(args.lib.health_check_interval, 120);
        assert_eq!(args.lib.max_unsafe_interval, 20);
        assert_eq!(args.rpc_host, "0.0.0.0");
        assert_eq!(args.rpc_port, 9090);
        assert!(args.tracing);
        assert!(args.metrics);
        assert_eq!(args.metrics_host, "192.168.1.1");
        assert_eq!(args.metrics_port, 8080);
        assert_eq!(args.log_level, Level::DEBUG);
        assert_eq!(args.debug_host, "localhost");
        assert_eq!(args.debug_server_port, 6666);

        let flashblocks = args
            .lib
            .flashblocks_p2p
            .expect("flashblocks should be Some when flag is passed");
        assert!(flashblocks.flashblocks_p2p);

        Ok(())
    }

    #[test]
    fn test_parse_args_missing_jwt_succeeds_at_parse_time() {
        // JWT validation happens at runtime, not parse time, so this should succeed
        let result =
            RollupBoostServiceArgs::try_parse_from(["rollup-boost", "--builder-jwt-token", SECRET]);

        assert!(result.is_ok());
        let args = result.unwrap();
        assert!(args.lib.builder.builder_jwt_token.is_some());
        assert!(args.lib.l2_client.l2_jwt_token.is_none());
    }

    #[test]
    fn test_parse_args_invalid_flashblocks_sk() {
        let result = RollupBoostServiceArgs::try_parse_from([
            "rollup-boost",
            "--builder-jwt-token",
            SECRET,
            "--l2-jwt-token",
            SECRET,
            "--flashblocks-p2p",
            "--flashblocks-authorizer-sk",
            "invalid_hex",
            "--flashblocks-builder-vk",
            FLASHBLOCKS_VK,
        ]);

        assert!(result.is_err());
    }

    #[test]
    fn test_parse_args_invalid_flashblocks_vk() {
        let result = RollupBoostServiceArgs::try_parse_from([
            "rollup-boost",
            "--builder-jwt-token",
            SECRET,
            "--l2-jwt-token",
            SECRET,
            "--flashblocks-p2p",
            "--flashblocks-authorizer-sk",
            FLASHBLOCKS_SK,
            "--flashblocks-builder-vk",
            "invalid_hex",
        ]);

        assert!(result.is_err());
    }

    #[test]
    fn test_log_format_parsing() -> Result<(), Box<dyn std::error::Error>> {
        let json_args = RollupBoostServiceArgs::try_parse_from([
            "rollup-boost",
            "--builder-jwt-token",
            SECRET,
            "--l2-jwt-token",
            SECRET,
            "--log-format",
            "json",
        ])?;

        match json_args.log_format {
            LogFormat::Json => {}
            LogFormat::Text => panic!("Expected Json format"),
        }

        let text_args = RollupBoostServiceArgs::try_parse_from([
            "rollup-boost",
            "--builder-jwt-token",
            SECRET,
            "--l2-jwt-token",
            SECRET,
            "--log-format",
            "text",
        ])?;

        match text_args.log_format {
            LogFormat::Text => {}
            LogFormat::Json => panic!("Expected Text format"),
        }

        Ok(())
    }

    #[test]
    fn test_parse_args_with_jwt_paths() -> Result<(), Box<dyn std::error::Error>> {
        use std::io::Write;
        use tempfile::NamedTempFile;

        let mut builder_jwt_file = NamedTempFile::new()?;
        writeln!(builder_jwt_file, "{}", SECRET)?;
        let builder_jwt_path = builder_jwt_file.path();

        let mut l2_jwt_file = NamedTempFile::new()?;
        writeln!(l2_jwt_file, "{}", SECRET)?;
        let l2_jwt_path = l2_jwt_file.path();

        let args = RollupBoostServiceArgs::try_parse_from([
            "rollup-boost",
            "--builder-jwt-path",
            builder_jwt_path.to_str().unwrap(),
            "--l2-jwt-path",
            l2_jwt_path.to_str().unwrap(),
        ])?;

        assert!(args.lib.builder.builder_jwt_path.is_some());
        assert!(args.lib.l2_client.l2_jwt_path.is_some());

        Ok(())
    }
}

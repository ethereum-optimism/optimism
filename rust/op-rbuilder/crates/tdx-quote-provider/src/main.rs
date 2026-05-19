use clap::Parser;
use dotenvy::dotenv;
use metrics_exporter_prometheus::PrometheusBuilder;
use tracing::{Level, info};
use tracing_subscriber::filter::EnvFilter;

use crate::server::{Server, ServerConfig};

mod metrics;
mod provider;
mod server;

#[derive(Clone, Parser, Debug)]
#[command(about = "TDX Quote Provider CLI")]
struct Args {
    /// Host to run the http server on
    #[arg(long, env, default_value = "127.0.0.1")]
    pub service_host: String,

    /// Port to run the http server on
    #[arg(long, env, default_value = "8181")]
    pub service_port: u16,

    // Enable Prometheus metrics
    #[arg(long, env, default_value = "false")]
    pub metrics: bool,

    /// Host to run the metrics server on
    #[arg(long, env, default_value = "127.0.0.1")]
    pub metrics_host: String,

    /// Port to run the metrics server on
    #[arg(long, env, default_value = "9090")]
    pub metrics_port: u16,

    /// Use mock attestation for testing
    #[arg(long, env, default_value = "false")]
    pub mock: bool,

    /// Path to the mock attestation file
    #[arg(long, env, default_value = "")]
    pub mock_attestation_path: String,

    /// Log level
    #[arg(long, env, default_value = "info")]
    pub log_level: Level,

    /// Log format
    #[arg(long, env, default_value = "text")]
    pub log_format: String,
}

#[tokio::main]
async fn main() -> eyre::Result<()> {
    dotenv().ok();

    let args = Args::parse();

    if args.log_format == "json" {
        tracing_subscriber::fmt()
            .json()
            .with_env_filter(EnvFilter::new(args.log_level.to_string()))
            .with_ansi(false)
            .init();
    } else {
        tracing_subscriber::fmt()
            .with_env_filter(EnvFilter::new(args.log_level.to_string()))
            .init();
    }

    info!("Starting TDX quote provider");

    if args.metrics {
        let metrics_addr = format!("{}:{}", args.metrics_host, args.metrics_port);
        info!(message = "starting metrics server", address = metrics_addr);
        let socket_addr: std::net::SocketAddr =
            metrics_addr.parse().expect("invalid metrics address");
        let builder = PrometheusBuilder::new().with_http_listener(socket_addr);

        builder
            .install()
            .expect("failed to setup Prometheus endpoint")
    }

    // Start the server
    let server = Server::new(ServerConfig {
        listen_addr: format!("{}:{}", args.service_host, args.service_port)
            .parse()
            .unwrap(),
        use_mock: args.mock,
        mock_attestation_path: args.mock_attestation_path,
    });

    server.listen().await
}

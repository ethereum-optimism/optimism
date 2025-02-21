use clap::{Parser, Subcommand, arg};
use client::{BuilderArgs, ExecutionClient, L2ClientArgs};
use debug_api::DebugClient;
use metrics::{ClientMetrics, ServerMetrics};
use server::ExecutionMode;
use std::{net::SocketAddr, sync::Arc};

use alloy_rpc_types_engine::JwtSecret;
use dotenv::dotenv;
use eyre::bail;
use http::StatusCode;
use hyper::service::service_fn;
use hyper::{Request, Response, server::conn::http1};
use hyper_util::rt::TokioIo;
use jsonrpsee::RpcModule;
use jsonrpsee::http_client::HttpBody;
use jsonrpsee::server::Server;
<<<<<<< HEAD
use jsonrpsee::RpcModule;
=======
use metrics::ServerMetrics;
>>>>>>> 44761b4 (Better LRU and fix for deadlock)
use metrics_exporter_prometheus::{PrometheusBuilder, PrometheusHandle};
use metrics_util::layers::{PrefixLayer, Stack};
use opentelemetry::global;
use opentelemetry_otlp::WithExportConfig;
use opentelemetry_sdk::{Resource, propagation::TraceContextPropagator, trace::Config};
use proxy::ProxyLayer;
use server::{PayloadSource, RollupBoostServer};

use tokio::net::TcpListener;
use tokio::signal::unix::{SignalKind, signal as unix_signal};
use tracing::{Level, error, info};
use tracing_subscriber::EnvFilter;

mod auth_layer;
mod client;
mod debug_api;
#[cfg(all(feature = "integration", test))]
mod integration;
mod metrics;
mod proxy;
mod server;

#[derive(Parser, Debug)]
#[clap(author, version, about)]
struct Args {
    #[command(subcommand)]
    command: Option<Commands>,

    #[clap(flatten)]
    builder: BuilderArgs,

    #[clap(flatten)]
    l2_client: L2ClientArgs,

    /// Disable using the proposer to sync the builder node
    #[arg(long, env, default_value = "false")]
    no_boost_sync: bool,

    /// Host to run the server on
    #[arg(long, env, default_value = "0.0.0.0")]
    rpc_host: String,

    /// Port to run the server on
    #[arg(long, env, default_value = "8081")]
    rpc_port: u16,

    // Enable tracing
    #[arg(long, env, default_value = "false")]
    tracing: bool,

    // Enable Prometheus metrics
    #[arg(long, env, default_value = "false")]
    metrics: bool,

    /// Host to run the metrics server on
    #[arg(long, env, default_value = "0.0.0.0")]
    metrics_host: String,

    /// Port to run the metrics server on
    #[arg(long, env, default_value = "9090")]
    metrics_port: u16,

    /// OTLP endpoint
    #[arg(long, env, default_value = "http://localhost:4317")]
    otlp_endpoint: String,

    /// Log level
    #[arg(long, env, default_value = "info")]
    log_level: Level,

    /// Log format
    #[arg(long, env, default_value = "text")]
    log_format: String,

    /// Debug server port
    #[arg(long, env, default_value = "5555")]
    debug_server_port: u16,
}

#[derive(Subcommand, Debug)]
enum Commands {
    /// Debug commands
    Debug {
        #[command(subcommand)]
        command: DebugCommands,
    },
}

#[derive(Subcommand, Debug)]
enum DebugCommands {
    /// Set the execution mode
    SetExecutionMode { execution_mode: ExecutionMode },

    /// Get the execution mode
    ExecutionMode {},
}

#[tokio::main]
async fn main() -> eyre::Result<()> {
    // Load .env file
    dotenv().ok();
    rustls::crypto::ring::default_provider()
        .install_default()
        .expect("Failed to install TLS ring CryptoProvider");
    let args: Args = Args::parse();

    // Handle commands if present
    if let Some(cmd) = args.command {
        match cmd {
            Commands::Debug { command } => match command {
                DebugCommands::SetExecutionMode { execution_mode } => {
                    let client = DebugClient::default();
                    let result = client.set_execution_mode(execution_mode).await.unwrap();
                    println!("Response: {:?}", result.execution_mode);

                    return Ok(());
                }
                DebugCommands::ExecutionMode {} => {
                    let client = DebugClient::default();
                    let result = client.get_execution_mode().await.unwrap();
                    println!("Execution mode: {:?}", result.execution_mode);

                    return Ok(());
                }
            },
        }
    }

    // Initialize logging
    let log_format = args.log_format.to_lowercase();
    let log_level = args.log_level.to_string();
    if log_format == "json" {
        // JSON log format
        tracing_subscriber::fmt()
            .json() // Use JSON format
            .with_env_filter(EnvFilter::new(log_level)) // Set log level
            .with_ansi(false) // Disable colored logging
            .init();
    } else {
        // Default (text) log format
        tracing_subscriber::fmt()
            .with_env_filter(EnvFilter::new(log_level)) // Set log level
            .with_ansi(false) // Disable colored logging
            .init();
    }

    let metrics = if args.metrics {
        let recorder = PrometheusBuilder::new().build_recorder();
        let handle = recorder.handle();

        // Build metrics stack
        Stack::new(recorder)
            .push(PrefixLayer::new("rollup-boost"))
            .install()?;

        // Start the metrics server
        let metrics_addr = format!("{}:{}", args.metrics_host, args.metrics_port);
        let addr: SocketAddr = metrics_addr.parse()?;
        tokio::spawn(init_metrics_server(addr, handle)); // Run the metrics server in a separate task
        Some(Arc::new(ServerMetrics::default()))
    } else {
        None
    };

    // Telemetry setup
    if args.tracing {
        init_tracing(&args.otlp_endpoint);
    }

    let l2_client_args = args.l2_client;

    let l2_auth_jwt = if let Some(secret) = l2_client_args.l2_jwt_token {
        secret
    } else if let Some(path) = l2_client_args.l2_jwt_path.as_ref() {
        JwtSecret::from_file(path)?
    } else {
        bail!("Missing L2 Client JWT secret");
    };

    let l2_metrics = if args.metrics {
        Some(Arc::new(ClientMetrics::new(&PayloadSource::L2)))
    } else {
        None
    };
    let l2_client = ExecutionClient::new(
        l2_client_args.l2_url.clone(),
        l2_auth_jwt,
        l2_client_args.l2_timeout,
        l2_metrics,
        PayloadSource::L2,
    )?;

    let builder_args = args.builder;
    let builder_auth_jwt = if let Some(secret) = builder_args.builder_jwt_token {
        secret
    } else if let Some(path) = builder_args.builder_jwt_path.as_ref() {
        JwtSecret::from_file(path)?
    } else {
        bail!("Missing Builder JWT secret");
    };

    let builder_metrics = if args.metrics {
        Some(Arc::new(ClientMetrics::new(&PayloadSource::Builder)))
    } else {
        None
    };

    let builder_client = ExecutionClient::new(
        builder_args.builder_url.clone(),
        builder_auth_jwt,
        builder_args.builder_timeout,
        builder_metrics,
        PayloadSource::Builder,
    )?;

    let boost_sync_enabled = !args.no_boost_sync;
    if boost_sync_enabled {
        info!("Boost sync enabled");
    }

    let rollup_boost = RollupBoostServer::new(
        l2_client,
        builder_client,
        boost_sync_enabled,
        metrics.clone(),
    );

    // Spawn the debug server
    rollup_boost
        .start_debug_server(args.debug_server_port)
        .await?;

    let module: RpcModule<()> = rollup_boost.try_into()?;

    // Build and start the server
    info!("Starting server on :{}", args.rpc_port);

    let service_builder = tower::ServiceBuilder::new().layer(ProxyLayer::new(
        l2_client_args.l2_url,
        l2_auth_jwt,
        builder_args.builder_url,
        builder_auth_jwt,
        metrics,
    ));

    let server = Server::builder()
        .set_http_middleware(service_builder)
        .build(format!("{}:{}", args.rpc_host, args.rpc_port).parse::<SocketAddr>()?)
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

fn init_tracing(endpoint: &str) {
    global::set_text_map_propagator(TraceContextPropagator::new());
    let provider = opentelemetry_otlp::new_pipeline()
        .tracing()
        .with_exporter(
            opentelemetry_otlp::new_exporter()
                .tonic()
                .with_endpoint(endpoint),
        )
        .with_trace_config(Config::default().with_resource(Resource::new(vec![
            opentelemetry::KeyValue::new("service.name", "rollup-boost"),
        ])))
        .install_batch(opentelemetry_sdk::runtime::Tokio);
    match provider {
        Ok(provider) => {
            let _ = global::set_tracer_provider(provider);
        }
        Err(e) => {
            error!(message = "failed to initiate tracing provider", "error" = %e);
        }
    }
}

async fn init_metrics_server(addr: SocketAddr, handle: PrometheusHandle) -> eyre::Result<()> {
    let listener = TcpListener::bind(addr).await?;
    info!("Metrics server running on {}", addr);

    loop {
        match listener.accept().await {
            Ok((stream, _)) => {
                let handle = handle.clone(); // Clone the handle for each connection
                tokio::task::spawn(async move {
                    let service = service_fn(move |_req: Request<hyper::body::Incoming>| {
                        let response = match _req.uri().path() {
                            "/metrics" => Response::builder()
                                .header("content-type", "text/plain")
                                .body(HttpBody::from(handle.render()))
                                .unwrap(),
                            _ => Response::builder()
                                .status(StatusCode::NOT_FOUND)
                                .body(HttpBody::empty())
                                .unwrap(),
                        };
                        async { Ok::<_, hyper::Error>(response) }
                    });

                    let io = TokioIo::new(stream);

                    if let Err(err) = http1::Builder::new().serve_connection(io, service).await {
                        error!(message = "Error serving metrics connection", error = %err);
                    }
                });
            }
            Err(e) => {
                error!(message = "Error accepting connection", error = %e);
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use assert_cmd::Command;
    use http::Uri;
    use jsonrpsee::core::client::ClientT;

    use crate::auth_layer::AuthClientService;
    use crate::server::PayloadSource;
    use alloy_rpc_types_engine::JwtSecret;
    use jsonrpsee::RpcModule;
    use jsonrpsee::http_client::HttpClient;
    use jsonrpsee::http_client::transport::Error as TransportError;
    use jsonrpsee::http_client::transport::HttpBackend;
    use jsonrpsee::{
        core::ClientError,
        rpc_params,
        server::{ServerBuilder, ServerHandle},
    };
    use predicates::prelude::*;
    use reth_rpc_layer::{AuthLayer, JwtAuthValidator};
    use std::result::Result;
    use std::str::FromStr;

    use super::*;

    const AUTH_PORT: u32 = 8550;
    const AUTH_ADDR: &str = "0.0.0.0";
    const SECRET: &str = "f79ae8046bc11c9927afe911db7143c51a806c4a537cc08e0d37140b0192f430";

    #[test]
    fn test_invalid_args() {
        let mut cmd = Command::cargo_bin("rollup-boost").unwrap();
        cmd.arg("--invalid-arg");

        cmd.assert().failure().stderr(predicate::str::contains(
            "error: unexpected argument '--invalid-arg' found",
        ));
    }

    #[tokio::test]
    async fn test_create_client() {
        valid_jwt().await;
        invalid_jwt().await;
    }

    async fn valid_jwt() {
        let secret = JwtSecret::from_hex(SECRET).unwrap();

        let auth_rpc = Uri::from_str(&format!("http://{}:{}", AUTH_ADDR, AUTH_PORT)).unwrap();
        let client = ExecutionClient::new(auth_rpc, secret, 1000, None, PayloadSource::L2).unwrap();
        let response = send_request(client.auth_client).await;
        assert!(response.is_ok());
        assert_eq!(response.unwrap(), "You are the dark lord");
    }

    async fn invalid_jwt() {
        let secret = JwtSecret::random();
        let auth_rpc = Uri::from_str(&format!("http://{}:{}", AUTH_ADDR, AUTH_PORT)).unwrap();
        let client = ExecutionClient::new(auth_rpc, secret, 1000, None, PayloadSource::L2).unwrap();
        let response = send_request(client.auth_client).await;
        assert!(response.is_err());
        assert!(matches!(
            response.unwrap_err(),
            ClientError::Transport(e)
                if matches!(e.downcast_ref::<TransportError>(), Some(TransportError::Rejected { status_code: 401 }))
        ));
    }

    async fn send_request(
        client: Arc<HttpClient<AuthClientService<HttpBackend>>>,
    ) -> Result<String, ClientError> {
        let server = spawn_server().await;

        let response = client
            .request::<String, _>("greet_melkor", rpc_params![])
            .await;

        server.stop().unwrap();
        server.stopped().await;

        response
    }

    /// Spawn a new RPC server equipped with a `JwtLayer` auth middleware.
    async fn spawn_server() -> ServerHandle {
        let secret = JwtSecret::from_hex(SECRET).unwrap();
        let addr = format!("{AUTH_ADDR}:{AUTH_PORT}");
        let validator = JwtAuthValidator::new(secret);
        let layer = AuthLayer::new(validator);
        let middleware = tower::ServiceBuilder::default().layer(layer);

        // Create a layered server
        let server = ServerBuilder::default()
            .set_http_middleware(middleware)
            .build(addr.parse::<SocketAddr>().unwrap())
            .await
            .unwrap();

        // Create a mock rpc module
        let mut module = RpcModule::new(());
        module
            .register_method("greet_melkor", |_, _, _| "You are the dark lord")
            .unwrap();

        server.start(module)
    }
}

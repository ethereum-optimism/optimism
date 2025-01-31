use std::{net::SocketAddr, path::PathBuf, sync::Arc, time::Duration};

use clap::{arg, ArgGroup, Parser};
use dotenv::dotenv;
use error::Error;
use http::{StatusCode, Uri};
use hyper::service::service_fn;
use hyper::{server::conn::http1, Request, Response};
use hyper_util::rt::TokioIo;
use jsonrpsee::{
    http_client::{transport::HttpBackend, HttpBody, HttpClient, HttpClientBuilder},
    server::Server,
    RpcModule,
};
use metrics::ServerMetrics;
use metrics_exporter_prometheus::{PrometheusBuilder, PrometheusHandle};
use metrics_util::layers::{PrefixLayer, Stack};
use opentelemetry::global;
use opentelemetry_otlp::WithExportConfig;
use opentelemetry_sdk::{propagation::TraceContextPropagator, trace::Config, Resource};
use proxy::ProxyLayer;
use reth_rpc_layer::{AuthClientLayer, AuthClientService, JwtSecret};
use server::{EngineApiServer, EthEngineApi, HttpClientWrapper};
use std::net::AddrParseError;
use tokio::net::TcpListener;
use tokio::signal::unix::{signal as unix_signal, SignalKind};
use tracing::{error, info, Level};
use tracing_subscriber::EnvFilter;

mod error;
mod metrics;
mod proxy;
mod server;

#[derive(Parser, Debug)]
#[clap(author, version, about)]
#[clap(group(ArgGroup::new("jwt").required(true).multiple(false).args(&["jwt_token", "jwt_path"])))]
struct Args {
    /// JWT token for authentication
    #[arg(long, env)]
    jwt_token: Option<String>,

    /// Path to the JWT secret file
    #[arg(long, env)]
    jwt_path: Option<PathBuf>,

    /// JWT token for authentication for the builder
    #[arg(long, env)]
    builder_jwt_token: Option<String>,

    /// Path to the JWT secret file for the builder
    #[arg(long, env)]
    builder_jwt_path: Option<PathBuf>,

    /// URL of the local l2 execution engine
    #[arg(long, env, default_value = "http://localhost:8551")]
    l2_url: String,

    /// URL of the builder execution engine
    #[arg(long, env)]
    builder_url: String,

    /// Use the proposer to sync the builder node
    #[arg(long, env, default_value = "false")]
    boost_sync: bool,

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

    /// Timeout for the builder client calls in milliseconds
    #[arg(long, env, default_value = "200")]
    builder_timeout: u64,

    /// Timeout for the l2 client calls in milliseconds
    #[arg(long, env, default_value = "2000")]
    l2_timeout: u64,
}

type Result<T> = core::result::Result<T, Error>;

#[tokio::main]
async fn main() -> Result<()> {
    // Load .env file
    dotenv().ok();
    let args: Args = Args::parse();

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
            .install()
            .map_err(|e| Error::InitMetrics(e.to_string()))?;

        // Start the metrics server
        let metrics_addr = format!("{}:{}", args.metrics_host, args.metrics_port);
        let addr: SocketAddr = metrics_addr
            .parse()
            .map_err(|e: AddrParseError| Error::InitMetrics(e.to_string()))?;
        tokio::spawn(init_metrics_server(addr, handle)); // Run the metrics server in a separate task

        Some(Arc::new(ServerMetrics::default()))
    } else {
        None
    };

    // Telemetry setup
    if args.tracing {
        init_tracing(&args.otlp_endpoint);
    }

    // Handle JWT secret
    let jwt_secret = match (args.jwt_path, args.jwt_token) {
        (Some(file), None) => {
            // Read JWT secret from file
            JwtSecret::from_file(&file).map_err(|e| Error::InvalidArgs(e.to_string()))?
        }
        (None, Some(secret)) => {
            // Use the provided JWT secret
            JwtSecret::from_hex(secret).map_err(|e| Error::InvalidArgs(e.to_string()))?
        }
        _ => {
            // This case should not happen due to ArgGroup
            return Err(Error::InvalidArgs(
                "Either jwt_file or jwt_secret must be provided".into(),
            ));
        }
    };

    let builder_jwt_secret = match (args.builder_jwt_path, args.builder_jwt_token) {
        (Some(file), None) => {
            JwtSecret::from_file(&file).map_err(|e| Error::InvalidArgs(e.to_string()))?
        }
        (None, Some(secret)) => {
            JwtSecret::from_hex(secret).map_err(|e| Error::InvalidArgs(e.to_string()))?
        }
        _ => jwt_secret,
    };

    // Initialize the l2 client
    let l2_client = create_client(&args.l2_url, jwt_secret, args.l2_timeout)?;

    // Initialize the builder client
    let builder_client =
        create_client(&args.builder_url, builder_jwt_secret, args.builder_timeout)?;

    // Construct the RPC module
    let eth_engine_api = EthEngineApi::new(
        Arc::new(l2_client),
        Arc::new(builder_client),
        args.boost_sync,
        metrics,
    );
    let mut module: RpcModule<()> = RpcModule::new(());
    module
        .merge(eth_engine_api.into_rpc())
        .map_err(|e| Error::InitRPCServer(e.to_string()))?;

    // Build and start the server
    info!("Starting server on :{}", args.rpc_port);
    let service_builder = tower::ServiceBuilder::new().layer(ProxyLayer::new(
        args.l2_url
            .parse::<Uri>()
            .map_err(|e| Error::InvalidArgs(e.to_string()))?,
    ));

    let server = Server::builder()
        .set_http_middleware(service_builder)
        .build(
            format!("{}:{}", args.rpc_host, args.rpc_port)
                .parse::<SocketAddr>()
                .map_err(|e| Error::InitRPCServer(e.to_string()))?,
        )
        .await
        .map_err(|e| Error::InitRPCServer(e.to_string()))?;

    let handle = server.start(module);
    let stop_handle = handle.clone();

    // Capture SIGINT and SIGTERM
    let mut sigint =
        unix_signal(SignalKind::interrupt()).map_err(|e| Error::InitRPCServer(e.to_string()))?;
    let mut sigterm =
        unix_signal(SignalKind::terminate()).map_err(|e| Error::InitRPCServer(e.to_string()))?;

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

fn create_client(
    url: &str,
    jwt_secret: JwtSecret,
    timeout: u64,
) -> Result<HttpClientWrapper<HttpClient<AuthClientService<HttpBackend>>>> {
    // Create a middleware that adds a new JWT token to every request.
    let auth_layer = AuthClientLayer::new(jwt_secret);
    let client_middleware = tower::ServiceBuilder::new().layer(auth_layer);

    let client = HttpClientBuilder::new()
        .set_http_middleware(client_middleware)
        .request_timeout(Duration::from_millis(timeout))
        .build(url)
        .map_err(|e| Error::InitRPCClient(e.to_string()))?;

    Ok(HttpClientWrapper::new(client, url.to_string()))
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

async fn init_metrics_server(addr: SocketAddr, handle: PrometheusHandle) -> Result<()> {
    let listener = TcpListener::bind(addr)
        .await
        .map_err(|e| Error::InitMetrics(e.to_string()))?;
    info!("Metrics server running on {}", addr);

    loop {
        match listener.accept().await {
            Ok((stream, _)) => {
                let handle = handle.clone(); // Clone the handle for each connection
                tokio::task::spawn(async move {
                    let service = service_fn(move |_req: Request<hyper::body::Incoming>| {
                        let response = match _req.uri().path() {
                            "/metrics" => Response::new(HttpBody::from(handle.render())),
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
    use jsonrpsee::core::client::ClientT;
    use jsonrpsee::http_client::transport::Error as TransportError;
    use jsonrpsee::{
        core::ClientError,
        rpc_params,
        server::{ServerBuilder, ServerHandle},
    };
    use predicates::prelude::*;
    use reth_rpc_layer::{AuthLayer, JwtAuthValidator};
    use std::result::Result;

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
        let url = format!("http://{}:{}", AUTH_ADDR, AUTH_PORT);
        let client = create_client(url.as_str(), secret, 2000);
        let response = send_request(client.unwrap().client).await;
        assert!(response.is_ok());
        assert_eq!(response.unwrap(), "You are the dark lord");
    }

    async fn invalid_jwt() {
        let secret = JwtSecret::random();
        let url = format!("http://{}:{}", AUTH_ADDR, AUTH_PORT);
        let client = create_client(url.as_str(), secret, 2000);
        let response = send_request(client.unwrap().client).await;
        assert!(response.is_err());
        assert!(matches!(
            response.unwrap_err(),
            ClientError::Transport(e)
                if matches!(e.downcast_ref::<TransportError>(), Some(TransportError::Rejected { status_code: 401 }))
        ));
    }

    async fn send_request(
        client: HttpClient<AuthClientService<HttpBackend>>,
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

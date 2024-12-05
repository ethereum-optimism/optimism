use clap::{arg, ArgGroup, Parser};
use dotenv::dotenv;
use error::Error;
use http::Uri;
use jsonrpsee::http_client::transport::HttpBackend;
use jsonrpsee::http_client::{HttpClient, HttpClientBuilder};
use jsonrpsee::server::Server;
use jsonrpsee::RpcModule;
use metrics::ServerMetrics;
use metrics_exporter_prometheus::PrometheusBuilder;
use metrics_util::layers::{PrefixLayer, Stack};
use opentelemetry::global;
use opentelemetry_otlp::WithExportConfig;
use opentelemetry_sdk::propagation::TraceContextPropagator;
use opentelemetry_sdk::trace::Config;
use opentelemetry_sdk::Resource;
use proxy::ProxyLayer;
use reth_rpc_layer::{AuthClientLayer, AuthClientService, JwtSecret};
use server::{EngineApiServer, EthEngineApi, HttpClientWrapper};
use std::sync::Arc;
use std::time::Duration;
use std::{net::SocketAddr, path::PathBuf};
use tracing::error;
use tracing::{info, Level};
use tracing_subscriber::EnvFilter;

mod error;
mod metrics;
mod proxy;
mod server;

#[derive(Parser, Debug)]
#[clap(author, version, about)]
#[clap(group(ArgGroup::new("jwt").multiple(false).args(&["jwt_token", "jwt_path"])))]
struct Args {
    /// JWT token for authentication
    #[arg(
        long,
        env,
        default_value = "688f5d737bad920bdfb2fc2f488d6b6209eebda1dae949a8de91398d932c517a"
    )]
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

    /// OTLP endpoint
    #[arg(long, env, default_value = "http://localhost:4317")]
    otlp_endpoint: String,

    /// Log level
    #[arg(long, env, default_value = "info")]
    log_level: Level,

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
    tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::new(args.log_level.to_string())) // Set the log level
        .with_ansi(false) // Disable colored logging
        .init();

    let (metrics, handler) = if args.metrics {
        let recorder = PrometheusBuilder::new().build_recorder();
        let handle = recorder.handle();

        // Build metrics stack
        Stack::new(recorder)
            .push(PrefixLayer::new("rollup-boost"))
            .install()
            .map_err(|e| Error::InitMetrics(e.to_string()))?;

        (Some(Arc::new(ServerMetrics::default())), Some(handle))
    } else {
        (None, None)
    };

    // telemetry setup
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

    // server setup
    info!("Starting server on :{}", args.rpc_port);
    let service_builder = tower::ServiceBuilder::new().layer(ProxyLayer::new(
        args.l2_url
            .parse::<Uri>()
            .map_err(|e| Error::InvalidArgs(e.to_string()))?,
        handler,
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
    handle.stopped().await;

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

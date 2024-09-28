use alloy_rpc_types_engine::JwtSecret;
use clap::{arg, ArgGroup, Parser};
use dotenv::dotenv;
use error::Error;
use http::Uri;
use jsonrpsee::http_client::transport::HttpBackend;
use jsonrpsee::http_client::{HttpClient, HttpClientBuilder};
use jsonrpsee::server::Server;
use jsonrpsee::RpcModule;
use proxy::ProxyLayer;
use reth_rpc_layer::{AuthClientLayer, AuthClientService};
use server::{EngineApiServer, EthEngineApi};
use std::sync::Arc;
use std::{net::SocketAddr, path::PathBuf};
use tracing::{info, Level};
use tracing_subscriber::EnvFilter;

mod error;
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

    /// URL of the local l2 execution engine
    #[arg(long, env)]
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

    /// Log level
    #[arg(long, env, default_value = "info")]
    log_level: Level,
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
        .init();

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

    // Initialize the l2 client
    let l2_client = create_client(&args.l2_url, jwt_secret)?;

    // Initialize the builder client
    let builder_client = create_client(&args.builder_url, jwt_secret)?;

    let eth_engine_api = EthEngineApi::new(
        Arc::new(l2_client),
        Arc::new(builder_client),
        args.boost_sync,
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
) -> Result<HttpClient<AuthClientService<HttpBackend>>> {
    // Create a middleware that adds a new JWT token to every request.
    let auth_layer = AuthClientLayer::new(jwt_secret);
    let client_middleware = tower::ServiceBuilder::new().layer(auth_layer);

    HttpClientBuilder::new()
        .set_http_middleware(client_middleware)
        .build(url)
        .map_err(|e| Error::InitRPCClient(e.to_string()))
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

    const AUTH_PORT: u32 = 8551;
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
        let client = create_client(url.as_str(), secret);
        let response = send_request(client.unwrap()).await;
        assert!(response.is_ok());
        assert_eq!(response.unwrap(), "You are the dark lord");
    }

    async fn invalid_jwt() {
        let secret = JwtSecret::random();
        let url = format!("http://{}:{}", AUTH_ADDR, AUTH_PORT);
        let client = create_client(url.as_str(), secret);
        let response = send_request(client.unwrap()).await;
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

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

    // Initialize logging
    tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::new(args.log_level.to_string())) // Set the log level
        .init();

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

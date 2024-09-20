use std::net::SocketAddr;
use std::sync::Arc;

use clap::{arg, Parser};
use error::Error;
use http::HeaderMap;
use http::{header::AUTHORIZATION, HeaderValue};
use jsonrpsee::http_client::HttpClientBuilder;
use jsonrpsee::server::Server;
use jsonrpsee::RpcModule;
use server::{EthApiServer, EthEngineApi};
use tracing::{info, Level};
use tracing_subscriber::EnvFilter;

mod error;
mod server;

#[derive(Parser, Debug)]
#[clap(author, version, about)]
struct Args {
    /// JWT token to authenticate communication between the consensus and execution clients
    #[arg(long, env)]
    jwt_token: String,

    /// URL of the local l2 execution engine
    #[arg(long, env)]
    l2_url: String,

    /// URL of the builder execution engine
    #[arg(long, env)]
    builder_url: String,

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
    let args: Args = Args::parse();

    // Initialize logging
    tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::new(args.log_level.to_string())) // Set the log level
        .init();

    // Initialize the local execution engine client
    let mut headers = HeaderMap::new();
    headers.insert(
        AUTHORIZATION,
        match HeaderValue::try_from(format!("Bearer {}", args.jwt_token)) {
            Ok(token) => token,
            Err(e) => return Err(Error::InvalidJWTToken(e.to_string())),
        },
    );
    let l2_client = HttpClientBuilder::new()
        .set_headers(headers.clone())
        .build(args.l2_url)
        .unwrap();

    let builder_client = HttpClientBuilder::new()
        .set_headers(headers)
        .build(args.builder_url)
        .unwrap();

    let eth_engine_api = EthEngineApi::new(Arc::new(l2_client), Arc::new(builder_client));
    let mut module = RpcModule::new(());
    module
        .merge(eth_engine_api.into_rpc())
        .map_err(|e| Error::InitRPCServerError(e.to_string()))?;

    // server setup
    info!("Starting server on :{}", args.rpc_port);
    let server = Server::builder()
        .build(
            format!("{}:{}", args.rpc_host, args.rpc_port)
                .parse::<SocketAddr>()
                .map_err(|e| Error::InitRPCServerError(e.to_string()))?,
        )
        .await
        .map_err(|e| Error::InitRPCServerError(e.to_string()))?;
    let handle = server.start(module);
    tokio::spawn(handle.stopped());

    Ok(())
}

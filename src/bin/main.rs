#![allow(clippy::complexity)]
use ::tracing::info;
use clap::Parser;
use rollup_boost::{
    Args, Commands, DebugClient, DebugCommands, PayloadSource, ProxyLayer, RollupBoostServer,
    RpcClient, init_metrics, init_tracing,
};
use std::net::SocketAddr;

use alloy_rpc_types_engine::JwtSecret;
use dotenv::dotenv;
use eyre::bail;
use jsonrpsee::RpcModule;
use jsonrpsee::server::Server;

use tokio::signal::unix::{SignalKind, signal as unix_signal};

#[tokio::main]
async fn main() -> eyre::Result<()> {
    // Load .env file
    dotenv().ok();
    rustls::crypto::ring::default_provider()
        .install_default()
        .expect("Failed to install TLS ring CryptoProvider");
    let args: Args = Args::parse();

    let debug_addr = format!("{}:{}", args.debug_host, args.debug_server_port);

    // Handle commands if present
    if let Some(cmd) = args.command {
        let debug_addr = format!("http://{}", debug_addr);
        return match cmd {
            Commands::Debug { command } => match command {
                DebugCommands::SetExecutionMode { execution_mode } => {
                    let client = DebugClient::new(debug_addr.as_str())?;
                    let result = client.set_execution_mode(execution_mode).await.unwrap();
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

    init_tracing(&args)?;
    init_metrics(&args)?;

    let l2_client_args = args.l2_client;

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

    let builder_args = args.builder;
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

    let boost_sync_enabled = !args.no_boost_sync;
    if boost_sync_enabled {
        info!("Boost sync enabled");
    }

    let rollup_boost = RollupBoostServer::new(
        l2_client,
        builder_client,
        boost_sync_enabled,
        args.execution_mode,
    );

    // Spawn the debug server
    rollup_boost.start_debug_server(debug_addr.as_str()).await?;

    let module: RpcModule<()> = rollup_boost.try_into()?;

    // Build and start the server
    info!("Starting server on :{}", args.rpc_port);

    let http_middleware = tower::ServiceBuilder::new().layer(ProxyLayer::new(
        l2_client_args.l2_url,
        l2_auth_jwt,
        builder_args.builder_url,
        builder_auth_jwt,
    ));

    let server = Server::builder()
        .set_http_middleware(http_middleware)
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

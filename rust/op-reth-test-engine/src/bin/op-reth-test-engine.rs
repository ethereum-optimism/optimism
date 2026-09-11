//! Unix-socket JSON-RPC server exposing one [`TestEngine`] over the `engine_*`, `eth_*`, and
//! `optest_*` namespaces (reth-ipc, go-ethereum `rpc.DialIPC`-compatible).
//!
//! The engine is configured entirely from its genesis (the op-geth `core.Genesis` JSON that the Go
//! action tests already build): the fork schedule, chain id, and predeploy state all come from
//! there. Logging goes to stderr, which the Go harness drains.

use std::sync::{Arc, Mutex};

use alloy_genesis::Genesis;
use clap::Parser;
use op_reth_test_engine::{TestEngine, rpc::build_module};

#[derive(Parser, Debug)]
#[command(about = "Ephemeral OP execution engine for op-e2e/actions parity tests")]
struct Args {
    /// Unix socket path to listen on.
    #[arg(long)]
    socket: String,
    /// Path to the op-geth `core.Genesis` JSON describing the chain (fork schedule, chain id,
    /// allocation).
    #[arg(long)]
    genesis: String,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    tracing_subscriber::fmt()
        .with_writer(std::io::stderr)
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| tracing_subscriber::EnvFilter::new("info")),
        )
        .init();

    let args = Args::parse();

    let genesis_json = std::fs::read_to_string(&args.genesis)
        .map_err(|e| format!("read genesis {}: {e}", args.genesis))?;
    let genesis: Genesis =
        serde_json::from_str(&genesis_json).map_err(|e| format!("parse genesis JSON: {e}"))?;
    let engine = TestEngine::new(genesis).map_err(|e| format!("init engine: {e}"))?;
    let module = build_module(Arc::new(Mutex::new(engine)));

    // Fresh socket path each run (Go passes a unique tmp path); remove any stale file.
    let _ = std::fs::remove_file(&args.socket);

    let server = reth_ipc::server::Builder::default().build(args.socket.clone());
    let handle = server.start(module).await?;
    tracing::info!(socket = %args.socket, "op-reth-test-engine listening");
    // A readiness line the Go harness can wait on (in addition to the socket appearing).
    eprintln!("op-reth-test-engine: ready on {}", args.socket);

    tokio::select! {
        () = handle.clone().stopped() => {}
        result = tokio::signal::ctrl_c() => {
            result?;
        }
    }
    Ok(())
}

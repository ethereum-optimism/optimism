//! UDS JSON-RPC server exposing one `ScriptHost` over the `script` namespace.
//! Context (chain-id, artifacts dir, flags) comes from CLI args (the #20415 decision:
//! configure via flags, not an init RPC).

use clap::Parser;
use op_script_engine::host::HostConfig;
use op_script_engine::rpc::{Engine, build_module};

#[derive(Parser, Debug)]
#[command(about = "Rust forge-script engine (op-geth decoupling spike)")]
struct Args {
    /// Unix socket path to listen on.
    #[arg(long)]
    socket: String,
    /// EVM chain id.
    #[arg(long, default_value_t = 1337)]
    chain_id: u64,
    /// forge artifacts directory (the `out/` dir).
    #[arg(long)]
    artifacts: Option<String>,
    /// Route CREATE2 broadcasts through the deterministic deployer.
    #[arg(long, default_value_t = false)]
    create2_deployer: bool,
    /// Disable the max contract code-size check.
    #[arg(long, default_value_t = false)]
    no_max_code_size: bool,
    /// Block number for the EVM block environment.
    #[arg(long, default_value_t = 0)]
    block_num: u64,
    /// Block timestamp for the EVM block environment.
    #[arg(long, default_value_t = 0)]
    timestamp: u64,
    /// Block prev-randao (mix hash) for the EVM block environment, 0x-prefixed 32-byte hex.
    #[arg(long)]
    prev_randao: Option<String>,
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

    // Fresh socket path each run (Go passes a unique tmp path); remove any stale file.
    let _ = std::fs::remove_file(&args.socket);

    let prev_randao = match &args.prev_randao {
        Some(s) => s.parse::<alloy_primitives::B256>()?,
        None => alloy_primitives::B256::ZERO,
    };
    let engine = Engine::spawn(HostConfig {
        chain_id: args.chain_id,
        no_max_code_size: args.no_max_code_size,
        use_create2_deployer: args.create2_deployer,
        artifacts_dir: args.artifacts.map(Into::into),
        block_num: args.block_num,
        timestamp: args.timestamp,
        prev_randao,
        // Fork mode bridges async RPC reads to revm's sync DB via this multi-thread runtime handle.
        runtime_handle: Some(tokio::runtime::Handle::current()),
    });
    let module = build_module(engine);

    let server = reth_ipc::server::Builder::default().build(args.socket.clone());
    let handle = server.start(module).await?;
    tracing::info!(socket = %args.socket, "op-script-engine listening");
    // Signal readiness on a line the Go harness can wait for if it wants to.
    eprintln!("op-script-engine: ready on {}", args.socket);

    tokio::select! {
        _ = handle.clone().stopped() => {}
        _ = tokio::signal::ctrl_c() => {}
    }
    Ok(())
}

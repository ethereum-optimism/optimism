//! CLI entrypoint for the kona-sp1 super-range executor.

use std::{path::PathBuf, process::ExitCode};

use alloy_primitives::B256;
use clap::Parser;
use kona_sp1_super_range_executor::{
    EXIT_INFRA, EXIT_INVALID, EXIT_VALID, RunConfig, Verdict, run,
};

/// Runs the kona-sp1 super-range program against live supernode data.
#[derive(Parser, Debug)]
#[command(
    name = "kona-sp1-super-range-executor",
    about = "Collect a super-range witness from the interop preimage server and execute the \
             kona-sp1 super-range ELF."
)]
struct Cli {
    /// Supernode JSON-RPC endpoint.
    #[arg(long, env = "SUPER_NODE_ADDRESS")]
    supernode_address: String,
    /// L1 execution JSON-RPC endpoint.
    #[arg(long, env)]
    l1_node_address: String,
    /// L1 beacon API endpoint.
    #[arg(long, env)]
    l1_beacon_address: String,
    /// L2 execution JSON-RPC endpoints, comma-separated.
    #[arg(long, value_delimiter = ',', env)]
    l2_node_addresses: Vec<String>,
    /// Trusted L1 head hash.
    #[arg(long, env)]
    l1_head: B256,
    /// Superchain timestamp to prove.
    #[arg(long, env)]
    end_timestamp: u64,
    /// Rollup config JSON paths, comma-separated.
    #[arg(long, alias = "rollup-cfgs", value_delimiter = ',', env)]
    rollup_config_paths: Option<Vec<PathBuf>>,
    /// L1 config JSON path.
    #[arg(long, alias = "l1-cfg", env)]
    l1_config_path: Option<PathBuf>,
    /// Dependency-set JSON path.
    #[arg(long, alias = "depset-cfg", env)]
    dependency_set_path: Option<PathBuf>,
}

fn main() -> ExitCode {
    init_tracing();
    let cli = Cli::parse();

    let runtime = match tokio::runtime::Builder::new_multi_thread().enable_all().build() {
        Ok(runtime) => runtime,
        Err(err) => return infra(format!("failed to build tokio runtime: {err}")),
    };

    let config = RunConfig {
        supernode_address: cli.supernode_address,
        l1_node_address: cli.l1_node_address,
        l1_beacon_address: cli.l1_beacon_address,
        l2_node_addresses: cli.l2_node_addresses,
        l1_head: cli.l1_head,
        end_timestamp: cli.end_timestamp,
        rollup_config_paths: cli.rollup_config_paths,
        l1_config_path: cli.l1_config_path,
        dependency_set_path: cli.dependency_set_path,
    };

    match runtime.block_on(run(config)) {
        Ok(Verdict::Valid) => ExitCode::from(EXIT_VALID),
        Ok(Verdict::Invalid) => ExitCode::from(EXIT_INVALID),
        Err(err) => infra(format!("failed to evaluate super-range claim: {err:?}")),
    }
}

fn infra(message: String) -> ExitCode {
    tracing::error!("{message}");
    eprintln!("{message}");
    ExitCode::from(EXIT_INFRA)
}

fn init_tracing() {
    let _ = tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env().unwrap_or_else(|_| "info".into()),
        )
        .try_init();
}

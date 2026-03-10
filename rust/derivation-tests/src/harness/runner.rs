//! Implementation runners for op-program and kona-host.

use alloy_primitives::B256;
use std::process::ExitStatus;

use super::test::DerivationTest;
use crate::server::TestServers;

/// Configuration for running a derivation implementation.
#[derive(Debug, Clone)]
pub struct RunConfig {
    /// L1 RPC URL.
    pub l1_rpc: String,
    /// L2 RPC URL.
    pub l2_rpc: String,
    /// Beacon API URL.
    pub beacon_url: String,
    /// Expected claim (super root or output root).
    pub expected_claim: B256,
    /// L1 head block hash.
    pub l1_head: B256,
    /// L2 head block hash.
    pub l2_head: B256,
    /// L2 output root at the starting block.
    pub l2_output_root: B256,
    /// L2 block number to derive to.
    pub l2_block_number: u64,
}

/// Build a `RunConfig` from a completed `DerivationTest` and its servers.
pub fn run_config_from_test(test: &DerivationTest, servers: &TestServers) -> RunConfig {
    RunConfig {
        l1_rpc: servers.l1_rpc_url().to_string(),
        l2_rpc: servers.l2_rpc_url().to_string(),
        beacon_url: servers.beacon_url().to_string(),
        expected_claim: test.expected_super_root(),
        l1_head: test.l1.head().header.hash(),
        l2_head: test.l2.head().header.hash(),
        l2_output_root: test.expected_output_root(),
        l2_block_number: test.l2.head().header.inner().number,
    }
}

/// Run op-program against the test configuration.
///
/// Requires `OP_PROGRAM_PATH` environment variable.
#[allow(dead_code)]
pub(super) async fn run_op_program(config: &RunConfig) -> Result<ExitStatus, Box<dyn std::error::Error>> {
    let program_path =
        std::env::var("OP_PROGRAM_PATH").map_err(|_| "OP_PROGRAM_PATH not set")?;

    let status = tokio::process::Command::new(&program_path)
        .arg("--l1")
        .arg(&config.l1_rpc)
        .arg("--l2")
        .arg(&config.l2_rpc)
        .arg("--l1.beacon")
        .arg(&config.beacon_url)
        .arg("--l1.head")
        .arg(format!("{:?}", config.l1_head))
        .arg("--l2.head")
        .arg(format!("{:?}", config.l2_head))
        .arg("--l2.outputroot")
        .arg(format!("{:?}", config.l2_output_root))
        .arg("--l2.claim")
        .arg(format!("{:?}", config.expected_claim))
        .arg("--l2.blocknumber")
        .arg(config.l2_block_number.to_string())
        .arg("--l1.rpckind=debug_geth")
        .status()
        .await?;

    Ok(status)
}

/// Run kona-host against the test configuration.
///
/// Requires `KONA_HOST_PATH` environment variable.
#[allow(dead_code)]
pub(super) async fn run_kona_host(
    config: &RunConfig,
    rollup_config: &kona_genesis::RollupConfig,
) -> Result<ExitStatus, Box<dyn std::error::Error>> {
    let host_path = std::env::var("KONA_HOST_PATH").map_err(|_| "KONA_HOST_PATH not set")?;

    // Write rollup config to temp file
    let rollup_config_file = tempfile::NamedTempFile::new()?;
    serde_json::to_writer(&rollup_config_file, rollup_config)?;

    let status = tokio::process::Command::new(&host_path)
        .arg("--l1-node-address")
        .arg(&config.l1_rpc)
        .arg("--l2-node-address")
        .arg(&config.l2_rpc)
        .arg("--beacon-address")
        .arg(&config.beacon_url)
        .arg("--l1-head")
        .arg(format!("{:?}", config.l1_head))
        .arg("--l2-head")
        .arg(format!("{:?}", config.l2_head))
        .arg("--l2-output-root")
        .arg(format!("{:?}", config.l2_output_root))
        .arg("--l2-claim")
        .arg(format!("{:?}", config.expected_claim))
        .arg("--l2-block-number")
        .arg(config.l2_block_number.to_string())
        .arg("--rollup-config-path")
        .arg(rollup_config_file.path())
        .status()
        .await?;

    Ok(status)
}

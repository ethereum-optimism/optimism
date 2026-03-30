//! Implementation runners for op-program and kona-host.

use alloy_primitives::B256;
use std::process::ExitStatus;

use super::test::DerivationTest;
use crate::{config::DeterministicConfig, l2::L2BlockRef, server::TestServers};

/// Configuration for running a derivation implementation.
#[derive(Debug, Clone)]
pub struct RunConfig {
    /// L1 RPC URL.
    pub l1_rpc: String,
    /// L2 RPC URL.
    pub l2_rpc: String,
    /// Beacon API URL.
    pub beacon_url: String,
    /// Expected claim (super root or output root) at the target block.
    pub expected_claim: B256,
    /// L1 head block hash.
    pub l1_head: B256,
    /// L2 agreed safe head block hash (the starting point, before the target).
    pub l2_head: B256,
    /// L2 output root at the agreed safe head.
    pub l2_output_root: B256,
    /// L2 block number to derive to (the target).
    pub l2_block_number: u64,
}

/// Build a `RunConfig` from a completed `DerivationTest` and its servers.
///
/// Uses the L2 genesis as the agreed starting point and the L2 head as the
/// target block to prove.
pub fn run_config_from_test(test: &DerivationTest, servers: &TestServers) -> RunConfig {
    let genesis_ref = L2BlockRef { index: 0 };
    let head = test.l2.head();

    let expected_claim = test.expected_output_root();

    RunConfig {
        l1_rpc: servers.l1_rpc_url().to_string(),
        l2_rpc: servers.l2_rpc_url().to_string(),
        beacon_url: servers.beacon_url().to_string(),
        // Agreed state: genesis
        l2_head: test.l2.block(genesis_ref).header.hash(),
        l2_output_root: test.output_root_at(genesis_ref),
        // Target/claimed state: head
        l2_block_number: head.header.inner().number,
        expected_claim,
        // L1 head: latest L1 block
        l1_head: test.l1.head().header.hash(),
    }
}

/// Run op-program against the test configuration.
///
/// Requires `OP_PROGRAM_PATH` environment variable.
/// Writes the rollup config, L1 genesis, and L2 genesis from the generated files
/// to temp files and passes them via command-line flags.
#[allow(dead_code)]
pub async fn run_op_program(
    run_config: &RunConfig,
    config: &DeterministicConfig,
) -> Result<ExitStatus, Box<dyn std::error::Error>> {
    let program_path = std::env::var("OP_PROGRAM_PATH").map_err(|_| "OP_PROGRAM_PATH not set")?;

    // Write rollup config to temp file (already in Go-compatible format from op-deployer)
    let rollup_config_file = tempfile::NamedTempFile::new()?;
    std::io::Write::write_all(&mut &rollup_config_file, &config.rollup_json)?;

    // Write L1 genesis to temp file (already in Go genesis.json format from op-deployer)
    let l1_config_file = tempfile::NamedTempFile::new()?;
    std::io::Write::write_all(&mut &l1_config_file, &config.l1_genesis_json)?;

    // Write L2 genesis to temp file (already in Go genesis.json format from op-deployer)
    let l2_genesis_file = tempfile::NamedTempFile::new()?;
    std::io::Write::write_all(&mut &l2_genesis_file, &config.l2_genesis_json)?;

    let child = tokio::process::Command::new(&program_path)
        .arg("--l1")
        .arg(&run_config.l1_rpc)
        .arg("--l2")
        .arg(&run_config.l2_rpc)
        .arg("--l1.beacon")
        .arg(&run_config.beacon_url)
        .arg("--l1.head")
        .arg(format!("{:?}", run_config.l1_head))
        .arg("--l2.head")
        .arg(format!("{:?}", run_config.l2_head))
        .arg("--l2.outputroot")
        .arg(format!("{:?}", run_config.l2_output_root))
        .arg("--l2.claim")
        .arg(format!("{:?}", run_config.expected_claim))
        .arg("--l2.blocknumber")
        .arg(run_config.l2_block_number.to_string())
        .arg("--l1.rpckind=debug_geth")
        .arg("--l2.custom")
        .arg("--rollup.config")
        .arg(rollup_config_file.path())
        .arg("--l1.chainconfig")
        .arg(l1_config_file.path())
        .arg("--l2.genesis")
        .arg(l2_genesis_file.path())
        .arg("--log.level=DEBUG")
        .stdout(std::process::Stdio::piped())
        .stderr(std::process::Stdio::piped())
        .output();

    // Give op-program a timeout to avoid hanging indefinitely
    let output = tokio::time::timeout(std::time::Duration::from_secs(120), child)
        .await
        .map_err(|_| "op-program timed out after 120 seconds")??;

    let stderr = String::from_utf8_lossy(&output.stderr);
    let stdout = String::from_utf8_lossy(&output.stdout);
    let combined = format!("{stdout}{stderr}");

    // Print op-program output for debugging
    eprintln!("--- op-program output ---");
    for line in combined.lines() {
        if line.contains("lvl=info") ||
            line.contains("lvl=warn") ||
            line.contains("lvl=crit") ||
            line.contains("lvl=error") ||
            line.contains("panic")
        {
            eprintln!("{line}");
        }
    }
    eprintln!("--- end op-program output ---");

    Ok(output.status)
}

/// Run kona-host against the test configuration.
///
/// Requires `KONA_HOST_PATH` environment variable.
/// Uses the `single` subcommand with `--native` to run the client in-process.
#[allow(dead_code)]
pub async fn run_kona_host(
    run_config: &RunConfig,
    config: &DeterministicConfig,
) -> Result<ExitStatus, Box<dyn std::error::Error>> {
    let host_path = std::env::var("KONA_HOST_PATH").map_err(|_| "KONA_HOST_PATH not set")?;

    // Write rollup config to temp file
    let rollup_config_file = tempfile::NamedTempFile::new()?;
    std::io::Write::write_all(&mut &rollup_config_file, &config.rollup_json)?;

    // Write L1 chain config to temp file
    let l1_config_file = tempfile::NamedTempFile::new()?;
    let l1_chain_config = config.l1_chain_config();
    serde_json::to_writer(&l1_config_file, &l1_chain_config)?;

    let status = tokio::process::Command::new(&host_path)
        .arg("single")
        .arg("--native")
        .arg("--l1-node-address")
        .arg(&run_config.l1_rpc)
        .arg("--l2-node-address")
        .arg(&run_config.l2_rpc)
        .arg("--l1-beacon-address")
        .arg(&run_config.beacon_url)
        .arg("--l1-head")
        .arg(format!("{:?}", run_config.l1_head))
        .arg("--agreed-l2-head-hash")
        .arg(format!("{:?}", run_config.l2_head))
        .arg("--agreed-l2-output-root")
        .arg(format!("{:?}", run_config.l2_output_root))
        .arg("--claimed-l2-output-root")
        .arg(format!("{:?}", run_config.expected_claim))
        .arg("--claimed-l2-block-number")
        .arg(run_config.l2_block_number.to_string())
        .arg("--rollup-config-path")
        .arg(rollup_config_file.path())
        .arg("--l1-config-path")
        .arg(l1_config_file.path())
        .status()
        .await?;

    Ok(status)
}

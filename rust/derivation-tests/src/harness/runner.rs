//! Implementation runners for op-program and kona-host.

use alloy_primitives::B256;
use std::process::ExitStatus;

use super::test::DerivationTest;
use crate::{l2::L2BlockRef, server::TestServers};

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

    RunConfig {
        l1_rpc: servers.l1_rpc_url().to_string(),
        l2_rpc: servers.l2_rpc_url().to_string(),
        beacon_url: servers.beacon_url().to_string(),
        // Agreed state: genesis
        l2_head: test.l2.block(genesis_ref).header.hash(),
        l2_output_root: test.output_root_at(genesis_ref),
        // Target/claimed state: head
        l2_block_number: head.header.inner().number,
        expected_claim: test.expected_output_root(),
        // L1 head: latest L1 block
        l1_head: test.l1.head().header.hash(),
    }
}

/// Serialize a rollup config to JSON compatible with Go's `op-program` parser.
///
/// Go's rollup config uses `DisallowUnknownFields`, so we must only include
/// fields that Go recognizes. Kona's `RollupConfig` serializes extra fields
/// (e.g. `eip1559Denominator`, `baseFeeScalar`) that Go rejects.
fn go_compatible_rollup_config_json(
    rollup_config: &kona_genesis::RollupConfig,
) -> Result<serde_json::Value, Box<dyn std::error::Error>> {
    let mut val = serde_json::to_value(rollup_config)?;
    // Go only accepts fields declared in op-node/rollup/types.go Config struct.
    // Any extra fields cause `json: unknown field` errors due to DisallowUnknownFields.
    let allowed_top_level: std::collections::HashSet<&str> = [
        "genesis",
        "block_time",
        "max_sequencer_drift",
        "seq_window_size",
        "channel_timeout",
        "l1_chain_id",
        "l2_chain_id",
        "regolith_time",
        "canyon_time",
        "delta_time",
        "ecotone_time",
        "fjord_time",
        "granite_time",
        "holocene_time",
        "isthmus_time",
        "jovian_time",
        "karst_time",
        "interop_time",
        "batch_inbox_address",
        "deposit_contract_address",
        "l1_system_config_address",
        "protocol_versions_address",
        "alt_da",
        "pectra_blob_schedule_time",
    ]
    .into_iter()
    .collect();

    if let Some(obj) = val.as_object_mut() {
        obj.retain(|k, _| allowed_top_level.contains(k.as_str()));
    }

    // Go's SystemConfig only accepts these fields (with DisallowUnknownFields)
    let allowed_sys_config: std::collections::HashSet<&str> = [
        "batcherAddr",
        "overhead",
        "scalar",
        "gasLimit",
        "eip1559Params",
        "operatorFeeParams",
        "minBaseFee",
    ]
    .into_iter()
    .collect();

    if let Some(genesis) = val.get_mut("genesis") &&
        let Some(sys_config) = genesis.get_mut("system_config") &&
        let Some(obj) = sys_config.as_object_mut()
    {
        // Pack eip1559Denominator + eip1559Elasticity into eip1559Params (B64).
        // Go expects a single packed field: [denominator_be32 ++ elasticity_be32].
        let denom = obj.get("eip1559Denominator").and_then(|v| v.as_u64()).unwrap_or(0) as u32;
        let elasticity = obj.get("eip1559Elasticity").and_then(|v| v.as_u64()).unwrap_or(0) as u32;
        if denom != 0 || elasticity != 0 {
            let mut packed = [0u8; 8];
            packed[0..4].copy_from_slice(&denom.to_be_bytes());
            packed[4..8].copy_from_slice(&elasticity.to_be_bytes());
            obj.insert(
                "eip1559Params".to_string(),
                serde_json::json!(format!("0x{}", alloy_primitives::hex::encode(packed))),
            );
        }

        obj.retain(|k, _| allowed_sys_config.contains(k.as_str()));
    }

    Ok(val)
}

/// Build a minimal Go-compatible genesis.json with an L1 chain config.
///
/// Go's `ChainConfig` uses `DisallowUnknownFields`, so we construct the config
/// manually with only the fields go-ethereum recognizes.
fn go_compatible_l1_genesis_json(chain_config: &alloy_genesis::ChainConfig) -> serde_json::Value {
    let mut config = serde_json::Map::new();
    config.insert("chainId".to_string(), serde_json::json!(chain_config.chain_id));

    // Go's IsCancun/IsShanghai require IsLondon(num), which checks LondonBlock != nil.
    // Without LondonBlock set, time-based forks like Cancun are never considered active
    // in latestBlobConfig, causing a nil panic in CalcBlobFee.
    config.insert("londonBlock".to_string(), serde_json::json!(0));

    if let Some(t) = chain_config.shanghai_time {
        config.insert("shanghaiTime".to_string(), serde_json::json!(t));
    }
    if let Some(t) = chain_config.cancun_time {
        config.insert("cancunTime".to_string(), serde_json::json!(t));
    }
    if let Some(t) = chain_config.prague_time {
        config.insert("pragueTime".to_string(), serde_json::json!(t));
    }

    // Go requires a blobSchedule config when Cancun is active.
    // Each active fork needs its own entry with the correct blob parameters.
    if chain_config.cancun_time.is_some() {
        let mut blob_schedule = serde_json::Map::new();
        blob_schedule.insert(
            "cancun".to_string(),
            serde_json::json!({
                "target": 3,
                "max": 6,
                "baseFeeUpdateFraction": 3338477
            }),
        );
        if chain_config.prague_time.is_some() {
            blob_schedule.insert(
                "prague".to_string(),
                serde_json::json!({
                    "target": 6,
                    "max": 9,
                    "baseFeeUpdateFraction": 5225352
                }),
            );
        }
        config.insert("blobSchedule".to_string(), serde_json::Value::Object(blob_schedule));
    }

    serde_json::json!({ "config": config })
}

/// Run op-program against the test configuration.
///
/// Requires `OP_PROGRAM_PATH` environment variable.
/// The rollup config and L1 chain config are written to temp files and passed
/// via `--rollup.config` and `--l1.chainconfig`.
#[allow(dead_code)]
pub async fn run_op_program(
    config: &RunConfig,
    rollup_config: &kona_genesis::RollupConfig,
    l1_chain_config: &alloy_genesis::ChainConfig,
) -> Result<ExitStatus, Box<dyn std::error::Error>> {
    let program_path = std::env::var("OP_PROGRAM_PATH").map_err(|_| "OP_PROGRAM_PATH not set")?;

    // Write rollup config to temp file (Go-compatible format)
    let rollup_config_file = tempfile::NamedTempFile::new()?;
    let go_config = go_compatible_rollup_config_json(rollup_config)?;
    serde_json::to_writer(&rollup_config_file, &go_config)?;

    // Write L1 chain config to temp file, wrapped in genesis.json format.
    // op-program expects {"config": <ChainConfig>, ...} (Go genesis.json format).
    // Go's ChainConfig uses DisallowUnknownFields, so we must only include
    // fields that go-ethereum recognizes.
    let l1_config_file = tempfile::NamedTempFile::new()?;
    let l1_genesis_json = go_compatible_l1_genesis_json(l1_chain_config);
    serde_json::to_writer(&l1_config_file, &l1_genesis_json)?;

    // Write L2 genesis (chain config) to temp file.
    // op-program requires this for the L2 chain ID and hardfork schedule.
    // The L2 chain config must include:
    // - londonBlock (required for time-based fork checks like IsCancun)
    // - Standard EVM time-based forks (shanghaiTime, cancunTime)
    // - OP Stack fork times (ecotoneTime through isthmusTime)
    // - optimism EIP-1559 parameters
    let l2_genesis_file = tempfile::NamedTempFile::new()?;
    let l2_genesis_json = serde_json::json!({
        "config": {
            "chainId": rollup_config.l2_chain_id.id(),
            "londonBlock": 0,
            // EVM hardfork times
            "shanghaiTime": 0,
            "cancunTime": 0,
            "pragueTime": 0,
            // OP Stack fork times (must match rollup config hardforks)
            "regolithTime": 0,
            "canyonTime": 0,
            "ecotoneTime": 0,
            "fjordTime": 0,
            "graniteTime": 0,
            "holoceneTime": 0,
            "isthmusTime": 0,
            "jovianTime": 0,
            "optimism": {
                "eip1559Elasticity": 6,
                "eip1559Denominator": 50,
                "eip1559DenominatorCanyon": 250
            }
        }
    });
    serde_json::to_writer(&l2_genesis_file, &l2_genesis_json)?;

    let child = tokio::process::Command::new(&program_path)
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
            line.contains("DEBUG")
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
    config: &RunConfig,
    rollup_config: &kona_genesis::RollupConfig,
    l1_chain_config: &alloy_genesis::ChainConfig,
) -> Result<ExitStatus, Box<dyn std::error::Error>> {
    let host_path = std::env::var("KONA_HOST_PATH").map_err(|_| "KONA_HOST_PATH not set")?;

    // Write rollup config to temp file
    let rollup_config_file = tempfile::NamedTempFile::new()?;
    serde_json::to_writer(&rollup_config_file, rollup_config)?;

    // Write L1 chain config to temp file
    let l1_config_file = tempfile::NamedTempFile::new()?;
    serde_json::to_writer(&l1_config_file, l1_chain_config)?;

    let status = tokio::process::Command::new(&host_path)
        .arg("single")
        .arg("--native")
        .arg("--l1-node-address")
        .arg(&config.l1_rpc)
        .arg("--l2-node-address")
        .arg(&config.l2_rpc)
        .arg("--l1-beacon-address")
        .arg(&config.beacon_url)
        .arg("--l1-head")
        .arg(format!("{:?}", config.l1_head))
        .arg("--agreed-l2-head-hash")
        .arg(format!("{:?}", config.l2_head))
        .arg("--agreed-l2-output-root")
        .arg(format!("{:?}", config.l2_output_root))
        .arg("--claimed-l2-output-root")
        .arg(format!("{:?}", config.expected_claim))
        .arg("--claimed-l2-block-number")
        .arg(config.l2_block_number.to_string())
        .arg("--rollup-config-path")
        .arg(rollup_config_file.path())
        .arg("--l1-config-path")
        .arg(l1_config_file.path())
        .status()
        .await?;

    Ok(status)
}

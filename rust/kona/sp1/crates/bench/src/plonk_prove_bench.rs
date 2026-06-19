//! Local PLONK aggregation-proof benchmark entrypoint for kona-sp1.

use std::{env, path::PathBuf, time::Instant};

use alloy_primitives::Address;
use anyhow::Context;
use clap::Parser;
use kona_sp1_client_utils::boot::BootInfoStruct;
use kona_sp1_host_utils::{fetcher::OPSuccinctDataFetcher, get_agg_proof_stdin, setup_logger};
use kona_sp1_proof_utils::{get_agg_elf, get_range_elf};
use sp1_sdk::{Elf, ProveRequest, Prover, ProverClient, ProvingKey, SP1ProofWithPublicValues};
use url::Url;

const REQUIRED_RPC_ENV: [&str; 3] = ["L1_RPC", "L2_RPC", "L2_NODE_RPC"];
const OPTIONAL_RPC_ENV: [&str; 1] = ["L1_BEACON_RPC"];

#[derive(Debug, Parser)]
#[command(about = "Produce and verify a PLONK aggregation proof")]
struct Args {
    /// Saved compressed range proofs, comma-separated, in ascending chain order.
    #[arg(long, value_delimiter = ',')]
    proofs: Vec<PathBuf>,

    /// Persist the verified PLONK aggregation proof to this path.
    #[arg(long)]
    save_proof: Option<PathBuf>,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let args = Args::parse();
    setup_logger();

    validate_proofs(&args.proofs)?;

    let range_elf = get_range_elf();
    let agg_elf = get_agg_elf();
    ensure_non_empty_elfs(range_elf, agg_elf)?;
    ensure_required_rpc_env()?;

    // Backend selected by the SP1_PROVER env var (default: cpu).
    let prover = ProverClient::from_env().await;

    let range_proving_key =
        prover.setup(Elf::Static(range_elf)).await.context("failed to set up range proving key")?;
    let range_verifying_key = range_proving_key.verifying_key().clone();
    let agg_proving_key = prover
        .setup(Elf::Static(agg_elf))
        .await
        .context("failed to set up aggregation proving key")?;

    let mut proofs = Vec::with_capacity(args.proofs.len());
    let mut boot_infos = Vec::with_capacity(args.proofs.len());
    for path in &args.proofs {
        let mut proof_with_public_values =
            SP1ProofWithPublicValues::load(path).with_context(|| {
                format!("failed to load compressed range proof from {}", path.display())
            })?;
        // The range program commits the BootInfoStruct via sp1_zkvm::io::commit (matching
        // op-succinct), so read it back through the SP1 public-values codec.
        let boot_info: BootInfoStruct = proof_with_public_values.public_values.read();
        proofs.push(proof_with_public_values.proof);
        boot_infos.push(boot_info);
    }

    let fetcher = OPSuccinctDataFetcher::new_with_rollup_config()
        .await
        .context("failed to initialize RPC data fetcher from environment")?;
    let checkpoint = fetcher
        .get_latest_l1_head_in_batch(&boot_infos)
        .await
        .context("failed to find latest L1 head across range proofs")?;
    let checkpoint_hash = checkpoint.hash_slow();
    let headers = fetcher
        .get_header_preimages(&boot_infos, checkpoint_hash)
        .await
        .context("failed to fetch L1 header preimages for aggregation")?;

    let stdin = get_agg_proof_stdin(
        proofs,
        boot_infos,
        headers,
        &range_verifying_key,
        checkpoint_hash,
        Address::ZERO,
    )
    .context("failed to serialize aggregation stdin")?;

    let execute_start = Instant::now();
    let (_public_values, report) = prover
        .execute(Elf::Static(agg_elf), stdin.clone())
        .calculate_gas(true)
        .deferred_proof_verification(false)
        .await
        .context("failed to execute aggregation program")?;
    tracing::info!(
        "aggregation execute: {} cycles, gas {:?}, {:?}",
        report.total_instruction_count(),
        report.gas(),
        execute_start.elapsed()
    );

    tracing::warn!(
        "PLONK proving runs the full aggregation pipeline (core -> compress -> shrink -> wrap -> \
         gnark). The default local CPU backend uses SP1's gnark Docker wrapper and the first run \
         may download PLONK circuit artifacts to ~/.sp1. SP1_PROVER=cuda sends PLONK proving to \
         sp1-gpu-server; SP1_PROVER=network or hosted uses the SP1 prover network instead."
    );

    let prove_start = Instant::now();
    let proof = prover.prove(&agg_proving_key, stdin).plonk().await.context(
        "failed to produce PLONK aggregation proof (check the selected SP1_PROVER backend; \
             local CPU PLONK requires Docker for gnark unless native gnark is enabled)",
    )?;
    let prove_elapsed = prove_start.elapsed();

    let verify_start = Instant::now();
    prover
        .verify(&proof, agg_proving_key.verifying_key(), None)
        .context("PLONK aggregation proof failed local verification")?;
    let verify_elapsed = verify_start.elapsed();

    let calldata_size = proof.bytes().len();
    tracing::info!(
        "PLONK aggregation prove wall-clock: {:?}; local verify: {:?}; on-chain calldata: {} bytes",
        prove_elapsed,
        verify_elapsed,
        calldata_size
    );

    if let Some(path) = &args.save_proof {
        proof
            .save(path)
            .with_context(|| format!("failed to save PLONK proof to {}", path.display()))?;
        tracing::info!("saved PLONK aggregation proof to {}", path.display());
    }

    Ok(())
}

fn validate_proofs(proofs: &[PathBuf]) -> anyhow::Result<()> {
    anyhow::ensure!(!proofs.is_empty(), "pass at least one --proofs path");
    Ok(())
}

fn ensure_non_empty_elfs(range_elf: &[u8], agg_elf: &[u8]) -> anyhow::Result<()> {
    anyhow::ensure!(
        !range_elf.is_empty() && !agg_elf.is_empty(),
        "range-elf or aggregation-elf is missing or empty. Build real ELFs first with: \
         cd rust/kona/sp1 && just build-elfs"
    );
    Ok(())
}

fn ensure_required_rpc_env() -> anyhow::Result<()> {
    validate_rpc_env_vars(|key| env::var(key).ok())
}

fn validate_rpc_env_vars(get_var: impl Fn(&str) -> Option<String>) -> anyhow::Result<()> {
    let env_values = REQUIRED_RPC_ENV
        .into_iter()
        .chain(OPTIONAL_RPC_ENV)
        .filter_map(|key| get_var(key).map(|value| (key, value)))
        .collect::<Vec<_>>();

    let missing = REQUIRED_RPC_ENV
        .into_iter()
        .filter(|key| {
            env_values.iter().all(|(env_key, value)| env_key != key || value.trim().is_empty())
        })
        .collect::<Vec<_>>();

    anyhow::ensure!(
        missing.is_empty(),
        "missing required RPC environment variable(s): {}. Set L1_RPC, L2_RPC, \
         and L2_NODE_RPC before running the benchmark. Set L1_BEACON_RPC as well \
         for post-Ecotone/blob-backed ranges.",
        missing.join(", ")
    );

    let invalid = env_values
        .iter()
        .filter_map(|(key, value)| {
            let trimmed = value.trim();
            if trimmed.is_empty() {
                return None;
            }

            Url::parse(trimmed).err().map(|err| (*key, err))
        })
        .collect::<Vec<_>>();

    anyhow::ensure!(
        invalid.is_empty(),
        "invalid RPC URL environment variable(s): {}",
        invalid.iter().map(|(key, err)| format!("{key} ({err})")).collect::<Vec<_>>().join(", ")
    );

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn get_var(values: &[(&str, &str)], key: &str) -> Option<String> {
        values.iter().find(|(candidate, _)| *candidate == key).map(|(_, value)| value.to_string())
    }

    #[test]
    fn validate_proofs_rejects_empty_inputs() {
        assert!(validate_proofs(&[PathBuf::from("range.bin")]).is_ok());

        let err = validate_proofs(&[]).unwrap_err();
        assert!(err.to_string().contains("pass at least one --proofs path"));
    }

    #[test]
    fn ensure_non_empty_elfs_rejects_missing_or_empty_elfs() {
        assert!(ensure_non_empty_elfs(&[1], &[1]).is_ok());

        let err = ensure_non_empty_elfs(&[], &[1]).unwrap_err();
        assert!(err.to_string().contains("missing or empty"));

        let err = ensure_non_empty_elfs(&[1], &[]).unwrap_err();
        assert!(err.to_string().contains("missing or empty"));
    }

    #[test]
    fn validate_rpc_env_vars_rejects_missing_required_values() {
        let values = [("L1_RPC", "https://example.com")];
        let err = validate_rpc_env_vars(|key| get_var(&values, key)).unwrap_err();

        let message = err.to_string();
        assert!(message.contains("missing required RPC environment variable(s)"));
        assert!(message.contains("L2_RPC"));
        assert!(message.contains("L2_NODE_RPC"));
    }

    #[test]
    fn validate_rpc_env_vars_rejects_malformed_urls() {
        let values = [
            ("L1_RPC", "https://example.com"),
            ("L2_RPC", "not a url"),
            ("L2_NODE_RPC", "https://example.com"),
            ("L1_BEACON_RPC", "also not a url"),
        ];
        let err = validate_rpc_env_vars(|key| get_var(&values, key)).unwrap_err();

        let message = err.to_string();
        assert!(message.contains("invalid RPC URL environment variable(s)"));
        assert!(message.contains("L2_RPC"));
        assert!(message.contains("L1_BEACON_RPC"));
    }
}

//! Local aggregation-program benchmark entrypoint for kona-sp1.

use std::{
    env,
    path::{Path, PathBuf},
    time::Instant,
};

use alloy_consensus::Header;
use alloy_primitives::{Address, B256};
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
#[command(about = "Aggregate saved kona-sp1 compressed range proofs")]
struct Args {
    /// Saved compressed range proofs, comma-separated, in ascending chain order.
    #[arg(long, value_delimiter = ',')]
    proofs: Vec<PathBuf>,

    /// Also produce a compressed aggregation proof after the execute-only stats pass.
    #[arg(long, default_value_t = false)]
    prove: bool,

    /// Persist the compressed aggregation proof to this path.
    #[arg(long, requires = "prove")]
    save_proof: Option<PathBuf>,

    /// Save the RPC-derived aggregation inputs to this path for later offline proving.
    #[arg(long, conflicts_with = "load_agg_inputs")]
    save_agg_inputs: Option<PathBuf>,

    /// Load RPC-derived aggregation inputs from this path instead of fetching from RPC.
    #[arg(long)]
    load_agg_inputs: Option<PathBuf>,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let args = Args::parse();
    setup_logger();

    validate_proofs(&args.proofs)?;

    let range_elf = get_range_elf();
    let agg_elf = get_agg_elf();
    ensure_non_empty_elfs(range_elf, agg_elf)?;
    if args.load_agg_inputs.is_none() {
        ensure_required_rpc_env()?;
    }

    // Backend selected by the SP1_PROVER env var (default: cpu).
    let prover = ProverClient::from_env().await;

    let range_proving_key =
        prover.setup(Elf::Static(range_elf)).await.context("failed to set up range proving key")?;
    let range_verifying_key = range_proving_key.verifying_key().clone();
    let agg_proving_key = prover
        .setup(Elf::Static(agg_elf))
        .await
        .context("failed to set up aggregation proving key")?;

    // TODO(kona-sp1): exercise this end-to-end to confirm the bincode (1.3.x) serialization
    // round-trips. The guest aggregation program derives each range proof's public-values digest
    // from `bincode::serialize(&BootInfoStruct)`, which must equal Sha256(<bytes the range program
    // committed via sp1_zkvm::io::commit under SP1 v6.2.4>) for `verify_sp1_proof` to succeed
    // during the aggregation proof below. This was reverted from the ABI public_values() scheme to
    // match op-succinct and has not yet been validated on this SP1 version — run `just agg-bench
    // --proofs ... --prove` against real range proofs to confirm the digests match.
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

    let (checkpoint_hash, headers) = if let Some(path) = &args.load_agg_inputs {
        let inputs = load_agg_inputs(path)?;
        tracing::info!("loaded aggregation inputs from {}", path.display());
        inputs
    } else {
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

        if let Some(path) = &args.save_agg_inputs {
            save_agg_inputs(checkpoint_hash, &headers, path)?;
            tracing::info!("saved aggregation inputs to {}", path.display());
        }

        (checkpoint_hash, headers)
    };

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

    if args.prove {
        let prove_start = Instant::now();
        let proof = prover
            .prove(&agg_proving_key, stdin)
            .compressed()
            .await
            .context("failed to produce compressed aggregation proof")?;
        tracing::info!("aggregation compressed prove wall-clock: {:?}", prove_start.elapsed());
        let verify_start = Instant::now();
        prover
            .verify(&proof, agg_proving_key.verifying_key(), None)
            .context("compressed aggregation proof failed local verification")?;
        tracing::info!("local verify: {:?}", verify_start.elapsed());
        if let Some(path) = &args.save_proof {
            proof.save(path).context("failed to save compressed aggregation proof")?;
            tracing::info!("saved compressed aggregation proof to {}", path.display());
        }
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

fn save_agg_inputs(checkpoint_hash: B256, headers: &[Header], path: &Path) -> anyhow::Result<()> {
    let bytes = serde_cbor::to_vec(&(checkpoint_hash, headers))
        .context("failed to serialize aggregation inputs")?;
    std::fs::write(path, bytes)
        .with_context(|| format!("failed to write aggregation inputs to {}", path.display()))?;
    Ok(())
}

fn load_agg_inputs(path: &Path) -> anyhow::Result<(B256, Vec<Header>)> {
    let bytes = std::fs::read(path)
        .with_context(|| format!("failed to read aggregation inputs from {}", path.display()))?;
    serde_cbor::from_slice(&bytes).context("failed to deserialize aggregation inputs")
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
    fn agg_inputs_round_trip() {
        let checkpoint = B256::repeat_byte(7);
        let headers = vec![Header::default(), Header::default()];
        let path = std::env::temp_dir()
            .join(format!("kona-sp1-agg-inputs-roundtrip-{}.cbor", std::process::id()));

        save_agg_inputs(checkpoint, &headers, &path).unwrap();
        let (loaded_hash, loaded_headers) = load_agg_inputs(&path).unwrap();
        assert_eq!(loaded_hash, checkpoint);
        assert_eq!(loaded_headers, headers);

        let _ = std::fs::remove_file(path);
    }

    #[test]
    fn load_agg_inputs_missing_file_errors() {
        let err = load_agg_inputs(Path::new("/nonexistent/kona-sp1-missing.cbor")).unwrap_err();

        assert!(err.to_string().contains("failed to read aggregation inputs"));
    }

    #[test]
    fn save_and_load_agg_inputs_conflict() {
        let err = Args::try_parse_from([
            "agg-bench",
            "--proofs",
            "a.bin",
            "--save-agg-inputs",
            "s.cbor",
            "--load-agg-inputs",
            "l.cbor",
        ])
        .unwrap_err();
        let message = err.to_string();

        assert!(
            message.to_lowercase().contains("cannot be used with") || message.contains("conflict")
        );
    }

    #[test]
    fn save_proof_requires_prove() {
        let err =
            Args::try_parse_from(["agg-bench", "--proofs", "a.bin", "--save-proof", "agg.bin"])
                .unwrap_err();
        assert!(err.to_string().contains("--prove"));

        Args::try_parse_from([
            "agg-bench",
            "--proofs",
            "a.bin",
            "--prove",
            "--save-proof",
            "agg.bin",
        ])
        .unwrap();
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

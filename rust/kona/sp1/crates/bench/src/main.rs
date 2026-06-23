//! Local range-program benchmark entrypoint for kona-sp1.

use std::{
    env,
    path::{Path, PathBuf},
    sync::Arc,
    time::Instant,
};

use anyhow::Context;
use clap::Parser;
use kona_sp1_host_utils::{
    fetcher::OPSuccinctDataFetcher, host::OPSuccinctHost, setup_logger, stats::ExecutionStats,
    witness_generation::traits::WitnessGenerator,
};
use kona_sp1_proof_utils::{get_range_elf, initialize_host};
use sp1_sdk::{Elf, ProveRequest, Prover, ProverClient, ProvingKey, SP1Stdin};
use url::Url;

const REQUIRED_RPC_ENV: [&str; 3] = ["L1_RPC", "L2_RPC", "L2_NODE_RPC"];
const OPTIONAL_RPC_ENV: [&str; 1] = ["L1_BEACON_RPC"];

#[derive(Debug, Parser)]
#[command(about = "Run the kona-sp1 range program against real RPC data, or a prebuilt SP1 stdin")]
struct Args {
    /// Inclusive starting L2 block number. The block hash at this height is the
    /// pre-state. Required unless --load-stdin is set.
    #[arg(long, required_unless_present = "load_stdin")]
    start: Option<u64>,

    /// Inclusive ending L2 block number. This block is the claimed post-state.
    /// Required unless --load-stdin is set.
    #[arg(long, required_unless_present = "load_stdin")]
    end: Option<u64>,

    /// Also produce a compressed proof after the execute-only stats pass.
    #[arg(long, default_value_t = false)]
    prove: bool,

    /// Persist the compressed range proof to this path.
    #[arg(long, requires = "prove")]
    save_proof: Option<PathBuf>,

    /// Allow timestamp-based L1-head fallback when `SafeDB` is unavailable.
    #[arg(long, default_value_t = false)]
    safe_db_fallback: bool,

    /// Save the obtained SP1 stdin to this path, then continue.
    #[arg(long)]
    save_stdin: Option<PathBuf>,

    /// Load a prebuilt SP1 stdin from this path instead of fetching from RPC.
    /// Needs no RPC env vars; --start and --end are not required.
    #[arg(long)]
    load_stdin: Option<PathBuf>,

    /// Skip the SP1 execute pass entirely. Use on a fetch-only host that just
    /// saves stdin (with --save-stdin) for proving elsewhere; no range ELF is
    /// required in that mode.
    #[arg(long, default_value_t = false)]
    no_execute: bool,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let args = Args::parse();
    setup_logger();

    let (stdin, stats_ctx) = if let Some(path) = &args.load_stdin {
        let stdin = load_stdin(path)?;
        tracing::info!("loaded SP1 stdin from {}", path.display());
        (stdin, None)
    } else {
        let start = args.start.expect("--start is required unless --load-stdin is set");
        let end = args.end.expect("--end is required unless --load-stdin is set");
        validate_block_range(start, end)?;
        ensure_required_rpc_env()?;

        let fetcher = Arc::new(
            OPSuccinctDataFetcher::new_with_rollup_config()
                .await
                .context("failed to initialize RPC data fetcher from environment")?,
        );
        let host = initialize_host(fetcher.clone());

        let witness_start = Instant::now();
        let host_args = host
            .fetch(start, end, None, args.safe_db_fallback)
            .await
            .context("failed to fetch host arguments")?;
        let witness = host.run(&host_args).await.context("failed to generate witness data")?;
        let stdin = host
            .witness_generator()
            .get_sp1_stdin(witness)
            .context("failed to serialize witness into SP1 stdin")?;
        let witness_secs = witness_start.elapsed().as_secs();

        (stdin, Some((fetcher, start, end, witness_secs)))
    };

    if let Some(path) = &args.save_stdin {
        save_stdin(&stdin, path)?;
        tracing::info!("saved SP1 stdin to {}", path.display());
    }

    if args.no_execute && !args.prove {
        tracing::info!(
            "skipping SP1 execute pass (--no-execute); stdin is ready for proving elsewhere"
        );
        return Ok(());
    }

    let elf = get_range_elf();
    ensure_non_empty_elf(elf)?;

    // Backend selected by the SP1_PROVER env var (default: cpu).
    let prover = ProverClient::from_env().await;

    if !args.no_execute {
        let execute_start = Instant::now();
        let (_public_values, report) = prover
            .execute(Elf::Static(elf), stdin.clone())
            .calculate_gas(true)
            .deferred_proof_verification(false)
            .await
            .context("failed to execute range program")?;
        let execute_secs = execute_start.elapsed().as_secs();

        match stats_ctx {
            Some((fetcher, start, end, witness_secs)) => {
                let block_data = fetcher
                    .get_l2_block_data_range(start, end)
                    .await
                    .context("failed to fetch L2 block stats")?;
                let stats =
                    ExecutionStats::new(0, &block_data, &report, witness_secs, execute_secs);
                println!("{stats}");
            }
            None => {
                tracing::info!(
                    "execute: {} cycles, gas {:?}, {execute_secs}s (per-block stats skipped: --load-stdin has no RPC)",
                    report.total_instruction_count(),
                    report.gas(),
                );
            }
        }
    }

    if args.prove {
        let proving_key =
            prover.setup(Elf::Static(elf)).await.context("failed to set up proving key")?;
        let prove_start = Instant::now();
        let proof = prover
            .prove(&proving_key, stdin)
            .compressed()
            .await
            .context("failed to produce compressed proof")?;
        tracing::info!("prove wall-clock: {:?}", prove_start.elapsed());
        let verify_start = Instant::now();
        prover
            .verify(&proof, proving_key.verifying_key(), None)
            .context("compressed range proof failed local verification")?;
        tracing::info!("local verify: {:?}", verify_start.elapsed());
        if let Some(path) = &args.save_proof {
            proof.save(path).context("failed to save compressed range proof")?;
            tracing::info!("saved compressed range proof to {}", path.display());
        }
    }

    Ok(())
}

fn save_stdin(stdin: &SP1Stdin, path: &Path) -> anyhow::Result<()> {
    let bytes = bincode::serde::encode_to_vec(stdin, bincode::config::standard())
        .context("failed to serialize SP1 stdin")?;
    std::fs::write(path, bytes)
        .with_context(|| format!("failed to write SP1 stdin to {}", path.display()))?;
    Ok(())
}

fn load_stdin(path: &Path) -> anyhow::Result<SP1Stdin> {
    let bytes = std::fs::read(path)
        .with_context(|| format!("failed to read SP1 stdin from {}", path.display()))?;
    let (stdin, _): (SP1Stdin, usize) =
        bincode::serde::decode_from_slice(&bytes, bincode::config::standard())
            .context("failed to deserialize SP1 stdin")?;
    Ok(stdin)
}

fn ensure_required_rpc_env() -> anyhow::Result<()> {
    validate_rpc_env_vars(|key| env::var(key).ok())
}

fn validate_block_range(start: u64, end: u64) -> anyhow::Result<()> {
    anyhow::ensure!(start < end, "--start must be less than --end");
    Ok(())
}

fn ensure_non_empty_elf(elf: &[u8]) -> anyhow::Result<()> {
    anyhow::ensure!(
        !elf.is_empty(),
        "range-elf is missing or empty. Build a real ELF first with: \
         cd rust/kona/sp1 && just build-elfs"
    );
    Ok(())
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
    fn validate_block_range_rejects_empty_or_reversed_ranges() {
        assert!(validate_block_range(1, 2).is_ok());

        let equal_err = validate_block_range(2, 2).unwrap_err();
        assert!(equal_err.to_string().contains("--start must be less than --end"));

        let reversed_err = validate_block_range(3, 2).unwrap_err();
        assert!(reversed_err.to_string().contains("--start must be less than --end"));
    }

    #[test]
    fn ensure_non_empty_elf_rejects_missing_or_empty_elf() {
        assert!(ensure_non_empty_elf(&[1]).is_ok());

        let err = ensure_non_empty_elf(&[]).unwrap_err();
        assert!(err.to_string().contains("range-elf is missing or empty"));
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

    #[test]
    fn validate_rpc_env_vars_accepts_valid_configuration() {
        let values = [
            ("L1_RPC", "https://example.com"),
            ("L2_RPC", "http://localhost:8545"),
            ("L2_NODE_RPC", "http://127.0.0.1:9545"),
            ("L1_BEACON_RPC", ""),
        ];
        validate_rpc_env_vars(|key| get_var(&values, key)).unwrap();
    }

    #[test]
    fn save_proof_requires_prove() {
        let err = Args::try_parse_from([
            "range-bench",
            "--start",
            "1",
            "--end",
            "2",
            "--save-proof",
            "r.bin",
        ])
        .unwrap_err();

        assert!(err.to_string().contains("--prove"));
    }

    #[test]
    fn save_then_load_stdin_round_trips() {
        let mut stdin = SP1Stdin::new();
        stdin.write_slice(&[9, 8, 7, 6]);
        let path = std::env::temp_dir()
            .join(format!("kona-sp1-range-stdin-roundtrip-{}.bin", std::process::id()));

        save_stdin(&stdin, &path).unwrap();
        let loaded = load_stdin(&path).unwrap();
        let _ = std::fs::remove_file(&path);

        assert_eq!(loaded.buffer, stdin.buffer);
        assert_eq!(loaded.ptr, stdin.ptr);
        assert!(loaded.proofs.is_empty());
    }

    #[test]
    fn load_stdin_missing_file_errors() {
        let err = load_stdin(Path::new("/nonexistent/kona-sp1-missing.bin")).unwrap_err();
        assert!(err.to_string().contains("failed to read SP1 stdin"));
    }

    #[test]
    fn no_execute_parses_in_fetch_mode() {
        let args =
            Args::try_parse_from(["range-bench", "--start", "1", "--end", "2", "--no-execute"])
                .unwrap();
        assert!(args.no_execute);
        assert!(!args.prove);

        // Defaults to false when the flag is absent.
        let args = Args::try_parse_from(["range-bench", "--start", "1", "--end", "2"]).unwrap();
        assert!(!args.no_execute);
    }

    #[test]
    fn load_stdin_allows_missing_block_range() {
        Args::try_parse_from(["range-bench", "--load-stdin", "s.bin"]).unwrap();

        let err = Args::try_parse_from(["range-bench"]).unwrap_err();
        let message = err.to_string();
        assert!(message.contains("--start") || message.contains("start"));
    }
}

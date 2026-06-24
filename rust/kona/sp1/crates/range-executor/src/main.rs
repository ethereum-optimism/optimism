//! Runs the kona-sp1 `range` guest program in SP1 **execute** mode (no proving) against a real
//! chain's witness, so the program can be exercised end-to-end by op-e2e action tests.
//!
//! The flow mirrors the native fault-proof program harness:
//! 1. Parse the same [`SingleChainHost`] boot inputs the native kona-host accepts.
//! 2. Run the kona-host preimage server and collect the witness with the SP1 witness generator.
//! 3. Execute the `range` ELF over that witness in the SP1 zkVM emulator.
//! 4. Read the committed [`BootInfoStruct`] and confirm it matches the claimed transition.
//!
//! Exit codes (matching the native harness convention, where exit 1 == "claim rejected"):
//! - `0`: the claimed state transition is valid (guest executed and committed the claim).
//! - `1`: the claimed state transition is invalid (the guest rejected it).
//! - `2`: an infrastructure error prevented evaluation (witness generation failed, ELF missing,
//!   etc.) — distinct from a claim verdict.

use std::process::ExitCode;

use alloy_primitives::B256;
use clap::Parser;
use kona_host::single::SingleChainHost;
use kona_preimage::{BidirectionalChannel, PreimageKey};
use kona_proof::boot::L2_CLAIM_KEY;
use kona_sp1_client_utils::{
    boot::BootInfoStruct,
    witness::{DefaultWitnessData, WitnessData},
};
use kona_sp1_ethereum_host_utils::witness_generator::ETHDAWitnessGenerator;
use kona_sp1_host_utils::witness_generation::WitnessGenerator;
use sp1_sdk::{Elf, Prover, ProverClient, SP1Stdin};

/// Exit code signalling a valid claim.
const EXIT_VALID: u8 = 0;
/// Exit code signalling an invalid claim (the guest rejected the transition).
const EXIT_INVALID: u8 = 1;
/// Exit code signalling an infrastructure error (no claim verdict was reached).
const EXIT_INFRA: u8 = 2;

/// Runs the kona-sp1 `range` guest in SP1 execute mode against a chain's witness.
#[derive(Parser, Debug)]
#[command(
    name = "kona-sp1-range-executor",
    about = "Generate a witness for a single-chain state transition and run the kona-sp1 range \
             guest in SP1 execute mode, validating the claimed output root."
)]
struct Cli {
    #[command(subcommand)]
    command: Command,
}

/// Subcommands, mirroring the native kona-host CLI shape (`kona-host single ...`) so the same
/// oracle-server command builder can target this binary.
#[derive(clap::Subcommand, Debug)]
enum Command {
    /// Generate the witness for a single-chain state transition and run the range guest in SP1
    /// execute mode, validating the claimed output root.
    Single(SingleArgs),
}

/// Arguments for the `single` subcommand.
///
/// Boot inputs and RPC endpoints are identical to the native kona-host `single` interface.
/// `SingleChainHost` requires `--native` or `--server`; callers pass `--server` (the value is
/// unused — we drive the preimage server directly).
#[derive(clap::Args, Debug)]
struct SingleArgs {
    /// Boot inputs and RPC endpoints (the native kona-host `single` flags).
    #[command(flatten)]
    host: SingleChainHost,
    /// Corrupt the claimed output root in the generated witness before execution, so the guest
    /// rejects it. Witness generation still runs on the real claim (and succeeds); only the
    /// guest's view of the claim is tampered. Used to exercise the invalid-claim (soundness)
    /// path: the guest re-derives the real root, finds it does not match, and aborts (exit 1).
    #[arg(long)]
    corrupt_claimed_root: bool,
}

/// The claim verdict reached by running the guest.
enum Verdict {
    /// The guest accepted and committed the claimed state transition.
    Valid,
    /// The guest rejected the claimed state transition.
    Invalid,
}

fn main() -> ExitCode {
    init_tracing();

    let Command::Single(args) = Cli::parse().command;
    let host = args.host;
    let claimed_output_root = host.claimed_l2_output_root;
    let claimed_block_number = host.claimed_l2_block_number;

    if kona_sp1_elfs::RANGE_ELF.is_empty() {
        return infra(
            "RANGE_ELF is an empty placeholder; build the guest ELFs first with \
             `cd rust/kona/sp1 && just build-elfs`"
                .to_string(),
        );
    }

    let runtime = match tokio::runtime::Builder::new_multi_thread().enable_all().build() {
        Ok(runtime) => runtime,
        Err(err) => return infra(format!("failed to build tokio runtime: {err}")),
    };

    match runtime.block_on(run(
        host,
        args.corrupt_claimed_root,
        claimed_output_root,
        claimed_block_number,
    )) {
        Ok(Verdict::Valid) => ExitCode::from(EXIT_VALID),
        Ok(Verdict::Invalid) => ExitCode::from(EXIT_INVALID),
        Err(err) => infra(format!("failed to evaluate the claim: {err:?}")),
    }
}

/// Generates the witness, runs the `range` ELF in SP1 execute mode, and reports the claim verdict.
async fn run(
    host: SingleChainHost,
    corrupt_claimed_root: bool,
    claimed_output_root: B256,
    claimed_block_number: u64,
) -> anyhow::Result<Verdict> {
    let stdin = generate_stdin(host, corrupt_claimed_root).await?;

    // `build()` and `execute(..)` are async; the latter runs the zkVM emulator (no proving).
    let client = ProverClient::builder().cpu().build().await;
    let elf = Elf::Static(kona_sp1_elfs::RANGE_ELF);
    let (mut public_values, _report) = match client.execute(elf, stdin).await {
        Ok(output) => output,
        Err(err) => {
            // An executor-level failure on a self-generated witness means the guest could not run
            // the transition to completion; treat it as a rejected claim.
            tracing::info!("range guest execution failed (claim rejected): {err}");
            return Ok(Verdict::Invalid);
        }
    };

    // When the guest rejects the transition (e.g. derived output root != claimed), it aborts before
    // committing, so `execute` still returns `Ok` but the public values are empty. A successful run
    // commits exactly one `BootInfoStruct` via `sp1_zkvm::io::commit`.
    if public_values.as_slice().is_empty() {
        tracing::info!("range guest committed no output; claim rejected");
        return Ok(Verdict::Invalid);
    }

    // Non-empty public values mean the guest ran to completion and committed the boot info. Confirm
    // it matches the requested claim; the guest already enforces `derived == claimed`, so a
    // mismatch here is unexpected (a witness-integrity guard rather than an independent
    // verdict).
    let boot: BootInfoStruct = public_values.read();
    if boot.l2PostRoot != claimed_output_root || boot.l2BlockNumber != claimed_block_number {
        tracing::error!(
            committed_root = %boot.l2PostRoot,
            committed_block = boot.l2BlockNumber,
            %claimed_output_root,
            claimed_block_number,
            "range guest committed boot info that does not match the requested claim",
        );
        return Ok(Verdict::Invalid);
    }

    tracing::info!(
        l2_block_number = boot.l2BlockNumber,
        l2_post_root = %boot.l2PostRoot,
        "range guest validated the claimed state transition",
    );
    Ok(Verdict::Valid)
}

/// Generates the witness for the configured state transition and encodes it as SP1 stdin.
///
/// This mirrors the witness-generation sequence in `OPSuccinctHost::run`
/// (`kona-sp1-host-utils`); keep the `start_server`/`abort` semantics in sync with it. We drive the
/// preimage server directly here rather than going through `SingleChainOPSuccinctHost`, which would
/// require an RPC-configured `OPSuccinctDataFetcher` we do not need.
async fn generate_stdin(
    host: SingleChainHost,
    corrupt_claimed_root: bool,
) -> anyhow::Result<SP1Stdin> {
    let witness_generator = ETHDAWitnessGenerator::new();

    let preimage = BidirectionalChannel::new()?;
    let hint = BidirectionalChannel::new()?;

    let server_task = host.start_server(hint.host, preimage.host).await?;
    let witness: DefaultWitnessData = witness_generator.run(preimage.client, hint.client).await?;
    // Match the host crate: abort the server task explicitly. Awaiting both tasks hangs because
    // the preimage server never returns on its own once the client is done.
    server_task.abort();

    let witness = if corrupt_claimed_root {
        corrupt_claim(witness, host.claimed_l2_output_root)?
    } else {
        witness
    };

    witness_generator.get_sp1_stdin(witness)
}

/// Overwrites the claimed-output-root boot preimage in the witness with a value guaranteed to
/// differ from the real one, so the guest's `derived == claimed` check fails and it aborts.
///
/// Witness generation already ran on the real claim, so the rest of the witness is valid; only the
/// guest's view of the claim is tampered. This is sound because boot-info local keys are not
/// hash-verified by `PreimageStore::check_preimages` (they are trusted bootstrap inputs, bound
/// on-chain via the public values), so the corrupted claim flows straight to the guest's equality
/// assertion.
fn corrupt_claim(
    witness: DefaultWitnessData,
    real_claim: B256,
) -> anyhow::Result<DefaultWitnessData> {
    let (mut preimage_store, blob_data) = witness.into_parts();

    // Flip a bit so the claim is guaranteed to differ from the real derived output root.
    let mut junk = real_claim;
    junk.0[0] ^= 0x01;

    preimage_store
        .overwrite_preimage(PreimageKey::new_local(L2_CLAIM_KEY.to()), junk.as_slice().to_vec())
        .map_err(|e| anyhow::anyhow!("failed to overwrite claimed-root preimage: {e:?}"))?;

    Ok(DefaultWitnessData::from_parts(preimage_store, blob_data))
}

/// Logs an infrastructure error and returns the infra exit code.
fn infra(message: String) -> ExitCode {
    tracing::error!("{message}");
    ExitCode::from(EXIT_INFRA)
}

/// Initializes a best-effort tracing subscriber driven by `RUST_LOG` (defaulting to `info`).
fn init_tracing() {
    use tracing_subscriber::{EnvFilter, fmt};
    let filter = EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new("info"));
    let _ = fmt().with_env_filter(filter).try_init();
}

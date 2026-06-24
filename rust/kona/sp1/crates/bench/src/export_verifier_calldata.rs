//! Export the on-chain verifier calldata for a saved aggregation proof.
//!
//! Loads a PLONK (or Groth16) aggregation proof produced by `agg-bench
//! --save-proof` / `plonk-prove-bench --save-proof`, derives the aggregation
//! program's verification key from the aggregation ELF, and prints the three
//! arguments that SP1's `verifyProof(bytes32,bytes,bytes)` Solidity verifier
//! takes -- `programVKey`, `publicValues`, `proofBytes` -- as `0x`-prefixed hex.
//!
//! The `verify-onchain` just target feeds these straight to `cast call` against
//! the real SP1 verifier contract (shipped in the circuit artifacts), so a saved
//! proof can be checked for on-chain acceptance locally.
//!
//! Output (stdout, one line): `CALLDATA <vkey> <publicValues> <proofBytes>` --
//! the `CALLDATA ` sentinel lets the caller pick the result line out cleanly even
//! if a dependency writes to stdout. All logging goes to stderr.

use std::path::PathBuf;

use alloy_primitives::hex;
use anyhow::Context;
use clap::Parser;
use kona_sp1_host_utils::setup_logger;
use kona_sp1_proof_utils::get_agg_elf;
use sp1_sdk::{Elf, HashableKey, Prover, ProverClient, ProvingKey, SP1ProofWithPublicValues};

#[derive(Debug, Parser)]
#[command(about = "Export programVKey/publicValues/proofBytes for on-chain SP1 verification")]
struct Args {
    /// Saved aggregation proof (PLONK or Groth16), as written by `--save-proof`.
    #[arg(long)]
    proof: PathBuf,
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let args = Args::parse();
    setup_logger();

    let agg_elf = get_agg_elf();
    anyhow::ensure!(
        !agg_elf.is_empty(),
        "aggregation-elf is missing or empty. Build it first with: \
         cd rust/kona/sp1 && just build-elfs"
    );

    let proof = SP1ProofWithPublicValues::load(&args.proof).with_context(|| {
        format!("failed to load aggregation proof from {}", args.proof.display())
    })?;

    // The on-chain verifier checks against the aggregation program's vkey, which
    // is not stored in the proof -- derive it from the aggregation ELF. Setup is
    // far cheaper than proving (no range proofs or RPC are needed here).
    let prover = ProverClient::from_env().await;
    let agg_proving_key = prover
        .setup(Elf::Static(agg_elf))
        .await
        .context("failed to set up aggregation proving key")?;
    let vkey = agg_proving_key.verifying_key().bytes32();

    // `proof.bytes()` is the on-chain encoding: the 4-byte verifier selector
    // followed by the proof; `public_values` are the committed bytes verbatim.
    let public_values = format!("0x{}", hex::encode(proof.public_values.as_slice()));
    let proof_bytes = format!("0x{}", hex::encode(proof.bytes()));

    tracing::info!("programVKey: {vkey}");
    tracing::info!("publicValues: {} bytes", proof.public_values.as_slice().len());
    tracing::info!("proofBytes: {} bytes", proof.bytes().len());

    println!("CALLDATA {vkey} {public_values} {proof_bytes}");
    Ok(())
}

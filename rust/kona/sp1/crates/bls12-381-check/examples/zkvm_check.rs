//! Runs the EIP-2537 differential check inside SP1's CPU executor.
//!
//! Executes `elf/bls12-381-check-elf` (built by `just build-bls12-381-check-elf` in
//! `rust/kona/sp1`) over [`full_corpus`], checks the committed case count and outcome digest
//! against the host arkworks oracle, and prints per-op cycle counts. This is an example rather
//! than a test because a run takes far longer than nextest's test timeout; use
//! `just check-bls12-381-zkvm` from `rust/kona/sp1`.

use std::time::Instant;

use anyhow::{Context, Result, ensure};
use kona_sp1_bls12_381_check::{ArkworksCrypto, corpus::full_corpus, public_values, run_call};
use sp1_sdk::{Elf, Prover, ProverClient, SP1Stdin};

#[tokio::main]
async fn main() -> Result<()> {
    let cases = full_corpus();
    let expected = public_values(
        &cases.iter().map(|c| run_call(&ArkworksCrypto, &c.call)).collect::<Vec<_>>(),
    );

    let elf = kona_sp1_elfs::load_elf("bls12-381-check-elf")
        .context("load bls12-381-check-elf (run `just build-bls12-381-check-elf`)")?;
    let mut stdin = SP1Stdin::new();
    stdin.write_slice(&rkyv::to_bytes::<rkyv::rancor::Error>(&cases)?);

    let started = Instant::now();
    let client = ProverClient::builder().cpu().build().await;
    let (committed, report) = client.execute(Elf::Dynamic(elf.into()), stdin).await?;
    let elapsed = started.elapsed();

    ensure!(report.exit_code == 0, "guest exited with code {}", report.exit_code);
    ensure!(
        committed.as_slice() == expected,
        "guest public values (case count, outcome digest) do not match the host arkworks oracle"
    );

    println!("zkVM BLS12-381 check passed: {} cases in {elapsed:.1?}", cases.len());
    println!(
        "{:<14} | {:>6} | {:>16} | {:>16} | {:>7}",
        "op", "cases", "arkworks cycles", "sp1 cycles", "ratio"
    );
    let mut ops: Vec<&str> = cases.iter().map(|c| c.call.op_name()).collect();
    ops.dedup(); // full_corpus() is grouped by op.
    for op in ops {
        let arkworks = report
            .cycle_tracker
            .get(&format!("bls-arkworks-{op}"))
            .copied()
            .with_context(|| format!("no arkworks cycles for {op}"))?;
        let sp1 = report
            .cycle_tracker
            .get(&format!("bls-sp1-{op}"))
            .copied()
            .with_context(|| format!("no sp1 cycles for {op}"))?;
        let invocations = report
            .invocation_tracker
            .get(&format!("bls-sp1-{op}"))
            .copied()
            .with_context(|| format!("no sp1 invocations for {op}"))?;
        println!(
            "{op:<14} | {invocations:>6} | {arkworks:>16} | {sp1:>16} | {:>6.1}x",
            arkworks as f64 / sp1 as f64
        );
    }
    println!("total instructions: {}", report.total_instruction_count());
    Ok(())
}

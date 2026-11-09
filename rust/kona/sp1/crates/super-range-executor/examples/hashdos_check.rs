//! Runs the hash-flooding (`HashDoS`) regression check inside SP1's CPU executor.
//!
//! Executes `elf/hashdos-check-elf` (built by `just build-hashdos-check-elf` in `rust/kona/sp1`): a
//! TSTORE loop through revm's transient storage map. Under the guest hash seed B:
//! - keys `top_limb << 192 | i` whose `top_limb` is word 0 of seed A's shared seed, which collide
//!   under seed A, must cost no more than control keys; under seed A they must still be quadratic,
//!   which shows the harness models the guest's hashing;
//! - keys `i << 64`, whose first and third words are zero, must cost no more than control keys.
//!   They collide under every seed if a map's per-hasher seed is zero.
//!
//! This is an example rather than a test because it builds its own ELF; use
//! `just check-hashdos-zkvm` from `rust/kona/sp1`.

use core::hash::Hasher;
use std::sync::Arc;

use anyhow::{Context, Result, ensure};
use sp1_sdk::{Elf, Prover, ProverClient, SP1Stdin};

/// Gas limit of the deposit. The loop runs until out of gas, so every run uses all of it.
const GAS: u64 = 250_000;
/// The guest hash seed that the colliding keys are built for.
const SEED_A: u64 = 1;
/// A different guest hash seed.
const SEED_B: u64 = 2;
/// A top limb that collides under neither seed.
const CONTROL_TOP_LIMB: u64 = 0x0123_4567_89AB_CDEF;

/// Word 0 of the shared seed that the guest derives from `seed`: with per-hasher seed 0,
/// `write_u128(1)` returns `folded_multiply(1, s0) = s0`.
fn s0(seed: u64) -> u64 {
    let shared = foldhash::SharedSeed::from_u64(seed);
    let mut hasher = foldhash::fast::FoldHasher::with_seed(0, &shared);
    hasher.write_u128(1);
    hasher.finish()
}

#[tokio::main]
async fn main() -> Result<()> {
    let elf = Arc::<[u8]>::from(
        kona_sp1_elfs::load_elf("hashdos-check-elf")
            .context("load hashdos-check-elf (run `just build-hashdos-check-elf`)")?,
    );
    let client = ProverClient::builder().cpu().build().await;

    // Executes the guest and returns its instruction count.
    let run = async |seed: u64, top_limb: u64, step_limb: u64| -> Result<u64> {
        let mut stdin = SP1Stdin::new();
        stdin.write(&seed);
        stdin.write(&(GAS, top_limb, step_limb));
        let (mut committed, report) =
            client.execute(Elf::Dynamic(elf.clone()), stdin).calculate_gas(true).await?;
        ensure!(report.exit_code == 0, "guest exited with code {}", report.exit_code);
        let gas_used = committed.read::<u64>();
        ensure!(gas_used == GAS, "deposit used {gas_used} gas, expected {GAS}");
        Ok(report.total_instruction_count())
    };

    let control = run(SEED_B, CONTROL_TOP_LIMB, 0).await?;
    let known = run(SEED_A, s0(SEED_A), 0).await?;
    let stale = run(SEED_B, s0(SEED_A), 0).await?;
    let zero_words = run(SEED_B, 0, 1).await?;
    println!(
        "instructions: control {control}, known-seed keys {known}, other-seed keys {stale}, \
         zero-word keys {zero_words}"
    );

    ensure!(
        known >= 10 * control,
        "positive control failed: keys colliding under the installed seed are not quadratic, so \
         this harness no longer models the guest's foldhash seeding"
    );
    ensure!(
        stale * 10 <= control * 12,
        "keys colliding under seed A cost {stale} instructions under seed B vs {control} for \
         control keys: the guest hash seed is not applied"
    );
    ensure!(
        zero_words * 10 <= control * 12,
        "keys whose first and third words are zero cost {zero_words} instructions vs {control} \
         for control keys: a map's per-hasher seed is zero"
    );
    println!("zkVM HashDoS check passed");
    Ok(())
}

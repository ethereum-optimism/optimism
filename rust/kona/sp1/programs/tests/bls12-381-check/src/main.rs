//! Test-only SP1 guest: differential check of `CustomCrypto`'s EIP-2537 ops against revm's
//! arkworks backend inside the zkVM.

#![cfg_attr(target_os = "zkvm", no_main)]
#[cfg(target_os = "zkvm")]
sp1_zkvm::entrypoint!(main);

use kona_sp1_bls12_381_check::{BlsCase, public_values, run_call};
use kona_sp1_client_utils::precompiles::CustomCrypto;
use revm::precompile::DefaultCrypto;

/// Entrypoint to the BLS12-381 check program.
pub fn main() {
    println!("{}", kona_sp1_build_info::BUILD_MARKER);

    let cases = rkyv::from_bytes::<Vec<BlsCase>, rkyv::rancor::Error>(&sp1_zkvm::io::read_vec())
        .expect("decode BLS cases");
    let custom = CustomCrypto::default();

    let mut outcomes = Vec::with_capacity(cases.len());
    for case in &cases {
        let op = case.call.op_name();

        println!("cycle-tracker-report-start: bls-arkworks-{op}");
        let reference = run_call(&DefaultCrypto, &case.call);
        println!("cycle-tracker-report-end: bls-arkworks-{op}");

        println!("cycle-tracker-report-start: bls-sp1-{op}");
        let accelerated = run_call(&custom, &case.call);
        println!("cycle-tracker-report-end: bls-sp1-{op}");

        assert_eq!(accelerated, reference, "case {}", case.label);
        outcomes.push(accelerated);
    }

    sp1_zkvm::io::commit_slice(&public_values(&outcomes));
}

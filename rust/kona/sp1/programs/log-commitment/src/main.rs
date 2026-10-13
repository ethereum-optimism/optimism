//! SP1 guest that commits to the logs under each input receipts root.

#![cfg_attr(target_os = "zkvm", no_main)]
#[cfg(target_os = "zkvm")]
sp1_zkvm::entrypoint!(main);

use alloy_primitives::B256;
use kona_sp1_client_utils::log_commitment::log_commitment_public_values;

/// Entrypoint to the log-commitment program.
pub fn main() {
    let receipts_roots = sp1_zkvm::io::read::<Vec<B256>>();
    let witness = sp1_zkvm::io::read_vec();
    sp1_zkvm::io::commit_slice(&log_commitment_public_values(&receipts_roots, &witness));
}

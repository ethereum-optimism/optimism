//! SP1 guest that commits to the logs under each input receipts root.

#![cfg_attr(target_os = "zkvm", no_main)]
#[cfg(target_os = "zkvm")]
sp1_zkvm::entrypoint!(main);

use alloy_primitives::B256;
use kona_sp1_client_utils::log_commitment::log_commitment_public_values;

/// Entrypoint to the log-commitment program.
///
/// Installs no guest hash seed (see `programs/foldhash-seeded`): this program builds no hash map,
/// and a foldhash map created here panics. A hash map added on its path needs a seed installed
/// first in `main`, from an input hash that no on-chain actor can predict or choose before the
/// map's keys are fixed, such as keccak256 over `receipts_roots`.
pub fn main() {
    println!("{}", kona_sp1_build_info::BUILD_MARKER);

    let receipts_roots = sp1_zkvm::io::read::<Vec<B256>>();
    let witness = sp1_zkvm::io::read_vec();
    sp1_zkvm::io::commit_slice(&log_commitment_public_values(&receipts_roots, &witness));
}

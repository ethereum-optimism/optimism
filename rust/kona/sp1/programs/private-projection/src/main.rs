//! SP1 guest of the `sp1-private-projection-v1` relation. It commits exactly the 672-byte
//! `PublicValuesV1` that admission rebuilds from its own state (spec-sound-profile §C.2).
#![cfg_attr(target_os = "zkvm", no_main)]
#[cfg(target_os = "zkvm")]
sp1_zkvm::entrypoint!(main);

use kona_sp1_client_utils::private_projection::execute_encoded;

/// Execute the same relation as the native tests and host, then commit its raw public values.
pub fn main() {
    println!("{}", kona_sp1_build_info::BUILD_MARKER);
    // Initialize the same fixed KZG backend as the super-range guest outside the pure relation.
    assert!(revm::precompile::install_crypto(
        kona_sp1_client_utils::precompiles::CustomCrypto::default()
    ));
    // The witness transport is JSON and not consensus; only the committed bytes are.
    let bytes = sp1_zkvm::io::read_vec();
    let public_values = execute_encoded(&bytes).expect("private projection relation failed");
    // Raw bytes, not `commit` (bincode): the proof's public values are exactly `PublicValuesV1`.
    sp1_zkvm::io::commit_slice(&public_values);
}

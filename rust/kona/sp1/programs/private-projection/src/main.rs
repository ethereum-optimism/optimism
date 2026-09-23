//! Experimental private execution-to-projection guest; network enforcement is separate.
#![cfg_attr(target_os = "zkvm", no_main)]
#[cfg(target_os = "zkvm")]
sp1_zkvm::entrypoint!(main);

use kona_sp1_client_utils::private_projection::execute_encoded;

/// Execute the same relation as native tests and commit only its public journal.
pub fn main() {
    println!("{}", kona_sp1_build_info::BUILD_MARKER);
    // Initialize the same fixed KZG backend as the super-range guest outside the pure relation.
    assert!(revm::precompile::install_crypto(
        kona_sp1_client_utils::precompiles::CustomCrypto::default()
    ));
    let bytes = sp1_zkvm::io::read_vec();
    let output = execute_encoded(&bytes).expect("private projection relation failed");
    sp1_zkvm::io::commit(&output);
}

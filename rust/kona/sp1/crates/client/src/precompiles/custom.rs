//! Custom crypto provider for KZG proof verification.

use kzg_rs::{Bytes32, Bytes48, KzgProof, KzgSettings};
use revm::precompile::{Crypto, PrecompileHalt};

/// Custom cryptography provider using kzg-rs for KZG proof verification.
#[derive(Debug)]
pub struct CustomCrypto {
    kzg_settings: KzgSettings,
}

impl Default for CustomCrypto {
    fn default() -> Self {
        Self { kzg_settings: KzgSettings::load_trusted_setup_file().unwrap() }
    }
}

impl Crypto for CustomCrypto {
    fn verify_kzg_proof(
        &self,
        z: &[u8; 32],
        y: &[u8; 32],
        commitment: &[u8; 48],
        proof: &[u8; 48],
    ) -> Result<(), PrecompileHalt> {
        let z = Bytes32::from_slice(z).map_err(|_| PrecompileHalt::BlobVerifyKzgProofFailed)?;
        let y = Bytes32::from_slice(y).map_err(|_| PrecompileHalt::BlobVerifyKzgProofFailed)?;
        let commitment = Bytes48::from_slice(commitment)
            .map_err(|_| PrecompileHalt::BlobVerifyKzgProofFailed)?;
        let proof =
            Bytes48::from_slice(proof).map_err(|_| PrecompileHalt::BlobVerifyKzgProofFailed)?;

        KzgProof::verify_kzg_proof(&commitment, &z, &y, &proof, &self.kzg_settings)
            .map_err(|_| PrecompileHalt::BlobVerifyKzgProofFailed)?;

        Ok(())
    }
}

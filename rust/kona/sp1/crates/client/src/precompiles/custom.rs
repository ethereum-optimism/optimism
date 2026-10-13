//! Custom crypto provider: KZG proof verification (`kzg-rs`) and EIP-2537 BLS12-381 operations
//! (`sp1_bls12_381`), both SP1-accelerated in the zkVM.

use kzg_rs::{Bytes32, Bytes48, KzgProof, KzgSettings};
use revm::precompile::{
    Crypto, PrecompileHalt,
    bls12_381::{G1Point, G1PointScalar, G2Point, G2PointScalar},
};

use super::bls12_381;

/// SP1-accelerated [`Crypto`]: KZG via `kzg-rs`, EIP-2537 via `sp1_bls12_381`.
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

        match KzgProof::verify_kzg_proof(&commitment, &z, &y, &proof, &self.kzg_settings) {
            Ok(true) => Ok(()),
            Ok(false) | Err(_) => Err(PrecompileHalt::BlobVerifyKzgProofFailed),
        }
    }

    /// UPSTREAM-MIRROR(override): revm-precompile@42.0.1
    /// `revm_precompile::interface::Crypto::bls12_381_g1_add`
    ///
    /// Must equal the default's `bls12_381::arkworks::p1_add_affine_bytes` bit for bit, errors
    /// included.
    fn bls12_381_g1_add(&self, a: G1Point, b: G1Point) -> Result<[u8; 96], PrecompileHalt> {
        bls12_381::g1_add(a, b)
    }

    /// UPSTREAM-MIRROR(override): revm-precompile@42.0.1
    /// `revm_precompile::interface::Crypto::bls12_381_g1_msm`
    ///
    /// Must equal the default's `bls12_381::arkworks::p1_msm_bytes` bit for bit, errors included.
    fn bls12_381_g1_msm(
        &self,
        pairs: &mut dyn Iterator<Item = Result<G1PointScalar, PrecompileHalt>>,
    ) -> Result<[u8; 96], PrecompileHalt> {
        bls12_381::g1_msm(pairs)
    }

    /// UPSTREAM-MIRROR(override): revm-precompile@42.0.1
    /// `revm_precompile::interface::Crypto::bls12_381_g2_add`
    ///
    /// Must equal the default's `bls12_381::arkworks::p2_add_affine_bytes` bit for bit, errors
    /// included.
    fn bls12_381_g2_add(&self, a: G2Point, b: G2Point) -> Result<[u8; 192], PrecompileHalt> {
        bls12_381::g2_add(a, b)
    }

    /// UPSTREAM-MIRROR(override): revm-precompile@42.0.1
    /// `revm_precompile::interface::Crypto::bls12_381_g2_msm`
    ///
    /// Must equal the default's `bls12_381::arkworks::p2_msm_bytes` bit for bit, errors included.
    fn bls12_381_g2_msm(
        &self,
        pairs: &mut dyn Iterator<Item = Result<G2PointScalar, PrecompileHalt>>,
    ) -> Result<[u8; 192], PrecompileHalt> {
        bls12_381::g2_msm(pairs)
    }

    /// UPSTREAM-MIRROR(override): revm-precompile@42.0.1
    /// `revm_precompile::interface::Crypto::bls12_381_pairing_check`
    ///
    /// Must equal the default's `bls12_381::arkworks::pairing_check_bytes` (via
    /// `pairing_common::pairing_check_bytes_generic`) bit for bit, errors included.
    fn bls12_381_pairing_check(
        &self,
        pairs: &[(G1Point, G2Point)],
    ) -> Result<bool, PrecompileHalt> {
        bls12_381::pairing_check(pairs)
    }

    /// UPSTREAM-MIRROR(override): revm-precompile@42.0.1
    /// `revm_precompile::interface::Crypto::bls12_381_fp_to_g1`
    ///
    /// Must equal the default's `bls12_381::arkworks::map_fp_to_g1_bytes` bit for bit, errors
    /// included.
    fn bls12_381_fp_to_g1(&self, fp: &[u8; 48]) -> Result<[u8; 96], PrecompileHalt> {
        bls12_381::map_fp_to_g1(fp)
    }

    /// UPSTREAM-MIRROR(override): revm-precompile@42.0.1
    /// `revm_precompile::interface::Crypto::bls12_381_fp2_to_g2`
    ///
    /// Must equal the default's `bls12_381::arkworks::map_fp2_to_g2_bytes` bit for bit, errors
    /// included.
    fn bls12_381_fp2_to_g2(&self, fp2: ([u8; 48], [u8; 48])) -> Result<[u8; 192], PrecompileHalt> {
        bls12_381::map_fp2_to_g2(&fp2.0, &fp2.1)
    }
}

#[cfg(test)]
mod tests {
    use alloy_primitives::hex_literal::hex;
    use revm::precompile::Crypto;

    use super::*;

    #[test]
    fn invalid_kzg_opening_is_rejected() {
        let commitment = hex!(
            "8f59a8d2a1a625a17f3fea0fe5eb8c896db3764f3185481bc22f91b4aaffcca25f26936857bc3a7c2539ea8ec3a952b7"
        );
        let z = hex!("73eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000000");
        let mut y = hex!("1522a4a7f34e1ea350ae07c29c96c7e79655aa926122e95fe69fcbd932ca49e9");
        let proof = hex!(
            "a62ad71d14c5719385c0686f1871430475bf3a00f0aa3f7b8dd99a9abc2160744faf0070725e00b60ad9a026a15b1a8c"
        );

        y[31] ^= 1;

        let crypto = CustomCrypto::default();
        let raw_result = KzgProof::verify_kzg_proof(
            &Bytes48::from_slice(&commitment).unwrap(),
            &Bytes32::from_slice(&z).unwrap(),
            &Bytes32::from_slice(&y).unwrap(),
            &Bytes48::from_slice(&proof).unwrap(),
            &crypto.kzg_settings,
        );
        assert!(matches!(raw_result, Ok(false)), "raw kzg-rs result: {raw_result:?}");

        assert!(matches!(
            crypto.verify_kzg_proof(&z, &y, &commitment, &proof),
            Err(PrecompileHalt::BlobVerifyKzgProofFailed)
        ));
    }

    #[test]
    fn valid_kzg_opening_is_accepted() {
        let commitment = hex!(
            "8f59a8d2a1a625a17f3fea0fe5eb8c896db3764f3185481bc22f91b4aaffcca25f26936857bc3a7c2539ea8ec3a952b7"
        );
        let z = hex!("73eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000000");
        let y = hex!("1522a4a7f34e1ea350ae07c29c96c7e79655aa926122e95fe69fcbd932ca49e9");
        let proof = hex!(
            "a62ad71d14c5719385c0686f1871430475bf3a00f0aa3f7b8dd99a9abc2160744faf0070725e00b60ad9a026a15b1a8c"
        );

        let crypto = CustomCrypto::default();
        assert_eq!(crypto.verify_kzg_proof(&z, &y, &commitment, &proof), Ok(()));
    }
}

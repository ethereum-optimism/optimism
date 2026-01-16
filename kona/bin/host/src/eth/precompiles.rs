//! Accelerated precompile runner for the host program.

use alloy_primitives::{Address, Bytes};
use anyhow::{Result, anyhow};
use revm::precompile::{self, Precompile};

/// List of precompiles that are accelerated by the host program.
pub(crate) const ACCELERATED_PRECOMPILES: &[Precompile] = &[
    precompile::secp256k1::ECRECOVER,          // ecRecover
    precompile::bn254::pair::ISTANBUL,         // ecPairing
    precompile::bls12_381::g1_add::PRECOMPILE, // BLS12-381 G1 Point Addition
    precompile::bls12_381::g1_msm::PRECOMPILE, /* BLS12-381 G1 Point Multi-scalar
                                                * Multiplication */
    precompile::bls12_381::g2_add::PRECOMPILE, // BLS12-381 G2 Point Addition
    precompile::bls12_381::g2_msm::PRECOMPILE, // BLS12-381 G2 Point Multi-scalar Multiplication
    precompile::bls12_381::map_fp2_to_g2::PRECOMPILE, // BLS12-381 FP2 to G2 Point Mapping
    precompile::bls12_381::map_fp_to_g1::PRECOMPILE, // BLS12-381 FP to G1 Point Mapping
    precompile::bls12_381::pairing::PRECOMPILE, // BLS12-381 pairing
    precompile::kzg_point_evaluation::POINT_EVALUATION, // KZG point evaluation
];

/// Executes an accelerated precompile on [revm].
pub(crate) fn execute<T: Into<Bytes>>(address: Address, input: T, gas: u64) -> Result<Vec<u8>> {
    if let Some(precompile) =
        ACCELERATED_PRECOMPILES.iter().find(|precompile| *precompile.address() == address)
    {
        let output = precompile.precompile()(&input.into(), gas)
            .map_err(|e| anyhow!("Failed precompile execution: {e}"))?;

        Ok(output.bytes.into())
    } else {
        anyhow::bail!("Precompile not accelerated");
    }
}

//! Accelerated precompile runner for the host program.

use crate::{HostError, Result};
use alloy_primitives::{Address, Bytes};
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

/// Result of executing an accelerated precompile.
#[derive(Debug, Clone, PartialEq, Eq)]
pub(crate) enum PrecompileExecution {
    /// The EVM precompile call succeeded and returned bytes.
    Success(Vec<u8>),
    /// The EVM precompile call executed but did not succeed.
    Failure(Vec<u8>),
}
impl PrecompileExecution {
    /// Encodes the precompile execution result for the preimage oracle.
    pub(crate) fn into_preimage_value(self) -> Vec<u8> {
        let (status, returndata) = match self {
            Self::Success(returndata) => (0x01, returndata),
            Self::Failure(returndata) => (0x00, returndata),
        };

        let mut result = Vec::with_capacity(1 + returndata.len());
        result.push(status);
        result.extend_from_slice(&returndata);
        result
    }
}

impl From<precompile::PrecompileOutput> for PrecompileExecution {
    fn from(output: precompile::PrecompileOutput) -> Self {
        if output.is_success() {
            Self::Success(output.bytes.into())
        } else {
            Self::Failure(output.bytes.into())
        }
    }
}

/// Executes an accelerated precompile on [revm].
pub(crate) fn execute<T: Into<Bytes>>(
    address: Address,
    input: T,
    gas: u64,
) -> Result<PrecompileExecution> {
    if let Some(precompile) =
        ACCELERATED_PRECOMPILES.iter().find(|precompile| *precompile.address() == address)
    {
        let output = precompile.execute(&input.into(), gas, 0).map_err(
            |e: revm::precompile::PrecompileError| {
                HostError::PrecompileExecutionFailed(e.to_string())
            },
        )?;

        Ok(output.into())
    } else {
        Err(HostError::PrecompileNotAccelerated)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::{Bytes, hex};
    use revm::precompile::{
        PrecompileOutput,
        bn254::{
            PAIR_ELEMENT_LEN,
            pair::{self, ISTANBUL_PAIR_BASE, ISTANBUL_PAIR_PER_POINT},
        },
    };

    const PAIRING_SUCCESS_OUTPUT: [u8; 32] =
        hex!("0000000000000000000000000000000000000000000000000000000000000001");

    #[test]
    fn execute_returns_success_for_empty_successful_bn254_pairing() {
        let output = execute(*pair::ISTANBUL.address(), Vec::<u8>::new(), u64::MAX)
            .expect("empty pairing should execute successfully");

        let bytes = match output {
            PrecompileExecution::Success(bytes) => bytes,
            PrecompileExecution::Failure(_) => panic!("successful pairing must not fail"),
        };
        assert_eq!(bytes.as_slice(), PAIRING_SUCCESS_OUTPUT);

        let encoded = PrecompileExecution::Success(bytes).into_preimage_value();
        assert_eq!(encoded[0], 0x01);
        assert_eq!(&encoded[1..], PAIRING_SUCCESS_OUTPUT);
    }

    #[test]
    fn execute_returns_halt_for_semantic_bn254_pairing_halt() {
        let input = vec![0xffu8; PAIR_ELEMENT_LEN];
        let gas = ISTANBUL_PAIR_BASE + ISTANBUL_PAIR_PER_POINT;

        let native_output = pair::ISTANBUL
            .execute(&Bytes::from(input.clone()), gas, 0)
            .expect("semantic precompile halt is returned as output");
        assert!(native_output.is_halt(), "malformed pair input should halt");

        let host_output =
            execute(*pair::ISTANBUL.address(), input, gas).expect("semantic halt is not fatal");
        let bytes = match host_output {
            PrecompileExecution::Failure(bytes) => bytes,
            PrecompileExecution::Success(_) => panic!("semantic halt must not return success"),
        };
        assert_eq!(bytes.as_slice(), native_output.bytes.as_ref());

        let encoded = PrecompileExecution::Failure(bytes).into_preimage_value();
        assert_eq!(encoded[0], 0x00);
        assert_eq!(&encoded[1..], native_output.bytes.as_ref());
    }

    #[test]
    fn execution_maps_revert_to_failure_status_and_preserves_bytes() {
        let revert_bytes = Bytes::from_static(b"revert");
        let output = PrecompileOutput::revert(12, revert_bytes.clone(), 0);

        match PrecompileExecution::from(output) {
            PrecompileExecution::Failure(bytes) => assert_eq!(bytes, revert_bytes),
            PrecompileExecution::Success(_) => panic!("revert must not return success"),
        }
    }
}

//! On-chain SP1 verifier compatibility.
//!
//! `SP1Verifier.verifyProof` reverts with `WrongVerifierSelector` unless the first four bytes of
//! the proof equal the first four bytes of its `VERIFIER_HASH()`. sp1-sdk prepends
//! `sha256(plonk_vk.bin)[..4]` to every PLONK proof, and `sp1_verifier::PLONK_VK_BYTES` embeds
//! that same file for the linked SDK version, so the hash a verifier must return is known
//! without generating a proof.

use alloy_primitives::{Address, B256};
use sha2::{Digest, Sha256};

/// The `VERIFIER_HASH()` an on-chain SP1 PLONK verifier must return to accept proofs produced
/// by the linked sp1-sdk.
pub fn expected_verifier_hash() -> B256 {
    B256::from(<[u8; 32]>::from(Sha256::digest(*sp1_verifier::PLONK_VK_BYTES)))
}

/// The on-chain verifier implements a different circuit than the linked sp1-sdk proves for.
#[derive(Debug, PartialEq, Eq, thiserror::Error)]
#[error(
    "SP1 verifier {verifier} returns VERIFIER_HASH {actual}; proofs from sp1-sdk circuit \
     {circuit} carry {expected}. Re-pin the verifier address or the SDK."
)]
pub struct VerifierHashMismatch {
    /// The raw SP1 verifier that was queried.
    pub verifier: Address,
    /// `sha256(PLONK_VK_BYTES)` for the linked sp1-sdk.
    pub expected: B256,
    /// `VERIFIER_HASH()` returned by the verifier.
    pub actual: B256,
    /// `sp1_sdk::SP1_CIRCUIT_VERSION` for the linked sp1-sdk.
    pub circuit: &'static str,
}

/// Rejects a verifier whose `VERIFIER_HASH()` differs from the linked SDK's circuit.
pub fn check_verifier_hash(verifier: Address, actual: B256) -> Result<(), VerifierHashMismatch> {
    let expected = expected_verifier_hash();
    if actual == expected {
        return Ok(());
    }
    Err(VerifierHashMismatch {
        verifier,
        expected,
        actual,
        circuit: sp1_sdk::SP1_CIRCUIT_VERSION.trim(),
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn check_verifier_hash_rejects_other_circuit() {
        let verifier = Address::repeat_byte(0x11);
        let actual = B256::repeat_byte(0xaa);
        let err = check_verifier_hash(verifier, actual).unwrap_err();
        assert_eq!(err.verifier, verifier);
        assert_eq!(err.actual, actual);
        assert_eq!(err.expected, expected_verifier_hash());
        assert_eq!(err.circuit, sp1_sdk::SP1_CIRCUIT_VERSION.trim());
    }
}

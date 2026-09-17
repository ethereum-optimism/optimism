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

    /// The release pin op-deployer and `VerifyOPCM` use for the approved raw verifier. The
    /// sepolia integration test in op-deployer holds it to the deployed address; this test
    /// holds it to the linked sp1-sdk.
    const RELEASE_PIN: &str = include_str!(concat!(
        env!("CARGO_MANIFEST_DIR"),
        "/../../../../../op-deployer/pkg/deployer/standard/sp1-verifier.json"
    ));

    #[derive(serde::Deserialize)]
    #[serde(rename_all = "camelCase")]
    struct ReleasePin {
        circuit_version: String,
        plonk_verifier_hash: B256,
    }

    #[test]
    fn sdk_plonk_vk_matches_release_pin() {
        let pin: ReleasePin = serde_json::from_str(RELEASE_PIN).expect("valid release pin");
        let remedy = "sp1-sdk moved to another circuit: update \
             op-deployer/pkg/deployer/standard/sp1-verifier.json with the new VERIFIER_HASH from \
             sp1-contracts and re-pin the verifier address in standard.SP1VerifierFor and \
             VerifyOPCM.s.sol together";
        assert_eq!(sp1_sdk::SP1_CIRCUIT_VERSION.trim(), pin.circuit_version, "{remedy}");
        assert_eq!(expected_verifier_hash(), pin.plonk_verifier_hash, "{remedy}");
    }

    #[test]
    fn check_verifier_hash_accepts_expected() {
        let verifier = Address::repeat_byte(0x11);
        assert_eq!(check_verifier_hash(verifier, expected_verifier_hash()), Ok(()));
    }

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

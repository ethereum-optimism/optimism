//! On-chain SP1 verifier compatibility.
//!
//! `SP1Verifier.verifyProof` reverts with `WrongVerifierSelector` unless the first four bytes of
//! the proof equal the first four bytes of its `VERIFIER_HASH()`. sp1-sdk prepends
//! `sha256(plonk_vk.bin)[..4]` to every PLONK proof, and `sp1_verifier::PLONK_VK_BYTES` embeds
//! that same file for the linked SDK version, so the hash a verifier must return is known
//! without generating a proof.

use alloy_primitives::{Address, B256};
use sha2::{Digest, Sha256};

// `sp1_sdk::SP1Proof` is a re-export of `sp1_verifier::SP1Proof`. This compiles only when the
// `sp1-verifier` crate linked here is the same instance sp1-sdk uses, so `PLONK_VK_BYTES` below
// is the key of the circuit the SDK proves for. A new-major sp1-sdk resolved beside an older
// direct `sp1-verifier` would otherwise pass every test and approve the wrong verifier.
const _: fn(sp1_sdk::SP1Proof) -> sp1_verifier::SP1Proof = std::convert::identity;

/// The `VERIFIER_HASH()` an on-chain SP1 PLONK verifier must return to accept proofs produced
/// by the linked sp1-sdk.
pub(crate) fn expected_verifier_hash() -> B256 {
    B256::from(<[u8; 32]>::from(Sha256::digest(*sp1_verifier::PLONK_VK_BYTES)))
}

/// The on-chain verifier implements a different circuit than the linked sp1-sdk proves for.
#[derive(Debug, PartialEq, Eq, thiserror::Error)]
#[error(
    "SP1 verifier behind adapter {adapter} returns VERIFIER_HASH {actual}; proofs from sp1-sdk \
     circuit {circuit} carry {expected}. Re-pin the verifier address or the SDK."
)]
pub(crate) struct VerifierHashMismatch {
    /// The `SP1PlonkAdapter` (a game's or the registered args' `verifier`) whose wrapped raw
    /// verifier was queried.
    pub adapter: Address,
    /// `sha256(PLONK_VK_BYTES)` for the linked sp1-sdk.
    pub expected: B256,
    /// `VERIFIER_HASH()` returned by the raw verifier.
    pub actual: B256,
    /// `sp1_sdk::SP1_CIRCUIT_VERSION` for the linked sp1-sdk.
    pub circuit: &'static str,
}

/// Rejects a verifier whose `VERIFIER_HASH()` differs from the linked SDK's circuit.
pub(crate) fn check_verifier_hash(
    adapter: Address,
    actual: B256,
) -> Result<(), VerifierHashMismatch> {
    let expected = expected_verifier_hash();
    if actual == expected {
        return Ok(());
    }
    Err(VerifierHashMismatch {
        adapter,
        expected,
        actual,
        circuit: sp1_sdk::SP1_CIRCUIT_VERSION.trim(),
    })
}

#[cfg(test)]
mod tests {
    use std::collections::BTreeMap;

    use super::*;

    /// Holds the linked sp1-sdk to the release-approved verifier on every chain in
    /// op-deployer's `sp1-verifier.json` (the file behind `standard.SP1VerifierHashFor`).
    /// Bumping sp1-sdk to another circuit fails here until the verifier address and that file
    /// move with it.
    #[test]
    fn sdk_circuit_matches_release_pin() {
        let pins: BTreeMap<String, B256> = serde_json::from_str(include_str!(concat!(
            env!("CARGO_MANIFEST_DIR"),
            "/../../../../../op-deployer/pkg/deployer/standard/sp1-verifier.json"
        )))
        .expect("valid release pin");
        assert!(!pins.is_empty());
        for (chain, hash) in pins {
            assert_eq!(
                expected_verifier_hash(),
                hash,
                "sp1-sdk moved to another circuit than chain {chain}'s release verifier; update \
                 op-deployer/pkg/deployer/standard/sp1-verifier.json and standard.SP1VerifierFor \
                 together"
            );
        }
    }

    #[test]
    fn check_verifier_hash_rejects_other_circuit() {
        let adapter = Address::repeat_byte(0x11);
        let actual = B256::repeat_byte(0xaa);
        let err = check_verifier_hash(adapter, actual).unwrap_err();
        assert_eq!(err.adapter, adapter);
        assert_eq!(err.actual, actual);
        assert_eq!(err.expected, expected_verifier_hash());
        assert_eq!(err.circuit, sp1_sdk::SP1_CIRCUIT_VERSION.trim());
    }
}

//! SP1 Groth16 verification of the private-projection proof (§F.2, §F.4), behind
//! the `sp1-projection-verifier` feature. Twin of Go `projection/sp1groth16.Verify`.

use alloy_primitives::B256;
use sha2::{Digest, Sha256};
use sp1_verifier::Groth16Verifier;

use super::{GROTH16_PROOF_LEN, ProjectionError, is_canonical_scalar, public_values_digest};

/// Pinned SP1 Groth16 circuit parameters: the gnark verifying key and the recursion vk root.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Circuit<'a> {
    vk: &'a [u8],
    vk_root: [u8; 32],
}

impl<'a> Circuit<'a> {
    /// Circuit from explicit parameters (used by fixture tests of other SP1 versions).
    pub const fn new(vk: &'a [u8], vk_root: [u8; 32]) -> Self {
        Self { vk, vk_root }
    }

    /// Gnark verifying-key bytes.
    pub const fn vk(&self) -> &'a [u8] {
        self.vk
    }

    /// Recursion verifying-key root embedded in every proof of this circuit.
    pub const fn vk_root(&self) -> [u8; 32] {
        self.vk_root
    }
}

/// The SP1 v6.1.0 Groth16 circuit shipped with `sp1-verifier` 6.8.0, pinned by
/// `sp1-private-projection-v1`. A function, not a `const`: `sp1-verifier` exposes the
/// key bytes through `lazy_static`.
pub fn circuit_v6_1_0() -> Circuit<'static> {
    Circuit { vk: *sp1_verifier::GROTH16_VK_BYTES, vk_root: *sp1_verifier::VK_ROOT_BYTES }
}

/// Verify a 356-byte SP1 Groth16 proof (`prefix ‖ exit ‖ vk_root ‖ nonce ‖ gnark proof`)
/// of `program_vkey` committing exactly `public_values`, under circuit `c`. Only the
/// sha256 committed-values digest is accepted.
pub fn verify_groth16(
    c: &Circuit<'_>,
    proof356: &[u8],
    program_vkey: B256,
    public_values: &[u8],
) -> Result<(), ProjectionError> {
    if proof356.len() != GROTH16_PROOF_LEN {
        return Err(ProjectionError("groth16 proof length"));
    }
    let prefix = Sha256::digest(c.vk);
    if proof356[..4] != prefix[..4] {
        return Err(ProjectionError("groth16 circuit prefix"));
    }
    let word = |i: usize| -> [u8; 32] { proof356[4 + 32 * i..36 + 32 * i].try_into().unwrap() };
    let (exit, vk_root, nonce) = (word(0), word(1), word(2));
    if exit != [0u8; 32] {
        return Err(ProjectionError("groth16 nonzero exit code"));
    }
    if vk_root != c.vk_root {
        return Err(ProjectionError("groth16 vk root"));
    }
    let inputs = [program_vkey.0, public_values_digest(public_values).0, exit, vk_root, nonce];
    if !inputs.iter().all(is_canonical_scalar) {
        return Err(ProjectionError("groth16 noncanonical public input"));
    }
    Groth16Verifier::verify_gnark_proof(&proof356[100..], &inputs, c.vk)
        .map_err(|_| ProjectionError("groth16 proof"))
}

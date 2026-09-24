//! The claim `proof` envelope of `sp1-private-projection-v1` (§F.2). Twin of Go
//! `projection.{Envelope, DecodeEnvelope, EncodeEnvelope}`.

use alloc::vec::Vec;
use alloy_primitives::B256;

use super::{PUBLIC_VALUES_LEN, ProjectionError, public_values_digest};

/// Envelope format version.
pub const ENVELOPE_VERSION: u8 = 0x01;
/// Byte length of an SP1 Groth16 proof (`SP1ProofWithPublicValues::bytes()`).
pub const GROTH16_PROOF_LEN: usize = 356;
/// Byte length of an SP1 mock proof: five 32-byte public-input words.
pub const MOCK_PROOF_LEN: usize = 160;

/// Proof system of an envelope.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[repr(u8)]
pub enum EnvelopeKind {
    /// SP1 Groth16 over the circuit pinned by the verifier ID.
    Groth16 = 0x01,
    /// SP1 mock proof (test-gated).
    Mock = 0x02,
}

impl EnvelopeKind {
    const fn proof_len(self) -> usize {
        match self {
            Self::Groth16 => GROTH16_PROOF_LEN,
            Self::Mock => MOCK_PROOF_LEN,
        }
    }
}

/// Decoded envelope: `version ‖ kind ‖ u16be(L) ‖ proof[L] ‖ public_values[672]`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Envelope {
    /// Proof system.
    pub kind: EnvelopeKind,
    /// Proof bytes, exactly the kind's length.
    pub proof: Vec<u8>,
    /// The committed `PublicValuesV1`.
    pub public_values: [u8; PUBLIC_VALUES_LEN],
}

/// Strict decoding: known version and kind, the kind's exact proof length, no
/// trailing bytes.
pub fn decode_envelope(b: &[u8]) -> Result<Envelope, ProjectionError> {
    if b.len() < 4 {
        return Err(ProjectionError("truncated proof envelope"));
    }
    if b[0] != ENVELOPE_VERSION {
        return Err(ProjectionError("unknown proof envelope version"));
    }
    let kind = match b[1] {
        0x01 => EnvelopeKind::Groth16,
        0x02 => EnvelopeKind::Mock,
        _ => return Err(ProjectionError("unknown proof envelope kind")),
    };
    let len = usize::from(u16::from_be_bytes([b[2], b[3]]));
    if len != kind.proof_len() {
        return Err(ProjectionError("proof length for envelope kind"));
    }
    if b.len() != 4 + len + PUBLIC_VALUES_LEN {
        return Err(ProjectionError("proof envelope length"));
    }
    let mut public_values = [0u8; PUBLIC_VALUES_LEN];
    public_values.copy_from_slice(&b[4 + len..]);
    Ok(Envelope { kind, proof: b[4..4 + len].to_vec(), public_values })
}

/// Encode an envelope. The caller supplies a proof of the kind's length.
pub fn encode_envelope(e: &Envelope) -> Vec<u8> {
    let mut out = Vec::with_capacity(4 + e.proof.len() + PUBLIC_VALUES_LEN);
    out.push(ENVELOPE_VERSION);
    out.push(e.kind as u8);
    out.extend_from_slice(&(e.proof.len() as u16).to_be_bytes());
    out.extend_from_slice(&e.proof);
    out.extend_from_slice(&e.public_values);
    out
}

/// The mock proof words `[vkeyHash, digest, exit, vkRoot, nonce]` the host's
/// `native-mock` prover emits for `public_values`.
pub fn mock_proof(program_vkey: B256, public_values: &[u8]) -> Vec<u8> {
    let mut out = Vec::with_capacity(MOCK_PROOF_LEN);
    out.extend_from_slice(program_vkey.as_slice());
    out.extend_from_slice(public_values_digest(public_values).as_slice());
    out.resize(MOCK_PROOF_LEN, 0);
    out
}

/// Check a mock proof against the pinned vkey and the envelope's public values:
/// `words[0] == program_vkey`, `words[1] == digest`, `words[2..5] == 0`.
pub fn check_mock_proof(
    proof: &[u8],
    program_vkey: B256,
    public_values: &[u8],
) -> Result<(), ProjectionError> {
    if proof.len() != MOCK_PROOF_LEN {
        return Err(ProjectionError("mock proof length"));
    }
    if proof[..32] != program_vkey[..] {
        return Err(ProjectionError("mock proof vkey"));
    }
    if proof[32..64] != public_values_digest(public_values)[..] {
        return Err(ProjectionError("mock proof digest"));
    }
    if proof[64..].iter().any(|b| *b != 0) {
        return Err(ProjectionError("mock proof exit, root or nonce"));
    }
    Ok(())
}

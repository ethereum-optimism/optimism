//! Test-only: EIP-2537 reference oracle, corpus, and zkVM differential-check support for
//! kona-sp1's `CustomCrypto`.
//!
//! - [`ArkworksCrypto`]: vendored revm-precompile arkworks EIP-2537 backend (see [`arkworks`]).
//! - [`corpus::full_corpus`]: the deterministic case list run on the host and inside the zkVM.
//! - [`BlsCall`] / [`run_call`]: one call model shared by the host tests, the zkVM guest, and the
//!   zkVM runner.

pub mod arkworks;
pub mod corpus;

pub use arkworks::ArkworksCrypto;

use revm::precompile::{
    Crypto, PrecompileHalt,
    bls12_381::{G1Point, G1PointScalar, G2Point, G2PointScalar},
};
use sha2::{Digest, Sha256};
use std::borrow::Cow;

/// One EIP-2537 [`Crypto`] call with its inputs.
#[derive(Clone, Debug, PartialEq, Eq, rkyv::Archive, rkyv::Serialize, rkyv::Deserialize)]
pub enum BlsCall {
    /// `bls12_381_g1_add(a, b)`.
    G1Add(G1Point, G1Point),
    /// `bls12_381_g1_msm`; `None` items are padding errors (see [`run_call`]).
    G1Msm(Vec<Option<G1PointScalar>>),
    /// `bls12_381_g2_add(a, b)`.
    G2Add(G2Point, G2Point),
    /// `bls12_381_g2_msm`; `None` items are padding errors (see [`run_call`]).
    G2Msm(Vec<Option<G2PointScalar>>),
    /// `bls12_381_pairing_check(pairs)`.
    Pairing(Vec<(G1Point, G2Point)>),
    /// `bls12_381_fp_to_g1(fp)`.
    MapFpToG1([u8; 48]),
    /// `bls12_381_fp2_to_g2((c0, c1))`.
    MapFp2ToG2([u8; 48], [u8; 48]),
}

impl BlsCall {
    /// Short operation name, used in labels and zkVM cycle-tracker names.
    pub const fn op_name(&self) -> &'static str {
        match self {
            Self::G1Add(..) => "g1-add",
            Self::G1Msm(_) => "g1-msm",
            Self::G2Add(..) => "g2-add",
            Self::G2Msm(_) => "g2-msm",
            Self::Pairing(_) => "pairing",
            Self::MapFpToG1(_) => "map-fp-to-g1",
            Self::MapFp2ToG2(..) => "map-fp2-to-g2",
        }
    }
}

/// A labelled [`BlsCall`].
#[derive(Clone, Debug, PartialEq, Eq, rkyv::Archive, rkyv::Serialize, rkyv::Deserialize)]
pub struct BlsCase {
    /// Unique label: `"<op>/<group>/<n>"`, or `"<op>/torsion/[g1|g2/]<point>/<what>"` for torsion
    /// cases.
    pub label: String,
    /// The call.
    pub call: BlsCall,
}

/// Runs `call` against `crypto`. MSM `None` items yield
/// [`PrecompileHalt::Bls12381FpPaddingInvalid`], the error revm's lazy MSM iterator produces for
/// bad padding; the pairing result is encoded as a single byte.
pub fn run_call(crypto: &dyn Crypto, call: &BlsCall) -> Result<Vec<u8>, PrecompileHalt> {
    match call {
        BlsCall::G1Add(a, b) => crypto.bls12_381_g1_add(*a, *b).map(|r| r.to_vec()),
        BlsCall::G1Msm(items) => crypto
            .bls12_381_g1_msm(
                &mut items.iter().map(|i| (*i).ok_or(PrecompileHalt::Bls12381FpPaddingInvalid)),
            )
            .map(|r| r.to_vec()),
        BlsCall::G2Add(a, b) => crypto.bls12_381_g2_add(*a, *b).map(|r| r.to_vec()),
        BlsCall::G2Msm(items) => crypto
            .bls12_381_g2_msm(
                &mut items.iter().map(|i| (*i).ok_or(PrecompileHalt::Bls12381FpPaddingInvalid)),
            )
            .map(|r| r.to_vec()),
        BlsCall::Pairing(pairs) => crypto.bls12_381_pairing_check(pairs).map(|b| vec![u8::from(b)]),
        BlsCall::MapFpToG1(fp) => crypto.bls12_381_fp_to_g1(fp).map(|r| r.to_vec()),
        BlsCall::MapFp2ToG2(c0, c1) => crypto.bls12_381_fp2_to_g2((*c0, *c1)).map(|r| r.to_vec()),
    }
}

/// SHA-256 over all outcomes in order. `Ok(b)` contributes `0x00 ‖ len_be32 ‖ b`; `Err(h)`
/// contributes `0x01 ‖ len_be32 ‖ format!("{h:?}")`.
fn outcomes_digest(outcomes: &[Result<Vec<u8>, PrecompileHalt>]) -> [u8; 32] {
    let mut hasher = Sha256::new();
    for outcome in outcomes {
        let (tag, body): (u8, Cow<'_, [u8]>) = match outcome {
            Ok(bytes) => (0, Cow::Borrowed(bytes)),
            Err(halt) => (1, Cow::Owned(format!("{halt:?}").into_bytes())),
        };
        hasher.update([tag]);
        hasher.update((body.len() as u32).to_be_bytes());
        hasher.update(&body);
    }
    hasher.finalize().into()
}

/// The check guest's public values: `case_count_le64 ‖ outcomes_digest(outcomes)`.
pub fn public_values(outcomes: &[Result<Vec<u8>, PrecompileHalt>]) -> [u8; 40] {
    let mut out = [0u8; 40];
    out[..8].copy_from_slice(&(outcomes.len() as u64).to_le_bytes());
    out[8..].copy_from_slice(&outcomes_digest(outcomes));
    out
}

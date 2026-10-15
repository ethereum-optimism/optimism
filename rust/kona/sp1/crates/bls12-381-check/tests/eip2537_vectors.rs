//! Official EIP-2537 vectors run through revm's precompile entry points with `CustomCrypto`
//! installed as the global crypto provider.
//!
//! Vectors (CC0) are vendored from
//! <https://github.com/ethereum/EIPs/tree/faaa35fff856a109c40d291e612447fadc2095fe/assets/eip-2537>.
//! All files are verbatim except `msm_G1_bls.json` and `msm_G2_bls.json` (10 MB together), which
//! are filtered with:
//!
//! ```sh
//! jq -c '[.[] | select((.Name | test("discount_table") | not) or (.Name | test("discount_table_(2|16)$")))]'
//! ```
//!
//! The dropped `discount_table_k` vectors (k ∉ {2, 16}) are k × (identity, scalar 0) and pin only
//! the per-k MSM gas, which revm computes before calling `Crypto`.
//!
//! This binary holds exactly one test: revm's global `Crypto` is a `OnceLock`, so
//! `CustomCrypto` must be installed once, before any precompile runs in the process.

use alloy_primitives::hex;
use kona_sp1_client_utils::precompiles::CustomCrypto;
use revm::precompile::{
    EthPrecompileResult, PrecompileHalt,
    bls12_381::{g1_add, g1_msm, g2_add, g2_msm, map_fp_to_g1, map_fp2_to_g2, pairing},
    install_crypto,
};
use serde::Deserialize;

#[derive(Deserialize)]
#[serde(rename_all = "PascalCase")]
struct SuccessVector {
    input: String,
    expected: String,
    gas: u64,
    name: String,
}

#[derive(Deserialize)]
#[serde(rename_all = "PascalCase")]
struct FailVector {
    input: String,
    expected_error: String,
    name: String,
}

#[derive(Clone, Copy, Debug)]
enum Op {
    G1Add,
    G2Add,
    G1Msm,
    G2Msm,
    Pairing,
    MapFpToG1,
    MapFp2ToG2,
}

impl Op {
    fn run(self, input: &[u8]) -> EthPrecompileResult {
        let f = match self {
            Self::G1Add => g1_add::g1_add,
            Self::G2Add => g2_add::g2_add,
            Self::G1Msm => g1_msm::g1_msm,
            Self::G2Msm => g2_msm::g2_msm,
            Self::Pairing => pairing::pairing,
            Self::MapFpToG1 => map_fp_to_g1::map_fp_to_g1,
            Self::MapFp2ToG2 => map_fp2_to_g2::map_fp2_to_g2,
        };
        f(input, u64::MAX)
    }

    const fn length_halt(self) -> PrecompileHalt {
        match self {
            Self::G1Add => PrecompileHalt::Bls12381G1AddInputLength,
            Self::G2Add => PrecompileHalt::Bls12381G2AddInputLength,
            Self::G1Msm => PrecompileHalt::Bls12381G1MsmInputLength,
            Self::G2Msm => PrecompileHalt::Bls12381G2MsmInputLength,
            Self::Pairing => PrecompileHalt::Bls12381PairingInputLength,
            Self::MapFpToG1 => PrecompileHalt::Bls12381MapFpToG1InputLength,
            Self::MapFp2ToG2 => PrecompileHalt::Bls12381MapFp2ToG2InputLength,
        }
    }

    /// Whether a "not on curve" error refers to G2.
    fn not_on_curve_is_g2(self, name: &str) -> bool {
        match self {
            Self::G1Add | Self::G1Msm | Self::MapFpToG1 => false,
            Self::G2Add | Self::G2Msm | Self::MapFp2ToG2 => true,
            Self::Pairing => {
                name.contains("G2_not_on_curve") ||
                    name.contains("G2_in_correct_subgroup_invalid_curve")
            }
        }
    }
}

fn expected_halt(op: Op, name: &str, msg: &str) -> PrecompileHalt {
    match msg {
        "invalid input length" => op.length_halt(),
        "invalid field element top bytes" => PrecompileHalt::Bls12381FpPaddingInvalid,
        "invalid fp.Element encoding" => PrecompileHalt::NonCanonicalFp,
        "g1 point is not in the correct subgroup" => PrecompileHalt::Bls12381G1NotInSubgroup,
        "g2 point is not in the correct subgroup" => PrecompileHalt::Bls12381G2NotInSubgroup,
        "invalid point: not on curve" if op.not_on_curve_is_g2(name) => {
            PrecompileHalt::Bls12381G2NotOnCurve
        }
        "invalid point: not on curve" => PrecompileHalt::Bls12381G1NotOnCurve,
        other => panic!("{name}: unmapped expected error {other:?}"),
    }
}

macro_rules! vectors {
    ($file:literal) => {
        include_str!(concat!(env!("CARGO_MANIFEST_DIR"), "/testdata/eip2537/", $file, ".json"))
    };
}

#[test]
fn eip2537_vectors_pass_with_custom_crypto() {
    assert!(
        install_crypto(CustomCrypto::default()),
        "CustomCrypto must be the first provider installed in this process"
    );

    let success: [(Op, &str); 9] = [
        (Op::G1Add, vectors!("add_G1_bls")),
        (Op::G2Add, vectors!("add_G2_bls")),
        (Op::G1Msm, vectors!("mul_G1_bls")),
        (Op::G1Msm, vectors!("msm_G1_bls")),
        (Op::G2Msm, vectors!("mul_G2_bls")),
        (Op::G2Msm, vectors!("msm_G2_bls")),
        (Op::Pairing, vectors!("pairing_check_bls")),
        (Op::MapFpToG1, vectors!("map_fp_to_G1_bls")),
        (Op::MapFp2ToG2, vectors!("map_fp2_to_G2_bls")),
    ];
    let mut ran = 0usize;
    for (op, json) in success {
        let vectors: Vec<SuccessVector> = serde_json::from_str(json).expect("success vectors");
        assert!(!vectors.is_empty());
        for v in vectors {
            let out = op
                .run(&hex::decode(&v.input).expect("hex input"))
                .unwrap_or_else(|e| panic!("{}: unexpected halt {e:?}", v.name));
            assert_eq!(hex::encode(&out.bytes), v.expected.to_lowercase(), "{}", v.name);
            assert_eq!(out.gas_used, v.gas, "{}: gas", v.name);
            ran += 1;
        }
    }

    let fail: [(Op, &str); 9] = [
        (Op::G1Add, vectors!("fail-add_G1_bls")),
        (Op::G2Add, vectors!("fail-add_G2_bls")),
        (Op::G1Msm, vectors!("fail-mul_G1_bls")),
        (Op::G1Msm, vectors!("fail-msm_G1_bls")),
        (Op::G2Msm, vectors!("fail-mul_G2_bls")),
        (Op::G2Msm, vectors!("fail-msm_G2_bls")),
        (Op::Pairing, vectors!("fail-pairing_check_bls")),
        (Op::MapFpToG1, vectors!("fail-map_fp_to_G1_bls")),
        (Op::MapFp2ToG2, vectors!("fail-map_fp2_to_G2_bls")),
    ];
    for (op, json) in fail {
        let vectors: Vec<FailVector> = serde_json::from_str(json).expect("fail vectors");
        assert!(!vectors.is_empty());
        for v in vectors {
            let got = op.run(&hex::decode(&v.input).expect("hex input")).map(|o| o.bytes);
            let want = expected_halt(op, &v.name, &v.expected_error);
            assert_eq!(got, Err(want), "{}", v.name);
            ran += 1;
        }
    }
    assert_eq!(ran, 179, "every vendored vector must run");
}

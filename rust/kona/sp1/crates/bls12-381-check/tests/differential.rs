//! Three-way differential tests: kona-sp1's `CustomCrypto` vs the vendored arkworks backend (the
//! zkVM guest's `DefaultCrypto`) vs revm's `DefaultCrypto` in this build (arkworks under `-p`,
//! blst under `--workspace` feature unification).

use ark_bls12_381_0_6::{Fq, G1Affine, G2Affine, g1};
use ark_ec_0_6::{
    AffineRepr,
    hashing::{
        curve_maps::{swu::SWUMap, wb::WBConfig},
        map_to_curve_hasher::MapToCurve,
    },
    short_weierstrass::Projective,
};
use ark_ff_0_6::{One, PrimeField, Zero};
use kona_sp1_bls12_381_check::{
    ArkworksCrypto, BlsCall, BlsCase,
    corpus::{self, ISO_KERNEL_U},
    run_call,
};
use kona_sp1_client_utils::precompiles::CustomCrypto;
use revm::precompile::{DefaultCrypto, PrecompileHalt};

/// `call` run through `CustomCrypto`, the vendored arkworks backend, and `DefaultCrypto`.
fn outcomes(
    custom: &CustomCrypto,
    call: &BlsCall,
) -> [(&'static str, Result<Vec<u8>, PrecompileHalt>); 3] {
    [
        ("CustomCrypto", run_call(custom, call)),
        ("arkworks", run_call(&ArkworksCrypto, call)),
        ("DefaultCrypto", run_call(&DefaultCrypto, call)),
    ]
}

fn assert_agree(custom: &CustomCrypto, case: &BlsCase) {
    let label = &case.label;
    let [(_, ours), (_, arkworks), (_, default)] = outcomes(custom, &case.call);
    assert_eq!(ours, arkworks, "{label}: CustomCrypto vs arkworks");
    assert_eq!(arkworks, default, "{label}: arkworks vs DefaultCrypto");
}

fn check_all(cases: Vec<BlsCase>) {
    assert!(!cases.is_empty());
    let custom = CustomCrypto::default();
    for case in &cases {
        assert_agree(&custom, case);
    }
}

#[test]
fn g1_add_matches_references() {
    check_all(corpus::g1_add_cases());
}

#[test]
fn g2_add_matches_references() {
    check_all(corpus::g2_add_cases());
}

#[test]
fn g1_msm_matches_references() {
    check_all(corpus::g1_msm_cases());
}

#[test]
fn g2_msm_matches_references() {
    check_all(corpus::g2_msm_cases());
}

#[test]
fn pairing_check_matches_references() {
    check_all(corpus::pairing_cases());
}

#[test]
fn map_fp_to_g1_matches_references() {
    check_all(corpus::map_fp_to_g1_cases());
}

#[test]
fn map_fp2_to_g2_matches_references() {
    check_all(corpus::map_fp2_to_g2_cases());
}

/// Expected outcome of a known-answer call.
enum Expect {
    Exact(Result<Vec<u8>, PrecompileHalt>),
    /// Any `Ok` output (the exact bytes are covered by the three-way agreement).
    Ok,
}

#[test]
fn known_answers() {
    use corpus::{
        ZCASH_INFINITY, ZERO_G1, ZERO_G2, be32, enc_fq, enc_g1, enc_g2, g1_not_on_curve,
        g1_off_subgroup, g1_order3, g1_sub, g2_not_on_curve, g2_off_subgroup, p_bytes, r_bytes,
        r_minus_1_bytes, set_g1, set_g2,
    };
    use revm::precompile::PrecompileHalt::{
        Bls12381FpPaddingInvalid, Bls12381G1NotInSubgroup, Bls12381G1NotOnCurve,
        Bls12381G2NotInSubgroup, Bls12381G2NotOnCurve, NonCanonicalFp,
    };

    let p = p_bytes();
    let g = enc_g1(G1Affine::generator());
    let neg_g = enc_g1(-G1Affine::generator());
    let g2 = enc_g2(G2Affine::generator());
    let p1 = enc_g1(g1_sub("known-answers/P", 1));
    let n1 = enc_g1(g1_off_subgroup("known-answers/N1", 0));
    let n2 = enc_g2(g2_off_subgroup("known-answers/N2", 0));
    let noc1 = g1_not_on_curve(10);
    let noc2 = g2_not_on_curve(10);
    let one_fp = enc_fq(Fq::one());
    let zeros = |n: usize| Expect::Exact(Ok(vec![0u8; n]));
    let halt = |h: PrecompileHalt| Expect::Exact(Err(h));
    let bit = |b: u8| Expect::Exact(Ok(vec![b]));

    let mut table: Vec<(&str, BlsCall, Expect)> = vec![
        ("g1 identity + identity", BlsCall::G1Add(ZERO_G1, ZERO_G1), zeros(96)),
        ("G + (-G)", BlsCall::G1Add(g, neg_g), zeros(96)),
        (
            "a not on curve beats b.x = p",
            BlsCall::G1Add(noc1, set_g1(g, 0, p)),
            halt(Bls12381G1NotOnCurve),
        ),
        (
            "a.y = p beats b not on curve",
            BlsCall::G1Add(set_g1(g, 1, p), noc1),
            halt(NonCanonicalFp),
        ),
        (
            "(0,2) + G: ADD has no subgroup check",
            BlsCall::G1Add(enc_g1(g1_order3(true)), g),
            Expect::Ok,
        ),
        ("a = (p, 0)", BlsCall::G1Add((p, [0u8; 48]), g), halt(NonCanonicalFp)),
        ("a = (0x40‖0, 0)", BlsCall::G1Add((ZCASH_INFINITY, [0u8; 48]), g), halt(NonCanonicalFp)),
        (
            "g2 a = (0, 0x40‖0, 0, 0)",
            BlsCall::G2Add(set_g2(ZERO_G2, 1, ZCASH_INFINITY), g2),
            halt(NonCanonicalFp),
        ),
        (
            "g2 (0, 0, 1, 0) + identity: a lone zero x is not the identity",
            BlsCall::G2Add(set_g2(ZERO_G2, 2, one_fp), ZERO_G2),
            halt(Bls12381G2NotOnCurve),
        ),
        ("g1 msm empty", BlsCall::G1Msm(vec![]), zeros(96)),
        ("[(N1, 0)]", BlsCall::G1Msm(vec![Some((n1, be32(0)))]), halt(Bls12381G1NotInSubgroup)),
        (
            "[Some(N1, 1), None]",
            BlsCall::G1Msm(vec![Some((n1, be32(1))), None]),
            halt(Bls12381G1NotInSubgroup),
        ),
        (
            "[None, Some(N1, 1)]",
            BlsCall::G1Msm(vec![None, Some((n1, be32(1)))]),
            halt(Bls12381FpPaddingInvalid),
        ),
        ("[(P1, r)]", BlsCall::G1Msm(vec![Some((p1, r_bytes()))]), zeros(96)),
        ("pairing []", BlsCall::Pairing(vec![]), bit(1)),
        ("[(G, G2), (-G, G2)]", BlsCall::Pairing(vec![(g, g2), (neg_g, g2)]), bit(1)),
        ("[(G, G2)]", BlsCall::Pairing(vec![(g, g2)]), bit(0)),
        ("[(0, N2)]", BlsCall::Pairing(vec![(ZERO_G1, n2)]), halt(Bls12381G2NotInSubgroup)),
        (
            "[(N1, G2 x0 = p)]: G1 subgroup check precedes G2 decoding",
            BlsCall::Pairing(vec![(n1, set_g2(g2, 0, p))]),
            halt(Bls12381G1NotInSubgroup),
        ),
        (
            "[(N1, G2 not on curve)]",
            BlsCall::Pairing(vec![(n1, noc2)]),
            halt(Bls12381G1NotInSubgroup),
        ),
        (
            "[(G, N2), (x = p, G2)]",
            BlsCall::Pairing(vec![(g, n2), (set_g1(g, 0, p), g2)]),
            halt(Bls12381G2NotInSubgroup),
        ),
        (
            "[(G1 not on curve, G2 not on curve)]",
            BlsCall::Pairing(vec![(noc1, noc2)]),
            halt(Bls12381G1NotOnCurve),
        ),
        ("map u = p", BlsCall::MapFpToG1(p), halt(NonCanonicalFp)),
        ("map fp2 (1, p)", BlsCall::MapFp2ToG2(one_fp, p), halt(NonCanonicalFp)),
        (
            "[(P1, 1), (P1, r - 1)]",
            BlsCall::G1Msm(vec![Some((p1, be32(1))), Some((p1, r_minus_1_bytes()))]),
            zeros(96),
        ),
    ];
    for u in ISO_KERNEL_U {
        table.push(("iso-kernel u", BlsCall::MapFpToG1(u), zeros(96)));
    }

    let custom = CustomCrypto::default();
    for (name, call, expect) in table {
        let outcomes = outcomes(&custom, &call);
        for (imp, got) in &outcomes {
            match &expect {
                Expect::Exact(want) => assert_eq!(got, want, "{name}: {imp}"),
                Expect::Ok => assert!(got.is_ok(), "{name}: {imp} returned {got:?}"),
            }
        }
        assert_eq!(outcomes[0].1, outcomes[1].1, "{name}: CustomCrypto vs arkworks");
        assert_eq!(outcomes[1].1, outcomes[2].1, "{name}: arkworks vs DefaultCrypto");
    }
}

/// Every torsion/mixed point passes ADD and fails MSM (any scalar) and pairing (any partner) with
/// `NotInSubgroup`, in all three implementations.
#[test]
fn torsion_points_fail_every_subgroup_check() {
    use corpus::{ZERO_G1, ZERO_G2, be32, enc_g1, enc_g2};
    use revm::precompile::PrecompileHalt::{Bls12381G1NotInSubgroup, Bls12381G2NotInSubgroup};

    let custom = CustomCrypto::default();
    let check = |name: &str, call: BlsCall, want: Option<PrecompileHalt>| {
        for (imp, got) in outcomes(&custom, &call) {
            match &want {
                Some(halt) => assert_eq!(got, Err(halt.clone()), "{name}: {imp}"),
                None => assert!(got.is_ok(), "{name}: {imp} returned {got:?}"),
            }
        }
    };
    let (g1, g2) = (enc_g1(G1Affine::generator()), enc_g2(G2Affine::generator()));

    let g1_points = corpus::g1_torsion_points();
    assert_eq!(g1_points.len(), 9 + 4 + 4 + 5, "G1 torsion coverage shrank");
    for (name, t) in g1_points {
        let t = enc_g1(t);
        check(&name, BlsCall::G1Add(t, g1), None);
        for s in [0, 1] {
            check(&name, BlsCall::G1Msm(vec![Some((t, be32(s)))]), Some(Bls12381G1NotInSubgroup));
        }
        check(&name, BlsCall::Pairing(vec![(t, g2)]), Some(Bls12381G1NotInSubgroup));
        check(&name, BlsCall::Pairing(vec![(t, ZERO_G2)]), Some(Bls12381G1NotInSubgroup));
    }

    let g2_points = corpus::g2_torsion_points();
    assert_eq!(g2_points.len(), 8 + 2 + 5 + 6, "G2 torsion coverage shrank");
    for (name, t) in g2_points {
        let t = enc_g2(t);
        check(&name, BlsCall::G2Add(t, g2), None);
        for s in [0, 1] {
            check(&name, BlsCall::G2Msm(vec![Some((t, be32(s)))]), Some(Bls12381G2NotInSubgroup));
        }
        check(&name, BlsCall::Pairing(vec![(g1, t)]), Some(Bls12381G2NotInSubgroup));
        check(&name, BlsCall::Pairing(vec![(ZERO_G1, t)]), Some(Bls12381G2NotInSubgroup));
    }
}

/// Each `ISO_KERNEL_U` maps under arkworks SSWU to a root of the 11-isogeny x-denominator.
#[test]
fn iso_kernel_u_hits_isogeny_kernel() {
    type Iso = <g1::Config as WBConfig>::IsogenousCurve;
    let den = <g1::Config as WBConfig>::ISOGENY_MAP.x_map_denominator;
    for u in ISO_KERNEL_U {
        let u_fq = Fq::from_be_bytes_mod_order(&u);
        assert_eq!(corpus::enc_fq(u_fq), u, "ISO_KERNEL_U entries must be canonical");
        let point = <SWUMap<Iso> as MapToCurve<Projective<Iso>>>::map_to_curve(u_fq)
            .expect("SSWU is total");
        let x = point.x;
        let value = den.iter().rev().fold(Fq::zero(), |acc, c| acc * x + c);
        assert!(value.is_zero(), "u = {u:02x?} does not reach the isogeny kernel");
    }
}

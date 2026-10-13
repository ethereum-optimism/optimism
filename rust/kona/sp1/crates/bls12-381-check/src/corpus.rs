//! Deterministic EIP-2537 case corpus. No RNG crates: every "seeded" value is derived from
//! SHA-256 over a label (see `seed`), and every constant documents its source.
//!
//! Cases are built with arkworks (ark 0.6, the version revm-precompile uses), never with the
//! implementation under test.

use crate::{BlsCall, BlsCase};
use alloy_primitives::{U256, aliases::U384, hex};
use ark_bls12_381_0_6::{Fq, Fq2, Fr, G1Affine, G1Projective, G2Affine, g1, g2};
use ark_ec_0_6::{
    AffineRepr, CurveGroup,
    hashing::curve_maps::{swu::SWUConfig, wb::WBConfig},
    short_weierstrass::{Affine, Projective, SWCurveConfig},
};
use ark_ff_0_6::{BigInteger, Field, One, PrimeField, Zero};
use op_revm::precompiles::bls12_381::{
    ISTHMUS_G1_MSM_MAX_INPUT_SIZE, ISTHMUS_G2_MSM_MAX_INPUT_SIZE, ISTHMUS_PAIRING_MAX_INPUT_SIZE,
    JOVIAN_G1_MSM_MAX_INPUT_SIZE, JOVIAN_G2_MSM_MAX_INPUT_SIZE, JOVIAN_PAIRING_MAX_INPUT_SIZE,
};
use revm::precompile::{
    bls12_381::{G1Point, G1PointScalar, G2Point, G2PointScalar},
    bls12_381_const::{G1_MSM_INPUT_LENGTH, G2_MSM_INPUT_LENGTH, PAIRING_INPUT_LENGTH},
};
use sha2::{Digest, Sha256};
use std::collections::{BTreeMap, HashSet};

/// Maximum G1 MSM pair counts under op-revm's Jovian (1806) and Isthmus (3211) input caps.
const G1_MSM_CAPS: [usize; 2] = [
    JOVIAN_G1_MSM_MAX_INPUT_SIZE / G1_MSM_INPUT_LENGTH,
    ISTHMUS_G1_MSM_MAX_INPUT_SIZE / G1_MSM_INPUT_LENGTH,
];
/// Maximum G2 MSM pair counts under op-revm's Jovian (968) and Isthmus (1696) input caps.
const G2_MSM_CAPS: [usize; 2] = [
    JOVIAN_G2_MSM_MAX_INPUT_SIZE / G2_MSM_INPUT_LENGTH,
    ISTHMUS_G2_MSM_MAX_INPUT_SIZE / G2_MSM_INPUT_LENGTH,
];
/// Maximum pairing pair counts under op-revm's Jovian (408) and Isthmus (612) input caps.
const PAIRING_CAPS: [usize; 2] = [
    JOVIAN_PAIRING_MAX_INPUT_SIZE / PAIRING_INPUT_LENGTH,
    ISTHMUS_PAIRING_MAX_INPUT_SIZE / PAIRING_INPUT_LENGTH,
];

/// The 16 field elements u for which simplified SWU on the G1 11-isogenous curve lands in the
/// isogeny kernel, so the EIP-2537 `MAP_FP_TO_G1` output is the identity (RFC 9380 §6.6.3).
///
/// Derived offline by taking the 5 Fp roots of the RFC 9380 App. E.2 x-denominator
/// (`ISOGENY_MAP.x_map_denominator`), whose y is Fp-rational for each, and inverting simplified
/// SWU (RFC 9380 §6.6.2) for both y signs through both SSWU branches. Arkworks and blst both map
/// each one to the identity. The test `iso_kernel_u_hits_isogeny_kernel` re-verifies the kernel
/// property independently of any implementation under test.
pub const ISO_KERNEL_U: [[u8; 48]; 16] = [
    hex!(
        "068951d10be6961019aa800a51cf48b707fc9e40700510406be9242d0c8dd866afdec0d66f9dc2cf1dc944702ec161bb"
    ),
    hex!(
        "0854a3cb180882d5b1efc1c3cc5b3fb33b27cb739f1389986ca46e1c5cb5010d8a06fd781c63074868f316d95b8f8405"
    ),
    hex!(
        "0fc521e8179e7ce1fa56812a062ccf7ffd45457d92c906be7a2dc0c20e586b0c0ba3dd2baed472eb91fc287b15e65ae1"
    ),
    hex!(
        "0fdb0c04a060175be7a91d3c2ee2d53bb7ccec610003a81199f7e2c3c3a488d4c2ecbaef1f3e91f1961d91cdad42da69"
    ),
    hex!(
        "0a2605e5991fcf3e63728a7a1468d79bacaa5f23f3816aadcd38efdd330c6d4f5bbf450f92156e0e23e16e3252bcd042"
    ),
    hex!(
        "0a3bf00221e169b850c5268c3d1edd576732060760bc0c00ed0311dee8588b18130822d3027f8d142802d784ea194fca"
    ),
    hex!(
        "11ac6e1f217763c4992be5f276f06d24294f801154718926fa8c648499fbf51694a5028694f0f8b7510be926a47026a6"
    ),
    hex!(
        "1377c0192d99508a317127abf17c64205c7aad448380027efb47ae73ea231dbd6ecd3f2841b63d309c35bb8fd13e48f0"
    ),
    hex!(
        "0998e1e079710a43d477d37ab7f0c430d601a85b9e8e6dbc808410dfd7dc5a5976e3bc7792cb83163665df3d00f0377c"
    ),
    hex!(
        "0ec1d2551f80abe70136a7f42e52133ebddf9b619a88147ae422a98e57581f2b0961dc019c74599f12a1b5513649a2e8"
    ),
    hex!(
        "0f6ece6ba8c39f6a0170531af7019877792795b7b98d2439a6112c76675ddf021372741a9089a27c552a9f0911e0ad82"
    ),
    hex!(
        "146850b3bdc2495ed73bb803dfaa951a88abff0acb5c7aeac52b48f3c808e87ce3885b98ce916e17caef21a6cbc6b598"
    ),
    hex!(
        "0598c1367bbd9d3b73dfefb263a117bcdbcb4c7a282897d4a20589ad2ea80da73b23a465e2c291e7ef0fde593438f513"
    ),
    hex!(
        "0a92437e90bc473049ab549b4c4a145feb4fb5cd39f7ee85c11fa62a8f5317220b398be420ca5d8364d460f6ee1efd29"
    ),
    hex!(
        "0b3f3f9519ff3ab349e4ffc214f99998a697b02358fcfe44830e29129f58d6f9154a23fd14dfa660a75d4aaec9b607c3"
    ),
    hex!(
        "10683009c00edc5676a3d43b8b5ae8a68e75a32954f6a502e6acc1c11ed49bcaa7c843871e887ce9839920c2ff0f732f"
    ),
];

const ZERO_FP: [u8; 48] = [0; 48];
/// `0x40‖0…`: the `ZCash` serialization's infinity flag, non-canonical as an EIP-2537 element.
pub const ZCASH_INFINITY: [u8; 48] = {
    let mut b = [0u8; 48];
    b[0] = 0x40;
    b
};
/// The EIP-2537 G1 identity encoding.
pub const ZERO_G1: G1Point = ([0; 48], [0; 48]);
/// The EIP-2537 G2 identity encoding.
pub const ZERO_G2: G2Point = ([0; 48], [0; 48], [0; 48], [0; 48]);

// ---------------------------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------------------------

/// BLS12-381 base-field modulus p, 48 bytes big-endian. Source: arkworks `Fq::MODULUS`.
pub fn p_bytes() -> [u8; 48] {
    to_array(Fq::MODULUS.to_bytes_be())
}

/// BLS12-381 scalar-field modulus r, 32 bytes big-endian. Source: arkworks `Fr::MODULUS`.
pub fn r_bytes() -> [u8; 32] {
    to_array(Fr::MODULUS.to_bytes_be())
}

/// r − 1, 32 bytes big-endian.
pub fn r_minus_1_bytes() -> [u8; 32] {
    (U256::from_be_bytes(r_bytes()) - U256::ONE).to_be_bytes()
}

/// A prime power `q^exponent` exactly dividing a cofactor.
#[derive(Clone, Copy, Debug)]
struct CofactorPrime {
    /// Display name of q, used in case labels.
    name: &'static str,
    /// q as little-endian u64 limbs.
    q: &'static [u64],
    /// The exponent of q in the cofactor.
    exponent: u32,
}

/// Prime factorization of the G1 cofactor h1 = (x − 1)²/3 = 3·11²·10177²·859267²·52437899²
/// (arkworks `g1::Config::COFACTOR`). Checked by `cofactor_factorizations_are_exact`.
const H1_FACTORS: [CofactorPrime; 5] = [
    CofactorPrime { name: "3", q: &[3], exponent: 1 },
    CofactorPrime { name: "11", q: &[11], exponent: 2 },
    CofactorPrime { name: "10177", q: &[10177], exponent: 2 },
    CofactorPrime { name: "859267", q: &[859267], exponent: 2 },
    CofactorPrime { name: "52437899", q: &[52437899], exponent: 2 },
];

/// Prime factorization of the G2 cofactor h2 = 13²·23²·2713·11953·262069·q448 (arkworks
/// `g2::Config::COFACTOR`), where q448 is a 448-bit prime (Miller–Rabin, checked offline). The
/// product is checked by `cofactor_factorizations_are_exact`.
const H2_FACTORS: [CofactorPrime; 6] = [
    CofactorPrime { name: "13", q: &[13], exponent: 2 },
    CofactorPrime { name: "23", q: &[23], exponent: 2 },
    CofactorPrime { name: "2713", q: &[2713], exponent: 1 },
    CofactorPrime { name: "11953", q: &[11953], exponent: 1 },
    CofactorPrime { name: "262069", q: &[262069], exponent: 1 },
    CofactorPrime {
        name: "q448",
        q: &[
            0x826d177200c0d3b1,
            0x77d87384d026cd73,
            0xfab9c0da5cf222c3,
            0xa9d75bb98b95878a,
            0xe0490c5afca1eeb2,
            0x423572788bea4d6a,
            0x8d9f503deeeb5d5c,
        ],
        exponent: 1,
    },
];

/// SSWU Z for the G1 11-isogenous curve (= 11). Source: arkworks
/// `<<g1::Config as WBConfig>::IsogenousCurve as SWUConfig>::ZETA`, = RFC 9380 §8.8.1.
const fn z1() -> Fq {
    <<g1::Config as WBConfig>::IsogenousCurve as SWUConfig>::ZETA
}

/// SSWU Z for the G2 3-isogenous curve (= −(2 + u)). Source: arkworks
/// `<<g2::Config as WBConfig>::IsogenousCurve as SWUConfig>::ZETA`, = RFC 9380 §8.8.2.
const fn z2() -> Fq2 {
    <<g2::Config as WBConfig>::IsogenousCurve as SWUConfig>::ZETA
}

/// The 8 non-canonical 48-byte field encodings: values derived from p (p, p+1, p+2 — an alias of
/// the canonical 2 —, 2^381−1, 2^384−1) and the `ZCash` flag-bit patterns (`0x40‖0…` infinity
/// flag, G1 generator x with the compression flag `0x80`, and with the sort flag `0x20`).
fn nc_values() -> Vec<[u8; 48]> {
    let p = U384::from_be_bytes(p_bytes());
    let gx = enc_fq(G1Affine::generator().x);
    let mut gx_compressed = gx;
    gx_compressed[0] |= 0x80;
    let mut gx_sort = gx;
    gx_sort[0] |= 0x20;
    vec![
        p.to_be_bytes(),
        (p + U384::ONE).to_be_bytes(),
        (p + U384::from(2)).to_be_bytes(),
        ((U384::ONE << 381usize) - U384::ONE).to_be_bytes(),
        U384::MAX.to_be_bytes(),
        ZCASH_INFINITY,
        gx_compressed,
        gx_sort,
    ]
}

/// EIP-2537 scalar edge cases (32 bytes big-endian): 0, 1, 2, r−1, r, r+1, 2^255, 2^256−1, and
/// 3 seeded 256-bit values. r is from arkworks `Fr::MODULUS`.
fn scalar_values() -> Vec<[u8; 32]> {
    let r = U256::from_be_bytes(r_bytes());
    let mut out = vec![
        be32(0),
        be32(1),
        be32(2),
        r_minus_1_bytes(),
        r.to_be_bytes(),
        (r + U256::ONE).to_be_bytes(),
        (U256::ONE << 255usize).to_be_bytes(),
        U256::MAX.to_be_bytes(),
    ];
    out.extend((0..3).map(|i| to_array(seed("scalar-edge", i, 32))));
    out
}

/// Scalars at both edges of every arkworks `msm_signed` size class (ark-ec 0.6
/// `scalar_mul/variable_base/mod.rs` `ScalarSize`): U1, U8, U16, U32, U64, their negations mod r,
/// and `BigInt`. Arkworks dispatches multi-term MSMs on these classes; `sp1_bls12_381` does not.
fn msm_size_class_scalars() -> Vec<[u8; 32]> {
    let r = U256::from_be_bytes(r_bytes());
    [1, 0xff, 0x100, 0xffff, 0x1_0000, 0xffff_ffff, 0x1_0000_0000, u64::MAX]
        .map(U256::from)
        .into_iter()
        .chain([U256::ONE << 64usize])
        .flat_map(|v| [v, r - v])
        .map(|v| v.to_be_bytes())
        .collect()
}

// ---------------------------------------------------------------------------------------------
// Seeded helpers and encodings
// ---------------------------------------------------------------------------------------------

/// `sha256(label ‖ i_be32 ‖ block_be32)` over blocks 0, 1, … concatenated and truncated to `len`.
fn seed(label: &str, i: u32, len: usize) -> Vec<u8> {
    let mut out = Vec::with_capacity(len + 32);
    let mut block = 0u32;
    while out.len() < len {
        let mut h = Sha256::new();
        h.update(label.as_bytes());
        h.update(i.to_be_bytes());
        h.update(block.to_be_bytes());
        out.extend_from_slice(&h.finalize());
        block += 1;
    }
    out.truncate(len);
    out
}

fn fq(label: &str, i: u32) -> Fq {
    Fq::from_be_bytes_mod_order(&seed(label, i, 64))
}

fn fq2(label: &str, i: u32) -> Fq2 {
    Fq2::new(fq(&format!("{label}/c0"), i), fq(&format!("{label}/c1"), i))
}

fn fr(label: &str, i: u32) -> Fr {
    Fr::from_be_bytes_mod_order(&seed(label, i, 64))
}

/// `x` as 48 bytes big-endian.
pub fn enc_fq(x: Fq) -> [u8; 48] {
    to_array(x.into_bigint().to_bytes_be())
}

/// EIP-2537 G1 encoding: all zeros for the identity, otherwise `x ‖ y`.
pub fn enc_g1(p: G1Affine) -> G1Point {
    if p.is_zero() { ZERO_G1 } else { (enc_fq(p.x), enc_fq(p.y)) }
}

/// EIP-2537 G2 encoding: all zeros for the identity, otherwise `x.c0 ‖ x.c1 ‖ y.c0 ‖ y.c1`.
pub fn enc_g2(p: G2Affine) -> G2Point {
    if p.is_zero() {
        ZERO_G2
    } else {
        (enc_fq(p.x.c0), enc_fq(p.x.c1), enc_fq(p.y.c0), enc_fq(p.y.c1))
    }
}

fn to_array<const N: usize>(v: Vec<u8>) -> [u8; N] {
    v.try_into().expect("length")
}

/// `v` as a 32-byte big-endian scalar.
pub fn be32(v: u64) -> [u8; 32] {
    U256::from(v).to_be_bytes()
}

/// Schoolbook product of two little-endian u64-limb integers, without trailing zero limbs.
fn mul_limbs(a: &[u64], b: &[u64]) -> Vec<u64> {
    let mut out = vec![0u64; a.len() + b.len()];
    for (i, &x) in a.iter().enumerate() {
        let mut carry = 0u128;
        for (j, &y) in b.iter().enumerate() {
            let t = u128::from(out[i + j]) + u128::from(x) * u128::from(y) + carry;
            out[i + j] = t as u64;
            carry = t >> 64;
        }
        out[i + b.len()] = carry as u64;
    }
    while out.len() > 1 && out.last() == Some(&0) {
        out.pop();
    }
    out
}

/// Product of `q^exponent` over `factors`, skipping index `skip`.
fn cofactor_product(factors: &[CofactorPrime], skip: Option<usize>) -> Vec<u64> {
    let mut acc = vec![1u64];
    for (i, f) in factors.iter().enumerate() {
        if Some(i) != skip {
            for _ in 0..f.exponent {
                acc = mul_limbs(&acc, f.q);
            }
        }
    }
    acc
}

// ---------------------------------------------------------------------------------------------
// Point builders
// ---------------------------------------------------------------------------------------------

/// A seeded G1 subgroup point `generator · fr(label, i)`.
pub fn g1_sub(label: &str, i: u32) -> G1Affine {
    (G1Affine::generator() * fr(label, i)).into_affine()
}

/// A seeded G2 subgroup point `generator · fr(label, i)`.
fn g2_sub(label: &str, i: u32) -> G2Affine {
    (G2Affine::generator() * fr(label, i)).into_affine()
}

/// `k` seeded subgroup points, computed as `base + j·step` with batch normalization (cheap to
/// build at the caps).
fn seeded_points<P: SWCurveConfig>(base: Affine<P>, step: Affine<P>, k: usize) -> Vec<Affine<P>> {
    let mut acc = Projective::<P>::from(base);
    let mut out = Vec::with_capacity(k);
    for _ in 0..k {
        out.push(acc);
        acc += step;
    }
    Projective::normalize_batch(&out)
}

/// An on-curve G1 point outside the r-order subgroup.
pub fn g1_off_subgroup(label: &str, i: u32) -> G1Affine {
    off_subgroup(|j| fq(&format!("{label}/{j}"), i))
}

/// An on-curve G2 point outside the r-order subgroup.
pub fn g2_off_subgroup(label: &str, i: u32) -> G2Affine {
    off_subgroup(|j| fq2(&format!("{label}/{j}"), i))
}

fn off_subgroup<P: SWCurveConfig>(x_at: impl Fn(u32) -> P::BaseField) -> Affine<P> {
    for j in 0..1024 {
        if let Some(p) = Affine::<P>::get_point_from_x_unchecked(x_at(j), false) {
            assert!(p.is_on_curve());
            assert!(!p.is_in_correct_subgroup_assuming_on_curve());
            return p;
        }
    }
    unreachable!("no x with a square right-hand side in 1024 tries");
}

/// A point of exact order `factors[idx].q`, from the q-primary part of the h-torsion of an
/// off-subgroup point: `(r · h/q^e) · N`, then multiplied by q until one more step would reach the
/// identity. Tries seeds `start..start + 64`; returns the point and the next unused seed.
fn prime_torsion<P: SWCurveConfig<ScalarField = Fr>>(
    off: &impl Fn(u32) -> Affine<P>,
    factors: &[CofactorPrime],
    idx: usize,
    start: u32,
) -> (Affine<P>, u32) {
    let q = factors[idx].q;
    let multiplier = cofactor_product(factors, Some(idx));
    for i in start..start + 64 {
        let mut t =
            off(i).mul_bigint(Fr::MODULUS).into_affine().mul_bigint(&multiplier).into_affine();
        if t.is_zero() {
            continue;
        }
        loop {
            let qt = t.mul_bigint(q).into_affine();
            if qt.is_zero() {
                break;
            }
            t = qt;
        }
        assert!(t.is_on_curve() && !t.is_zero());
        return (t, i + 1);
    }
    unreachable!("no point of order {} found", factors[idx].name);
}

/// Points of exact order `factors[idx].q`: one per order-q subgroup needed to cover the q-part.
/// When q² divides the cofactor the q-part of these curves is Z/q × Z/q (q + 1 subgroups of order
/// q), so a second point from a different subgroup is added; independence is checked by
/// [`in_cyclic_span`].
fn prime_torsion_points<P: SWCurveConfig<ScalarField = Fr>>(
    off: impl Fn(u32) -> Affine<P>,
    factors: &[CofactorPrime],
    idx: usize,
) -> Vec<Affine<P>> {
    let (first, mut next) = prime_torsion(&off, factors, idx, 0);
    if factors[idx].exponent == 1 {
        return vec![first];
    }
    let [q] = factors[idx].q else { unreachable!("squared factors fit in one limb") };
    for _ in 0..64 {
        let (second, after) = prime_torsion(&off, factors, idx, next);
        if !in_cyclic_span(first, second, *q) {
            return vec![first, second];
        }
        next = after;
    }
    unreachable!("no second order-{q} subgroup found: the {q}-part is not Z/q × Z/q");
}

/// Whether `target` is a multiple of `base`, where both have prime order `q` (baby-step
/// giant-step).
fn in_cyclic_span<P: SWCurveConfig>(base: Affine<P>, target: Affine<P>, q: u64) -> bool {
    let m = q.isqrt() + 1;
    let mut baby = Vec::with_capacity(m as usize);
    let mut acc = Projective::<P>::zero();
    for _ in 0..m {
        baby.push(acc);
        acc += base;
    }
    let baby: HashSet<Affine<P>> = Projective::normalize_batch(&baby).into_iter().collect();
    let stride = base.mul_bigint([m]);
    let mut giant = Vec::with_capacity(m as usize);
    let mut acc = Projective::<P>::from(target);
    for _ in 0..m {
        giant.push(acc);
        acc -= stride;
    }
    Projective::normalize_batch(&giant).iter().any(|g| baby.contains(g))
}

fn first_torsion<P: SWCurveConfig<ScalarField = Fr>>(
    off: impl Fn(u32) -> Affine<P>,
    factors: &[CofactorPrime],
    q: u64,
) -> Affine<P> {
    let idx = factors.iter().position(|f| f.q == [q]).expect("q divides the cofactor");
    prime_torsion(&off, factors, idx, 0).0
}

/// A G1 point of exact prime order `q`, where `q` divides the G1 cofactor (e.g. 11, 10177).
fn g1_torsion(q: u64) -> G1Affine {
    first_torsion(|i| g1_off_subgroup("g1-torsion", i), &H1_FACTORS, q)
}

/// A G2 point of exact prime order `q`, where `q` divides the G2 cofactor (e.g. 13, 23).
fn g2_torsion(q: u64) -> G2Affine {
    first_torsion(|i| g2_off_subgroup("g2-torsion", i), &H2_FACTORS, q)
}

/// Labelled on-curve points outside the r-order subgroup built from every prime factor of the
/// cofactor: pure torsion points of each prime order (two from different subgroups when q²
/// divides the cofactor, labels `T{q}a`/`T{q}b`), their sum `T{q}a+T{q}b` (order q, a third
/// subgroup), sums of consecutive primes' points (composite order), and a subgroup point plus
/// each `T{q}a` (`P+T{q}a`, which has both an r-part and a torsion part).
fn torsion_points<P: SWCurveConfig<ScalarField = Fr>>(
    off: impl Fn(u32) -> Affine<P> + Copy,
    factors: &[CofactorPrime],
    subgroup_point: Affine<P>,
) -> Vec<(String, Affine<P>)> {
    let mut out = Vec::new();
    let mut firsts = Vec::new();
    for (idx, f) in factors.iter().enumerate() {
        let points = prime_torsion_points(off, factors, idx);
        for (p, suffix) in points.iter().zip(["a", "b"]) {
            out.push((format!("T{}{suffix}", f.name), *p));
        }
        if let [a, b] = points[..] {
            out.push((format!("T{0}a+T{0}b", f.name), (a + b).into_affine()));
        }
        firsts.push((f.name, points[0]));
    }
    for pair in firsts.windows(2) {
        let ((n1, a), (n2, b)) = (pair[0], pair[1]);
        out.push((format!("T{n1}a+T{n2}a"), (a + b).into_affine()));
    }
    for (name, t) in &firsts {
        out.push((format!("P+T{name}a"), (subgroup_point + t).into_affine()));
    }
    out
}

/// `torsion_points` on G1.
pub fn g1_torsion_points() -> Vec<(String, G1Affine)> {
    torsion_points(|i| g1_off_subgroup("g1-torsion", i), &H1_FACTORS, g1_sub("g1-torsion/P", 0))
}

/// `torsion_points` on G2.
pub fn g2_torsion_points() -> Vec<(String, G2Affine)> {
    torsion_points(|i| g2_off_subgroup("g2-torsion", i), &H2_FACTORS, g2_sub("g2-torsion/P", 0))
}

/// The G1 order-3 points (0, ±2): 2² = 0³ + 4.
pub fn g1_order3(positive: bool) -> G1Affine {
    let two = Fq::from(2u64);
    G1Affine::new_unchecked(Fq::zero(), if positive { two } else { -two })
}

/// A seeded G1 encoding (x, y) not on the curve.
pub fn g1_not_on_curve(i: u32) -> G1Point {
    let p = G1Affine::new_unchecked(fq("g1-not-on-curve/x", i), fq("g1-not-on-curve/y", i));
    assert!(!p.is_on_curve());
    enc_g1(p)
}

/// A seeded G2 encoding not on the curve.
pub fn g2_not_on_curve(i: u32) -> G2Point {
    let p = G2Affine::new_unchecked(fq2("g2-not-on-curve/x", i), fq2("g2-not-on-curve/y", i));
    assert!(!p.is_on_curve());
    enc_g2(p)
}

/// `p` with c0/c1 swapped in both coordinates (the `ZCash` vs EIP-2537 order mix-up).
fn g2_swap(p: G2Point) -> G2Point {
    let swapped = (p.1, p.0, p.3, p.2);
    let x =
        Fq2::new(Fq::from_be_bytes_mod_order(&swapped.0), Fq::from_be_bytes_mod_order(&swapped.1));
    let y =
        Fq2::new(Fq::from_be_bytes_mod_order(&swapped.2), Fq::from_be_bytes_mod_order(&swapped.3));
    assert!(!G2Affine::new_unchecked(x, y).is_on_curve());
    swapped
}

/// The four canonical G2 encodings with a single coordinate equal to 1 and the rest 0. None is on
/// the curve, and none is the identity (which requires all four coordinates to be zero).
fn g2_single_coordinate() -> [G2Point; 4] {
    core::array::from_fn(|pos| {
        let point = set_g2(ZERO_G2, pos, enc_fq(Fq::one()));
        let fq = |b: &[u8; 48]| Fq::from_be_bytes_mod_order(b);
        let (x, y) = (Fq2::new(fq(&point.0), fq(&point.1)), Fq2::new(fq(&point.2), fq(&point.3)));
        assert!(!G2Affine::new_unchecked(x, y).is_on_curve());
        point
    })
}

/// `p` with coordinate `pos` (0 = x, 1 = y) replaced by the raw bytes `v`.
pub fn set_g1(p: G1Point, pos: usize, v: [u8; 48]) -> G1Point {
    let mut c = [p.0, p.1];
    c[pos] = v;
    c.into()
}

/// `p` with coordinate `pos` (0 = x.c0, 1 = x.c1, 2 = y.c0, 3 = y.c1) replaced by the raw bytes
/// `v`.
pub fn set_g2(p: G2Point, pos: usize, v: [u8; 48]) -> G2Point {
    let mut c = [p.0, p.1, p.2, p.3];
    c[pos] = v;
    c.into()
}

/// Every 7th scalar in a seeded MSM is an edge-case scalar.
fn msm_scalar(label: &str, i: usize, edges: &[[u8; 32]]) -> [u8; 32] {
    if i.is_multiple_of(7) {
        edges[(i / 7) % edges.len()]
    } else {
        to_array(seed(label, i as u32, 32))
    }
}

// ---------------------------------------------------------------------------------------------
// Case builders
// ---------------------------------------------------------------------------------------------

struct Cases {
    op: &'static str,
    out: Vec<BlsCase>,
    counters: BTreeMap<&'static str, usize>,
}

impl Cases {
    const fn new(op: &'static str) -> Self {
        Self { op, out: Vec::new(), counters: BTreeMap::new() }
    }

    fn push(&mut self, group: &'static str, call: BlsCall) {
        let n = *self.counters.entry(group).and_modify(|n| *n += 1).or_default();
        self.out.push(BlsCase { label: format!("{}/{group}/{n}", self.op), call });
    }

    fn push_labelled(&mut self, label: String, call: BlsCall) {
        self.out.push(BlsCase { label: format!("{}/{label}", self.op), call });
    }
}

/// G1ADD cases: all ordered pairs over a set mixing subgroup, low-order and mixed points;
/// non-canonical coordinates in every position; non-canonical identities; not-on-curve points;
/// and error-precedence pairs.
pub fn g1_add_cases() -> Vec<BlsCase> {
    let mut c = Cases::new("g1-add");
    let g = G1Affine::generator();
    let p1 = g1_sub("g1-add/P", 1);
    let p2 = g1_sub("g1-add/P", 2);
    let n1 = g1_off_subgroup("g1-add/N", 1);
    let set: Vec<G1Point> = [
        G1Affine::zero(),
        g,
        -g,
        (g + g).into_affine(),
        p1,
        p2,
        -p1,
        g1_order3(true),
        g1_order3(false),
        g1_torsion(11),
        g1_torsion(10177),
        n1,
        (p1 + g1_order3(true)).into_affine(),
    ]
    .into_iter()
    .map(enc_g1)
    .collect();
    for a in &set {
        for b in &set {
            c.push("pairs", BlsCall::G1Add(*a, *b));
        }
    }

    let (eg, ep1) = (enc_g1(g), enc_g1(p1));
    for nc in nc_values() {
        for pos in 0..4 {
            let (a, b) =
                if pos < 2 { (set_g1(eg, pos, nc), ep1) } else { (eg, set_g1(ep1, pos - 2, nc)) };
            c.push("non-canonical", BlsCall::G1Add(a, b));
        }
    }

    let p = p_bytes();
    for id in [(p, ZERO_FP), (ZERO_FP, p), (p, p), (ZCASH_INFINITY, ZERO_FP)] {
        c.push("non-canonical-identity", BlsCall::G1Add(id, eg));
        c.push("non-canonical-identity", BlsCall::G1Add(eg, id));
    }

    let not_on_curve =
        [g1_not_on_curve(0), ([0; 48], enc_fq(Fq::one())), (enc_fq(Fq::one()), [0; 48])];
    for noc in not_on_curve {
        for other in [ZERO_G1, eg] {
            c.push("not-on-curve", BlsCall::G1Add(noc, other));
            c.push("not-on-curve", BlsCall::G1Add(other, noc));
        }
    }

    c.push("precedence", BlsCall::G1Add(not_on_curve[0], set_g1(eg, 0, p)));
    c.push("precedence", BlsCall::G1Add(set_g1(eg, 1, p), not_on_curve[0]));
    c.push("precedence", BlsCall::G1Add(enc_g1(n1), not_on_curve[0]));
    for (name, t) in g1_torsion_points() {
        let (et, eneg) = (enc_g1(t), enc_g1(-t));
        for (what, a, b) in [
            ("plus-identity", et, ZERO_G1),
            ("identity-plus", ZERO_G1, et),
            ("double", et, et),
            ("plus-negation", et, eneg),
            ("plus-G", et, eg),
            ("G-plus", eg, et),
        ] {
            c.push_labelled(format!("torsion/{name}/{what}"), BlsCall::G1Add(a, b));
        }
    }
    c.out
}

/// G2ADD cases, mirroring [`g1_add_cases`].
pub fn g2_add_cases() -> Vec<BlsCase> {
    let mut c = Cases::new("g2-add");
    let g = G2Affine::generator();
    let q1 = g2_sub("g2-add/Q", 1);
    let q2 = g2_sub("g2-add/Q", 2);
    let n2 = g2_off_subgroup("g2-add/N", 1);
    let t13 = g2_torsion(13);
    let set: Vec<G2Point> = [
        G2Affine::zero(),
        g,
        -g,
        (g + g).into_affine(),
        q1,
        q2,
        -q1,
        t13,
        g2_torsion(23),
        n2,
        (q1 + t13).into_affine(),
    ]
    .into_iter()
    .map(enc_g2)
    .collect();
    for a in &set {
        for b in &set {
            c.push("pairs", BlsCall::G2Add(*a, *b));
        }
    }

    let (eg, eq1) = (enc_g2(g), enc_g2(q1));
    for nc in nc_values() {
        for pos in 0..8 {
            let (a, b) =
                if pos < 4 { (set_g2(eg, pos, nc), eq1) } else { (eg, set_g2(eq1, pos - 4, nc)) };
            c.push("non-canonical", BlsCall::G2Add(a, b));
        }
    }

    let p = p_bytes();
    for v in [p, ZCASH_INFINITY] {
        for pos in 0..4 {
            let id = set_g2(ZERO_G2, pos, v);
            c.push("non-canonical-identity", BlsCall::G2Add(id, eg));
            c.push("non-canonical-identity", BlsCall::G2Add(eg, id));
        }
    }

    let not_on_curve = [g2_not_on_curve(0), g2_swap(eq1)];
    for noc in not_on_curve.into_iter().chain(g2_single_coordinate()) {
        for other in [ZERO_G2, eg] {
            c.push("not-on-curve", BlsCall::G2Add(noc, other));
            c.push("not-on-curve", BlsCall::G2Add(other, noc));
        }
    }

    c.push("precedence", BlsCall::G2Add(not_on_curve[0], set_g2(eg, 0, p)));
    c.push("precedence", BlsCall::G2Add(set_g2(eg, 2, p), not_on_curve[0]));
    c.push("precedence", BlsCall::G2Add(enc_g2(n2), not_on_curve[0]));
    for (name, t) in g2_torsion_points() {
        let (et, eneg) = (enc_g2(t), enc_g2(-t));
        for (what, a, b) in [
            ("plus-identity", et, ZERO_G2),
            ("identity-plus", ZERO_G2, et),
            ("double", et, et),
            ("plus-negation", et, eneg),
            ("plus-G", et, eg),
            ("G-plus", eg, et),
        ] {
            c.push_labelled(format!("torsion/{name}/{what}"), BlsCall::G2Add(a, b));
        }
    }
    c.out
}

/// G1MSM cases: scalar edges, identity points, zero scalars, invalid points behind skip-eligible
/// scalars, cancellation, padding-error precedence, both sides of `CustomCrypto`'s per-term/bucket
/// threshold (3 vs 4 kept terms), every arkworks multi-term scalar size class, and seeded sizes up
/// to the Jovian and Isthmus caps.
pub fn g1_msm_cases() -> Vec<BlsCase> {
    let mut c = Cases::new("g1-msm");
    let sc = scalar_values();
    let g = enc_g1(G1Affine::generator());
    let p1a = g1_sub("g1-msm/P", 1);
    let p1 = enc_g1(p1a);
    let neg_p1 = enc_g1(-p1a);
    let n1 = enc_g1(g1_off_subgroup("g1-msm/N", 1));
    let t11 = enc_g1(g1_torsion(11));
    let o3 = enc_g1(g1_order3(true));
    let mixed = enc_g1((p1a + g1_order3(true)).into_affine());
    let noc = g1_not_on_curve(1);
    let x_p = set_g1(g, 0, p_bytes());
    let r = r_bytes();
    let some = |p: G1Point, s: [u8; 32]| -> Option<G1PointScalar> { Some((p, s)) };

    for s in &sc {
        c.push("scalar-edge", BlsCall::G1Msm(vec![some(g, *s)]));
        c.push("scalar-edge", BlsCall::G1Msm(vec![some(p1, *s)]));
    }
    c.push("identity", BlsCall::G1Msm(vec![some(ZERO_G1, be32(5))]));
    c.push("identity", BlsCall::G1Msm(vec![some(ZERO_G1, be32(0))]));
    c.push("identity", BlsCall::G1Msm(vec![some(ZERO_G1, be32(1)), some(g, be32(2))]));
    c.push("zero-scalars", BlsCall::G1Msm(vec![some(g, be32(0)), some(p1, be32(0))]));
    let one = enc_fq(Fq::one());
    for (pt, s) in [
        (n1, 0),
        (t11, 0),
        (o3, 0),
        (mixed, 1),
        (noc, 0),
        (x_p, 0),
        ((ZERO_FP, one), 0),
        ((one, ZERO_FP), 0),
        ((ZCASH_INFINITY, ZERO_FP), 0),
    ] {
        c.push("invalid-point", BlsCall::G1Msm(vec![some(pt, be32(s))]));
    }
    let (a, b) = (to_array(seed("g1-msm/a", 0, 32)), to_array(seed("g1-msm/b", 0, 32)));
    c.push("cancel", BlsCall::G1Msm(vec![some(p1, a), some(p1, b)]));
    c.push("cancel", BlsCall::G1Msm(vec![some(p1, a), some(neg_p1, a)]));
    c.push("cancel", BlsCall::G1Msm(vec![some(p1, r)]));
    c.push("cancel", BlsCall::G1Msm(vec![some(p1, be32(1)), some(p1, r_minus_1_bytes())]));
    c.push("padding", BlsCall::G1Msm(vec![some(g, be32(1)), None]));
    c.push("padding", BlsCall::G1Msm(vec![some(n1, be32(1)), None]));
    c.push("padding", BlsCall::G1Msm(vec![None, some(n1, be32(1))]));
    c.push("padding", BlsCall::G1Msm(vec![some(g, be32(1)), some(x_p, be32(1)), None]));

    // Kept-term counts 3 and 4 straddle `CustomCrypto`'s `SMALL_MSM = 4` per-term/bucket threshold
    // (`client/src/precompiles/bls12_381.rs`); update them if it changes. The bucket-path cases
    // repeat the identity, duplicate, and cancellation edges with 4 kept terms.
    let qa = g1_sub("g1-msm/Q", 1);
    let (q, neg_q) = (enc_g1(qa), enc_g1(-qa));
    let rr = enc_g1(g1_sub("g1-msm/R", 1));
    let (cc, d) = (to_array(seed("g1-msm/c", 0, 32)), to_array(seed("g1-msm/d", 0, 32)));
    for items in [
        vec![some(g, a), some(p1, b), some(q, cc)],
        vec![some(g, a), some(p1, b), some(q, cc), some(rr, d)],
        vec![some(g, a), some(p1, be32(0)), some(q, cc), some(rr, d)],
        vec![some(g, a), some(p1, b), some(q, be32(0)), some(rr, cc), some(g, d)],
    ] {
        c.push("threshold", BlsCall::G1Msm(items));
    }
    for items in [
        vec![some(ZERO_G1, be32(5)), some(g, a), some(p1, b), some(q, cc)],
        vec![some(p1, a), some(p1, b), some(q, cc), some(q, cc)],
        vec![some(p1, a), some(neg_p1, a), some(q, b), some(neg_q, b)],
        vec![some(p1, be32(1)), some(p1, r_minus_1_bytes()), some(q, a), some(neg_q, a)],
    ] {
        c.push("bucket-edge", BlsCall::G1Msm(items));
    }
    let size_classes = msm_size_class_scalars();
    let points = seeded_points(g1_sub("g1-msm/size-classes", 0), qa, size_classes.len());
    c.push(
        "size-classes",
        BlsCall::G1Msm(
            points.into_iter().zip(size_classes).map(|(p, s)| some(enc_g1(p), s)).collect(),
        ),
    );

    for k in [2, 3, 4, 5, 8, 16, 31, 32, 33, 64, 128, 256].into_iter().chain(G1_MSM_CAPS) {
        let label = format!("g1-msm/seeded/{k}");
        let points = seeded_points(g1_sub(&label, 0), g1_sub(&label, 1), k);
        let items = points
            .into_iter()
            .enumerate()
            .map(|(i, p)| some(enc_g1(p), msm_scalar(&label, i, &sc)))
            .collect();
        c.push("seeded", BlsCall::G1Msm(items));
    }
    for (name, t) in g1_torsion_points() {
        let et = enc_g1(t);
        for (what, items) in [
            ("times-0", vec![some(et, be32(0))]),
            ("times-1", vec![some(et, be32(1))]),
            ("times-r", vec![some(et, r)]),
            ("after-G", vec![some(g, be32(1)), some(et, be32(1))]),
        ] {
            c.push_labelled(format!("torsion/{name}/{what}"), BlsCall::G1Msm(items));
        }
    }
    c.out
}

/// G2MSM cases, mirroring [`g1_msm_cases`].
pub fn g2_msm_cases() -> Vec<BlsCase> {
    let mut c = Cases::new("g2-msm");
    let sc = scalar_values();
    let g = enc_g2(G2Affine::generator());
    let q1a = g2_sub("g2-msm/Q", 1);
    let q1 = enc_g2(q1a);
    let neg_q1 = enc_g2(-q1a);
    let n2 = enc_g2(g2_off_subgroup("g2-msm/N", 1));
    let t13a = g2_torsion(13);
    let t13 = enc_g2(t13a);
    let t23 = enc_g2(g2_torsion(23));
    let mixed = enc_g2((q1a + t13a).into_affine());
    let noc = g2_not_on_curve(1);
    let x_p = set_g2(g, 0, p_bytes());
    let r = r_bytes();
    let some = |p: G2Point, s: [u8; 32]| -> Option<G2PointScalar> { Some((p, s)) };

    for s in &sc {
        c.push("scalar-edge", BlsCall::G2Msm(vec![some(g, *s)]));
        c.push("scalar-edge", BlsCall::G2Msm(vec![some(q1, *s)]));
    }
    c.push("identity", BlsCall::G2Msm(vec![some(ZERO_G2, be32(5))]));
    c.push("identity", BlsCall::G2Msm(vec![some(ZERO_G2, be32(0))]));
    c.push("identity", BlsCall::G2Msm(vec![some(ZERO_G2, be32(1)), some(g, be32(2))]));
    c.push("zero-scalars", BlsCall::G2Msm(vec![some(g, be32(0)), some(q1, be32(0))]));
    let invalid = [(n2, 0), (t13, 0), (t23, 0), (mixed, 1), (noc, 0), (x_p, 0)]
        .into_iter()
        .chain(g2_single_coordinate().map(|pt| (pt, 0)))
        .chain([(set_g2(ZERO_G2, 1, ZCASH_INFINITY), 0)]);
    for (pt, s) in invalid {
        c.push("invalid-point", BlsCall::G2Msm(vec![some(pt, be32(s))]));
    }
    let (a, b) = (to_array(seed("g2-msm/a", 0, 32)), to_array(seed("g2-msm/b", 0, 32)));
    c.push("cancel", BlsCall::G2Msm(vec![some(q1, a), some(q1, b)]));
    c.push("cancel", BlsCall::G2Msm(vec![some(q1, a), some(neg_q1, a)]));
    c.push("cancel", BlsCall::G2Msm(vec![some(q1, r)]));
    c.push("cancel", BlsCall::G2Msm(vec![some(q1, be32(1)), some(q1, r_minus_1_bytes())]));
    c.push("padding", BlsCall::G2Msm(vec![some(g, be32(1)), None]));
    c.push("padding", BlsCall::G2Msm(vec![some(n2, be32(1)), None]));
    c.push("padding", BlsCall::G2Msm(vec![None, some(n2, be32(1))]));
    c.push("padding", BlsCall::G2Msm(vec![some(g, be32(1)), some(x_p, be32(1)), None]));

    // Mirrors the G1 threshold and bucket-edge cases.
    let qa = g2_sub("g2-msm/Q", 2);
    let (q, neg_q) = (enc_g2(qa), enc_g2(-qa));
    let rr = enc_g2(g2_sub("g2-msm/R", 1));
    let (cc, d) = (to_array(seed("g2-msm/c", 0, 32)), to_array(seed("g2-msm/d", 0, 32)));
    for items in [
        vec![some(g, a), some(q1, b), some(q, cc)],
        vec![some(g, a), some(q1, b), some(q, cc), some(rr, d)],
        vec![some(g, a), some(q1, be32(0)), some(q, cc), some(rr, d)],
        vec![some(g, a), some(q1, b), some(q, be32(0)), some(rr, cc), some(g, d)],
    ] {
        c.push("threshold", BlsCall::G2Msm(items));
    }
    for items in [
        vec![some(ZERO_G2, be32(5)), some(g, a), some(q1, b), some(q, cc)],
        vec![some(q1, a), some(q1, b), some(q, cc), some(q, cc)],
        vec![some(q1, a), some(neg_q1, a), some(q, b), some(neg_q, b)],
        vec![some(q1, be32(1)), some(q1, r_minus_1_bytes()), some(q, a), some(neg_q, a)],
    ] {
        c.push("bucket-edge", BlsCall::G2Msm(items));
    }
    let size_classes = msm_size_class_scalars();
    let points = seeded_points(g2_sub("g2-msm/size-classes", 0), qa, size_classes.len());
    c.push(
        "size-classes",
        BlsCall::G2Msm(
            points.into_iter().zip(size_classes).map(|(p, s)| some(enc_g2(p), s)).collect(),
        ),
    );

    for k in [2, 3, 4, 5, 8, 16, 31, 32, 33, 64, 128].into_iter().chain(G2_MSM_CAPS) {
        let label = format!("g2-msm/seeded/{k}");
        let points = seeded_points(g2_sub(&label, 0), g2_sub(&label, 1), k);
        let items = points
            .into_iter()
            .enumerate()
            .map(|(i, p)| some(enc_g2(p), msm_scalar(&label, i, &sc)))
            .collect();
        c.push("seeded", BlsCall::G2Msm(items));
    }
    for (name, t) in g2_torsion_points() {
        let et = enc_g2(t);
        for (what, items) in [
            ("times-0", vec![some(et, be32(0))]),
            ("times-1", vec![some(et, be32(1))]),
            ("times-r", vec![some(et, r)]),
            ("after-G", vec![some(g, be32(1)), some(et, be32(1))]),
        ] {
            c.push_labelled(format!("torsion/{name}/{what}"), BlsCall::G2Msm(items));
        }
    }
    c.out
}

/// `PAIRING_CHECK` cases: true/false products (including the Jovian and Isthmus caps), identity
/// combinations, and every validation error in every position, including ordering within and
/// between pairs.
pub fn pairing_cases() -> Vec<BlsCase> {
    let mut c = Cases::new("pairing");
    let g1 = G1Affine::generator();
    let g2 = G2Affine::generator();
    let (eg1, eg2) = (enc_g1(g1), enc_g2(g2));
    let neg_g1 = enc_g1(-g1);
    let n1 = enc_g1(g1_off_subgroup("pairing/N1", 0));
    let n2 = enc_g2(g2_off_subgroup("pairing/N2", 0));
    let p = p_bytes();

    c.push("true", BlsCall::Pairing(vec![]));
    c.push("true", BlsCall::Pairing(vec![(eg1, eg2), (neg_g1, eg2)]));
    for i in 0..2 {
        let (a, b) = (fr("pairing/a", i), fr("pairing/b", i));
        c.push(
            "true",
            BlsCall::Pairing(vec![
                (enc_g1((g1 * a).into_affine()), enc_g2((g2 * b).into_affine())),
                (enc_g1((g1 * -(a * b)).into_affine()), eg2),
            ]),
        );
    }
    for k in std::iter::once(4).chain(PAIRING_CAPS) {
        let label = format!("pairing/balanced/{k}");
        let scalars: Vec<Fr> = (0..k as u32 - 1).map(|i| fr(&label, i)).collect();
        let sum: Fr = scalars.iter().sum();
        let projective: Vec<G1Projective> =
            scalars.iter().map(|s| g1 * s).chain([g1 * -sum]).collect();
        let pairs = G1Projective::normalize_batch(&projective)
            .into_iter()
            .map(|p| (enc_g1(p), eg2))
            .collect();
        c.push("true", BlsCall::Pairing(pairs));
    }

    c.push("false", BlsCall::Pairing(vec![(eg1, eg2)]));
    for k in [1u32, 2, 3, 4, 8] {
        let label = format!("pairing/false/{k}");
        let pairs =
            (0..k).map(|i| (enc_g1(g1_sub(&label, i)), enc_g2(g2_sub(&label, i)))).collect();
        c.push("false", BlsCall::Pairing(pairs));
    }

    c.push("identity", BlsCall::Pairing(vec![(ZERO_G1, eg2)]));
    c.push("identity", BlsCall::Pairing(vec![(eg1, ZERO_G2)]));
    c.push("identity", BlsCall::Pairing(vec![(ZERO_G1, ZERO_G2)]));
    c.push("identity", BlsCall::Pairing(vec![(ZERO_G1, eg2), (eg1, eg2), (neg_g1, eg2)]));
    c.push("identity", BlsCall::Pairing(vec![(ZERO_G1, eg2), (eg1, eg2)]));

    let o3 = enc_g1(g1_order3(true));
    let t13 = enc_g2(g2_torsion(13));
    for pair in [(ZERO_G1, n2), (n1, ZERO_G2), (n1, eg2), (eg1, n2), (o3, eg2), (eg1, t13)] {
        c.push("subgroup", BlsCall::Pairing(vec![pair]));
    }
    let (noc1, noc2) = (g1_not_on_curve(2), g2_not_on_curve(2));
    let g1_x0 = (ZERO_FP, enc_fq(Fq::one()));
    for pair in [(noc1, eg2), (eg1, noc2), (noc1, noc2), (g1_x0, eg2), (g1_x0, ZERO_G2)] {
        c.push("curve", BlsCall::Pairing(vec![pair]));
    }
    for g2_single in g2_single_coordinate() {
        c.push("curve", BlsCall::Pairing(vec![(eg1, g2_single)]));
        c.push("curve", BlsCall::Pairing(vec![(ZERO_G1, g2_single)]));
    }
    for pair in [
        (eg1, set_g2(eg2, 1, p)),
        (set_g1(eg1, 1, p), n2),
        (ZERO_G1, set_g2(eg2, 0, p)),
        (set_g1(eg1, 0, p), ZERO_G2),
        ((p, ZERO_FP), eg2),
        ((ZCASH_INFINITY, ZERO_FP), eg2),
        (eg1, set_g2(ZERO_G2, 1, ZCASH_INFINITY)),
    ] {
        c.push("encoding", BlsCall::Pairing(vec![pair]));
    }
    // G1 is fully checked (subgroup included) before G2 is decoded.
    for pair in [(n1, noc2), (n1, set_g2(eg2, 0, p)), (n1, set_g2(eg2, 3, p)), (o3, noc2)] {
        c.push("within-pair", BlsCall::Pairing(vec![pair]));
    }
    c.push("ordering", BlsCall::Pairing(vec![(eg1, n2), (set_g1(eg1, 0, p), eg2)]));
    c.push("ordering", BlsCall::Pairing(vec![(eg1, eg2), (n1, ZERO_G2)]));
    for (name, t) in g1_torsion_points() {
        let et = enc_g1(t);
        for (what, pairs) in [
            ("with-G2", vec![(et, eg2)]),
            ("with-0", vec![(et, ZERO_G2)]),
            ("after-true-product", vec![(eg1, eg2), (neg_g1, eg2), (et, eg2)]),
        ] {
            c.push_labelled(format!("torsion/g1/{name}/{what}"), BlsCall::Pairing(pairs));
        }
    }
    for (name, t) in g2_torsion_points() {
        let et = enc_g2(t);
        for (what, pairs) in [
            ("with-G1", vec![(eg1, et)]),
            ("with-0", vec![(ZERO_G1, et)]),
            ("after-true-product", vec![(eg1, eg2), (neg_g1, eg2), (eg1, et)]),
        ] {
            c.push_labelled(format!("torsion/g2/{name}/{what}"), BlsCall::Pairing(pairs));
        }
    }
    c.out
}

/// `MAP_FP_TO_G1` cases: small and boundary values, the SSWU exceptional inputs, every isogeny
/// kernel input, seeded values, and every non-canonical encoding.
pub fn map_fp_to_g1_cases() -> Vec<BlsCase> {
    let mut c = Cases::new("map-fp-to-g1");
    let half = Fq::from_bigint(Fq::MODULUS_MINUS_ONE_DIV_TWO).expect("(p-1)/2 < p");
    for u in [Fq::zero(), Fq::one(), Fq::from(2u64), -Fq::one(), half, half + Fq::one()] {
        c.push("boundary", BlsCall::MapFpToG1(enc_fq(u)));
    }
    let exceptional = (-z1().inverse().expect("Z1 != 0")).sqrt().expect("-1/Z1 is a square in Fp");
    c.push("exceptional", BlsCall::MapFpToG1(enc_fq(exceptional)));
    c.push("exceptional", BlsCall::MapFpToG1(enc_fq(-exceptional)));
    for u in ISO_KERNEL_U {
        c.push("iso-kernel", BlsCall::MapFpToG1(u));
    }
    for i in 0..32 {
        c.push("seeded", BlsCall::MapFpToG1(enc_fq(fq("map-fp-to-g1/u", i))));
    }
    for nc in nc_values() {
        c.push("non-canonical", BlsCall::MapFpToG1(nc));
    }
    c.out
}

/// `MAP_FP2_TO_G2` cases: small and boundary values (including u = 0, the only SSWU exceptional
/// input), single-component values, seeded values, and non-canonical encodings in each component.
pub fn map_fp2_to_g2_cases() -> Vec<BlsCase> {
    let mut c = Cases::new("map-fp2-to-g2");
    let (zero, one, m1) = (enc_fq(Fq::zero()), enc_fq(Fq::one()), enc_fq(-Fq::one()));
    for (c0, c1) in [(zero, zero), (one, zero), (zero, one), (m1, m1)] {
        c.push("boundary", BlsCall::MapFp2ToG2(c0, c1));
    }
    for i in 0..4 {
        let u = enc_fq(fq("map-fp2-to-g2/single", i));
        c.push("single", BlsCall::MapFp2ToG2(u, zero));
        c.push("single", BlsCall::MapFp2ToG2(zero, u));
    }
    // u = 0 is the only SSWU exceptional input: u² = −1/Z2 has no root in Fp2.
    assert!((-z2().inverse().expect("Z2 != 0")).sqrt().is_none(), "-1/Z2 is a square in Fp2");
    for i in 0..32 {
        let u = fq2("map-fp2-to-g2/u", i);
        c.push("seeded", BlsCall::MapFp2ToG2(enc_fq(u.c0), enc_fq(u.c1)));
    }
    for nc in nc_values() {
        for (c0, c1) in [(nc, zero), (zero, nc), (nc, nc), (one, nc), (nc, one)] {
            c.push("non-canonical", BlsCall::MapFp2ToG2(c0, c1));
        }
    }
    c.out
}

/// Every case: the single corpus, run both on the host and inside the zkVM.
pub fn full_corpus() -> Vec<BlsCase> {
    let mut out = g1_add_cases();
    out.extend(g2_add_cases());
    out.extend(g1_msm_cases());
    out.extend(g2_msm_cases());
    out.extend(pairing_cases());
    out.extend(map_fp_to_g1_cases());
    out.extend(map_fp2_to_g2_cases());
    out
}

#[cfg(test)]
mod tests {
    use super::*;
    use ark_ec_0_6::CurveConfig;
    use std::collections::HashMap;

    fn trimmed(limbs: &[u64]) -> Vec<u64> {
        mul_limbs(limbs, &[1])
    }

    #[test]
    fn cofactor_factorizations_are_exact() {
        assert_eq!(
            cofactor_product(&H1_FACTORS, None),
            trimmed(<g1::Config as CurveConfig>::COFACTOR)
        );
        assert_eq!(
            cofactor_product(&H2_FACTORS, None),
            trimmed(<g2::Config as CurveConfig>::COFACTOR)
        );
        for f in H1_FACTORS.iter().chain(&H2_FACTORS) {
            if let [q] = f.q {
                assert!((2..).take_while(|d| d * d <= *q).all(|d| q % d != 0), "{q} is not prime");
            }
        }
    }

    /// Every pure torsion point has exactly its prime order, the two points of a Z/q × Z/q part
    /// lie in different subgroups, and every point is on the curve but outside the r-subgroup.
    fn check_structure<P: SWCurveConfig<ScalarField = Fr>>(
        points: Vec<(String, Affine<P>)>,
        factors: &[CofactorPrime],
    ) {
        let by_name: HashMap<_, _> = points.iter().cloned().collect();
        for (name, p) in &points {
            assert!(p.is_on_curve(), "{name} is off the curve");
            assert!(!p.is_in_correct_subgroup_assuming_on_curve(), "{name} is in the r-subgroup");
        }
        for f in factors {
            let a = by_name[&format!("T{}a", f.name)];
            assert!(!a.is_zero() && a.mul_bigint(f.q).is_zero(), "T{}a order", f.name);
            if f.exponent == 2 {
                let b = by_name[&format!("T{}b", f.name)];
                assert!(!b.is_zero() && b.mul_bigint(f.q).is_zero(), "T{}b order", f.name);
                assert!(!in_cyclic_span(a, b, f.q[0]), "T{0}a and T{0}b share a subgroup", f.name);
            } else {
                assert!(!by_name.contains_key(&format!("T{}b", f.name)));
            }
            let mixed = by_name[&format!("P+T{}a", f.name)];
            assert!(
                !mixed.mul_bigint(Fr::MODULUS).is_zero(),
                "P+T{}a lacks a torsion part",
                f.name
            );
            assert!(!mixed.mul_bigint(f.q).is_zero(), "P+T{}a lacks an r-part", f.name);
        }
    }

    #[test]
    fn torsion_points_have_claimed_structure() {
        check_structure(g1_torsion_points(), &H1_FACTORS);
        check_structure(g2_torsion_points(), &H2_FACTORS);
    }
}

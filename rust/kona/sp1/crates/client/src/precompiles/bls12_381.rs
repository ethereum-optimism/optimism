//! EIP-2537 BLS12-381 operations backed by `sp1_bls12_381`, accelerated in the SP1 zkVM.
//!
//! Contract: every output and every [`PrecompileHalt`] (including which error wins when an input
//! is invalid in several ways) is bit-identical to revm-precompile's arkworks EIP-2537 backend,
//! which is what the guest's `DefaultCrypto` runs.
//!
//! # Banned `sp1_bls12_381` APIs (never use in this module)
//!
//! - `G1Affine`/`G2Affine` `::from_uncompressed{,_unchecked}`, `::from_compressed{,_unchecked}`,
//!   `::to_uncompressed`, `::to_compressed`: `ZCash` serialization. The top 3 bits are flags, so
//!   `0x40‖0…` decodes as the identity while all-zero is *not* the identity, all failures collapse
//!   into one `None`, G2 coordinates are ordered c1‖c0, and the identity serializes with `0x40`.
//!   EIP-2537 has none of this: decode with [`read_fp`] / [`read_fp2`], encode with [`encode_g1`] /
//!   [`encode_g2`].
//! - `Fp2::from_bytes`, `Fp6::from_bytes`, `Fp12::from_bytes`: panic on non-canonical input.
//! - `Scalar::from_bytes`: rejects values ≥ r, but EIP-2537 scalars are arbitrary 256-bit integers;
//!   use [`read_scalar`]. `Mul<Scalar>` is only valid on canonical `Scalar`s.
//! - `G1Affine::from(&G1Projective)` / `G2Affine::from(&G2Projective)` directly: use
//!   [`g1_to_affine`] / [`g2_to_affine`].
//! - `CtOption::unwrap` on anything derived from untrusted input.
//! - Struct-literal points with `infinity = 1` and arbitrary coordinates: the identity must come
//!   from `G1Affine::identity()` / `G2Affine::identity()`.
//!
//! # zkVM hint paths and preconditions (re-audit when bumping `sp1_bls12_381`)
//!
//! - `Fp::invert` asserts its hint; [`g1_to_affine`] and the zkVM `mul_by_x` in `is_torsion_free` /
//!   `clear_h` rely on that.
//! - `Fp2::invert` returns `None` on a forged hint; never collapse that with `unwrap_or` (as
//!   `G2Affine::from` does, hence [`g2_to_affine`]). `final_exponentiation` unwraps it through
//!   `Fp6`/`Fp12::invert`, so [`pairing_check`] panics instead of returning a wrong result.
//! - G2 `map_to_curve_simple_swu` yields the off-curve (0, 0) at u = 0: call `map_to_curve` only
//!   behind [`map_fp2_to_g2`]'s zero check.
//! - `G1Affine::add_affine` / `double` require on-curve inputs: off-curve input gives a wrong but
//!   provable zkVM result (the add syscall with distinct x, the double syscall), not a panic.
//! - `Fp::sqrt` / `Fp2::sqrt` take hints (`Fp2::sqrt` returns `None` on a forged one), but no path
//!   here reaches them: `map_to_curve` uses fixed addition chains.

use alloc::{vec, vec::Vec};
use alloy_primitives::{U256, hex};
use revm::precompile::{
    PrecompileHalt,
    bls12_381::{G1Point, G1PointScalar, G2Point, G2PointScalar},
    bls12_381_const::{FP_LENGTH, G1_LENGTH, G2_LENGTH, SCALAR_LENGTH},
};
use sp1_bls12_381::{
    G1Affine, G1Projective, G2Affine, G2Prepared, G2Projective, Gt, Scalar, fp::Fp, fp2::Fp2,
    hash_to_curve::MapToCurve, multi_miller_loop,
};

/// BLS12-381 base-field modulus p, big-endian (EIP-2537 "Curve parameters"; equals
/// `ark_bls12_381::Fq::MODULUS` and `sp1_bls12_381` `fp.rs` `MODULUS`).
const FP_MODULUS_BE: [u8; FP_LENGTH] = hex!(
    "1a0111ea397fe69a4b1ba7b6434bacd764774b84f38512bf6730d2a0f6b0f6241eabfffeb153ffffb9feffffffffaaab"
);

/// `MAP_FP2_TO_G2(0, 0)` output from revm-precompile 42.0.1 (arkworks and blst agree).
const MAP_FP2_TO_G2_ZERO: [u8; G2_LENGTH] = hex!(
    "018320896ec9eef9d5e619848dc29ce266f413d02dd31d9b9d44ec0c79cd61f18b075ddba6d7bd20b7ff27a4b324bfce"
    "0a67d12118b5a35bb02d2e86b3ebfa7e23410db93de39fb06d7025fa95e96ffa428a7a27c3ae4dd4b40bd251ac658892"
    "0260e03644d1a2c321256b3246bad2b895cad13890cbe6f85df55106a0d334604fb143c7a042d878006271865bc35941"
    "04c69777a43f0bda07679d5805e63f18cf4e0e7c6112ac7f70266d199b4f76ae27c6269a3ceebdae30806e9a76aadf5c"
);

/// Decodes a 48-byte big-endian Fp element (no flag bits); values ≥ p are
/// [`PrecompileHalt::NonCanonicalFp`]. The pre-check keeps non-canonical input away from the zkVM
/// `mulmod` syscalls that `Fp::from_bytes` issues unconditionally.
fn read_fp(bytes: &[u8; FP_LENGTH]) -> Result<Fp, PrecompileHalt> {
    if bytes >= &FP_MODULUS_BE {
        return Err(PrecompileHalt::NonCanonicalFp);
    }
    Option::<Fp>::from(Fp::from_bytes(bytes)).ok_or(PrecompileHalt::NonCanonicalFp)
}

/// Decodes an Fp2 element from its EIP-2537 `c0`, `c1` coordinates, `c0` first.
fn read_fp2(c0: &[u8; FP_LENGTH], c1: &[u8; FP_LENGTH]) -> Result<Fp2, PrecompileHalt> {
    Ok(Fp2 { c0: read_fp(c0)?, c1: read_fp(c1)? })
}

/// Decodes a G1 point without a subgroup check: `x`, then `y`, then (0, 0) is the identity,
/// otherwise the point must satisfy y² = x³ + 4.
fn read_g1_unchecked(x: &[u8; FP_LENGTH], y: &[u8; FP_LENGTH]) -> Result<G1Affine, PrecompileHalt> {
    let (x, y) = (read_fp(x)?, read_fp(y)?);
    if bool::from(x.is_zero() & y.is_zero()) {
        return Ok(G1Affine::identity());
    }
    let p = G1Affine { x, y, infinity: 0u8.into() };
    if !bool::from(p.is_on_curve()) {
        return Err(PrecompileHalt::Bls12381G1NotOnCurve);
    }
    Ok(p)
}

/// [`read_g1_unchecked`] followed by the G1 subgroup check. The identity passes.
fn read_g1_checked(x: &[u8; FP_LENGTH], y: &[u8; FP_LENGTH]) -> Result<G1Affine, PrecompileHalt> {
    let p = read_g1_unchecked(x, y)?;
    if !bool::from(p.is_torsion_free()) {
        return Err(PrecompileHalt::Bls12381G1NotInSubgroup);
    }
    Ok(p)
}

/// Decodes a G2 point without a subgroup check: `x0`, `x1`, `y0`, `y1`, then all-zero is the
/// identity, otherwise the point must satisfy y² = x³ + 4(1 + u).
fn read_g2_unchecked(
    x0: &[u8; FP_LENGTH],
    x1: &[u8; FP_LENGTH],
    y0: &[u8; FP_LENGTH],
    y1: &[u8; FP_LENGTH],
) -> Result<G2Affine, PrecompileHalt> {
    let x = read_fp2(x0, x1)?;
    let y = read_fp2(y0, y1)?;
    if bool::from(x.is_zero() & y.is_zero()) {
        return Ok(G2Affine::identity());
    }
    let p = G2Affine { x, y, infinity: 0u8.into() };
    if !bool::from(p.is_on_curve()) {
        return Err(PrecompileHalt::Bls12381G2NotOnCurve);
    }
    Ok(p)
}

/// [`read_g2_unchecked`] followed by the G2 subgroup check. The identity passes.
fn read_g2_checked(
    x0: &[u8; FP_LENGTH],
    x1: &[u8; FP_LENGTH],
    y0: &[u8; FP_LENGTH],
    y1: &[u8; FP_LENGTH],
) -> Result<G2Affine, PrecompileHalt> {
    let p = read_g2_unchecked(x0, x1, y0, y1)?;
    if !bool::from(p.is_torsion_free()) {
        return Err(PrecompileHalt::Bls12381G2NotInSubgroup);
    }
    Ok(p)
}

/// Reduces an arbitrary 256-bit big-endian EIP-2537 scalar mod r.
fn read_scalar(be: &[u8; SCALAR_LENGTH]) -> Scalar {
    Scalar::from_raw(U256::from_be_bytes(*be).into_limbs())
}

/// EIP-2537 G1 encoding: 96 zero bytes for the identity, otherwise `x ‖ y` big-endian.
fn encode_g1(p: &G1Affine) -> [u8; G1_LENGTH] {
    let mut out = [0u8; G1_LENGTH];
    if !bool::from(p.is_identity()) {
        out[..FP_LENGTH].copy_from_slice(&p.x.to_bytes());
        out[FP_LENGTH..].copy_from_slice(&p.y.to_bytes());
    }
    out
}

/// EIP-2537 G2 encoding: 192 zero bytes for the identity, otherwise `x.c0 ‖ x.c1 ‖ y.c0 ‖ y.c1`
/// big-endian.
fn encode_g2(p: &G2Affine) -> [u8; G2_LENGTH] {
    let mut out = [0u8; G2_LENGTH];
    if !bool::from(p.is_identity()) {
        for (chunk, fp) in out.chunks_exact_mut(FP_LENGTH).zip([p.x.c0, p.x.c1, p.y.c0, p.y.c1]) {
            chunk.copy_from_slice(&fp.to_bytes());
        }
    }
    out
}

/// G2 projective-to-affine that panics on a forged `Fp2::invert` hint (see module docs).
fn g2_to_affine(p: &G2Projective) -> G2Affine {
    let a = G2Affine::from(p);
    assert_eq!(
        bool::from(a.is_identity()),
        bool::from(p.is_identity()),
        "invalid Fp2 inverse hint"
    );
    a
}

/// G1 projective-to-affine; `Fp::invert` asserts its hint, so no check is needed (see module docs).
fn g1_to_affine(p: &G1Projective) -> G1Affine {
    G1Affine::from(p)
}

/// Pippenger bucket MSM over G2 (`sp1_bls12_381` only provides `G1Projective::msm_variable_base`),
/// per Bernstein–Doumen–Lange–Oosterwijk (<https://eprint.iacr.org/2012/549>). Window size and
/// running-sum reduction as in `msm_variable_base`; windows combined high-to-low, Horner-style.
fn g2_msm_bucket(points: &[G2Affine], scalars: &[Scalar]) -> G2Projective {
    assert_eq!(points.len(), scalars.len());
    // `msm_variable_base`'s `ln_without_floats(n) + 2`, with log2(n) as n's bit length.
    let n = points.len();
    let c = if n < 32 { 3 } else { (usize::BITS - n.leading_zeros()) as usize * 69 / 100 + 2 };
    // Canonical little-endian; every scalar is < r < 2^255.
    let scalars: Vec<[u8; 32]> = scalars.iter().map(Scalar::to_bytes).collect();
    let mut acc = G2Projective::identity();
    let mut buckets = vec![G2Projective::identity(); (1 << c) - 1];
    for window in (0..255usize.div_ceil(c)).rev() {
        for _ in 0..c {
            acc = acc.double();
        }
        buckets.fill(G2Projective::identity());
        for (point, scalar) in points.iter().zip(&scalars) {
            let digit = scalar_window(scalar, window * c, c);
            if digit != 0 {
                buckets[digit - 1] += point;
            }
        }
        let mut running = G2Projective::identity();
        for bucket in buckets.iter().rev() {
            running += bucket;
            acc += &running;
        }
    }
    acc
}

/// Returns bits `start..start + width` of the little-endian integer `le`; bits ≥ 256 read as 0.
fn scalar_window(le: &[u8; 32], start: usize, width: usize) -> usize {
    (0..width).filter(|i| start + i < 256).fold(0, |acc, i| {
        let bit = start + i;
        acc | (usize::from((le[bit / 8] >> (bit % 8)) & 1) << i)
    })
}

/// EIP-2537 G1ADD. Neither input is subgroup-checked.
pub(super) fn g1_add(a: G1Point, b: G1Point) -> Result<[u8; G1_LENGTH], PrecompileHalt> {
    let p = read_g1_unchecked(&a.0, &a.1)?;
    let q = read_g1_unchecked(&b.0, &b.1)?;
    // Complete for on-curve inputs (P = Q, P = −Q, identity); point syscalls in the zkVM.
    Ok(encode_g1(&p.add_affine(&q)))
}

/// EIP-2537 G2ADD. Neither input is subgroup-checked.
pub(super) fn g2_add(a: G2Point, b: G2Point) -> Result<[u8; G2_LENGTH], PrecompileHalt> {
    let p = read_g2_unchecked(&a.0, &a.1, &a.2, &a.3)?;
    let q = read_g2_unchecked(&b.0, &b.1, &b.2, &b.3)?;
    Ok(encode_g2(&g2_to_affine(&(G2Projective::from(&p) + q))))
}

/// MSMs with fewer terms use per-term double-and-add, which is cheaper than the bucket method's
/// fixed per-window cost.
const SMALL_MSM: usize = 4;

/// Decodes MSM pairs in item order, so the first error (padding, encoding, curve, subgroup) wins;
/// zero-scalar pairs are dropped only after their point is validated. Never collect `pairs` first.
fn read_msm_pairs<P, A>(
    pairs: &mut dyn Iterator<Item = Result<(P, [u8; SCALAR_LENGTH]), PrecompileHalt>>,
    read_point: impl Fn(&P) -> Result<A, PrecompileHalt>,
) -> Result<(Vec<A>, Vec<Scalar>), PrecompileHalt> {
    let mut points = Vec::with_capacity(pairs.size_hint().0);
    let mut scalars = Vec::with_capacity(pairs.size_hint().0);
    for item in pairs {
        let (point, scalar) = item?;
        let point = read_point(&point)?;
        if scalar.iter().all(|&b| b == 0) {
            continue;
        }
        points.push(point);
        scalars.push(read_scalar(&scalar));
    }
    Ok((points, scalars))
}

/// EIP-2537 G1MSM.
pub(super) fn g1_msm(
    pairs: &mut dyn Iterator<Item = Result<G1PointScalar, PrecompileHalt>>,
) -> Result<[u8; G1_LENGTH], PrecompileHalt> {
    let (points, scalars) =
        read_msm_pairs(pairs, |(x, y)| read_g1_checked(x, y).map(|p| G1Projective::from(&p)))?;
    if points.is_empty() {
        return Ok([0u8; G1_LENGTH]);
    }
    let sum = if points.len() < SMALL_MSM {
        points.iter().zip(&scalars).map(|(p, s)| p * s).sum()
    } else {
        G1Projective::msm_variable_base(&points, &scalars)
    };
    Ok(encode_g1(&g1_to_affine(&sum)))
}

/// EIP-2537 G2MSM.
pub(super) fn g2_msm(
    pairs: &mut dyn Iterator<Item = Result<G2PointScalar, PrecompileHalt>>,
) -> Result<[u8; G2_LENGTH], PrecompileHalt> {
    let (points, scalars) =
        read_msm_pairs(pairs, |(x0, x1, y0, y1)| read_g2_checked(x0, x1, y0, y1))?;
    if points.is_empty() {
        return Ok([0u8; G2_LENGTH]);
    }
    let sum = if points.len() < SMALL_MSM {
        points.iter().zip(&scalars).map(|(p, s)| p * s).sum()
    } else {
        g2_msm_bucket(&points, &scalars)
    };
    Ok(encode_g2(&g2_to_affine(&sum)))
}

/// EIP-2537 `PAIRING_CHECK`.
pub(super) fn pairing_check(pairs: &[(G1Point, G2Point)]) -> Result<bool, PrecompileHalt> {
    let mut kept: Vec<(G1Affine, G2Prepared)> = Vec::with_capacity(pairs.len());
    for ((g1x, g1y), (g2x0, g2x1, g2y0, g2y1)) in pairs {
        // Raw-byte zero test: a non-canonical zero (e.g. x = p) is not zero and fails decoding.
        let g1_zero = g1x.iter().chain(g1y).all(|&b| b == 0);
        let g2_zero = g2x0.iter().chain(g2x1).chain(g2y0).chain(g2y1).all(|&b| b == 0);
        if g1_zero || g2_zero {
            if !g1_zero {
                read_g1_checked(g1x, g1y)?;
            }
            if !g2_zero {
                read_g2_checked(g2x0, g2x1, g2y0, g2y1)?;
            }
            continue;
        }
        let g1 = read_g1_checked(g1x, g1y)?;
        let g2 = read_g2_checked(g2x0, g2x1, g2y0, g2y1)?;
        kept.push((g1, G2Prepared::from(g2)));
    }
    if kept.is_empty() {
        return Ok(true);
    }
    let terms: Vec<(&G1Affine, &G2Prepared)> = kept.iter().map(|(g1, g2)| (g1, g2)).collect();
    Ok(multi_miller_loop(&terms).final_exponentiation() == Gt::identity())
}

/// EIP-2537 `MAP_FP_TO_G1`.
pub(super) fn map_fp_to_g1(fp: &[u8; FP_LENGTH]) -> Result<[u8; G1_LENGTH], PrecompileHalt> {
    let q = <G1Projective as MapToCurve>::map_to_curve(&read_fp(fp)?);
    // RFC 9380 §6.6.3: isogeny-kernel inputs (check crate's `ISO_KERNEL_U`) map to the identity.
    if bool::from(q.is_identity()) {
        return Ok([0u8; G1_LENGTH]);
    }
    Ok(encode_g1(&g1_to_affine(&q.clear_h())))
}

/// EIP-2537 `MAP_FP2_TO_G2`.
pub(super) fn map_fp2_to_g2(
    c0: &[u8; FP_LENGTH],
    c1: &[u8; FP_LENGTH],
) -> Result<[u8; G2_LENGTH], PrecompileHalt> {
    let u = read_fp2(c0, c1)?;
    // `map_to_curve_simple_swu` yields the off-curve (0, 0) at u = 0 (see module docs).
    if bool::from(u.is_zero()) {
        return Ok(MAP_FP2_TO_G2_ZERO);
    }
    // #E2'(Fp2) ≡ 1 (mod 3), so no input reaches the 3-isogeny kernel (unlike G1).
    Ok(encode_g2(&g2_to_affine(&<G2Projective as MapToCurve>::map_to_curve(&u).clear_h())))
}

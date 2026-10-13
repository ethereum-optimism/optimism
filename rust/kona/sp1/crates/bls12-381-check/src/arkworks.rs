//! Vendored from revm-precompile 42.0.1 `src/bls12_381/arkworks.rs` + `pairing_common.rs`: the
//! backend the zkVM guest's `DefaultCrypto` runs. Re-vendor when bumping revm-precompile.
//! MIT-licensed; copyright and permission notice in `rust/kona/sp1/LICENSE-THIRD-PARTY`.
//!
//! Only mechanical edits: crate paths, the ark 0.6 aliases, `pub(crate)` → private, this lint
//! allowance for upstream's doc style, and the [`ArkworksCrypto`] wrapper at the end.
#![allow(clippy::doc_markdown)]

use ark_bls12_381_0_6 as ark_bls12_381;
use ark_ec_0_6 as ark_ec;
use ark_ff_0_6 as ark_ff;
use ark_serialize_0_6 as ark_serialize;
use revm::precompile::{
    Crypto, PrecompileHalt,
    bls12_381::{G1Point, G1PointScalar, G2Point, G2PointScalar},
    bls12_381_const::{FP_LENGTH, G1_LENGTH, G2_LENGTH, SCALAR_LENGTH},
};

use ark_bls12_381::{Bls12_381, Fq, Fq2, Fr, G1Affine, G1Projective, G2Affine, G2Projective};
use ark_ec::{
    AffineRepr, CurveGroup, VariableBaseMSM,
    hashing::{curve_maps::wb::WBMap, map_to_curve_hasher::MapToCurve},
    pairing::Pairing,
};
use ark_ff::{One, PrimeField, Zero};

use ark_serialize::{CanonicalDeserialize, CanonicalSerialize};
use std::vec::Vec;

type PairingPair = (G1Point, G2Point);

/// Reads a single `Fp` field element from the input slice.
///
/// Takes a byte slice in Big Endian format and attempts to interpret it as an
/// elliptic curve field element. Returns an error if the bytes do not form
/// a valid field element.
///
/// # Panics
///
/// Panics if the input is not exactly 48 bytes long.
#[inline]
fn read_fp(input_be: &[u8]) -> Result<Fq, PrecompileHalt> {
    assert_eq!(input_be.len(), FP_LENGTH, "input must be {FP_LENGTH} bytes");

    let mut input_le = [0u8; FP_LENGTH];
    input_le.copy_from_slice(input_be);

    // Reverse in-place to convert from big-endian to little-endian.
    input_le.reverse();

    Fq::deserialize_uncompressed(&input_le[..]).map_err(|_| PrecompileHalt::NonCanonicalFp)
}

/// Encodes an `Fp` field element into a big-endian byte array.
///
/// # Panics
///
/// Panics if serialization fails, which should not occur for a valid field element.
fn encode_fp(fp: &Fq) -> [u8; FP_LENGTH] {
    let mut bytes = [0u8; FP_LENGTH];
    fp.serialize_uncompressed(&mut bytes[..]).expect("Failed to serialize field element");
    bytes.reverse();
    bytes
}

/// Reads a Fp2 (quadratic extension field element) from the input slices.
///
/// Parses two Fp field elements in Big Endian format for the Fp2 element.
///
/// # Panics
///
/// Panics if either input is not exactly 48 bytes long.
#[inline]
fn read_fp2(input_1: &[u8; FP_LENGTH], input_2: &[u8; FP_LENGTH]) -> Result<Fq2, PrecompileHalt> {
    let fp_1 = read_fp(input_1)?;
    let fp_2 = read_fp(input_2)?;

    Ok(Fq2::new(fp_1, fp_2))
}

/// Creates a new `G1` point from the given `x` and `y` coordinates.
///
/// Constructs a point on the G1 curve from its affine coordinates.
///
/// Note: The point at infinity which is represented as (0,0) is
/// handled specifically.
#[inline]
fn new_g1_point_no_subgroup_check(px: Fq, py: Fq) -> Result<G1Affine, PrecompileHalt> {
    if px.is_zero() && py.is_zero() {
        Ok(G1Affine::zero())
    } else {
        // We cannot use `G1Affine::new` because that triggers an assert if the point is not on the
        // curve.
        let point = G1Affine::new_unchecked(px, py);
        if !point.is_on_curve() {
            return Err(PrecompileHalt::Bls12381G1NotOnCurve);
        }
        Ok(point)
    }
}

/// Creates a new `G2` point from the given Fq2 coordinates.
///
/// G2 points in BLS12_381 are defined over a quadratic extension field Fq2.
/// This function takes two Fq2 elements representing the x and y coordinates
/// and creates a G2 point.
///
/// Note: The point at infinity which is represented as (0,0) is
/// handled specifically.
#[inline]
fn new_g2_point_no_subgroup_check(x: Fq2, y: Fq2) -> Result<G2Affine, PrecompileHalt> {
    let point = if x.is_zero() && y.is_zero() {
        G2Affine::zero()
    } else {
        // We cannot use `G2Affine::new` because that triggers an assert if the point is not on the
        // curve.
        let point = G2Affine::new_unchecked(x, y);
        if !point.is_on_curve() {
            return Err(PrecompileHalt::Bls12381G2NotOnCurve);
        }
        point
    };

    Ok(point)
}

/// Reads a G1 point from the input slices.
///
/// Parses a G1 point from byte slices by reading two field elements
/// representing the x and y coordinates in Big Endian format.
/// Also performs a subgroup check to ensure the point is in the correct subgroup.
///
/// # Panics
///
/// Panics if the inputs are not exactly 48 bytes long.
#[inline]
fn read_g1(x: &[u8; FP_LENGTH], y: &[u8; FP_LENGTH]) -> Result<G1Affine, PrecompileHalt> {
    let point = read_g1_no_subgroup_check(x, y)?;
    if !point.is_in_correct_subgroup_assuming_on_curve() {
        return Err(PrecompileHalt::Bls12381G1NotInSubgroup);
    }
    Ok(point)
}

/// Reads a G1 point without performing a subgroup check.
///
/// Note: Skipping subgroup checks can introduce security issues.
/// This method should only be called if:
///     - The EIP specifies that no subgroup check should be performed
///     - One can be certain that the point is in the correct subgroup.
#[inline]
fn read_g1_no_subgroup_check(
    x: &[u8; FP_LENGTH],
    y: &[u8; FP_LENGTH],
) -> Result<G1Affine, PrecompileHalt> {
    let px = read_fp(x)?;
    let py = read_fp(y)?;
    new_g1_point_no_subgroup_check(px, py)
}

/// Encodes a G1 point into a byte array.
///
/// Converts a G1 point to affine coordinates and serializes the x and y coordinates
/// as big-endian byte arrays.
#[inline]
fn encode_g1_point(input: &G1Affine) -> [u8; G1_LENGTH] {
    let mut output = [0u8; G1_LENGTH];

    let Some((x, y)) = input.xy() else {
        return output; // Point at infinity, return all zeros
    };

    let x_encoded = encode_fp(&x);
    let y_encoded = encode_fp(&y);

    // Copy the encoded values to the output
    output[..FP_LENGTH].copy_from_slice(&x_encoded);
    output[FP_LENGTH..].copy_from_slice(&y_encoded);

    output
}

/// Reads a G2 point from the input slices.
///
/// Parses a G2 point from byte slices by reading four field elements
/// representing the x and y coordinates in Big Endian format.
/// Also performs a subgroup check to ensure the point is in the correct subgroup.
#[inline]
fn read_g2(
    a_x_0: &[u8; FP_LENGTH],
    a_x_1: &[u8; FP_LENGTH],
    a_y_0: &[u8; FP_LENGTH],
    a_y_1: &[u8; FP_LENGTH],
) -> Result<G2Affine, PrecompileHalt> {
    let point = read_g2_no_subgroup_check(a_x_0, a_x_1, a_y_0, a_y_1)?;
    if !point.is_in_correct_subgroup_assuming_on_curve() {
        return Err(PrecompileHalt::Bls12381G2NotInSubgroup);
    }
    Ok(point)
}

/// Reads a G2 point without performing a subgroup check.
///
/// Note: Skipping subgroup checks can introduce security issues.
/// This method should only be called if:
///     - The EIP specifies that no subgroup check should be performed
///     - One can be certain that the point is in the correct subgroup.
#[inline]
fn read_g2_no_subgroup_check(
    a_x_0: &[u8; FP_LENGTH],
    a_x_1: &[u8; FP_LENGTH],
    a_y_0: &[u8; FP_LENGTH],
    a_y_1: &[u8; FP_LENGTH],
) -> Result<G2Affine, PrecompileHalt> {
    let x = read_fp2(a_x_0, a_x_1)?;
    let y = read_fp2(a_y_0, a_y_1)?;
    new_g2_point_no_subgroup_check(x, y)
}

/// Encodes a G2 point into a byte array.
///
/// Converts a G2 point to affine coordinates and serializes the coordinates
/// as big-endian byte arrays.
#[inline]
fn encode_g2_point(input: &G2Affine) -> [u8; G2_LENGTH] {
    let mut output = [0u8; G2_LENGTH];

    let Some((x, y)) = input.xy() else {
        return output; // Point at infinity, return all zeros
    };

    let x_c0_encoded = encode_fp(&x.c0);
    let x_c1_encoded = encode_fp(&x.c1);
    let y_c0_encoded = encode_fp(&y.c0);
    let y_c1_encoded = encode_fp(&y.c1);

    output[..FP_LENGTH].copy_from_slice(&x_c0_encoded);
    output[FP_LENGTH..2 * FP_LENGTH].copy_from_slice(&x_c1_encoded);
    output[2 * FP_LENGTH..3 * FP_LENGTH].copy_from_slice(&y_c0_encoded);
    output[3 * FP_LENGTH..4 * FP_LENGTH].copy_from_slice(&y_c1_encoded);

    output
}

/// Extracts a scalar from a byte slice representation, decoding the input as a Big Endian
/// unsigned integer.
///
/// Note: We do not check that the scalar is a canonical Fr element, because the EIP specifies:
/// * The corresponding integer is not required to be less than or equal than main subgroup order.
#[inline]
fn read_scalar(input: &[u8]) -> Result<Fr, PrecompileHalt> {
    if input.len() != SCALAR_LENGTH {
        return Err(PrecompileHalt::Bls12381ScalarInputLength);
    }

    Ok(Fr::from_be_bytes_mod_order(input))
}

/// Performs point addition on two G1 points.
#[inline]
fn p1_add_affine(p1: &G1Affine, p2: &G1Affine) -> G1Affine {
    let p1_proj: G1Projective = (*p1).into();
    let p3 = p1_proj + p2;
    p3.into_affine()
}

/// Performs point addition on two G2 points.
#[inline]
fn p2_add_affine(p1: &G2Affine, p2: &G2Affine) -> G2Affine {
    let p1_proj: G2Projective = (*p1).into();
    let p3 = p1_proj + p2;
    p3.into_affine()
}

/// Performs multi-scalar multiplication (MSM) for G1 points
///
/// Takes a vector of G1 points and corresponding scalars, and returns their weighted sum
///
/// Note: This method assumes that `g1_points` does not contain any points at infinity.
#[inline]
fn p1_msm(g1_points: Vec<G1Affine>, scalars: Vec<Fr>) -> G1Affine {
    assert_eq!(
        g1_points.len(),
        scalars.len(),
        "number of scalars should equal the number of g1 points"
    );

    if g1_points.is_empty() {
        return G1Affine::zero();
    }

    if g1_points.len() == 1 {
        let big_int = scalars[0].into_bigint();
        return g1_points[0].mul_bigint(big_int).into_affine();
    }

    // Perform multi-scalar multiplication
    G1Projective::msm(&g1_points, &scalars).expect("MSM should succeed").into_affine()
}

/// Performs multi-scalar multiplication (MSM) for G2 points
///
/// Takes a vector of G2 points and corresponding scalars, and returns their weighted sum
///
/// Note: This method assumes that `g2_points` does not contain any points at infinity.
#[inline]
fn p2_msm(g2_points: Vec<G2Affine>, scalars: Vec<Fr>) -> G2Affine {
    assert_eq!(
        g2_points.len(),
        scalars.len(),
        "number of scalars should equal the number of g2 points"
    );

    if g2_points.is_empty() {
        return G2Affine::zero();
    }

    if g2_points.len() == 1 {
        let big_int = scalars[0].into_bigint();
        return g2_points[0].mul_bigint(big_int).into_affine();
    }

    // Perform multi-scalar multiplication
    G2Projective::msm(&g2_points, &scalars).expect("MSM should succeed").into_affine()
}

/// Maps a field element to a G1 point
///
/// Takes a field element (Fq) and returns the corresponding G1 point in affine form
#[inline]
fn map_fp_to_g1(fp: &Fq) -> G1Affine {
    WBMap::map_to_curve(*fp).expect("map_to_curve is infallible").clear_cofactor()
}

/// Maps a field element to a G2 point
///
/// Takes a field element (Fq2) and returns the corresponding G2 point in affine form
#[inline]
fn map_fp2_to_g2(fp2: &Fq2) -> G2Affine {
    WBMap::map_to_curve(*fp2).expect("map_to_curve is infallible").clear_cofactor()
}

/// pairing_check performs a pairing check on a list of G1 and G2 point pairs and
/// returns true if the result is equal to the identity element.
#[inline]
fn pairing_check(pairs: &[(G1Affine, G2Affine)]) -> bool {
    if pairs.is_empty() {
        return true;
    }

    let (g1_points, g2_points): (Vec<G1Affine>, Vec<G2Affine>) = pairs.iter().copied().unzip();

    let pairing_result = Bls12_381::multi_pairing(&g1_points, &g2_points);
    pairing_result.0.is_one()
}

/// pairing_check_bytes performs a pairing check on a list of G1 and G2 point pairs taking byte
/// inputs.
#[inline]
fn pairing_check_bytes(pairs: &[PairingPair]) -> Result<bool, PrecompileHalt> {
    pairing_check_bytes_generic(pairs, read_g1, read_g2, pairing_check)
}

// Byte-oriented versions of the functions for external API compatibility

/// Performs point addition on two G1 points taking byte coordinates.
#[inline]
fn p1_add_affine_bytes(a: G1Point, b: G1Point) -> Result<[u8; G1_LENGTH], PrecompileHalt> {
    let (a_x, a_y) = a;
    let (b_x, b_y) = b;
    // Parse first point
    let p1 = read_g1_no_subgroup_check(&a_x, &a_y)?;

    // Parse second point
    let p2 = read_g1_no_subgroup_check(&b_x, &b_y)?;

    // Perform addition
    let result = p1_add_affine(&p1, &p2);

    // Encode result
    Ok(encode_g1_point(&result))
}

/// Performs point addition on two G2 points taking byte coordinates.
#[inline]
fn p2_add_affine_bytes(a: G2Point, b: G2Point) -> Result<[u8; G2_LENGTH], PrecompileHalt> {
    let (a_x_0, a_x_1, a_y_0, a_y_1) = a;
    let (b_x_0, b_x_1, b_y_0, b_y_1) = b;
    // Parse first point
    let p1 = read_g2_no_subgroup_check(&a_x_0, &a_x_1, &a_y_0, &a_y_1)?;

    // Parse second point
    let p2 = read_g2_no_subgroup_check(&b_x_0, &b_x_1, &b_y_0, &b_y_1)?;

    // Perform addition
    let result = p2_add_affine(&p1, &p2);

    // Encode result
    Ok(encode_g2_point(&result))
}

/// Maps a field element to a G1 point from bytes
#[inline]
fn map_fp_to_g1_bytes(fp_bytes: &[u8; FP_LENGTH]) -> Result<[u8; G1_LENGTH], PrecompileHalt> {
    let fp = read_fp(fp_bytes)?;
    let result = map_fp_to_g1(&fp);
    Ok(encode_g1_point(&result))
}

/// Maps field elements to a G2 point from bytes
#[inline]
fn map_fp2_to_g2_bytes(
    fp2_x: &[u8; FP_LENGTH],
    fp2_y: &[u8; FP_LENGTH],
) -> Result<[u8; G2_LENGTH], PrecompileHalt> {
    let fp2 = read_fp2(fp2_x, fp2_y)?;
    let result = map_fp2_to_g2(&fp2);
    Ok(encode_g2_point(&result))
}

/// Performs multi-scalar multiplication (MSM) for G1 points taking byte inputs.
#[inline]
fn p1_msm_bytes(
    point_scalar_pairs: impl Iterator<Item = Result<(G1Point, [u8; SCALAR_LENGTH]), PrecompileHalt>>,
) -> Result<[u8; G1_LENGTH], PrecompileHalt> {
    let (lower, _) = point_scalar_pairs.size_hint();
    let mut g1_points = Vec::with_capacity(lower);
    let mut scalars = Vec::with_capacity(lower);

    // Parse all points and scalars
    for pair_result in point_scalar_pairs {
        let ((x, y), scalar_bytes) = pair_result?;

        // NB: MSM requires subgroup check
        let point = read_g1(&x, &y)?;

        // Skip zero scalars after validating the point
        if scalar_bytes.iter().all(|&b| b == 0) {
            continue;
        }

        let scalar = read_scalar(&scalar_bytes)?;
        g1_points.push(point);
        scalars.push(scalar);
    }

    // Return point at infinity if no pairs were provided or all scalars were zero
    if g1_points.is_empty() {
        return Ok([0u8; G1_LENGTH]);
    }

    // Perform MSM
    let result = p1_msm(g1_points, scalars);

    // Encode result
    Ok(encode_g1_point(&result))
}

/// Performs multi-scalar multiplication (MSM) for G2 points taking byte inputs.
#[inline]
fn p2_msm_bytes(
    point_scalar_pairs: impl Iterator<Item = Result<(G2Point, [u8; SCALAR_LENGTH]), PrecompileHalt>>,
) -> Result<[u8; G2_LENGTH], PrecompileHalt> {
    let (lower, _) = point_scalar_pairs.size_hint();
    let mut g2_points = Vec::with_capacity(lower);
    let mut scalars = Vec::with_capacity(lower);

    // Parse all points and scalars
    for pair_result in point_scalar_pairs {
        let ((x_0, x_1, y_0, y_1), scalar_bytes) = pair_result?;

        // NB: MSM requires subgroup check
        let point = read_g2(&x_0, &x_1, &y_0, &y_1)?;

        // Skip zero scalars after validating the point
        if scalar_bytes.iter().all(|&b| b == 0) {
            continue;
        }

        let scalar = read_scalar(&scalar_bytes)?;
        g2_points.push(point);
        scalars.push(scalar);
    }

    // Return point at infinity if no pairs were provided or all scalars were zero
    if g2_points.is_empty() {
        return Ok([0u8; G2_LENGTH]);
    }

    // Perform MSM
    let result = p2_msm(g2_points, scalars);

    // Encode result
    Ok(encode_g2_point(&result))
}

/// Shared implementation of `pairing_check_bytes`.
#[inline]
fn pairing_check_bytes_generic<G1, G2, ReadG1, ReadG2, PairingCheck>(
    pairs: &[PairingPair],
    read_g1: ReadG1,
    read_g2: ReadG2,
    pairing_check: PairingCheck,
) -> Result<bool, PrecompileHalt>
where
    ReadG1: Fn(&[u8; 48], &[u8; 48]) -> Result<G1, PrecompileHalt>,
    ReadG2: Fn(&[u8; 48], &[u8; 48], &[u8; 48], &[u8; 48]) -> Result<G2, PrecompileHalt>,
    PairingCheck: FnOnce(&[(G1, G2)]) -> bool,
{
    if pairs.is_empty() {
        return Ok(true);
    }

    let mut parsed_pairs = Vec::with_capacity(pairs.len());
    for ((g1_x, g1_y), (g2_x_0, g2_x_1, g2_y_0, g2_y_1)) in pairs {
        // Check if G1 point is zero (point at infinity)
        let g1_is_zero = g1_x.iter().all(|&b| b == 0) && g1_y.iter().all(|&b| b == 0);

        // Check if G2 point is zero (point at infinity)
        let g2_is_zero = g2_x_0.iter().all(|&b| b == 0) &&
            g2_x_1.iter().all(|&b| b == 0) &&
            g2_y_0.iter().all(|&b| b == 0) &&
            g2_y_1.iter().all(|&b| b == 0);

        // Skip this pair if either point is at infinity as it's a no-op
        if g1_is_zero || g2_is_zero {
            // Still need to validate the non-zero point if one exists
            if !g1_is_zero {
                let _ = read_g1(g1_x, g1_y)?;
            }
            if !g2_is_zero {
                let _ = read_g2(g2_x_0, g2_x_1, g2_y_0, g2_y_1)?;
            }
            continue;
        }

        let g1_point = read_g1(g1_x, g1_y)?;
        let g2_point = read_g2(g2_x_0, g2_x_1, g2_y_0, g2_y_1)?;
        parsed_pairs.push((g1_point, g2_point));
    }

    // If all pairs were filtered out, return true (identity element)
    if parsed_pairs.is_empty() {
        return Ok(true);
    }

    Ok(pairing_check(&parsed_pairs))
}

/// [`Crypto`] provider running only the vendored arkworks EIP-2537 backend above.
#[derive(Clone, Copy, Debug)]
pub struct ArkworksCrypto;

impl Crypto for ArkworksCrypto {
    fn bls12_381_g1_add(&self, a: G1Point, b: G1Point) -> Result<[u8; 96], PrecompileHalt> {
        p1_add_affine_bytes(a, b)
    }

    fn bls12_381_g1_msm(
        &self,
        pairs: &mut dyn Iterator<Item = Result<G1PointScalar, PrecompileHalt>>,
    ) -> Result<[u8; 96], PrecompileHalt> {
        p1_msm_bytes(pairs)
    }

    fn bls12_381_g2_add(&self, a: G2Point, b: G2Point) -> Result<[u8; 192], PrecompileHalt> {
        p2_add_affine_bytes(a, b)
    }

    fn bls12_381_g2_msm(
        &self,
        pairs: &mut dyn Iterator<Item = Result<G2PointScalar, PrecompileHalt>>,
    ) -> Result<[u8; 192], PrecompileHalt> {
        p2_msm_bytes(pairs)
    }

    fn bls12_381_pairing_check(
        &self,
        pairs: &[(G1Point, G2Point)],
    ) -> Result<bool, PrecompileHalt> {
        pairing_check_bytes(pairs)
    }

    fn bls12_381_fp_to_g1(&self, fp: &[u8; 48]) -> Result<[u8; 96], PrecompileHalt> {
        map_fp_to_g1_bytes(fp)
    }

    fn bls12_381_fp2_to_g2(&self, fp2: ([u8; 48], [u8; 48])) -> Result<[u8; 192], PrecompileHalt> {
        map_fp2_to_g2_bytes(&fp2.0, &fp2.1)
    }
}

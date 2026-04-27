//! Contains the [`PrecompileProvider`] implementation that serves FPVM-accelerated OP Stack
//! precompiles.
//!
//! [`PrecompileProvider`]: revm::handler::PrecompileProvider

#![allow(clippy::redundant_pub_crate, clippy::large_stack_arrays)]
// SAFETY: `pub(crate)` items are required to satisfy the workspace-level `unreachable_pub` lint.
// Large arrays in BLS12/BN128 helpers are a fixed-size scratch buffer that the FPVM ABI requires.

mod provider;
pub(crate) use provider::OpFpvmPrecompiles;

mod bls12_g1_add;
mod bls12_g1_msm;
mod bls12_g2_add;
mod bls12_g2_msm;
mod bls12_map_fp;
mod bls12_map_fp2;
mod bls12_pair;
mod bn128_pair;
mod ecrecover;
mod kzg_point_eval;
mod utils;

#[cfg(test)]
mod test_utils;

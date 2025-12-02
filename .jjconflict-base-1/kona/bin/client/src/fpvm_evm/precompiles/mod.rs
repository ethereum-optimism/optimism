//! Contains the [`PrecompileProvider`] implementation that serves FPVM-accelerated OP Stack
//! precompiles.
//!
//! [`PrecompileProvider`]: revm::handler::PrecompileProvider

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

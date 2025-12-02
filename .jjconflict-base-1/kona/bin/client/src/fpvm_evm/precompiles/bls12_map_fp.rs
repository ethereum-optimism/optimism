//! Contains the accelerated precompile for the BLS12-381 curve FP to G1 Mapping.
//!
//! BLS12-381 is introduced in [EIP-2537](https://eips.ethereum.org/EIPS/eip-2537).
//!
//! For constants and logic, see the [revm implementation].
//!
//! [revm implementation]: https://github.com/bluealloy/revm/blob/main/crates/precompile/src/bls12_381/map_fp_to_g1.rs

use crate::fpvm_evm::precompiles::utils::precompile_run;
use alloc::string::ToString;
use kona_preimage::{HintWriterClient, PreimageOracleClient};
use revm::precompile::{
    PrecompileError, PrecompileOutput, PrecompileResult, bls12_381,
    bls12_381_const::{MAP_FP_TO_G1_BASE_GAS_FEE, PADDED_FP_LENGTH},
};

/// Performs an FPVM-accelerated BLS12-381 map fp check.
///
/// Notice, there is no input size limit for this precompile.
/// See: <https://specs.optimism.io/protocol/isthmus/exec-engine.html#evm-changes>
pub(crate) fn fpvm_bls12_map_fp<H, O>(
    input: &[u8],
    gas_limit: u64,
    hint_writer: &H,
    oracle_reader: &O,
) -> PrecompileResult
where
    H: HintWriterClient + Send + Sync,
    O: PreimageOracleClient + Send + Sync,
{
    if MAP_FP_TO_G1_BASE_GAS_FEE > gas_limit {
        return Err(PrecompileError::OutOfGas);
    }

    if input.len() != PADDED_FP_LENGTH {
        return Err(PrecompileError::Other(
            alloc::format!(
                "MAP_FP_TO_G1 input should be {PADDED_FP_LENGTH} bytes, was {}",
                input.len()
            )
            .into(),
        ));
    }

    let precompile = bls12_381::map_fp_to_g1::PRECOMPILE;

    let result_data = kona_proof::block_on(precompile_run! {
        hint_writer,
        oracle_reader,
        &[precompile.address().as_slice(), &MAP_FP_TO_G1_BASE_GAS_FEE.to_be_bytes(), input]
    })
    .map_err(|e| PrecompileError::Other(e.to_string().into()))?;

    Ok(PrecompileOutput::new(MAP_FP_TO_G1_BASE_GAS_FEE, result_data.into()))
}

#[cfg(test)]
mod test {
    use alloy_primitives::hex;

    use super::*;
    use crate::fpvm_evm::precompiles::test_utils::{
        execute_native_precompile, test_accelerated_precompile,
    };

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_bls12_381_map_fp_g1() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            // https://github.com/ethereum/execution-spec-tests/blob/a1c4eeff347a64ad6c5aedd51314d4ffc067346b/tests/prague/eip2537_bls_12_381_precompiles/vectors/map_fp_to_G1_bls.json
            let input = hex!("00000000000000000000000000000000156c8a6a2c184569d69a76be144b5cdc5141d2d2ca4fe341f011e25e3969c55ad9e9b9ce2eb833c81a908e5fa4ac5f03");
            let expected = hex!("00000000000000000000000000000000184bb665c37ff561a89ec2122dd343f20e0f4cbcaec84e3c3052ea81d1834e192c426074b02ed3dca4e7676ce4ce48ba0000000000000000000000000000000004407b8d35af4dacc809927071fc0405218f1401a6d15af775810e4e460064bcc9468beeba82fdc751be70476c888bf3");

            let accelerated_result = fpvm_bls12_map_fp(&input, 5500, hint_writer, oracle_reader).unwrap();
            let native_result = execute_native_precompile(*bls12_381::map_fp_to_g1::PRECOMPILE.address(), input, 5500).unwrap();

            assert_eq!(accelerated_result.bytes.as_ref(), expected.as_ref());
            assert_eq!(accelerated_result.bytes, native_result.bytes);
            assert_eq!(accelerated_result.gas_used, native_result.gas_used);
        })
        .await;
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_bls12_381_map_fp_g1_bad_gas_limit() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let accelerated_result =
                fpvm_bls12_map_fp(&[], 0, hint_writer, oracle_reader).unwrap_err();
            assert!(matches!(accelerated_result, PrecompileError::OutOfGas));
        })
        .await;
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_bls12_381_map_fp_g1_bad_input_size() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let accelerated_result =
                fpvm_bls12_map_fp(&[], u64::MAX, hint_writer, oracle_reader).unwrap_err();
            assert!(matches!(accelerated_result, PrecompileError::Other(_)));
        })
        .await;
    }
}

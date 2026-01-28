//! Contains the accelerated precompile for the BLS12-381 curve FP2 to G2 Mapping.
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
    bls12_381_const::{MAP_FP2_TO_G2_BASE_GAS_FEE, PADDED_FP2_LENGTH},
};

/// Performs an FPVM-accelerated BLS12-381 map fp2 check.
///
/// Notice, there is no input size limit for this precompile.
/// See: <https://specs.optimism.io/protocol/isthmus/exec-engine.html#evm-changes>
pub(crate) fn fpvm_bls12_map_fp2<H, O>(
    input: &[u8],
    gas_limit: u64,
    hint_writer: &H,
    oracle_reader: &O,
) -> PrecompileResult
where
    H: HintWriterClient + Send + Sync,
    O: PreimageOracleClient + Send + Sync,
{
    if MAP_FP2_TO_G2_BASE_GAS_FEE > gas_limit {
        return Err(PrecompileError::OutOfGas);
    }

    if input.len() != PADDED_FP2_LENGTH {
        return Err(PrecompileError::Other(
            alloc::format!(
                "MAP_FP2_TO_G2 input should be {PADDED_FP2_LENGTH} bytes, was {}",
                input.len()
            )
            .into(),
        ));
    }

    let precompile = bls12_381::map_fp2_to_g2::PRECOMPILE;

    let result_data = kona_proof::block_on(precompile_run! {
        hint_writer,
        oracle_reader,
        &[precompile.address().as_slice(), &MAP_FP2_TO_G2_BASE_GAS_FEE.to_be_bytes(), input]
    })
    .map_err(|e| PrecompileError::Other(e.to_string().into()))?;

    Ok(PrecompileOutput::new(MAP_FP2_TO_G2_BASE_GAS_FEE, result_data.into()))
}

#[cfg(test)]
mod test {
    use alloy_primitives::hex;

    use super::*;
    use crate::fpvm_evm::precompiles::test_utils::{
        execute_native_precompile, test_accelerated_precompile,
    };

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_bls12_381_map_fp_g2() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            // https://github.com/ethereum/execution-spec-tests/blob/a1c4eeff347a64ad6c5aedd51314d4ffc067346b/tests/prague/eip2537_bls_12_381_precompiles/vectors/map_fp2_to_G2_bls.json
            let input = hex!("0000000000000000000000000000000007355d25caf6e7f2f0cb2812ca0e513bd026ed09dda65b177500fa31714e09ea0ded3a078b526bed3307f804d4b93b040000000000000000000000000000000002829ce3c021339ccb5caf3e187f6370e1e2a311dec9b75363117063ab2015603ff52c3d3b98f19c2f65575e99e8b78c");
            let expected = hex!("0000000000000000000000000000000000e7f4568a82b4b7dc1f14c6aaa055edf51502319c723c4dc2688c7fe5944c213f510328082396515734b6612c4e7bb700000000000000000000000000000000126b855e9e69b1f691f816e48ac6977664d24d99f8724868a184186469ddfd4617367e94527d4b74fc86413483afb35b000000000000000000000000000000000caead0fd7b6176c01436833c79d305c78be307da5f6af6c133c47311def6ff1e0babf57a0fb5539fce7ee12407b0a42000000000000000000000000000000001498aadcf7ae2b345243e281ae076df6de84455d766ab6fcdaad71fab60abb2e8b980a440043cd305db09d283c895e3d");

            let accelerated_result = fpvm_bls12_map_fp2(&input, 23800, hint_writer, oracle_reader).unwrap();
            let native_result = execute_native_precompile(*bls12_381::map_fp2_to_g2::PRECOMPILE.address(), input, 23800).unwrap();

            assert_eq!(accelerated_result.bytes.as_ref(), expected.as_ref());
            assert_eq!(accelerated_result.bytes, native_result.bytes);
            assert_eq!(accelerated_result.gas_used, native_result.gas_used);
        })
        .await;
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_bls12_381_map_fp_g2_bad_gas_limit() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let accelerated_result =
                fpvm_bls12_map_fp2(&[], 0, hint_writer, oracle_reader).unwrap_err();
            assert!(matches!(accelerated_result, PrecompileError::OutOfGas));
        })
        .await;
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_bls12_381_map_fp_g2_bad_input_size() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let accelerated_result =
                fpvm_bls12_map_fp2(&[], u64::MAX, hint_writer, oracle_reader).unwrap_err();
            assert!(matches!(accelerated_result, PrecompileError::Other(_)));
        })
        .await;
    }
}

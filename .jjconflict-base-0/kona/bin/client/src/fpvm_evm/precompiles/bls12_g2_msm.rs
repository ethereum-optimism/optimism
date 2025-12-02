//! Contains the accelerated precompile for the BLS12-381 curve G2 MSM.
//!
//! BLS12-381 is introduced in [EIP-2537](https://eips.ethereum.org/EIPS/eip-2537).
//!
//! For constants and logic, see the [revm implementation].
//!
//! [revm implementation]: https://github.com/bluealloy/revm/blob/main/crates/precompile/src/bls12_381/g2_msm.rs

use crate::fpvm_evm::precompiles::utils::precompile_run;
use alloc::string::ToString;
use kona_preimage::{HintWriterClient, PreimageOracleClient};
use revm::precompile::{
    PrecompileError, PrecompileOutput, PrecompileResult, bls12_381,
    bls12_381_const::{DISCOUNT_TABLE_G2_MSM, G2_MSM_BASE_GAS_FEE, G2_MSM_INPUT_LENGTH},
    bls12_381_utils::msm_required_gas,
};

/// The maximum input size for the BLS12-381 g2 msm operation after the Isthmus Hardfork.
///
/// See: <https://specs.optimism.io/protocol/isthmus/exec-engine.html#evm-changes>
const BLS12_MAX_G2_MSM_SIZE_ISTHMUS: usize = 488448;

/// The maximum input size for the BLS12-381 g2 msm operation after the Jovian Hardfork.
const BLS12_MAX_G2_MSM_SIZE_JOVIAN: usize = 278_784;

/// Performs an FPVM-accelerated BLS12-381 G2 msm check after the Isthmus Hardfork.
pub(crate) fn fpvm_bls12_g2_msm<H, O>(
    input: &[u8],
    gas_limit: u64,
    hint_writer: &H,
    oracle_reader: &O,
) -> PrecompileResult
where
    H: HintWriterClient + Send + Sync,
    O: PreimageOracleClient + Send + Sync,
{
    let input_len = input.len();

    if input_len > BLS12_MAX_G2_MSM_SIZE_ISTHMUS {
        return Err(PrecompileError::Other(
            alloc::format!("G2MSM input length must be at most {BLS12_MAX_G2_MSM_SIZE_ISTHMUS}")
                .into(),
        ));
    }

    if input_len == 0 || !input_len.is_multiple_of(G2_MSM_INPUT_LENGTH) {
        return Err(PrecompileError::Other(
            alloc::format!(
                "G2MSM input length should be multiple of {G2_MSM_INPUT_LENGTH}, was {input_len}"
            )
            .into(),
        ));
    }

    let k = input_len / G2_MSM_INPUT_LENGTH;
    let required_gas = msm_required_gas(k, &DISCOUNT_TABLE_G2_MSM, G2_MSM_BASE_GAS_FEE);
    if required_gas > gas_limit {
        return Err(PrecompileError::OutOfGas);
    }

    let precompile = bls12_381::g2_msm::PRECOMPILE;

    let result_data = kona_proof::block_on(precompile_run! {
        hint_writer,
        oracle_reader,
        &[precompile.address().as_slice(), &required_gas.to_be_bytes(), input]
    })
    .map_err(|e| PrecompileError::Other(e.to_string().into()))?;

    Ok(PrecompileOutput::new(required_gas, result_data.into()))
}

/// Performs an FPVM-accelerated BLS12-381 G2 msm check after the Jovian Hardfork.
pub(crate) fn fpvm_bls12_g2_msm_jovian<H, O>(
    input: &[u8],
    gas_limit: u64,
    hint_writer: &H,
    oracle_reader: &O,
) -> PrecompileResult
where
    H: HintWriterClient + Send + Sync,
    O: PreimageOracleClient + Send + Sync,
{
    if input.len() > BLS12_MAX_G2_MSM_SIZE_JOVIAN {
        return Err(PrecompileError::Other(
            alloc::format!("G2MSM input length must be at most {BLS12_MAX_G2_MSM_SIZE_JOVIAN}")
                .into(),
        ));
    }

    fpvm_bls12_g2_msm(input, gas_limit, hint_writer, oracle_reader)
}

#[cfg(test)]
mod test {
    use alloy_primitives::hex;

    use super::*;
    use crate::fpvm_evm::precompiles::test_utils::{
        execute_native_precompile, test_accelerated_precompile,
    };

    // https://raw.githubusercontent.com/ethereum/execution-spec-tests/a1c4eeff347a64ad6c5aedd51314d4ffc067346b/tests/prague/eip2537_bls_12_381_precompiles/vectors/msm_G2_bls.json
    const TEST_INPUT: [u8; 288] = hex!(
        "00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051c6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb80000000000000000000000000000000013e02b6052719f607dacd3a088274f65596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351aadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99cb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be0000000000000000000000000000000000000000000000000000000000000002"
    );
    const EXPECTED_OUTPUT: [u8; 256] = hex!(
        "000000000000000000000000000000001638533957d540a9d2370f17cc7ed5863bc0b995b8825e0ee1ea1e1e4d00dbae81f14b0bf3611b78c952aacab827a053000000000000000000000000000000000a4edef9c1ed7f729f520e47730a124fd70662a904ba1074728114d1031e1572c6c886f6b57ec72a6178288c47c33577000000000000000000000000000000000468fb440d82b0630aeb8dca2b5256789a66da69bf91009cbfe6bd221e47aa8ae88dece9764bf3bd999d95d71e4c9899000000000000000000000000000000000f6d4552fa65dd2638b361543f887136a43253d9c66c411697003f7a13c308f5422e1aa0a59c8967acdefd8b6e36ccf3"
    );

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_bls12_381_g2_msm() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let accelerated_result =
                fpvm_bls12_g2_msm(TEST_INPUT.as_ref(), 22500, hint_writer, oracle_reader).unwrap();
            let native_result = execute_native_precompile(
                *bls12_381::g2_msm::PRECOMPILE.address(),
                TEST_INPUT.as_ref(),
                22500,
            )
            .unwrap();

            assert_eq!(accelerated_result.bytes.as_ref(), EXPECTED_OUTPUT.as_ref());
            assert_eq!(accelerated_result.bytes, native_result.bytes);
            assert_eq!(accelerated_result.gas_used, native_result.gas_used);
        })
        .await;
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_bls12_381_g2_msm_bad_input_len_isthmus() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let accelerated_result = fpvm_bls12_g2_msm(
                &[0u8; BLS12_MAX_G2_MSM_SIZE_ISTHMUS + 1],
                u64::MAX,
                hint_writer,
                oracle_reader,
            )
            .unwrap_err();
            assert!(matches!(accelerated_result, PrecompileError::Other(_)));
        })
        .await;
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_bls12_381_g2_msm_bad_input_len() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let accelerated_result =
                fpvm_bls12_g2_msm(&[], u64::MAX, hint_writer, oracle_reader).unwrap_err();
            assert!(matches!(accelerated_result, PrecompileError::Other(_)));
        })
        .await;
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_bls12_381_g2_msm_bad_gas_limit() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let accelerated_result =
                fpvm_bls12_g2_msm(&[0u8; G2_MSM_INPUT_LENGTH], 0, hint_writer, oracle_reader)
                    .unwrap_err();
            assert!(matches!(accelerated_result, PrecompileError::OutOfGas));
        })
        .await;
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_bls12_381_g2_msm_jovian() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let base_result =
                fpvm_bls12_g2_msm(TEST_INPUT.as_ref(), 22500, hint_writer, oracle_reader).unwrap();
            let jovian_result =
                fpvm_bls12_g2_msm_jovian(TEST_INPUT.as_ref(), 22500, hint_writer, oracle_reader)
                    .unwrap();

            assert_eq!(base_result.bytes, jovian_result.bytes);
            assert_eq!(base_result.gas_used, jovian_result.gas_used);
        })
        .await;
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_bls12_381_g2_msm_bad_input_len_jovian() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            // Calculate the next aligned size (multiple of G2_MSM_INPUT_LENGTH) that exceeds
            // BLS12_MAX_G2_MSM_SIZE_JOVIAN
            const INPUT_SIZE: usize =
                ((BLS12_MAX_G2_MSM_SIZE_JOVIAN / G2_MSM_INPUT_LENGTH) + 1) * G2_MSM_INPUT_LENGTH;
            let input = [0u8; INPUT_SIZE];
            let accelerated_result =
                fpvm_bls12_g2_msm_jovian(&input, u64::MAX, hint_writer, oracle_reader).unwrap_err();
            assert!(matches!(accelerated_result, PrecompileError::Other(_)));
        })
        .await;
    }
}

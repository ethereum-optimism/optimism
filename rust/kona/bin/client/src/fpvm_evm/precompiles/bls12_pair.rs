//! Contains the accelerated precompile for the BLS12-381 curve.
//!
//! BLS12-381 is introduced in [EIP-2537](https://eips.ethereum.org/EIPS/eip-2537).
//!
//! For constants and logic, see the [revm implementation].
//!
//! [revm implementation]: https://github.com/bluealloy/revm/blob/main/crates/precompile/src/bls12_381/pairing.rs

use crate::fpvm_evm::precompiles::utils::precompile_run;
use alloc::string::ToString;
use kona_preimage::{HintWriterClient, PreimageOracleClient};
use revm::precompile::{
    PrecompileError, PrecompileOutput, PrecompileResult, bls12_381,
    bls12_381_const::{PAIRING_INPUT_LENGTH, PAIRING_MULTIPLIER_BASE, PAIRING_OFFSET_BASE},
};

/// The max pairing size for BLS12-381 input given a 20M gas limit.
const BLS12_MAX_PAIRING_SIZE_ISTHMUS: usize = 235_008;

/// The max pairing size for BLS12-381 input after the Jovian Hardfork.
const BLS12_MAX_PAIRING_SIZE_JOVIAN: usize = 156_672;

/// Performs an FPVM-accelerated BLS12-381 pairing check after the Isthmus Hardfork.
pub(crate) fn fpvm_bls12_pairing<H, O>(
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

    if input_len > BLS12_MAX_PAIRING_SIZE_ISTHMUS {
        return Err(PrecompileError::Other(
            alloc::format!("Pairing input length must be at most {BLS12_MAX_PAIRING_SIZE_ISTHMUS}")
                .into(),
        ));
    }

    if !input_len.is_multiple_of(PAIRING_INPUT_LENGTH) {
        return Err(PrecompileError::Other(
            alloc::format!(
                "Pairing input length should be multiple of {PAIRING_INPUT_LENGTH}, was {input_len}"
            )
            .into(),
        ));
    }

    let k = input_len / PAIRING_INPUT_LENGTH;
    let required_gas: u64 = PAIRING_MULTIPLIER_BASE * k as u64 + PAIRING_OFFSET_BASE;
    if required_gas > gas_limit {
        return Err(PrecompileError::OutOfGas);
    }

    let precompile = bls12_381::pairing::PRECOMPILE;

    let result_data = kona_proof::block_on(precompile_run! {
        hint_writer,
        oracle_reader,
        &[precompile.address().as_slice(), &required_gas.to_be_bytes(), input]
    })
    .map_err(|e| PrecompileError::Other(e.to_string().into()))?;

    Ok(PrecompileOutput::new(required_gas, result_data.into()))
}

/// Performs an FPVM-accelerated BLS12-381 pairing check after the Jovian Hardfork.
pub(crate) fn fpvm_bls12_pairing_jovian<H, O>(
    input: &[u8],
    gas_limit: u64,
    hint_writer: &H,
    oracle_reader: &O,
) -> PrecompileResult
where
    H: HintWriterClient + Send + Sync,
    O: PreimageOracleClient + Send + Sync,
{
    if input.len() > BLS12_MAX_PAIRING_SIZE_JOVIAN {
        return Err(PrecompileError::Other(
            alloc::format!("Pairing input length must be at most {BLS12_MAX_PAIRING_SIZE_JOVIAN}")
                .into(),
        ));
    }

    fpvm_bls12_pairing(input, gas_limit, hint_writer, oracle_reader)
}

#[cfg(test)]
mod test {
    use super::*;
    use crate::fpvm_evm::precompiles::test_utils::{
        execute_native_precompile, test_accelerated_precompile,
    };
    use alloy_primitives::hex;

    // https://github.com/ethereum/execution-spec-tests/blob/a1c4eeff347a64ad6c5aedd51314d4ffc067346b/tests/prague/eip2537_bls_12_381_precompiles/vectors/pairing_check_bls.json
    const TEST_INPUT: [u8; 384] = hex!(
        "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
    );
    const EXPECTED_OUTPUT: [u8; 32] =
        hex!("0000000000000000000000000000000000000000000000000000000000000001");

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_bls12_381_pairing() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let accelerated_result =
                fpvm_bls12_pairing(TEST_INPUT.as_ref(), 70300, hint_writer, oracle_reader).unwrap();
            let native_result = execute_native_precompile(
                *bls12_381::pairing::PRECOMPILE.address(),
                TEST_INPUT.as_ref(),
                70300,
            )
            .unwrap();

            assert_eq!(accelerated_result.bytes.as_ref(), EXPECTED_OUTPUT.as_ref());
            assert_eq!(accelerated_result.bytes, native_result.bytes);
            assert_eq!(accelerated_result.gas_used, native_result.gas_used);
        })
        .await;
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_bls12_381_pairing_bad_input_len_isthmus() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let accelerated_result = fpvm_bls12_pairing(
                &[0u8; BLS12_MAX_PAIRING_SIZE_ISTHMUS + 1],
                0,
                hint_writer,
                oracle_reader,
            )
            .unwrap_err();
            assert!(matches!(accelerated_result, PrecompileError::Other(_)));
        })
        .await;
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_bls12_381_pairing_bad_input_len() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let accelerated_result =
                fpvm_bls12_pairing(&[0u8; PAIRING_INPUT_LENGTH - 1], 0, hint_writer, oracle_reader)
                    .unwrap_err();
            assert!(matches!(accelerated_result, PrecompileError::Other(_)));
        })
        .await;
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_bls12_381_pairing_bad_gas_limit() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let accelerated_result =
                fpvm_bls12_pairing(&[0u8; PAIRING_INPUT_LENGTH], 0, hint_writer, oracle_reader)
                    .unwrap_err();
            assert!(matches!(accelerated_result, PrecompileError::OutOfGas));
        })
        .await;
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_bls12_381_pairing_jovian() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let base_result =
                fpvm_bls12_pairing(TEST_INPUT.as_ref(), 70300, hint_writer, oracle_reader).unwrap();
            let jovian_result =
                fpvm_bls12_pairing_jovian(TEST_INPUT.as_ref(), 70300, hint_writer, oracle_reader)
                    .unwrap();

            assert_eq!(base_result.bytes, jovian_result.bytes);
            assert_eq!(base_result.gas_used, jovian_result.gas_used);
        })
        .await;
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_bls12_381_pairing_bad_input_len_jovian() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            // Calculate the next aligned size (multiple of PAIRING_INPUT_LENGTH) that exceeds
            // BLS12_MAX_PAIRING_SIZE_JOVIAN
            const INPUT_SIZE: usize =
                ((BLS12_MAX_PAIRING_SIZE_JOVIAN / PAIRING_INPUT_LENGTH) + 1) * PAIRING_INPUT_LENGTH;
            let input = [0u8; INPUT_SIZE];
            let accelerated_result =
                fpvm_bls12_pairing_jovian(&input, u64::MAX, hint_writer, oracle_reader)
                    .unwrap_err();
            assert!(matches!(accelerated_result, PrecompileError::Other(_)));
        })
        .await;
    }
}

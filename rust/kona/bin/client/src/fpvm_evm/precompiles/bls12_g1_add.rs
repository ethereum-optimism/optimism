//! Contains the accelerated precompile for the BLS12-381 curve G1 Point Addition.
//!
//! BLS12-381 is introduced in [EIP-2537](https://eips.ethereum.org/EIPS/eip-2537).
//!
//! For constants and logic, see the [revm implementation].
//!
//! [revm implementation]: https://github.com/bluealloy/revm/blob/main/crates/precompile/src/bls12_381/g1_add.rs

use crate::fpvm_evm::precompiles::utils::precompile_run;
use alloc::string::ToString;
use kona_preimage::{HintWriterClient, PreimageOracleClient};
use revm::precompile::{
    EthPrecompileOutput, EthPrecompileResult, PrecompileHalt, bls12_381,
    bls12_381_const::{G1_ADD_BASE_GAS_FEE, G1_ADD_INPUT_LENGTH},
};

/// Performs an FPVM-accelerated BLS12-381 G1 addition check.
///
/// Notice, there is no input size limit for this precompile.
/// See: <https://specs.optimism.io/protocol/isthmus/exec-engine.html#evm-changes>
pub(crate) fn fpvm_bls12_g1_add<H, O>(
    input: &[u8],
    gas_limit: u64,
    hint_writer: &H,
    oracle_reader: &O,
) -> EthPrecompileResult
where
    H: HintWriterClient + Send + Sync,
    O: PreimageOracleClient + Send + Sync,
{
    /// Oracle/L1 staticcall gas required by [EIP-7904](https://eips.ethereum.org/EIPS/eip-7904).
    const G1_ADD_ORACLE_GAS: u64 = 643;

    if G1_ADD_BASE_GAS_FEE > gas_limit {
        return Err(PrecompileHalt::OutOfGas);
    }

    let input_len = input.len();
    if input_len != G1_ADD_INPUT_LENGTH {
        return Err(PrecompileHalt::Other(
            alloc::format!(
                "G1 addition input length should be {G1_ADD_INPUT_LENGTH} bytes, was {input_len}"
            )
            .into(),
        ));
    }

    let precompile = bls12_381::g1_add::PRECOMPILE;

    let result_data = kona_proof::block_on(precompile_run! {
        hint_writer,
        oracle_reader,
        &[precompile.address().as_slice(), &G1_ADD_ORACLE_GAS.to_be_bytes(), input]
    })
    .map_err(|e| PrecompileHalt::Other(e.to_string().into()))?;

    Ok(EthPrecompileOutput::new(G1_ADD_BASE_GAS_FEE, result_data.into()))
}

#[cfg(test)]
mod test {
    use super::*;
    use crate::fpvm_evm::precompiles::test_utils::{
        execute_native_precompile, test_accelerated_precompile,
        test_accelerated_precompile_capture_hint,
    };

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_bls12_381_g1_add() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            // G1.INF + G1.INF = G1.INF
            let input = [0u8; G1_ADD_INPUT_LENGTH];
            let accelerated_result =
                fpvm_bls12_g1_add(&input, u64::MAX, hint_writer, oracle_reader).unwrap();
            let native_result = execute_native_precompile(
                *bls12_381::g1_add::PRECOMPILE.address(),
                input,
                u64::MAX,
            )
            .unwrap();

            assert_eq!(accelerated_result.bytes, native_result.bytes);
            assert_eq!(accelerated_result.gas_used, native_result.gas_used);
        })
        .await;
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_bls12_381_g1_add_bad_input_len() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let accelerated_result =
                fpvm_bls12_g1_add(&[], u64::MAX, hint_writer, oracle_reader).unwrap_err();
            assert!(matches!(accelerated_result, PrecompileHalt::Other(_)));
        })
        .await;
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_bls12_381_g1_add_bad_gas_limit() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let accelerated_result =
                fpvm_bls12_g1_add(&[], 0, hint_writer, oracle_reader).unwrap_err();
            assert!(matches!(accelerated_result, PrecompileHalt::OutOfGas));
        })
        .await;
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_bls12_381_g1_add_oracle_gas_carries_l1_cost() {
        let recorded_gas_used: std::sync::Arc<std::sync::Mutex<Option<u64>>> =
            std::sync::Arc::new(std::sync::Mutex::new(None));
        let recorded_gas_used_in = recorded_gas_used.clone();

        let captured =
            test_accelerated_precompile_capture_hint(move |hint_writer, oracle_reader| {
                let input = [0u8; G1_ADD_INPUT_LENGTH];
                let result =
                    fpvm_bls12_g1_add(&input, u64::MAX, hint_writer, oracle_reader).unwrap();
                *recorded_gas_used_in.lock().unwrap() = Some(result.gas_used);
            })
            .await;

        // Oracle hint must carry the current L1 G1Add cost (EIP-7904: 643).
        assert_eq!(captured.oracle_gas(), 643);
        // L2 charge must remain unchanged (G1_ADD_BASE_GAS_FEE = 375).
        assert_eq!(recorded_gas_used.lock().unwrap().unwrap(), G1_ADD_BASE_GAS_FEE);
    }
}

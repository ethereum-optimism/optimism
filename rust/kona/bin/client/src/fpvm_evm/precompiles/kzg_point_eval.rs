//! Contains the accelerated version of the KZG point evaluation precompile.

use crate::fpvm_evm::precompiles::utils::precompile_run;
use alloc::string::ToString;
use alloy_primitives::Address;
use kona_preimage::{HintWriterClient, PreimageOracleClient};
use revm::precompile::{EthPrecompileOutput, EthPrecompileResult, PrecompileHalt};

/// Address of the KZG point evaluation precompile.
pub(crate) const KZG_POINT_EVAL_ADDR: Address = revm::precompile::u64_to_address(0x0A);

/// Runs the FPVM-accelerated `kzgPointEval` precompile call.
pub(crate) fn fpvm_kzg_point_eval<H, O>(
    input: &[u8],
    gas_limit: u64,
    hint_writer: &H,
    oracle_reader: &O,
) -> EthPrecompileResult
where
    H: HintWriterClient + Send + Sync,
    O: PreimageOracleClient + Send + Sync,
{
    const GAS_COST: u64 = 50_000;
    /// Oracle/L1 staticcall gas required by [EIP-7904](https://eips.ethereum.org/EIPS/eip-7904).
    const KZG_ORACLE_GAS: u64 = 89_363;

    if gas_limit < GAS_COST {
        return Err(PrecompileHalt::OutOfGas);
    }

    if input.len() != 192 {
        return Err(PrecompileHalt::BlobInvalidInputLength);
    }

    let result_data = kona_proof::block_on(precompile_run! {
        hint_writer,
        oracle_reader,
        &[KZG_POINT_EVAL_ADDR.as_slice(), &KZG_ORACLE_GAS.to_be_bytes(), input]
    })
    .map_err(|e| PrecompileHalt::Other(e.to_string().into()))?;

    Ok(EthPrecompileOutput::new(GAS_COST, result_data.into()))
}

#[cfg(test)]
mod test {
    use super::*;
    use crate::fpvm_evm::precompiles::test_utils::{
        VALID_KZG_EVALUATION, execute_native_precompile, kzg_point_eval_input,
        test_accelerated_precompile, test_accelerated_precompile_capture_hint,
    };
    use alloy_primitives::hex;

    fn valid_kzg_input() -> Vec<u8> {
        kzg_point_eval_input(VALID_KZG_EVALUATION)
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_kzg_point_eval() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let input = valid_kzg_input();

            let expected_result = hex!("000000000000000000000000000000000000000000000000000000000000100073eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000001");

            let accelerated_result = fpvm_kzg_point_eval(&input, u64::MAX, hint_writer, oracle_reader).unwrap();
            let native_result = execute_native_precompile(KZG_POINT_EVAL_ADDR, input, u64::MAX).unwrap();

            assert_eq!(accelerated_result.bytes.as_ref(), expected_result.as_ref());
            assert_eq!(accelerated_result.bytes, native_result.bytes);
            assert_eq!(accelerated_result.gas_used, native_result.gas_used);
        })
        .await;
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_kzg_point_eval_oracle_gas_carries_l1_cost() {
        let recorded_gas_used: std::sync::Arc<std::sync::Mutex<Option<u64>>> =
            std::sync::Arc::new(std::sync::Mutex::new(None));
        let recorded_gas_used_in = recorded_gas_used.clone();

        let captured =
            test_accelerated_precompile_capture_hint(move |hint_writer, oracle_reader| {
                let input = valid_kzg_input();
                let result =
                    fpvm_kzg_point_eval(&input, u64::MAX, hint_writer, oracle_reader).unwrap();
                *recorded_gas_used_in.lock().unwrap() = Some(result.gas_used);
            })
            .await;

        // Oracle hint must carry the current L1 KZG cost (EIP-7904: 89_363).
        assert_eq!(captured.oracle_gas(), 89_363);
        // L2 charge must remain unchanged (Cancun KZG = 50_000).
        assert_eq!(recorded_gas_used.lock().unwrap().unwrap(), 50_000);
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_kzg_point_eval_at_l2_gas_boundary() {
        // OOG guard is `<` not `<=`, so gas_limit == GAS_COST must succeed.
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let input = valid_kzg_input();
            let result = fpvm_kzg_point_eval(&input, 50_000, hint_writer, oracle_reader);
            assert!(result.is_ok(), "expected Ok at gas_limit == GAS_COST, got {result:?}");
        })
        .await;
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_kzg_point_eval_out_of_gas() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let accelerated_result =
                fpvm_kzg_point_eval(&[], 0, hint_writer, oracle_reader).unwrap_err();
            assert!(matches!(accelerated_result, PrecompileHalt::OutOfGas));
        })
        .await;
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_kzg_point_eval_bad_input_size() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let accelerated_result =
                fpvm_kzg_point_eval(&[], u64::MAX, hint_writer, oracle_reader).unwrap_err();
            assert!(matches!(accelerated_result, PrecompileHalt::BlobInvalidInputLength));
        })
        .await;
    }
}

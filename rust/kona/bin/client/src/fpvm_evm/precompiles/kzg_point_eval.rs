//! Contains the accelerated version of the KZG point evaluation precompile.

use crate::fpvm_evm::precompiles::utils::precompile_run;
use alloc::string::ToString;
use alloy_primitives::Address;
use kona_preimage::{HintWriterClient, PreimageOracleClient};
use revm::precompile::{PrecompileError, PrecompileOutput, PrecompileResult};

/// Address of the KZG point evaluation precompile.
pub(crate) const KZG_POINT_EVAL_ADDR: Address = revm::precompile::u64_to_address(0x0A);

/// Runs the FPVM-accelerated `kzgPointEval` precompile call.
pub(crate) fn fpvm_kzg_point_eval<H, O>(
    input: &[u8],
    gas_limit: u64,
    hint_writer: &H,
    oracle_reader: &O,
) -> PrecompileResult
where
    H: HintWriterClient + Send + Sync,
    O: PreimageOracleClient + Send + Sync,
{
    const GAS_COST: u64 = 50_000;

    if gas_limit < GAS_COST {
        return Err(PrecompileError::OutOfGas);
    }

    if input.len() != 192 {
        return Err(PrecompileError::BlobInvalidInputLength);
    }

    let result_data = kona_proof::block_on(precompile_run! {
        hint_writer,
        oracle_reader,
        &[KZG_POINT_EVAL_ADDR.as_slice(), &GAS_COST.to_be_bytes(), input]
    })
    .map_err(|e| PrecompileError::Other(e.to_string().into()))?;

    Ok(PrecompileOutput::new(GAS_COST, result_data.into()))
}

#[cfg(test)]
mod test {
    use super::*;
    use crate::fpvm_evm::precompiles::test_utils::{
        execute_native_precompile, test_accelerated_precompile,
    };
    use alloy_eips::eip4844::VERSIONED_HASH_VERSION_KZG;
    use alloy_primitives::hex;
    use sha2::{Digest, Sha256};

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_kzg_point_eval() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let commitment = hex!("8f59a8d2a1a625a17f3fea0fe5eb8c896db3764f3185481bc22f91b4aaffcca25f26936857bc3a7c2539ea8ec3a952b7").to_vec();
            let mut versioned_hash = Sha256::digest(&commitment).to_vec();
            versioned_hash[0] = VERSIONED_HASH_VERSION_KZG;
            let z = hex!("73eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000000").to_vec();
            let y = hex!("1522a4a7f34e1ea350ae07c29c96c7e79655aa926122e95fe69fcbd932ca49e9").to_vec();
            let proof = hex!("a62ad71d14c5719385c0686f1871430475bf3a00f0aa3f7b8dd99a9abc2160744faf0070725e00b60ad9a026a15b1a8c").to_vec();

            let input = [versioned_hash, z, y, commitment, proof].concat();

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
    async fn test_accelerated_kzg_point_eval_out_of_gas() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let accelerated_result =
                fpvm_kzg_point_eval(&[], 0, hint_writer, oracle_reader).unwrap_err();
            assert!(matches!(accelerated_result, PrecompileError::OutOfGas));
        })
        .await;
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_accelerated_kzg_point_eval_bad_input_size() {
        test_accelerated_precompile(|hint_writer, oracle_reader| {
            let accelerated_result =
                fpvm_kzg_point_eval(&[], u64::MAX, hint_writer, oracle_reader).unwrap_err();
            assert!(matches!(accelerated_result, PrecompileError::BlobInvalidInputLength));
        })
        .await;
    }
}

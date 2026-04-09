//! Verification of blocks w.r.t. Optimism hardforks.

pub mod canyon;
pub mod isthmus;

// Re-export the decode_holocene_base_fee function for compatibility
use reth_execution_types::BlockExecutionResult;
pub use reth_optimism_chainspec::decode_holocene_base_fee;

use crate::proof::calculate_receipt_root_optimism;
use alloc::vec::Vec;
use alloy_consensus::{BlockHeader, EMPTY_OMMER_ROOT_HASH, TxReceipt};
use alloy_eips::Encodable2718;
use alloy_primitives::{B256, Bloom, Bytes};
use alloy_trie::EMPTY_ROOT_HASH;
use reth_consensus::ConsensusError;
use reth_optimism_forks::OpHardforks;
use reth_optimism_primitives::DepositReceipt;
use reth_primitives_traits::{BlockBody, GotExpected, receipt::gas_spent_by_transactions};

/// Ensures the block response data matches the header.
///
/// This ensures the body response items match the header's hashes:
///   - ommer hash
///   - transaction root
///   - withdrawals root: the body's withdrawals root must only match the header's before isthmus
pub fn validate_body_against_header_op<B, H>(
    chain_spec: impl OpHardforks,
    body: &B,
    header: &H,
) -> Result<(), ConsensusError>
where
    B: BlockBody,
    H: reth_primitives_traits::BlockHeader,
{
    let ommers_hash = body.calculate_ommers_root();
    if Some(header.ommers_hash()) != ommers_hash {
        return Err(ConsensusError::BodyOmmersHashDiff(
            GotExpected {
                got: ommers_hash.unwrap_or(EMPTY_OMMER_ROOT_HASH),
                expected: header.ommers_hash(),
            }
            .into(),
        ));
    }

    let tx_root = body.calculate_tx_root();
    if header.transactions_root() != tx_root {
        return Err(ConsensusError::BodyTransactionRootDiff(
            GotExpected { got: tx_root, expected: header.transactions_root() }.into(),
        ));
    }

    match (header.withdrawals_root(), body.calculate_withdrawals_root()) {
        (Some(header_withdrawals_root), Some(withdrawals_root)) => {
            // after isthmus, the withdrawals root field is repurposed and no longer mirrors the
            // withdrawals root computed from the body
            if chain_spec.is_isthmus_active_at_timestamp(header.timestamp()) {
                // After isthmus we only ensure that the body has empty withdrawals
                if withdrawals_root != EMPTY_ROOT_HASH {
                    return Err(ConsensusError::BodyWithdrawalsRootDiff(
                        GotExpected { got: withdrawals_root, expected: EMPTY_ROOT_HASH }.into(),
                    ));
                }
            } else {
                // before isthmus we ensure that the header root matches the body
                if withdrawals_root != header_withdrawals_root {
                    return Err(ConsensusError::BodyWithdrawalsRootDiff(
                        GotExpected { got: withdrawals_root, expected: header_withdrawals_root }
                            .into(),
                    ));
                }
            }
        }
        (None, None) => {
            // this is ok because we assume the fork is not active in this case
        }
        _ => return Err(ConsensusError::WithdrawalsRootUnexpected),
    }

    Ok(())
}

/// Validate a block with regard to execution results:
///
/// - Compares the receipts root in the block header to the block body
/// - Compares the gas used in the block header to the actual gas usage after execution
///
/// `receipt_root_bloom` is accepted for API compatibility with upstream consensus hooks but is
/// **ignored** for receipt checks: parallel execution may supply a root from the generic Ethereum
/// receipts trie, which incorrectly includes Regolith `deposit_nonce` in deposit leaves. OP Stack
/// headers use op-geth `Receipts.EncodeIndex` rules; we always recompute via
/// [`verify_receipts_optimism`] from `result.receipts`.
pub fn validate_block_post_execution<R: DepositReceipt>(
    header: impl BlockHeader,
    chain_spec: impl OpHardforks,
    result: &BlockExecutionResult<R>,
    receipt_root_bloom: Option<(B256, Bloom)>,
) -> Result<(), ConsensusError> {
    // Validate that the blob gas used is present and correctly computed if Jovian is active.
    if chain_spec.is_jovian_active_at_timestamp(header.timestamp()) {
        let computed_blob_gas_used = result.blob_gas_used;
        let header_blob_gas_used =
            header.blob_gas_used().ok_or(ConsensusError::BlobGasUsedMissing)?;

        if computed_blob_gas_used != header_blob_gas_used {
            return Err(ConsensusError::BlobGasUsedDiff(GotExpected {
                got: computed_blob_gas_used,
                expected: header_blob_gas_used,
            }));
        }
    }

    let receipts = &result.receipts;

    // Before Byzantium, receipts contained state root that would mean that expensive
    // operation as hashing that is required for state root got calculated in every
    // transaction This was replaced with is_success flag.
    // See more about EIP here: https://eips.ethereum.org/EIPS/eip-658
    if chain_spec.is_byzantium_active_at_block(header.number()) {
        match receipt_root_bloom.as_ref() {
            Some((precomputed_root, _precomputed_bloom)) => {
                // Parallel execution (e.g. StateRootTask) may pass this; we still recompute below
                // using OP trie rules (see module docs).
                tracing::info!(
                    block_number = header.number(),
                    precomputed_receipts_root = %precomputed_root,
                    receipt_count = receipts.len(),
                    "blablabla: validate_block_post_execution received receipt_root_bloom, recomputing OP receipts root from execution receipts"
                );
            }
            None => {
                tracing::info!(
                    block_number = header.number(),
                    header_receipts_root = %header.receipts_root(),
                    receipt_count = receipts.len(),
                    "blablabla: validate_block_post_execution verifying optimism receipts root (no precomputed receipt_root_bloom)"
                );
            }
        }
        if let Err(error) =
            verify_receipts_optimism(header.receipts_root(), header.logs_bloom(), receipts)
        {
            let receipts = receipts
                .iter()
                .map(|r| Bytes::from(r.with_bloom_ref().encoded_2718()))
                .collect::<Vec<_>>();
            tracing::debug!(%error, ?receipts, "receipts verification failed");
            return Err(error);
        }
    }

    // Check if gas used matches the value set in header.
    let cumulative_gas_used =
        receipts.last().map(|receipt| receipt.cumulative_gas_used()).unwrap_or(0);
    if header.gas_used() != cumulative_gas_used {
        return Err(ConsensusError::BlockGasUsed {
            gas: GotExpected { got: cumulative_gas_used, expected: header.gas_used() },
            gas_spent_by_tx: gas_spent_by_transactions(receipts),
        });
    }

    Ok(())
}

/// Verify the calculated receipts root against the expected receipts root.
fn verify_receipts_optimism<R: DepositReceipt>(
    expected_receipts_root: B256,
    expected_logs_bloom: Bloom,
    receipts: &[R],
) -> Result<(), ConsensusError> {
    // Calculate receipts root.
    let receipts_with_bloom = receipts.iter().map(TxReceipt::with_bloom_ref).collect::<Vec<_>>();
    tracing::info!(
        receipt_count = receipts_with_bloom.len(),
        "blablabla232211: verify_receipts_optimism computing root via calculate_receipt_root_optimism"
    );
    let receipts_root = calculate_receipt_root_optimism(&receipts_with_bloom);

    // Calculate header logs bloom.
    let logs_bloom = receipts_with_bloom.iter().fold(Bloom::ZERO, |bloom, r| bloom | r.bloom_ref());

    compare_receipts_root_and_logs_bloom(
        receipts_root,
        logs_bloom,
        expected_receipts_root,
        expected_logs_bloom,
    )?;

    Ok(())
}

/// Compare the calculated receipts root with the expected receipts root, also compare
/// the calculated logs bloom with the expected logs bloom.
fn compare_receipts_root_and_logs_bloom(
    calculated_receipts_root: B256,
    calculated_logs_bloom: Bloom,
    expected_receipts_root: B256,
    expected_logs_bloom: Bloom,
) -> Result<(), ConsensusError> {
    if calculated_receipts_root != expected_receipts_root {
        return Err(ConsensusError::BodyReceiptRootDiff(
            GotExpected { got: calculated_receipts_root, expected: expected_receipts_root }.into(),
        ));
    }

    if calculated_logs_bloom != expected_logs_bloom {
        return Err(ConsensusError::BodyBloomLogDiff(
            GotExpected { got: calculated_logs_bloom, expected: expected_logs_bloom }.into(),
        ));
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::{Header, Receipt};
    use alloy_eips::eip7685::Requests;
    use alloy_primitives::{Bloom, Bytes, U256, b256, hex};
    use op_alloy_consensus::{OpDepositReceipt, OpTxEnvelope};
    use reth_chainspec::{BaseFeeParams, ChainSpec, EthChainSpec, ForkCondition, Hardfork};
    use reth_optimism_chainspec::{BASE_SEPOLIA, OpChainSpec};
    use reth_optimism_forks::{BASE_SEPOLIA_HARDFORKS, OpHardfork};
    use reth_optimism_primitives::OpReceipt;
    use std::sync::Arc;

    const HOLOCENE_TIMESTAMP: u64 = 1700000000;
    const ISTHMUS_TIMESTAMP: u64 = 1750000000;
    const JOVIAN_TIMESTAMP: u64 = 1800000000;
    const BLOCK_TIME_SECONDS: u64 = 2;

    fn holocene_chainspec() -> Arc<OpChainSpec> {
        let mut hardforks = BASE_SEPOLIA_HARDFORKS.clone();
        hardforks
            .insert(OpHardfork::Holocene.boxed(), ForkCondition::Timestamp(HOLOCENE_TIMESTAMP));
        Arc::new(OpChainSpec {
            inner: ChainSpec {
                chain: BASE_SEPOLIA.inner.chain,
                genesis: BASE_SEPOLIA.inner.genesis.clone(),
                genesis_header: BASE_SEPOLIA.inner.genesis_header.clone(),
                paris_block_and_final_difficulty: Some((0, U256::from(0))),
                hardforks,
                base_fee_params: BASE_SEPOLIA.inner.base_fee_params.clone(),
                prune_delete_limit: 10000,
                ..Default::default()
            },
        })
    }

    fn isthmus_chainspec() -> OpChainSpec {
        let mut chainspec = BASE_SEPOLIA.as_ref().clone();
        chainspec
            .inner
            .hardforks
            .insert(OpHardfork::Isthmus.boxed(), ForkCondition::Timestamp(ISTHMUS_TIMESTAMP));
        chainspec
    }

    fn jovian_chainspec() -> OpChainSpec {
        let mut chainspec = BASE_SEPOLIA.as_ref().clone();
        chainspec
            .inner
            .hardforks
            .insert(OpHardfork::Jovian.boxed(), ForkCondition::Timestamp(JOVIAN_TIMESTAMP));
        chainspec
    }

    #[test]
    fn test_get_base_fee_pre_holocene() {
        let op_chain_spec = BASE_SEPOLIA.clone();
        let parent = Header {
            base_fee_per_gas: Some(1),
            gas_used: 15763614,
            gas_limit: 144000000,
            ..Default::default()
        };
        let base_fee =
            reth_optimism_chainspec::OpChainSpec::next_block_base_fee(&op_chain_spec, &parent, 0);
        assert_eq!(
            base_fee.unwrap(),
            op_chain_spec.next_block_base_fee(&parent, 0).unwrap_or_default()
        );
    }

    #[test]
    fn test_get_base_fee_holocene_extra_data_not_set() {
        let op_chain_spec = holocene_chainspec();
        let parent = Header {
            base_fee_per_gas: Some(1),
            gas_used: 15763614,
            gas_limit: 144000000,
            timestamp: HOLOCENE_TIMESTAMP + 3,
            extra_data: Bytes::from_static(&[0, 0, 0, 0, 0, 0, 0, 0, 0]),
            ..Default::default()
        };
        let base_fee = reth_optimism_chainspec::OpChainSpec::next_block_base_fee(
            &op_chain_spec,
            &parent,
            HOLOCENE_TIMESTAMP + 5,
        );
        assert_eq!(
            base_fee.unwrap(),
            op_chain_spec.next_block_base_fee(&parent, 0).unwrap_or_default()
        );
    }

    #[test]
    fn test_get_base_fee_holocene_extra_data_set() {
        let parent = Header {
            base_fee_per_gas: Some(1),
            gas_used: 15763614,
            gas_limit: 144000000,
            extra_data: Bytes::from_static(&[0, 0, 0, 0, 8, 0, 0, 0, 8]),
            timestamp: HOLOCENE_TIMESTAMP + 3,
            ..Default::default()
        };

        let base_fee = reth_optimism_chainspec::OpChainSpec::next_block_base_fee(
            &holocene_chainspec(),
            &parent,
            HOLOCENE_TIMESTAMP + 5,
        );
        assert_eq!(
            base_fee.unwrap(),
            parent
                .next_block_base_fee(BaseFeeParams::new(0x00000008, 0x00000008))
                .unwrap_or_default()
        );
    }

    // <https://sepolia.basescan.org/block/19773628>
    #[test]
    fn test_get_base_fee_holocene_extra_data_set_base_sepolia() {
        let parent = Header {
            base_fee_per_gas: Some(507),
            gas_used: 4847634,
            gas_limit: 60000000,
            extra_data: hex!("00000000fa0000000a").into(),
            timestamp: 1735315544,
            ..Default::default()
        };

        let base_fee = reth_optimism_chainspec::OpChainSpec::next_block_base_fee(
            &*BASE_SEPOLIA,
            &parent,
            1735315546,
        )
        .unwrap();
        assert_eq!(base_fee, 507);
    }

    #[test]
    fn test_get_base_fee_holocene_extra_data_set_and_min_base_fee_set() {
        const MIN_BASE_FEE: u64 = 10;

        let mut extra_data = Vec::new();
        // eip1559 params
        extra_data.append(&mut hex!("00000000fa0000000a").to_vec());
        // min base fee
        extra_data.append(&mut MIN_BASE_FEE.to_be_bytes().to_vec());
        let extra_data = Bytes::from(extra_data);

        let parent = Header {
            base_fee_per_gas: Some(507),
            gas_used: 4847634,
            gas_limit: 60000000,
            extra_data,
            timestamp: 1735315544,
            ..Default::default()
        };

        let base_fee = reth_optimism_chainspec::OpChainSpec::next_block_base_fee(
            &*BASE_SEPOLIA,
            &parent,
            1735315546,
        );
        assert_eq!(base_fee, None);
    }

    /// The version byte for Jovian is 1.
    const JOVIAN_EXTRA_DATA_VERSION_BYTE: u8 = 1;

    #[test]
    fn test_get_base_fee_jovian_extra_data_and_min_base_fee_not_set() {
        let op_chain_spec = jovian_chainspec();

        let mut extra_data = Vec::new();
        extra_data.push(JOVIAN_EXTRA_DATA_VERSION_BYTE);
        // eip1559 params
        extra_data.append(&mut [0_u8; 8].to_vec());
        let extra_data = Bytes::from(extra_data);

        let parent = Header {
            base_fee_per_gas: Some(1),
            gas_used: 15763614,
            gas_limit: 144000000,
            timestamp: JOVIAN_TIMESTAMP,
            extra_data,
            ..Default::default()
        };
        let base_fee = reth_optimism_chainspec::OpChainSpec::next_block_base_fee(
            &op_chain_spec,
            &parent,
            JOVIAN_TIMESTAMP + BLOCK_TIME_SECONDS,
        );
        assert_eq!(base_fee, None);
    }

    /// After Jovian, the next block base fee cannot be less than the minimum base fee.
    #[test]
    fn test_get_base_fee_jovian_default_extra_data_and_min_base_fee() {
        const CURR_BASE_FEE: u64 = 1;
        const MIN_BASE_FEE: u64 = 10;

        let mut extra_data = Vec::new();
        extra_data.push(JOVIAN_EXTRA_DATA_VERSION_BYTE);
        // eip1559 params
        extra_data.append(&mut [0_u8; 8].to_vec());
        // min base fee
        extra_data.append(&mut MIN_BASE_FEE.to_be_bytes().to_vec());
        let extra_data = Bytes::from(extra_data);

        let op_chain_spec = jovian_chainspec();
        let parent = Header {
            base_fee_per_gas: Some(CURR_BASE_FEE),
            gas_used: 15763614,
            gas_limit: 144000000,
            timestamp: JOVIAN_TIMESTAMP,
            extra_data,
            ..Default::default()
        };
        let base_fee = reth_optimism_chainspec::OpChainSpec::next_block_base_fee(
            &op_chain_spec,
            &parent,
            JOVIAN_TIMESTAMP + BLOCK_TIME_SECONDS,
        );
        assert_eq!(base_fee, Some(MIN_BASE_FEE));
    }

    /// After Jovian, the next block base fee cannot be less than the minimum base fee.
    #[test]
    fn test_jovian_min_base_fee_cannot_decrease() {
        const MIN_BASE_FEE: u64 = 10;

        let mut extra_data = Vec::new();
        extra_data.push(JOVIAN_EXTRA_DATA_VERSION_BYTE);
        // eip1559 params
        extra_data.append(&mut [0_u8; 8].to_vec());
        // min base fee
        extra_data.append(&mut MIN_BASE_FEE.to_be_bytes().to_vec());
        let extra_data = Bytes::from(extra_data);

        let op_chain_spec = jovian_chainspec();

        // If we're currently at the minimum base fee, the next block base fee cannot decrease.
        let parent = Header {
            base_fee_per_gas: Some(MIN_BASE_FEE),
            gas_used: 10,
            gas_limit: 144000000,
            timestamp: JOVIAN_TIMESTAMP,
            extra_data: extra_data.clone(),
            ..Default::default()
        };
        let base_fee = reth_optimism_chainspec::OpChainSpec::next_block_base_fee(
            &op_chain_spec,
            &parent,
            JOVIAN_TIMESTAMP + BLOCK_TIME_SECONDS,
        );
        assert_eq!(base_fee, Some(MIN_BASE_FEE));

        // The next block can increase the base fee
        let parent = Header {
            base_fee_per_gas: Some(MIN_BASE_FEE),
            gas_used: 144000000,
            gas_limit: 144000000,
            timestamp: JOVIAN_TIMESTAMP,
            extra_data,
            ..Default::default()
        };
        let base_fee = reth_optimism_chainspec::OpChainSpec::next_block_base_fee(
            &op_chain_spec,
            &parent,
            JOVIAN_TIMESTAMP + 2 * BLOCK_TIME_SECONDS,
        );
        assert_eq!(base_fee, Some(MIN_BASE_FEE + 1));
    }

    #[test]
    fn test_jovian_base_fee_can_decrease_if_above_min_base_fee() {
        const MIN_BASE_FEE: u64 = 10;

        let mut extra_data = Vec::new();
        extra_data.push(JOVIAN_EXTRA_DATA_VERSION_BYTE);
        // eip1559 params
        extra_data.append(&mut [0_u8; 8].to_vec());
        // min base fee
        extra_data.append(&mut MIN_BASE_FEE.to_be_bytes().to_vec());
        let extra_data = Bytes::from(extra_data);

        let op_chain_spec = jovian_chainspec();

        let parent = Header {
            base_fee_per_gas: Some(100 * MIN_BASE_FEE),
            gas_used: 10,
            gas_limit: 144000000,
            timestamp: JOVIAN_TIMESTAMP,
            extra_data,
            ..Default::default()
        };
        let base_fee = reth_optimism_chainspec::OpChainSpec::next_block_base_fee(
            &op_chain_spec,
            &parent,
            JOVIAN_TIMESTAMP + BLOCK_TIME_SECONDS,
        )
        .unwrap();
        assert_eq!(
            base_fee,
            op_chain_spec
                .inner
                .next_block_base_fee(&parent, JOVIAN_TIMESTAMP + BLOCK_TIME_SECONDS)
                .unwrap()
        );
    }

    #[test]
    fn body_against_header_isthmus() {
        let chainspec = isthmus_chainspec();
        let header = Header {
            base_fee_per_gas: Some(507),
            gas_used: 4847634,
            gas_limit: 60000000,
            extra_data: hex!("00000000fa0000000a").into(),
            timestamp: 1800000000,
            withdrawals_root: Some(b256!(
                "0x611e1d75cbb77fa782d79485a8384e853bc92e56883c313a51e3f9feef9a9a71"
            )),
            ..Default::default()
        };
        let mut body = alloy_consensus::BlockBody::<OpTxEnvelope> {
            transactions: vec![],
            ommers: vec![],
            withdrawals: Some(Default::default()),
        };
        validate_body_against_header_op(&chainspec, &body, &header).unwrap();

        body.withdrawals.take();
        validate_body_against_header_op(&chainspec, &body, &header).unwrap_err();
    }

    #[test]
    fn test_jovian_blob_gas_used_validation() {
        const BLOB_GAS_USED: u64 = 1000;
        const GAS_USED: u64 = 5000;

        let chainspec = jovian_chainspec();
        let header = Header {
            timestamp: JOVIAN_TIMESTAMP,
            blob_gas_used: Some(BLOB_GAS_USED),
            ..Default::default()
        };

        let result = BlockExecutionResult::<OpReceipt> {
            blob_gas_used: BLOB_GAS_USED,
            receipts: vec![],
            requests: Requests::default(),
            gas_used: GAS_USED,
        };
        validate_block_post_execution(&header, &chainspec, &result, None).unwrap();
    }

    #[test]
    fn test_jovian_blob_gas_used_validation_mismatched() {
        const BLOB_GAS_USED: u64 = 1000;
        const GAS_USED: u64 = 5000;

        let chainspec = jovian_chainspec();
        let header = Header {
            timestamp: JOVIAN_TIMESTAMP,
            blob_gas_used: Some(BLOB_GAS_USED + 1),
            ..Default::default()
        };

        let result = BlockExecutionResult::<OpReceipt> {
            blob_gas_used: BLOB_GAS_USED,
            receipts: vec![],
            requests: Requests::default(),
            gas_used: GAS_USED,
        };
        assert!(matches!(
            validate_block_post_execution(&header, &chainspec, &result, None).unwrap_err(),
            ConsensusError::BlobGasUsedDiff(diff)
                if diff.got == BLOB_GAS_USED && diff.expected == BLOB_GAS_USED + 1
        ));
    }

    /// End-to-end: `validate_block_post_execution` → `verify_receipts_optimism` →
    /// `calculate_receipt_root_optimism` for Manta Sepolia block 863731 (Regolith deposit, pre-Canyon).
    #[test]
    fn validate_post_execution_matches_manta_sepolia_863731_header_receipts_root() {
        let chainspec = BASE_SEPOLIA.clone();
        let header = Header {
            number: 863_731,
            timestamp: 1,
            gas_used: 0xf9f5,
            receipts_root: b256!(
                "0x3c715dd96d2597ccd46fde046da5e4b13e0a5b7d0a2ff60c3ee6c92fee9600ea"
            ),
            logs_bloom: Bloom::ZERO,
            ..Default::default()
        };
        let receipts = vec![OpReceipt::Deposit(OpDepositReceipt {
            inner: Receipt { status: true.into(), cumulative_gas_used: 0xf9f5, logs: vec![] },
            deposit_nonce: Some(0xd2df2),
            deposit_receipt_version: None,
        })];
        let result = BlockExecutionResult {
            blob_gas_used: 0,
            receipts,
            requests: Requests::default(),
            gas_used: 0xf9f5,
        };
        validate_block_post_execution(&header, chainspec.as_ref(), &result, None).unwrap();
    }
}

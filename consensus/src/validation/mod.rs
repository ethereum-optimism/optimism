//! Verification of blocks w.r.t. Optimism hardforks.

pub mod canyon;
pub mod isthmus;
pub mod shanghai;

use crate::proof::calculate_receipt_root_optimism;
use alloc::vec::Vec;
use alloy_consensus::{BlockHeader, TxReceipt};
use alloy_primitives::{Bloom, B256};
use op_alloy_consensus::{decode_holocene_extra_data, EIP1559ParamError};
use reth_chainspec::{BaseFeeParams, EthChainSpec};
use reth_consensus::ConsensusError;
use reth_optimism_forks::OpHardforks;
use reth_optimism_primitives::DepositReceipt;
use reth_primitives_traits::{receipt::gas_spent_by_transactions, GotExpected};

/// Validate a block with regard to execution results:
///
/// - Compares the receipts root in the block header to the block body
/// - Compares the gas used in the block header to the actual gas usage after execution
pub fn validate_block_post_execution<R: DepositReceipt>(
    header: impl BlockHeader,
    chain_spec: impl OpHardforks,
    receipts: &[R],
) -> Result<(), ConsensusError> {
    // Before Byzantium, receipts contained state root that would mean that expensive
    // operation as hashing that is required for state root got calculated in every
    // transaction This was replaced with is_success flag.
    // See more about EIP here: https://eips.ethereum.org/EIPS/eip-658
    if chain_spec.is_byzantium_active_at_block(header.number()) {
        if let Err(error) = verify_receipts_optimism(
            header.receipts_root(),
            header.logs_bloom(),
            receipts,
            chain_spec,
            header.timestamp(),
        ) {
            tracing::debug!(%error, ?receipts, "receipts verification failed");
            return Err(error)
        }
    }

    // Check if gas used matches the value set in header.
    let cumulative_gas_used =
        receipts.last().map(|receipt| receipt.cumulative_gas_used()).unwrap_or(0);
    if header.gas_used() != cumulative_gas_used {
        return Err(ConsensusError::BlockGasUsed {
            gas: GotExpected { got: cumulative_gas_used, expected: header.gas_used() },
            gas_spent_by_tx: gas_spent_by_transactions(receipts),
        })
    }

    Ok(())
}

/// Verify the calculated receipts root against the expected receipts root.
fn verify_receipts_optimism<R: DepositReceipt>(
    expected_receipts_root: B256,
    expected_logs_bloom: Bloom,
    receipts: &[R],
    chain_spec: impl OpHardforks,
    timestamp: u64,
) -> Result<(), ConsensusError> {
    // Calculate receipts root.
    let receipts_with_bloom = receipts.iter().cloned().map(Into::into).collect::<Vec<_>>();
    let receipts_root =
        calculate_receipt_root_optimism(&receipts_with_bloom, chain_spec, timestamp);

    // Calculate header logs bloom.
    let logs_bloom = receipts_with_bloom.iter().fold(Bloom::ZERO, |bloom, r| bloom | r.bloom());

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
        ))
    }

    if calculated_logs_bloom != expected_logs_bloom {
        return Err(ConsensusError::BodyBloomLogDiff(
            GotExpected { got: calculated_logs_bloom, expected: expected_logs_bloom }.into(),
        ))
    }

    Ok(())
}

/// Extracts the Holocene 1599 parameters from the encoded extra data from the parent header.
///
/// Caution: Caller must ensure that holocene is active in the parent header.
///
/// See also [Base fee computation](https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/holocene/exec-engine.md#base-fee-computation)
pub fn decode_holocene_base_fee(
    chain_spec: impl EthChainSpec + OpHardforks,
    parent: impl BlockHeader,
    timestamp: u64,
) -> Result<u64, EIP1559ParamError> {
    let (elasticity, denominator) = decode_holocene_extra_data(parent.extra_data())?;
    let base_fee_params = if elasticity == 0 && denominator == 0 {
        chain_spec.base_fee_params_at_timestamp(timestamp)
    } else {
        BaseFeeParams::new(denominator as u128, elasticity as u128)
    };

    Ok(parent.next_block_base_fee(base_fee_params).unwrap_or_default())
}

/// Read from parent to determine the base fee for the next block
///
/// See also [Base fee computation](https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/holocene/exec-engine.md#base-fee-computation)
pub fn next_block_base_fee(
    chain_spec: impl EthChainSpec + OpHardforks,
    parent: impl BlockHeader,
    timestamp: u64,
) -> Result<u64, EIP1559ParamError> {
    // If we are in the Holocene, we need to use the base fee params
    // from the parent block's extra data.
    // Else, use the base fee params (default values) from chainspec
    if chain_spec.is_holocene_active_at_timestamp(parent.timestamp()) {
        Ok(decode_holocene_base_fee(chain_spec, parent, timestamp)?)
    } else {
        Ok(parent
            .next_block_base_fee(chain_spec.base_fee_params_at_timestamp(timestamp))
            .unwrap_or_default())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::Header;
    use alloy_primitives::{hex, Bytes, U256};
    use reth_chainspec::{ChainSpec, ForkCondition, Hardfork};
    use reth_optimism_chainspec::{OpChainSpec, BASE_SEPOLIA};
    use reth_optimism_forks::{OpHardfork, BASE_SEPOLIA_HARDFORKS};
    use std::sync::Arc;

    fn holocene_chainspec() -> Arc<OpChainSpec> {
        let mut hardforks = BASE_SEPOLIA_HARDFORKS.clone();
        hardforks.insert(OpHardfork::Holocene.boxed(), ForkCondition::Timestamp(1800000000));
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

    #[test]
    fn test_get_base_fee_pre_holocene() {
        let op_chain_spec = BASE_SEPOLIA.clone();
        let parent = Header {
            base_fee_per_gas: Some(1),
            gas_used: 15763614,
            gas_limit: 144000000,
            ..Default::default()
        };
        let base_fee = next_block_base_fee(&op_chain_spec, &parent, 0);
        assert_eq!(
            base_fee.unwrap(),
            parent
                .next_block_base_fee(op_chain_spec.base_fee_params_at_timestamp(0))
                .unwrap_or_default()
        );
    }

    #[test]
    fn test_get_base_fee_holocene_extra_data_not_set() {
        let op_chain_spec = holocene_chainspec();
        let parent = Header {
            base_fee_per_gas: Some(1),
            gas_used: 15763614,
            gas_limit: 144000000,
            timestamp: 1800000003,
            extra_data: Bytes::from_static(&[0, 0, 0, 0, 0, 0, 0, 0, 0]),
            ..Default::default()
        };
        let base_fee = next_block_base_fee(&op_chain_spec, &parent, 1800000005);
        assert_eq!(
            base_fee.unwrap(),
            parent
                .next_block_base_fee(op_chain_spec.base_fee_params_at_timestamp(0))
                .unwrap_or_default()
        );
    }

    #[test]
    fn test_get_base_fee_holocene_extra_data_set() {
        let parent = Header {
            base_fee_per_gas: Some(1),
            gas_used: 15763614,
            gas_limit: 144000000,
            extra_data: Bytes::from_static(&[0, 0, 0, 0, 8, 0, 0, 0, 8]),
            timestamp: 1800000003,
            ..Default::default()
        };

        let base_fee = next_block_base_fee(holocene_chainspec(), &parent, 1800000005);
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

        let base_fee = next_block_base_fee(&*BASE_SEPOLIA, &parent, 1735315546).unwrap();
        assert_eq!(base_fee, 507);
    }
}

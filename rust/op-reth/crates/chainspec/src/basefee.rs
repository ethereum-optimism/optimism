//! Base fee related utilities for Optimism chains.

use core::cmp::max;

use alloy_consensus::BlockHeader;
use alloy_eips::calc_next_block_base_fee;
use op_alloy_consensus::{EIP1559ParamError, decode_holocene_extra_data, decode_jovian_extra_data};
use reth_chainspec::{BaseFeeParams, EthChainSpec};
use reth_optimism_forks::OpHardforks;

/// Extracts the Holocene 1599 parameters from the encoded extra data from the parent header.
///
/// Caution: Caller must ensure that holocene is active in the parent header.
///
/// See also [Base fee computation](https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/holocene/exec-engine.md#base-fee-computation)
pub fn decode_holocene_base_fee<H>(
    chain_spec: impl EthChainSpec + OpHardforks,
    parent: &H,
    timestamp: u64,
) -> Result<u64, EIP1559ParamError>
where
    H: BlockHeader,
{
    let (elasticity, denominator) = decode_holocene_extra_data(parent.extra_data())?;

    let base_fee_params = if elasticity == 0 && denominator == 0 {
        chain_spec.base_fee_params_at_timestamp(timestamp)
    } else {
        BaseFeeParams::new(denominator as u128, elasticity as u128)
    };

    Ok(parent.next_block_base_fee(base_fee_params).unwrap_or_default())
}

/// Extracts the Jovian 1599 parameters from the encoded extra data from the parent header.
/// Additionally to [`decode_holocene_base_fee`], checks if the next block base fee is less than the
/// minimum base fee, then the minimum base fee is returned.
///
/// Caution: Caller must ensure that jovian is active in the parent header.
///
/// See also [Base fee computation](https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/jovian/exec-engine.md#base-fee-computation)
/// and [Minimum base fee in block header](https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/jovian/exec-engine.md#minimum-base-fee-in-block-header)
pub fn compute_jovian_base_fee<H>(
    chain_spec: impl EthChainSpec + OpHardforks,
    parent: &H,
    timestamp: u64,
) -> Result<u64, EIP1559ParamError>
where
    H: BlockHeader,
{
    let (elasticity, denominator, min_base_fee) = decode_jovian_extra_data(parent.extra_data())?;

    let base_fee_params = if elasticity == 0 && denominator == 0 {
        chain_spec.base_fee_params_at_timestamp(timestamp)
    } else {
        BaseFeeParams::new(denominator as u128, elasticity as u128)
    };

    // Starting from Jovian, we use the maximum of the gas used and the blob gas used to calculate
    // the next base fee.
    let gas_used = max(parent.gas_used(), parent.blob_gas_used().unwrap_or_default());

    let next_base_fee = calc_next_block_base_fee(
        gas_used,
        parent.gas_limit(),
        parent.base_fee_per_gas().unwrap_or_default(),
        base_fee_params,
    );

    if next_base_fee < min_base_fee {
        return Ok(min_base_fee);
    }

    Ok(next_base_fee)
}

#[cfg(test)]
mod tests {
    use alloc::sync::Arc;

    use op_alloy_consensus::encode_jovian_extra_data;
    use reth_chainspec::{ChainSpec, ForkCondition, Hardfork};
    use reth_optimism_forks::OpHardfork;

    use crate::{BASE_SEPOLIA, OpChainSpec};

    use super::*;

    const JOVIAN_TIMESTAMP: u64 = 1900000000;

    fn get_chainspec() -> Arc<OpChainSpec> {
        let mut base_sepolia_spec = BASE_SEPOLIA.inner.clone();
        base_sepolia_spec
            .hardforks
            .insert(OpHardfork::Jovian.boxed(), ForkCondition::Timestamp(JOVIAN_TIMESTAMP));
        Arc::new(OpChainSpec {
            inner: ChainSpec {
                chain: base_sepolia_spec.chain,
                genesis: base_sepolia_spec.genesis,
                genesis_header: base_sepolia_spec.genesis_header,
                ..Default::default()
            },
        })
    }

    #[test]
    fn test_next_base_fee_jovian_blob_gas_used_greater_than_gas_used() {
        let chain_spec = get_chainspec();
        let mut parent = chain_spec.genesis_header().clone();
        let timestamp = JOVIAN_TIMESTAMP;

        const GAS_LIMIT: u64 = 10_000_000_000;
        const BLOB_GAS_USED: u64 = 5_000_000_000;
        const GAS_USED: u64 = 1_000_000_000;
        const MIN_BASE_FEE: u64 = 100_000_000;

        parent.extra_data =
            encode_jovian_extra_data([0; 8].into(), BaseFeeParams::base_sepolia(), MIN_BASE_FEE)
                .unwrap();
        parent.blob_gas_used = Some(BLOB_GAS_USED);
        parent.gas_used = GAS_USED;
        parent.gas_limit = GAS_LIMIT;

        let expected_base_fee = calc_next_block_base_fee(
            BLOB_GAS_USED,
            parent.gas_limit(),
            parent.base_fee_per_gas().unwrap_or_default(),
            BaseFeeParams::base_sepolia(),
        );
        assert_eq!(
            expected_base_fee,
            compute_jovian_base_fee(chain_spec, &parent, timestamp).unwrap()
        );
        assert_ne!(
            expected_base_fee,
            calc_next_block_base_fee(
                GAS_USED,
                parent.gas_limit(),
                parent.base_fee_per_gas().unwrap_or_default(),
                BaseFeeParams::base_sepolia(),
            )
        )
    }

    #[test]
    fn test_next_base_fee_jovian_blob_gas_used_less_than_gas_used() {
        let chain_spec = get_chainspec();
        let mut parent = chain_spec.genesis_header().clone();
        let timestamp = JOVIAN_TIMESTAMP;

        const GAS_LIMIT: u64 = 10_000_000_000;
        const BLOB_GAS_USED: u64 = 100_000_000;
        const GAS_USED: u64 = 1_000_000_000;
        const MIN_BASE_FEE: u64 = 100_000_000;

        parent.extra_data =
            encode_jovian_extra_data([0; 8].into(), BaseFeeParams::base_sepolia(), MIN_BASE_FEE)
                .unwrap();
        parent.blob_gas_used = Some(BLOB_GAS_USED);
        parent.gas_used = GAS_USED;
        parent.gas_limit = GAS_LIMIT;

        let expected_base_fee = calc_next_block_base_fee(
            GAS_USED,
            parent.gas_limit(),
            parent.base_fee_per_gas().unwrap_or_default(),
            BaseFeeParams::base_sepolia(),
        );
        assert_eq!(
            expected_base_fee,
            compute_jovian_base_fee(chain_spec, &parent, timestamp).unwrap()
        );
    }

    #[test]
    fn test_next_base_fee_jovian_min_base_fee() {
        let chain_spec = get_chainspec();
        let mut parent = chain_spec.genesis_header().clone();
        let timestamp = JOVIAN_TIMESTAMP;

        const GAS_LIMIT: u64 = 10_000_000_000;
        const BLOB_GAS_USED: u64 = 100_000_000;
        const GAS_USED: u64 = 1_000_000_000;
        const MIN_BASE_FEE: u64 = 5_000_000_000;

        parent.extra_data =
            encode_jovian_extra_data([0; 8].into(), BaseFeeParams::base_sepolia(), MIN_BASE_FEE)
                .unwrap();
        parent.blob_gas_used = Some(BLOB_GAS_USED);
        parent.gas_used = GAS_USED;
        parent.gas_limit = GAS_LIMIT;

        let expected_base_fee = MIN_BASE_FEE;
        assert_eq!(
            expected_base_fee,
            compute_jovian_base_fee(chain_spec, &parent, timestamp).unwrap()
        );
    }
}

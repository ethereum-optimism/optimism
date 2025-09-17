//! Base fee related utilities for Optimism chains.

use alloy_consensus::BlockHeader;
use op_alloy_consensus::{decode_holocene_extra_data, decode_jovian_extra_data, EIP1559ParamError};
use reth_chainspec::{BaseFeeParams, EthChainSpec};
use reth_optimism_forks::OpHardforks;

fn next_base_fee_params<H: BlockHeader>(
    chain_spec: impl EthChainSpec + OpHardforks,
    parent: &H,
    timestamp: u64,
    denominator: u32,
    elasticity: u32,
) -> u64 {
    let base_fee_params = if elasticity == 0 && denominator == 0 {
        chain_spec.base_fee_params_at_timestamp(timestamp)
    } else {
        BaseFeeParams::new(denominator as u128, elasticity as u128)
    };

    parent.next_block_base_fee(base_fee_params).unwrap_or_default()
}

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

    Ok(next_base_fee_params(chain_spec, parent, timestamp, denominator, elasticity))
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

    let next_base_fee =
        next_base_fee_params(chain_spec, parent, timestamp, denominator, elasticity);

    if next_base_fee < min_base_fee {
        return Ok(min_base_fee);
    }

    Ok(next_base_fee)
}

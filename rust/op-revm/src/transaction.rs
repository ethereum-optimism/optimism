//! Contains the `[OpTransaction]` type and its implementation.
pub mod abstraction;
pub mod deposit;
pub mod error;

pub use abstraction::{OpTransaction, OpTxTr};
pub use error::OpTransactionError;

use crate::fast_lz::flz_compress_len;
use alloy_eips::Encodable2718;
use op_alloy_consensus::POST_EXEC_TX_TYPE_ID;

/// <https://github.com/ethereum-optimism/op-geth/blob/647c346e2bef36219cc7b47d76b1cb87e7ca29e4/core/types/rollup_cost.go#L79>
const L1_COST_FASTLZ_COEF: u64 = 836_500;

/// <https://github.com/ethereum-optimism/op-geth/blob/647c346e2bef36219cc7b47d76b1cb87e7ca29e4/core/types/rollup_cost.go#L78>
/// Inverted to be used with `saturating_sub`.
const L1_COST_INTERCEPT: u64 = 42_585_600;

/// <https://github.com/ethereum-optimism/op-geth/blob/647c346e2bef36219cc7b47d76b1cb87e7ca29e4/core/types/rollup_cost.go#82>
const MIN_TX_SIZE_SCALED: u64 = 100 * 1_000_000;

/// Estimates the compressed size of a transaction.
pub fn estimate_tx_compressed_size(input: &[u8]) -> u64 {
    let fastlz_size = flz_compress_len(input) as u64;

    fastlz_size
        .saturating_mul(L1_COST_FASTLZ_COEF)
        .saturating_sub(L1_COST_INTERCEPT)
        .max(MIN_TX_SIZE_SCALED)
}

/// Computes a transaction's DA footprint from its EIP-2718 encoding, or zero for a post-exec
/// transaction.
///
/// <https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/jovian/exec-engine.md#da-footprint-block-limit>
pub fn tx_da_footprint<T: Encodable2718>(tx: &T, da_footprint_gas_scalar: u64) -> u64 {
    encoded_tx_da_footprint(&tx.encoded_2718(), da_footprint_gas_scalar)
}

/// Computes a transaction's DA footprint from its complete EIP-2718 encoding, or zero for a
/// post-exec transaction.
///
/// `encoded_2718` must include the EIP-2718 transaction type byte, not only the inner transaction
/// payload.
///
/// <https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/jovian/exec-engine.md#da-footprint-block-limit>
pub fn encoded_tx_da_footprint(encoded_2718: &[u8], da_footprint_gas_scalar: u64) -> u64 {
    if encoded_2718.first() == Some(&POST_EXEC_TX_TYPE_ID) {
        return 0;
    }

    estimate_tx_compressed_size(encoded_2718)
        .saturating_div(1_000_000)
        .saturating_mul(da_footprint_gas_scalar)
}

#[cfg(test)]
mod tests {
    use super::*;
    use op_alloy_consensus::{SDMGasEntry, build_post_exec_tx};

    #[test]
    fn post_exec_tx_da_footprint_is_zero() {
        const SCALAR: u64 = 100;
        let tx = build_post_exec_tx(1, 1, vec![SDMGasEntry { index: 1, gas_refund: 77 }]);
        let encoded = tx.encoded_2718();

        let encoded_size_footprint =
            estimate_tx_compressed_size(&encoded).saturating_div(1_000_000).saturating_mul(SCALAR);
        assert!(encoded_size_footprint > 0);
        assert_eq!(encoded_tx_da_footprint(&encoded, SCALAR), 0);
        assert_eq!(tx_da_footprint(&tx, SCALAR), 0);
    }
}

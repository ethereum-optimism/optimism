//! Canyon consensus rule checks.

use alloy_consensus::BlockHeader;
use alloy_trie::EMPTY_ROOT_HASH;
use reth_consensus::ConsensusError;
use reth_primitives_traits::{BlockBody, GotExpected};

use crate::OpConsensusError;

/// Verifies that withdrawals root in block header (Shanghai) is always [`EMPTY_ROOT_HASH`] in
/// Canyon.
#[inline]
pub fn ensure_empty_withdrawals_root<H: BlockHeader>(header: &H) -> Result<(), ConsensusError> {
    // Shanghai rule
    let header_withdrawals_root =
        &header.withdrawals_root().ok_or(ConsensusError::WithdrawalsRootMissing)?;

    //  Canyon rules
    if *header_withdrawals_root != EMPTY_ROOT_HASH {
        return Err(ConsensusError::BodyWithdrawalsRootDiff(
            GotExpected { got: *header_withdrawals_root, expected: EMPTY_ROOT_HASH }.into(),
        ));
    }

    Ok(())
}

/// Verifies that withdrawals in block body (Shanghai) is always empty in Canyon.
/// <https://specs.optimism.io/protocol/rollup-node-p2p.html#block-validation>
#[inline]
pub fn ensure_empty_shanghai_withdrawals<T: BlockBody>(body: &T) -> Result<(), OpConsensusError> {
    // Shanghai rule
    let withdrawals = body.withdrawals().ok_or(ConsensusError::BodyWithdrawalsMissing)?;

    //  Canyon rule
    if !withdrawals.as_ref().is_empty() {
        return Err(OpConsensusError::WithdrawalsNonEmpty)
    }

    Ok(())
}

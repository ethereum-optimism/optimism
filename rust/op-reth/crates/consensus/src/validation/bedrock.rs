//! Bedrock consensus rule checks.

use alloy_primitives::{Address, TxKind, address};
use op_alloy_consensus::{OpTransaction, predeploys::L1_BLOCK_ADDRESS};
use reth_primitives_traits::BlockBody;

use crate::OpConsensusError;

/// Sender of the L1 attributes deposit transaction.
pub(crate) const L1_INFO_DEPOSITOR_ADDRESS: Address =
    address!("0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001");

/// Verifies that the first transaction in the block body is the L1 attributes deposit.
/// <https://specs.optimism.io/protocol/deposits.html#l1-attributes-deposited-transaction>
#[inline]
pub fn ensure_l1_info_deposit_first<T>(body: &T) -> Result<(), OpConsensusError>
where
    T: BlockBody<Transaction: OpTransaction>,
{
    let opens_with_l1_info_deposit =
        body.transactions().first().and_then(OpTransaction::as_deposit).is_some_and(|deposit| {
            deposit.from == L1_INFO_DEPOSITOR_ADDRESS &&
                deposit.to == TxKind::Call(L1_BLOCK_ADDRESS)
        });

    if !opens_with_l1_info_deposit {
        return Err(OpConsensusError::L1InfoDepositNotFirst);
    }

    Ok(())
}

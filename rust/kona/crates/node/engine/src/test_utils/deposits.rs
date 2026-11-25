//! Deposit transaction fixtures.

use alloy_primitives::Sealed;
use kona_protocol::L1BlockInfoBedrock;
use op_alloy_consensus::{OpTxEnvelope, TxDeposit};

/// The L1-info deposit a block needs for `L2BlockInfo::from_block_and_genesis` to read its L1
/// origin.
pub fn l1_info_deposit() -> OpTxEnvelope {
    l1_info_deposit_for(&L1BlockInfoBedrock::default())
}

/// The L1-info deposit for `info`, which every non-genesis L2 block starts with.
pub fn l1_info_deposit_for(info: &L1BlockInfoBedrock) -> OpTxEnvelope {
    OpTxEnvelope::Deposit(Sealed::new(TxDeposit {
        input: info.encode_calldata(),
        ..Default::default()
    }))
}

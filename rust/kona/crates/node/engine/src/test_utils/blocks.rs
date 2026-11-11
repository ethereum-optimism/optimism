use alloy_consensus::{Header, transaction::Recovered};
use alloy_primitives::Sealed;
use alloy_rpc_types_eth::{Block, BlockTransactions};
use kona_protocol::L1BlockInfoBedrock;
use op_alloy_consensus::{OpTxEnvelope, TxDeposit};

/// The L1 info deposit for `info`, which every non-genesis L2 block starts with.
pub fn l1_info_deposit(info: &L1BlockInfoBedrock) -> OpTxEnvelope {
    OpTxEnvelope::Deposit(Sealed::new(TxDeposit {
        input: info.encode_calldata(),
        ..Default::default()
    }))
}

/// An L2 block with `header` and only the L1 info deposit for `info`, as an RPC returns it with
/// full transactions. The block hash is computed from `header`.
pub fn rpc_l2_block(
    header: Header,
    info: &L1BlockInfoBedrock,
) -> Block<op_alloy_rpc_types::Transaction> {
    let number = header.number;
    let mut block = Block::<op_alloy_rpc_types::Transaction>::default();
    block.header.hash = header.hash_slow();
    block.header.inner = header;
    block.transactions = BlockTransactions::Full(vec![op_alloy_rpc_types::Transaction {
        inner: alloy_rpc_types_eth::Transaction {
            block_hash: None,
            block_number: Some(number),
            block_timestamp: None,
            effective_gas_price: Some(1),
            inner: Recovered::new_unchecked(l1_info_deposit(info), Default::default()),
            transaction_index: Some(0),
        },
        deposit_nonce: None,
        deposit_receipt_version: None,
    }]);
    block
}

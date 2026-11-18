//! L2 blocks, and their RPC form as the execution layer serves them.

use super::{l1_info_deposit, l1_info_deposit_for};
use alloy_consensus::{Header, transaction::Recovered};
use alloy_primitives::{Address, B256, Bytes};
use alloy_rpc_types_eth::{BlockTransactions, Header as RpcHeader};
use kona_protocol::L1BlockInfoBedrock;
use op_alloy_consensus::OpBlock;

/// A full L2 block as the execution layer serves it over RPC.
pub type TestRpcBlock = alloy_rpc_types_eth::Block<op_alloy_rpc_types::Transaction>;

/// An L2 block at `number` on `parent_hash` whose only transaction is the L1-info deposit, so it
/// converts into an `L2BlockInfo` without being the configured genesis. Blocks with different
/// `extra_data` share a height but not a hash.
pub fn test_l2_block(number: u64, parent_hash: B256, extra_data: u8) -> OpBlock {
    let mut block = OpBlock::default();
    block.header.extra_data = Bytes::from(vec![extra_data]);
    block.header.number = number;
    block.header.parent_hash = parent_hash;
    block.header.timestamp = number * 2;
    block.body.transactions.push(l1_info_deposit());
    block
}

/// `block` as the execution layer serves it over RPC.
pub fn test_rpc_block(block: &OpBlock) -> TestRpcBlock {
    let transactions = block
        .body
        .transactions
        .iter()
        .cloned()
        .map(|transaction| {
            op_alloy_rpc_types::Transaction::from_transaction(
                Recovered::new_unchecked(transaction, Address::ZERO),
                Default::default(),
            )
        })
        .collect();
    TestRpcBlock::new(RpcHeader::new(block.header.clone()), BlockTransactions::Full(transactions))
}

/// An L2 block with `header` and only the L1 info deposit for `info`, as an RPC returns it with
/// full transactions. The block hash is computed from `header`.
pub fn rpc_l2_block(header: Header, info: &L1BlockInfoBedrock) -> TestRpcBlock {
    let number = header.number;
    let mut block = TestRpcBlock::default();
    block.header.hash = header.hash_slow();
    block.header.inner = header;
    block.transactions = BlockTransactions::Full(vec![op_alloy_rpc_types::Transaction {
        inner: alloy_rpc_types_eth::Transaction {
            block_hash: None,
            block_number: Some(number),
            block_timestamp: None,
            effective_gas_price: Some(1),
            inner: Recovered::new_unchecked(l1_info_deposit_for(info), Default::default()),
            transaction_index: Some(0),
        },
        deposit_nonce: None,
        deposit_receipt_version: None,
    }]);
    block
}

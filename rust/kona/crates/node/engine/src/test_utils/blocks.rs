//! L2 blocks, and their RPC form as the execution layer serves them.

use super::l1_info_deposit;
use alloy_consensus::transaction::Recovered;
use alloy_primitives::{Address, B256, Bytes};
use alloy_rpc_types_eth::{BlockTransactions, Header as RpcHeader};
use kona_protocol::OpAttributesWithParent;
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

/// The block that `attributes` derive, as the execution layer serves it over RPC, so that
/// consolidating `attributes` against it finds a match. `attributes` must carry only the L1-info
/// deposit and a gas limit.
pub fn test_rpc_block_from_attributes(attributes: &OpAttributesWithParent) -> TestRpcBlock {
    let payload_attributes = &attributes.attributes.payload_attributes;
    let mut block = OpBlock::default();
    block.header.beneficiary = payload_attributes.suggested_fee_recipient;
    block.header.gas_limit = attributes.attributes.gas_limit.expect("attributes set a gas limit");
    block.header.mix_hash = payload_attributes.prev_randao;
    block.header.number = attributes.block_number();
    block.header.parent_beacon_block_root = payload_attributes.parent_beacon_block_root;
    block.header.parent_hash = attributes.parent.block_info.hash;
    block.header.timestamp = payload_attributes.timestamp;
    block.body.transactions.push(l1_info_deposit());
    test_rpc_block(&block)
}

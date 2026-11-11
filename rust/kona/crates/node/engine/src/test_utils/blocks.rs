//! L2 blocks, and their RPC form as the execution layer serves them.

use super::{l1_info_deposit, l1_info_deposit_at};
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
    test_l2_block_at_l1_origin(number, parent_hash, extra_data, 0)
}

/// [`test_l2_block`] with its L1 origin at `l1_origin_number`.
pub fn test_l2_block_at_l1_origin(
    number: u64,
    parent_hash: B256,
    extra_data: u8,
    l1_origin_number: u64,
) -> OpBlock {
    let mut block = OpBlock::default();
    block.header.extra_data = Bytes::from(vec![extra_data]);
    block.header.number = number;
    block.header.parent_hash = parent_hash;
    block.header.timestamp = number * 2;
    block.body.transactions.push(l1_info_deposit_at(l1_origin_number));
    block
}

/// Blocks `from..=to` built on `parent_hash` with [`test_l2_block`], all tagged `extra_data`, so
/// that chains with different tags share heights but not hashes.
pub fn test_l2_chain(parent_hash: B256, from: u64, to: u64, extra_data: u8) -> Vec<OpBlock> {
    let mut parent_hash = parent_hash;
    (from..=to)
        .map(|number| {
            let block = test_l2_block(number, parent_hash, extra_data);
            parent_hash = block.header.hash_slow();
            block
        })
        .collect()
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

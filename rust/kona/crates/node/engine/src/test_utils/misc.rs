use alloy_eips::BlockNumHash;
use alloy_primitives::{B256, Sealed};
use kona_protocol::{BlockInfo, L1BlockInfoBedrock, L2BlockInfo};

/// Helper to create a test `L2BlockInfo` at a specific block number
pub fn test_block_info(number: u64) -> L2BlockInfo {
    L2BlockInfo {
        block_info: BlockInfo {
            number,
            hash: B256::random(),
            parent_hash: B256::random(),
            timestamp: number * 2,
        },
        l1_origin: BlockNumHash::default(),
        seq_num: 0,
    }
}

/// Builds a full L2 block with a real L1-info deposit for reset traversal tests.
pub fn test_l2_block(
    number: u64,
    origin: u64,
    sequence: u64,
    parent: B256,
) -> alloy_rpc_types_eth::Block<op_alloy_rpc_types::Transaction> {
    let data = L1BlockInfoBedrock::new(
        origin,
        0,
        0,
        B256::with_last_byte(origin as u8),
        sequence,
        Default::default(),
        Default::default(),
        Default::default(),
    )
    .encode_calldata();
    let transaction =
        op_alloy_consensus::OpTxEnvelope::Deposit(Sealed::new(op_alloy_consensus::TxDeposit {
            input: data,
            ..Default::default()
        }));
    let transaction = op_alloy_rpc_types::Transaction::from_transaction(
        alloy_consensus::transaction::Recovered::new_unchecked(transaction, Default::default()),
        Default::default(),
    );
    let mut block: alloy_rpc_types_eth::Block<op_alloy_rpc_types::Transaction> = Default::default();
    block.header.inner.number = number;
    block.header.inner.parent_hash = parent;
    block.header.inner.timestamp = number * 2;
    block.header.hash = block.header.inner.hash_slow();
    block.transactions = alloy_rpc_types_eth::BlockTransactions::Full(vec![transaction]);
    block
}

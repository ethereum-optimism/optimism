use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use kona_protocol::{BlockInfo, L2BlockInfo};

/// Helper to create a test L2BlockInfo at a specific block number
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

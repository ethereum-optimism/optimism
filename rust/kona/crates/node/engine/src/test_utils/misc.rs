use crate::{ConsolidateInput, DelegatedSafeHeads};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use kona_protocol::{BlockInfo, L2BlockInfo};

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

/// Delegated safe heads as a consolidation input. Pass the same block twice for the heads of a
/// delegate before interop.
pub fn delegated_safe_heads(
    local_safe_head: L2BlockInfo,
    safe_head: L2BlockInfo,
) -> ConsolidateInput {
    DelegatedSafeHeads::try_new(local_safe_head, safe_head)
        .expect("the cross-safe head must be at or below the local-safe head, on its chain")
        .into()
}

//! Finding the highest unsafe block whose L1 origin is canonical.

use crate::{EngineClient, SyncStartError};
use alloy_eips::{BlockId, BlockNumHash};
use kona_protocol::L2BlockInfo;
use tracing::info;

/// Walks back from `unsafe_head` to the highest block whose L1 origin is canonical, never below
/// `floor`, a block whose L1 data is canonical.
///
/// An L1 origin is canonical when the L1 node has that block at that height. A block built on an
/// origin the L1 node has replaced cannot become safe. One built on an origin the L1 node has not
/// seen yet is not known to be canonical, and is walked back from too; op-node keeps such a head
/// as plausible. The walk visits only blocks built on such origins, checking each origin once,
/// and links the block just above `floor` to it by its parent hash.
///
/// Fails with [`SyncStartError::UnsafeHeadBelowFloor`] if `unsafe_head` is below `floor`, with
/// [`SyncStartError::BlockNotFound`] if the engine lacks a block on the way, and with
/// [`SyncStartError::UnsafeChainMismatch`] if the chain does not lead back to `floor`.
pub(super) async fn find_canonical_unsafe_head(
    engine_client: &EngineClient,
    mut unsafe_head: L2BlockInfo,
    floor: L2BlockInfo,
) -> Result<L2BlockInfo, SyncStartError> {
    let (head, floor_number) = (unsafe_head.block_info, floor.block_info.number);
    if head.number < floor_number {
        return Err(SyncStartError::UnsafeHeadBelowFloor {
            unsafe_head: head.number,
            floor: floor_number,
        });
    }
    if head.number == floor_number {
        return if head.hash == floor.block_info.hash {
            Ok(floor)
        } else {
            Err(SyncStartError::UnsafeChainMismatch {
                expected: floor.block_info.hash,
                found: head.hash,
            })
        };
    }

    let genesis = &engine_client.cfg().genesis;
    // Consecutive blocks mostly share an L1 origin; check each replaced one only once.
    let mut replaced_origin = None;
    loop {
        if replaced_origin != Some(unsafe_head.l1_origin) {
            if is_canonical_l1(engine_client, unsafe_head.l1_origin).await? {
                return Ok(unsafe_head);
            }
            info!(
                target: "sync_start",
                l2_unsafe = %unsafe_head.block_info.number,
                l1_origin = ?unsafe_head.l1_origin,
                "Unsafe block's L1 origin is not canonical; walking back"
            );
            replaced_origin = Some(unsafe_head.l1_origin);
        }

        let parent_hash = unsafe_head.block_info.parent_hash;
        if unsafe_head.block_info.number == floor_number + 1 {
            return if parent_hash == floor.block_info.hash {
                Ok(floor)
            } else {
                Err(SyncStartError::UnsafeChainMismatch {
                    expected: floor.block_info.hash,
                    found: parent_hash,
                })
            };
        }
        let parent = engine_client
            .get_l2_block(parent_hash.into())
            .full()
            .await?
            .ok_or(SyncStartError::BlockNotFound(parent_hash.into()))?;
        unsafe_head = L2BlockInfo::from_block_and_genesis(&parent.into_consensus(), genesis)?;
    }
}

/// Whether `block` is the canonical L1 block at its height. A block the L1 node does not have at
/// that height yet is not known to be canonical.
pub(super) async fn is_canonical_l1(
    engine_client: &EngineClient,
    block: BlockNumHash,
) -> Result<bool, SyncStartError> {
    Ok(engine_client
        .get_l1_block(BlockId::number(block.number))
        .await?
        .is_some_and(|canonical| canonical.header.hash == block.hash))
}

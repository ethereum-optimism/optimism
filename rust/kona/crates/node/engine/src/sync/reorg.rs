//! Choosing the forkchoice to keep across an L1 reorg.
//!
//! [`find_starting_forkchoice`] cannot know which L1 block made the engine's safe head safe, so it
//! walks the safe head back a full sequencing window, stopping early only at the finalized head.
//! That is the right answer on a cold start, but after an L1 reorg detected mid-derivation it
//! makes the cost of a reset independent of the reorg's depth: a one-block replacement of the L1
//! tip rewinds the safe head as far as a cold start does, and while L1 finality is stalled that is
//! the whole sequencing window.
//!
//! Derivation does know where each safe head came from. A [`DerivedSafeHead`] records that all of
//! the L1 data `safe_head` depends on is at or below `derived_from`. Derivation is a function of
//! the L1 chain it has read, so while `derived_from` is canonical the reorg changed none of that
//! input and `safe_head` is still the safe head that L1 chain derives. Keeping it is then exactly
//! as sound as any reset: the pipeline restarts from a known-correct safe head, a channel timeout
//! behind its L1 origin, as it does after [`find_starting_forkchoice`].
//!
//! The unsafe chain is checked on its own terms. An unsafe block whose L1 origin the reorg
//! replaced commits to a non-canonical L1 block and can never become safe, so the unsafe head
//! moves back to the highest block whose L1 origin is still canonical. That walk visits only the
//! blocks built on replaced L1 blocks, and never passes the kept safe head, whose L1 data is
//! canonical.
//!
//! [`find_starting_forkchoice`]: crate::find_starting_forkchoice

use super::unsafe_head::{find_canonical_unsafe_head, is_canonical_l1};
use crate::{EngineClient, EngineState, EngineSyncState, L2ForkchoiceState, SyncStartError};
use alloy_eips::{BlockId, BlockNumHash};
use kona_protocol::L2BlockInfo;
use tracing::info;

/// A safe head, and an L1 block that all of the L1 data the safe head depends on is at or below.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct DerivedSafeHead {
    /// The safe head.
    pub safe_head: L2BlockInfo,
    /// An L1 block at or above every L1 block `safe_head` depends on, including through its
    /// ancestors.
    pub derived_from: BlockNumHash,
}

/// Picks the forkchoice to keep after an L1 reorg, or `None` when only a full reset is sound.
///
/// `candidates` must be ordered newest first. The safe head is the first candidate whose
/// `derived_from` is still canonical, kept only if it is still on the engine's chain, at or below
/// the local safe head and at or above the finalized head. The unsafe head is the highest block
/// between it and the current unsafe head whose L1 origin is still canonical. The finalized head
/// is unchanged.
pub async fn find_reorg_forkchoice(
    engine_client: &EngineClient,
    state: &EngineState,
    candidates: &[DerivedSafeHead],
) -> Result<Option<L2ForkchoiceState>, SyncStartError> {
    let sync_state = state.sync_state;
    let Some(safe) = find_reorg_safe_head(engine_client, &sync_state, candidates).await? else {
        return Ok(None);
    };
    let un_safe =
        match find_canonical_unsafe_head(engine_client, sync_state.unsafe_head(), safe).await {
            Ok(un_safe) => un_safe,
            // An unsafe chain that misses a block or does not lead to the kept safe head is left to
            // the full reset.
            Err(
                SyncStartError::BlockNotFound(_) |
                SyncStartError::UnsafeChainMismatch { .. } |
                SyncStartError::UnsafeHeadBelowFloor { .. },
            ) => {
                return Ok(None);
            }
            Err(err) => return Err(err),
        };

    info!(
        target: "sync_start",
        safe_head = %safe.block_info.number,
        unsafe_head = %un_safe.block_info.number,
        "Keeping heads derived from canonical L1 data across the L1 reorg"
    );
    Ok(Some(L2ForkchoiceState { un_safe, safe, finalized: sync_state.finalized_head() }))
}

/// The first of `candidates` whose L1 data is canonical, if the engine can keep it as its safe
/// head.
async fn find_reorg_safe_head(
    engine_client: &EngineClient,
    sync_state: &EngineSyncState,
    candidates: &[DerivedSafeHead],
) -> Result<Option<L2BlockInfo>, SyncStartError> {
    for candidate in candidates {
        if !is_canonical_l1(engine_client, candidate.derived_from).await? {
            continue;
        }

        let safe_head = candidate.safe_head;
        let number = safe_head.block_info.number;
        if number > sync_state.local_safe_head().block_info.number ||
            number < sync_state.finalized_head().block_info.number
        {
            return Ok(None);
        }
        let on_chain = engine_client
            .get_l2_block(BlockId::number(number))
            .await?
            .is_some_and(|block| block.header.hash == safe_head.block_info.hash);
        return Ok(on_chain.then_some(safe_head));
    }

    Ok(None)
}

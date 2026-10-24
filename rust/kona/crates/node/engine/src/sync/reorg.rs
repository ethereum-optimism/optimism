//! Choosing a safe head to keep across an L1 reorg.
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
//! [`find_starting_forkchoice`]: crate::find_starting_forkchoice

use crate::{EngineClient, EngineState, SyncStartError};
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

/// Picks the safe head to keep after an L1 reorg, or `None` when only a full reset is sound.
///
/// `candidates` must be ordered newest first. The first candidate whose `derived_from` is still
/// canonical is chosen. It is kept only if it is still on the engine's chain, at or below the
/// local safe head and at or above the finalized head. No candidate is kept if the unsafe head's
/// L1 origin was reorged out: [`find_starting_forkchoice`] must rewind that unsafe chain.
///
/// [`find_starting_forkchoice`]: crate::find_starting_forkchoice
pub async fn find_reorg_safe_head(
    engine_client: &EngineClient,
    state: &EngineState,
    candidates: &[DerivedSafeHead],
) -> Result<Option<L2BlockInfo>, SyncStartError> {
    let sync_state = state.sync_state;
    let unsafe_head = sync_state.unsafe_head();
    if !is_canonical_l1(engine_client, unsafe_head.l1_origin).await? {
        info!(
            target: "sync_start",
            l1_origin = ?unsafe_head.l1_origin,
            "Unsafe head's L1 origin is no longer canonical; a full reset is needed"
        );
        return Ok(None);
    }

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
        if !on_chain {
            return Ok(None);
        }

        info!(
            target: "sync_start",
            safe_head = %number,
            derived_from = ?candidate.derived_from,
            "Keeping a safe head derived from canonical L1 data across the L1 reorg"
        );
        return Ok(Some(safe_head));
    }

    Ok(None)
}

/// Whether `block` is the canonical L1 block at its height. A block the L1 node does not have at
/// that height yet is not known to be canonical.
async fn is_canonical_l1(
    engine_client: &EngineClient,
    block: BlockNumHash,
) -> Result<bool, SyncStartError> {
    Ok(engine_client
        .get_l1_block(BlockId::number(block.number))
        .await?
        .is_some_and(|canonical| canonical.header.hash == block.hash))
}

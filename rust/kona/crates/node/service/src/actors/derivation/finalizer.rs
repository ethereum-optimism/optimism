//! The [`L2Finalizer`] for the derivation actor.

use kona_protocol::{BlockInfo, OpAttributesWithParent};
use std::collections::BTreeMap;

/// An internal type alias for L1 block numbers.
type L1BlockNumber = u64;

/// An internal type alias for L2 block numbers.
type L2BlockNumber = u64;

/// The [`L2Finalizer`] is responsible for tracking L2 blocks derived from L1 blocks and
/// determining which L2 blocks can be finalized when L1 blocks are finalized.
///
/// It maintains a queue of derived L2 blocks that are awaiting finalization, and returns
/// the L2 block numbers that can be finalized as new finalized L1 blocks are received.
#[derive(Debug, Default)]
pub(crate) struct L2Finalizer {
    /// A map of `L1 block number -> highest derived L2 block number` within the L1 epoch, used to
    /// track derived [`OpAttributesWithParent`] awaiting finalization. When a new finalized L1
    /// block is received, the highest L2 block whose inputs are contained within the finalized
    /// L1 chain is finalized.
    awaiting_finalization: BTreeMap<L1BlockNumber, L2BlockNumber>,
}

impl L2Finalizer {
    /// Enqueues a derived [`OpAttributesWithParent`] for finalization. When a new finalized L1
    /// block is observed that is `>=` the height of [`OpAttributesWithParent::derived_from`], the
    /// L2 block associated with the payload attributes will be finalized.
    pub(crate) fn enqueue_for_finalization(&mut self, attributes: &OpAttributesWithParent) {
        self.awaiting_finalization
            .entry(
                attributes.derived_from.map(|b| b.number).expect(
                    "Fatal: Cannot enqueue attributes for finalization that weren't derived",
                ),
            )
            .and_modify(|n| *n = (*n).max(attributes.block_number()))
            .or_insert(attributes.block_number());
    }

    /// Clears the finalization queue.
    pub(crate) fn clear(&mut self) {
        self.awaiting_finalization.clear();
    }

    /// Attempts to find L2 blocks that can be finalized based on the new finalized L1 block.
    ///
    /// Returns `Some(l2_block_number)` if there is an L2 block that can be finalized,
    /// or `None` if no blocks are ready for finalization.
    pub(crate) fn try_finalize_next(
        &mut self,
        new_finalized_l1_block: BlockInfo,
    ) -> Option<L2BlockNumber> {
        // Find the highest safe L2 block that is contained within the finalized chain,
        // that the finalizer is aware of.
        let highest_safe =
            self.awaiting_finalization.range(..=new_finalized_l1_block.number).next_back();

        // If the highest safe block is found, return it and drain the
        // queue of all L1 blocks not contained in the finalized L1 chain.
        if let Some((_, highest_safe_number)) = highest_safe {
            let result = *highest_safe_number;
            self.awaiting_finalization.retain(|&number, _| number > new_finalized_l1_block.number);
            Some(result)
        } else {
            None
        }
    }
}

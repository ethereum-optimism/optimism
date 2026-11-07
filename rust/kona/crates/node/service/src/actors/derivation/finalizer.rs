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
    /// The number of the highest finalized L1 block received.
    finalized_l1_number: Option<L1BlockNumber>,
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
            .or_insert_with(|| attributes.block_number());
    }

    /// Clears the finalization queue. The finalized L1 number is kept, since an L2 reset does not
    /// change L1 finality.
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
        if self.finalized_l1_number.is_none_or(|known| known < new_finalized_l1_block.number) {
            self.finalized_l1_number = Some(new_finalized_l1_block.number);
        }
        self.try_finalize_against_known_l1()
    }

    /// Attempts to finalize L2 blocks that became safe after the highest finalized L1 block was
    /// received, such as blocks derived again after a reset, without waiting for L1 finality to
    /// move.
    ///
    /// The entries up to the finalized L1 block are drained with the returned block: a block whose
    /// finalization does not go through has to be queued again to be returned again.
    pub(crate) fn try_finalize_against_known_l1(&mut self) -> Option<L2BlockNumber> {
        let finalized_l1_number = self.finalized_l1_number?;

        // Find the highest safe L2 block that is contained within the finalized chain,
        // that the finalizer is aware of.
        let (_, &highest_safe_number) =
            self.awaiting_finalization.range(..=finalized_l1_number).next_back()?;

        // Drain the queue of all L1 blocks contained in the finalized L1 chain.
        self.awaiting_finalization.retain(|&number, _| number > finalized_l1_number);
        Some(highest_safe_number)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use kona_protocol::L2BlockInfo;
    use op_alloy_rpc_types_engine::OpPayloadAttributes;
    use rstest::rstest;

    const DERIVING_L1: u64 = 5;
    const DERIVED_L2: u64 = 100;

    fn l1_block(number: u64) -> BlockInfo {
        BlockInfo { number, ..Default::default() }
    }

    /// Attributes for L2 block [`DERIVED_L2`], derived from L1 block [`DERIVING_L1`].
    fn derived_attributes() -> OpAttributesWithParent {
        let mut parent = L2BlockInfo::default();
        parent.block_info.number = DERIVED_L2 - 1;
        OpAttributesWithParent {
            attributes: OpPayloadAttributes::default(),
            derived_from: Some(l1_block(DERIVING_L1)),
            is_last_in_span: true,
            parent,
        }
    }

    /// A queued block is finalized once L1 finality reaches the block it was derived from.
    #[rstest]
    #[case::before_its_l1_block(DERIVING_L1 - 1, None)]
    #[case::at_its_l1_block(DERIVING_L1, Some(DERIVED_L2))]
    #[case::past_its_l1_block(DERIVING_L1 + 1, Some(DERIVED_L2))]
    fn a_queued_block_finalizes_with_the_l1_block_it_was_derived_from(
        #[case] finalized_l1: u64,
        #[case] finalized_l2: Option<u64>,
    ) {
        let mut finalizer = L2Finalizer::default();
        finalizer.enqueue_for_finalization(&derived_attributes());

        assert_eq!(finalizer.try_finalize_next(l1_block(finalized_l1)), finalized_l2);
    }

    /// A finalized L1 block below the highest one received does not lower it.
    #[test]
    fn a_lower_finalized_l1_block_keeps_the_highest_one() {
        let mut finalizer = L2Finalizer::default();
        assert_eq!(finalizer.try_finalize_next(l1_block(DERIVING_L1)), None);
        finalizer.enqueue_for_finalization(&derived_attributes());

        assert_eq!(finalizer.try_finalize_next(l1_block(DERIVING_L1 - 1)), Some(DERIVED_L2));
    }

    /// Clearing the queue keeps the finalized L1 block, so a block queued again afterwards is
    /// finalized without new L1 finality.
    #[test]
    fn a_block_queued_after_a_clear_finalizes_against_the_kept_l1_block() {
        let mut finalizer = L2Finalizer::default();
        finalizer.enqueue_for_finalization(&derived_attributes());
        assert_eq!(finalizer.try_finalize_next(l1_block(DERIVING_L1)), Some(DERIVED_L2));
        finalizer.clear();
        finalizer.enqueue_for_finalization(&derived_attributes());

        assert_eq!(finalizer.try_finalize_against_known_l1(), Some(DERIVED_L2));
    }

    /// Nothing is finalized before any L1 finality is known, or once the queue is drained.
    #[test]
    fn nothing_finalizes_without_l1_finality_or_a_queued_block() {
        let mut finalizer = L2Finalizer::default();
        finalizer.enqueue_for_finalization(&derived_attributes());
        assert_eq!(finalizer.try_finalize_against_known_l1(), None);
        assert_eq!(finalizer.try_finalize_next(l1_block(DERIVING_L1)), Some(DERIVED_L2));

        assert_eq!(finalizer.try_finalize_against_known_l1(), None);
    }
}

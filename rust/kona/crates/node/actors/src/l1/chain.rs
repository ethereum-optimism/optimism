//! The canonical L1 chain the actor holds.

use kona_protocol::BlockInfo;
use std::collections::VecDeque;

/// The canonical L1 chain from the latest finalized block to the head, and the latest safe block.
#[derive(Debug, Clone, PartialEq, Eq)]
pub(super) struct Chain {
    /// The latest finalized block. Blocks before it cannot reorg, so the chain does not hold them.
    finalized: BlockInfo,
    /// The canonical descendants of the finalized block, oldest first. Each block's parent is the
    /// block before it.
    descendants: VecDeque<BlockInfo>,
    /// The latest safe block.
    safe: BlockInfo,
    /// Whether the actor is linking a head it polled to the chain. Until it has, a height after
    /// the head may already be on L1.
    linking: bool,
}

/// Where a height falls relative to the chain.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(super) enum Height {
    /// Before the finalized block.
    Final,
    /// The canonical block at the height.
    Held(BlockInfo),
    /// After the head.
    Ahead,
}

/// How a block joins the chain.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(super) enum Join {
    /// The block is held.
    Held,
    /// The block's parent is held.
    Extends,
    /// The block's parent is not held and is newer than the finalized block, so it must join
    /// first.
    NeedsParent,
    /// The block does not descend from the finalized block.
    Rejected,
}

impl Chain {
    /// Creates a chain holding only the finalized block, which links the first head it polls.
    pub(super) const fn new(finalized: BlockInfo) -> Self {
        Self { finalized, descendants: VecDeque::new(), safe: finalized, linking: true }
    }

    /// Returns whether the actor is linking a head it polled, so that a height after the head may
    /// already be on L1.
    pub(super) const fn linking(&self) -> bool {
        self.linking
    }

    /// Sets whether the actor is linking a head it polled. Returns whether it changed.
    pub(super) const fn set_linking(&mut self, linking: bool) -> bool {
        std::mem::replace(&mut self.linking, linking) != linking
    }

    /// Returns the canonical head.
    pub(super) fn head(&self) -> BlockInfo {
        *self.descendants.back().unwrap_or(&self.finalized)
    }

    /// Returns the latest finalized block.
    pub(super) const fn finalized(&self) -> BlockInfo {
        self.finalized
    }

    /// Returns the latest safe block.
    pub(super) const fn safe(&self) -> BlockInfo {
        self.safe
    }

    /// Returns where `number` falls relative to the chain.
    pub(super) fn at(&self, number: u64) -> Height {
        let Some(offset) = number.checked_sub(self.finalized.number) else {
            return Height::Final;
        };
        let Some(index) = offset.checked_sub(1) else {
            return Height::Held(self.finalized);
        };
        usize::try_from(index)
            .ok()
            .and_then(|index| self.descendants.get(index))
            .map_or(Height::Ahead, |block| Height::Held(*block))
    }

    /// Returns whether the chain holds `block`.
    fn holds(&self, block: &BlockInfo) -> bool {
        self.at(block.number) == Height::Held(*block)
    }

    /// Returns how `block` joins the chain.
    pub(super) fn join(&self, block: &BlockInfo) -> Join {
        if self.holds(block) {
            return Join::Held;
        }
        if block.number <= self.finalized.number {
            return Join::Rejected;
        }
        match self.at(block.number - 1) {
            Height::Held(parent) if parent.hash == block.parent_hash => Join::Extends,
            // A different block is the finalized block's height.
            Height::Held(_) if block.number - 1 == self.finalized.number => Join::Rejected,
            _ => Join::NeedsParent,
        }
    }

    /// Makes the last block of `branch` the head. `branch` is oldest first, each block's parent is
    /// the block before it, and its first block joins the chain as [`Join::Held`] or
    /// [`Join::Extends`]. Returns how many blocks were reorged out.
    pub(super) fn set_head(&mut self, branch: Vec<BlockInfo>) -> usize {
        let Some(first) = branch.first() else { return 0 };
        let (last_kept, skip) =
            if self.holds(first) { (first.number, 1) } else { (first.number - 1, 0) };
        let kept = self.offset(last_kept);
        let orphaned = self.descendants.len().saturating_sub(kept);
        self.descendants.truncate(kept);
        self.descendants.extend(branch.into_iter().skip(skip));
        orphaned
    }

    /// Moves the chain's start to a newly finalized block, dropping the blocks before it. A block
    /// older than the finalized block is ignored. Returns whether the chain did not hold the block,
    /// and so restarted from it.
    pub(super) fn finalize(&mut self, finalized: BlockInfo) -> bool {
        if finalized.number < self.finalized.number {
            return false;
        }
        let restarted = !self.holds(&finalized);
        if restarted {
            // The finalized block is ahead of the head, or the held chain is not canonical.
            self.descendants.clear();
        } else {
            let dropped = self.offset(finalized.number);
            self.descendants.drain(..dropped);
        }
        self.finalized = finalized;
        restarted
    }

    /// Returns how many blocks after the finalized block lead up to the held height `number`.
    fn offset(&self, number: u64) -> usize {
        usize::try_from(number - self.finalized.number)
            .expect("the chain holds its blocks in memory")
    }

    /// Sets the latest safe block. Returns whether it changed.
    pub(super) fn set_safe(&mut self, safe: BlockInfo) -> bool {
        std::mem::replace(&mut self.safe, safe) != safe
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::l1::test_utils::{block, chain as blocks, chain_of};

    fn held(chain: &Chain) -> Vec<BlockInfo> {
        std::iter::once(chain.finalized).chain(chain.descendants.iter().copied()).collect()
    }

    #[test]
    fn places_heights_relative_to_the_chain() {
        let chain = chain_of(&blocks(3..=5, 1));
        assert_eq!(chain.at(2), Height::Final);
        assert_eq!(chain.at(3), Height::Held(block(3, 1, 1)));
        assert_eq!(chain.at(5), Height::Held(block(5, 1, 1)));
        assert_eq!(chain.at(6), Height::Ahead);
        assert_eq!(chain.head(), block(5, 1, 1));
        assert_eq!(chain.finalized(), block(3, 1, 1));
    }

    #[test]
    fn joins_blocks_that_descend_from_the_finalized_block() {
        let chain = chain_of(&blocks(3..=5, 1));
        assert_eq!(chain.join(&block(4, 1, 1)), Join::Held);
        assert_eq!(chain.join(&block(6, 1, 1)), Join::Extends);
        // A fork off a held block.
        assert_eq!(chain.join(&block(5, 2, 1)), Join::Extends);
        // A fork whose parent is not held.
        assert_eq!(chain.join(&block(5, 2, 2)), Join::NeedsParent);
        assert_eq!(chain.join(&block(8, 1, 1)), Join::NeedsParent);
        // A different block at, or a child of a different block at, the finalized height.
        assert_eq!(chain.join(&block(3, 9, 9)), Join::Rejected);
        assert_eq!(chain.join(&block(4, 9, 9)), Join::Rejected);
        assert_eq!(chain.join(&block(2, 1, 1)), Join::Rejected);
    }

    #[test]
    fn setting_the_head_reorgs_out_the_blocks_it_replaces() {
        let mut chain = chain_of(&blocks(1..=5, 1));
        assert_eq!(chain.set_head(vec![block(6, 1, 1)]), 0);
        assert_eq!(held(&chain), blocks(1..=6, 1));

        // Blocks 4 to 6 are replaced.
        assert_eq!(chain.set_head(vec![block(4, 2, 1), block(5, 2, 2)]), 3);
        assert_eq!(chain.head(), block(5, 2, 2));
        assert_eq!(chain.at(3), Height::Held(block(3, 1, 1)));

        // The head moves back to a held block.
        assert_eq!(chain.set_head(vec![block(4, 2, 1)]), 1);
        assert_eq!(chain.head(), block(4, 2, 1));

        // The head moves back to the finalized block.
        assert_eq!(chain.set_head(vec![block(1, 1, 1)]), 3);
        assert_eq!(held(&chain), vec![block(1, 1, 1)]);
    }

    #[test]
    fn finalizing_a_held_block_drops_older_blocks() {
        let mut chain = chain_of(&blocks(1..=5, 1));
        assert!(!chain.finalize(block(4, 1, 1)));
        assert_eq!(held(&chain), blocks(4..=5, 1));
        // An older finalized block changes nothing.
        assert!(!chain.finalize(block(2, 1, 1)));
        assert_eq!(held(&chain), blocks(4..=5, 1));
        // Finalizing the head leaves only the head.
        assert!(!chain.finalize(block(5, 1, 1)));
        assert_eq!(held(&chain), vec![block(5, 1, 1)]);
    }

    #[test]
    fn finalizing_a_block_the_chain_does_not_hold_restarts_from_it() {
        // Ahead of the head.
        let mut chain = chain_of(&blocks(1..=5, 1));
        assert!(chain.finalize(block(8, 1, 1)));
        assert_eq!(held(&chain), vec![block(8, 1, 1)]);

        // Conflicting with the held chain.
        let mut chain = chain_of(&blocks(1..=5, 1));
        assert!(chain.finalize(block(4, 2, 2)));
        assert_eq!(held(&chain), vec![block(4, 2, 2)]);
    }

    #[test]
    fn records_the_safe_block() {
        let mut chain = Chain::new(block(1, 1, 1));
        assert_eq!(chain.safe(), block(1, 1, 1));
        assert!(chain.set_safe(block(2, 1, 1)));
        assert!(!chain.set_safe(block(2, 1, 1)));
        assert_eq!(chain.safe(), block(2, 1, 1));
    }
}

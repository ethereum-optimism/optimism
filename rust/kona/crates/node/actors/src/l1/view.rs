//! The actor's view of L1.

use super::{Update, non_empty::NonEmptyVecDeque};
use alloy_primitives::B256;
use kona_protocol::BlockInfo;
use std::{collections::HashMap, num::NonZeroUsize};

/// The canonical L1 chain from the latest finalized block to the head, and other blocks it knows.
#[derive(Debug)]
pub(super) struct View {
    /// Canonical blocks, oldest first, starting with the latest finalized block. Each block's
    /// parent is the block before it.
    blocks: NonEmptyVecDeque<BlockInfo>,
    /// Other known blocks, by hash, no older than the finalized block: blocks reorged out of
    /// `blocks`, and new heads and their ancestors. A new head joins `blocks` through them once
    /// they link it to `blocks`. A consumer only learns blocks from the view, so its cursor can be
    /// traced through them to the canonical chain. A block can also be canonical; canonical blocks
    /// are searched first.
    detached: HashMap<B256, BlockInfo>,
}

/// What [`View::extend`] did with a new head.
#[derive(Debug, Clone, PartialEq, Eq)]
pub(super) enum Extension {
    /// The new head is the view's head. `orphaned` blocks were reorged out.
    Moved {
        /// How many blocks were reorged out.
        orphaned: usize,
    },
    /// The view needs the block with this hash, an ancestor of the new head, to link the head to
    /// the canonical chain.
    NeedsParent(B256),
    /// The new head does not descend from the finalized block, so it is not canonical.
    Rejected,
}

impl View {
    /// Creates a view holding only the finalized block.
    pub(super) fn new(finalized: BlockInfo) -> Self {
        Self { blocks: NonEmptyVecDeque::new(finalized), detached: HashMap::new() }
    }

    /// Returns the canonical head.
    pub(super) fn head(&self) -> BlockInfo {
        *self.blocks.back()
    }

    /// Returns the latest finalized block, the oldest block the view holds.
    pub(super) fn finalized(&self) -> BlockInfo {
        *self.blocks.front()
    }

    /// Returns the index of the block, if it is canonical and held.
    fn index_of(&self, number: u64, hash: B256) -> Option<usize> {
        let index = usize::try_from(number.checked_sub(self.finalized().number)?).ok()?;
        (self.blocks.get(index)?.hash == hash).then_some(index)
    }

    /// Returns the index of the block's parent, if it is canonical and held.
    fn parent_index(&self, block: &BlockInfo) -> Option<usize> {
        self.index_of(block.number.checked_sub(1)?, block.parent_hash)
    }

    /// Returns what changed since `cursor`, or `None` if nothing did.
    pub(super) fn updates_since(&self, cursor: BlockInfo, max: NonZeroUsize) -> Option<Update> {
        let base = self.finalized();
        if cursor.number < base.number {
            return Some(Update::Behind { base });
        }
        let after = |index: usize| -> Vec<BlockInfo> {
            self.blocks.iter().skip(index + 1).take(max.get()).copied().collect()
        };
        if let Some(index) = self.index_of(cursor.number, cursor.hash) {
            let blocks = after(index);
            return (!blocks.is_empty()).then_some(Update::Advanced(blocks));
        }
        // Follow the cursor's chain back through the detached blocks until it rejoins the canonical
        // chain. Each step visits a different block, which bounds the walk.
        let mut block = cursor;
        for _ in 0..=self.detached.len() {
            if let Some(index) = self.parent_index(&block) {
                return Some(Update::Reorged {
                    ancestor: self.blocks[index],
                    blocks: after(index),
                });
            }
            match self.detached.get(&block.parent_hash) {
                Some(parent) => block = *parent,
                None => break,
            }
        }
        Some(Update::Unknown)
    }

    /// Adds a block the view may need to link a new head to the canonical chain.
    pub(super) fn insert(&mut self, block: BlockInfo) {
        if block.number > self.finalized().number {
            self.detached.insert(block.hash, block);
        }
    }

    /// Moves the view's head to `head`, linking it to the canonical chain through the detached
    /// blocks. If they do not link it, the view is unchanged and needs the missing ancestor.
    pub(super) fn extend(&mut self, head: BlockInfo) -> Extension {
        if let Some(index) = self.index_of(head.number, head.hash) {
            // Unchanged, or the head moved back to one of its ancestors.
            return Extension::Moved { orphaned: self.detach_after(index) };
        }
        // The head and its known ancestors, newest first. Each step visits a different detached
        // block, which bounds the walk.
        let mut branch = vec![head];
        for _ in 0..=self.detached.len() {
            let oldest = branch[branch.len() - 1];
            if let Some(index) = self.parent_index(&oldest) {
                let orphaned = self.detach_after(index);
                for block in branch.into_iter().rev() {
                    self.blocks.push_back(block);
                }
                return Extension::Moved { orphaned };
            }
            // The parent is not held, so if it would be at or behind the finalized block, it is not
            // canonical.
            if oldest.number.saturating_sub(1) <= self.finalized().number {
                break;
            }
            match self.detached.get(&oldest.parent_hash) {
                Some(parent) => branch.push(*parent),
                None => return Extension::NeedsParent(oldest.parent_hash),
            }
        }
        Extension::Rejected
    }

    /// Moves the view's start to a newly finalized block, dropping the blocks before it. Returns
    /// whether the view did not hold the block, and so restarted from it.
    ///
    /// A block older than the view's finalized block is ignored.
    pub(super) fn finalize(&mut self, finalized: BlockInfo) -> bool {
        if finalized.number < self.finalized().number {
            return false;
        }
        let restarted = match self.index_of(finalized.number, finalized.hash) {
            Some(index) => {
                for _ in 0..index {
                    self.blocks.pop_front();
                }
                false
            }
            None => {
                // The finalized block is ahead of the head, or the held chain is not canonical.
                let held = std::mem::replace(&mut self.blocks, NonEmptyVecDeque::new(finalized));
                self.detached.extend(held.into_iter().map(|block| (block.hash, block)));
                true
            }
        };
        let finalized = finalized.number;
        self.detached.retain(|_, block| block.number >= finalized);
        restarted
    }

    /// Keeps the blocks up to `index` and detaches the rest, returning how many it detached.
    fn detach_after(&mut self, index: usize) -> usize {
        let detached = self.blocks.split_after(index);
        let count = detached.len();
        self.detached.extend(detached.into_iter().map(|block| (block.hash, block)));
        count
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::l1::test_utils::{MAX, block, chain};

    /// Returns a view of `blocks`, oldest first; the oldest is finalized.
    fn view_of(blocks: Vec<BlockInfo>) -> View {
        let mut view = View::new(blocks[0]);
        moves_to(&mut view, &blocks[1..], 0);
        view
    }

    /// Learns `blocks`, then moves the view's head to the last of them, reorging out `orphaned`.
    fn moves_to(view: &mut View, blocks: &[BlockInfo], orphaned: usize) {
        for &block in blocks {
            view.insert(block);
        }
        if let Some(&head) = blocks.last() {
            assert_eq!(view.extend(head), Extension::Moved { orphaned });
        }
    }

    fn held(view: &View) -> Vec<BlockInfo> {
        view.blocks.iter().copied().collect()
    }

    #[test]
    fn advances_from_a_canonical_cursor() {
        let view = view_of(chain(1..=5, 1));
        assert_eq!(
            view.updates_since(block(2, 1, 1), MAX),
            Some(Update::Advanced(chain(3..=5, 1)))
        );
        assert_eq!(
            view.updates_since(block(2, 1, 1), NonZeroUsize::MIN),
            Some(Update::Advanced(vec![block(3, 1, 1)]))
        );
        // Nothing after the head.
        assert_eq!(view.updates_since(block(5, 1, 1), MAX), None);
    }

    #[test]
    fn traces_a_reorged_cursor_to_the_common_ancestor() {
        let mut view = view_of(chain(1..=5, 1));
        // Blocks 4 and 5 are replaced.
        moves_to(&mut view, &[block(4, 2, 1), block(5, 2, 2), block(6, 2, 2)], 2);
        assert_eq!(
            view.updates_since(block(5, 1, 1), MAX),
            Some(Update::Reorged {
                ancestor: block(3, 1, 1),
                blocks: vec![block(4, 2, 1), block(5, 2, 2), block(6, 2, 2)],
            })
        );
        // A cursor before the fork only advances.
        assert_eq!(
            view.updates_since(block(3, 1, 1), MAX),
            Some(Update::Advanced(vec![block(4, 2, 1), block(5, 2, 2), block(6, 2, 2)]))
        );

        // A second reorg replaces block 5 again. A cursor on the first fork still finds its
        // ancestor through both.
        moves_to(&mut view, &[block(5, 3, 2)], 2);
        assert_eq!(
            view.updates_since(block(5, 1, 1), MAX),
            Some(Update::Reorged {
                ancestor: block(3, 1, 1),
                blocks: vec![block(4, 2, 1), block(5, 3, 2)]
            })
        );
        assert_eq!(
            view.updates_since(block(6, 2, 2), MAX),
            Some(Update::Reorged { ancestor: block(4, 2, 1), blocks: vec![block(5, 3, 2)] })
        );
    }

    #[test]
    fn a_head_that_moves_back_reorgs_cursors_past_it() {
        let mut view = view_of(chain(1..=5, 1));
        assert_eq!(view.extend(block(3, 1, 1)), Extension::Moved { orphaned: 2 });
        assert_eq!(view.head(), block(3, 1, 1));
        assert_eq!(
            view.updates_since(block(5, 1, 1), MAX),
            Some(Update::Reorged { ancestor: block(3, 1, 1), blocks: vec![] })
        );
        assert_eq!(view.updates_since(block(3, 1, 1), MAX), None);
    }

    #[test]
    fn needs_the_ancestors_it_does_not_know() {
        let mut view = view_of(chain(1..=3, 1));
        assert_eq!(view.extend(block(6, 1, 1)), Extension::NeedsParent(block(5, 1, 1).hash));
        view.insert(block(5, 1, 1));
        assert_eq!(view.extend(block(6, 1, 1)), Extension::NeedsParent(block(4, 1, 1).hash));
        assert_eq!(held(&view), chain(1..=3, 1));
        view.insert(block(4, 1, 1));
        assert_eq!(view.extend(block(6, 1, 1)), Extension::Moved { orphaned: 0 });
        assert_eq!(held(&view), chain(1..=6, 1));
    }

    #[test]
    fn a_head_returning_to_a_reorged_out_chain_needs_no_blocks() {
        let mut view = view_of(chain(1..=5, 1));
        moves_to(&mut view, &[block(4, 2, 1)], 2);
        // A child of block 5 links through the reorged-out blocks 4 and 5.
        assert_eq!(view.extend(block(6, 1, 1)), Extension::Moved { orphaned: 1 });
        assert_eq!(held(&view), chain(1..=6, 1));
    }

    #[test]
    fn rejects_heads_that_do_not_descend_from_the_finalized_block() {
        let mut view = view_of(chain(3..=5, 1));
        // A different block at the finalized height.
        assert_eq!(view.extend(block(3, 9, 9)), Extension::Rejected);
        // A head whose ancestors reach the finalized height at a different block.
        view.insert(block(5, 9, 9));
        assert_eq!(view.extend(block(6, 9, 9)), Extension::NeedsParent(block(4, 9, 9).hash));
        view.insert(block(4, 9, 9));
        assert_eq!(view.extend(block(6, 9, 9)), Extension::Rejected);
        // A head behind the finalized block.
        assert_eq!(view.extend(block(2, 1, 1)), Extension::Rejected);
        assert_eq!(held(&view), chain(3..=5, 1));
    }

    #[test]
    fn behind_and_unknown_cursors() {
        let view = view_of(chain(3..=5, 1));
        assert_eq!(
            view.updates_since(block(2, 1, 1), MAX),
            Some(Update::Behind { base: block(3, 1, 1) })
        );
        assert_eq!(
            view.updates_since(BlockInfo::default(), MAX),
            Some(Update::Behind { base: block(3, 1, 1) })
        );
        assert_eq!(view.updates_since(block(4, 9, 9), MAX), Some(Update::Unknown));
    }

    #[test]
    fn finalizing_a_held_block_drops_older_blocks() {
        let mut view = view_of(chain(1..=5, 1));
        moves_to(&mut view, &[block(3, 2, 1), block(4, 2, 2), block(5, 2, 2)], 3);
        assert!(!view.finalize(block(4, 2, 2)));
        assert_eq!(held(&view), vec![block(4, 2, 2), block(5, 2, 2)]);
        assert!(view.detached.values().all(|block| block.number >= 4));
        assert_eq!(
            view.updates_since(block(3, 1, 1), MAX),
            Some(Update::Behind { base: block(4, 2, 2) })
        );
        // Blocks no newer than the finalized block are not kept.
        view.insert(block(4, 7, 7));
        assert!(!view.detached.contains_key(&block(4, 7, 7).hash));
        // An older finalized block changes nothing.
        assert!(!view.finalize(block(2, 1, 1)));
        assert_eq!(view.finalized(), block(4, 2, 2));
    }

    #[test]
    fn finalizing_a_block_the_view_does_not_hold_restarts_from_it() {
        // Ahead of the head.
        let mut view = view_of(chain(1..=5, 1));
        assert!(view.finalize(block(8, 1, 1)));
        assert_eq!(held(&view), vec![block(8, 1, 1)]);
        assert_eq!(view.head(), block(8, 1, 1));
        assert!(view.detached.is_empty());

        // Conflicting with the held chain: cursors on the held fork are unknown.
        let mut view = view_of(chain(1..=5, 1));
        assert!(view.finalize(block(4, 2, 2)));
        assert_eq!(held(&view), vec![block(4, 2, 2)]);
        assert_eq!(view.updates_since(block(5, 1, 1), MAX), Some(Update::Unknown));
    }
}

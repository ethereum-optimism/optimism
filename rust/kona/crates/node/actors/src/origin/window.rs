//! The L1 blocks the sequencer may build on next.

use alloy_eips::BlockNumHash;
use kona_protocol::BlockInfo;
use std::collections::VecDeque;

/// The L1 blocks the sequencer may build on next.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(super) struct Window {
    /// Consecutive L1 blocks, oldest first, each the parent of the next: the L1 origin of the
    /// engine's unsafe head, then the canonical, confirmed L1 blocks after it.
    pub(super) blocks: VecDeque<BlockInfo>,
    /// The L1 origin of the engine's unsafe head, if L1 reorged it out.
    pub(super) reorged: Option<BlockNumHash>,
}

/// What the window holds for an L1 origin.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(super) enum Origins {
    /// The origin, and the block after it if the window holds one.
    Held {
        /// The origin.
        current: BlockInfo,
        /// The canonical, confirmed child of the origin.
        next: Option<BlockInfo>,
    },
    /// L1 reorged out the origin.
    Reorged,
}

impl Window {
    /// Returns what the window holds for `origin`, or `None` if it holds nothing for it yet.
    pub(super) fn origins(&self, origin: BlockNumHash) -> Option<Origins> {
        if self.reorged == Some(origin) {
            return Some(Origins::Reorged);
        }
        let first = self.blocks.front()?;
        let index = usize::try_from(origin.number.checked_sub(first.number)?).ok()?;
        let current = *self.blocks.get(index)?;
        if current.hash != origin.hash {
            // The window holds a different block at the origin's height.
            return Some(Origins::Reorged);
        }
        Some(Origins::Held { current, next: self.blocks.get(index + 1).copied() })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::l1::test_utils::{block, chain};

    #[test]
    fn holds_origins_and_their_next_blocks() {
        let window = Window { blocks: chain(3..=5, 1).into(), reorged: None };
        assert_eq!(
            window.origins(block(3, 1, 1).id()),
            Some(Origins::Held { current: block(3, 1, 1), next: Some(block(4, 1, 1)) })
        );
        assert_eq!(
            window.origins(block(5, 1, 1).id()),
            Some(Origins::Held { current: block(5, 1, 1), next: None })
        );
        // Another block at a held height.
        assert_eq!(window.origins(block(4, 2, 1).id()), Some(Origins::Reorged));
        // Heights the window does not hold.
        assert_eq!(window.origins(block(2, 1, 1).id()), None);
        assert_eq!(window.origins(block(6, 1, 1).id()), None);
        assert_eq!(Window::default().origins(block(3, 1, 1).id()), None);
    }

    #[test]
    fn reports_a_reorged_out_origin() {
        let window = Window { blocks: VecDeque::new(), reorged: Some(block(3, 1, 1).id()) };
        assert_eq!(window.origins(block(3, 1, 1).id()), Some(Origins::Reorged));
        assert_eq!(window.origins(block(4, 1, 1).id()), None);
    }
}

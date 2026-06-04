//! In-memory cache data structures for the runtime cross-unsafe head.

use alloy_primitives::{B256, U64};
use std::collections::BTreeMap;

use super::CrossUnsafeHead;

/// A source-chain block that a cached cross-unsafe block depends on.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(super) struct SourceRef {
    pub(super) chain_id: u64,
    pub(super) block_number: u64,
    pub(super) block_hash: B256,
}

/// In-memory cache of locally validated cross-unsafe blocks.
///
/// `safe` is the floor anchor; `validated` holds the contiguous chain of validated blocks strictly
/// above it. The head is always the highest validated block, or `safe` when none are cached.
#[derive(Debug, Default)]
pub(super) struct CrossUnsafeState {
    safe: CachedBlock,
    pub(super) validated: BTreeMap<u64, CachedBlock>,
}

impl CrossUnsafeState {
    pub(super) fn head(&self) -> &CachedBlock {
        self.validated.values().next_back().unwrap_or(&self.safe)
    }

    /// Re-anchors to the current safe head, dropping any cached block at or below it.
    pub(super) fn reseed_safe(&mut self, safe: CachedBlock) {
        self.validated = self.validated.split_off(&safe.number.saturating_add(1));
        self.safe = safe;
    }

    pub(super) fn is_validated(&self, number: u64, hash: B256, parent_hash: B256) -> bool {
        self.validated
            .get(&number)
            .is_some_and(|block| block.hash == hash && block.parent_hash == parent_hash)
    }

    pub(super) fn insert(&mut self, block: CachedBlock) {
        self.validated.insert(block.number, block);
    }

    /// Drops every cached block at or above `number`.
    pub(super) fn truncate_from(&mut self, number: u64) {
        self.validated.split_off(&number);
    }
}

#[derive(Debug, Clone, Default)]
pub(super) struct CachedBlock {
    pub(super) number: u64,
    pub(super) hash: B256,
    pub(super) parent_hash: B256,
    /// Distinct source blocks the executing messages in this block depend on.
    pub(super) sources: Vec<SourceRef>,
}

impl From<&CachedBlock> for CrossUnsafeHead {
    fn from(value: &CachedBlock) -> Self {
        Self { number: U64::from(value.number), hash: value.hash }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn block(number: u64, tag: u8, parent: u8) -> CachedBlock {
        CachedBlock {
            number,
            hash: B256::repeat_byte(tag),
            parent_hash: B256::repeat_byte(parent),
            sources: Vec::new(),
        }
    }

    #[test]
    fn head_defaults_to_safe_anchor() {
        let mut state = CrossUnsafeState::default();
        state.reseed_safe(block(5, 5, 0));
        assert_eq!(state.head().number, 5);
        assert_eq!(state.head().hash, B256::repeat_byte(5));
    }

    #[test]
    fn insert_advances_head_and_truncate_rewinds_to_safe() {
        let mut state = CrossUnsafeState::default();
        state.reseed_safe(block(5, 5, 0));
        state.insert(block(6, 6, 5));
        state.insert(block(7, 7, 6));
        assert_eq!(state.head().number, 7);

        state.truncate_from(7);
        assert_eq!(state.head().number, 6);

        state.truncate_from(6);
        assert_eq!(state.head().number, 5);
        assert_eq!(state.head().hash, B256::repeat_byte(5));
    }

    #[test]
    fn reseed_safe_drops_entries_at_or_below_safe() {
        let mut state = CrossUnsafeState::default();
        state.reseed_safe(block(5, 5, 0));
        state.insert(block(6, 6, 5));
        state.insert(block(7, 7, 6));

        state.reseed_safe(block(6, 60, 0));
        assert!(!state.validated.contains_key(&6));
        assert_eq!(state.safe.number, 6);
        assert_eq!(state.head().number, 7);

        // Re-anchoring above the cached head leaves only the safe anchor.
        state.reseed_safe(block(9, 90, 0));
        assert!(state.validated.is_empty());
        assert_eq!(state.head().number, 9);
    }

    #[test]
    fn is_validated_requires_hash_and_parent_match() {
        let mut state = CrossUnsafeState::default();
        state.reseed_safe(block(5, 5, 0));
        state.insert(block(6, 6, 5));

        assert!(state.is_validated(6, B256::repeat_byte(6), B256::repeat_byte(5)));
        assert!(!state.is_validated(6, B256::repeat_byte(99), B256::repeat_byte(5)));
        assert!(!state.is_validated(6, B256::repeat_byte(6), B256::repeat_byte(99)));
        assert!(!state.is_validated(8, B256::repeat_byte(8), B256::repeat_byte(7)));
    }
}

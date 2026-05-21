//! State management for the live trie collector.

#[cfg(feature = "metrics")]
use super::metrics::BufferMetrics;
use super::overlay::MemoryOverlayOpProofsStateProviderRef;
use crate::{BlockStateDiff, OpProofsProviderRO, provider::OpProofsStateProviderRef};
use alloy_eips::{NumHash, eip1898::BlockWithParent};
use alloy_primitives::{B256, map::HashMap};
use parking_lot::RwLock;
use std::{collections::BTreeMap, sync::Arc};

/// Buffer for holding blocks waiting to be persisted.
///
/// This acts as the in-memory "tip" of the chain for the trie calculator.
#[derive(Debug)]
struct TrieBuffer {
    /// All blocks that are not on disk yet.
    blocks: RwLock<HashMap<B256, Arc<(BlockWithParent, BlockStateDiff)>>>,
    /// Mapping of block numbers to block hashes.
    numbers: RwLock<BTreeMap<u64, B256>>,
    #[cfg(feature = "metrics")]
    metrics: BufferMetrics,
}

impl TrieBuffer {
    /// Create a new empty in-memory state.
    fn new() -> Self {
        Self {
            blocks: RwLock::new(HashMap::default()),
            numbers: RwLock::new(BTreeMap::new()),
            #[cfg(feature = "metrics")]
            metrics: BufferMetrics::new_with_labels(&[] as &[(&str, &str)]),
        }
    }

    /// Insert a block into the buffer.
    fn insert(&self, block: BlockWithParent, diff: BlockStateDiff) {
        let hash = block.block.hash;
        let number = block.block.number;
        let state = Arc::new((block, diff));

        // Write locks
        let mut blocks = self.blocks.write();
        let mut numbers = self.numbers.write();

        blocks.insert(hash, state);
        numbers.insert(number, hash);

        #[cfg(feature = "metrics")]
        self.metrics.buffer_size.set(blocks.len() as f64);
    }

    /// Returns the number of buffered blocks.
    fn len(&self) -> usize {
        self.blocks.read().len()
    }

    /// Prunes blocks from the buffer that are strictly before the given block number.
    fn prune(&self, number: u64) {
        let mut blocks = self.blocks.write();
        let mut numbers = self.numbers.write();

        // Split off the entries we want to keep (number >= threshold).
        // The original map now contains only entries to prune (num < number).
        let to_keep = numbers.split_off(&number);
        for hash in numbers.values() {
            blocks.remove(hash);
        }
        *numbers = to_keep;

        #[cfg(feature = "metrics")]
        self.metrics.buffer_size.set(blocks.len() as f64);
    }

    /// Removes blocks starting from `from` (inclusive) through the tip.
    ///
    /// Mirrors the disk `unwind_history(to)` semantics where `to.block.number` is the
    /// first block removed. After this call, only blocks with number < `from` remain.
    fn unwind(&self, from: u64) {
        let mut blocks = self.blocks.write();
        let mut numbers = self.numbers.write();

        // Split off all entries with num >= from; those are the ones to remove.
        let to_remove = numbers.split_off(&from);
        for hash in to_remove.values() {
            blocks.remove(hash);
        }

        #[cfg(feature = "metrics")]
        self.metrics.buffer_size.set(blocks.len() as f64);
    }
}

/// Manager for the in-memory state of the live trie.
#[derive(Debug, Clone)]
pub(crate) struct TrieBufferState {
    inner: Arc<TrieBuffer>,
}

impl Default for TrieBufferState {
    fn default() -> Self {
        Self::new()
    }
}

impl TrieBufferState {
    /// Create a new live trie state manager.
    pub(crate) fn new() -> Self {
        Self { inner: Arc::new(TrieBuffer::new()) }
    }

    /// Insert a block into the buffer.
    pub(in crate::engine) fn insert(&self, block: BlockWithParent, diff: BlockStateDiff) {
        self.inner.insert(block, diff);
    }

    /// Returns the number of buffered blocks.
    pub(in crate::engine) fn len(&self) -> usize {
        self.inner.len()
    }

    /// Returns `true` if the buffer contains no blocks.
    pub(in crate::engine) fn is_empty(&self) -> bool {
        self.inner.len() == 0
    }

    /// Prunes blocks from the buffer that are strictly before the given block number.
    pub(in crate::engine) fn prune(&self, number: u64) {
        self.inner.prune(number);
    }

    /// Removes blocks starting from `from` (inclusive) through the tip.
    pub(in crate::engine) fn unwind(&self, from: u64) {
        self.inner.unwind(from);
    }

    /// Returns the highest buffered block as a [`NumHash`], or `None` if the buffer is empty.
    pub(in crate::engine) fn tip(&self) -> Option<NumHash> {
        let numbers = self.inner.numbers.read();
        numbers.iter().next_back().map(|(&num, &hash)| NumHash::new(num, hash))
    }

    /// Returns all buffered blocks ordered oldest to newest.
    pub(in crate::engine) fn blocks_ordered(&self) -> Vec<Arc<(BlockWithParent, BlockStateDiff)>> {
        let numbers = self.inner.numbers.read();
        let blocks = self.inner.blocks.read();
        let mut out = Vec::with_capacity(numbers.len());
        for hash in numbers.values() {
            let state = blocks.get(hash).unwrap_or_else(|| {
                panic!("trie buffer invariant violated: missing block state for hash referenced in number index: {hash:?}")
            });
            out.push(Arc::clone(state));
        }
        out
    }

    /// Return state provider with reference to in-memory blocks that overlay storage state.
    ///
    /// This retrieves the chain of blocks ending at `hash` from the in-memory buffer,
    /// providing a view that includes both the buffered changes and the underlying disk state.
    pub(crate) fn state_provider<'a, P>(
        &self,
        hash: B256,
        inner: OpProofsStateProviderRef<'a, P>,
    ) -> MemoryOverlayOpProofsStateProviderRef<'a, P>
    where
        P: OpProofsProviderRO + Clone,
    {
        let mut in_memory = Vec::new();
        let blocks = self.inner.blocks.read();

        // Trace back from the requested hash to finding no parent in memory
        let mut current_hash = hash;
        while let Some(state) = blocks.get(&current_hash) {
            in_memory.push(state.clone());
            current_hash = state.0.parent;
        }

        // The vector is currently Newest -> Oldest. Reverse it to Oldest -> Newest
        // as expected by the overlay provider for correct replay.
        in_memory.reverse();

        MemoryOverlayOpProofsStateProviderRef::new(inner, in_memory)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::BlockStateDiff;
    use alloy_eips::NumHash;
    use alloy_primitives::B256;

    /// Build a `(BlockWithParent, BlockStateDiff)` pair with predictable hashes.
    ///
    /// `hash_byte` uniquely identifies the block; `parent_byte` identifies its parent.
    fn make_block(
        number: u64,
        hash_byte: u8,
        parent_byte: u8,
    ) -> (BlockWithParent, BlockStateDiff) {
        let block = BlockWithParent::new(
            B256::repeat_byte(parent_byte),
            NumHash::new(number, B256::repeat_byte(hash_byte)),
        );
        (block, BlockStateDiff::default())
    }

    fn insert(state: &TrieBufferState, number: u64, hash_byte: u8, parent_byte: u8) {
        let (block, diff) = make_block(number, hash_byte, parent_byte);
        state.insert(block, diff);
    }

    #[test]
    fn empty_state() {
        let state = TrieBufferState::new();
        assert_eq!(state.len(), 0);
        assert!(state.tip().is_none());
    }

    #[test]
    fn tip_tracks_highest_block() {
        let state = TrieBufferState::new();
        insert(&state, 3, 0x03, 0x02);
        insert(&state, 1, 0x01, 0x00);
        insert(&state, 2, 0x02, 0x01);

        let tip = state.tip().unwrap();
        assert_eq!(tip.number, 3);
        assert_eq!(tip.hash, B256::repeat_byte(0x03));
        assert_eq!(state.len(), 3);
    }

    #[test]
    fn prune_removes_blocks_strictly_before_number() {
        let state = TrieBufferState::new();
        for i in 1..=5u64 {
            insert(&state, i, i as u8, (i - 1) as u8);
        }

        state.prune(3); // removes 1, 2 — keeps 3, 4, 5

        assert_eq!(state.len(), 3);
        assert_eq!(state.tip().unwrap().number, 5);
        // blocks_ordered should start at 3
        let ordered = state.blocks_ordered();
        assert_eq!(ordered[0].0.block.number, 3);
    }

    #[test]
    fn prune_keeps_block_at_boundary() {
        let state = TrieBufferState::new();
        for i in 1..=3u64 {
            insert(&state, i, i as u8, (i - 1) as u8);
        }

        state.prune(2); // removes only block 1 — keeps 2, 3
        assert_eq!(state.len(), 2);
        assert_eq!(state.blocks_ordered()[0].0.block.number, 2);
    }

    #[test]
    fn prune_all_blocks() {
        let state = TrieBufferState::new();
        for i in 1..=3u64 {
            insert(&state, i, i as u8, (i - 1) as u8);
        }

        state.prune(4); // strictly before 4 → removes all
        assert_eq!(state.len(), 0);
        assert!(state.tip().is_none());
    }

    #[test]
    fn prune_noop_when_number_at_or_below_oldest() {
        let state = TrieBufferState::new();
        for i in 5..=8u64 {
            insert(&state, i, i as u8, (i - 1) as u8);
        }

        state.prune(5); // nothing is strictly before 5
        assert_eq!(state.len(), 4);
    }

    #[test]
    fn unwind_removes_from_number_inclusive_to_tip() {
        let state = TrieBufferState::new();
        for i in 1..=5u64 {
            insert(&state, i, i as u8, (i - 1) as u8);
        }

        state.unwind(3); // removes 3, 4, 5 — keeps 1, 2

        assert_eq!(state.len(), 2);
        assert_eq!(state.tip().unwrap().number, 2);
    }

    #[test]
    fn unwind_all_blocks() {
        let state = TrieBufferState::new();
        for i in 1..=3u64 {
            insert(&state, i, i as u8, (i - 1) as u8);
        }

        state.unwind(1); // removes everything
        assert_eq!(state.len(), 0);
        assert!(state.tip().is_none());
    }

    #[test]
    fn unwind_noop_when_from_above_tip() {
        let state = TrieBufferState::new();
        for i in 1..=3u64 {
            insert(&state, i, i as u8, (i - 1) as u8);
        }

        state.unwind(10); // nothing to remove
        assert_eq!(state.len(), 3);
    }

    #[test]
    fn blocks_ordered_returns_oldest_to_newest() {
        let state = TrieBufferState::new();
        // Insert in non-sequential order
        insert(&state, 3, 0x03, 0x02);
        insert(&state, 1, 0x01, 0x00);
        insert(&state, 2, 0x02, 0x01);

        let ordered = state.blocks_ordered();
        assert_eq!(ordered.len(), 3);
        assert_eq!(ordered[0].0.block.number, 1);
        assert_eq!(ordered[1].0.block.number, 2);
        assert_eq!(ordered[2].0.block.number, 3);
    }

    #[test]
    fn blocks_ordered_empty() {
        let state = TrieBufferState::new();
        assert!(state.blocks_ordered().is_empty());
    }
}

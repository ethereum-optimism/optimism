//! Pending block state for speculative flashblock building.
//!
//! This module provides types for tracking execution state from flashblock builds,
//! enabling speculative building of subsequent blocks before their parent canonical
//! block arrives via P2P.

use alloy_primitives::B256;
use reth_execution_types::BlockExecutionOutput;
use reth_primitives_traits::{HeaderTy, NodePrimitives, SealedHeader};
use reth_revm::cached::CachedReads;
use std::{
    collections::{HashMap, VecDeque},
    sync::Arc,
};

/// Tracks the execution state from building a pending block.
///
/// This is used to enable speculative building of subsequent blocks:
/// - When flashblocks for block N+1 arrive before canonical block N
/// - The pending state from building block N's flashblocks can be used
/// - This allows continuous flashblock processing without waiting for P2P
#[derive(Debug, Clone)]
pub struct PendingBlockState<N: NodePrimitives> {
    /// Locally computed block hash for this built block.
    ///
    /// This hash is used to match subsequent flashblock sequences by `parent_hash`
    /// during speculative chaining.
    pub block_hash: B256,
    /// Block number that was built.
    pub block_number: u64,
    /// Parent hash of the built block (may be non-canonical for speculative builds).
    pub parent_hash: B256,
    /// Canonical anchor hash for state lookups.
    ///
    /// This is the hash used for `history_by_block_hash` when loading state.
    /// For canonical builds, this equals `parent_hash`.
    /// For speculative builds, this is the canonical block hash that the chain
    /// of speculative builds is rooted at (forwarded from parent's anchor).
    pub canonical_anchor_hash: B256,
    /// Execution outcome containing state changes.
    pub execution_outcome: Arc<BlockExecutionOutput<N::Receipt>>,
    /// Cached reads from execution for reuse.
    pub cached_reads: CachedReads,
    /// Sealed header for this built block.
    ///
    /// Used as the parent header for speculative child builds.
    pub sealed_header: Option<SealedHeader<HeaderTy<N>>>,
}

impl<N: NodePrimitives> PendingBlockState<N> {
    /// Creates a new pending block state.
    pub const fn new(
        block_hash: B256,
        block_number: u64,
        parent_hash: B256,
        canonical_anchor_hash: B256,
        execution_outcome: Arc<BlockExecutionOutput<N::Receipt>>,
        cached_reads: CachedReads,
    ) -> Self {
        Self {
            block_hash,
            block_number,
            parent_hash,
            canonical_anchor_hash,
            execution_outcome,
            cached_reads,
            sealed_header: None,
        }
    }

    /// Attaches a sealed header for use as parent context in speculative builds.
    pub fn with_sealed_header(mut self, sealed_header: SealedHeader<HeaderTy<N>>) -> Self {
        self.sealed_header = Some(sealed_header);
        self
    }
}

/// Registry of pending block states for speculative building.
///
/// Maintains a small cache of recently built pending blocks, allowing
/// subsequent flashblock sequences to build on top of them even before
/// the canonical blocks arrive.
#[derive(Debug)]
pub struct PendingStateRegistry<N: NodePrimitives> {
    /// Executed pending states keyed by locally computed block hash.
    by_block_hash: HashMap<B256, PendingBlockState<N>>,
    /// Insertion order for bounded eviction.
    insertion_order: VecDeque<B256>,
    /// Most recently recorded block hash.
    latest_block_hash: Option<B256>,
    /// Maximum number of tracked pending states.
    max_entries: usize,
}

impl<N: NodePrimitives> PendingStateRegistry<N> {
    const DEFAULT_MAX_ENTRIES: usize = 64;

    /// Creates a new pending state registry.
    pub fn new() -> Self {
        Self::with_max_entries(Self::DEFAULT_MAX_ENTRIES)
    }

    /// Creates a new pending state registry with an explicit entry bound.
    pub fn with_max_entries(max_entries: usize) -> Self {
        let max_entries = max_entries.max(1);
        Self {
            by_block_hash: HashMap::with_capacity(max_entries),
            insertion_order: VecDeque::with_capacity(max_entries),
            latest_block_hash: None,
            max_entries,
        }
    }

    /// Records a completed build's state for potential use by subsequent builds.
    pub fn record_build(&mut self, state: PendingBlockState<N>) {
        let block_hash = state.block_hash;

        if self.by_block_hash.contains_key(&block_hash) {
            self.insertion_order.retain(|hash| *hash != block_hash);
        }

        self.by_block_hash.insert(block_hash, state);
        self.insertion_order.push_back(block_hash);
        self.latest_block_hash = Some(block_hash);

        while self.by_block_hash.len() > self.max_entries {
            let Some(evicted_hash) = self.insertion_order.pop_front() else {
                break;
            };
            self.by_block_hash.remove(&evicted_hash);
            if self.latest_block_hash == Some(evicted_hash) {
                self.latest_block_hash = self.insertion_order.back().copied();
            }
        }
    }

    /// Gets the pending state for a given parent hash, if available.
    ///
    /// Returns `Some` if we have pending state whose `block_hash` matches the requested
    /// `parent_hash`.
    pub fn get_state_for_parent(&self, parent_hash: B256) -> Option<&PendingBlockState<N>> {
        self.by_block_hash.get(&parent_hash)
    }

    /// Clears all pending state.
    pub fn clear(&mut self) {
        self.by_block_hash.clear();
        self.insertion_order.clear();
        self.latest_block_hash = None;
    }

    /// Returns the current pending state, if any.
    pub fn current(&self) -> Option<&PendingBlockState<N>> {
        self.latest_block_hash.and_then(|hash| self.by_block_hash.get(&hash))
    }
}

impl<N: NodePrimitives> Default for PendingStateRegistry<N> {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use reth_optimism_primitives::OpPrimitives;

    type TestRegistry = PendingStateRegistry<OpPrimitives>;

    #[test]
    fn test_registry_returns_state_for_matching_parent() {
        let mut registry = TestRegistry::new();

        let block_hash = B256::repeat_byte(1);
        let parent_hash = B256::repeat_byte(0);
        let state = PendingBlockState {
            block_hash,
            block_number: 100,
            parent_hash,
            canonical_anchor_hash: parent_hash,
            execution_outcome: Arc::new(BlockExecutionOutput::default()),
            cached_reads: CachedReads::default(),
            sealed_header: None,
        };
        registry.record_build(state);

        // Should find state when querying with matching block_hash as parent
        let result = registry.get_state_for_parent(block_hash);
        assert!(result.is_some());
        assert_eq!(result.unwrap().block_number, 100);
    }

    #[test]
    fn test_registry_returns_none_for_wrong_parent() {
        let mut registry = TestRegistry::new();

        let parent_hash = B256::repeat_byte(0);
        let state = PendingBlockState {
            block_hash: B256::repeat_byte(1),
            block_number: 100,
            parent_hash,
            canonical_anchor_hash: parent_hash,
            execution_outcome: Arc::new(BlockExecutionOutput::default()),
            cached_reads: CachedReads::default(),
            sealed_header: None,
        };
        registry.record_build(state);

        // Different parent hash should return None
        assert!(registry.get_state_for_parent(B256::repeat_byte(2)).is_none());
    }

    #[test]
    fn test_registry_clear() {
        let mut registry = TestRegistry::new();

        let parent_hash = B256::repeat_byte(0);
        let state = PendingBlockState {
            block_hash: B256::repeat_byte(1),
            block_number: 100,
            parent_hash,
            canonical_anchor_hash: parent_hash,
            execution_outcome: Arc::new(BlockExecutionOutput::default()),
            cached_reads: CachedReads::default(),
            sealed_header: None,
        };
        registry.record_build(state);
        assert!(registry.current().is_some());

        registry.clear();
        assert!(registry.current().is_none());
    }

    #[test]
    fn test_registry_tracks_multiple_states_by_hash() {
        let mut registry = TestRegistry::new();

        let anchor = B256::repeat_byte(0);
        let state_100 = PendingBlockState {
            block_hash: B256::repeat_byte(1),
            block_number: 100,
            parent_hash: anchor,
            canonical_anchor_hash: anchor,
            execution_outcome: Arc::new(BlockExecutionOutput::default()),
            cached_reads: CachedReads::default(),
            sealed_header: None,
        };
        let state_101 = PendingBlockState {
            block_hash: B256::repeat_byte(2),
            block_number: 101,
            parent_hash: state_100.block_hash,
            canonical_anchor_hash: anchor,
            execution_outcome: Arc::new(BlockExecutionOutput::default()),
            cached_reads: CachedReads::default(),
            sealed_header: None,
        };

        registry.record_build(state_100.clone());
        registry.record_build(state_101.clone());

        assert_eq!(registry.current().map(|s| s.block_number), Some(101));
        assert_eq!(
            registry.get_state_for_parent(state_100.block_hash).map(|s| s.block_number),
            Some(100)
        );
        assert_eq!(
            registry.get_state_for_parent(state_101.block_hash).map(|s| s.block_number),
            Some(101)
        );
    }

    #[test]
    fn test_registry_eviction_respects_max_entries() {
        let mut registry = PendingStateRegistry::<OpPrimitives>::with_max_entries(2);
        let anchor = B256::repeat_byte(0);

        let state_100 = PendingBlockState {
            block_hash: B256::repeat_byte(1),
            block_number: 100,
            parent_hash: anchor,
            canonical_anchor_hash: anchor,
            execution_outcome: Arc::new(BlockExecutionOutput::default()),
            cached_reads: CachedReads::default(),
            sealed_header: None,
        };
        let state_101 = PendingBlockState {
            block_hash: B256::repeat_byte(2),
            block_number: 101,
            parent_hash: state_100.block_hash,
            canonical_anchor_hash: anchor,
            execution_outcome: Arc::new(BlockExecutionOutput::default()),
            cached_reads: CachedReads::default(),
            sealed_header: None,
        };
        let state_102 = PendingBlockState {
            block_hash: B256::repeat_byte(3),
            block_number: 102,
            parent_hash: state_101.block_hash,
            canonical_anchor_hash: anchor,
            execution_outcome: Arc::new(BlockExecutionOutput::default()),
            cached_reads: CachedReads::default(),
            sealed_header: None,
        };

        registry.record_build(state_100);
        registry.record_build(state_101.clone());
        registry.record_build(state_102.clone());

        assert!(registry.get_state_for_parent(B256::repeat_byte(1)).is_none());
        assert_eq!(
            registry.get_state_for_parent(state_101.block_hash).map(|s| s.block_number),
            Some(101)
        );
        assert_eq!(
            registry.get_state_for_parent(state_102.block_hash).map(|s| s.block_number),
            Some(102)
        );
        assert_eq!(registry.current().map(|s| s.block_number), Some(102));
    }

    /// Tests that `canonical_anchor_hash` is distinct from `parent_hash` in speculative chains.
    ///
    /// When building speculatively:
    /// - Block N (canonical): `parent_hash` = N-1, `canonical_anchor` = N-1 (same)
    /// - Block N+1 (speculative): `parent_hash` = N, `canonical_anchor` = N-1 (forwarded)
    /// - Block N+2 (speculative): `parent_hash` = N+1, `canonical_anchor` = N-1 (still forwarded)
    ///
    /// The `canonical_anchor_hash` always points to the last canonical block used for
    /// `history_by_block_hash` lookups.
    #[test]
    fn test_canonical_anchor_forwarding_semantics() {
        // Canonical block N-1 (the anchor for speculative chain)
        let canonical_anchor = B256::repeat_byte(0x00);

        // Block N built on canonical - anchor equals parent
        let block_n_hash = B256::repeat_byte(0x01);
        let state_n = PendingBlockState::<OpPrimitives> {
            block_hash: block_n_hash,
            block_number: 100,
            parent_hash: canonical_anchor,
            canonical_anchor_hash: canonical_anchor, // Same as parent for canonical build
            execution_outcome: Arc::new(BlockExecutionOutput::default()),
            cached_reads: CachedReads::default(),
            sealed_header: None,
        };

        // Verify block N's anchor is the canonical block
        assert_eq!(state_n.canonical_anchor_hash, canonical_anchor);
        assert_eq!(state_n.parent_hash, state_n.canonical_anchor_hash);

        // Block N+1 built speculatively on N - anchor is FORWARDED from N
        let block_n1_hash = B256::repeat_byte(0x02);
        let state_n1 = PendingBlockState::<OpPrimitives> {
            block_hash: block_n1_hash,
            block_number: 101,
            parent_hash: block_n_hash, // Parent is block N
            canonical_anchor_hash: state_n.canonical_anchor_hash, // Forwarded from N
            execution_outcome: Arc::new(BlockExecutionOutput::default()),
            cached_reads: CachedReads::default(),
            sealed_header: None,
        };

        // Verify N+1's anchor is still the canonical block, NOT block N
        assert_eq!(state_n1.canonical_anchor_hash, canonical_anchor);
        assert_ne!(state_n1.parent_hash, state_n1.canonical_anchor_hash);

        // Block N+2 built speculatively on N+1 - anchor still forwarded
        let block_n2_hash = B256::repeat_byte(0x03);
        let state_n2 = PendingBlockState::<OpPrimitives> {
            block_hash: block_n2_hash,
            block_number: 102,
            parent_hash: block_n1_hash, // Parent is block N+1
            canonical_anchor_hash: state_n1.canonical_anchor_hash, // Forwarded from N+1
            execution_outcome: Arc::new(BlockExecutionOutput::default()),
            cached_reads: CachedReads::default(),
            sealed_header: None,
        };

        // Verify N+2's anchor is STILL the original canonical block
        assert_eq!(state_n2.canonical_anchor_hash, canonical_anchor);
        assert_ne!(state_n2.parent_hash, state_n2.canonical_anchor_hash);

        // All three blocks should have the same canonical anchor
        assert_eq!(state_n.canonical_anchor_hash, state_n1.canonical_anchor_hash);
        assert_eq!(state_n1.canonical_anchor_hash, state_n2.canonical_anchor_hash);
    }
}

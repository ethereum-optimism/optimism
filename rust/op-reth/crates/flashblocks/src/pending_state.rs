//! Pending block state for speculative flashblock building.
//!
//! This module provides types for tracking execution state from flashblock builds,
//! enabling speculative building of subsequent blocks before their parent canonical
//! block arrives via P2P.

use alloy_primitives::B256;
use reth_execution_types::BlockExecutionOutput;
use reth_primitives_traits::NodePrimitives;
use reth_revm::cached::CachedReads;
use std::sync::Arc;

/// Tracks the execution state from building a pending block.
///
/// This is used to enable speculative building of subsequent blocks:
/// - When flashblocks for block N+1 arrive before canonical block N
/// - The pending state from building block N's flashblocks can be used
/// - This allows continuous flashblock processing without waiting for P2P
#[derive(Debug, Clone)]
pub struct PendingBlockState<N: NodePrimitives> {
    /// Hash of the block that was built (the pending block's hash).
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
        }
    }
}

/// Registry of pending block states for speculative building.
///
/// Maintains a small cache of recently built pending blocks, allowing
/// subsequent flashblock sequences to build on top of them even before
/// the canonical blocks arrive.
#[derive(Debug, Default)]
pub struct PendingStateRegistry<N: NodePrimitives> {
    /// Most recent pending block state (the one we'd build on top of).
    current: Option<PendingBlockState<N>>,
}

impl<N: NodePrimitives> PendingStateRegistry<N> {
    /// Creates a new pending state registry.
    pub const fn new() -> Self {
        Self { current: None }
    }

    /// Records a completed build's state for potential use by subsequent builds.
    pub fn record_build(&mut self, state: PendingBlockState<N>) {
        self.current = Some(state);
    }

    /// Gets the pending state for a given parent hash, if available.
    ///
    /// Returns `Some` if we have pending state whose `block_hash` matches the requested
    /// `parent_hash`.
    pub fn get_state_for_parent(&self, parent_hash: B256) -> Option<&PendingBlockState<N>> {
        self.current.as_ref().filter(|state| state.block_hash == parent_hash)
    }

    /// Clears all pending state.
    pub fn clear(&mut self) {
        self.current = None;
    }

    /// Returns the current pending state, if any.
    pub const fn current(&self) -> Option<&PendingBlockState<N>> {
        self.current.as_ref()
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
        };
        registry.record_build(state);
        assert!(registry.current().is_some());

        registry.clear();
        assert!(registry.current().is_none());
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
        };

        // Verify N+2's anchor is STILL the original canonical block
        assert_eq!(state_n2.canonical_anchor_hash, canonical_anchor);
        assert_ne!(state_n2.parent_hash, state_n2.canonical_anchor_hash);

        // All three blocks should have the same canonical anchor
        assert_eq!(state_n.canonical_anchor_hash, state_n1.canonical_anchor_hash);
        assert_eq!(state_n1.canonical_anchor_hash, state_n2.canonical_anchor_hash);
    }
}

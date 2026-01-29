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
    /// Parent hash of the built block.
    pub parent_hash: B256,
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
        execution_outcome: Arc<BlockExecutionOutput<N::Receipt>>,
        cached_reads: CachedReads,
    ) -> Self {
        Self { block_hash, block_number, parent_hash, execution_outcome, cached_reads }
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
        let state = PendingBlockState {
            block_hash,
            block_number: 100,
            parent_hash: B256::repeat_byte(0),
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

        let state = PendingBlockState {
            block_hash: B256::repeat_byte(1),
            block_number: 100,
            parent_hash: B256::repeat_byte(0),
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

        let state = PendingBlockState {
            block_hash: B256::repeat_byte(1),
            block_number: 100,
            parent_hash: B256::repeat_byte(0),
            execution_outcome: Arc::new(BlockExecutionOutput::default()),
            cached_reads: CachedReads::default(),
        };
        registry.record_build(state);
        assert!(registry.current().is_some());

        registry.clear();
        assert!(registry.current().is_none());
    }
}

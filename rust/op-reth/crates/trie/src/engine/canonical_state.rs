//! Access to reth's canonical in-memory state for providers the engine is generic over.

use reth_chain_state::CanonicalInMemoryState;
use reth_primitives_traits::NodePrimitives;
use reth_provider::{
    NodePrimitivesProvider,
    noop::NoopProvider,
    providers::{BlockchainProvider, ProviderNodeTypes},
};

/// A type that provides access to the [`CanonicalInMemoryState`].
///
/// This exposes the canonical blocks that are not yet on disk, for example to read the execution
/// outcome of a block whose state is not yet persisted.
pub trait CanonicalInMemoryStateProvider: NodePrimitivesProvider + Send + Sync {
    /// Returns a handle to the canonical in-memory state.
    fn canonical_in_memory_state(&self) -> CanonicalInMemoryState<Self::Primitives>;
}

impl<N: ProviderNodeTypes> CanonicalInMemoryStateProvider for BlockchainProvider<N> {
    fn canonical_in_memory_state(&self) -> CanonicalInMemoryState<Self::Primitives> {
        // Resolves to the inherent method, which clones the provider's shared handle.
        self.canonical_in_memory_state()
    }
}

impl<C: Send + Sync, N: NodePrimitives> CanonicalInMemoryStateProvider for NoopProvider<C, N> {
    fn canonical_in_memory_state(&self) -> CanonicalInMemoryState<N> {
        CanonicalInMemoryState::empty()
    }
}

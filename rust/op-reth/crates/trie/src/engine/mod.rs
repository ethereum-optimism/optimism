//! Live trie collector for external proofs storage.
//!
//! The collector runs as an **engine** on a dedicated background thread.  Callers
//! interact with it through [`EngineHandle`], a thin channel-based
//! handle whose methods mirror the old `LiveTrieCollector` API.
//!
//! Internally the engine owns *all* mutable state (memory buffer, persistence
//! handle, in-flight tracking) and processes engine action messages one at
//! a time, which structurally enforces the serial-call invariant.

mod buffer;
pub mod persistence;
mod tasks;

mod error;
pub use error::EngineError;

mod handle;
pub use handle::EngineHandle;

#[cfg(feature = "metrics")]
mod metrics;
mod runner;
mod state;

/// Default number of blocks to keep in memory before persisting.
const DEFAULT_PERSISTENCE_THRESHOLD: u64 = 5;

/// Default number of blocks where we block execution to allow persistence to catch up.
const DEFAULT_BACKPRESSURE_THRESHOLD: u64 = 10;

/// Default timeout for waiting on a persistence save/unwind operation (in seconds).
const DEFAULT_PERSISTENCE_TIMEOUT_SECS: u64 = 60;

/// Messages sent from [`EngineHandle`] to the engine thread.
enum EngineAction<Block: reth_primitives_traits::Block> {
    /// Execute a block via the EVM and index the resulting trie diff.
    ExecuteBlock(tasks::ExecuteBlockTask<Block>),
    /// Index pre-computed trie updates for a block (no EVM execution).
    IndexBlock(tasks::IndexBlockTask),
    /// Handle a reorg: unwind to the common ancestor then index the new chain.
    Reorg(tasks::ReorgTask),
    /// Unwind indexed data back to a given block.
    Unwind(tasks::UnwindTask),
    /// Block the caller until any in-flight persistence completes.
    #[cfg(test)]
    Flush(tasks::FlushTask),
    /// Update the sync catch-up target (fire-and-forget).
    SyncTo(tasks::SyncToTask),
}

impl<Block: reth_primitives_traits::Block> EngineAction<Block> {
    fn execute<Evm, Provider, Store>(self, state: &mut state::EngineState<Evm, Provider, Store>)
    where
        Evm: reth_evm::ConfigureEvm<
                Primitives: reth_primitives_traits::NodePrimitives<Block = Block>,
            >,
        Provider: reth_provider::BlockHashReader
            + reth_provider::StateReader
            + reth_provider::DatabaseProviderFactory
            + reth_provider::StateProviderFactory
            + reth_provider::BlockReader<Block = Block>
            + Clone
            + 'static,
        Store: crate::OpProofsStore + Clone + 'static,
    {
        match self {
            Self::ExecuteBlock(task) => task.execute(state),
            Self::IndexBlock(task) => task.execute(state),
            Self::Reorg(task) => task.execute(state),
            Self::Unwind(task) => task.execute(state),
            #[cfg(test)]
            Self::Flush(task) => task.execute(state),
            Self::SyncTo(task) => task.execute(state),
        }
    }
}

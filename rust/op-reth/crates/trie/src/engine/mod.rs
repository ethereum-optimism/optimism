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
pub use tasks::ReorgBlock;

mod error;
pub use error::EngineError;

mod handle;
pub use handle::EngineHandle;

#[cfg(feature = "metrics")]
mod metrics;
mod runner;
mod service_guard;
mod state;

use std::time::Duration;

/// Default number of blocks to keep in memory before persisting.
const DEFAULT_PERSISTENCE_THRESHOLD: u64 = 5;

/// Default number of blocks where we block execution to allow persistence to catch up.
const DEFAULT_BACKPRESSURE_THRESHOLD: u64 = 10;

/// Default timeout for waiting on a persistence save/unwind operation.
const DEFAULT_PERSISTENCE_TIMEOUT: Duration = Duration::from_secs(60);

/// How long the engine waits by default with a non-empty memory buffer before flushing it even
/// if the persistence threshold has not been reached.
///
/// Without this, a paused chain (e.g. fault-proof tests that freeze the sequencer) would leave
/// buffered blocks unpersisted indefinitely, breaking the proofs RPC's strict "is this block
/// persisted?" check.
const DEFAULT_IDLE_FLUSH_INTERVAL: Duration = Duration::from_secs(10);

/// How the engine paces its saves and how long it waits on its persistence service.
#[derive(Clone, Copy, Debug)]
pub struct EngineConfig {
    /// Buffered blocks at which the engine stops taking actions until the save in flight
    /// completes.
    pub backpressure_threshold: u64,
    /// How long buffered blocks below the persistence threshold wait before a save starts anyway.
    pub idle_flush_interval: Duration,
    /// Buffered blocks that start a save.
    pub persistence_threshold: u64,
    /// How long the engine waits for a save or an unwind before giving up on it.
    pub persistence_timeout: Duration,
}

impl Default for EngineConfig {
    fn default() -> Self {
        Self {
            backpressure_threshold: DEFAULT_BACKPRESSURE_THRESHOLD,
            idle_flush_interval: DEFAULT_IDLE_FLUSH_INTERVAL,
            persistence_threshold: DEFAULT_PERSISTENCE_THRESHOLD,
            persistence_timeout: DEFAULT_PERSISTENCE_TIMEOUT,
        }
    }
}

/// Messages sent from [`EngineHandle`] to the engine thread.
enum EngineAction<Block: reth_primitives_traits::Block> {
    /// Execute a block via the EVM and index the resulting trie diff.
    ExecuteBlock(tasks::ExecuteBlockTask<Block>),
    /// Index pre-computed trie updates for a block (no EVM execution).
    IndexBlock(tasks::IndexBlockTask),
    /// Handle a reorg: unwind to the common ancestor then index or execute the new chain.
    Reorg(tasks::ReorgTask<Block>),
    /// Unwind indexed data back to a given block.
    Unwind(tasks::UnwindTask),
    /// Block the caller until any in-flight persistence completes.
    #[cfg(any(test, feature = "test-utils"))]
    Flush(tasks::FlushTask),
    /// Update the sync catch-up target (fire-and-forget).
    SyncTo(tasks::SyncToTask),
    /// Report the engine's tip.
    #[cfg(any(test, feature = "test-utils"))]
    Tip(tasks::TipTask),
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
            #[cfg(any(test, feature = "test-utils"))]
            Self::Flush(task) => task.execute(state),
            Self::SyncTo(task) => task.execute(state),
            #[cfg(any(test, feature = "test-utils"))]
            Self::Tip(task) => task.execute(state),
        }
    }
}

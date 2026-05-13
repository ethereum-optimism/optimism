//! [`EngineHandle`] — the public, cloneable, Send + Sync interface.

use super::{
    DEFAULT_BACKPRESSURE_THRESHOLD, DEFAULT_PERSISTENCE_THRESHOLD, EngineAction,
    error::EngineError,
    runner::Engine,
    service_guard::ServiceGuard,
    tasks::{ExecuteBlockTask, IndexBlockTask, ReorgTask, SyncToTask, UnwindTask},
};
use crate::{OpProofStoragePruner, OpProofsStore};
use alloy_eips::eip1898::BlockWithParent;
use crossbeam_channel::{Sender, bounded};
use reth_evm::ConfigureEvm;
use reth_primitives_traits::{NodePrimitives, RecoveredBlock};
use reth_provider::{
    BlockHashReader, BlockReader, DatabaseProviderFactory, StateProviderFactory, StateReader,
};
use reth_trie_common::{HashedPostStateSorted, updates::TrieUpdatesSorted};
use std::{panic, sync::Arc, thread};
use tracing::error;

/// A thin, cloneable handle used to communicate with the collector engine.
///
/// Every public method (except [`Self::sync_to`]) sends an engine action to the
/// engine thread and blocks on a one-shot reply channel.
#[derive(Debug, Clone)]
pub struct EngineHandle<Block: reth_primitives_traits::Block> {
    sender: Sender<EngineAction<Block>>,
    _service_guard: Arc<ServiceGuard>,
}

impl<Block: reth_primitives_traits::Block + Send + 'static> EngineHandle<Block> {
    /// Spawn the collector engine on a new thread and return a handle.
    pub fn spawn<Evm, Provider, Store>(
        evm_config: Evm,
        provider: Provider,
        storage: Store,
        pruner: OpProofStoragePruner<Store, Provider>,
    ) -> Self
    where
        Evm: ConfigureEvm<Primitives: NodePrimitives<Block = Block>> + 'static,
        Provider: BlockHashReader
            + StateReader
            + DatabaseProviderFactory
            + StateProviderFactory
            + BlockReader<Block = Block>
            + Clone
            + 'static,
        Store: OpProofsStore + Clone + 'static,
    {
        Self::spawn_with_thresholds(
            evm_config,
            provider,
            storage,
            pruner,
            DEFAULT_PERSISTENCE_THRESHOLD,
            DEFAULT_BACKPRESSURE_THRESHOLD,
        )
    }

    /// Spawn the collector engine with custom persistence and backpressure thresholds.
    pub fn spawn_with_thresholds<Evm, Provider, Store>(
        evm_config: Evm,
        provider: Provider,
        storage: Store,
        pruner: OpProofStoragePruner<Store, Provider>,
        persistence_threshold: u64,
        backpressure_threshold: u64,
    ) -> Self
    where
        Evm: ConfigureEvm<Primitives: NodePrimitives<Block = Block>> + 'static,
        Provider: BlockHashReader
            + StateReader
            + DatabaseProviderFactory
            + StateProviderFactory
            + BlockReader<Block = Block>
            + Clone
            + 'static,
        Store: OpProofsStore + Clone + 'static,
    {
        let (tx, rx) = bounded(10);
        let engine = Engine::new(evm_config, provider, storage, pruner, rx)
            .with_persistence_threshold(persistence_threshold)
            .with_backpressure_threshold(backpressure_threshold);

        let join_handle = thread::Builder::new()
            .name("live-trie-collector".into())
            .spawn(move || {
                if let Err(panic) = panic::catch_unwind(panic::AssertUnwindSafe(|| engine.run())) {
                    let msg = panic
                        .downcast_ref::<&str>()
                        .copied()
                        .or_else(|| panic.downcast_ref::<String>().map(String::as_str))
                        .unwrap_or("unknown");
                    error!(target: "live-trie::engine", %msg, "Collector engine panicked");
                }
            })
            .expect("failed to spawn live-trie-collector thread");

        Self { sender: tx, _service_guard: Arc::new(ServiceGuard::new(join_handle)) }
    }

    fn send_and_recv(
        &self,
        make_action: impl FnOnce(Sender<Result<(), EngineError>>) -> EngineAction<Block>,
    ) -> Result<(), EngineError> {
        let (reply_tx, reply_rx) = bounded(1);
        self.sender.send(make_action(reply_tx)).map_err(|_| EngineError::EngineDied)?;
        reply_rx.recv().map_err(|_| EngineError::EngineDied)?
    }

    /// Execute a block through the EVM and buffer the resulting trie updates.
    pub fn execute_block(&self, block: &RecoveredBlock<Block>) -> Result<(), EngineError>
    where
        Block: Clone,
    {
        self.send_and_recv(|reply| {
            EngineAction::ExecuteBlock(ExecuteBlockTask { block: block.clone(), reply })
        })
    }

    /// Buffer pre-computed trie updates for `block` (no EVM execution).
    pub fn index_block(
        &self,
        block: BlockWithParent,
        sorted_trie_updates: TrieUpdatesSorted,
        sorted_post_state: HashedPostStateSorted,
    ) -> Result<(), EngineError> {
        self.send_and_recv(|reply| {
            EngineAction::IndexBlock(IndexBlockTask {
                block,
                sorted_trie_updates,
                sorted_post_state,
                reply,
            })
        })
    }

    /// Handle a chain reorg: unwind to the common ancestor then buffer new fork blocks.
    pub fn reorg(
        &self,
        block_updates: Vec<(BlockWithParent, Arc<TrieUpdatesSorted>, Arc<HashedPostStateSorted>)>,
    ) -> Result<(), EngineError> {
        self.send_and_recv(|reply| EngineAction::Reorg(ReorgTask { block_updates, reply }))
    }

    /// Unwind indexed data back to `to` (first block number removed, inclusive).
    pub fn unwind(&self, to: BlockWithParent) -> Result<(), EngineError> {
        self.send_and_recv(|reply| EngineAction::Unwind(UnwindTask { to, reply }))
    }

    /// Update the sync catch-up target (fire-and-forget).
    ///
    /// The engine will execute blocks up to `target` during its idle time,
    /// interleaving catch-up work with incoming actions.
    pub fn sync_to(&self, target: u64) -> Result<(), EngineError> {
        self.sender
            .send(EngineAction::SyncTo(SyncToTask { target }))
            .map_err(|_| EngineError::EngineDied)
    }

    /// Block until any in-progress background persistence completes (test/utility only).
    #[cfg(any(test, feature = "test-utils"))]
    pub fn flush(&self) {
        use super::tasks::FlushTask;
        let (reply_tx, reply_rx) = bounded(1);
        if self.sender.send(EngineAction::Flush(FlushTask { reply: reply_tx })).is_ok() {
            let _ = reply_rx.recv();
        }
    }
}

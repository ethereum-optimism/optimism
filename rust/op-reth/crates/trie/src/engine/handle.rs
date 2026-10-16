//! [`EngineHandle`] — the public, cloneable, Send + Sync interface.

use super::{
    EngineAction, EngineConfig,
    error::EngineError,
    runner::Engine,
    service_guard::ServiceGuard,
    tasks::{ExecuteBlockTask, IndexBlockTask, ReorgBlock, ReorgTask, SyncToTask, UnwindTask},
};
use crate::{OpProofStoragePruner, OpProofsStore};
#[cfg(any(test, feature = "test-utils"))]
use alloy_eips::NumHash;
use alloy_eips::eip1898::BlockWithParent;
use crossbeam_channel::{Receiver, Sender, bounded, select_biased};
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
/// engine thread and blocks on a one-shot reply channel, or until the engine exits.
#[derive(Debug, Clone)]
pub struct EngineHandle<Block: reth_primitives_traits::Block> {
    /// Disconnects when the engine thread exits by any path, so that no call waits on a reply
    /// the engine will never send: an action queued when the engine exits stays in the channel,
    /// with its reply sender, for as long as a handle lives.
    engine_exited: Receiver<()>,
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
        Self::spawn_with_config(evm_config, provider, storage, pruner, EngineConfig::default())
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
        let config =
            EngineConfig { backpressure_threshold, persistence_threshold, ..Default::default() };
        Self::spawn_with_config(evm_config, provider, storage, pruner, config)
    }

    /// Spawn the collector engine with `config`.
    pub fn spawn_with_config<Evm, Provider, Store>(
        evm_config: Evm,
        provider: Provider,
        storage: Store,
        pruner: OpProofStoragePruner<Store, Provider>,
        config: EngineConfig,
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
        let (exited, engine_exited) = bounded(0);
        let engine = Engine::new(evm_config, provider, storage, pruner, rx, exited, config);

        let join_handle = thread::Builder::new()
            .name("live-trie-collector".into())
            .spawn(move || {
                if let Err(panic) = panic::catch_unwind(panic::AssertUnwindSafe(|| engine.run())) {
                    let msg = panic
                        .downcast_ref::<&str>()
                        .copied()
                        .or_else(|| panic.downcast_ref::<String>().map(String::as_str))
                        .unwrap_or("unknown");
                    error!(target: "trie::engine::handle", %msg, "Collector engine panicked");
                }
            })
            .expect("failed to spawn live-trie-collector thread");

        Self { engine_exited, sender: tx, _service_guard: Arc::new(ServiceGuard::new(join_handle)) }
    }

    /// Waits for the reply to an action, or for the engine to exit; a reply already there wins.
    fn recv_reply<Reply>(&self, reply: &Receiver<Reply>) -> Result<Reply, EngineError> {
        select_biased! {
            recv(reply) -> reply => reply.map_err(|_| EngineError::EngineDied),
            recv(self.engine_exited) -> _ => Err(EngineError::EngineDied),
        }
    }

    fn send_and_recv(
        &self,
        make_action: impl FnOnce(Sender<Result<(), EngineError>>) -> EngineAction<Block>,
    ) -> Result<(), EngineError> {
        let (reply_tx, reply_rx) = bounded(1);
        self.sender.send(make_action(reply_tx)).map_err(|_| EngineError::EngineDied)?;
        self.recv_reply(&reply_rx)?
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

    /// Handle a chain reorg: unwind to the common ancestor then buffer the new chain, indexing or
    /// executing each block as its [`ReorgBlock`] says.
    ///
    /// Fails without unwinding when the unwind would reach the window's earliest block, or when
    /// the first block's parent differs from the stored block below it and the engine knows that
    /// block's hash (it has buffered or persisted it, or it is a window end). A block that fails
    /// after the unwind leaves the store unwound with the blocks before it buffered.
    pub fn reorg(&self, blocks: Vec<ReorgBlock<Block>>) -> Result<(), EngineError> {
        self.send_and_recv(|reply| EngineAction::Reorg(ReorgTask { blocks, reply }))
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
            let _ = self.recv_reply(&reply_rx);
        }
    }

    /// The engine's tip: the newest buffered block, or the store's latest block when nothing is
    /// buffered (test/utility only).
    #[cfg(any(test, feature = "test-utils"))]
    pub fn tip(&self) -> Result<NumHash, EngineError> {
        use super::tasks::TipTask;
        let (reply_tx, reply_rx) = bounded(1);
        self.sender
            .send(EngineAction::Tip(TipTask { reply: reply_tx }))
            .map_err(|_| EngineError::EngineDied)?;
        self.recv_reply(&reply_rx)?
    }
}

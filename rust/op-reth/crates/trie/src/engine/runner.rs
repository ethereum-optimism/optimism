//! [`Engine`] — the thin event-loop dispatcher.

use super::{
    DEFAULT_BACKPRESSURE_THRESHOLD, DEFAULT_PERSISTENCE_THRESHOLD, EngineAction,
    error::EngineError, state::EngineState as State,
};
use crate::{OpProofStoragePruner, OpProofsStore};
use crossbeam_channel::Receiver;
use reth_evm::ConfigureEvm;
use reth_primitives_traits::BlockTy;
use reth_provider::{
    BlockHashReader, BlockReader, DatabaseProviderFactory, StateProviderFactory, StateReader,
    TransactionVariant,
};
use std::{
    ops::ControlFlow,
    time::{Duration, Instant},
};
use tracing::{debug, error};

/// The engine that runs on a dedicated thread, dispatching [`EngineAction`]
/// messages to self-contained task structs that operate on the engine state.
#[allow(missing_debug_implementations)]
pub(super) struct Engine<Evm, Provider, Store>
where
    Evm: ConfigureEvm,
    Provider: StateReader + DatabaseProviderFactory + StateProviderFactory + BlockReader,
{
    state: State<Evm, Provider, Store>,
    incoming: Receiver<EngineAction<BlockTy<Evm::Primitives>>>,
    persistence_threshold: u64,
    backpressure_threshold: u64,
}

impl<Evm, Provider, Store> Engine<Evm, Provider, Store>
where
    Evm: ConfigureEvm,
    Provider: BlockHashReader
        + StateReader
        + DatabaseProviderFactory
        + StateProviderFactory
        + BlockReader<Block = BlockTy<Evm::Primitives>>
        + Clone
        + 'static,
    Store: OpProofsStore + Clone + 'static,
{
    pub(super) fn new(
        evm_config: Evm,
        provider: Provider,
        storage: Store,
        pruner: OpProofStoragePruner<Store, Provider>,
        incoming: Receiver<EngineAction<BlockTy<Evm::Primitives>>>,
    ) -> Self {
        Self {
            state: State::new(evm_config, provider, storage, pruner),
            incoming,
            persistence_threshold: DEFAULT_PERSISTENCE_THRESHOLD,
            backpressure_threshold: DEFAULT_BACKPRESSURE_THRESHOLD,
        }
    }

    pub(super) const fn with_persistence_threshold(mut self, threshold: u64) -> Self {
        self.persistence_threshold = threshold;
        self
    }

    pub(super) const fn with_backpressure_threshold(mut self, threshold: u64) -> Self {
        self.backpressure_threshold = threshold;
        self
    }

    /// Returns `true` if the engine is behind its sync target.
    fn needs_sync(&self) -> bool {
        let current_tip = self.state.get_tip().map(|t| t.number).unwrap_or(0);
        self.state.sync_target > current_tip
    }

    /// Returns `true` if the buffer is above the backpressure threshold with a save in-flight.
    fn backpressure_active(&self) -> bool {
        self.state.persistence.in_flight.is_some() &&
            self.state.memory.len() as u64 >= self.backpressure_threshold
    }

    /// Start a background persistence save if the memory buffer has reached the threshold.
    fn maybe_start_save(&mut self) {
        if self.state.memory.len() as u64 >= self.persistence_threshold &&
            let Err(e) = self.state.advance_persistence()
        {
            error!(target: "live-trie::engine", ?e, "Failed to start persistence save");
        }
    }

    /// Execute the next sequential block (`current_tip + 1`) to advance toward the sync target.
    fn advance_sync(&mut self) -> Result<(), EngineError> {
        let current_tip = self.state.get_tip()?.number;

        if self.state.sync_target <= current_tip {
            return Ok(());
        }

        let block_num = current_tip + 1;
        let block = self
            .state
            .provider
            .recovered_block(block_num.into(), TransactionVariant::NoHash)?
            .ok_or(EngineError::BlockNotFound(block_num))?;

        super::tasks::execute_block(&block, &mut self.state)
    }

    /// Process one event from the action, persistence, or sync channel.
    ///
    /// Three receivers compete in a single `select!`:
    /// - **action**: a new [`EngineAction`] from a caller, or [`crossbeam_channel::never`] while
    ///   backpressure is active — callers naturally block in their bounded `send` until the
    ///   in-flight save completes and memory is pruned.
    /// - **persistence**: signals a completed background save.
    /// - **sync**: a zero-duration timer that fires immediately when the engine is behind its sync
    ///   target and not under backpressure; [`crossbeam_channel::never`] otherwise.
    ///
    /// Returns [`ControlFlow::Break`] when the action channel disconnects.
    fn process_next_event(&mut self) -> ControlFlow<()> {
        let backpressure = self.backpressure_active();

        let persist_rx =
            self.state.persistence.in_flight.clone().unwrap_or_else(crossbeam_channel::never);

        // Gate new actions while backpressure is active — don't grow memory while draining it.
        let incoming_rx: Receiver<EngineAction<BlockTy<Evm::Primitives>>> =
            if backpressure { crossbeam_channel::never() } else { self.incoming.clone() };

        // Fire immediately when there is sync work to do; block indefinitely otherwise.
        let sync_rx: Receiver<Instant> = if self.needs_sync() && !backpressure {
            crossbeam_channel::after(Duration::ZERO)
        } else {
            crossbeam_channel::never()
        };

        crossbeam_channel::select! {
            recv(incoming_rx) -> msg => match msg {
                Ok(action) => action.execute(&mut self.state),
                Err(_) => return ControlFlow::Break(()),
            },
            recv(persist_rx) -> result => self.state.persistence.on_complete(result, &self.state.memory),
            recv(sync_rx) -> _ => if let Err(err) = self.advance_sync() {
                error!(target: "live-trie::engine", ?err, "Sync step failed");
            },
        }
        ControlFlow::Continue(())
    }

    /// Runs the main loop of the engine, processing incoming actions.
    pub(super) fn run(mut self) {
        debug_assert!(
            self.persistence_threshold < self.backpressure_threshold,
            "backpressure_threshold ({}) must be greater than persistence_threshold ({})",
            self.backpressure_threshold,
            self.persistence_threshold,
        );
        debug!(target: "live-trie::engine", "Collector engine started");

        loop {
            match self.process_next_event() {
                ControlFlow::Break(()) => break,
                ControlFlow::Continue(()) => {}
            }
            self.maybe_start_save();
        }

        debug!(target: "live-trie::engine", "Collector engine shutting down, draining in-flight persist");
        self.state.drain_persistence();
        debug!(target: "live-trie::engine", "Collector engine stopped");
    }
}

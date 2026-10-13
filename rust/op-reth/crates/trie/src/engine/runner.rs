//! [`Engine`] — the thin event-loop dispatcher.

use super::{
    CanonicalInMemoryStateProvider, DEFAULT_BACKPRESSURE_THRESHOLD, DEFAULT_PERSISTENCE_THRESHOLD,
    EngineAction, IDLE_FLUSH_INTERVAL, error::EngineError, state::EngineState as State,
};
use crate::{OpProofStoragePruner, OpProofsProviderRO, OpProofsStore};
use alloy_eips::NumHash;
use crossbeam_channel::Receiver;
use reth_evm::ConfigureEvm;
use reth_primitives_traits::BlockTy;
use reth_provider::{BlockHashReader, BlockReader, DatabaseProviderFactory, TransactionVariant};
use std::{ops::ControlFlow, time::Instant};
use tokio::sync::watch;
use tracing::{debug, error, info};

/// The engine that runs on a dedicated thread, dispatching [`EngineAction`]
/// messages to self-contained task structs that operate on the engine state.
#[allow(missing_debug_implementations)]
pub(super) struct Engine<Evm, Provider, Store>
where
    Evm: ConfigureEvm,
    Provider: CanonicalInMemoryStateProvider + DatabaseProviderFactory + BlockReader,
{
    state: State<Evm, Provider, Store>,
    incoming: Receiver<EngineAction<BlockTy<Evm::Primitives>>>,
    persistence_threshold: u64,
    backpressure_threshold: u64,
    /// Proofs tip at which syncing paused because reth's next block doesn't build on it. Syncing
    /// resumes once an action moves the tip.
    sync_paused_at: Option<NumHash>,
    /// Publishes the highest block whose proofs data is persisted.
    persisted_tip: watch::Sender<Option<NumHash>>,
    /// Whether the last persisted-tip read failed, so a failure is logged once, not per event.
    persisted_tip_read_failed: bool,
}

impl<Evm, Provider, Store> Engine<Evm, Provider, Store>
where
    Evm: ConfigureEvm,
    Provider: BlockHashReader
        + CanonicalInMemoryStateProvider
        + DatabaseProviderFactory
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
        persisted_tip: watch::Sender<Option<NumHash>>,
    ) -> Self {
        Self {
            state: State::new(evm_config, provider, storage, pruner),
            incoming,
            persistence_threshold: DEFAULT_PERSISTENCE_THRESHOLD,
            backpressure_threshold: DEFAULT_BACKPRESSURE_THRESHOLD,
            sync_paused_at: None,
            persisted_tip,
            persisted_tip_read_failed: false,
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

    /// Publishes the persisted tip if it changed.
    ///
    /// Only persisted blocks are published: buffered blocks are lost on a crash and must then be
    /// re-executed from reth, so reth must not prune them yet.
    fn publish_persisted_tip(&mut self) {
        let tip = match self.state.storage.provider_ro().and_then(|p| p.get_latest_block()) {
            Ok(tip) => tip,
            Err(err) => {
                if !self.persisted_tip_read_failed {
                    error!(target: "trie::engine::runner", ?err, "Failed to read persisted tip");
                }
                self.persisted_tip_read_failed = true;
                return;
            }
        };
        if std::mem::take(&mut self.persisted_tip_read_failed) {
            info!(target: "trie::engine::runner", "Persisted tip readable again");
        }
        self.persisted_tip.send_if_modified(|published| {
            let changed = *published != Some(tip);
            *published = Some(tip);
            changed
        });
    }

    /// Returns `true` if the engine is behind its sync target and syncing isn't paused at the
    /// current tip.
    fn needs_sync(&self) -> bool {
        let Ok(tip) = self.state.get_tip() else { return false };
        self.state.sync_target > tip.number && self.sync_paused_at != Some(tip)
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
            error!(target: "trie::engine::runner", ?e, "Failed to start persistence save");
        }
    }

    /// Execute the next sequential block (`current_tip + 1`) to advance toward the sync target.
    ///
    /// The block comes from reth's canonical chain, so a failure only goes away if reth replaces
    /// the block. An error returned here would repeat and stops the engine.
    fn advance_sync(&mut self) -> Result<(), EngineError> {
        let tip = self.state.get_tip()?;
        let current_tip = tip.number;

        if self.state.sync_target <= current_tip {
            return Ok(());
        }

        let block_num = current_tip + 1;

        // Mitigation for upstream reth's mmap/ftruncate race
        // (`https://github.com/paradigmxyz/reth/issues/24411`). Reading via
        // `provider.recovered_block(...)` dereferences the static-file `Arc<DataReader>`
        // mmap; if reth's engine tree is unwinding past the requested block at the same
        // moment, the load can land on a kernel page reclaimed by `ftruncate(2)` and
        // SIGBUS the process. We narrow the race window by refusing to read past the
        // chain's current best block — blocks beyond it are the ones most likely to be
        // mid-truncate. Remove once the upstream fix lands.
        let best_block = self.state.provider.best_block_number()?;
        if block_num > best_block {
            // Clamp `sync_target` to the chain's best block so the runner doesn't
            // busy-loop on `advance_sync`.
            debug!(
                target: "trie::engine::runner",
                block_num,
                best_block,
                sync_target = self.state.sync_target,
                "Clamping sync_target to best_block to avoid mmap/ftruncate race"
            );
            self.state.sync_target = best_block;
            return Ok(());
        }

        let Some(block) =
            self.state.provider.recovered_block(block_num.into(), TransactionVariant::NoHash)?
        else {
            // reth unwound past the block since the best-block read; its notification moves the
            // tip.
            self.sync_paused_at = Some(tip);
            return Ok(());
        };

        match super::tasks::execute_block(&block, &mut self.state) {
            Ok(()) => Ok(()),
            // The proofs tip is off reth's chain: either a reorg notification is still on its way,
            // or the proofs store is on a fork reth abandoned (e.g. after an unclean shutdown).
            // Executing again can't help, so wait for the next action to move the tip. A stale
            // store fails on the same mismatch when the next notification arrives.
            Err(err @ EngineError::ParentHashMismatch { .. }) => {
                debug!(target: "trie::engine::runner", %err, "Pausing sync until the proofs tip moves");
                self.sync_paused_at = Some(tip);
                Ok(())
            }
            // reth replaced the block while it executed (e.g. it reads bytecode from a fork reth
            // has left), so the next step runs reth's new block at this height.
            Err(err) if self.state.provider.block_hash(block_num)? != Some(block.hash()) => {
                debug!(target: "trie::engine::runner", block_num, %err, "Block left reth's chain during sync");
                Ok(())
            }
            Err(err) => Err(err),
        }
    }

    /// Process one event from the action, persistence, sync, or idle-flush channel.
    ///
    /// Four receivers compete in a single `select!`:
    /// - **action**: a new [`EngineAction`] from a caller, or [`crossbeam_channel::never`] while
    ///   backpressure is active — callers naturally block in their bounded `send` until the
    ///   in-flight save completes and memory is pruned.
    /// - **persistence**: signals a completed background save.
    /// - **sync**: a zero-duration timer that fires immediately when the engine is behind its sync
    ///   target and not under backpressure; [`crossbeam_channel::never`] otherwise.
    /// - **idle-flush**: fires after [`IDLE_FLUSH_INTERVAL`] when memory holds buffered blocks but
    ///   the persistence threshold hasn't been reached and no save is in flight. Keeps a paused
    ///   chain from leaving buffered blocks invisible to the proofs RPC indefinitely.
    ///
    /// Returns [`ControlFlow::Break`] when the action channel disconnects or a sync step fails.
    fn process_next_event(&mut self) -> ControlFlow<()> {
        let backpressure = self.backpressure_active();
        let save_in_flight = self.state.persistence.in_flight.is_some();

        let persist_rx =
            self.state.persistence.in_flight.clone().unwrap_or_else(crossbeam_channel::never);

        // Gate new actions while backpressure is active — don't grow memory while draining it.
        let incoming_rx: Receiver<EngineAction<BlockTy<Evm::Primitives>>> =
            if backpressure { crossbeam_channel::never() } else { self.incoming.clone() };

        // Fire when there is sync work to do; block indefinitely otherwise.
        let sync_rx: Receiver<Instant> = if self.needs_sync() && !backpressure {
            crossbeam_channel::at(Instant::now())
        } else {
            crossbeam_channel::never()
        };

        // Arm the idle-flush timer only when there's buffered work that nothing else will flush.
        let idle_flush_rx: Receiver<Instant> =
            if !self.state.memory.is_empty() && !save_in_flight && !backpressure {
                crossbeam_channel::after(IDLE_FLUSH_INTERVAL)
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
                // Stopping the engine closes its persisted-tip channel, which the ExEx treats as
                // fatal, so the node shuts down.
                error!(target: "trie::engine::runner", %err, "Proofs sync failed; stopping the proofs engine");
                return ControlFlow::Break(());
            },
            recv(idle_flush_rx) -> _ => if let Err(e) = self.state.advance_persistence() {
                error!(target: "trie::engine::runner", ?e, "Idle flush failed");
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
        debug!(target: "trie::engine::runner", "Collector engine started");
        self.publish_persisted_tip();

        loop {
            match self.process_next_event() {
                ControlFlow::Break(()) => break,
                ControlFlow::Continue(()) => {}
            }
            self.maybe_start_save();
            self.publish_persisted_tip();
        }

        debug!(target: "trie::engine::runner", "Collector engine shutting down, draining in-flight persist");
        self.state.drain_persistence();
        debug!(target: "trie::engine::runner", "Collector engine stopped");
    }
}

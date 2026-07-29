//! [`Engine`] — the thin event-loop dispatcher.

use super::{
    DEFAULT_BACKPRESSURE_THRESHOLD, DEFAULT_PERSISTENCE_THRESHOLD, EngineAction,
    IDLE_FLUSH_INTERVAL, error::EngineError, state::EngineState as State,
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

/// First retry delay after a sync step fails. Successful steps reset the delay to zero.
const SYNC_BACKOFF_INITIAL: Duration = Duration::from_millis(100);

/// Upper bound for the exponential sync-retry delay. Caps log volume during persistent
/// failure modes (e.g. provider regression that outlives our state) to ~0.1/sec.
const SYNC_BACKOFF_MAX: Duration = Duration::from_secs(10);

fn next_sync_backoff(current: Duration) -> Duration {
    current.saturating_mul(2).clamp(SYNC_BACKOFF_INITIAL, SYNC_BACKOFF_MAX)
}

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
    /// Current retry delay for the sync arm; grows on failure, resets on success.
    sync_backoff: Duration,
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
            sync_backoff: Duration::ZERO,
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
            error!(target: "trie::engine::runner", ?e, "Failed to start persistence save");
        }
    }

    /// Execute the next sequential block (`current_tip + 1`) to advance toward the sync target.
    fn advance_sync(&mut self) -> Result<(), EngineError> {
        let current_tip = self.state.get_tip()?.number;

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

        let block = self
            .state
            .provider
            .recovered_block(block_num.into(), TransactionVariant::NoHash)?
            .ok_or(EngineError::BlockNotFound(block_num))?;

        super::tasks::execute_block(&block, &mut self.state)
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
    /// Returns [`ControlFlow::Break`] when the action channel disconnects.
    fn process_next_event(&mut self) -> ControlFlow<()> {
        let backpressure = self.backpressure_active();
        let save_in_flight = self.state.persistence.in_flight.is_some();

        let persist_rx =
            self.state.persistence.in_flight.clone().unwrap_or_else(crossbeam_channel::never);

        // Gate new actions while backpressure is active — don't grow memory while draining it.
        let incoming_rx: Receiver<EngineAction<BlockTy<Evm::Primitives>>> =
            if backpressure { crossbeam_channel::never() } else { self.incoming.clone() };

        // Fire when there is sync work to do; block indefinitely otherwise. After a failed
        // sync step we delay by `sync_backoff` so a sustained failure (e.g. a stale sync target
        // ahead of the provider's tip) doesn't spin the CPU and flood logs.
        let sync_rx: Receiver<Instant> = if self.needs_sync() && !backpressure {
            crossbeam_channel::after(self.sync_backoff)
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
            recv(sync_rx) -> _ => match self.advance_sync() {
                Ok(()) => self.sync_backoff = Duration::ZERO,
                Err(err) => {
                    self.sync_backoff = next_sync_backoff(self.sync_backoff);
                    error!(target: "trie::engine::runner", ?err, backoff = ?self.sync_backoff, "Sync step failed");
                }
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

        loop {
            match self.process_next_event() {
                ControlFlow::Break(()) => break,
                ControlFlow::Continue(()) => {}
            }
            self.maybe_start_save();
        }

        debug!(target: "trie::engine::runner", "Collector engine shutting down, draining in-flight persist");
        self.state.drain_persistence();
        debug!(target: "trie::engine::runner", "Collector engine stopped");
    }
}

#[cfg(test)]
mod tests {
    use super::{SYNC_BACKOFF_INITIAL, SYNC_BACKOFF_MAX, next_sync_backoff};
    use std::time::Duration;

    #[test]
    fn zero_backoff_jumps_to_initial() {
        // First failure after a clean run: backoff seeded at the configured initial delay.
        assert_eq!(next_sync_backoff(Duration::ZERO), SYNC_BACKOFF_INITIAL);
    }

    #[test]
    fn doubles_below_cap() {
        let stepped = next_sync_backoff(SYNC_BACKOFF_INITIAL);
        assert_eq!(stepped, SYNC_BACKOFF_INITIAL * 2);
    }

    #[test]
    fn clamps_to_max() {
        // Halfway to the cap doubles into the cap.
        let half = SYNC_BACKOFF_MAX / 2;
        assert_eq!(next_sync_backoff(half), SYNC_BACKOFF_MAX);

        // Already at the cap stays at the cap.
        assert_eq!(next_sync_backoff(SYNC_BACKOFF_MAX), SYNC_BACKOFF_MAX);

        // Just past the cap (defensive) is still clamped.
        assert_eq!(
            next_sync_backoff(SYNC_BACKOFF_MAX + Duration::from_millis(1)),
            SYNC_BACKOFF_MAX
        );
    }

    #[test]
    fn does_not_overflow_on_huge_input() {
        let near_max = Duration::new(u64::MAX / 2 + 1, 0);
        assert_eq!(next_sync_backoff(near_max), SYNC_BACKOFF_MAX);
    }
}

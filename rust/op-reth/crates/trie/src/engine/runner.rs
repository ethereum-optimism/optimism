//! [`Engine`] — the thin event-loop dispatcher.

use super::{EngineAction, EngineConfig, error::EngineError, state::EngineState as State};
use crate::{OpProofStoragePruner, OpProofsStore};
use crossbeam_channel::{Receiver, Sender};
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
    /// Dropped when the engine exits by any path, which every call waiting on the handle takes
    /// as `EngineDied`. Must stay declared first: fields drop in declaration order, and dropping
    /// `state` joins the persistence thread, which a slow save can hold indefinitely.
    _exited: Sender<()>,
    incoming: Receiver<EngineAction<BlockTy<Evm::Primitives>>>,
    state: State<Evm, Provider, Store>,
    config: EngineConfig,
    /// Current retry delay for the sync arm; grows on failure, resets on success.
    sync_backoff: Duration,
    /// Earliest instant the next sync step may run: already past unless the last step failed.
    /// Only a failed step moves it, so no other event can postpone a retry.
    sync_retry_at: Instant,
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
        exited: Sender<()>,
        config: EngineConfig,
    ) -> Self {
        Self {
            _exited: exited,
            incoming,
            state: State::new(evm_config, provider, storage, pruner, config.persistence_timeout),
            config,
            sync_backoff: Duration::ZERO,
            sync_retry_at: Instant::now(),
        }
    }

    /// Returns `true` if the engine is behind its sync target.
    fn needs_sync(&self) -> bool {
        let current_tip = self.state.get_tip().map(|t| t.number).unwrap_or(0);
        self.state.sync_target > current_tip
    }

    /// Returns `true` if the buffer is above the backpressure threshold with a save in-flight.
    fn backpressure_active(&self) -> bool {
        self.state.persistence.in_flight.is_some() &&
            self.state.memory.len() as u64 >= self.config.backpressure_threshold
    }

    /// Start a background persistence save if the memory buffer has reached the threshold.
    fn maybe_start_save(&mut self) {
        if self.state.memory.len() as u64 >= self.config.persistence_threshold &&
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
    ///   backpressure is active — callers' actions then wait in the channel until the in-flight
    ///   save completes and memory is pruned.
    /// - **persistence**: signals a completed background save.
    /// - **sync**: fires once `sync_retry_at` has passed, at once unless the last step failed,
    ///   while the engine is behind its sync target and not under backpressure;
    ///   [`crossbeam_channel::never`] otherwise.
    /// - **idle-flush**: fires after the idle flush interval when memory holds buffered blocks but
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

        // Fire when there is sync work to do; block indefinitely otherwise. A failed step pushes
        // `sync_retry_at` out so a sustained failure (e.g. a stale sync target ahead of the
        // provider's tip) doesn't spin the CPU and flood logs.
        let sync_rx: Receiver<Instant> = if self.needs_sync() && !backpressure {
            crossbeam_channel::at(self.sync_retry_at)
        } else {
            crossbeam_channel::never()
        };

        // Arm the idle-flush timer only when there's buffered work that nothing else will flush.
        let idle_flush_rx: Receiver<Instant> =
            if !self.state.memory.is_empty() && !save_in_flight && !backpressure {
                crossbeam_channel::after(self.config.idle_flush_interval)
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
                    self.sync_retry_at = Instant::now() + self.sync_backoff;
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
            self.config.persistence_threshold < self.config.backpressure_threshold,
            "backpressure_threshold ({}) must be greater than persistence_threshold ({})",
            self.config.backpressure_threshold,
            self.config.persistence_threshold,
        );
        debug!(target: "trie::engine::runner", "Collector engine started");

        loop {
            match self.process_next_event() {
                ControlFlow::Break(()) => break,
                ControlFlow::Continue(()) => {}
            }
            self.maybe_start_save();
            if self.state.persistence.is_disconnected() {
                error!(target: "trie::engine::runner", "Persistence service gone, stopping the collector engine");
                break;
            }
        }

        debug!(target: "trie::engine::runner", "Collector engine shutting down, draining in-flight persist");
        self.state.drain_persistence();
        debug!(target: "trie::engine::runner", "Collector engine stopped");
    }
}

#[cfg(test)]
mod tests {
    use super::{Engine, SYNC_BACKOFF_INITIAL, SYNC_BACKOFF_MAX, next_sync_backoff};
    use crate::{
        InMemoryProofsStorage, OpProofStoragePruner, OpProofsInitProvider, OpProofsStore,
        engine::{EngineAction, EngineConfig, tasks::SyncToTask},
    };
    use alloy_consensus::Header;
    use alloy_eips::BlockNumHash;
    use crossbeam_channel::{Sender, bounded};
    use reth_chainspec::MAINNET;
    use reth_db_common::init::init_genesis;
    use reth_ethereum_primitives::Block;
    use reth_evm_ethereum::EthEvmConfig;
    use reth_primitives_traits::SealedHeader;
    use reth_provider::{
        providers::BlockchainProvider,
        test_utils::{MockNodeTypesWithDB, create_test_provider_factory_with_chain_spec},
    };
    use std::{
        ops::ControlFlow,
        thread,
        time::{Duration, Instant},
    };

    type TestEngine =
        Engine<EthEvmConfig, BlockchainProvider<MockNodeTypesWithDB>, InMemoryProofsStorage>;

    /// An engine whose store is anchored at genesis and whose sync target is block 1, which reth
    /// reports as its head but does not hold, so that every sync step fails.
    fn engine_behind_a_missing_block() -> (TestEngine, Sender<EngineAction<Block>>) {
        let chain_spec = MAINNET.clone();
        let factory = create_test_provider_factory_with_chain_spec(chain_spec.clone());
        let genesis_hash = init_genesis(&factory).expect("init genesis");
        let provider = BlockchainProvider::new(factory).expect("blockchain provider");
        let missing_head = Header { number: 1, parent_hash: genesis_hash, ..Default::default() };
        provider
            .canonical_in_memory_state()
            .set_canonical_head(SealedHeader::seal_slow(missing_head));
        let storage = InMemoryProofsStorage::new();
        let init = storage.initialization_provider().expect("init provider");
        init.set_initial_state_anchor(BlockNumHash::new(0, genesis_hash)).expect("set anchor");
        init.commit_initial_state().expect("commit initial state");
        OpProofsInitProvider::commit(init).expect("commit");
        let pruner = OpProofStoragePruner::new(storage.clone(), provider.clone(), 1000);
        let (actions, incoming) = bounded(10);
        let (exited, _engine_exited) = bounded(0);
        let mut engine = Engine::new(
            EthEvmConfig::ethereum(chain_spec),
            provider,
            storage,
            pruner,
            incoming,
            exited,
            EngineConfig::default(),
        );
        engine.state.sync_target = 1;
        (engine, actions)
    }

    fn step(engine: &mut TestEngine) {
        assert_eq!(engine.process_next_event(), ControlFlow::Continue(()));
    }

    /// Sync-target updates arriving faster than the retry delay neither postpone the retry
    /// (ethereum-optimism/protocol-team#477) nor bring it forward.
    #[test]
    fn sync_retry_is_not_postponed_by_incoming_actions() {
        let (mut engine, actions) = engine_behind_a_missing_block();
        // Three failed steps grow the retry delay to four times the initial one.
        for _ in 0..3 {
            step(&mut engine);
        }
        let third_failure = Instant::now();
        let retry_delay = engine.sync_backoff;
        assert_eq!(retry_delay, 4 * SYNC_BACKOFF_INITIAL);
        let action_gap = retry_delay / 8;
        let patience = 10 * retry_delay;

        // Every step finds an action queued, which is what kept re-arming the retry timer.
        let deadline = Instant::now() + patience;
        while engine.sync_backoff == retry_delay {
            assert!(
                Instant::now() < deadline,
                "no sync step within {patience:?} while actions kept arriving every {action_gap:?}"
            );
            actions.send(EngineAction::SyncTo(SyncToTask { target: 1 })).expect("engine alive");
            step(&mut engine);
            thread::sleep(action_gap);
        }

        assert_eq!(engine.sync_backoff, 2 * retry_delay, "the step ran once and failed again");
        assert!(
            third_failure.elapsed() >= retry_delay - action_gap,
            "the step ran before its retry delay had passed"
        );
    }

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

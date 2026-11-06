//! Liveness of the proofs engine over a slow, stalling or dying persistence service.
//!
//! The engine waits for the save in flight before every unwind and gates actions under
//! backpressure, so a slow store stalls the ExEx, and reth behind it, for as long as the engine
//! waits. These cases feed the engine over a store whose writes outlast its idle flush interval
//! or its persistence timeout and check that every call returns rather than hangs, that
//! backpressure holds and releases, that a failed save is retried, that a reorg or a revert under
//! a save in flight completes or times out as the persistence timeout says, and that a dead
//! persistence service or a panic on the engine thread reaches callers, gated ones included, as
//! `EngineDied` instead of a hang.

#[path = "live/chain.rs"]
#[allow(dead_code, reason = "each test target uses part of this module")]
mod chain;
#[path = "live/simulated_reth.rs"]
#[allow(dead_code, reason = "each test target uses part of this module")]
mod simulated_reth;
#[path = "live/store_hooks.rs"]
#[allow(dead_code, reason = "each test target uses part of this module")]
mod store_hooks;
#[path = "live/store_view.rs"]
#[allow(dead_code, reason = "each test target uses part of this module")]
mod store_view;
#[path = "live/waits.rs"]
#[allow(dead_code, reason = "each test target uses part of this module")]
mod waits;

use alloy_consensus::BlockHeader;
use alloy_eips::NumHash;
use chain::*;
use eyre::WrapErr;
use reth_chain_state::ExecutedBlock;
use reth_ethereum_primitives::Block;
use reth_optimism_trie::{
    MdbxProofsStorage,
    engine::{EngineConfig, EngineError, EngineHandle, ReorgBlock},
};
use rstest::rstest;
use simulated_reth::{
    DEFAULT_PERSISTENCE_THRESHOLD, RethPersistence, SimulatedReth, fixed_signer,
    index_executed_block, reorg_block,
};
use std::{
    sync::Arc,
    time::{Duration, Instant},
};
use store_hooks::{HeldWrite, StoreWithHooks};
use store_view::{BlockView, ProofTarget, proof_window};
use waits::{PendingCall, wait_until};

/// The engine's idle flush interval and persistence timeout, lowered from their defaults so
/// that a save slower than either fits the time budget.
const IDLE_FLUSH_INTERVAL: Duration = Duration::from_millis(100);
const PERSISTENCE_TIMEOUT: Duration = Duration::from_millis(400);

/// A save slower than the idle flush interval, within the persistence timeout.
const SAVE_OVER_IDLE_FLUSH: Duration = Duration::from_millis(200);

/// A save slower than the persistence timeout.
const SAVE_OVER_PERSISTENCE_TIMEOUT: Duration = Duration::from_millis(800);

/// How long a call or a wait may take before it counts as hung; the saves in its way take at
/// most a few seconds.
const CALL_TIMEOUT: Duration = Duration::from_secs(10);

/// How long a reply that must come at once may take.
const PROMPT_REPLY_TIMEOUT: Duration = Duration::from_secs(2);

/// How long a call that must stay blocked is watched.
const BLOCKED_PROBE_DURATION: Duration = Duration::from_millis(300);

/// How reth leaves the chain the engine holds.
#[derive(Clone, Copy, Debug)]
enum Unwind {
    /// reth replaces its two newest blocks.
    Reorg,
    /// reth drops its newest block.
    Revert,
}

/// A gated engine call: the hold on the save it waits for, and the call itself.
type GatedCall = (HeldWrite, PendingCall<Result<(), EngineError>>);

/// reth, the engine and the store it runs over, with every engine call bounded.
struct Node {
    cadence: FlushCadence,
    engine: EngineHandle<Block>,
    reth: SimulatedReth,
    serial: u64,
    store: StoreWithHooks<Arc<MdbxProofsStorage>>,
    targets: Vec<ProofTarget>,
}

impl Node {
    /// A node whose store opens every write transaction `save_delay` late.
    fn new(cadence: FlushCadence, save_delay: Duration) -> eyre::Result<Self> {
        let key_pair = fixed_signer();
        let sender = public_key_to_address(key_pair.public_key());
        let chain_spec = chain_spec_with_address(sender);
        let persistence = RethPersistence::threshold(DEFAULT_PERSISTENCE_THRESHOLD);
        let reth = SimulatedReth::new(chain_spec, key_pair, persistence)?;
        let store = StoreWithHooks::new(create_mdbx_proofs_storage());
        initialize_storage(store.clone(), &reth.factory, reth.genesis())?;
        store.delay_writes(save_delay);
        let config = EngineConfig {
            backpressure_threshold: cadence.backpressure,
            idle_flush_interval: IDLE_FLUSH_INTERVAL,
            persistence_threshold: cadence.persistence,
            persistence_timeout: PERSISTENCE_TIMEOUT,
        };
        let engine = reth.spawn_engine_with_config(store.clone(), config, WHOLE_WINDOW);
        let targets = proof_targets_for_transfers(sender);
        Ok(Self { cadence, engine, reth, serial: 0, store, targets })
    }

    fn start_call<T: Send + 'static>(
        &self,
        call_name: &'static str,
        call: impl FnOnce(EngineHandle<Block>) -> T + Send + 'static,
    ) -> PendingCall<T> {
        PendingCall::spawn(self.engine.clone(), call_name, call)
    }

    /// Runs `call` against the engine, bounded by `bound`.
    fn call<T: Send + 'static>(
        &self,
        call_name: &'static str,
        bound: Duration,
        call: impl FnOnce(EngineHandle<Block>) -> T + Send + 'static,
    ) -> eyre::Result<(T, Duration)> {
        Ok(self.start_call(call_name, call).wait(bound)?)
    }

    const fn next_serial(&mut self) -> u64 {
        self.serial += 1;
        self.serial
    }

    /// Commits the next block to reth.
    fn next_block(&mut self) -> eyre::Result<ExecutedBlock> {
        let serial = self.next_serial();
        self.reth.commit(&serial_transfer_block(serial))
    }

    /// Commits the next block to reth and hands it to the engine, which must take it.
    fn index_next_block(&mut self) -> eyre::Result<()> {
        let block = self.next_block()?;
        let number = block.recovered_block().number();
        let (result, _) = self.call("index_block", CALL_TIMEOUT, move |engine| {
            index_executed_block(&engine, &block)
        })?;
        result.map_err(|err| eyre::eyre!("the engine refused block {number}: {err}"))
    }

    /// Commits and indexes `count` blocks back to back.
    fn feed(&mut self, count: u64) -> eyre::Result<()> {
        for _ in 0..count {
            self.index_next_block()?;
        }
        Ok(())
    }

    /// Feeds the blocks that bring an empty buffer to the persistence threshold, by which a save
    /// has started: at the threshold, or at the idle flush when a feed gap outlasted it.
    fn fill_to_persistence_threshold(&mut self) -> eyre::Result<()> {
        self.feed(self.cadence.persistence)
    }

    /// Starts a save and returns once the store has it, so that its delay runs from here.
    fn start_save(&mut self) -> eyre::Result<()> {
        let hold = self.store.hold_next_write();
        self.fill_to_persistence_threshold()?;
        hold.wait_until_entered();
        hold.release();
        Ok(())
    }

    fn tip(&self) -> eyre::Result<NumHash> {
        Ok(self.call("tip", CALL_TIMEOUT, |engine| engine.tip())?.0?)
    }

    fn store_latest(&self) -> eyre::Result<NumHash> {
        Ok(proof_window(self.store.inner())?.latest)
    }

    /// Waits until the store holds reth's head, which the engine's saves and then its idle flush
    /// bring about on their own once nothing more is fed. Polls the store rather than the engine:
    /// an engine action would postpone the idle flush this waits for.
    fn settle(&self) -> eyre::Result<()> {
        let head = self.reth.head();
        let mut latest = None;
        let settled = wait_until("the store reaching reth's head", CALL_TIMEOUT, || {
            latest = Some(self.store_latest()?);
            Ok(latest == Some(head))
        });
        settled.wrap_err_with(|| {
            format!("the store's latest block is {latest:?}, reth's head {head:?}")
        })
    }

    /// The engine's buffer, what its tip holds above the store's latest block, must not exceed
    /// the backpressure threshold: at the threshold the engine takes nothing in until the save
    /// in flight completes.
    fn ensure_buffer_within_backpressure(&self) -> eyre::Result<()> {
        let tip = self.tip()?;
        let latest = self.store_latest()?;
        let buffered = tip.number.checked_sub(latest.number).ok_or_else(|| {
            eyre::eyre!("the store's latest block {latest:?} is above the engine's tip {tip:?}")
        })?;
        eyre::ensure!(
            buffered <= self.cadence.backpressure,
            "the engine holds {buffered} blocks above the store's latest block {}, more than \
             its backpressure threshold {}",
            latest.number,
            self.cadence.backpressure
        );
        Ok(())
    }

    /// Holds the next save inside the store, fills the buffer to the backpressure threshold and
    /// issues one more `index_block`, which must stay blocked while the save is held. The node
    /// must hold nothing buffered before.
    fn gate_a_call(&mut self) -> eyre::Result<GatedCall> {
        let hold = self.store.hold_next_write();
        self.fill_to_persistence_threshold()?;
        hold.wait_until_entered();
        self.feed(self.cadence.backpressure - self.cadence.persistence)?;
        let block = self.next_block()?;
        let gated = self.start_call("index_block under backpressure", move |engine| {
            index_executed_block(&engine, &block)
        });
        gated.ensure_blocked_for(BLOCKED_PROBE_DURATION)?;
        Ok((hold, gated))
    }

    /// A call gated by backpressure must return once the held save is released.
    fn probe_backpressure(&mut self) -> eyre::Result<()> {
        self.settle()?;
        let (hold, gated) = self.gate_a_call()?;
        hold.release();
        let (result, _) = gated.wait(CALL_TIMEOUT)?;
        result.map_err(|err| eyre::eyre!("the engine refused the block it had gated: {err}"))
    }

    /// Fails the next save: the engine must take the next block meanwhile, retry the save and
    /// catch the store up.
    fn fail_a_save(&mut self) -> eyre::Result<()> {
        self.settle()?;
        self.store.fail_next_write();
        self.fill_to_persistence_threshold()?;
        self.index_next_block()?;
        self.settle()
    }

    /// Blocks fed back to back, the buffer held within the backpressure threshold, a gated call
    /// released, a failed save retried, and the store at reth's head after all of it.
    fn feed_under_backpressure_and_a_failed_save(&mut self) -> eyre::Result<()> {
        self.feed(3 * self.cadence.backpressure)?;
        self.ensure_buffer_within_backpressure()?;
        self.probe_backpressure()?;
        self.fail_a_save()?;
        self.verify_store_at_head()
    }

    /// Starts a save and, while it is in flight, hands the engine `unwind` of the chain whose
    /// newest block the save holds.
    fn unwind_under_a_save(
        &mut self,
        unwind: Unwind,
    ) -> eyre::Result<(Result<(), EngineError>, Duration)> {
        self.settle()?;
        self.start_save()?;
        match unwind {
            Unwind::Reorg => self.reorg(),
            Unwind::Revert => self.revert(),
        }
    }

    /// Replaces reth's two newest blocks and hands the reorg to the engine.
    fn reorg(&mut self) -> eyre::Result<(Result<(), EngineError>, Duration)> {
        let first = self.reth.head().number - 1;
        let serials = [self.next_serial(), self.next_serial()];
        let specs: Vec<BlockSpec> = serials.into_iter().map(serial_transfer_block).collect();
        let new = self.reth.reorg_with(first, &specs)?;
        let blocks: Vec<ReorgBlock<Block>> =
            new.iter().map(|block| reorg_block(block, false)).collect();
        self.call("reorg", CALL_TIMEOUT, move |engine| engine.reorg(blocks))
    }

    /// Drops reth's newest block and hands the revert to the engine.
    fn revert(&mut self) -> eyre::Result<(Result<(), EngineError>, Duration)> {
        let head = self.reth.head().number;
        let to = self.reth.block(head).block_with_parent();
        self.reth.revert_to(head - 1);
        self.call("unwind", CALL_TIMEOUT, move |engine| engine.unwind(to))
    }

    /// The store's latest block must be reth's head, serving proofs that hash to its state root.
    fn verify_store_at_head(&self) -> eyre::Result<()> {
        let head = self.reth.head();
        let latest = self.store_latest()?;
        eyre::ensure!(
            latest == head,
            "the store's latest block {latest:?} is not reth's head {head:?}"
        );
        BlockView::read(self.store.inner(), head.number, &self.targets)?
            .verify_proofs(head.number, self.reth.state_root(head.number))
    }
}

/// Blocks fed back to back over a store slower than the engine's idle flush interval: every call
/// returns, the buffer never outgrows the backpressure threshold, a call gated by backpressure
/// stays blocked until the save in flight completes, a failed save is retried, and a reorg or a
/// revert issued under a save in flight completes once the save has, after which the store
/// serves reth's head.
#[rstest]
fn engine_stays_live_over_a_slow_store(
    #[values(FlushCadence::EVERY_BLOCK, FlushCadence::EVERY_FIFTH_BLOCK)] cadence: FlushCadence,
    #[values(Unwind::Reorg, Unwind::Revert)] unwind: Unwind,
) -> eyre::Result<()> {
    let mut node = Node::new(cadence, SAVE_OVER_IDLE_FLUSH)?;
    node.feed_under_backpressure_and_a_failed_save()?;
    let (result, took) = node.unwind_under_a_save(unwind)?;
    result.map_err(|err| eyre::eyre!("the {unwind:?} under a save in flight failed: {err}"))?;
    eyre::ensure!(
        took >= SAVE_OVER_IDLE_FLUSH,
        "the {unwind:?} returned after {took:?}, before the save in flight could complete"
    );
    node.settle()?;
    node.verify_store_at_head()
}

/// With a save slower than the persistence timeout in flight, the engine gives up waiting for it
/// before unwinding, and gives up on its own unwind when the store, still busy with the save,
/// does not answer within a second timeout: the reorg or revert ends with `PersistenceTimeout`
/// after at least one timeout, and the engine still answers. The engine then holds a chain the
/// store no longer does, so nothing further is fed.
#[rstest]
fn an_unwind_under_a_save_slower_than_the_persistence_timeout_times_out(
    #[values(FlushCadence::EVERY_BLOCK, FlushCadence::EVERY_FIFTH_BLOCK)] cadence: FlushCadence,
    #[values(Unwind::Reorg, Unwind::Revert)] unwind: Unwind,
) -> eyre::Result<()> {
    let mut node = Node::new(cadence, SAVE_OVER_PERSISTENCE_TIMEOUT)?;
    node.feed_under_backpressure_and_a_failed_save()?;
    let (result, took) = node.unwind_under_a_save(unwind)?;
    eyre::ensure!(
        matches!(result, Err(EngineError::PersistenceTimeout)),
        "the {unwind:?} under a save slower than the persistence timeout ended with {result:?} \
         instead of a persistence timeout"
    );
    eyre::ensure!(
        took >= PERSISTENCE_TIMEOUT,
        "the {unwind:?} returned after {took:?}, before the persistence timeout"
    );
    node.tip()?;
    Ok(())
}

/// Once the persistence service's thread has died, nothing can be saved any more: the engine
/// must stop and refuse every further call with `EngineDied`, which the ExEx turns into an
/// error, rather than keep taking blocks into a buffer it can never flush.
#[rstest]
fn a_dead_persistence_service_reaches_callers_as_engine_died(
    #[values(FlushCadence::EVERY_BLOCK, FlushCadence::EVERY_FIFTH_BLOCK)] cadence: FlushCadence,
) -> eyre::Result<()> {
    let mut node = Node::new(cadence, Duration::ZERO)?;
    node.store.panic_in_next_write();

    // The save that kills the service starts by the persistence threshold at the latest.
    let deadline = Instant::now() + CALL_TIMEOUT;
    loop {
        let block = node.next_block()?;
        let (result, _) = node.call("index_block", CALL_TIMEOUT, move |engine| {
            index_executed_block(&engine, &block)
        })?;
        match result {
            Err(EngineError::EngineDied) => return Ok(()),
            Ok(()) => eyre::ensure!(
                Instant::now() < deadline,
                "the engine kept taking blocks for {CALL_TIMEOUT:?} after the panic was armed in \
                 its persistence service"
            ),
            Err(err) => eyre::bail!("the engine refused a block with {err} instead of EngineDied"),
        }
    }
}

/// A caller gated by backpressure waits for the save in flight; when that save kills the
/// persistence service instead of completing, the caller must get `EngineDied` at once rather
/// than wait forever on an action the engine will never take.
#[rstest]
fn a_dead_persistence_service_reaches_a_gated_caller_as_engine_died(
    #[values(FlushCadence::EVERY_BLOCK, FlushCadence::EVERY_FIFTH_BLOCK)] cadence: FlushCadence,
) -> eyre::Result<()> {
    let mut node = Node::new(cadence, Duration::ZERO)?;
    let (hold, gated) = node.gate_a_call()?;
    hold.release_into_panic();
    let (result, _) = gated.wait(PROMPT_REPLY_TIMEOUT)?;
    eyre::ensure!(
        matches!(result, Err(EngineError::EngineDied)),
        "the gated caller got {result:?} instead of EngineDied once the save killed the \
         persistence service"
    );
    Ok(())
}

/// A panic on the engine thread drops the reply of the call it was serving, and must reach every
/// other caller as `EngineDied` at once, even while the persistence thread is still inside a
/// save: no caller may have to wait for the store to learn that the engine is gone.
#[rstest]
fn an_engine_panic_reaches_waiting_callers_as_engine_died(
    #[values(FlushCadence::EVERY_BLOCK, FlushCadence::EVERY_FIFTH_BLOCK)] cadence: FlushCadence,
) -> eyre::Result<()> {
    let mut node = Node::new(cadence, Duration::ZERO)?;
    let hold = node.store.hold_next_write();
    node.fill_to_persistence_threshold()?;
    hold.wait_until_entered();

    node.store.panic_in_next_read();
    let block = node.next_block()?;
    let (result, _) = node.call("execute_block", CALL_TIMEOUT, move |engine| {
        engine.execute_block(block.recovered_block())
    })?;
    eyre::ensure!(
        matches!(result, Err(EngineError::EngineDied)),
        "the call the engine panicked in ended with {result:?} instead of EngineDied"
    );
    eyre::ensure!(
        !node.store.is_next_read_armed(),
        "the engine never opened the read view the panic was armed in"
    );

    let (result, _) =
        node.call("tip after the engine died", PROMPT_REPLY_TIMEOUT, |engine| engine.tip())?;
    eyre::ensure!(
        matches!(result, Err(EngineError::EngineDied)),
        "a caller after the engine's death got {result:?} instead of EngineDied"
    );
    hold.release();
    Ok(())
}

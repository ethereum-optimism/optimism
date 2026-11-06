//! Drives reth, the ExEx's decisions and the engine through a sequence of actions; the checks
//! after every step are in `invariants.rs`.

use super::{
    BlockSpec,
    actions::{Action, Config, Rebuild},
    chain_spec_with_address, initialize_storage,
    model::{Model, ModelBlock, Reply},
    proof_targets_for_transfers, public_key_to_address, serial_transfer_block,
    simulated_reth::{
        SimulatedReth, UNRELATED_PARENT, fixed_signer, index_executed_block, reorg_block,
        reorg_block_without_data,
    },
    store_hooks::StoreWithHooks,
    store_view::{ProofTarget, proof_window},
    waits::{CallError, PendingCall},
};
use alloy_consensus::BlockHeader;
use alloy_eips::{NumHash, eip1898::BlockWithParent};
use alloy_primitives::B256;
use reth_chain_state::ExecutedBlock;
use reth_ethereum_primitives::Block;
use reth_optimism_trie::{
    MdbxProofsStorage, OpProofsStorageError, ProofWindowRange,
    engine::{EngineError, EngineHandle, ReorgBlock},
};
use std::{mem, ops::ControlFlow, sync::Arc, time::Duration};
use tempfile::TempDir;

/// How long one engine call, or the engine settling after a step, may take before the case
/// fails instead of hanging.
pub(super) const STEP_TIMEOUT: Duration = Duration::from_secs(30);

fn model_block(block: &ExecutedBlock, serial: u64, persisted_before_commit: u64) -> ModelBlock {
    let recovered = block.recovered_block();
    ModelBlock {
        hash: recovered.hash(),
        parent: recovered.parent_hash(),
        persisted_before_commit,
        serial,
        state_root: recovered.state_root(),
    }
}

/// The engine and the store it runs over, dropped engine first so that every handle on the
/// store is gone before the store is reopened.
struct ProofsNode {
    engine: EngineHandle<Block>,
    store: StoreWithHooks<Arc<MdbxProofsStorage>>,
}

pub(super) struct Harness {
    pub(super) config: Config,
    /// Set when an engine call hung; the node is then leaked rather than joined.
    engine_hung: bool,
    /// The fork point of the last reorg or revert, a height the store had to unwind to.
    pub(super) last_fork: Option<u64>,
    log: Vec<String>,
    pub(super) model: Model,
    node: Option<ProofsNode>,
    pub(super) reth: SimulatedReth,
    serial: u64,
    store_dir: TempDir,
    pub(super) targets: Vec<ProofTarget>,
}

impl Drop for Harness {
    fn drop(&mut self) {
        if self.engine_hung {
            mem::forget(self.node.take());
        }
    }
}

impl Harness {
    pub(super) fn new(config: Config) -> eyre::Result<Self> {
        let key_pair = fixed_signer();
        let sender = public_key_to_address(key_pair.public_key());
        let chain_spec = chain_spec_with_address(sender);
        let genesis = ModelBlock {
            hash: chain_spec.genesis_hash(),
            parent: B256::ZERO,
            persisted_before_commit: 0,
            serial: 0,
            state_root: chain_spec.genesis_header().state_root,
        };
        let reth = if config.body_pruning {
            SimulatedReth::with_body_files_of(chain_spec, key_pair, config.persistence, 1)?
        } else {
            SimulatedReth::new(chain_spec, key_pair, config.persistence)?
        };
        let store_dir = TempDir::new()?;
        let store = StoreWithHooks::new(Arc::new(MdbxProofsStorage::new(store_dir.path())?));
        initialize_storage(store.clone(), &reth.factory, reth.genesis())?;
        let engine = Self::spawn_engine(&reth, &store, config);
        let targets = proof_targets_for_transfers(sender);
        Ok(Self {
            config,
            engine_hung: false,
            last_fork: None,
            log: Vec::new(),
            model: Model::new(genesis, config.window),
            node: Some(ProofsNode { engine, store }),
            reth,
            serial: 0,
            store_dir,
            targets,
        })
    }

    fn spawn_engine(
        reth: &SimulatedReth,
        store: &StoreWithHooks<Arc<MdbxProofsStorage>>,
        config: Config,
    ) -> EngineHandle<Block> {
        reth.spawn_engine_with_window(
            store.clone(),
            config.cadence.persistence,
            config.cadence.backpressure,
            config.window,
        )
    }

    const fn node(&self) -> &ProofsNode {
        self.node.as_ref().expect("the node is up between steps")
    }

    const fn engine(&self) -> &EngineHandle<Block> {
        &self.node().engine
    }

    pub(super) const fn store(&self) -> &StoreWithHooks<Arc<MdbxProofsStorage>> {
        &self.node().store
    }

    /// Runs `actions` in order, checking the invariants after each, and reports a failure with
    /// the steps that led to it; `ENGINE_MODEL_TRANSCRIPT` in the environment prints the steps
    /// of a passing case too.
    pub(super) fn run(&mut self, actions: &[Action]) -> eyre::Result<()> {
        let result = self.run_steps(actions);
        if std::env::var_os("ENGINE_MODEL_TRANSCRIPT").is_some() {
            eprintln!("{}\n", self.log.join("\n"));
        }
        result.map_err(|err| eyre::eyre!("{err:#}\n--- steps ---\n{}", self.log.join("\n")))
    }

    fn run_steps(&mut self, actions: &[Action]) -> eyre::Result<()> {
        for (index, action) in actions.iter().enumerate() {
            self.log(format!("step {index}: {action:?}"));
            if self.step(*action)?.is_break() {
                return Ok(());
            }
            self.check_invariants()?;
        }
        self.settle_store()?;
        self.check_settled()
    }

    pub(super) fn log(&mut self, line: String) {
        self.log.push(line);
    }

    fn step(&mut self, action: Action) -> eyre::Result<ControlFlow<()>> {
        match action {
            Action::BogusReorg { offset } => self.bogus_reorg(offset)?,
            Action::Commit { count } => {
                for _ in 0..count {
                    let block = self.commit_block()?;
                    self.deliver(&block)?;
                }
            }
            Action::Crash(rebuild) => self.crash(rebuild)?,
            Action::FailNextSave => self.fail_next_save()?,
            Action::Flush => self.flush()?,
            Action::Reorg { depth, length } => return self.reorg(depth, length),
            Action::Revert { depth } => return self.revert(depth),
            Action::StaleCommit { depth, length } => return self.stale_commit(depth, length),
            Action::SyncTo { count } => {
                for _ in 0..count {
                    self.commit_block()?;
                }
                let head = self.reth.head().number;
                // The ExEx acknowledges the tip as soon as it has queued the sync target, so
                // reth's pruner can run before the engine's loop reads its first block.
                self.prune_bodies_through(head)?;
                self.engine().sync_to(head)?;
                self.model.sync_to(head);
            }
        }
        Ok(ControlFlow::Continue(()))
    }

    /// Runs `call` against the engine on its own thread and fails the case if it does not return
    /// within [`STEP_TIMEOUT`], which a stalled engine would otherwise turn into a hang.
    pub(super) fn call<T: Send + 'static>(
        &mut self,
        call_name: &'static str,
        call: impl FnOnce(EngineHandle<Block>) -> T + Send + 'static,
    ) -> eyre::Result<T> {
        match PendingCall::spawn(self.engine().clone(), call_name, call).wait(STEP_TIMEOUT) {
            Ok((result, _)) => Ok(result),
            Err(err @ CallError::Hung { .. }) => {
                self.engine_hung = true;
                Err(err.into())
            }
            Err(err) => Err(err.into()),
        }
    }

    pub(super) fn window(&self) -> eyre::Result<ProofWindowRange> {
        proof_window(self.store())
    }

    const fn next_serial(&mut self) -> u64 {
        self.serial += 1;
        self.serial
    }

    /// Commits a new block to reth and to the model.
    fn commit_block(&mut self) -> eyre::Result<ExecutedBlock> {
        let serial = self.next_serial();
        let persisted_before_commit = self.reth.persisted().number;
        let block = self.reth.commit(&serial_transfer_block(serial))?;
        self.model.push(model_block(&block, serial, persisted_before_commit));
        Ok(block)
    }

    /// Replaces reth's chain above `fork` with `length` new blocks, in reth and in the model.
    fn fork_chain(&mut self, fork: u64, length: u64) -> eyre::Result<Vec<ExecutedBlock>> {
        let serials: Vec<u64> = (0..length).map(|_| self.next_serial()).collect();
        let specs: Vec<BlockSpec> = serials.iter().copied().map(serial_transfer_block).collect();
        // Every block of the fork becomes canonical with reth persisted at least this far.
        let persisted_before_the_fork = self.reth.persisted().number;
        let new = self.reth.reorg_with(fork + 1, &specs)?;
        self.model.truncate(fork);
        for (block, serial) in new.iter().zip(serials) {
            self.model.push(model_block(block, serial, persisted_before_the_fork));
        }
        Ok(new)
    }

    /// Hands `block` to the engine the way the ExEx does for a committed block, along the index
    /// or the execute path, and acknowledges it to reth.
    fn deliver(&mut self, block: &ExecutedBlock) -> eyre::Result<()> {
        let recovered = block.recovered_block();
        let (number, num_hash, parent) =
            (recovered.number(), recovered.num_hash(), recovered.parent_hash());
        let executes = self.config.executes(number);
        let parent_available =
            !executes || self.model.is_canonical(NumHash::new(number - 1, parent));
        let result = if executes {
            let recovered = Arc::clone(&block.recovered_block);
            self.call("execute_block", move |engine| engine.execute_block(&recovered))?
        } else {
            let block = block.clone();
            self.call("index_block", move |engine| index_executed_block(&engine, &block))?
        };
        let reply = match result {
            Ok(()) => Reply::Taken,
            Err(EngineError::ParentHashMismatch {
                block_number,
                expected_parent_hash,
                actual_parent_hash,
            }) if (block_number, actual_parent_hash) == (number, parent) => {
                self.log(format!(
                    "  the engine refused block {number}: its tip at {} is {expected_parent_hash}",
                    number - 1
                ));
                Reply::Refused { tip_hash: expected_parent_hash }
            }
            Err(err) => eyre::bail!("the engine refused block {number}: {err}"),
        };
        self.model.deliver_block(num_hash, parent, parent_available, reply)?;
        self.prune_bodies_through(number)
    }

    /// reth's bodies pruner following the ExEx's acknowledgement of a notification at `number`,
    /// which is the notification's tip whether or not the store holds it.
    fn prune_bodies_through(&mut self, number: u64) -> eyre::Result<()> {
        if self.config.body_pruning &&
            let Some(pruned) = self.reth.prune_bodies_through(number)?
        {
            self.log(format!("  reth pruned bodies through block {pruned}"));
        }
        Ok(())
    }

    /// `blocks` the way the ExEx hands a reorg to the engine, indexed or executed as the
    /// verification interval says.
    fn reorg_blocks(&self, blocks: &[ExecutedBlock]) -> Vec<ReorgBlock<Block>> {
        blocks
            .iter()
            .map(|block| reorg_block(block, self.config.executes(block.recovered_block().number())))
            .collect()
    }

    /// The fork point of a reorg or revert `depth` blocks deep, clamped to what reth holds in
    /// memory; none when nothing is.
    fn fork_point(&mut self, depth: u64) -> Option<u64> {
        let head = self.reth.head().number;
        let in_memory = head - self.reth.persisted().number;
        if in_memory == 0 {
            self.log("  reth holds no block in memory, nothing to reorg".to_owned());
            return None;
        }
        Some(head - depth.min(in_memory))
    }

    /// Checks the engine's reply to a reorg or revert whose first removed block is `first`. A
    /// refusal must name a block at or below the earliest the window held around the call; it
    /// ends the case, since reth has moved on without the store. An `Ok` must not have unwound
    /// at or below the window's earliest block.
    fn unwind_outcome(
        &mut self,
        what: &str,
        result: Result<(), EngineError>,
        first: u64,
        earliest_before: u64,
    ) -> eyre::Result<ControlFlow<()>> {
        match result {
            Ok(()) => {
                eyre::ensure!(
                    first > earliest_before,
                    "the engine applied a {what} from block {first}, at or below its earliest \
                     block {earliest_before}"
                );
                Ok(ControlFlow::Continue(()))
            }
            Err(EngineError::Storage(OpProofsStorageError::UnwindBeyondEarliest {
                unwind_block_number,
                earliest_block_number,
            })) => {
                let earliest_after = self.window()?.earliest.number;
                eyre::ensure!(
                    unwind_block_number == first &&
                        unwind_block_number <= earliest_block_number &&
                        (earliest_before..=earliest_after).contains(&earliest_block_number),
                    "the engine refused a {what} from block {first} citing block \
                     {unwind_block_number} and earliest block {earliest_block_number}, while the \
                     window's earliest was {earliest_before} before and {earliest_after} after"
                );
                self.log(format!(
                    "  the engine refused the {what} from block {first} at or below its earliest \
                     block {earliest_block_number}; reth moved on without the store"
                ));
                Ok(ControlFlow::Break(()))
            }
            Err(err) => eyre::bail!("the engine refused a valid {what} from block {first}: {err}"),
        }
    }

    /// Delivers a reorg to `new` and, if the engine took it, records it in the model.
    fn deliver_reorg(
        &mut self,
        new: &[ExecutedBlock],
        head_before: u64,
    ) -> eyre::Result<ControlFlow<()>> {
        let first = new[0].recovered_block().number();
        let last = new.last().expect("a reorg has a block").recovered_block().num_hash();
        let blocks = self.reorg_blocks(new);
        let earliest_before = self.window()?.earliest.number;
        let result = self.call("reorg", move |engine| engine.reorg(blocks))?;
        self.last_fork = Some(first - 1);
        let outcome = self.unwind_outcome("reorg", result, first, earliest_before)?;
        if outcome.is_continue() {
            self.model.reorg(first, last, head_before);
            self.prune_bodies_through(self.reth.head().number)?;
        }
        Ok(outcome)
    }

    fn reorg(&mut self, depth: u64, length: u64) -> eyre::Result<ControlFlow<()>> {
        let head_before = self.reth.head().number;
        let Some(fork) = self.fork_point(depth) else {
            return Ok(ControlFlow::Continue(()));
        };
        let new = self.fork_chain(fork, length)?;
        self.deliver_reorg(&new, head_before)
    }

    fn revert(&mut self, depth: u64) -> eyre::Result<ControlFlow<()>> {
        let head_before = self.reth.head().number;
        let Some(fork) = self.fork_point(depth) else {
            return Ok(ControlFlow::Continue(()));
        };
        let first = fork + 1;
        let to = self.model.block_with_parent(first);
        self.reth.revert_to(fork);
        self.model.truncate(fork);
        let earliest_before = self.window()?.earliest.number;
        let result = self.call("unwind", move |engine| engine.unwind(to))?;
        self.last_fork = Some(fork);
        let outcome = self.unwind_outcome("revert", result, first, earliest_before)?;
        if outcome.is_continue() {
            self.model.revert(first, head_before);
        }
        Ok(outcome)
    }

    /// reth commits `depth` blocks and reorgs them away before the ExEx processes their
    /// notifications, so the engine takes the stale blocks first and the reorg after.
    fn stale_commit(&mut self, depth: u64, length: u64) -> eyre::Result<ControlFlow<()>> {
        // reth persists all but its newest `memory_block_buffer_target` blocks, and only
        // in-memory blocks can be reorged away.
        let depth = depth.min(self.reth.persistence.memory_block_buffer_target);
        let fork = self.reth.head().number;
        let stale: Vec<ExecutedBlock> =
            (0..depth).map(|_| self.commit_block()).collect::<eyre::Result<_>>()?;
        let head_before = self.reth.head().number;
        let new = self.fork_chain(fork, length)?;
        for block in &stale {
            self.deliver(block)?;
        }
        self.deliver_reorg(&new, head_before)
    }

    pub(super) fn flush(&mut self) -> eyre::Result<()> {
        self.call("flush", |engine| engine.flush())?;
        let latest = self.window()?.latest.number;
        self.model.note_persisted(latest);
        Ok(())
    }

    /// Arms a failure for the next write, commits enough blocks to start a save, and flushes
    /// twice: once for a save already in flight and once for the one that then starts and fails.
    fn fail_next_save(&mut self) -> eyre::Result<()> {
        self.store().fail_next_write();
        for _ in 0..self.config.cadence.persistence {
            let block = self.commit_block()?;
            self.deliver(&block)?;
        }
        self.flush()?;
        self.flush()?;
        if self.store().disarm_next_write() {
            self.log("  no save started, the injected failure is withdrawn".to_owned());
        }
        Ok(())
    }

    /// A one-block reorg on a parent no block has, at a fork point the engine knows the hash of,
    /// which it must refuse, or skip while its tip is still below the fork point.
    fn bogus_reorg(&mut self, offset: u64) -> eyre::Result<()> {
        let points = self.model.known_fork_points();
        if points.is_empty() {
            self.log("  the engine knows no fork point for certain, nothing to refuse".to_owned());
            return Ok(());
        }
        let fork = points[(offset % points.len() as u64) as usize];
        let first = fork + 1;
        let bogus = reorg_block_without_data(BlockWithParent::new(
            UNRELATED_PARENT,
            NumHash::new(first, B256::repeat_byte(0xB1)),
        ));
        let stored_fork_hash = self.model.num_hash(fork).hash;
        let result = self.call("reorg", move |engine| engine.reorg(vec![bogus]))?;
        let refused = match result {
            Ok(()) => false,
            Err(EngineError::ParentHashMismatch {
                block_number,
                expected_parent_hash,
                actual_parent_hash,
            }) => {
                eyre::ensure!(
                    (block_number, expected_parent_hash, actual_parent_hash) ==
                        (first, stored_fork_hash, UNRELATED_PARENT),
                    "the engine refused the reorg at block {first} with a mismatch at block \
                     {block_number} expecting {expected_parent_hash} and seeing \
                     {actual_parent_hash}, not the stored block {fork} {stored_fork_hash}"
                );
                true
            }
            Err(EngineError::Storage(OpProofsStorageError::UnwindBeyondEarliest {
                unwind_block_number,
                earliest_block_number,
            })) => {
                eyre::ensure!(
                    unwind_block_number == first && first <= earliest_block_number,
                    "the engine refused the reorg at block {first} citing block \
                     {unwind_block_number} and earliest block {earliest_block_number}"
                );
                true
            }
            Err(err) => eyre::bail!(
                "unexpected reply to a reorg at block {first} not building on the stored block \
                 {fork}: {err}"
            ),
        };
        self.model.bogus_reorg(first, refused)?;
        self.log(format!(
            "  the engine {} the reorg at block {first} on a parent no block has",
            if refused { "refused" } else { "skipped" }
        ));
        Ok(())
    }

    /// reth and the node stop and restart: reth loses what it held only in memory, the engine
    /// its buffer (its save in flight completes, as the engine thread drains it on drop), and
    /// both come back the way a restart brings them back today: a fresh engine over the reopened
    /// store, a sync target at reth's head, and reth's rebuilt blocks as live notifications.
    fn crash(&mut self, rebuild: Rebuild) -> eyre::Result<()> {
        let persisted = self.reth.persisted().number;
        let lost = self.model.serials_above(persisted);
        self.log(match lost.len() {
            0 => format!("  crash: reth loses no block ({rebuild:?})"),
            count => format!(
                "  crash: reth loses blocks {}..={} and rebuilds them ({rebuild:?})",
                persisted + 1,
                persisted + count as u64
            ),
        });
        self.node = None;
        self.reth.crash_and_restart()?;
        let store = StoreWithHooks::new(Arc::new(MdbxProofsStorage::new(self.store_dir.path())?));
        let engine = Self::spawn_engine(&self.reth, &store, self.config);
        self.node = Some(ProofsNode { engine, store });
        let tip = self.call("tip", |engine| engine.tip())??;
        self.model.restart(tip)?;
        self.log(format!(
            "  restart: the store serves through block {}, reth's head is {persisted}",
            tip.number
        ));
        self.engine().sync_to(persisted)?;
        self.model.sync_to(persisted);
        match rebuild {
            Rebuild::Same => {
                for serial in lost {
                    let block = self.reth.commit(&serial_transfer_block(serial))?;
                    let number = block.recovered_block().number();
                    eyre::ensure!(
                        block.recovered_block().hash() == self.model.block(number).hash,
                        "reth rebuilt block {number} differently from the chain it lost"
                    );
                    self.deliver(&block)?;
                }
            }
            Rebuild::Different => {
                self.model.truncate(persisted);
                for _ in lost {
                    let block = self.commit_block()?;
                    self.deliver(&block)?;
                }
            }
        }
        Ok(())
    }
}

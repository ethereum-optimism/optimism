//! Drives the engine's reorg and unwind tasks at boundaries a random chain rarely reaches: a
//! reorg whose first block does not build on the stored fork block, a target at or below the
//! window's earliest block, a revert that only touches the buffer, and a reorg or a revert that
//! arrives while a save is in flight.
//!
//! Every case commits the same transfer chain to reth, indexes it under the case's flush cadence
//! and settles, then acts. The engine's tip is read through the parent check of a probe block,
//! and the store's persisted history is held to a fresh store indexed over reth's final canonical
//! chain once a save has carried the case's blocks to disk.

use super::{
    Backend, BlockSpec, FlushCadence, TxSpec, WHOLE_WINDOW, chain_spec_with_address,
    create_block_from_spec, create_mdbx_proofs_storage, initialize_storage, public_key_to_address,
    simulated_reth::{
        RethPersistence, SimulatedReth, UNRELATED_PARENT, fixed_signer, index_executed_block,
        reorg_block, reorg_block_without_data,
    },
    store_hooks::{HeldWrite, StoreWithHooks},
    store_view::{BlockView, ProofTarget, assert_same_view, proof_window},
};
use alloy_eips::{NumHash, eip1898::BlockWithParent};
use alloy_primitives::{Address, B256, U256};
use reth_chain_state::ExecutedBlock;
use reth_chainspec::ChainSpec;
use reth_ethereum_primitives::Block;
use reth_optimism_trie::{
    InMemoryProofsStorage, OpProofsProviderRO, OpProofsStorageError, OpProofsStore,
    ProofWindowRange,
    engine::{EngineError, EngineHandle, ReorgBlock},
};
use reth_trie_common::{HashedPostStateSorted, updates::TrieUpdatesSorted};
use rstest::rstest;
use secp256k1::Keypair;
use serial_test::serial;
use std::{collections::BTreeMap, ops::RangeInclusive, sync::Arc, thread};

/// Blocks every case commits and settles before it acts.
const CHAIN_LENGTH: u64 = 7;

/// Blocks a case may hold at most, which bounds the flushes a settle needs and the recipients.
const MAX_BLOCKS_PER_CASE: u64 = 2 * CHAIN_LENGTH;

/// reth's persistence threshold, above any chain here so that reth persists nothing on its own:
/// a case persists reth through the block its window starts at, which keeps `reorg_with` free
/// to fork the chain anywhere above.
const RETH_PERSISTENCE_THRESHOLD: u64 = 2 * MAX_BLOCKS_PER_CASE;

/// Blocks the pruned window keeps, so that pruning moves the earliest block above the anchor
/// under either cadence.
const PRUNED_WINDOW: u64 = 3;

const fn recipient(number: u64) -> Address {
    Address::repeat_byte(0x40 + number as u8)
}

const fn fork_recipient(number: u64) -> Address {
    Address::repeat_byte(0x80 + number as u8)
}

/// Block `number` of the chain: `number` wei to the recipient only this block pays.
fn chain_block(number: u64) -> BlockSpec {
    BlockSpec::new(vec![TxSpec::transfer(recipient(number), U256::from(number))])
}

/// Blocks `numbers` of a fork: `number` wei each to a recipient the chain never pays.
fn fork_blocks(numbers: RangeInclusive<u64>) -> Vec<BlockSpec> {
    numbers
        .map(|number| {
            BlockSpec::new(vec![TxSpec::transfer(fork_recipient(number), U256::from(number))])
        })
        .collect()
}

/// Every address the chain or a fork can touch, none of which has storage.
fn proof_targets(sender: Address) -> Vec<ProofTarget> {
    ProofTarget::accounts([sender, Address::ZERO].into_iter().chain(
        (1..=MAX_BLOCKS_PER_CASE).flat_map(|number| [recipient(number), fork_recipient(number)]),
    ))
}

/// The disk tip once the engine has saved what its cadence lets it save: `buffered` blocks went
/// into the buffer above `base`, and the first save starts at the persistence threshold with
/// exactly that many blocks. Below twice the threshold no second save starts, so the outcome does
/// not depend on when the first save completes.
const fn settled_disk_tip(cadence: FlushCadence, base: u64, buffered: u64) -> u64 {
    assert!(
        cadence.persistence == 1 || buffered < 2 * cadence.persistence,
        "a second save would race the commits"
    );
    base + buffered - buffered % cadence.persistence
}

/// How the engine receives the blocks of a new chain.
#[derive(Clone, Copy, Debug)]
enum ReorgPath {
    Execute,
    Index,
    /// Executes every other block, starting with the first, as the ExEx does when the
    /// verification interval falls inside the new chain.
    Mixed,
}

impl ReorgPath {
    /// The way the ExEx hands the `position`th block of a reorg to the engine along this path.
    fn reorg_block(self, position: usize, block: &ExecutedBlock) -> ReorgBlock<Block> {
        let execute = match self {
            Self::Execute => true,
            Self::Index => false,
            Self::Mixed => position.is_multiple_of(2),
        };
        reorg_block(block, execute)
    }
}

/// `blocks` the way the ExEx hands a reorg to the engine along `path`.
fn reorg_blocks(blocks: &[ExecutedBlock], path: ReorgPath) -> Vec<ReorgBlock<Block>> {
    blocks.iter().enumerate().map(|(position, block)| path.reorg_block(position, block)).collect()
}

/// Which engine task unwinds.
#[derive(Clone, Copy, Debug)]
enum UnwindingTask {
    Reorg,
    Unwind,
}

impl UnwindingTask {
    fn run(self, engine: &EngineHandle<Block>, to: BlockWithParent) -> Result<(), EngineError> {
        match self {
            Self::Reorg => engine.reorg(vec![reorg_block_without_data(to)]),
            Self::Unwind => engine.unwind(to),
        }
    }
}

/// Where an unwind target sits relative to the proof window.
#[derive(Clone, Copy, Debug)]
enum UnwindTarget {
    /// The init anchor, block 0: the earliest block of a whole window, below a pruned one.
    Anchor,
    BelowEarliest,
    Earliest,
}

impl UnwindTarget {
    const fn block_number(self, earliest: u64) -> u64 {
        match self {
            Self::Anchor => 0,
            Self::BelowEarliest => {
                earliest.checked_sub(1).expect("the window must have been pruned past the anchor")
            }
            Self::Earliest => earliest,
        }
    }
}

/// What the store serves at one moment: its window and every persisted view in it.
struct DiskSnapshot {
    views: BTreeMap<u64, BlockView>,
    window: ProofWindowRange,
}

/// reth holding the chain, and the engine indexing it into the store under test.
struct Scenario<S> {
    cadence: FlushCadence,
    chain_spec: Arc<ChainSpec>,
    engine: EngineHandle<Block>,
    gated_store: StoreWithHooks<S>,
    key_pair: Keypair,
    reth: SimulatedReth,
    store: S,
    targets: Vec<ProofTarget>,
    window: u64,
}

impl<S: OpProofsStore + Clone + 'static> Scenario<S> {
    /// Commits the chain to reth, indexes it into a store pruned to `window` blocks, and settles.
    fn new(
        make_store: impl FnOnce() -> S,
        cadence: FlushCadence,
        window: u64,
    ) -> eyre::Result<Self> {
        let key_pair = fixed_signer();
        let sender = public_key_to_address(key_pair.public_key());
        let chain_spec = chain_spec_with_address(sender);
        let persistence = RethPersistence::threshold(RETH_PERSISTENCE_THRESHOLD);
        let reth = SimulatedReth::new(chain_spec.clone(), key_pair, persistence)?;
        let store = make_store();
        initialize_storage(store.clone(), &reth.factory, reth.genesis())?;
        let gated_store = StoreWithHooks::new(store.clone());
        let engine = Self::spawn_engine(&reth, &gated_store, cadence, window);
        let mut scenario = Self {
            cadence,
            chain_spec,
            engine,
            gated_store,
            key_pair,
            reth,
            store,
            targets: proof_targets(sender),
            window,
        };
        let settled_tip = settled_disk_tip(cadence, 0, CHAIN_LENGTH);
        scenario.commit_blocks_toward_a_save_at(CHAIN_LENGTH, settled_tip)?;
        let disk_tip = scenario.settle()?;
        assert_eq!(disk_tip.number, settled_tip, "settled disk tip");
        Ok(scenario)
    }

    fn spawn_engine(
        reth: &SimulatedReth,
        gated_store: &StoreWithHooks<S>,
        cadence: FlushCadence,
        window: u64,
    ) -> EngineHandle<Block> {
        reth.spawn_engine_with_window(
            gated_store.clone(),
            cadence.persistence,
            cadence.backpressure,
            window,
        )
    }

    /// Replaces the engine with one started over the same store, as a restart does.
    fn restart_engine(self) -> Self {
        let Self {
            cadence,
            chain_spec,
            engine,
            gated_store,
            key_pair,
            reth,
            store,
            targets,
            window,
        } = self;
        drop(engine);
        let engine = Self::spawn_engine(&reth, &gated_store, cadence, window);
        Self { cadence, chain_spec, engine, gated_store, key_pair, reth, store, targets, window }
    }

    /// Commits `count` chain blocks to reth, persists reth through the block the save landing at
    /// `save_tip` prunes the window to, and indexes the blocks. reth must be persisted exactly
    /// through that block: any lower and a pruner holding the window back at reth's persisted
    /// head prunes less than one that does not; any higher and the reference store, which starts
    /// at reth's database tip, cannot serve the earliest block.
    fn commit_blocks_toward_a_save_at(&mut self, count: u64, save_tip: u64) -> eyre::Result<()> {
        let blocks =
            (0..count).map(|_| self.commit_chain_block()).collect::<eyre::Result<Vec<_>>>()?;
        self.reth.persist_through(save_tip.saturating_sub(self.window))?;
        for block in &blocks {
            index_executed_block(&self.engine, block)?;
        }
        Ok(())
    }

    /// Commits the next block of the chain to reth.
    fn commit_chain_block(&mut self) -> eyre::Result<ExecutedBlock> {
        let number = self.reth.head().number + 1;
        self.reth.commit(&chain_block(number))
    }

    /// Commits enough blocks for a save to carry everything buffered to disk, and settles.
    fn commit_through_a_save(&mut self) -> eyre::Result<()> {
        let save_tip = self.settle()?.number + self.cadence.persistence;
        self.commit_blocks_toward_a_save_at(self.cadence.persistence, save_tip)?;
        self.settle()?;
        Ok(())
    }

    /// Flushes until nothing the engine could still save is unsaved, and returns the disk tip.
    /// A flush waits for the save in flight, and the engine starts the next save only after
    /// replying, so one flush per block the case could have buffered drains them all.
    fn settle(&self) -> eyre::Result<NumHash> {
        for _ in 0..=MAX_BLOCKS_PER_CASE {
            self.engine.flush();
        }
        Ok(self.proof_window()?.latest)
    }

    fn proof_window(&self) -> eyre::Result<ProofWindowRange> {
        proof_window(&self.store)
    }

    /// Arms a hold on the engine's next write, the save the following commits start.
    fn hold_next_save(&self) -> HeldWrite {
        self.gated_store.hold_next_write()
    }

    /// Commits blocks until the buffer reaches the persistence threshold and the save it starts
    /// is inside `hold`.
    fn commit_until_a_save_is_in_flight(&mut self, hold: &HeldWrite) -> eyre::Result<()> {
        let save_tip = self.proof_window()?.latest.number + self.cadence.persistence;
        let count = save_tip - self.reth.head().number;
        self.commit_blocks_toward_a_save_at(count, save_tip)?;
        hold.wait_until_entered();
        Ok(())
    }

    /// The canonical block `number` as an unwind target: the target's parent is the block below.
    fn canonical_target(&self, number: u64) -> BlockWithParent {
        let parent = match number {
            0 => B256::ZERO,
            _ => self.reth.num_hash(number - 1).hash,
        };
        BlockWithParent::new(parent, self.reth.num_hash(number))
    }

    /// A one-block reorg at `number` whose block builds on a parent no block here has.
    fn reorg_on_unrelated_parent(&self, number: u64, path: ReorgPath) -> Vec<ReorgBlock<Block>> {
        let block = match path {
            ReorgPath::Execute | ReorgPath::Mixed => {
                ReorgBlock::Execute(Arc::new(create_block_from_spec(
                    &BlockSpec::empty(),
                    number,
                    UNRELATED_PARENT,
                    &self.chain_spec,
                    self.key_pair,
                    &mut 0,
                )))
            }
            ReorgPath::Index => reorg_block_without_data(BlockWithParent::new(
                UNRELATED_PARENT,
                NumHash::new(number, B256::repeat_byte(0xB1)),
            )),
        };
        vec![block]
    }

    fn disk_snapshot(&self) -> eyre::Result<DiskSnapshot> {
        let window = self.proof_window()?;
        let views = (window.earliest.number..=window.latest.number)
            .map(|number| Ok((number, BlockView::read(&self.store, number, &self.targets)?)))
            .collect::<eyre::Result<_>>()?;
        Ok(DiskSnapshot { views, window })
    }

    /// The store's window and every persisted view are as in `before`.
    fn assert_disk_unchanged(&self, before: &DiskSnapshot) -> eyre::Result<()> {
        assert_eq!(self.proof_window()?, before.window, "proof window");
        for (number, expected) in &before.views {
            let actual = BlockView::read(&self.store, *number, &self.targets)?;
            assert_same_view(*number, "under test", expected, &actual)?;
        }
        Ok(())
    }

    /// The engine's tip is `expected`, read through the parent check: a block numbered one above
    /// it on an unrelated parent is refused with the tip's hash as the expected parent.
    fn assert_tip(&self, expected: NumHash) {
        let probe = BlockWithParent::new(
            UNRELATED_PARENT,
            NumHash::new(expected.number + 1, B256::repeat_byte(0xBB)),
        );
        let err = self
            .engine
            .index_block(probe, TrieUpdatesSorted::default(), HashedPostStateSorted::default())
            .expect_err("the probe does not build on the tip");
        let EngineError::ParentHashMismatch { block_number, expected_parent_hash, .. } = err else {
            panic!("expected a parent-hash mismatch, got {err:?}");
        };
        assert_eq!((block_number, expected_parent_hash), (expected.number + 1, expected.hash));
    }

    /// The store's persisted history reaches `through` and is what a fresh store indexed over
    /// reth's canonical chain serves, with proofs that hash to reth's state roots.
    fn assert_disk_matches_reth_through(&self, through: u64) -> eyre::Result<()> {
        let window = self.proof_window()?;
        eyre::ensure!(
            window.latest.number >= through,
            "the disk tip {} is below block {through}",
            window.latest.number
        );
        eyre::ensure!(
            self.reth.locate(window.latest).canonical,
            "the disk tip {:?} is not canonical in reth",
            window.latest
        );
        let database_tip = self.reth.persisted().number;
        eyre::ensure!(
            window.earliest.number >= database_tip,
            "the earliest block {} is below reth's database tip {database_tip}, where the \
             reference store starts",
            window.earliest.number
        );
        let reference = self.reference_store()?;
        for number in window.earliest.number..=window.latest.number {
            let expected = BlockView::read(&reference, number, &self.targets)?;
            let actual = BlockView::read(&self.store, number, &self.targets)?;
            assert_same_view(number, "under test", &expected, &actual)?;
            actual.verify_proofs(number, self.reth.state_root(number))?;
        }
        Ok(())
    }

    /// A fresh store initialized at reth's database tip, the only block whose trie reth's
    /// database holds, and indexed over reth's canonical chain above it.
    fn reference_store(&self) -> eyre::Result<InMemoryProofsStorage> {
        let anchor = self.reth.persisted();
        let store = InMemoryProofsStorage::new();
        initialize_storage(store.clone(), &self.reth.factory, anchor)?;
        let engine = self.reth.spawn_engine(store.clone(), 1, 2);
        let head = self.reth.head().number;
        for number in anchor.number + 1..=head {
            index_executed_block(&engine, self.reth.executed_block(number))?;
        }
        for _ in 0..=head {
            engine.flush();
        }
        eyre::ensure!(
            store.provider_ro()?.get_latest_block()? == self.reth.head(),
            "the reference store holds reth's whole chain"
        );
        Ok(store)
    }
}

fn run_reorg_case<S: OpProofsStore + Clone + 'static>(
    make_store: impl FnOnce() -> S,
    cadence: FlushCadence,
    path: ReorgPath,
    fork: u64,
) -> eyre::Result<()> {
    let mut scenario = Scenario::new(make_store, cadence, WHOLE_WINDOW)?;
    let new_chain = scenario.reth.reorg_with(fork + 1, &fork_blocks(fork + 1..=fork + 3))?;

    scenario.engine.reorg(reorg_blocks(&new_chain, path))?;

    scenario.assert_tip(scenario.reth.head());
    scenario.commit_through_a_save()?;
    scenario.assert_disk_matches_reth_through(fork + 3)
}

/// A reorg replaces the stored blocks above the fork with the new chain, indexed, executed or
/// both, whether the fork is buffered (block 6 under the slower cadence) or on disk.
#[rstest]
#[case::indexed_from_block_six(ReorgPath::Index, 6)]
#[case::executed_from_block_three(ReorgPath::Execute, 3)]
#[case::mixed_from_block_four(ReorgPath::Mixed, 4)]
#[serial]
fn reorg_replaces_the_stored_blocks_above_the_fork(
    #[values(Backend::InMemory, Backend::Mdbx)] backend: Backend,
    #[values(FlushCadence::EVERY_BLOCK, FlushCadence::EVERY_FIFTH_BLOCK)] cadence: FlushCadence,
    #[case] path: ReorgPath,
    #[case] fork: u64,
) -> eyre::Result<()> {
    over_backend!(backend, run_reorg_case(cadence, path, fork))
}

fn run_wrong_fork_parent_case<S: OpProofsStore + Clone + 'static>(
    make_store: impl FnOnce() -> S,
    cadence: FlushCadence,
    path: ReorgPath,
    fork: u64,
) -> eyre::Result<()> {
    let mut scenario = Scenario::new(make_store, cadence, WHOLE_WINDOW)?;
    let before = scenario.disk_snapshot()?;

    let err = scenario
        .engine
        .reorg(scenario.reorg_on_unrelated_parent(fork + 1, path))
        .expect_err("the reorg does not build on the stored fork block");

    let EngineError::ParentHashMismatch { block_number, expected_parent_hash, actual_parent_hash } =
        err
    else {
        panic!("expected a parent-hash mismatch, got {err:?}");
    };
    assert_eq!(
        (block_number, expected_parent_hash, actual_parent_hash),
        (fork + 1, scenario.reth.num_hash(fork).hash, UNRELATED_PARENT)
    );
    scenario.assert_disk_unchanged(&before)?;
    scenario.assert_tip(scenario.reth.head());
    scenario.commit_through_a_save()?;
    scenario.assert_disk_matches_reth_through(CHAIN_LENGTH)
}

/// A reorg whose first block does not build on the stored fork block fails with the stored hash
/// and changes nothing, whether the engine knows that block from its buffer (block 6 under the
/// slower cadence), from a save it made, or as the window's earliest block.
#[rstest]
#[case::indexed_on_block_six(ReorgPath::Index, 6)]
#[case::executed_on_block_two(ReorgPath::Execute, 2)]
#[case::indexed_on_the_anchor(ReorgPath::Index, 0)]
#[serial]
fn reorg_not_building_on_the_stored_fork_block_fails_and_changes_nothing(
    #[values(Backend::InMemory, Backend::Mdbx)] backend: Backend,
    #[values(FlushCadence::EVERY_BLOCK, FlushCadence::EVERY_FIFTH_BLOCK)] cadence: FlushCadence,
    #[case] path: ReorgPath,
    #[case] fork: u64,
) -> eyre::Result<()> {
    over_backend!(backend, run_wrong_fork_parent_case(cadence, path, fork))
}

fn run_restarted_engine_case<S: OpProofsStore + Clone + 'static>(
    make_store: impl FnOnce() -> S,
) -> eyre::Result<()> {
    let fork = 2;
    let scenario = Scenario::new(make_store, FlushCadence::EVERY_BLOCK, WHOLE_WINDOW)?;
    let before = scenario.disk_snapshot()?;
    let scenario = scenario.restart_engine();

    let err = scenario
        .engine
        .reorg(scenario.reorg_on_unrelated_parent(fork + 1, ReorgPath::Index))
        .expect_err("the reorg does not build on the stored fork block");

    assert!(matches!(err, EngineError::ParentHashMismatch { .. }), "got {err:?}");
    scenario.assert_disk_unchanged(&before)
}

/// A restarted engine knows no hash below its store's latest block, so a reorg onto a wrong
/// fork block there goes through unverified; checking it needs the store to keep every block's
/// hash.
#[rstest]
#[ignore = "a restarted engine cannot verify a fork below its disk tip; the store keeps no per-block hashes (#23391, tracked under #23385)"]
#[serial]
fn restarted_engine_rejects_a_reorg_not_building_on_a_block_below_its_disk_tip(
    #[values(Backend::InMemory, Backend::Mdbx)] backend: Backend,
) -> eyre::Result<()> {
    over_backend!(backend, run_restarted_engine_case())
}

fn run_below_window_case<S: OpProofsStore + Clone + 'static>(
    make_store: impl FnOnce() -> S,
    cadence: FlushCadence,
    task: UnwindingTask,
    target: UnwindTarget,
    window: u64,
) -> eyre::Result<()> {
    let scenario = Scenario::new(make_store, cadence, window)?;
    let before = scenario.disk_snapshot()?;
    let earliest = before.window.earliest.number;
    let number = target.block_number(earliest);

    let err = task
        .run(&scenario.engine, scenario.canonical_target(number))
        .expect_err("the unwind would remove the window's earliest block");

    assert!(
        matches!(
            err,
            EngineError::Storage(OpProofsStorageError::UnwindBeyondEarliest {
                unwind_block_number,
                earliest_block_number,
            }) if unwind_block_number == number && earliest_block_number == earliest
        ),
        "got {err:?}"
    );
    scenario.assert_disk_unchanged(&before)?;
    scenario.assert_tip(scenario.reth.head());
    Ok(())
}

/// An unwind or a reorg whose target is at or below the window's earliest block, the init anchor
/// or a block pruning moved the window past, fails before touching the store or the buffer.
#[rstest]
#[case::unwind_at_the_earliest_block(UnwindingTask::Unwind, UnwindTarget::Earliest, PRUNED_WINDOW)]
#[case::reorg_below_the_earliest_block(
    UnwindingTask::Reorg,
    UnwindTarget::BelowEarliest,
    PRUNED_WINDOW
)]
#[case::reorg_at_the_anchor_below_the_window(
    UnwindingTask::Reorg,
    UnwindTarget::Anchor,
    PRUNED_WINDOW
)]
#[case::unwind_at_the_anchor_as_earliest_block(
    UnwindingTask::Unwind,
    UnwindTarget::Anchor,
    WHOLE_WINDOW
)]
#[serial]
fn unwind_or_reorg_at_or_below_the_earliest_block_fails_and_changes_nothing(
    #[values(Backend::InMemory, Backend::Mdbx)] backend: Backend,
    #[values(FlushCadence::EVERY_BLOCK, FlushCadence::EVERY_FIFTH_BLOCK)] cadence: FlushCadence,
    #[case] task: UnwindingTask,
    #[case] target: UnwindTarget,
    #[case] window: u64,
) -> eyre::Result<()> {
    over_backend!(backend, run_below_window_case(cadence, task, target, window))
}

fn run_unwind_to_above_earliest_case<S: OpProofsStore + Clone + 'static>(
    make_store: impl FnOnce() -> S,
    cadence: FlushCadence,
) -> eyre::Result<()> {
    let mut scenario = Scenario::new(make_store, cadence, PRUNED_WINDOW)?;
    let earliest = scenario.proof_window()?.earliest;

    scenario.engine.unwind(scenario.canonical_target(earliest.number + 1))?;

    assert_eq!(scenario.proof_window()?, ProofWindowRange { earliest, latest: earliest });
    scenario.assert_tip(earliest);
    for number in earliest.number + 1..=CHAIN_LENGTH {
        index_executed_block(&scenario.engine, scenario.reth.executed_block(number))?;
    }
    scenario.commit_through_a_save()?;
    scenario.assert_disk_matches_reth_through(CHAIN_LENGTH)
}

/// An unwind to the block right above the earliest leaves the window at that one block, from
/// which the engine indexes the chain again.
#[rstest]
#[serial]
fn unwind_to_the_block_above_the_earliest_keeps_only_the_earliest(
    #[values(Backend::InMemory, Backend::Mdbx)] backend: Backend,
    #[values(FlushCadence::EVERY_BLOCK, FlushCadence::EVERY_FIFTH_BLOCK)] cadence: FlushCadence,
) -> eyre::Result<()> {
    over_backend!(backend, run_unwind_to_above_earliest_case(cadence))
}

fn run_unwind_at_a_block_the_save_prunes_case<S: OpProofsStore + Clone + 'static>(
    make_store: impl FnOnce() -> S,
    cadence: FlushCadence,
) -> eyre::Result<()> {
    let mut scenario = Scenario::new(make_store, cadence, PRUNED_WINDOW)?;
    let earliest_before = scenario.proof_window()?.earliest.number;
    let hold = scenario.hold_next_save();
    scenario.commit_until_a_save_is_in_flight(&hold)?;
    let head = scenario.reth.head();
    // The save carries the chain through the head and prunes the window to end at the head.
    let target = head.number - PRUNED_WINDOW;
    assert!(target > earliest_before, "the target is inside the window until the save prunes it");
    let to = scenario.canonical_target(target);
    let engine = scenario.engine.clone();

    let unwinding = thread::spawn(move || engine.unwind(to));
    hold.release();
    let err = unwinding
        .join()
        .expect("the unwind thread finishes")
        .expect_err("the save moved the earliest block up to the target");

    assert!(
        matches!(
            err,
            EngineError::Storage(OpProofsStorageError::UnwindBeyondEarliest {
                unwind_block_number,
                earliest_block_number,
            }) if unwind_block_number == target && earliest_block_number == target
        ),
        "got {err:?}"
    );
    assert_eq!(
        scenario.proof_window()?,
        ProofWindowRange { earliest: scenario.reth.num_hash(target), latest: head }
    );
    scenario.assert_tip(head);
    scenario.assert_disk_matches_reth_through(head.number)
}

/// An unwind whose target the in-flight save's pruning moves the window past is refused against
/// the window as the save leaves it, before the persistence service sees the unwind.
#[rstest]
#[serial]
fn unwind_at_a_block_the_in_flight_save_prunes_fails_after_the_save(
    #[values(Backend::InMemory, Backend::Mdbx)] backend: Backend,
    #[values(FlushCadence::EVERY_BLOCK, FlushCadence::EVERY_FIFTH_BLOCK)] cadence: FlushCadence,
) -> eyre::Result<()> {
    over_backend!(backend, run_unwind_at_a_block_the_save_prunes_case(cadence))
}

fn run_buffered_revert_case<S: OpProofsStore + Clone + 'static>(
    make_store: impl FnOnce() -> S,
    revert_from: u64,
) -> eyre::Result<()> {
    let cadence = FlushCadence::EVERY_FIFTH_BLOCK;
    let mut scenario = Scenario::new(make_store, cadence, WHOLE_WINDOW)?;
    let before = scenario.disk_snapshot()?;
    assert!(before.window.latest.number < revert_from, "the reverted blocks are buffered");

    scenario.engine.unwind(scenario.canonical_target(revert_from))?;

    scenario.assert_disk_unchanged(&before)?;
    scenario.assert_tip(scenario.reth.num_hash(revert_from - 1));
    let new_chain =
        scenario.reth.reorg_with(revert_from, &fork_blocks(revert_from..=CHAIN_LENGTH))?;
    for block in &new_chain {
        index_executed_block(&scenario.engine, block)?;
    }
    scenario.commit_through_a_save()?;
    scenario.assert_disk_matches_reth_through(CHAIN_LENGTH)
}

/// A revert of blocks still in the buffer unwinds the buffer, leaves the disk untouched and
/// moves the tip to the block below the revert. Under the every-block cadence no block stays
/// buffered outside a save in flight, which the in-flight case covers.
#[rstest]
#[serial]
fn revert_of_buffered_blocks_unwinds_the_buffer_and_leaves_the_disk_untouched(
    #[values(Backend::InMemory, Backend::Mdbx)] backend: Backend,
    #[values(6, 7)] revert_from: u64,
) -> eyre::Result<()> {
    over_backend!(backend, run_buffered_revert_case(revert_from))
}

fn run_revert_during_save_case<S: OpProofsStore + Clone + 'static>(
    make_store: impl FnOnce() -> S,
    cadence: FlushCadence,
) -> eyre::Result<()> {
    let mut scenario = Scenario::new(make_store, cadence, WHOLE_WINDOW)?;
    let before = scenario.disk_snapshot()?;
    let disk_tip = before.window.latest;
    let hold = scenario.hold_next_save();
    scenario.commit_until_a_save_is_in_flight(&hold)?;
    let revert = scenario.canonical_target(disk_tip.number + 1);
    let engine = scenario.engine.clone();

    let reverting = thread::spawn(move || engine.unwind(revert));
    hold.release();
    reverting.join().expect("the revert thread finishes")?;

    scenario.assert_disk_unchanged(&before)?;
    scenario.assert_tip(disk_tip);
    let new_chain = scenario
        .reth
        .reorg_with(disk_tip.number + 1, &fork_blocks(disk_tip.number + 1..=disk_tip.number + 3))?;
    for block in &new_chain {
        index_executed_block(&scenario.engine, block)?;
    }
    scenario.commit_through_a_save()?;
    scenario.assert_disk_matches_reth_through(disk_tip.number + 3)
}

/// A revert of every block above the disk tip, sent while their save is in flight, ends with the
/// disk as it was and the tip at the disk tip: the engine either drains the save before the
/// unwind or sees its completion first, and both orders must end the same.
#[rstest]
#[serial]
fn revert_sent_during_an_in_flight_save_ends_with_the_disk_as_it_was(
    #[values(Backend::InMemory, Backend::Mdbx)] backend: Backend,
    #[values(FlushCadence::EVERY_BLOCK, FlushCadence::EVERY_FIFTH_BLOCK)] cadence: FlushCadence,
) -> eyre::Result<()> {
    over_backend!(backend, run_revert_during_save_case(cadence))
}

fn run_reorg_during_save_case<S: OpProofsStore + Clone + 'static>(
    make_store: impl FnOnce() -> S,
    cadence: FlushCadence,
    path: ReorgPath,
) -> eyre::Result<()> {
    let mut scenario = Scenario::new(make_store, cadence, WHOLE_WINDOW)?;
    let fork = scenario.proof_window()?.latest.number - 2;
    let hold = scenario.hold_next_save();
    scenario.commit_until_a_save_is_in_flight(&hold)?;
    let new_chain = scenario.reth.reorg_with(fork + 1, &fork_blocks(fork + 1..=fork + 3))?;
    let blocks = reorg_blocks(&new_chain, path);
    let engine = scenario.engine.clone();

    let reorging = thread::spawn(move || engine.reorg(blocks));
    hold.release();
    reorging.join().expect("the reorg thread finishes")?;

    scenario.assert_tip(scenario.reth.head());
    let disk_tip = scenario.settle()?;
    assert_eq!(disk_tip, scenario.reth.num_hash(settled_disk_tip(cadence, fork, 3)));
    scenario.assert_disk_matches_reth_through(disk_tip.number)?;
    scenario.commit_through_a_save()?;
    scenario.assert_disk_matches_reth_through(fork + 3)
}

/// A reorg from a fork on disk, sent while a save of the blocks above it is in flight, ends with
/// the disk unwound to the fork and the new chain buffered, so that the disk and the buffer agree
/// with reth whichever of the save and the reorg the engine handles first.
#[rstest]
#[serial]
fn reorg_sent_during_an_in_flight_save_ends_consistent_with_reth(
    #[values(Backend::InMemory, Backend::Mdbx)] backend: Backend,
    #[values(FlushCadence::EVERY_BLOCK, FlushCadence::EVERY_FIFTH_BLOCK)] cadence: FlushCadence,
    #[values(ReorgPath::Index, ReorgPath::Execute)] path: ReorgPath,
) -> eyre::Result<()> {
    over_backend!(backend, run_reorg_during_save_case(cadence, path))
}

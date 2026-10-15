//! Drives the engine's reorg and unwind tasks at the boundaries the model-based test may not
//! reach: a reorg whose first block does not build on the stored fork block, a target at or
//! below the window's earliest block, a revert that only touches the buffer, and a reorg or a
//! revert that arrives while a save is in flight.
//!
//! Every case commits the same transfer chain to reth, indexes it under the case's flush cadence
//! and settles, then acts. The engine's tip is read through the parent check of a probe block,
//! and the store's persisted history is held to a fresh store indexed over reth's final canonical
//! chain once a save has carried the case's blocks to disk.

use super::{
    Backend, BlockSpec, FlushCadence, TxSpec, chain_spec_with_address, create_block_from_spec,
    create_mdbx_proofs_storage, initialize_storage, public_key_to_address,
    simulated_reth::{RethPersistence, SimulatedReth, fixed_signer},
    store_view::{BlockView, ProofTarget, assert_same_view},
};
use alloy_eips::{NumHash, eip1898::BlockWithParent};
use alloy_primitives::{Address, B256, U256};
use parking_lot::Mutex;
use reth_chain_state::ExecutedBlock;
use reth_chainspec::ChainSpec;
use reth_ethereum_primitives::Block;
use reth_optimism_trie::{
    InMemoryProofsStorage, OpProofsProviderRO, OpProofsStorageError, OpProofsStorageResult,
    OpProofsStore, ProofWindowRange,
    engine::{EngineError, EngineHandle, ReorgBlock},
};
use reth_trie_common::{HashedPostStateSorted, updates::TrieUpdatesSorted};
use rstest::rstest;
use secp256k1::Keypair;
use serial_test::serial;
use std::{
    fmt,
    ops::RangeInclusive,
    sync::{
        Arc,
        mpsc::{Receiver, SyncSender, sync_channel},
    },
    thread,
};

/// Blocks every case commits and settles before it acts.
const CHAIN_LENGTH: u64 = 7;

/// Blocks a case may hold at most, which bounds the flushes a settle needs and the recipients.
const MAX_BLOCKS_PER_CASE: u64 = 2 * CHAIN_LENGTH;

/// reth's persistence threshold, above any chain here so that reth keeps every block in memory,
/// where `reorg_with` can fork it anywhere and the reference store still finds genesis state.
const RETH_PERSISTENCE_THRESHOLD: u64 = 2 * MAX_BLOCKS_PER_CASE;

/// Blocks the pruned window keeps, so that pruning moves the earliest block above the anchor
/// under either cadence.
const PRUNED_WINDOW: u64 = 3;

/// A window no chain here fills, so that the earliest block stays the anchor.
const WHOLE_WINDOW: u64 = 1000;

/// A parent hash no block here has.
const UNRELATED_PARENT: B256 = B256::repeat_byte(0xBA);

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
    [sender, Address::ZERO]
        .into_iter()
        .chain(
            (1..=MAX_BLOCKS_PER_CASE)
                .flat_map(|number| [recipient(number), fork_recipient(number)]),
        )
        .map(|address| ProofTarget { address, slots: Vec::new() })
        .collect()
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
}

/// `blocks` the way the ExEx hands a reorg to the engine along `path`.
fn reorg_blocks(blocks: &[ExecutedBlock], path: ReorgPath) -> Vec<ReorgBlock<Block>> {
    blocks
        .iter()
        .map(|block| match path {
            ReorgPath::Execute => ReorgBlock::Execute(Arc::clone(&block.recovered_block)),
            ReorgPath::Index => ReorgBlock::Index {
                block: block.recovered_block().block_with_parent(),
                hashed_state: block.hashed_state(),
                trie_updates: block.trie_updates(),
            },
        })
        .collect()
}

/// The reorg task's input for `block` without trie data, enough to reach the unwind checks.
fn reorg_block_without_data(block: BlockWithParent) -> ReorgBlock<Block> {
    ReorgBlock::Index {
        block,
        hashed_state: Arc::new(HashedPostStateSorted::default()),
        trie_updates: Arc::new(TrieUpdatesSorted::default()),
    }
}

/// Which engine task unwinds.
#[derive(Clone, Copy, Debug)]
enum Task {
    Reorg,
    Unwind,
}

/// Where an unwind target sits relative to the proof window.
#[derive(Clone, Copy, Debug)]
enum Target {
    /// The init anchor, block 0: the earliest block of a whole window, below a pruned one.
    Anchor,
    BelowEarliest,
    Earliest,
}

/// The store's end of a held write.
struct WriteHold {
    entered: SyncSender<()>,
    release: Receiver<()>,
}

/// The case's end of a held write.
struct HeldWrite {
    entered: Receiver<()>,
    release: SyncSender<()>,
}

impl HeldWrite {
    /// Blocks until the engine's persistence thread is inside the held write.
    fn wait_until_entered(&self) {
        self.entered.recv().expect("the store keeps the hold until a write enters it");
    }

    /// Lets the held write go on, or the next write through if none is held yet.
    fn release(self) {
        let _ = self.release.send(());
    }
}

/// A proofs store whose next write transaction, once a hold is armed, blocks until the hold is
/// released, which keeps a save of the engine in flight for as long as a case needs.
#[derive(Clone)]
struct StoreWithWriteGate<S> {
    inner: S,
    next_write: Arc<Mutex<Option<WriteHold>>>,
}

impl<S> StoreWithWriteGate<S> {
    fn new(inner: S) -> Self {
        Self { inner, next_write: Arc::default() }
    }

    fn hold_next_write(&self) -> HeldWrite {
        let (entered_tx, entered_rx) = sync_channel(1);
        let (release_tx, release_rx) = sync_channel(1);
        *self.next_write.lock() = Some(WriteHold { entered: entered_tx, release: release_rx });
        HeldWrite { entered: entered_rx, release: release_tx }
    }
}

impl<S: fmt::Debug> fmt::Debug for StoreWithWriteGate<S> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.debug_struct("StoreWithWriteGate").field("inner", &self.inner).finish_non_exhaustive()
    }
}

impl<S: OpProofsStore> OpProofsStore for StoreWithWriteGate<S> {
    type ProviderRO<'a>
        = S::ProviderRO<'a>
    where
        Self: 'a;

    type ProviderRw<'a>
        = S::ProviderRw<'a>
    where
        Self: 'a;

    type Initializer<'a>
        = S::Initializer<'a>
    where
        Self: 'a;

    fn provider_ro<'a>(&'a self) -> OpProofsStorageResult<Self::ProviderRO<'a>> {
        self.inner.provider_ro()
    }

    fn provider_rw<'a>(&'a self) -> OpProofsStorageResult<Self::ProviderRw<'a>> {
        let hold = self.next_write.lock().take();
        if let Some(hold) = hold {
            let _ = hold.entered.send(());
            let _ = hold.release.recv();
        }
        self.inner.provider_rw()
    }

    fn initialization_provider<'a>(&'a self) -> OpProofsStorageResult<Self::Initializer<'a>> {
        self.inner.initialization_provider()
    }
}

/// What the store serves at one moment: its window and every persisted view in it.
struct DiskSnapshot {
    views: Vec<(u64, BlockView)>,
    window: ProofWindowRange,
}

/// reth holding the chain, and the engine indexing it into the store under test.
struct Scenario<S> {
    cadence: FlushCadence,
    chain_spec: Arc<ChainSpec>,
    engine: EngineHandle<Block>,
    gated: StoreWithWriteGate<S>,
    key_pair: Keypair,
    reth: SimulatedReth,
    store: S,
    targets: Vec<ProofTarget>,
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
        initialize_storage(store.clone(), &reth.factory, reth.head())?;
        let gated = StoreWithWriteGate::new(store.clone());
        let engine = reth.spawn_engine_with_window(
            gated.clone(),
            cadence.persistence,
            cadence.backpressure,
            window,
        );
        let mut scenario = Self {
            cadence,
            chain_spec,
            engine,
            gated,
            key_pair,
            reth,
            store,
            targets: proof_targets(sender),
        };
        for number in 1..=CHAIN_LENGTH {
            scenario.commit(&chain_block(number))?;
        }
        let disk_tip = scenario.settle()?;
        assert_eq!(disk_tip.number, settled_disk_tip(cadence, 0, CHAIN_LENGTH), "settled disk tip");
        Ok(scenario)
    }

    /// Commits `spec` to reth and indexes it.
    fn commit(&mut self, spec: &BlockSpec) -> eyre::Result<()> {
        let block = self.reth.commit(spec)?;
        self.index(&block)?;
        Ok(())
    }

    fn index(&self, block: &ExecutedBlock) -> Result<(), EngineError> {
        self.engine.index_block(
            block.recovered_block().block_with_parent(),
            (*block.trie_updates()).clone(),
            (*block.hashed_state()).clone(),
        )
    }

    /// Commits enough blocks for a save to carry everything buffered to disk, and settles.
    fn commit_through_a_save(&mut self) -> eyre::Result<()> {
        for _ in 0..self.cadence.persistence {
            let number = self.reth.head().number + 1;
            self.commit(&chain_block(number))?;
        }
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
        Ok(self.window()?.latest)
    }

    fn window(&self) -> eyre::Result<ProofWindowRange> {
        Ok(self.store.provider_ro()?.get_proof_window()?)
    }

    /// Arms a hold on the engine's next write, the save the following commits start.
    fn hold_next_save(&self) -> HeldWrite {
        self.gated.hold_next_write()
    }

    /// Commits blocks until the buffer reaches the persistence threshold and the save it starts
    /// is inside `hold`.
    fn commit_until_a_save_is_in_flight(&mut self, hold: &HeldWrite) -> eyre::Result<()> {
        let buffered = self.reth.head().number - self.window()?.latest.number;
        for _ in buffered..self.cadence.persistence {
            let number = self.reth.head().number + 1;
            self.commit(&chain_block(number))?;
        }
        hold.wait_until_entered();
        Ok(())
    }

    /// Replaces the engine with one started over the same store, as a restart does.
    fn restart_engine(&mut self, window: u64) {
        self.engine = self.reth.spawn_engine_with_window(
            self.gated.clone(),
            self.cadence.persistence,
            self.cadence.backpressure,
            window,
        );
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
            ReorgPath::Execute => ReorgBlock::Execute(Arc::new(create_block_from_spec(
                &BlockSpec::empty(),
                number,
                UNRELATED_PARENT,
                &self.chain_spec,
                self.key_pair,
                &mut 0,
            ))),
            ReorgPath::Index => reorg_block_without_data(BlockWithParent::new(
                UNRELATED_PARENT,
                NumHash::new(number, B256::repeat_byte(0xB1)),
            )),
        };
        vec![block]
    }

    fn disk_snapshot(&self) -> eyre::Result<DiskSnapshot> {
        let window = self.window()?;
        let views = (window.earliest.number..=window.latest.number)
            .map(|number| Ok((number, BlockView::read(&self.store, number, &self.targets)?)))
            .collect::<eyre::Result<_>>()?;
        Ok(DiskSnapshot { views, window })
    }

    /// The store's window and every persisted view are as in `before`.
    fn assert_disk_unchanged(&self, before: &DiskSnapshot) -> eyre::Result<()> {
        assert_eq!(self.window()?, before.window, "proof window");
        for (number, expected) in &before.views {
            let actual = BlockView::read(&self.store, *number, &self.targets)?;
            assert_same_view(*number, "unwound", expected, &actual)?;
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
        let window = self.window()?;
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
        let reference = self.reference_store()?;
        for number in window.earliest.number..=window.latest.number {
            let expected = BlockView::read(&reference, number, &self.targets)?;
            let actual = BlockView::read(&self.store, number, &self.targets)?;
            assert_same_view(number, "under test", &expected, &actual)?;
            actual.verify_proofs(number, self.reth.state_root(number))?;
        }
        Ok(())
    }

    /// A fresh store indexed over reth's canonical chain from genesis.
    fn reference_store(&self) -> eyre::Result<InMemoryProofsStorage> {
        eyre::ensure!(
            !self.reth.locate(self.reth.num_hash(1)).persisted,
            "reth's database must still hold genesis state"
        );
        let store = InMemoryProofsStorage::new();
        initialize_storage(store.clone(), &self.reth.factory, self.reth.genesis())?;
        let engine = self.reth.spawn_engine(store.clone(), 1, 2);
        let head = self.reth.head().number;
        for number in 1..=head {
            let block = self.reth.executed_block(number);
            engine.index_block(
                block.recovered_block().block_with_parent(),
                (*block.trie_updates()).clone(),
                (*block.hashed_state()).clone(),
            )?;
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

/// Runs `$run` over a fresh store of `$backend`.
macro_rules! over_backend {
    ($backend:expr, $run:ident($($arg:expr),*)) => {
        match $backend {
            Backend::InMemory => $run(InMemoryProofsStorage::new, $($arg),*),
            Backend::Mdbx => $run(create_mdbx_proofs_storage, $($arg),*),
        }
    };
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

/// A reorg replaces the stored blocks above the fork with the new chain, indexed or executed,
/// whether the fork is buffered (block 6 under the slower cadence) or on disk.
#[rstest]
#[case::indexed_from_block_six(ReorgPath::Index, 6)]
#[case::executed_from_block_three(ReorgPath::Execute, 3)]
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
    let mut scenario = Scenario::new(make_store, FlushCadence::EVERY_BLOCK, WHOLE_WINDOW)?;
    let before = scenario.disk_snapshot()?;
    scenario.restart_engine(WHOLE_WINDOW);

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
    task: Task,
    target: Target,
    window: u64,
) -> eyre::Result<()> {
    let scenario = Scenario::new(make_store, cadence, window)?;
    let before = scenario.disk_snapshot()?;
    let earliest = before.window.earliest.number;
    let number = match target {
        Target::Anchor => 0,
        Target::BelowEarliest => earliest - 1,
        Target::Earliest => earliest,
    };
    let to = scenario.canonical_target(number);

    let result = match task {
        Task::Reorg => scenario.engine.reorg(vec![reorg_block_without_data(to)]),
        Task::Unwind => scenario.engine.unwind(to),
    };

    let err = result.expect_err("the unwind would remove the window's earliest block");
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
#[case::unwind_at_the_earliest_block(Task::Unwind, Target::Earliest, PRUNED_WINDOW)]
#[case::reorg_below_the_earliest_block(Task::Reorg, Target::BelowEarliest, PRUNED_WINDOW)]
#[case::reorg_at_the_anchor_below_the_window(Task::Reorg, Target::Anchor, PRUNED_WINDOW)]
#[case::unwind_at_the_anchor_as_earliest_block(Task::Unwind, Target::Anchor, WHOLE_WINDOW)]
#[serial]
fn unwind_or_reorg_at_or_below_the_earliest_block_fails_and_changes_nothing(
    #[values(Backend::InMemory, Backend::Mdbx)] backend: Backend,
    #[values(FlushCadence::EVERY_BLOCK, FlushCadence::EVERY_FIFTH_BLOCK)] cadence: FlushCadence,
    #[case] task: Task,
    #[case] target: Target,
    #[case] window: u64,
) -> eyre::Result<()> {
    over_backend!(backend, run_below_window_case(cadence, task, target, window))
}

fn run_unwind_to_above_earliest_case<S: OpProofsStore + Clone + 'static>(
    make_store: impl FnOnce() -> S,
    cadence: FlushCadence,
) -> eyre::Result<()> {
    let mut scenario = Scenario::new(make_store, cadence, PRUNED_WINDOW)?;
    let earliest = scenario.window()?.earliest;

    scenario.engine.unwind(scenario.canonical_target(earliest.number + 1))?;

    assert_eq!(scenario.window()?, ProofWindowRange { earliest, latest: earliest });
    scenario.assert_tip(earliest);
    for number in earliest.number + 1..=CHAIN_LENGTH {
        scenario.index(scenario.reth.executed_block(number))?;
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
        scenario.index(block)?;
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
        scenario.index(block)?;
    }
    scenario.commit_through_a_save()?;
    scenario.assert_disk_matches_reth_through(disk_tip.number + 3)
}

/// A revert of every block above the disk tip while their save is in flight waits for the save,
/// then removes them again: the disk ends as it was and the tip is the disk tip.
#[rstest]
#[serial]
fn revert_during_an_in_flight_save_waits_for_the_save(
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
    let fork = scenario.window()?.latest.number - 2;
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

/// A reorg from a fork on disk while a save of the blocks above it is in flight waits for the
/// save, unwinds the disk to the fork and buffers the new chain, so that the disk and the buffer
/// agree with reth.
#[rstest]
#[serial]
fn reorg_from_a_disk_fork_during_an_in_flight_save_waits_for_the_save(
    #[values(Backend::InMemory, Backend::Mdbx)] backend: Backend,
    #[values(FlushCadence::EVERY_BLOCK, FlushCadence::EVERY_FIFTH_BLOCK)] cadence: FlushCadence,
    #[values(ReorgPath::Index, ReorgPath::Execute)] path: ReorgPath,
) -> eyre::Result<()> {
    over_backend!(backend, run_reorg_during_save_case(cadence, path))
}

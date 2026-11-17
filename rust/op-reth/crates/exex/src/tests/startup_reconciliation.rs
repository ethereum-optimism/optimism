//! Startup of the proofs ExEx over every relation between the proofs storage's window and reth's
//! canonical chain, and the pruner cap that keeps the window reconcilable after a crash.
//!
//! reth's chain is split between its database and its canonical in-memory state, as it is while
//! reth re-imports the blocks it dropped at an unclean shutdown. The canonical blocks are empty
//! blocks whose only state change is the block reward, so every block's header carries a distinct
//! state root that executing the block reproduces: the stored blocks between the window bounds
//! are matched by state root, and the engine can execute the blocks the ExEx syncs.

use super::*;
use alloy_primitives::{Address, U256, keccak256};
use reth_chain_state::{ExecutedBlock, NewCanonicalChain};
use reth_ethereum_primitives::EthPrimitives;
use reth_evm_ethereum::EthEvmConfig;
use reth_exex::{ExExNotifications, Wal};
use reth_exex_test_utils::{Adapter, TestNode, TmpDB};
use reth_node_api::FullNodeTypes;
use reth_node_builder::{NodeAdapter, RethFullAdapter, components::Components};
use reth_optimism_trie::OpProofsStorageError;
use reth_provider::StageCheckpointWriter;
use reth_stages_types::{StageCheckpoint, StageId};
use reth_trie::{TrieAccount, root::state_root_unsorted};
use rstest::rstest;
use std::time::Duration;
use tempfile::TempDir;
use tokio::sync::mpsc::Sender;

/// reth's best block in every scenario.
const RETH_TIP: u64 = 60;

/// Highest block of the canonical chain; the proofs storage can run ahead of reth up to here.
const CANONICAL_CHAIN_TIP: u64 = 75;

/// Number of reth's blocks held in its canonical in-memory state when its persisted head is
/// block 53: a scenario's fork point at or below 53 is in reth's database, one above it in memory.
const FEW_BLOCKS_IN_MEMORY: u64 = 7;

/// Number of reth's blocks held in its canonical in-memory state when its persisted head is
/// block 10, below every scenario's fork point.
const MOST_BLOCKS_IN_MEMORY: u64 = 50;

/// Beneficiary of every canonical block.
const COINBASE: Address = Address::ZERO;

/// Reward of a Frontier block on reth's mainnet test chain, the only state change executing an
/// empty canonical block makes.
const BLOCK_REWARD_WEI: u128 = 5_000_000_000_000_000_000;

/// Hashed address of the account fork blocks change and canonical blocks never touch.
const FORK_ACCOUNT: B256 = B256::new([0xff; 32]);

/// Pruner window that prunes nothing.
const NO_PRUNING: u64 = u64::MAX;

/// Longest wait for the engine to persist the blocks it indexes or syncs.
const PERSISTENCE_DEADLINE: Duration = Duration::from_secs(30);

/// Persistence and backpressure thresholds of the proofs engine.
#[derive(Clone, Copy, Debug)]
struct FlushCadence {
    backpressure_threshold: u64,
    persistence_threshold: u64,
}

const FLUSH_EVERY_BLOCK: FlushCadence =
    FlushCadence { backpressure_threshold: 2, persistence_threshold: 1 };

const FLUSH_EVERY_THREE_BLOCKS: FlushCadence =
    FlushCadence { backpressure_threshold: 6, persistence_threshold: 3 };

/// Proofs storage backend a scenario runs against.
#[derive(Clone, Copy, Debug)]
enum Backend {
    InMemory,
    Mdbx,
}

/// Runs `$body` with `$store` bound to a fresh storage of the given backend.
macro_rules! with_store {
    ($backend:expr, | $store:ident | $body:expr) => {
        match $backend {
            Backend::InMemory => {
                let $store = InMemoryProofsStorage::new();
                $body
            }
            Backend::Mdbx => {
                let dir = tempfile::tempdir().expect("tempdir");
                let $store = Arc::new(MdbxProofsStorage::new(dir.path()).expect("env"));
                $body
            }
        }
    };
}

/// Shape of the proofs storage relative to the canonical chain: blocks `earliest..=tip` are
/// stored, and those above `canonical_through` are on a fork; a storage with no canonical block
/// is on a fork throughout.
#[derive(Clone, Copy, Debug)]
struct StoreShape {
    canonical_through: Option<u64>,
    earliest: u64,
    tip: u64,
}

/// A storage holding canonical blocks `earliest..=tip`.
const fn canonical(earliest: u64, tip: u64) -> StoreShape {
    StoreShape { canonical_through: Some(tip), earliest, tip }
}

/// A storage holding blocks `earliest..=tip` that follow a fork above `fork_point`.
const fn forked(earliest: u64, fork_point: u64, tip: u64) -> StoreShape {
    StoreShape { canonical_through: Some(fork_point), earliest, tip }
}

/// A storage holding fork blocks `earliest..=tip` and no canonical block.
const fn foreign(earliest: u64, tip: u64) -> StoreShape {
    StoreShape { canonical_through: None, earliest, tip }
}

impl StoreShape {
    fn is_canonical_at(&self, number: u64) -> bool {
        self.canonical_through.is_some_and(|through| number <= through)
    }

    /// The block the storage holds at `number`: the canonical chain's, or the fork's.
    fn block_at(&self, chain: &[RecoveredBlock<Block>], number: u64) -> NumHash {
        if self.is_canonical_at(number) {
            chain[number as usize].num_hash()
        } else {
            NumHash::new(number, fork_hash(number))
        }
    }

    /// The account the block the storage holds at `number` changes, by hashed address, and its
    /// state after that block.
    fn account_at(&self, number: u64) -> (B256, Account) {
        if self.is_canonical_at(number) {
            (keccak256(COINBASE), canonical_account(number))
        } else {
            (FORK_ACCOUNT, Account { nonce: number, ..Default::default() })
        }
    }

    /// The state diff of the block the storage holds at `number`.
    fn diff_at(&self, number: u64) -> BlockStateDiff {
        let (hashed_address, account) = self.account_at(number);
        account_diff(hashed_address, account)
    }
}

/// Expected result of startup, or of running the ExEx, and its effect on the proofs storage.
#[derive(Clone, Copy, Debug)]
enum Outcome {
    /// Startup fails with an error naming this text and the init command; the storage is left as
    /// it was.
    Fails(&'static str),
    /// `run` unwinds the storage onto reth's chain and its engine syncs it to reth's tip.
    SyncedToRethTip,
    /// The storage window is left as it was.
    Untouched,
    /// The storage is unwound to this canonical block.
    UnwoundTo(u64),
}

/// The coinbase after canonical block `number`: its balance holds every block reward so far.
fn canonical_account(number: u64) -> Account {
    let balance = U256::from(BLOCK_REWARD_WEI * u128::from(number));
    Account { balance, ..Default::default() }
}

/// State diff of a block that leaves the account at `hashed_address` as `account`.
fn account_diff(hashed_address: B256, account: Account) -> BlockStateDiff {
    let mut post_state = HashedPostState::default();
    post_state.accounts.insert(hashed_address, Some(account));
    BlockStateDiff { sorted_post_state: post_state.into_sorted(), ..Default::default() }
}

/// State diff of canonical block `number`.
fn canonical_diff(number: u64) -> BlockStateDiff {
    account_diff(keccak256(COINBASE), canonical_account(number))
}

/// State root of canonical block `number`.
fn canonical_state_root(number: u64) -> B256 {
    let account = TrieAccount { balance: canonical_account(number).balance, ..Default::default() };
    state_root_unsorted([(keccak256(COINBASE), account)])
}

/// State root the proofs storage serves at block `number`.
fn state_root_at<S: OpProofsStore>(store: &S, number: u64) -> B256 {
    let provider_ro = store.provider_ro().expect("provider ro");
    StateRoot::overlay_root(&provider_ro, number, HashedPostState::default()).expect("state root")
}

fn fork_hash(number: u64) -> B256 {
    let mut hash = [0xf0; 32];
    hash[24..].copy_from_slice(&number.to_be_bytes());
    B256::new(hash)
}

/// Builds the canonical chain `1..=CANONICAL_CHAIN_TIP` on reth's genesis: empty blocks whose
/// headers carry [`canonical_state_root`]. Returns the blocks indexed by number, genesis included.
fn canonical_chain(handle: &TestExExHandle) -> Vec<RecoveredBlock<Block>> {
    let mut chain = vec![handle.genesis.clone()];
    for number in 1..=CANONICAL_CHAIN_TIP {
        let header = Header {
            beneficiary: COINBASE,
            number,
            parent_hash: chain[number as usize - 1].hash(),
            state_root: canonical_state_root(number),
            ..Default::default()
        };
        let block = SealedBlock::seal_slow(Block { header, body: Default::default() })
            .try_recover()
            .expect("empty block");
        chain.push(block);
    }
    chain
}

fn executed_blocks(blocks: &[RecoveredBlock<Block>]) -> Vec<ExecutedBlock> {
    blocks
        .iter()
        .map(|block| ExecutedBlock {
            recovered_block: Arc::new(block.clone()),
            ..Default::default()
        })
        .collect()
}

/// Makes blocks `1..=RETH_TIP` of `chain` reth's canonical chain: the last `in_memory_blocks` of
/// them through reth's canonical in-memory state, the rest through its database with the `Finish`
/// stage checkpoint, reth's persisted head, at the highest of them.
fn seed_reth_chain<P>(
    handle: &TestExExHandle,
    provider: &P,
    chain: &[RecoveredBlock<Block>],
    in_memory_blocks: u64,
) where
    P: CanonChainTracker<Header = Header>
        + CanonicalInMemoryStateProvider<Primitives = EthPrimitives>,
{
    let persisted = RETH_TIP - in_memory_blocks;
    let rw = handle.provider_factory.provider_rw().expect("provider rw");
    for block in &chain[1..=persisted as usize] {
        rw.insert_block(block).expect("insert block");
    }
    rw.save_stage_checkpoint(StageId::Finish, StageCheckpoint::new(persisted))
        .expect("finish checkpoint");
    rw.commit().expect("commit");

    extend_reth_in_memory(provider, &chain[persisted as usize + 1..=RETH_TIP as usize]);
}

/// Appends `blocks` to reth's canonical in-memory state and moves its head to the last of them.
fn extend_reth_in_memory<P>(provider: &P, blocks: &[RecoveredBlock<Block>])
where
    P: CanonChainTracker<Header = Header>
        + CanonicalInMemoryStateProvider<Primitives = EthPrimitives>,
{
    let head = blocks.last().expect("at least one block").clone_sealed_header();
    provider
        .canonical_in_memory_state()
        .update_chain(NewCanonicalChain::Commit { new: executed_blocks(blocks) });
    provider.set_canonical_head(head);
}

/// Node components of `test_node` with an EVM that executes blocks; the test context's own EVM
/// panics on any call.
fn executing_node(
    test_node: Adapter,
) -> impl FullNodeComponents<
    Types = TestNode,
    DB = TmpDB,
    Provider = <Adapter as FullNodeTypes>::Provider,
    Evm = EthEvmConfig,
> {
    let NodeAdapter { components, provider, task_executor } = test_node;
    let Components { consensus, evm_config: _, network, payload_builder_handle, transaction_pool } =
        components;
    NodeAdapter::<RethFullAdapter<TmpDB, TestNode>, _> {
        components: Components {
            consensus,
            evm_config: EthEvmConfig::mainnet(),
            network,
            payload_builder_handle,
            transaction_pool,
        },
        provider,
        task_executor,
    }
}

/// An ExEx test context whose reth holds the canonical chain up to [`RETH_TIP`] and whose EVM
/// executes blocks.
struct RethFixture<Node: FullNodeComponents> {
    chain: Vec<RecoveredBlock<Block>>,
    ctx: ExExContext<Node>,
    /// Owns the ExEx event receiver; dropping it would fail the ExEx's `FinishedHeight` sends.
    _handle: TestExExHandle,
    /// Keeps the ExEx's notification stream open, so `run` waits for notifications instead of
    /// returning.
    _notifications: Sender<ExExNotification>,
    _wal_directory: TempDir,
}

async fn reth_at_tip(
    in_memory_blocks: u64,
) -> RethFixture<
    impl FullNodeComponents<
        Types = TestNode,
        DB = TmpDB,
        Provider = <Adapter as FullNodeTypes>::Provider,
        Evm = EthEvmConfig,
    >,
> {
    let (ctx, handle) = reth_exex_test_utils::test_exex_context().await.expect("exex test context");
    let chain = canonical_chain(&handle);
    seed_reth_chain(&handle, &ctx.components.provider, &chain, in_memory_blocks);

    let ExExContext { components, config, events, head, notifications: _, reth_config } = ctx;
    let components = executing_node(components);
    let (notifications_tx, notifications_rx) = tokio::sync::mpsc::channel(1);
    let wal_directory = tempfile::tempdir().expect("tempdir");
    let wal = Wal::new(wal_directory.path()).expect("wal");
    let notifications = ExExNotifications::new(
        head,
        components.provider().clone(),
        components.evm_config().clone(),
        notifications_rx,
        wal.handle(),
    );
    let ctx = ExExContext { components, config, events, head, notifications, reth_config };
    RethFixture {
        chain,
        ctx,
        _handle: handle,
        _notifications: notifications_tx,
        _wal_directory: wal_directory,
    }
}

/// Seeds `store` with `shape` and returns its proof window.
fn seed_store<S: OpProofsStore + Clone>(
    store: &S,
    chain: &[RecoveredBlock<Block>],
    shape: StoreShape,
) -> ProofWindowRange {
    let (hashed_address, account) = shape.account_at(shape.earliest);
    let anchor = shape.block_at(chain, shape.earliest);
    init_storage_at(store.clone(), anchor, vec![(hashed_address, Some(account))]);
    let rw = store.provider_rw().expect("provider rw");
    for number in shape.earliest + 1..=shape.tip {
        let parent = shape.block_at(chain, number - 1).hash;
        let block = BlockWithParent::new(parent, shape.block_at(chain, number));
        rw.store_trie_updates(block, shape.diff_at(number)).expect("store block");
    }
    OpProofsProviderRw::commit(rw).expect("commit");
    proof_window(store).expect("proof window")
}

fn proof_window<S: OpProofsStore>(store: &S) -> Result<ProofWindowRange, OpProofsStorageError> {
    store.provider_ro().expect("provider ro").get_proof_window()
}

fn latest_block<S: OpProofsStore>(store: &S) -> NumHash {
    store.provider_ro().expect("provider ro").get_latest_block().expect("latest block")
}

/// Waits until the storage's latest block is `expected`.
async fn wait_for_latest_block<S: OpProofsStore>(store: &S, expected: NumHash) {
    let reached = async {
        loop {
            if latest_block(store) == expected {
                break;
            }
            tokio::time::sleep(Duration::from_millis(10)).await;
        }
    };
    let reached_in_time = tokio::time::timeout(PERSISTENCE_DEADLINE, reached).await.is_ok();
    assert!(reached_in_time, "storage is at {:?}, expected {expected:?}", latest_block(store));
}

/// Waits for the engine's in-flight persistence, off the runtime thread so the deadline holds
/// even if the engine never replies.
async fn flush_engine(engine_handle: &EngineHandle<Block>) {
    let engine_handle = engine_handle.clone();
    let flushed = tokio::task::spawn_blocking(move || engine_handle.flush());
    tokio::time::timeout(PERSISTENCE_DEADLINE, flushed)
        .await
        .expect("engine flush should complete within the deadline")
        .expect("engine flush should not panic");
}

/// A startup error names `message` and the init command, and leaves the storage's window as
/// seeded.
fn assert_startup_failure<S: OpProofsStore>(
    err: &eyre::Report,
    message: &str,
    store: &S,
    seeded_window: ProofWindowRange,
) {
    assert!(err.to_string().contains(message), "{err:#}");
    assert!(err.to_string().contains(PROOFS_INIT_COMMAND), "{err:#}");
    assert_eq!(proof_window(store).expect("proof window"), seeded_window);
}

/// Seeds reth and the proofs storage, runs startup, and checks the outcome.
async fn startup_case<S: OpProofsStore + Clone + 'static>(
    store: S,
    shape: StoreShape,
    expected: Outcome,
    reth_in_memory_blocks: u64,
) {
    let reth = reth_at_tip(reth_in_memory_blocks).await;
    let seeded_window = seed_store(&store, &reth.chain, shape);

    let exex = build_test_exex(reth.ctx, store.clone());
    let result = exex.startup();

    match expected {
        Outcome::Fails(message) => {
            let err = result.expect_err("startup should fail");
            assert_startup_failure(&err, message, &store, seeded_window);
        }
        Outcome::SyncedToRethTip => panic!("{expected:?} is an outcome of running the ExEx"),
        Outcome::Untouched => {
            result.expect("startup should succeed");
            assert_eq!(proof_window(&store).expect("proof window"), seeded_window);
        }
        Outcome::UnwoundTo(number) => {
            result.expect("startup should succeed");
            let window = proof_window(&store).expect("proof window");
            assert_eq!(window.earliest, seeded_window.earliest);
            assert_eq!(window.latest, reth.chain[number as usize].num_hash());
            assert_eq!(state_root_at(&store, number), canonical_state_root(number));
        }
    }
}

/// Startup over the relations between the proofs storage's window and reth's chain, whose best
/// block is [`RETH_TIP`], that are settled by the window bounds alone and probe a block below
/// reth's persisted head, in reth's database or in its in-memory state.
#[rstest]
#[case::behind_reth(canonical(0, 50), Outcome::Untouched)]
#[case::fork_below_window(foreign(30, 58), Outcome::Fails("shares no block"))]
#[tokio::test]
async fn startup_reconciles_storage_with_reth_chain(
    #[case] shape: StoreShape,
    #[case] expected: Outcome,
    #[values(FEW_BLOCKS_IN_MEMORY, MOST_BLOCKS_IN_MEMORY)] reth_in_memory_blocks: u64,
    #[values(Backend::Mdbx, Backend::InMemory)] backend: Backend,
) {
    with_store!(backend, |store| startup_case(store, shape, expected, reth_in_memory_blocks).await);
}

/// Startup over the relations between the proofs storage's window and reth's chain that probe
/// no block below reth's persisted head: genesis, reth's tip, or nothing but the window bounds.
/// The single foreign block at height 0 leaves no block below it to probe.
#[rstest]
#[case::at_reth_tip(canonical(0, 60), Outcome::Untouched)]
#[case::initialized_at_reth_tip(canonical(60, 60), Outcome::Untouched)]
#[case::ahead_on_reth_chain(canonical(0, 65), Outcome::UnwoundTo(60))]
#[case::ahead_from_reth_tip(canonical(60, 65), Outcome::UnwoundTo(60))]
#[case::ahead_on_fork_above_reth_tip(forked(0, 60, 65), Outcome::UnwoundTo(60))]
#[case::single_foreign_block(foreign(0, 0), Outcome::Fails("shares no block"))]
#[case::window_above_reth_tip(canonical(70, 75), Outcome::Fails("starts at block 70"))]
#[tokio::test]
async fn startup_reconciles_storage_with_reth_tip(
    #[case] shape: StoreShape,
    #[case] expected: Outcome,
    #[values(Backend::Mdbx, Backend::InMemory)] backend: Backend,
) {
    with_store!(backend, |store| startup_case(store, shape, expected, FEW_BLOCKS_IN_MEMORY).await);
}

/// Startup unwinds a storage on a fork below reth's tip to the fork point, which the state-root
/// probes find whether the probed blocks are in reth's database or in its in-memory state.
#[rstest]
#[case::ahead_on_fork_just_below_reth_tip(forked(0, 59, 65))]
#[case::ahead_on_fork_below_reth_tip(forked(0, 40, 65))]
#[case::at_reth_tip_on_fork(forked(0, 40, 60))]
#[case::below_reth_tip_on_fork(forked(0, 40, 58))]
#[case::tip_replaced_below_reth_tip(forked(0, 57, 58))]
#[case::one_block_above_window_start_on_fork(forked(40, 40, 41))]
#[case::fork_from_window_start(forked(30, 30, 58))]
#[tokio::test]
async fn startup_unwinds_storage_to_fork_point(
    #[case] shape: StoreShape,
    #[values(FEW_BLOCKS_IN_MEMORY, MOST_BLOCKS_IN_MEMORY)] reth_in_memory_blocks: u64,
    #[values(Backend::Mdbx, Backend::InMemory)] backend: Backend,
) {
    let expected = Outcome::UnwoundTo(shape.canonical_through.expect("fork point"));
    with_store!(backend, |store| startup_case(store, shape, expected, reth_in_memory_blocks).await);
}

async fn uninitialized_case<S: OpProofsStore + Clone + 'static>(store: S) {
    let (ctx, _handle) =
        reth_exex_test_utils::test_exex_context().await.expect("exex test context");

    let exex = build_test_exex(ctx, store.clone());
    let err = exex.startup().expect_err("startup should fail");

    assert!(err.to_string().contains("not initialized"), "{err:#}");
    assert!(err.to_string().contains(PROOFS_INIT_COMMAND), "{err:#}");
    assert!(matches!(proof_window(&store), Err(OpProofsStorageError::NoBlocksFound)));
}

/// An uninitialized proofs storage fails startup before any reconciliation, naming the init
/// command.
#[rstest]
#[tokio::test]
async fn startup_fails_on_uninitialized_storage(
    #[values(Backend::Mdbx, Backend::InMemory)] backend: Backend,
) {
    with_store!(backend, |store| uninitialized_case(store).await);
}

/// Seeds reth and the proofs storage, runs the ExEx, and checks the outcome.
async fn run_case<S: OpProofsStore + Clone + 'static>(
    store: S,
    shape: StoreShape,
    expected: Outcome,
) {
    let reth = reth_at_tip(FEW_BLOCKS_IN_MEMORY).await;
    let seeded_window = seed_store(&store, &reth.chain, shape);

    let run = tokio::spawn(build_test_exex(reth.ctx, store.clone()).run());
    match expected {
        Outcome::Fails(message) => {
            let err = tokio::time::timeout(PERSISTENCE_DEADLINE, run)
                .await
                .expect("run should return")
                .expect("run should not panic")
                .expect_err("run should fail");
            assert_startup_failure(&err, message, &store, seeded_window);
        }
        Outcome::SyncedToRethTip => {
            wait_for_latest_block(&store, reth.chain[RETH_TIP as usize].num_hash()).await;
            run.abort();
            let _ = run.await;
            assert_eq!(state_root_at(&store, RETH_TIP), canonical_state_root(RETH_TIP));
        }
        Outcome::Untouched | Outcome::UnwoundTo(_) => {
            panic!("{expected:?} is an outcome of startup alone")
        }
    }
}

/// Running the ExEx: a startup error returns from `run`, and a startup unwind is followed by the
/// engine executing the canonical blocks above the unwound block up to reth's tip.
#[rstest]
#[case::startup_error(canonical(70, 75), Outcome::Fails("starts at block 70"))]
#[case::unwind_then_sync(forked(0, 50, 65), Outcome::SyncedToRethTip)]
#[tokio::test]
async fn run_reconciles_storage_then_syncs(
    #[case] shape: StoreShape,
    #[case] expected: Outcome,
    #[values(Backend::Mdbx, Backend::InMemory)] backend: Backend,
) {
    with_store!(backend, |store| run_case(store, shape, expected).await);
}

/// Seeds the proofs storage ahead of reth, runs startup, then has reth re-import the blocks above
/// its best block and checks that the ExEx indexes them on top of the unwound storage.
async fn reimport_case<S: OpProofsStore + Clone + 'static>(
    store: S,
    shape: StoreShape,
    cadence: FlushCadence,
) {
    let reth = reth_at_tip(FEW_BLOCKS_IN_MEMORY).await;
    let provider = reth.ctx.provider().clone();
    let evm_config = reth.ctx.evm_config().clone();
    seed_store(&store, &reth.chain, shape);

    let exex = build_test_exex(reth.ctx, store.clone());
    exex.startup().expect("startup should succeed");
    assert_eq!(latest_block(&store), reth.chain[RETH_TIP as usize].num_hash());

    let reimported = &reth.chain[RETH_TIP as usize + 1..=shape.tip as usize];
    extend_reth_in_memory(&provider, reimported);

    let trie_data = reimported
        .iter()
        .map(|block| {
            let hashed_state = Arc::new(canonical_diff(block.number()).sorted_post_state);
            let computed =
                ComputedTrieData::new(hashed_state, Arc::new(TrieUpdatesSorted::default()));
            (block.number(), LazyTrieData::ready(computed))
        })
        .collect();
    let notification = ExExNotification::ChainCommitted {
        new: Arc::new(mk_chain(reimported.to_vec(), RETH_TIP + 1, trie_data)),
    };
    let pruner = OpProofStoragePruner::new(store.clone(), provider.clone(), NO_PRUNING);
    let engine_handle = EngineHandle::spawn_with_thresholds(
        evm_config,
        provider,
        store.clone(),
        pruner,
        cadence.persistence_threshold,
        cadence.backpressure_threshold,
    );
    exex.handle_notification(notification, &engine_handle).expect("handle notification");
    flush_engine(&engine_handle).await;
    wait_for_latest_block(&store, reth.chain[shape.tip as usize].num_hash()).await;

    assert_eq!(state_root_at(&store, shape.tip), canonical_state_root(shape.tip));
}

/// After the startup unwind, reth's re-import of the blocks above its best block is indexed on
/// top of the storage: the same blocks when reth only dropped them, replacements when the chain
/// moved on while the node was down. The fork's state must not survive in the storage.
#[rstest]
#[case::same_blocks(canonical(0, 66))]
#[case::replaced_blocks(forked(0, 60, 66))]
#[tokio::test]
async fn startup_unwind_lets_reth_reimport_be_indexed(
    #[case] shape: StoreShape,
    #[values(FLUSH_EVERY_BLOCK, FLUSH_EVERY_THREE_BLOCKS)] cadence: FlushCadence,
    #[values(Backend::Mdbx, Backend::InMemory)] backend: Backend,
) {
    with_store!(backend, |store| reimport_case(store, shape, cadence).await);
}

/// Seeds the proofs storage with canonical blocks `0..=RETH_TIP` and prunes it with an interval
/// shorter than reth's in-memory chain, so the cap at reth's persisted head binds.
async fn prune_case<S: OpProofsStore + Clone + 'static>(store: S) {
    let reth = reth_at_tip(FEW_BLOCKS_IN_MEMORY).await;
    seed_store(&store, &reth.chain, canonical(0, RETH_TIP));

    let provider = reth.ctx.provider().clone();
    let pruner = OpProofStoragePruner::new(store.clone(), provider, FEW_BLOCKS_IN_MEMORY - 1);
    let provider_rw = store.provider_rw().expect("provider rw");
    pruner.prune_with_provider(&provider_rw).expect("prune");
    OpProofsProviderRw::commit(provider_rw).expect("commit");

    let persisted_head = RETH_TIP - FEW_BLOCKS_IN_MEMORY;
    let window = proof_window(&store).expect("proof window");
    assert_eq!(window.earliest, reth.chain[persisted_head as usize].num_hash());
    assert_eq!(window.latest, reth.chain[RETH_TIP as usize].num_hash());
}

/// The pruner never moves the window's earliest block above reth's `Finish` stage checkpoint,
/// its persisted head, which it reads from the real provider: reth may drop its in-memory blocks
/// at a crash, and the storage can only be unwound down to its earliest block.
#[rstest]
#[tokio::test]
async fn pruner_caps_earliest_at_reth_persisted_head(
    #[values(Backend::Mdbx, Backend::InMemory)] backend: Backend,
) {
    with_store!(backend, |store| prune_case(store).await);
}

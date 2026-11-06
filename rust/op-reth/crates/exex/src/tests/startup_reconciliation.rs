//! Startup of the proofs ExEx over every relation between the proofs storage's window and reth's
//! canonical chain, and the pruner cap that keeps the window reconcilable after a crash.
//!
//! reth's chain is split between its database and its canonical in-memory state, as it is while
//! reth re-imports the blocks it dropped at an unclean shutdown. The stored blocks between the
//! window bounds are matched by state root, so every canonical block carries a distinct one.

use super::*;
use reth_chain_state::{ExecutedBlock, NewCanonicalChain};
use reth_ethereum_primitives::EthPrimitives;
use reth_exex_test_utils::Adapter;
use reth_optimism_trie::OpProofsStorageError;
use reth_trie::{TrieAccount, root::state_root_unsorted};
use rstest::rstest;

/// reth's best block in every scenario.
const RETH_TIP: u64 = 60;

/// Highest block of the canonical chain; the proofs storage can run ahead of reth up to here.
const CANONICAL_CHAIN_TIP: u64 = 75;

/// reth's persistence threshold on develop, used as the number of reth's blocks held in memory:
/// the scenarios' fork points stay in reth's database.
const PERSISTENCE_THRESHOLD_DEVELOP: u64 = 7;

/// reth's persistence threshold since reth 2.6, used as the number of reth's blocks held in
/// memory: the scenarios' fork points are in memory.
const PERSISTENCE_THRESHOLD_RETH_2_6: u64 = 50;

/// Hashed address of the account every canonical block changes.
const ACCOUNT: B256 = B256::new([0xaa; 32]);

/// Hashed address of the account fork blocks change and canonical blocks never touch.
const FORK_ACCOUNT: B256 = B256::new([0xff; 32]);

/// Pruner window that prunes nothing.
const NO_PRUNING: u64 = u64::MAX;

/// Persistence and backpressure thresholds of the proofs engine.
#[derive(Clone, Copy, Debug)]
struct FlushCadence {
    backpressure_threshold: u64,
    persistence_threshold: u64,
}

const FLUSH_EVERY_BLOCK: FlushCadence =
    FlushCadence { backpressure_threshold: 2, persistence_threshold: 1 };

const FLUSH_EVERY_FIVE_BLOCKS: FlushCadence =
    FlushCadence { backpressure_threshold: 10, persistence_threshold: 5 };

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
                let dir = tempdir_path();
                let $store = Arc::new(MdbxProofsStorage::new(dir.as_path()).expect("env"));
                $body
            }
        }
    };
}

/// Shape of the proofs storage relative to the canonical chain: blocks `earliest..=tip` are
/// stored, and those above `canonical_through` are on a fork.
#[derive(Clone, Copy, Debug)]
struct StoreShape {
    canonical_through: u64,
    earliest: u64,
    tip: u64,
}

/// A storage holding canonical blocks `earliest..=tip`.
const fn canonical(earliest: u64, tip: u64) -> StoreShape {
    StoreShape { canonical_through: tip, earliest, tip }
}

/// A storage holding blocks `earliest..=tip` that follow a fork above `fork_point`.
const fn forked(earliest: u64, fork_point: u64, tip: u64) -> StoreShape {
    StoreShape { canonical_through: fork_point, earliest, tip }
}

impl StoreShape {
    /// The block the storage holds at `number`: the canonical chain's, or the fork's.
    fn block_at(&self, chain: &[RecoveredBlock<Block>], number: u64) -> NumHash {
        if number <= self.canonical_through {
            chain[number as usize].num_hash()
        } else {
            NumHash::new(number, fork_hash(number))
        }
    }

    /// The state diff of the block the storage holds at `number`.
    fn diff_at(&self, number: u64) -> BlockStateDiff {
        if number <= self.canonical_through {
            account_diff(ACCOUNT, number)
        } else {
            account_diff(FORK_ACCOUNT, number)
        }
    }
}

/// Expected result of the startup steps and their effect on the proofs storage.
#[derive(Clone, Copy, Debug)]
enum Outcome {
    /// Startup fails with an error naming this text and the init command; the storage is left as
    /// it was.
    Fails(&'static str),
    /// The storage window is left as it was.
    Untouched,
    /// The storage is unwound to this canonical block.
    UnwoundTo(u64),
}

/// State diff of a block that sets the nonce of the account at `hashed_address` to `nonce`.
fn account_diff(hashed_address: B256, nonce: u64) -> BlockStateDiff {
    let mut post_state = HashedPostState::default();
    post_state.accounts.insert(hashed_address, Some(Account { nonce, ..Default::default() }));
    BlockStateDiff { sorted_post_state: post_state.into_sorted(), ..Default::default() }
}

/// State root of canonical block `number`, whose state is [`ACCOUNT`] with nonce `number`.
fn canonical_state_root(number: u64) -> B256 {
    state_root_unsorted([(ACCOUNT, TrieAccount { nonce: number, ..Default::default() })])
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

/// Makes blocks `1..=RETH_TIP` of `chain` reth's canonical chain, the last `in_memory_blocks` of
/// them through reth's canonical in-memory state and the rest through its database.
fn seed_reth_chain<P>(
    handle: &TestExExHandle,
    provider: &P,
    chain: &[RecoveredBlock<Block>],
    in_memory_blocks: u64,
) where
    P: CanonChainTracker<Header = Header>
        + CanonicalInMemoryStateProvider<Primitives = EthPrimitives>,
{
    let persisted = (RETH_TIP - in_memory_blocks) as usize;
    let rw = handle.provider_factory.provider_rw().expect("provider rw");
    for block in &chain[1..=persisted] {
        rw.insert_block(block).expect("insert block");
    }
    rw.commit().expect("commit");

    let in_memory = executed_blocks(&chain[persisted + 1..=RETH_TIP as usize]);
    provider.canonical_in_memory_state().update_chain(NewCanonicalChain::Commit { new: in_memory });
    provider.set_canonical_head(chain[RETH_TIP as usize].clone_sealed_header());
}

/// An ExEx test context whose reth holds the canonical chain up to [`RETH_TIP`].
struct RethFixture {
    chain: Vec<RecoveredBlock<Block>>,
    ctx: TestExExContext,
    /// Owns the ExEx event receiver; dropping it would fail the ExEx's `FinishedHeight` sends.
    _handle: TestExExHandle,
}

async fn reth_at_tip(in_memory_blocks: u64) -> RethFixture {
    let (ctx, handle) = reth_exex_test_utils::test_exex_context().await.expect("exex test context");
    let chain = canonical_chain(&handle);
    seed_reth_chain(&handle, &ctx.components.provider, &chain, in_memory_blocks);
    RethFixture { chain, ctx, _handle: handle }
}

/// Seeds `store` with `shape` and returns its proof window.
fn seed_store<S: OpProofsStore + Clone>(
    store: &S,
    chain: &[RecoveredBlock<Block>],
    shape: StoreShape,
) -> ProofWindowRange {
    init_storage_at(store.clone(), shape.block_at(chain, shape.earliest));
    for number in shape.earliest + 1..=shape.tip {
        let parent = shape.block_at(chain, number - 1).hash;
        store_block(store, parent, shape.block_at(chain, number), shape.diff_at(number));
    }
    proof_window(store).expect("proof window")
}

fn proof_window<S: OpProofsStore>(store: &S) -> Result<ProofWindowRange, OpProofsStorageError> {
    store.provider_ro().expect("provider ro").get_proof_window()
}

/// Runs the startup steps of [`OpProofsExEx::run`] that precede the engine.
fn startup<S: OpProofsStore + Clone + 'static>(
    exex: &OpProofsExEx<Adapter, S>,
) -> eyre::Result<()> {
    exex.ensure_initialized().and_then(|window| exex.reconcile_with_canonical_chain(window))
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
    let result = startup(&exex);

    let window = proof_window(&store).expect("proof window");
    match expected {
        Outcome::Fails(message) => {
            let err = result.expect_err("startup should fail");
            assert!(err.to_string().contains(message), "{err:#}");
            assert!(err.to_string().contains(PROOFS_INIT_COMMAND), "{err:#}");
            assert_eq!(window, seeded_window);
        }
        Outcome::Untouched => {
            result.expect("startup should succeed");
            assert_eq!(window, seeded_window);
        }
        Outcome::UnwoundTo(number) => {
            result.expect("startup should succeed");
            assert_eq!(window.earliest, seeded_window.earliest);
            assert_eq!(window.latest, reth.chain[number as usize].num_hash());
        }
    }
}

/// Startup over every relation between the proofs storage's window and reth's chain, whose best
/// block is [`RETH_TIP`].
#[rstest]
#[case::behind_reth(canonical(0, 50), Outcome::Untouched)]
#[case::at_reth_tip(canonical(0, 60), Outcome::Untouched)]
#[case::initialized_at_reth_tip(canonical(60, 60), Outcome::Untouched)]
#[case::ahead_on_reth_chain(canonical(0, 65), Outcome::UnwoundTo(60))]
#[case::ahead_from_reth_tip(canonical(60, 65), Outcome::UnwoundTo(60))]
#[case::ahead_on_fork_above_reth_tip(forked(0, 60, 65), Outcome::UnwoundTo(60))]
#[case::ahead_on_fork_just_below_reth_tip(forked(0, 59, 65), Outcome::UnwoundTo(59))]
#[case::ahead_on_fork_below_reth_tip(forked(0, 40, 65), Outcome::UnwoundTo(40))]
#[case::at_reth_tip_on_fork(forked(0, 40, 60), Outcome::UnwoundTo(40))]
#[case::below_reth_tip_on_fork(forked(0, 40, 58), Outcome::UnwoundTo(40))]
#[case::tip_replaced_below_reth_tip(forked(0, 57, 58), Outcome::UnwoundTo(57))]
#[case::one_block_above_window_start_on_fork(forked(40, 40, 41), Outcome::UnwoundTo(40))]
#[case::fork_from_window_start(forked(30, 30, 58), Outcome::UnwoundTo(30))]
#[case::fork_below_window(forked(30, 29, 58), Outcome::Fails("shares no block"))]
#[case::window_above_reth_tip(canonical(70, 75), Outcome::Fails("starts at block 70"))]
#[tokio::test]
async fn startup_reconciles_storage_with_reth_chain(
    #[case] shape: StoreShape,
    #[case] expected: Outcome,
    #[values(PERSISTENCE_THRESHOLD_DEVELOP, PERSISTENCE_THRESHOLD_RETH_2_6)] reth_in_memory_blocks: u64,
    #[values(Backend::Mdbx, Backend::InMemory)] backend: Backend,
) {
    with_store!(backend, |store| startup_case(store, shape, expected, reth_in_memory_blocks).await);
}

async fn uninitialized_case<S: OpProofsStore + Clone + 'static>(store: S) {
    let (ctx, _handle) =
        reth_exex_test_utils::test_exex_context().await.expect("exex test context");

    let exex = build_test_exex(ctx, store.clone());
    let err = startup(&exex).expect_err("startup should fail");

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

/// Seeds the proofs storage ahead of reth, runs startup, then has reth re-import the blocks above
/// its best block and checks that the ExEx indexes them on top of the unwound storage.
async fn reimport_case<S: OpProofsStore + Clone + 'static>(
    store: S,
    shape: StoreShape,
    cadence: FlushCadence,
) {
    let reth = reth_at_tip(PERSISTENCE_THRESHOLD_DEVELOP).await;
    let provider = reth.ctx.components.provider.clone();
    let evm_config = reth.ctx.components.components.evm_config.clone();
    seed_store(&store, &reth.chain, shape);

    let exex = build_test_exex(reth.ctx, store.clone());
    startup(&exex).expect("startup should succeed");
    assert_eq!(
        proof_window(&store).expect("proof window").latest,
        reth.chain[RETH_TIP as usize].num_hash()
    );

    let reimported = &reth.chain[RETH_TIP as usize + 1..=shape.tip as usize];
    provider
        .canonical_in_memory_state()
        .update_chain(NewCanonicalChain::Commit { new: executed_blocks(reimported) });
    provider.set_canonical_head(reth.chain[shape.tip as usize].clone_sealed_header());

    let trie_data = reimported
        .iter()
        .map(|block| {
            let hashed_state = Arc::new(account_diff(ACCOUNT, block.number()).sorted_post_state);
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
    engine_handle.flush();

    let provider_ro = store.provider_ro().expect("provider ro");
    assert_eq!(
        provider_ro.get_latest_block().expect("latest block"),
        reth.chain[shape.tip as usize].num_hash()
    );
    let state_root = StateRoot::overlay_root(&provider_ro, shape.tip, HashedPostState::default())
        .expect("state root");
    assert_eq!(state_root, canonical_state_root(shape.tip));
}

/// After the startup unwind, reth's re-import of the blocks above its best block is indexed on
/// top of the storage: the same blocks when reth only dropped them, replacements when the chain
/// moved on while the node was down. The fork's state must not survive in the storage.
#[rstest]
#[case::same_blocks(canonical(0, 65))]
#[case::replaced_blocks(forked(0, 60, 65))]
#[tokio::test]
async fn startup_unwind_lets_reth_reimport_be_indexed(
    #[case] shape: StoreShape,
    #[values(FLUSH_EVERY_BLOCK, FLUSH_EVERY_FIVE_BLOCKS)] cadence: FlushCadence,
    #[values(Backend::Mdbx, Backend::InMemory)] backend: Backend,
) {
    with_store!(backend, |store| reimport_case(store, shape, cadence).await);
}

/// Seeds the proofs storage at `store_earliest..=RETH_TIP` on reth's chain, prunes it down to
/// `min_block_interval` blocks, and checks the window's earliest block.
async fn prune_case<S: OpProofsStore + Clone + 'static>(
    store: S,
    store_earliest: u64,
    reth_in_memory_blocks: u64,
    min_block_interval: u64,
    expected_earliest: u64,
) {
    let reth = reth_at_tip(reth_in_memory_blocks).await;
    seed_store(&store, &reth.chain, canonical(store_earliest, RETH_TIP));

    let provider = reth.ctx.components.provider.clone();
    let pruner = OpProofStoragePruner::new(store.clone(), provider, min_block_interval);
    let provider_rw = store.provider_rw().expect("provider rw");
    pruner.prune_with_provider(&provider_rw).expect("prune");
    OpProofsProviderRw::commit(provider_rw).expect("commit");

    let window = proof_window(&store).expect("proof window");
    assert_eq!(window.earliest, reth.chain[expected_earliest as usize].num_hash());
    assert_eq!(window.latest, reth.chain[RETH_TIP as usize].num_hash());
}

/// The pruner moves the window's earliest block to `latest - min_block_interval`, but never above
/// reth's persisted head: reth may drop its in-memory blocks at a crash, and the storage can only
/// be unwound down to its earliest block. Columns: storage earliest, reth in-memory blocks,
/// `min_block_interval`, expected earliest.
#[rstest]
#[case::capped_at_persisted_head(0, PERSISTENCE_THRESHOLD_DEVELOP, 2, 53)]
#[case::capped_at_persisted_head_threshold_50(0, PERSISTENCE_THRESHOLD_RETH_2_6, 2, 10)]
#[case::below_persisted_head(0, PERSISTENCE_THRESHOLD_DEVELOP, 20, 40)]
#[case::below_persisted_head_threshold_50(0, PERSISTENCE_THRESHOLD_RETH_2_6, 55, 5)]
#[case::nothing_persisted_above_earliest(0, RETH_TIP, 2, 0)]
#[case::persisted_head_below_earliest(30, PERSISTENCE_THRESHOLD_RETH_2_6, 2, 30)]
#[tokio::test]
async fn pruner_keeps_earliest_at_or_below_reth_persisted_head(
    #[case] store_earliest: u64,
    #[case] reth_in_memory_blocks: u64,
    #[case] min_block_interval: u64,
    #[case] expected_earliest: u64,
    #[values(Backend::Mdbx, Backend::InMemory)] backend: Backend,
) {
    with_store!(backend, |store| prune_case(
        store,
        store_earliest,
        reth_in_memory_blocks,
        min_block_interval,
        expected_earliest
    )
    .await);
}

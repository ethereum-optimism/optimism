//! Delegated derivation end to end: the real actors against a scripted delegate and a strict EL.

use super::{
    DelegateDerivationActor,
    actor::DERIVATION_DELEGATE_POLL_INTERVAL as POLL_INTERVAL,
    test_utils::{ScriptedDelegate, sync_status},
};
use crate::{
    EngineActor, EngineActorRequest, NodeActor, QueuedDerivationEngineClient,
    QueuedEngineDerivationClient, ResetRequest,
    actors::engine::{INITIAL_RETRY_DELAY, MAX_RETRY_DELAY, WAIT_RETRY_INTERVAL},
};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use alloy_rpc_types_engine::PayloadStatusEnum;
use kona_derive::test_utils::TestChainProvider;
use kona_engine::{
    Engine, EngineState, EngineSyncState, EngineSyncStateUpdate, NoopBlockSink,
    test_utils::{
        RpcMock, TestEngineStateBuilder, expect_sync_start, test_engine_client, test_l2_block,
        test_rpc_block,
    },
};
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo, SyncStatus};
use op_alloy_consensus::OpBlock;
use std::sync::Arc;
use tokio::{
    sync::{mpsc, watch},
    task::{JoinError, JoinHandle},
    time::{self, Duration, Instant},
};

/// How long the engine may take to reach the labels a step expects.
const LABELS_DEADLINE: Duration = Duration::from_secs(1);

/// The longest an engine task left queued or parked waits before it retries.
const LONGEST_RETRY_DELAY: Duration = if MAX_RETRY_DELAY.as_nanos() > WAIT_RETRY_INTERVAL.as_nanos()
{
    MAX_RETRY_DELAY
} else {
    WAIT_RETRY_INTERVAL
};

/// Long enough for any task the engine left queued or parked to retry with an unexpected request.
const QUIET_TIME: Duration = LONGEST_RETRY_DELAY.saturating_add(INITIAL_RETRY_DELAY);

/// Height of the canonical chain the scenarios draw blocks from.
const CHAIN_LENGTH: u64 = 40;

/// The L1 origin every test L2 block reports, which the delegate actor checks is canonical.
const L1_ORIGIN: BlockInfo =
    BlockInfo { hash: B256::ZERO, number: 0, parent_hash: B256::ZERO, timestamp: 0 };

/// The canonical L1 block a delegate reports as its `current_l1` when its safe head is at
/// `number`, so each accepted poll publishes a distinct derivation origin.
fn l1_block(number: u64) -> BlockInfo {
    if number == 0 {
        return L1_ORIGIN;
    }
    BlockInfo { hash: B256::with_last_byte(number as u8), number, ..Default::default() }
}

/// A canonical L2 chain of decodable blocks, linked by parent hash from the configured genesis.
struct TestChain {
    blocks: Vec<OpBlock>,
    config: Arc<RollupConfig>,
    infos: Vec<L2BlockInfo>,
}

impl TestChain {
    fn new() -> Self {
        let genesis = OpBlock::default();
        let mut config = RollupConfig::default();
        config.genesis.l2 = BlockNumHash { hash: genesis.header.hash_slow(), number: 0 };
        let mut blocks = vec![genesis];
        for number in 1..=CHAIN_LENGTH {
            let parent_hash = blocks[number as usize - 1].header.hash_slow();
            blocks.push(test_l2_block(number, parent_hash, 0));
        }
        let infos = blocks.iter().map(|block| l2_block_info(block, &config)).collect();
        Self { blocks, config: Arc::new(config), infos }
    }

    fn block(&self, number: u64) -> L2BlockInfo {
        self.infos[number as usize]
    }

    /// A block at `number` on a fork that leaves the canonical chain right below it.
    fn fork_block(&self, number: u64) -> L2BlockInfo {
        let parent_hash = self.block(number - 1).block_info.hash;
        l2_block_info(&test_l2_block(number, parent_hash, 1), &self.config)
    }

    fn el_serves(&self, el: &RpcMock, number: u64) {
        el.expect_block_by_number(number, Some(test_rpc_block(&self.blocks[number as usize])));
    }

    fn el_serves_header(&self, el: &RpcMock, number: u64) {
        el.expect_block_header_by_number(
            number,
            Some(test_rpc_block(&self.blocks[number as usize])),
        );
    }

    fn el_lacks(&self, el: &RpcMock, number: u64) {
        el.expect_block_by_number(number, None);
    }

    /// Expects the reset's sync start to find the execution layer's labels all at genesis.
    fn el_reset_to_genesis(&self, l1: &RpcMock, el: &RpcMock) {
        let genesis = self.block(0);
        expect_sync_start(
            l1,
            el,
            &test_rpc_block(&self.blocks[0]),
            all_at(genesis).sync_state.create_forkchoice_state(),
        );
    }
}

fn l2_block_info(block: &OpBlock, config: &RollupConfig) -> L2BlockInfo {
    L2BlockInfo::from_block_and_genesis(block, &config.genesis).unwrap()
}

/// A synced engine whose local-safe and safe heads are the same block, as before interop.
fn synced_engine(
    finalized: L2BlockInfo,
    safe: L2BlockInfo,
    unsafe_head: L2BlockInfo,
) -> EngineState {
    TestEngineStateBuilder::new()
        .with_finalized_head(finalized)
        .with_safe_head(safe)
        .with_unsafe_head(unsafe_head)
        .build()
}

fn all_at(block: L2BlockInfo) -> EngineState {
    synced_engine(block, block, block)
}

/// The label moves the scenarios expect of the engine.
trait LabelMoves {
    fn with_finalized(self, finalized: L2BlockInfo) -> Self;

    /// Local-safe and safe both at `safe`, as before interop.
    fn with_safe(self, safe: L2BlockInfo) -> Self;

    fn with_safe_heads(self, local_safe: L2BlockInfo, safe: L2BlockInfo) -> Self;

    /// The unsafe head moved, forward or back, to a delegated safe head.
    fn with_safe_and_unsafe(self, safe: L2BlockInfo) -> Self;
}

impl LabelMoves for EngineSyncState {
    fn with_finalized(self, finalized: L2BlockInfo) -> Self {
        self.apply_update(EngineSyncStateUpdate {
            finalized_head: Some(finalized),
            ..Default::default()
        })
    }

    fn with_safe(self, safe: L2BlockInfo) -> Self {
        self.with_safe_heads(safe, safe)
    }

    fn with_safe_heads(self, local_safe: L2BlockInfo, safe: L2BlockInfo) -> Self {
        self.apply_update(EngineSyncStateUpdate {
            local_safe_head: Some(local_safe),
            safe_head: Some(safe),
            ..Default::default()
        })
    }

    fn with_safe_and_unsafe(self, safe: L2BlockInfo) -> Self {
        self.with_safe(safe)
            .apply_update(EngineSyncStateUpdate { unsafe_head: Some(safe), ..Default::default() })
    }
}

fn pre_interop_status(safe: L2BlockInfo, finalized: L2BlockInfo) -> SyncStatus {
    sync_status(l1_block(safe.block_info.number), safe, safe, finalized)
}

/// A delegated kona-node whose delegate and engine actors run as tasks, as in production.
struct DelegatedNode {
    answered_polls: watch::Receiver<u32>,
    answered_polls_seen: u32,
    delegate: ScriptedDelegate,
    delegate_task: JoinHandle<()>,
    derivation_origin: watch::Receiver<Option<BlockInfo>>,
    el: RpcMock,
    engine_requests: mpsc::Sender<EngineActorRequest>,
    engine_state: watch::Receiver<EngineState>,
    engine_task: JoinHandle<()>,
    l1: RpcMock,
    polls: watch::Receiver<u32>,
    polls_checked: u32,
    polls_checked_at: Instant,
}

impl DelegatedNode {
    /// Starts the node at `initial`; the delegate first polls once the engine reports sync
    /// complete.
    fn start(chain: &TestChain, initial: EngineState) -> Self {
        let (client, l1, el) = test_engine_client(chain.config.clone());
        let (derivation_request_tx, derivation_request_rx) = mpsc::channel(32);
        let (engine_requests, engine_request_rx) = mpsc::channel(32);
        let (engine_state_tx, engine_state) = watch::channel(initial);
        let engine = EngineActor::new(
            Arc::new(client),
            chain.config.clone(),
            QueuedEngineDerivationClient { derivation_actor_request_tx: derivation_request_tx },
            Engine::new(initial, engine_state_tx),
            None,
            engine_request_rx,
            Arc::new(NoopBlockSink),
        );

        let mut canonical_l1 = TestChainProvider::default();
        for number in 0..=CHAIN_LENGTH {
            canonical_l1.insert_block(number, l1_block(number));
        }
        let delegate = ScriptedDelegate::lenient();
        let (derivation_origin_tx, derivation_origin) = watch::channel(None);
        let delegate_actor = DelegateDerivationActor::new(
            QueuedDerivationEngineClient::new(engine_requests.clone()),
            derivation_request_rx,
            delegate.clone(),
            canonical_l1,
            derivation_origin_tx,
            engine_state.clone(),
        );

        Self {
            answered_polls: delegate.answered_polls(),
            answered_polls_seen: 0,
            delegate_task: run(delegate_actor, "delegate"),
            derivation_origin,
            el,
            engine_requests,
            engine_state,
            engine_task: run(engine, "engine"),
            l1,
            polls: delegate.polls(),
            polls_checked: 0,
            polls_checked_at: Instant::now(),
            delegate,
        }
    }

    /// Scripts the next poll's reply: `safe` as both the local-safe and the safe head.
    fn delegate_reports(&self, safe: L2BlockInfo, finalized: L2BlockInfo) {
        self.delegate.reports(pre_interop_status(safe, finalized));
    }

    /// Requests an engine reset, which runs on the engine's next step.
    async fn request_reset(&self) {
        let (result_tx, _result_rx) = mpsc::channel(1);
        self.engine_requests
            .send(EngineActorRequest::Reset(Box::new(ResetRequest { result_tx })))
            .await
            .unwrap();
    }

    /// Waits for the delegate to answer the scripted poll.
    async fn next_poll(&mut self) {
        if time::timeout(POLL_INTERVAL * 2, self.answered_polls.changed()).await.is_err() {
            self.assert_running().await;
            panic!("the delegate did not answer a poll within {:?}", POLL_INTERVAL * 2);
        }
        let answered = *self.answered_polls.borrow_and_update();
        assert_eq!(answered, self.answered_polls_seen + 1, "the scenario missed a delegate poll");
        self.answered_polls_seen = answered;
        self.assert_poll_cadence();
    }

    /// Fails if the delegate was polled more often than once per poll interval since the last
    /// check. Unscripted polls fail and change nothing, so only their count can catch an actor that
    /// polls too often.
    fn assert_poll_cadence(&mut self) {
        let polls = *self.polls.borrow();
        let elapsed = Instant::now() - self.polls_checked_at;
        let allowed = elapsed.as_nanos() / POLL_INTERVAL.as_nanos() + 1;
        let polled = u128::from(polls - self.polls_checked);
        assert!(
            polled <= allowed,
            "the delegate was polled {polled} times in {elapsed:?}, at most {allowed} expected"
        );
        self.polls_checked = polls;
        self.polls_checked_at = Instant::now();
    }

    /// Waits `duration`, failing if the delegate answers a poll meanwhile and its requests mix with
    /// these. Polls with no reply scripted fail and change nothing.
    async fn wait_without_answered_poll(&mut self, duration: Duration) {
        time::sleep(duration).await;
        assert_eq!(
            *self.answered_polls.borrow(),
            self.answered_polls_seen,
            "the delegate answered a poll during a {duration:?} wait"
        );
        self.assert_poll_cadence();
        self.assert_running().await;
    }

    /// Fails with every stopped actor's panic, so a secondary failure cannot hide the first.
    async fn assert_running(&mut self) {
        let mut stopped = Vec::new();
        for (name, task) in
            [("engine", &mut self.engine_task), ("delegate", &mut self.delegate_task)]
        {
            if task.is_finished() {
                stopped.push(format!("{name}: {}", stop_reason(task.await)));
            }
        }
        assert!(stopped.is_empty(), "actors stopped:\n{}", stopped.join("\n"));
    }

    /// Waits for the engine to reach `expected`, then asserts it, and every scripted reply and
    /// request consumed, once any task the engine left queued or parked has had time to retry.
    async fn assert_labels(&mut self, expected: EngineSyncState) {
        let reached = time::timeout(
            LABELS_DEADLINE,
            self.engine_state.wait_for(|state| state.sync_state == expected),
        )
        .await
        .is_ok_and(|watched| watched.is_ok());
        if !reached {
            self.assert_running().await;
            panic!(
                "the engine did not reach the expected labels within {LABELS_DEADLINE:?}: {:?}",
                self.engine_state.borrow().sync_state
            );
        }
        self.wait_without_answered_poll(QUIET_TIME).await;
        self.delegate.assert_finished();
        self.l1.assert_finished();
        self.el.assert_finished();
        assert_eq!(self.engine_state.borrow().sync_state, expected);
    }

    /// Asserts the derivation origin the delegate actor last published.
    fn assert_origin(&self, expected: Option<BlockInfo>) {
        assert_eq!(*self.derivation_origin.borrow(), expected, "published derivation origin");
    }

    /// Polls canonical `safe` and `finalized`, each applied by a forkchoice update; returns the
    /// labels. A safe head below the unsafe head is looked up first; one at it extends the chain.
    async fn poll_applies_safe_and_finalized(
        &mut self,
        chain: &TestChain,
        labels: EngineSyncState,
        safe: u64,
        finalized: u64,
    ) -> EngineSyncState {
        self.delegate_reports(chain.block(safe), chain.block(finalized));
        if safe < labels.unsafe_head().block_info.number {
            chain.el_serves(&self.el, safe);
        }
        let labels = labels.with_safe(chain.block(safe));
        self.expect_forkchoice(labels, PayloadStatusEnum::Valid);
        chain.el_serves(&self.el, finalized);
        let labels = labels.with_finalized(chain.block(finalized));
        self.expect_forkchoice(labels, PayloadStatusEnum::Valid);
        self.next_poll().await;
        self.assert_labels(labels).await;
        self.assert_origin(Some(l1_block(safe)));
        labels
    }

    fn expect_forkchoice(&self, labels: EngineSyncState, status: PayloadStatusEnum) {
        self.el.expect_forkchoice_updated(labels.create_forkchoice_state(), status);
    }
}

impl Drop for DelegatedNode {
    /// Stops both actors, also when a scenario panics, so their later failures cannot interleave
    /// with its message.
    fn drop(&mut self) {
        self.engine_task.abort();
        self.delegate_task.abort();
    }
}

/// Why an actor task stopped: its panic message, or the join error.
fn stop_reason(result: Result<(), JoinError>) -> String {
    match result {
        Ok(()) => "returned".to_string(),
        Err(err) if err.is_panic() => {
            let payload = err.into_panic();
            payload
                .downcast_ref::<String>()
                .cloned()
                .or_else(|| payload.downcast_ref::<&str>().map(|message| (*message).to_string()))
                .unwrap_or_else(|| "a panic without a message".to_string())
        }
        Err(err) => err.to_string(),
    }
}

/// Steps `actor` until it fails, which ends the task with a panic carrying the error.
fn run<Actor>(mut actor: Actor, name: &'static str) -> JoinHandle<()>
where
    Actor: NodeActor + Send + 'static,
    Actor::Error: std::fmt::Debug,
{
    tokio::spawn(async move {
        loop {
            if let Err(err) = actor.step().await {
                panic!("{name} actor stopped: {err:?}");
            }
        }
    })
}

/// With no unsafe blocks, safe and finalized follow the delegate by hash and match the last FCU.
#[tokio::test(start_paused = true)]
async fn safe_and_finalized_progress_by_hash_with_unsafe_production_paused() {
    let chain = TestChain::new();
    let initial = synced_engine(chain.block(5), chain.block(10), chain.block(30));
    let mut node = DelegatedNode::start(&chain, initial);

    let mut labels = initial.sync_state;
    for (safe, finalized) in [(20, 10), (25, 20), (30, 25)] {
        labels = node.poll_applies_safe_and_finalized(&chain, labels, safe, finalized).await;
    }
}

/// A delegated safe head on another branch reorgs the EL onto it: the EL syncs the branch while
/// the shared ancestor finalizes, then validates it on the next poll.
#[tokio::test(start_paused = true)]
async fn delegated_safe_head_on_another_branch_reorgs_the_el_onto_it() {
    let chain = TestChain::new();
    let initial = synced_engine(chain.block(5), chain.block(10), chain.block(30));
    let (fork_safe, shared_finalized) = (chain.fork_block(25), chain.block(20));
    let reorged = initial.sync_state.with_safe_and_unsafe(fork_safe);
    let finalized_on_branch = reorged.with_finalized(shared_finalized);
    let mut node = DelegatedNode::start(&chain, initial);

    // The EL still has the canonical block at 25, which proves the divergence, and syncs the
    // branch.
    node.delegate_reports(fork_safe, shared_finalized);
    chain.el_serves(&node.el, fork_safe.block_info.number);
    node.expect_forkchoice(reorged, PayloadStatusEnum::Syncing);
    chain.el_serves(&node.el, shared_finalized.block_info.number);
    node.expect_forkchoice(finalized_on_branch, PayloadStatusEnum::Syncing);
    node.next_poll().await;
    node.assert_labels(finalized_on_branch).await;

    // Once the EL has the branch, the same target extends the engine's chain and is validated.
    node.delegate_reports(fork_safe, shared_finalized);
    node.expect_forkchoice(finalized_on_branch, PayloadStatusEnum::Valid);
    node.next_poll().await;
    node.assert_labels(finalized_on_branch).await;
}

/// An unchanged target is resent through `SYNCING` until `VALID`, then finalizes once.
#[tokio::test(start_paused = true)]
async fn source_ahead_catch_up_retries_unchanged_target_through_syncing_until_valid() {
    let chain = TestChain::new();
    let initial = synced_engine(chain.block(5), chain.block(10), chain.block(10));
    let (target_safe, target_finalized) = (chain.block(30), chain.block(20));
    let catching_up = initial.sync_state.with_safe_and_unsafe(target_safe);
    let caught_up = catching_up.with_finalized(target_finalized);
    let mut node = DelegatedNode::start(&chain, initial);

    for _ in 0..2 {
        node.delegate_reports(target_safe, target_finalized);
        node.expect_forkchoice(catching_up, PayloadStatusEnum::Syncing);
        chain.el_lacks(&node.el, target_finalized.block_info.number);
        node.next_poll().await;
        node.assert_labels(catching_up).await;
    }

    node.delegate_reports(target_safe, target_finalized);
    node.expect_forkchoice(catching_up, PayloadStatusEnum::Valid);
    chain.el_serves(&node.el, target_finalized.block_info.number);
    node.expect_forkchoice(caught_up, PayloadStatusEnum::Valid);
    node.next_poll().await;
    node.assert_labels(caught_up).await;

    node.delegate_reports(target_safe, target_finalized);
    node.expect_forkchoice(caught_up, PayloadStatusEnum::Valid);
    node.next_poll().await;
    node.assert_labels(caught_up).await;
}

/// A stale or diverged finalized reference from the delegate.
#[derive(Clone, Copy, Debug)]
enum StaleFinalized {
    /// A different block above the engine's finalized head, at a height the execution layer has.
    ConflictingAbove,
    /// A different block at the engine's finalized height.
    ConflictingAtFinalizedHeight,
    /// A block below the engine's finalized head, from a delegate that lags.
    Regressed,
}

impl StaleFinalized {
    /// The status a stale delegate reports while the engine is safe at 20 and finalized at 15.
    fn status(self, chain: &TestChain) -> SyncStatus {
        match self {
            Self::ConflictingAbove => pre_interop_status(chain.block(20), chain.fork_block(17)),
            Self::ConflictingAtFinalizedHeight => {
                pre_interop_status(chain.block(20), chain.fork_block(15))
            }
            Self::Regressed => pre_interop_status(chain.block(18), chain.block(12)),
        }
    }

    /// What the EL is asked: only a conflicting target above finality reaches the engine.
    fn expect_engine_requests(
        self,
        chain: &TestChain,
        node: &DelegatedNode,
        labels: EngineSyncState,
    ) {
        match self {
            Self::ConflictingAbove => {
                chain.el_serves(&node.el, 20);
                node.expect_forkchoice(labels, PayloadStatusEnum::Valid);
                chain.el_serves(&node.el, 17);
            }
            Self::ConflictingAtFinalizedHeight | Self::Regressed => {}
        }
    }
}

/// Starts a node safe at 20 whose first poll finalizes block 15.
async fn start_finalized_at_15(chain: &TestChain) -> (DelegatedNode, EngineSyncState) {
    let initial = synced_engine(chain.block(10), chain.block(20), chain.block(30));
    let mut node = DelegatedNode::start(chain, initial);
    let finalized_at_15 =
        node.poll_applies_safe_and_finalized(chain, initial.sync_state, 20, 15).await;
    (node, finalized_at_15)
}

/// Stale or conflicting finality never moves finalized; the next canonical one applies.
#[rstest::rstest]
#[case::conflicting_above(StaleFinalized::ConflictingAbove, 25, 20)]
#[case::conflicting_above_then_canonical_at_its_height(StaleFinalized::ConflictingAbove, 20, 17)]
#[case::conflicting_at_finalized_height(StaleFinalized::ConflictingAtFinalizedHeight, 25, 20)]
#[case::regressed(StaleFinalized::Regressed, 25, 20)]
#[tokio::test(start_paused = true)]
async fn stale_source_finalized_reference_never_moves_finalized_back(
    #[case] stale: StaleFinalized,
    #[case] recovered_safe: u64,
    #[case] recovered_finalized: u64,
) {
    let chain = TestChain::new();
    let (mut node, finalized_at_15) = start_finalized_at_15(&chain).await;

    node.delegate.reports(stale.status(&chain));
    stale.expect_engine_requests(&chain, &node, finalized_at_15);
    node.next_poll().await;
    node.assert_labels(finalized_at_15).await;
    node.assert_origin(Some(l1_block(20)));

    node.poll_applies_safe_and_finalized(
        &chain,
        finalized_at_15,
        recovered_safe,
        recovered_finalized,
    )
    .await;
}

/// Local-safe and cross-safe labels move separately; finalizing above cross-safe is rejected whole.
#[tokio::test(start_paused = true)]
async fn local_safe_ahead_of_cross_safe_moves_each_label_separately_and_finalizes_below_cross_safe()
{
    let chain = TestChain::new();
    let initial = synced_engine(chain.block(5), chain.block(10), chain.block(35));
    let mut node = DelegatedNode::start(&chain, initial);

    node.delegate.reports(sync_status(
        L1_ORIGIN,
        chain.block(25),
        chain.block(20),
        chain.block(18),
    ));
    chain.el_serves(&node.el, 25);
    chain.el_serves_header(&node.el, 20);
    let cross_safe_at_20 = initial.sync_state.with_safe_heads(chain.block(25), chain.block(20));
    node.expect_forkchoice(cross_safe_at_20, PayloadStatusEnum::Valid);
    chain.el_serves(&node.el, 18);
    let finalized_at_18 = cross_safe_at_20.with_finalized(chain.block(18));
    node.expect_forkchoice(finalized_at_18, PayloadStatusEnum::Valid);
    node.next_poll().await;
    node.assert_labels(finalized_at_18).await;

    node.delegate.reports(sync_status(
        L1_ORIGIN,
        chain.block(30),
        chain.block(22),
        chain.block(25),
    ));
    node.next_poll().await;
    node.assert_labels(finalized_at_18).await;

    node.delegate.reports(sync_status(
        L1_ORIGIN,
        chain.block(32),
        chain.block(28),
        chain.block(25),
    ));
    chain.el_serves(&node.el, 32);
    chain.el_serves_header(&node.el, 28);
    let cross_safe_at_28 = finalized_at_18.with_safe_heads(chain.block(32), chain.block(28));
    node.expect_forkchoice(cross_safe_at_28, PayloadStatusEnum::Valid);
    chain.el_serves(&node.el, 25);
    let finalized_at_25 = cross_safe_at_28.with_finalized(chain.block(25));
    node.expect_forkchoice(finalized_at_25, PayloadStatusEnum::Valid);
    node.next_poll().await;
    node.assert_labels(finalized_at_25).await;
}

/// From a wiped EL, a reset returns to genesis and the target finalizes once the EL has it.
#[tokio::test(start_paused = true)]
async fn reset_to_genesis_while_finalized_is_missing_finalizes_once_the_el_has_it() {
    let chain = TestChain::new();
    let genesis = all_at(chain.block(0)).sync_state;
    let (target_safe, target_finalized) = (chain.block(30), chain.block(20));
    let catching_up = genesis.with_safe_and_unsafe(target_safe);
    let caught_up = catching_up.with_finalized(target_finalized);
    let mut node = DelegatedNode::start(&chain, EngineState::default());

    // The startup reset finds only genesis, and the first poll starts the EL's sync.
    chain.el_reset_to_genesis(&node.l1, &node.el);
    node.delegate_reports(target_safe, target_finalized);
    node.expect_forkchoice(catching_up, PayloadStatusEnum::Syncing);
    chain.el_lacks(&node.el, target_finalized.block_info.number);
    node.request_reset().await;
    node.next_poll().await;
    node.assert_labels(catching_up).await;

    chain.el_reset_to_genesis(&node.l1, &node.el);
    node.request_reset().await;
    node.assert_labels(genesis).await;

    node.delegate_reports(target_safe, target_finalized);
    node.expect_forkchoice(catching_up, PayloadStatusEnum::Syncing);
    chain.el_lacks(&node.el, target_finalized.block_info.number);
    node.next_poll().await;
    node.assert_labels(catching_up).await;

    node.delegate_reports(target_safe, target_finalized);
    node.expect_forkchoice(catching_up, PayloadStatusEnum::Valid);
    chain.el_serves(&node.el, target_finalized.block_info.number);
    node.expect_forkchoice(caught_up, PayloadStatusEnum::Valid);
    node.next_poll().await;
    node.assert_labels(caught_up).await;
}

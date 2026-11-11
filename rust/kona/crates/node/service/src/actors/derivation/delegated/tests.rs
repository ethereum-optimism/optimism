use super::{
    DelegateDerivationActor, DerivationDelegateClientError,
    actor::{
        CHAIN_BELOW_FINALITY_RETRY_INTERVAL, DERIVATION_DELEGATE_POLL_INTERVAL as POLL_INTERVAL,
    },
    test_utils::{ScriptedDelegate, SyncStatusReply, sync_status},
};
use crate::{DerivationActorRequest, EngineActorRequest, NodeActor, QueuedDerivationEngineClient};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use kona_derive::{ResetSignal, Signal, test_utils::TestChainProvider};
use kona_engine::{
    ConsolidateInput, DelegatedRejection, DelegatedSafeHeads, EngineState, EngineSyncState,
    EngineSyncStateUpdate, FinalizeBlockId,
};
use kona_protocol::{BlockInfo, L2BlockInfo, SyncStatus};
use rstest::rstest;
use tokio::{
    sync::{mpsc, watch},
    time::{self, Instant},
};

type HarnessActor =
    DelegateDerivationActor<QueuedDerivationEngineClient, ScriptedDelegate, TestChainProvider>;

const SAFE_ORIGIN: u64 = 1;
const FINALIZED_ORIGIN: u64 = 2;
const CURRENT_L1: u64 = 3;

/// A delegate actor over a canonical L1 chain, with the engine requests it sends and the
/// `current_l1` it publishes.
struct Harness {
    actor: HarnessActor,
    current_l1: watch::Receiver<Option<BlockInfo>>,
    delegate: ScriptedDelegate,
    derivation_requests: mpsc::Sender<DerivationActorRequest>,
    engine_requests: mpsc::Receiver<EngineActorRequest>,
    engine_state: watch::Sender<EngineState>,
}

impl Harness {
    fn new(replies: impl IntoIterator<Item = SyncStatusReply>) -> Self {
        let mut l1 = TestChainProvider::default();
        for number in [SAFE_ORIGIN, FINALIZED_ORIGIN, CURRENT_L1] {
            l1.insert_block(number, canonical_l1_block(number));
        }
        let (derivation_requests, derivation_request_rx) = mpsc::channel(8);
        let (engine_request_tx, engine_requests) = mpsc::channel(8);
        let (current_l1_tx, current_l1) = watch::channel(None);
        let (engine_state, engine_state_rx) = watch::channel(EngineState::default());
        let delegate = ScriptedDelegate::strict(replies);
        let actor = HarnessActor::new(
            QueuedDerivationEngineClient::new(engine_request_tx),
            derivation_request_rx,
            delegate.clone(),
            l1,
            current_l1_tx,
            engine_state_rx,
        );
        Self { actor, current_l1, delegate, derivation_requests, engine_requests, engine_state }
    }

    async fn complete_engine_sync(&mut self) {
        self.complete_engine_sync_with(EngineSyncState::default()).await;
    }

    /// Completes the engine sync with `sync_state` as the engine's labels, which the engine
    /// publishes before it reports the completion.
    async fn complete_engine_sync_with(&mut self, sync_state: EngineSyncState) {
        self.publish_engine_sync_state(sync_state);
        self.derivation_requests
            .send(DerivationActorRequest::ProcessEngineSyncCompletionRequest(Box::new(sync_state)))
            .await
            .unwrap();
        self.actor.step().await.unwrap();
    }

    /// Publishes the engine's labels as the engine does after each task it runs.
    fn publish_engine_sync_state(&self, sync_state: EngineSyncState) {
        self.engine_state.send_modify(|state| state.sync_state = sync_state);
    }

    /// Hands the actor `request`, which it handles before its next poll.
    async fn handle(&mut self, request: DerivationActorRequest) {
        self.derivation_requests.send(request).await.unwrap();
        self.actor.step().await.unwrap();
    }

    /// Publishes the engine's labels along with whether its execution layer applied the engine's
    /// last forkchoice.
    fn publish_engine_state(&self, sync_state: EngineSyncState, el_applied_last_forkchoice: bool) {
        self.engine_state.send_modify(|state| {
            state.sync_state = sync_state;
            state.el_applied_last_forkchoice = el_applied_last_forkchoice;
        });
    }

    /// The engine requests sent since the last call, in order.
    fn forwarded(&mut self) -> Vec<Forwarded> {
        std::iter::from_fn(|| self.engine_requests.try_recv().ok()).map(Forwarded::from).collect()
    }
}

impl Drop for Harness {
    fn drop(&mut self) {
        if !std::thread::panicking() {
            self.delegate.assert_finished();
        }
    }
}

/// The parts of an engine request that the delegate actor decides.
#[derive(Debug, PartialEq, Eq)]
enum Forwarded {
    Finalized(FinalizeBlockId),
    Other(String),
    Safe(Box<DelegatedSafeHeads>),
}

impl From<EngineActorRequest> for Forwarded {
    fn from(request: EngineActorRequest) -> Self {
        match request {
            EngineActorRequest::ProcessFinalizedL2Block(finalized) => Self::Finalized(*finalized),
            EngineActorRequest::ProcessSafeL2Signal(ConsolidateInput::Delegated(heads)) => {
                Self::Safe(heads)
            }
            other => Self::Other(format!("{other:?}")),
        }
    }
}

/// What one poll of `status` forwards: the safe heads, then the finalized block pinned by hash.
fn forwarded_for(status: &SyncStatus) -> Vec<Forwarded> {
    vec![
        Forwarded::Safe(Box::new(
            DelegatedSafeHeads::try_new(status.local_safe_l2, status.safe_l2).unwrap(),
        )),
        Forwarded::Finalized(FinalizeBlockId::ByHash(status.finalized_l2.block_info.id())),
    ]
}

/// What one poll forwards when the status is accepted, and nothing when it is rejected.
fn forwarded_if(accepted: bool, status: &SyncStatus) -> Vec<Forwarded> {
    if accepted { forwarded_for(status) } else { Vec::new() }
}

fn canonical_l1_block(number: u64) -> BlockInfo {
    BlockInfo { hash: B256::with_last_byte(number as u8), number, ..Default::default() }
}

fn l2_block(number: u64, l1_origin: u64) -> L2BlockInfo {
    L2BlockInfo {
        block_info: BlockInfo {
            hash: B256::with_last_byte(0x80 | number as u8),
            number,
            ..Default::default()
        },
        l1_origin: canonical_l1_block(l1_origin).id(),
        seq_num: 0,
    }
}

/// A sync status whose safe origin, finalized origin and `current_l1` are all canonical, with an
/// unsafe head above its safe heads.
fn delegate_status() -> SyncStatus {
    SyncStatus {
        unsafe_l2: l2_block(25, SAFE_ORIGIN),
        ..sync_status(
            canonical_l1_block(CURRENT_L1),
            l2_block(20, SAFE_ORIGIN),
            l2_block(20, SAFE_ORIGIN),
            l2_block(15, FINALIZED_ORIGIN),
        )
    }
}

fn non_canonical(id: BlockNumHash) -> BlockNumHash {
    BlockNumHash { hash: B256::repeat_byte(0xee), ..id }
}

/// Returns `status` with its finalized L2 block replaced by `finalized_l2`.
fn with_finalized_l2(finalized_l2: L2BlockInfo) -> SyncStatus {
    SyncStatus { finalized_l2, ..delegate_status() }
}

/// An engine sync state whose finalized head is `finalized_l2`.
fn engine_finalized_at(finalized_l2: L2BlockInfo) -> EngineSyncState {
    EngineSyncState::default().apply_update(EngineSyncStateUpdate {
        finalized_head: Some(finalized_l2),
        ..Default::default()
    })
}

fn conflicting(block: L2BlockInfo) -> L2BlockInfo {
    L2BlockInfo {
        block_info: BlockInfo {
            hash: non_canonical(block.block_info.id()).hash,
            ..block.block_info
        },
        ..block
    }
}

/// The engine's finalized head when its sync completes is the baseline, so a delegate that starts
/// behind it, or on another chain at its height, is rejected whole from the first poll on.
#[rstest]
#[case::advancing(l2_block(16, FINALIZED_ORIGIN), true)]
#[case::unchanged(l2_block(15, FINALIZED_ORIGIN), true)]
#[case::regressed(l2_block(14, FINALIZED_ORIGIN), false)]
#[case::conflicting(conflicting(l2_block(15, FINALIZED_ORIGIN)), false)]
#[tokio::test(start_paused = true)]
async fn first_poll_rejects_status_behind_engine_finality(
    #[case] finalized_l2: L2BlockInfo,
    #[case] accepted: bool,
) {
    let status = with_finalized_l2(finalized_l2);
    let mut harness = Harness::new([Ok(status.clone())]);
    harness.complete_engine_sync_with(engine_finalized_at(l2_block(15, FINALIZED_ORIGIN))).await;

    harness.actor.step().await.unwrap();

    assert_eq!(harness.forwarded(), forwarded_if(accepted, &status));
    assert_eq!(harness.current_l1.borrow().is_some(), accepted);
}

/// Once the engine applies a forwarded finalized block, a status whose finalized block is behind
/// it, or is a different block at its height, is rejected whole: a lagging or diverged delegate's
/// safe head must not reach the engine either, and its `current_l1` is not published.
#[rstest]
#[case::advancing(l2_block(16, FINALIZED_ORIGIN), true)]
#[case::unchanged(l2_block(15, FINALIZED_ORIGIN), true)]
#[case::regressed(l2_block(14, FINALIZED_ORIGIN), false)]
#[case::conflicting(conflicting(l2_block(15, FINALIZED_ORIGIN)), false)]
#[tokio::test(start_paused = true)]
async fn poll_rejects_status_whose_finality_moves_back(
    #[case] next_finalized_l2: L2BlockInfo,
    #[case] accepted: bool,
) {
    let next_status = with_finalized_l2(next_finalized_l2);
    let mut harness = Harness::new([Ok(delegate_status()), Ok(next_status.clone())]);
    harness.complete_engine_sync().await;
    harness.actor.step().await.unwrap();
    assert_eq!(harness.forwarded(), forwarded_for(&delegate_status()));
    harness.publish_engine_sync_state(engine_finalized_at(delegate_status().finalized_l2));
    harness.current_l1.borrow_and_update();

    harness.actor.step().await.unwrap();

    assert_eq!(harness.forwarded(), forwarded_if(accepted, &next_status));
    assert_eq!(harness.current_l1.has_changed().unwrap(), accepted);
}

/// A finalized block the engine skipped, such as one on another chain, does not become the
/// baseline: the canonical block at its height, or below it but above the engine's finalized
/// head, is still accepted, so the delegate's safe head keeps reaching the engine.
#[rstest]
#[case::canonical_at_skipped_height(l2_block(17, FINALIZED_ORIGIN))]
#[case::canonical_below_skipped_height(l2_block(16, FINALIZED_ORIGIN))]
#[tokio::test(start_paused = true)]
async fn poll_accepts_status_after_engine_skipped_forwarded_finality(
    #[case] canonical_finalized_l2: L2BlockInfo,
) {
    let skipped_status = with_finalized_l2(conflicting(l2_block(17, FINALIZED_ORIGIN)));
    let canonical_status = with_finalized_l2(canonical_finalized_l2);
    let mut harness = Harness::new([Ok(skipped_status.clone()), Ok(canonical_status.clone())]);
    harness.complete_engine_sync_with(engine_finalized_at(l2_block(15, FINALIZED_ORIGIN))).await;
    harness.actor.step().await.unwrap();
    assert_eq!(harness.forwarded(), forwarded_for(&skipped_status));

    harness.actor.step().await.unwrap();

    assert_eq!(harness.forwarded(), forwarded_for(&canonical_status));
}

#[derive(Clone, Copy, Debug)]
enum DelegateReply {
    CurrentL1NotCanonical,
    FetchFailed,
    FinalizedOriginNotCanonical,
    LocalSafeOriginNotCanonical,
    SafeOriginNotCanonical,
    Valid,
}

impl DelegateReply {
    fn sync_status(self) -> SyncStatusReply {
        let mut status = delegate_status();
        match self {
            Self::CurrentL1NotCanonical => {
                status.current_l1.hash = non_canonical(status.current_l1.id()).hash
            }
            Self::FetchFailed => {
                return Err(DerivationDelegateClientError::FetchFailed("unreachable".into()));
            }
            Self::FinalizedOriginNotCanonical => {
                status.finalized_l2.l1_origin = non_canonical(status.finalized_l2.l1_origin)
            }
            Self::LocalSafeOriginNotCanonical => {
                status.local_safe_l2.l1_origin = non_canonical(status.local_safe_l2.l1_origin)
            }
            Self::SafeOriginNotCanonical => {
                status.safe_l2.l1_origin = non_canonical(status.safe_l2.l1_origin)
            }
            Self::Valid => {}
        }
        Ok(status)
    }
}

/// Polling starts only once the engine has finished syncing.
#[tokio::test(start_paused = true)]
async fn polls_wait_for_engine_sync_completion() {
    let mut harness = Harness::new([]);

    assert!(time::timeout(POLL_INTERVAL * 3, harness.actor.step()).await.is_err());

    assert_eq!(harness.forwarded(), Vec::new());
    assert_eq!(*harness.current_l1.borrow(), None);
}

/// A valid status forwards the safe head and then the hash-pinned finalized block, and publishes
/// the delegate's `current_l1`. A failed fetch, or any L1 reference that is not canonical, skips
/// the poll without forwarding or publishing anything and without stopping the actor.
#[rstest]
#[case::valid(DelegateReply::Valid, true)]
#[case::fetch_failed(DelegateReply::FetchFailed, false)]
#[case::local_safe_origin_not_canonical(DelegateReply::LocalSafeOriginNotCanonical, false)]
#[case::safe_origin_not_canonical(DelegateReply::SafeOriginNotCanonical, false)]
#[case::finalized_origin_not_canonical(DelegateReply::FinalizedOriginNotCanonical, false)]
#[case::current_l1_not_canonical(DelegateReply::CurrentL1NotCanonical, false)]
#[tokio::test(start_paused = true)]
async fn poll_forwards_only_canonical_status(#[case] reply: DelegateReply, #[case] accepted: bool) {
    let mut harness = Harness::new([reply.sync_status()]);
    harness.complete_engine_sync().await;

    harness.actor.step().await.unwrap();

    assert_eq!(harness.forwarded(), forwarded_if(accepted, &delegate_status()));
    assert_eq!(*harness.current_l1.borrow(), accepted.then(|| delegate_status().current_l1));
}

/// An unchanged status is forwarded again on every poll, one poll interval apart, so a target
/// that could not be applied on one poll is offered again on the next.
#[tokio::test(start_paused = true)]
async fn unchanged_status_is_resent_on_every_poll() {
    const POLLS: u32 = 3;
    let mut harness = Harness::new((0..POLLS).map(|_| DelegateReply::Valid.sync_status()));
    harness.complete_engine_sync().await;
    let first_poll = Instant::now();

    for _ in 0..POLLS {
        harness.actor.step().await.unwrap();
        assert_eq!(harness.forwarded(), forwarded_for(&delegate_status()));
    }

    assert_eq!(Instant::now() - first_poll, POLL_INTERVAL * (POLLS - 1));
}

/// A status whose labels are out of order is rejected whole: a finalized block ahead of the safe
/// head would let a stale snapshot's safe head sit below finality, and a safe head ahead of the
/// local safe head is not a state any delegate reaches.
#[rstest]
#[case::ordered(15, 20, 20, true)]
#[case::finalized_at_safe(20, 20, 20, true)]
#[case::safe_behind_local_safe(15, 18, 20, true)]
#[case::finalized_ahead_of_safe(16, 15, 20, false)]
#[case::safe_ahead_of_local_safe(15, 21, 20, false)]
#[tokio::test(start_paused = true)]
async fn poll_rejects_status_with_labels_out_of_order(
    #[case] finalized: u64,
    #[case] safe: u64,
    #[case] local_safe: u64,
    #[case] accepted: bool,
) {
    let status = SyncStatus {
        finalized_l2: l2_block(finalized, FINALIZED_ORIGIN),
        local_safe_l2: l2_block(local_safe, SAFE_ORIGIN),
        safe_l2: l2_block(safe, SAFE_ORIGIN),
        ..delegate_status()
    };
    let mut harness = Harness::new([Ok(status.clone())]);
    harness.complete_engine_sync().await;

    harness.actor.step().await.unwrap();

    assert_eq!(harness.forwarded(), forwarded_if(accepted, &status));
    assert_eq!(harness.current_l1.borrow().is_some(), accepted);
}

/// The delegate's local-safe and cross-safe heads are forwarded as separate labels, so a cross-safe
/// head that lags the local-safe head under interop is not raised to it, and before interop both
/// carry the same block.
#[rstest]
#[case::pre_interop(20, 20)]
#[case::cross_safe_lags_local_safe(18, 20)]
#[tokio::test(start_paused = true)]
async fn poll_forwards_local_safe_and_cross_safe_heads_separately(
    #[case] safe: u64,
    #[case] local_safe: u64,
) {
    let status = SyncStatus {
        local_safe_l2: l2_block(local_safe, SAFE_ORIGIN),
        safe_l2: l2_block(safe, SAFE_ORIGIN),
        ..delegate_status()
    };
    let mut harness = Harness::new([Ok(status.clone())]);
    harness.complete_engine_sync().await;

    harness.actor.step().await.unwrap();

    let expected_heads =
        DelegatedSafeHeads::try_new(l2_block(local_safe, SAFE_ORIGIN), l2_block(safe, SAFE_ORIGIN))
            .unwrap();
    assert_eq!(harness.forwarded()[0], Forwarded::Safe(Box::new(expected_heads)));
}

/// The safe heads of `status`.
fn safe_heads_of(status: &SyncStatus) -> DelegatedSafeHeads {
    DelegatedSafeHeads::try_new(status.local_safe_l2, status.safe_l2).unwrap()
}

/// What a poll of `status` forwards while its safe heads are rejected: the finalized block alone.
fn forwarded_without_safe_heads(status: &SyncStatus) -> Vec<Forwarded> {
    vec![Forwarded::Finalized(FinalizeBlockId::ByHash(status.finalized_l2.block_info.id()))]
}

/// The engine's report that it rejected the safe heads of `status` for `rejection`.
fn rejected(status: &SyncStatus, rejection: DelegatedRejection) -> DerivationActorRequest {
    DerivationActorRequest::ProcessRejectedDelegatedSafeHeads {
        heads: Box::new(safe_heads_of(status)),
        rejection,
    }
}

/// Safe heads the engine rejected permanently are not sent again while the delegate keeps
/// reporting them, but the finalized block still is; a reset rebuilds the engine's labels, after
/// which they are sent again.
#[tokio::test(start_paused = true)]
async fn rejected_safe_heads_are_not_sent_again_until_a_reset() {
    let status = delegate_status();
    let mut harness = Harness::new((0..3).map(|_| Ok(status.clone())));
    harness.complete_engine_sync().await;
    harness.actor.step().await.unwrap();
    assert_eq!(harness.forwarded(), forwarded_for(&status));
    harness.handle(rejected(&status, DelegatedRejection::HeadLabels)).await;

    harness.actor.step().await.unwrap();
    assert_eq!(harness.forwarded(), forwarded_without_safe_heads(&status));

    let reset = Signal::Reset(ResetSignal { l2_safe_head: L2BlockInfo::default() });
    harness.handle(DerivationActorRequest::ProcessEngineSignalRequest(Box::new(reset))).await;
    harness.actor.step().await.unwrap();
    assert_eq!(harness.forwarded(), forwarded_for(&status));
}

/// Only the exact rejected safe heads are held back: a status with another cross-safe head is
/// sent, since the rejection may have been that head's alone.
#[tokio::test(start_paused = true)]
async fn safe_heads_other_than_the_rejected_ones_are_sent() {
    let rejected_status = delegate_status();
    let next = sync_status(
        canonical_l1_block(CURRENT_L1),
        l2_block(20, SAFE_ORIGIN),
        l2_block(18, SAFE_ORIGIN),
        l2_block(15, FINALIZED_ORIGIN),
    );
    let mut harness = Harness::new([Ok(next.clone())]);
    harness.complete_engine_sync().await;
    harness.handle(rejected(&rejected_status, DelegatedRejection::HeadLabels)).await;

    harness.actor.step().await.unwrap();

    assert_eq!(harness.forwarded(), forwarded_for(&next));
}

/// A status on a delegate chain with blocks tagged `chain`, with its local-safe and safe heads at
/// `safe`, its unsafe head two blocks above and its finalized block at `finalized`. Each block's
/// parent is the block below it on the same chain.
fn chain_status(chain: u8, safe: u64, finalized: u64) -> SyncStatus {
    let block = |number: u64, l1_origin: u64| L2BlockInfo {
        block_info: BlockInfo {
            hash: chain_hash(chain, number),
            number,
            parent_hash: chain_hash(chain, number - 1),
            ..Default::default()
        },
        ..l2_block(number, l1_origin)
    };
    SyncStatus {
        unsafe_l2: block(safe + 2, SAFE_ORIGIN),
        ..sync_status(
            canonical_l1_block(CURRENT_L1),
            block(safe, SAFE_ORIGIN),
            block(safe, SAFE_ORIGIN),
            block(finalized, FINALIZED_ORIGIN),
        )
    }
}

fn chain_hash(chain: u8, number: u64) -> B256 {
    let mut hash = B256::with_last_byte(number as u8);
    hash.0[0] = chain;
    hash
}

const FORK: u8 = 0xf0;
const CANONICAL: u8 = 0xc0;

/// The engine's finalized head when it rejects a delegate's chain below its finality.
fn engine_finalized_below_the_fork() -> EngineSyncState {
    engine_finalized_at(l2_block(10, FINALIZED_ORIGIN))
}

/// Polls `rejected_status`, whose chain the engine then reports below its finality.
async fn reject_chain_below_finality(harness: &mut Harness, rejected_status: &SyncStatus) {
    harness.complete_engine_sync_with(engine_finalized_below_the_fork()).await;
    harness.actor.step().await.unwrap();
    assert_eq!(harness.forwarded(), forwarded_for(rejected_status));
    harness.handle(rejected(rejected_status, DelegatedRejection::ChainBelowFinality)).await;
}

/// After the engine found a delegate's chain below its finality, the delegate's safe heads stay
/// held back while it advances on that chain as a live delegate does: its unsafe head several
/// blocks per poll, its safe heads a span batch at a time and its finality an epoch at a time, so
/// no status shares a block with an earlier one. Its finalized block is still sent.
#[tokio::test(start_paused = true)]
async fn delegate_advancing_on_a_rejected_chain_is_held_back() {
    let rejected_status = chain_status(FORK, 20, 15);
    let advancing =
        [chain_status(FORK, 26, 15), chain_status(FORK, 32, 15), chain_status(FORK, 38, 26)];
    let mut harness = Harness::new(
        std::iter::once(&rejected_status).chain(&advancing).map(|status| Ok(status.clone())),
    );
    reject_chain_below_finality(&mut harness, &rejected_status).await;

    for status in &advancing {
        harness.actor.step().await.unwrap();
        assert_eq!(harness.forwarded(), forwarded_without_safe_heads(status));
    }
}

/// The engine's finality moves only to a canonical finalized target, which shows the delegate is
/// back on the engine's chain, so its safe heads are sent again.
#[tokio::test(start_paused = true)]
async fn engine_finality_moving_lifts_the_hold_on_a_rejected_chain() {
    let rejected_status = chain_status(FORK, 20, 15);
    let canonical = chain_status(CANONICAL, 26, 20);
    let mut harness =
        Harness::new([Ok(rejected_status.clone()), Ok(canonical.clone()), Ok(canonical.clone())]);
    reject_chain_below_finality(&mut harness, &rejected_status).await;
    harness.actor.step().await.unwrap();
    assert_eq!(harness.forwarded(), forwarded_without_safe_heads(&canonical));

    harness.publish_engine_sync_state(engine_finalized_at(canonical.finalized_l2));
    harness.actor.step().await.unwrap();

    assert_eq!(harness.forwarded(), forwarded_for(&canonical));
}

/// Without the engine's finality moving, the delegate's safe heads are sent once more after the
/// retry interval, and held back again if the engine rejects the chain again.
#[tokio::test(start_paused = true)]
async fn delegate_on_a_rejected_chain_is_retried_after_the_interval() {
    let rejected_status = chain_status(FORK, 20, 15);
    let (held, retried, held_again) =
        (chain_status(FORK, 26, 15), chain_status(FORK, 32, 15), chain_status(FORK, 38, 15));
    let mut harness = Harness::new([
        Ok(rejected_status.clone()),
        Ok(held.clone()),
        Ok(retried.clone()),
        Ok(held_again.clone()),
    ]);
    reject_chain_below_finality(&mut harness, &rejected_status).await;
    harness.actor.step().await.unwrap();
    assert_eq!(harness.forwarded(), forwarded_without_safe_heads(&held));

    time::sleep(CHAIN_BELOW_FINALITY_RETRY_INTERVAL).await;
    harness.actor.step().await.unwrap();
    assert_eq!(harness.forwarded(), forwarded_for(&retried));

    harness.handle(rejected(&retried, DelegatedRejection::ChainBelowFinality)).await;
    harness.actor.step().await.unwrap();
    assert_eq!(harness.forwarded(), forwarded_without_safe_heads(&held_again));
}

/// A zero local safe head is a label the delegate has not derived or does not report, so it does
/// not put the safe head out of order and the status is forwarded.
#[tokio::test(start_paused = true)]
async fn unreported_local_safe_head_does_not_order_the_labels() {
    let status = SyncStatus { local_safe_l2: L2BlockInfo::default(), ..delegate_status() };
    let mut harness = Harness::new([Ok(status.clone())]);
    harness.complete_engine_sync().await;

    harness.actor.step().await.unwrap();

    let reported_as_safe = SyncStatus { local_safe_l2: status.safe_l2, ..status };
    assert_eq!(harness.forwarded(), forwarded_for(&reported_as_safe));
}

/// An engine sync state with consistent labels: unsafe at 30, safe at `safe`, finalized at 15.
fn engine_safe_at(safe: u64) -> EngineSyncState {
    EngineSyncState::default().apply_update(EngineSyncStateUpdate {
        finalized_head: Some(l2_block(15, FINALIZED_ORIGIN)),
        local_safe_head: Some(l2_block(safe, SAFE_ORIGIN)),
        safe_head: Some(l2_block(safe, SAFE_ORIGIN)),
        unsafe_head: Some(l2_block(30, SAFE_ORIGIN)),
    })
}

/// Safe heads behind the engine's are forwarded once while the execution layer applied the
/// engine's forkchoice, since the engine's answer to it cannot change; finalized still goes out
/// on every poll. A moved engine safe head, or an execution layer that did not apply the last
/// forkchoice, sends it again.
#[rstest]
#[case::same_engine_view(true, 25, forwarded_without_safe_heads)]
#[case::el_not_applied(false, 25, forwarded_for)]
#[case::engine_safe_moved(true, 26, forwarded_for)]
#[tokio::test(start_paused = true)]
async fn lagging_safe_heads_are_forwarded_once_per_engine_view(
    #[case] el_applied_last_forkchoice: bool,
    #[case] engine_safe_on_second_poll: u64,
    #[case] second_poll_forwards: fn(&SyncStatus) -> Vec<Forwarded>,
) {
    let status = delegate_status();
    let mut harness = Harness::new([Ok(status.clone()), Ok(status.clone())]);
    harness.complete_engine_sync_with(engine_safe_at(25)).await;
    harness.publish_engine_state(engine_safe_at(25), el_applied_last_forkchoice);
    harness.actor.step().await.unwrap();
    assert_eq!(harness.forwarded(), forwarded_for(&status));
    harness.publish_engine_state(
        engine_safe_at(engine_safe_on_second_poll),
        el_applied_last_forkchoice,
    );

    harness.actor.step().await.unwrap();

    assert_eq!(harness.forwarded(), second_poll_forwards(&status));
}

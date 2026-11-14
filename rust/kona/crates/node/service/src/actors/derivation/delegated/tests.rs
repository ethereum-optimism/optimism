use super::{
    DelegateDerivationActor, DerivationDelegateClientError, DerivationDelegateProvider,
    actor::DERIVATION_DELEGATE_POLL_INTERVAL,
};
use crate::{DerivationActorRequest, EngineActorRequest, NodeActor, QueuedDerivationEngineClient};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use async_trait::async_trait;
use kona_derive::test_utils::TestChainProvider;
use kona_engine::{
    ConsolidateInput, EngineState, EngineSyncState, EngineSyncStateUpdate, FinalizeBlockId,
};
use kona_protocol::{BlockInfo, L2BlockInfo, SyncStatus};
use rstest::rstest;
use std::{collections::VecDeque, sync::Mutex};
use tokio::{
    sync::{mpsc, watch},
    time::{self, Instant},
};

const SAFE_ORIGIN: u64 = 1;
const FINALIZED_ORIGIN: u64 = 2;
const CURRENT_L1: u64 = 3;

type SyncStatusReply = Result<SyncStatus, DerivationDelegateClientError>;

/// Answers each sync status poll with the next scripted reply.
#[derive(Debug)]
struct ScriptedDelegate(Mutex<VecDeque<SyncStatusReply>>);

#[async_trait]
impl DerivationDelegateProvider for ScriptedDelegate {
    async fn fetch_sync_status(&self) -> SyncStatusReply {
        self.0.lock().unwrap().pop_front().expect("unexpected sync status poll")
    }
}

/// A delegate actor over a canonical L1 chain, with the engine requests it sends and the
/// `current_l1` it publishes.
struct Harness {
    actor:
        DelegateDerivationActor<QueuedDerivationEngineClient, ScriptedDelegate, TestChainProvider>,
    current_l1: watch::Receiver<Option<BlockInfo>>,
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
        let actor = DelegateDerivationActor::new(
            QueuedDerivationEngineClient::new(engine_request_tx),
            derivation_request_rx,
            ScriptedDelegate(Mutex::new(replies.into_iter().collect())),
            l1,
            current_l1_tx,
            engine_state_rx,
        );
        Self { actor, current_l1, derivation_requests, engine_requests, engine_state }
    }

    async fn complete_engine_sync(&mut self) {
        self.complete_engine_sync_with(EngineSyncState::default()).await;
    }

    /// Completes the engine sync with `sync_state` as the engine's labels, which the engine
    /// publishes before it reports the completion.
    async fn complete_engine_sync_with(&mut self, sync_state: EngineSyncState) {
        self.publish_engine_sync_state(sync_state);
        self.derivation_requests
            .send(DerivationActorRequest::ProcessEngineSyncCompletionRequest(Box::new(
                sync_state.local_safe_head(),
            )))
            .await
            .unwrap();
        self.actor.step().await.unwrap();
    }

    /// Publishes the engine's labels as the engine does after each task it runs.
    fn publish_engine_sync_state(&self, sync_state: EngineSyncState) {
        self.engine_state.send_modify(|state| state.sync_state = sync_state);
    }

    /// The engine requests sent since the last call, in order.
    fn forwarded(&mut self) -> Vec<Forwarded> {
        std::iter::from_fn(|| self.engine_requests.try_recv().ok()).map(Forwarded::from).collect()
    }
}

/// The parts of an engine request that the delegate actor decides.
#[derive(Debug, PartialEq, Eq)]
enum Forwarded {
    Finalized(FinalizeBlockId),
    Other(String),
    Safe(L2BlockInfo),
}

impl From<EngineActorRequest> for Forwarded {
    fn from(request: EngineActorRequest) -> Self {
        match request {
            EngineActorRequest::ProcessFinalizedL2Block(finalized) => Self::Finalized(*finalized),
            EngineActorRequest::ProcessSafeL2Signal(ConsolidateInput::BlockInfo(safe)) => {
                Self::Safe(safe)
            }
            other => Self::Other(format!("{other:?}")),
        }
    }
}

/// What one poll of `status` forwards: the safe head, then the finalized block pinned by hash.
fn forwarded_for(status: &SyncStatus) -> Vec<Forwarded> {
    vec![
        Forwarded::Safe(status.safe_l2),
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

/// A sync status whose safe origin, finalized origin and `current_l1` are all canonical.
fn delegate_status() -> SyncStatus {
    SyncStatus {
        current_l1: canonical_l1_block(CURRENT_L1),
        current_l1_finalized: BlockInfo::default(),
        finalized_l1: BlockInfo::default(),
        finalized_l2: l2_block(15, FINALIZED_ORIGIN),
        head_l1: BlockInfo::default(),
        local_safe_l2: l2_block(20, SAFE_ORIGIN),
        safe_l1: BlockInfo::default(),
        safe_l2: l2_block(20, SAFE_ORIGIN),
        unsafe_l2: l2_block(25, SAFE_ORIGIN),
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

    assert!(
        time::timeout(DERIVATION_DELEGATE_POLL_INTERVAL * 3, harness.actor.step()).await.is_err()
    );

    assert_eq!(harness.forwarded(), Vec::new());
    assert_eq!(*harness.current_l1.borrow(), None);
}

/// A valid status forwards the safe head and then the hash-pinned finalized block, and publishes
/// the delegate's `current_l1`. A failed fetch, or any L1 reference that is not canonical, skips
/// the poll without forwarding or publishing anything and without stopping the actor.
#[rstest]
#[case::valid(DelegateReply::Valid, true)]
#[case::fetch_failed(DelegateReply::FetchFailed, false)]
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

    assert_eq!(Instant::now() - first_poll, DERIVATION_DELEGATE_POLL_INTERVAL * (POLLS - 1));
}

/// Before the engine establishes finality, any finalized block from the delegate may initialize
/// it, including one at genesis height.
#[tokio::test(start_paused = true)]
async fn first_poll_accepts_finality_before_the_engine_establishes_it() {
    let status = with_finalized_l2(l2_block(0, FINALIZED_ORIGIN));
    let mut harness = Harness::new([Ok(status.clone())]);
    harness.complete_engine_sync().await;

    harness.actor.step().await.unwrap();

    assert_eq!(harness.forwarded(), forwarded_for(&status));
}

/// A status whose finalized block is above its safe block is inconsistent and rejected whole.
#[rstest]
#[case::at_safe_head(l2_block(20, FINALIZED_ORIGIN), true)]
#[case::ahead_of_safe_head(l2_block(21, FINALIZED_ORIGIN), false)]
#[tokio::test(start_paused = true)]
async fn poll_rejects_status_whose_finalized_is_ahead_of_its_safe(
    #[case] finalized_l2: L2BlockInfo,
    #[case] accepted: bool,
) {
    let status = with_finalized_l2(finalized_l2);
    let mut harness = Harness::new([Ok(status.clone())]);
    harness.complete_engine_sync().await;

    harness.actor.step().await.unwrap();

    assert_eq!(harness.forwarded(), forwarded_if(accepted, &status));
}

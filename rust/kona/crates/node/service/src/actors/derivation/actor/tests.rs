use super::*;
use crate::{
    DerivationActorRequest, DerivationState, NodeActor,
    actors::derivation::engine_client::MockDerivationEngineClient,
};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use async_trait::async_trait;
use kona_derive::{
    ActivationSignal, Pipeline, PipelineError, PipelineErrorKind, PipelineResult, ResetError,
    Signal, SignalReceiver, StepResult,
};
use kona_engine::FinalizeBlockId;
use kona_genesis::{HardForkConfig, RollupConfig, SystemConfig};
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use kona_safedb::{SafeDatabase, SafeDb, SafeDbError, SafeHeadRecord};
use op_alloy_rpc_types_engine::OpPayloadAttributes;
use rstest::rstest;
use std::{collections::VecDeque, sync::Arc};
use tempfile::TempDir;
use tokio::sync::{mpsc, watch};

/// Reset must adopt the engine's head and resume without waiting for another queue message.
/// A later L1 update must remain valid after the reset exhausts available data.
#[rstest]
#[case::unchanged(RESUME_HEAD)]
#[case::rewound(DERIVED_HEAD)]
#[tokio::test]
async fn reset_adopts_safe_head_and_resumes_atomically(#[case] previous_head: u64) {
    let (request_tx, request_rx) = mpsc::channel(2);
    let mut actor = DerivationActor::new(
        MockDerivationEngineClient::new(),
        request_rx,
        StubPipeline {
            rollup_config: Arc::new(RollupConfig::default()),
            origin: BlockInfo::default(),
            step_result: StepResult::StepFailed(PipelineError::Eof.temp()),
            signal_error: None,
        },
        None,
        l1_block(0).id(),
        watch::channel(None).0,
    );
    request_tx
        .send(DerivationActorRequest::ProcessEngineSyncCompletionRequest(Box::new(l2_block(
            previous_head,
        ))))
        .await
        .unwrap();
    actor.step().await.unwrap();

    request_tx
        .send(DerivationActorRequest::ProcessEngineSignalRequest(Box::new(Signal::Reset(
            kona_derive::ResetSignal { l2_safe_head: l2_block(RESUME_HEAD) },
        ))))
        .await
        .unwrap();
    request_tx
        .send(DerivationActorRequest::ProcessL1HeadUpdateRequest(Box::new(l1_block(
            DERIVED_FROM_L1,
        ))))
        .await
        .unwrap();
    actor.step().await.unwrap();
    assert_eq!(actor.derivation_state_machine.last_confirmed_safe_head(), l2_block(RESUME_HEAD));
    assert_eq!(actor.derivation_state_machine.current_state(), DerivationState::AwaitingL1Data);
    actor.step().await.unwrap();
    assert_eq!(actor.derivation_state_machine.current_state(), DerivationState::AwaitingL1Data);
}

/// An L1 update following reset cannot make a delayed reset acknowledgement consume the newly
/// derived block's outstanding confirmation or its safe-head database record.
#[tokio::test]
async fn reset_keeps_new_attributes_pending_across_an_l1_update() {
    let mut node = RecordingActor::new(engine_client_accepting_attributes());
    node.actor.pipeline.attributes.clear();
    node.complete_el_sync_at(DERIVED_HEAD).await;
    node.signal(Signal::Reset(kona_derive::ResetSignal { l2_safe_head: l2_block(RESUME_HEAD) }))
        .await;
    assert_eq!(
        node.actor.derivation_state_machine.last_confirmed_safe_head(),
        l2_block(RESUME_HEAD)
    );
    assert_eq!(
        node.actor.derivation_state_machine.current_state(),
        DerivationState::AwaitingSafeHeadConfirmation
    );
    let pending = node.actor.attributes_awaiting_confirmation.clone().unwrap();
    node.handle(DerivationActorRequest::ProcessL1HeadUpdateRequest(Box::new(l1_block(
        DERIVED_FROM_L1 + 1,
    ))))
    .await;
    assert_eq!(node.actor.attributes_awaiting_confirmation, Some(pending));
    node.confirm_safe_head(DERIVED_HEAD).await;
    assert_eq!(node.db.last_entry().unwrap(), record(DERIVED_FROM_L1, DERIVED_HEAD));
    assert_eq!(
        node.actor.derivation_state_machine.current_state(),
        DerivationState::AwaitingL1Data
    );
}

/// The error a reset sees when the L2 provider fails during the pipeline's initial reset.
fn l2_provider_error() -> PipelineError {
    PipelineError::Provider("l2 rpc timed out".into())
}

/// Fails the next pipeline signal with the injected error, if any, consuming it.
fn take_injected_signal_error(signal_error: &mut Option<PipelineErrorKind>) -> PipelineResult<()> {
    signal_error.take().map_or(Ok(()), Err)
}

/// A pipeline stub that returns `step_result` once, then yields. Origin advances increment
/// its L1 block number, and a reset moves it to the safe head's L1 origin.
#[derive(Debug)]
struct StubPipeline {
    rollup_config: Arc<RollupConfig>,
    origin: BlockInfo,
    step_result: StepResult,
    signal_error: Option<PipelineErrorKind>,
}

impl Iterator for StubPipeline {
    type Item = OpAttributesWithParent;

    fn next(&mut self) -> Option<Self::Item> {
        None
    }
}

impl kona_derive::OriginProvider for StubPipeline {
    fn origin(&self) -> Option<BlockInfo> {
        Some(self.origin)
    }
}

#[async_trait]
impl SignalReceiver for StubPipeline {
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
        take_injected_signal_error(&mut self.signal_error)?;
        if let Signal::Reset(reset) = signal {
            self.origin = BlockInfo {
                number: reset.l2_safe_head.l1_origin.number,
                hash: reset.l2_safe_head.l1_origin.hash,
                ..Default::default()
            };
        }
        Ok(())
    }
}

#[async_trait]
impl Pipeline for StubPipeline {
    fn peek(&self) -> Option<&OpAttributesWithParent> {
        None
    }

    async fn step(&mut self, _: L2BlockInfo) -> StepResult {
        let result = std::mem::replace(
            &mut self.step_result,
            StepResult::StepFailed(PipelineError::Eof.temp()),
        );
        if matches!(result, StepResult::AdvancedOrigin | StepResult::OriginAdvanceErr(_)) {
            self.origin.number += 1;
        }
        result
    }

    fn rollup_config(&self) -> &RollupConfig {
        &self.rollup_config
    }

    async fn system_config_by_l2_hash(
        &mut self,
        _: B256,
    ) -> Result<SystemConfig, PipelineErrorKind> {
        Ok(SystemConfig::default())
    }
}

/// A pipeline-driven reset must always reach the engine actor. The engine's reset is what
/// sends the pipeline its [`Signal`] back; without it the actor parks in
/// [`DerivationState::AwaitingSignal`] forever.
#[rstest]
#[case::interop_inactive(None)]
#[case::interop_active(Some(0))]
#[tokio::test]
async fn test_pipeline_reset_always_resets_engine(#[case] lagoon_time: Option<u64>) {
    let rollup_config = Arc::new(RollupConfig {
        hardforks: HardForkConfig { lagoon_time, ..Default::default() },
        ..Default::default()
    });

    let mut engine_client = MockDerivationEngineClient::new();
    engine_client.expect_reset_engine_forkchoice().times(1).returning(|| Ok(()));

    let (derivation_origin_tx, derivation_origin_rx) = watch::channel(None);
    let (request_tx, request_rx) = mpsc::channel(1);
    let mut actor = DerivationActor::new(
        engine_client,
        request_rx,
        StubPipeline {
            rollup_config: rollup_config.clone(),
            origin: BlockInfo::default(),
            signal_error: None,
            step_result: StepResult::StepFailed(
                ResetError::ReorgDetected(B256::ZERO, B256::repeat_byte(1)).reset(),
            ),
        },
        None,
        l1_block(0).id(),
        derivation_origin_tx,
    );

    request_tx
        .send(DerivationActorRequest::ProcessEngineSignalRequest(Box::new(Signal::Reset(
            kona_derive::ResetSignal { l2_safe_head: L2BlockInfo::default() },
        ))))
        .await
        .unwrap();
    actor.step().await.unwrap();
    assert!(derivation_origin_rx.borrow().is_some());

    // Complete EL sync so the actor starts deriving, then let it hit the reorg.
    request_tx
        .send(DerivationActorRequest::ProcessEngineSyncCompletionRequest(Box::default()))
        .await
        .unwrap();
    actor.step().await.unwrap();

    assert_eq!(actor.derivation_state_machine.current_state(), DerivationState::AwaitingSignal);
    assert!(derivation_origin_rx.borrow().is_none());
}

/// `optimism_syncStatus` serves the published origin as `current_l1`, so it must follow the
/// pipeline forward, including across a Holocene activation, and back on a reset.
#[rstest]
#[case::advanced_origin(StepResult::AdvancedOrigin)]
#[case::holocene_activation(StepResult::OriginAdvanceErr(ResetError::HoloceneActivation.reset()))]
#[tokio::test]
async fn test_origin_is_published(#[case] step_result: StepResult) {
    let (derivation_origin_tx, derivation_origin_rx) = watch::channel(None);
    let (request_tx, request_rx) = mpsc::channel(1);
    let mut actor = DerivationActor::new(
        MockDerivationEngineClient::new(),
        request_rx,
        StubPipeline {
            rollup_config: Arc::new(RollupConfig::default()),
            origin: BlockInfo::default(),
            step_result,
            signal_error: None,
        },
        None,
        l1_block(0).id(),
        derivation_origin_tx,
    );

    let mut safe_head = L2BlockInfo::default();
    safe_head.l1_origin.number = 10;
    safe_head.l1_origin.hash = B256::repeat_byte(10);
    let reset = || {
        DerivationActorRequest::ProcessEngineSignalRequest(Box::new(Signal::Reset(
            kona_derive::ResetSignal { l2_safe_head: safe_head },
        )))
    };

    request_tx.send(reset()).await.unwrap();
    actor.step().await.unwrap();
    assert_eq!(derivation_origin_rx.borrow().map(|origin| origin.id()), Some(safe_head.l1_origin));

    request_tx
        .send(DerivationActorRequest::ProcessEngineSyncCompletionRequest(Box::new(safe_head)))
        .await
        .unwrap();
    actor.step().await.unwrap();
    assert_eq!(derivation_origin_rx.borrow().unwrap().number, 11);

    request_tx.send(reset()).await.unwrap();
    actor.step().await.unwrap();
    assert_eq!(derivation_origin_rx.borrow().map(|origin| origin.id()), Some(safe_head.l1_origin));
}

#[rstest]
#[case::activation(Signal::Activation(ActivationSignal::default()))]
#[case::flush_channel(Signal::FlushChannel)]
#[tokio::test]
async fn test_non_reset_signal_does_not_publish_uninitialized_origin(#[case] signal: Signal) {
    let (derivation_origin_tx, derivation_origin_rx) = watch::channel(None);
    let (_request_tx, request_rx) = mpsc::channel(1);
    let mut actor = DerivationActor::new(
        MockDerivationEngineClient::new(),
        request_rx,
        StubPipeline {
            rollup_config: Arc::new(RollupConfig::default()),
            origin: BlockInfo::default(),
            step_result: StepResult::AdvancedOrigin,
            signal_error: None,
        },
        None,
        l1_block(0).id(),
        derivation_origin_tx,
    );

    actor.signal(signal).await.unwrap();
    assert!(derivation_origin_rx.borrow().is_none());
    assert!(matches!(actor.produce_next_attributes().await, Err(DerivationError::Yield)));
    assert_eq!(actor.pipeline.origin.number, 1);
    assert!(derivation_origin_rx.borrow().is_none());
}

#[tokio::test]
async fn failed_reset_stops_actor_with_origin_unpublished() {
    let (derivation_origin_tx, derivation_origin_rx) = watch::channel(None);
    let (request_tx, request_rx) = mpsc::channel(1);
    let mut actor = DerivationActor::new(
        MockDerivationEngineClient::new(),
        request_rx,
        StubPipeline {
            rollup_config: Arc::new(RollupConfig::default()),
            origin: BlockInfo::default(),
            signal_error: None,
            step_result: StepResult::AdvancedOrigin,
        },
        None,
        l1_block(0).id(),
        derivation_origin_tx,
    );

    let mut safe_head = L2BlockInfo::default();
    safe_head.l1_origin.number = 10;
    safe_head.l1_origin.hash = B256::repeat_byte(10);
    let reset = || {
        DerivationActorRequest::ProcessEngineSignalRequest(Box::new(Signal::Reset(
            kona_derive::ResetSignal { l2_safe_head: safe_head },
        )))
    };
    request_tx.send(reset()).await.unwrap();
    actor.step().await.unwrap();
    assert_eq!(derivation_origin_rx.borrow().map(|origin| origin.id()), Some(safe_head.l1_origin));

    // A reset can fail before it replaces the old origin, for example on an L2 RPC error.
    actor.pipeline.signal_error = Some(l2_provider_error().temp());
    request_tx.send(reset()).await.unwrap();
    assert!(matches!(
        actor.step().await,
        Err(DerivationError::Pipeline(PipelineErrorKind::Temporary(PipelineError::Provider(_))))
    ));
    assert_eq!(actor.pipeline.origin.id(), safe_head.l1_origin);
    assert!(derivation_origin_rx.borrow().is_none());
}

/// Activation and channel flush failures leave the derivation state intact, so the actor
/// keeps running and a later reset still publishes its origin.
#[rstest]
#[case::activation(Signal::Activation(ActivationSignal::default()))]
#[case::flush_channel(Signal::FlushChannel)]
#[tokio::test]
async fn failed_non_reset_signal_keeps_actor_running(#[case] signal: Signal) {
    let (derivation_origin_tx, derivation_origin_rx) = watch::channel(None);
    let (request_tx, request_rx) = mpsc::channel(1);
    let mut actor = DerivationActor::new(
        MockDerivationEngineClient::new(),
        request_rx,
        StubPipeline {
            rollup_config: Arc::new(RollupConfig::default()),
            origin: BlockInfo::default(),
            signal_error: Some(l2_provider_error().temp()),
            step_result: StepResult::AdvancedOrigin,
        },
        None,
        l1_block(0).id(),
        derivation_origin_tx,
    );

    request_tx
        .send(DerivationActorRequest::ProcessEngineSignalRequest(Box::new(signal)))
        .await
        .unwrap();
    actor.step().await.unwrap();
    assert!(derivation_origin_rx.borrow().is_none());

    let mut safe_head = L2BlockInfo::default();
    safe_head.l1_origin.number = 10;
    safe_head.l1_origin.hash = B256::repeat_byte(10);
    request_tx
        .send(DerivationActorRequest::ProcessEngineSignalRequest(Box::new(Signal::Reset(
            kona_derive::ResetSignal { l2_safe_head: safe_head },
        ))))
        .await
        .unwrap();
    actor.step().await.unwrap();
    assert_eq!(derivation_origin_rx.borrow().map(|origin| origin.id()), Some(safe_head.l1_origin));
}

/// A pipeline stub that yields queued attributes one at a time, then reports its data
/// source exhausted. A reset re-queues the derived block, as re-derivation would.
#[derive(Debug)]
struct QueuedPipeline {
    attributes: VecDeque<OpAttributesWithParent>,
    derived_from: BlockInfo,
    rollup_config: Arc<RollupConfig>,
    signal_error: Option<PipelineErrorKind>,
}

impl Iterator for QueuedPipeline {
    type Item = OpAttributesWithParent;

    fn next(&mut self) -> Option<Self::Item> {
        self.attributes.pop_front()
    }
}

impl kona_derive::OriginProvider for QueuedPipeline {
    fn origin(&self) -> Option<BlockInfo> {
        Some(self.derived_from)
    }
}

#[async_trait]
impl SignalReceiver for QueuedPipeline {
    async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
        take_injected_signal_error(&mut self.signal_error)?;
        if matches!(signal, Signal::Reset(_)) {
            self.attributes.push_back(derived_attributes());
        }
        Ok(())
    }
}

#[async_trait]
impl Pipeline for QueuedPipeline {
    fn peek(&self) -> Option<&OpAttributesWithParent> {
        self.attributes.front()
    }

    async fn step(&mut self, _: L2BlockInfo) -> StepResult {
        if self.attributes.is_empty() {
            StepResult::StepFailed(PipelineError::Eof.temp())
        } else {
            StepResult::PreparedAttributes
        }
    }

    fn rollup_config(&self) -> &RollupConfig {
        &self.rollup_config
    }

    async fn system_config_by_l2_hash(
        &mut self,
        _: B256,
    ) -> Result<SystemConfig, PipelineErrorKind> {
        Ok(SystemConfig::default())
    }
}

const DERIVED_FROM_L1: u64 = 100;
const EARLIER_L1: u64 = 90;
const EARLIEST_L1: u64 = 80;
/// Below every L1 number above: a reset seeks from the safe head's L1 origin, so this keeps
/// every recorded entry in view of the truncation.
const SAFE_HEAD_L1_ORIGIN: u64 = 42;
const RESUME_HEAD: u64 = 20;
const DERIVED_HEAD: u64 = RESUME_HEAD + 1;

fn l1_block(number: u64) -> BlockInfo {
    BlockInfo { hash: B256::with_last_byte(number as u8), number, ..Default::default() }
}

fn l2_block(number: u64) -> L2BlockInfo {
    L2BlockInfo {
        block_info: BlockInfo {
            hash: B256::with_last_byte(number as u8),
            number,
            parent_hash: B256::with_last_byte(number.saturating_sub(1) as u8),
            timestamp: 0,
        },
        l1_origin: BlockNumHash { hash: B256::ZERO, number: SAFE_HEAD_L1_ORIGIN },
        seq_num: 0,
    }
}

fn record(l1: u64, safe_head: u64) -> SafeHeadRecord {
    SafeHeadRecord { l1: l1_block(l1).id(), safe_head: l2_block(safe_head).block_info.id() }
}

/// Attributes for the child of [`RESUME_HEAD`], derived from [`DERIVED_FROM_L1`].
fn derived_attributes() -> OpAttributesWithParent {
    OpAttributesWithParent {
        attributes: OpPayloadAttributes::default(),
        parent: l2_block(RESUME_HEAD),
        derived_from: Some(l1_block(DERIVED_FROM_L1)),
        is_last_in_span: true,
    }
}

fn engine_client_accepting_attributes() -> MockDerivationEngineClient {
    let mut engine_client = MockDerivationEngineClient::new();
    engine_client.expect_send_safe_l2_signal().returning(|_| Ok(()));
    engine_client
}

struct RecordingActor {
    actor: DerivationActor<MockDerivationEngineClient, QueuedPipeline>,
    db: Arc<SafeDatabase>,
    request_tx: mpsc::Sender<DerivationActorRequest>,
    _dir: TempDir,
}

impl RecordingActor {
    /// An actor whose pipeline derives the child of [`RESUME_HEAD`], writing to a fresh
    /// on-disk database.
    fn new(engine_client: MockDerivationEngineClient) -> Self {
        let dir = TempDir::new().unwrap();
        let db = Arc::new(SafeDatabase::new(dir.path()).unwrap());
        let pipeline = QueuedPipeline {
            attributes: VecDeque::from([derived_attributes()]),
            derived_from: l1_block(DERIVED_FROM_L1),
            rollup_config: Arc::new(RollupConfig::default()),
            signal_error: None,
        };
        let (request_tx, request_rx) = mpsc::channel(1);
        let actor = DerivationActor::new(
            engine_client,
            request_rx,
            pipeline,
            Some(db.clone()),
            l1_block(0).id(),
            watch::channel(None).0,
        );
        Self { actor, db, request_tx, _dir: dir }
    }

    async fn handle(&mut self, request: DerivationActorRequest) {
        self.request_tx.send(request).await.unwrap();
        self.actor.step().await.unwrap();
    }

    async fn complete_el_sync_at(&mut self, head: u64) {
        self.handle(DerivationActorRequest::ProcessEngineSyncCompletionRequest(Box::new(
            l2_block(head),
        )))
        .await;
    }

    async fn confirm_safe_head(&mut self, head: u64) {
        self.handle(DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(Box::new(
            l2_block(head),
        )))
        .await;
    }

    async fn signal(&mut self, signal: Signal) {
        self.handle(DerivationActorRequest::ProcessEngineSignalRequest(Box::new(signal))).await;
    }

    async fn finalize_l1(&mut self, number: u64) {
        self.handle(DerivationActorRequest::ProcessFinalizedL1Block(Box::new(l1_block(number))))
            .await;
    }

    /// Completes EL sync at [`RESUME_HEAD`], derives its child, and has the engine confirm it.
    async fn derive_and_confirm_one_block(&mut self) {
        self.complete_el_sync_at(RESUME_HEAD).await;
        self.confirm_safe_head(DERIVED_HEAD).await;
    }

    /// Resets to [`RESUME_HEAD`], adopting the head and resuming in the same request.
    async fn reset_to_resume_head(&mut self) {
        self.signal(Signal::Reset(kona_derive::ResetSignal {
            l2_safe_head: l2_block(RESUME_HEAD),
        }))
        .await;
    }
}

#[tokio::test]
async fn a_confirmed_block_is_recorded_against_its_derivation_source() {
    let mut node = RecordingActor::new(engine_client_accepting_attributes());

    node.derive_and_confirm_one_block().await;

    assert_eq!(node.db.last_entry().unwrap(), record(DERIVED_FROM_L1, DERIVED_HEAD));
}

/// With no history below the reset point, the L1 block that made the reset head safe is
/// unknown, so nothing is re-recorded and the reset head itself must not be written.
#[tokio::test]
async fn a_reset_with_no_earlier_history_leaves_the_database_empty() {
    let mut node = RecordingActor::new(engine_client_accepting_attributes());
    node.derive_and_confirm_one_block().await;

    node.reset_to_resume_head().await;

    assert!(matches!(node.db.last_entry(), Err(SafeDbError::NotFound)));
}

/// With history below the reset point, everything from the entry where the reset head first
/// became safe is replaced by the reset head at that L1 block, and the history below is
/// untouched.
#[tokio::test]
async fn a_reset_truncates_to_the_resume_head_and_keeps_earlier_history() {
    let mut node = RecordingActor::new(engine_client_accepting_attributes());
    node.db.safe_head_updated(l2_block(RESUME_HEAD - 10), l1_block(EARLIEST_L1).id()).unwrap();
    node.db.safe_head_updated(l2_block(RESUME_HEAD), l1_block(EARLIER_L1).id()).unwrap();
    node.derive_and_confirm_one_block().await;

    node.reset_to_resume_head().await;

    assert_eq!(node.db.first_entry().unwrap(), record(EARLIEST_L1, RESUME_HEAD - 10));
    assert_eq!(node.db.last_entry().unwrap(), record(EARLIER_L1, RESUME_HEAD));
    assert_eq!(node.db.l1_at_safe_head(RESUME_HEAD).unwrap(), record(EARLIER_L1, RESUME_HEAD));
    assert!(
        matches!(node.db.l1_at_safe_head(DERIVED_HEAD), Err(SafeDbError::L1AtSafeHeadNotFound)),
        "the truncated block must read as not yet safe, not as safe at a later L1 block"
    );
}

#[tokio::test]
async fn a_block_derived_again_after_a_reset_is_recorded_again() {
    let mut node = RecordingActor::new(engine_client_accepting_attributes());
    node.derive_and_confirm_one_block().await;
    node.reset_to_resume_head().await;

    node.confirm_safe_head(DERIVED_HEAD).await;

    assert_eq!(node.db.last_entry().unwrap(), record(DERIVED_FROM_L1, DERIVED_HEAD));
}

#[tokio::test]
async fn a_signal_other_than_a_reset_leaves_the_history_alone() {
    let mut node = RecordingActor::new(engine_client_accepting_attributes());
    node.derive_and_confirm_one_block().await;

    node.signal(Signal::FlushChannel).await;

    assert_eq!(node.db.last_entry().unwrap(), record(DERIVED_FROM_L1, DERIVED_HEAD));
}

/// Derivation never revisits the blocks between an older tip and the resume point, so the
/// history is cleared rather than left with a gap that queries cannot detect.
#[tokio::test]
async fn resuming_above_the_recorded_tip_clears_the_history() {
    let mut node = RecordingActor::new(engine_client_accepting_attributes());
    node.db.safe_head_updated(l2_block(RESUME_HEAD - 10), l1_block(EARLIER_L1).id()).unwrap();

    node.complete_el_sync_at(RESUME_HEAD).await;

    assert!(matches!(node.db.first_entry(), Err(SafeDbError::NotFound)));
}

/// Finalization waits for the record, so the engine's finalized head cannot pass the
/// recorded history and leave a block behind that a restart would never record.
#[tokio::test]
async fn a_block_is_queued_for_finalization_only_once_it_is_recorded() {
    let mut engine_client = engine_client_accepting_attributes();
    engine_client
        .expect_send_finalized_l2_block()
        .withf(|block| matches!(block, FinalizeBlockId::ByNumber(DERIVED_HEAD)))
        .times(1)
        .returning(|_| Ok(()));
    let mut node = RecordingActor::new(engine_client);
    node.complete_el_sync_at(RESUME_HEAD).await;

    node.finalize_l1(DERIVED_FROM_L1).await;
    node.confirm_safe_head(DERIVED_HEAD).await;
    node.finalize_l1(DERIVED_FROM_L1).await;
}

mod database;

mod safedb_tests;

mod service_shutdown;

mod signal_failures;

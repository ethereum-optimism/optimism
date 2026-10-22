//! [`NodeActor`] implementation for the derivation sub-routine.

use crate::{
    DerivationActorRequest, DerivationEngineClient, DerivationState, DerivationStateMachine,
    DerivationStateTransitionError, DerivationStateUpdate, Metrics, NodeActor,
    actors::derivation::{L2Finalizer, SafeHeadRecorder, TruncationReason},
};
use async_trait::async_trait;
use kona_derive::{
    ActivationSignal, Pipeline, PipelineError, PipelineErrorKind, ResetError, Signal,
    SignalReceiver, StepResult,
};
use kona_engine::FinalizeBlockId;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use kona_safedb::{SafeDb, SafeDbError};
use std::sync::Arc;
use thiserror::Error;
use tokio::sync::mpsc;

/// The [`NodeActor`] for the derivation sub-routine.
///
/// This actor is responsible for receiving messages from [`NodeActor`]s and stepping the
/// derivation pipeline forward to produce new payload attributes. The actor then sends the payload
/// to the [`NodeActor`] responsible for the execution sub-routine.
#[derive(Debug)]
pub struct DerivationActor<DerivationEngineClient_, PipelineSignalReceiver>
where
    DerivationEngineClient_: DerivationEngineClient,
    PipelineSignalReceiver: Pipeline + SignalReceiver,
{
    /// The channel on which all inbound requests are received by the [`DerivationActor`].
    inbound_request_rx: mpsc::Receiver<DerivationActorRequest>,
    /// The Engine client used to interact with the engine.
    engine_client: DerivationEngineClient_,

    /// The derivation pipeline.
    pipeline: PipelineSignalReceiver,
    /// The state machine controlling when derivation can occur.
    derivation_state_machine: DerivationStateMachine,
    /// The [`L2Finalizer`] tracks derived L2 blocks awaiting finalization.
    pub(crate) finalizer: L2Finalizer,
    /// Records the safe head reached as of each L1 block.
    safe_head_recorder: SafeHeadRecorder,
    /// The attributes sent to the engine and not yet confirmed as a safe head.
    attributes_awaiting_confirmation: Option<OpAttributesWithParent>,
}

impl<DerivationEngineClient_, PipelineSignalReceiver>
    DerivationActor<DerivationEngineClient_, PipelineSignalReceiver>
where
    DerivationEngineClient_: DerivationEngineClient,
    PipelineSignalReceiver: Pipeline + SignalReceiver,
{
    /// Creates a new instance of the [`DerivationActor`].
    pub fn new(
        engine_client: DerivationEngineClient_,
        inbound_request_rx: mpsc::Receiver<DerivationActorRequest>,
        pipeline: PipelineSignalReceiver,
        safe_db: Arc<dyn SafeDb>,
    ) -> Self {
        Self {
            pipeline,
            inbound_request_rx,
            engine_client,
            derivation_state_machine: DerivationStateMachine::default(),
            finalizer: L2Finalizer::default(),
            safe_head_recorder: SafeHeadRecorder::new(safe_db),
            attributes_awaiting_confirmation: None,
        }
    }

    /// Handles a [`Signal`] received over the derivation signal receiver channel.
    ///
    /// A reset signal carries the head derivation restarts from, so the recorded safe head
    /// history is truncated to that same block before the pipeline is reset.
    async fn signal(&mut self, signal: Signal) -> Result<(), DerivationError> {
        if let Signal::Reset(reset) = &signal {
            self.finalizer.clear();
            self.attributes_awaiting_confirmation = None;
            self.safe_head_recorder
                .truncate(reset.l2_safe_head, TruncationReason::EngineReset)
                .await?;
        }

        match self.pipeline.signal(signal).await {
            Ok(_) => info!(target: "derivation", ?signal, "[SIGNAL] Executed Successfully"),
            Err(e) => {
                error!(target: "derivation", ?e, ?signal, "Failed to signal derivation pipeline")
            }
        }
        Ok(())
    }

    /// Attempts to step the derivation pipeline forward as much as possible in order to produce the
    /// next safe payload.
    async fn produce_next_attributes(&mut self) -> Result<OpAttributesWithParent, DerivationError> {
        // As we start the safe head at the disputed block's parent, we step the pipeline until the
        // first attributes are produced. All batches at and before the safe head will be
        // dropped, so the first payload will always be the disputed one.
        loop {
            match self.pipeline.step(self.derivation_state_machine.last_confirmed_safe_head()).await
            {
                StepResult::PreparedAttributes => { /* continue; attributes will be sent off. */ }
                StepResult::AdvancedOrigin => {
                    let origin =
                        self.pipeline.origin().ok_or(PipelineError::MissingOrigin.crit())?.number;

                    kona_macros::set!(counter, Metrics::DERIVATION_L1_ORIGIN, origin);
                    debug!(target: "derivation", l1_block = origin, "Advanced L1 origin");
                }
                StepResult::OriginAdvanceErr(e) | StepResult::StepFailed(e) => {
                    match e {
                        PipelineErrorKind::Temporary(e) => {
                            // NotEnoughData is transient, and doesn't imply we need to wait for
                            // more data. We can continue stepping until we receive an Eof.
                            if matches!(e, PipelineError::NotEnoughData) {
                                continue;
                            }

                            debug!(
                                target: "derivation",
                                "Exhausted data source for now; Yielding until the chain has extended."
                            );
                            return Err(DerivationError::Yield);
                        }
                        PipelineErrorKind::Reset(e) => {
                            warn!(target: "derivation", "Derivation pipeline is being reset: {e}");

                            if matches!(e, ResetError::HoloceneActivation) {
                                self.pipeline
                                    .signal(Signal::Activation(ActivationSignal {
                                        l2_safe_head: self
                                            .derivation_state_machine
                                            .last_confirmed_safe_head(),
                                    }))
                                    .await?;
                            } else {
                                if let ResetError::ReorgDetected(expected, new) = e {
                                    warn!(
                                        target: "derivation",
                                        "L1 reorg detected! Expected: {expected} | New: {new}"
                                    );

                                    kona_macros::inc!(counter, Metrics::L1_REORG_COUNT);
                                }
                                self.engine_client.reset_engine_forkchoice().await.map_err(|e| {
                                    error!(target: "derivation", ?e, "Failed to send reset request");
                                    DerivationError::Sender(Box::new(e))
                                })?;
                                self.derivation_state_machine
                                    .update(&DerivationStateUpdate::SignalNeeded)?;
                                return Err(DerivationError::Yield);
                            }
                        }
                        PipelineErrorKind::Critical(_) => {
                            error!(target: "derivation", "Critical derivation error: {e}");
                            kona_macros::inc!(counter, Metrics::DERIVATION_CRITICAL_ERROR);
                            return Err(e.into());
                        }
                    }
                }
            }

            // If there are any new attributes, send them to the execution actor.
            if let Some(attrs) = self.pipeline.next() {
                return Ok(attrs);
            }
        }
    }

    async fn handle_derivation_actor_request(
        &mut self,
        request_type: DerivationActorRequest,
    ) -> Result<(), DerivationError> {
        match request_type {
            DerivationActorRequest::ProcessEngineSignalRequest(signal) => {
                self.signal(*signal).await?;
                self.derivation_state_machine.update(&DerivationStateUpdate::SignalProcessed)?;
            }
            DerivationActorRequest::ProcessFinalizedL1Block(finalized_l1_block) => {
                // Attempt to finalize the block. If successful, notify engine.
                if let Some(l2_block_number) = self.finalizer.try_finalize_next(*finalized_l1_block)
                {
                    // Local L1-finality: the engine's own canonical chain is the authoritative
                    // source at this height, so finalize by number.
                    self.engine_client
                        .send_finalized_l2_block(FinalizeBlockId::ByNumber(l2_block_number))
                        .await
                        .map_err(|e| DerivationError::Sender(Box::new(e)))?;
                }
            }
            DerivationActorRequest::ProcessL1HeadUpdateRequest(l1_head) => {
                info!(target: "derivation", l1_head = ?*l1_head, "Processing l1 head update");

                self.derivation_state_machine.update(&DerivationStateUpdate::L1DataReceived)?;

                self.attempt_derivation().await?;
            }
            DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(safe_head) => {
                info!(target: "derivation", safe_head = ?*safe_head, "Received safe head from engine.");
                let confirmed_safe_head = *safe_head;
                self.derivation_state_machine
                    .update(&DerivationStateUpdate::NewAttributesConfirmed(safe_head))?;
                self.settle_derived_attributes(confirmed_safe_head).await?;

                self.attempt_derivation().await?;
            }
            DerivationActorRequest::ProcessEngineSyncCompletionRequest(safe_head) => {
                info!(target: "derivation", "Engine finished syncing, starting derivation.");
                let resume_head = *safe_head;
                self.derivation_state_machine
                    .update(&DerivationStateUpdate::ELSyncCompleted(safe_head))?;
                self.safe_head_recorder
                    .truncate(resume_head, TruncationReason::ElSyncCompleted)
                    .await?;

                self.attempt_derivation().await?;
            }
        }

        Ok(())
    }

    /// Records and queues for finalization the attributes that `safe_head` confirms.
    ///
    /// A head that confirms nothing pending was adopted by the engine rather than derived here,
    /// so there is nothing to record. Finalization is queued only once the record is written, so
    /// the engine's finalized head can never run ahead of the recorded history.
    async fn settle_derived_attributes(
        &mut self,
        safe_head: L2BlockInfo,
    ) -> Result<(), DerivationError> {
        let Some(attributes) = self.attributes_awaiting_confirmation.take() else {
            debug!(
                target: "derivation",
                safe_head = safe_head.block_info.number,
                "Safe head confirms no derived attributes"
            );
            return Ok(());
        };
        if !attributes_produce(&attributes, &safe_head) {
            warn!(
                target: "derivation",
                expected = attributes.block_number(),
                safe_head = safe_head.block_info.number,
                "Safe head does not match the derived attributes awaiting confirmation"
            );
            return Ok(());
        }
        if let Some(derived_from) = attributes.derived_from {
            self.safe_head_recorder.record(safe_head, derived_from.id()).await?;
        }
        self.finalizer.enqueue_for_finalization(&attributes);
        Ok(())
    }

    /// Attempts to process the next payload attributes.
    async fn attempt_derivation(&mut self) -> Result<(), DerivationError> {
        if self.derivation_state_machine.current_state() != DerivationState::Deriving {
            info!(target: "derivation", derivation_state=?self.derivation_state_machine, "Skipping derivation.");
            return Ok(());
        }

        info!(target: "derivation", derivation_state=?self.derivation_state_machine, "Attempting derivation.");

        // Advance the pipeline as much as possible, new data may be available or there still may be
        // payloads in the attributes queue.
        let payload_attributes = match self.produce_next_attributes().await {
            Ok(attrs) => attrs,
            Err(DerivationError::Yield) => {
                info!(target: "derivation", "Yielding derivation until more data is available.");
                self.derivation_state_machine.update(&DerivationStateUpdate::MoreDataNeeded)?;
                return Ok(());
            }
            Err(e) => {
                return Err(e);
            }
        };
        trace!(target: "derivation", ?payload_attributes, "Produced payload attributes.");

        self.derivation_state_machine.update(&DerivationStateUpdate::NewAttributesDerived(
            Box::new(payload_attributes.clone()),
        ))?;
        self.attributes_awaiting_confirmation = Some(payload_attributes.clone());

        // Send payload attributes out for processing.
        self.engine_client
            .send_safe_l2_signal(payload_attributes.into())
            .await
            .map_err(|e| DerivationError::Sender(Box::new(e)))?;

        Ok(())
    }
}

/// Whether `safe_head` is the block that `attributes` build on their parent.
fn attributes_produce(attributes: &OpAttributesWithParent, safe_head: &L2BlockInfo) -> bool {
    attributes.block_number() == safe_head.block_info.number &&
        attributes.parent.block_info.hash == safe_head.block_info.parent_hash
}

#[async_trait]
impl<DerivationEngineClient_, PipelineSignalReceiver> NodeActor
    for DerivationActor<DerivationEngineClient_, PipelineSignalReceiver>
where
    DerivationEngineClient_: DerivationEngineClient + 'static,
    PipelineSignalReceiver: Pipeline + SignalReceiver + Send + Sync + 'static,
{
    type Error = DerivationError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        let request = self.inbound_request_rx.recv().await.ok_or_else(|| {
            error!(
                target: "derivation",
                "DerivationActor inbound request receiver closed unexpectedly",
            );
            DerivationError::RequestReceiveFailed
        })?;
        self.handle_derivation_actor_request(request).await
    }
}

/// An error from the [`DerivationActor`].
#[derive(Error, Debug)]
pub enum DerivationError {
    /// An error originating from the derivation pipeline.
    #[error(transparent)]
    Pipeline(#[from] PipelineErrorKind),
    /// Waiting for more data to be available.
    #[error("Waiting for more data to be available")]
    Yield,
    /// An error originating from the broadcast sender.
    #[error("Failed to send event to broadcast sender: {0}")]
    Sender(Box<dyn std::error::Error>),
    /// Failed to receive inbound request
    #[error("Failed to receive inbound request")]
    RequestReceiveFailed,
    /// A safe head database write failed. The actor stops rather than leave a gap that later
    /// queries would serve as an older safe head.
    #[error("safe head db write failed")]
    SafeDb(#[from] SafeDbError),
    /// An invalid state transition occurred.
    #[error(transparent)]
    StateTransitionError(#[from] DerivationStateTransitionError),
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::actors::derivation::engine_client::MockDerivationEngineClient;
    use alloy_eips::BlockNumHash;
    use alloy_primitives::B256;
    use kona_derive::{PipelineError, PipelineResult};
    use kona_genesis::{HardForkConfig, RollupConfig, SystemConfig};
    use kona_protocol::BlockInfo;
    use kona_safedb::{DisabledDatabase, SafeDatabase, SafeHeadRecord};
    use op_alloy_rpc_types_engine::OpPayloadAttributes;
    use rstest::rstest;
    use std::collections::VecDeque;
    use tempfile::TempDir;

    /// A pipeline stub whose every step reports an L1 reorg, forcing the reset branch of
    /// [`DerivationActor::produce_next_attributes`].
    #[derive(Debug)]
    struct ReorgingPipeline {
        rollup_config: Arc<RollupConfig>,
    }

    impl Iterator for ReorgingPipeline {
        type Item = OpAttributesWithParent;

        fn next(&mut self) -> Option<Self::Item> {
            None
        }
    }

    impl kona_derive::OriginProvider for ReorgingPipeline {
        fn origin(&self) -> Option<BlockInfo> {
            Some(BlockInfo::default())
        }
    }

    #[async_trait]
    impl SignalReceiver for ReorgingPipeline {
        async fn signal(&mut self, _: Signal) -> PipelineResult<()> {
            Ok(())
        }
    }

    #[async_trait]
    impl Pipeline for ReorgingPipeline {
        fn peek(&self) -> Option<&OpAttributesWithParent> {
            None
        }

        async fn step(&mut self, _: L2BlockInfo) -> StepResult {
            StepResult::StepFailed(PipelineErrorKind::Reset(ResetError::ReorgDetected(
                B256::ZERO,
                B256::repeat_byte(1),
            )))
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

        let (request_tx, request_rx) = mpsc::channel(1);
        let mut actor = DerivationActor::new(
            engine_client,
            request_rx,
            ReorgingPipeline { rollup_config: rollup_config.clone() },
            Arc::new(DisabledDatabase),
        );

        // Complete EL sync so the actor starts deriving, then let it hit the reorg.
        request_tx
            .send(DerivationActorRequest::ProcessEngineSyncCompletionRequest(Box::default()))
            .await
            .unwrap();
        actor.step().await.unwrap();

        assert_eq!(actor.derivation_state_machine.current_state(), DerivationState::AwaitingSignal);
    }

    /// A pipeline stub that yields queued attributes one at a time, then reports its data
    /// source exhausted. A reset re-queues the derived block, as re-derivation would.
    #[derive(Debug)]
    struct QueuedPipeline {
        attributes: VecDeque<OpAttributesWithParent>,
        derived_from: BlockInfo,
        rollup_config: Arc<RollupConfig>,
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
                parent_hash: B256::with_last_byte(number as u8 - 1),
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
            };
            let (request_tx, request_rx) = mpsc::channel(1);
            let actor = DerivationActor::new(engine_client, request_rx, pipeline, db.clone());
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
            self.handle(DerivationActorRequest::ProcessFinalizedL1Block(Box::new(l1_block(
                number,
            ))))
            .await;
        }

        /// Completes EL sync at [`RESUME_HEAD`], derives its child, and has the engine confirm it.
        async fn derive_and_confirm_one_block(&mut self) {
            self.complete_el_sync_at(RESUME_HEAD).await;
            self.confirm_safe_head(DERIVED_HEAD).await;
        }

        /// Resets to [`RESUME_HEAD`] and lets the engine report it as the new safe head.
        async fn reset_to_resume_head(&mut self) {
            self.signal(Signal::Reset(kona_derive::ResetSignal {
                l2_safe_head: l2_block(RESUME_HEAD),
            }))
            .await;
            self.confirm_safe_head(RESUME_HEAD).await;
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
}

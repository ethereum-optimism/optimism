//! [`NodeActor`] implementation for the derivation sub-routine.

use crate::{
    DerivationActorRequest, DerivationEngineClient, DerivationState, DerivationStateMachine,
    DerivationStateTransitionError, DerivationStateUpdate, DerivationStatus, Metrics, NodeActor,
    derivation::L2Finalizer,
};
use async_trait::async_trait;
use kona_derive::{
    ActivationSignal, Pipeline, PipelineError, PipelineErrorKind, ResetError, Signal,
    SignalReceiver, StepResult,
};
use kona_engine::FinalizeBlockId;
use kona_protocol::{BlockInfo, OpAttributesWithParent};
use thiserror::Error;
use tokio::sync::{mpsc, watch};

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
    /// The latest published L1 head.
    head: watch::Receiver<BlockInfo>,
    /// The latest published finalized L1 head.
    finalized: watch::Receiver<BlockInfo>,
    /// The Engine client used to interact with the engine.
    engine_client: DerivationEngineClient_,

    /// Progress published to RPC readers.
    status: watch::Sender<DerivationStatus>,

    /// The derivation pipeline.
    pipeline: PipelineSignalReceiver,
    /// The state machine controlling when derivation can occur.
    derivation_state_machine: DerivationStateMachine,
    /// The [`L2Finalizer`] tracks derived L2 blocks awaiting finalization.
    pub(crate) finalizer: L2Finalizer,
}

impl<DerivationEngineClient_, PipelineSignalReceiver>
    DerivationActor<DerivationEngineClient_, PipelineSignalReceiver>
where
    DerivationEngineClient_: DerivationEngineClient,
    PipelineSignalReceiver: Pipeline + SignalReceiver,
{
    /// Creates a new instance of the [`DerivationActor`].
    ///
    /// L1 updates are consumed as changes; initial snapshots do not trigger processing.
    pub fn new(
        engine_client: DerivationEngineClient_,
        inbound_request_rx: mpsc::Receiver<DerivationActorRequest>,
        pipeline: PipelineSignalReceiver,
        head: watch::Receiver<BlockInfo>,
        finalized: watch::Receiver<BlockInfo>,
    ) -> Self {
        let (status, _) = watch::channel(DerivationStatus { current_l1: pipeline.origin() });
        Self {
            status,
            pipeline,
            inbound_request_rx,
            head,
            finalized,
            engine_client,
            derivation_state_machine: DerivationStateMachine::default(),
            finalizer: L2Finalizer::default(),
        }
    }

    /// Subscribes to the actor's current derivation progress.
    pub fn state_receiver(&self) -> watch::Receiver<DerivationStatus> {
        self.status.subscribe()
    }

    fn publish_status(&self) {
        let current_l1 = self.pipeline.origin();
        self.status.send_if_modified(|state| {
            std::mem::replace(&mut state.current_l1, current_l1) != current_l1
        });
    }

    /// Handles a [`Signal`] received over the derivation signal receiver channel.
    async fn signal(&mut self, signal: Signal) {
        if matches!(signal, Signal::Reset(_)) {
            // Clear the finalization queue on reset.
            self.finalizer.clear();
        }

        let result = self.pipeline.signal(signal).await;
        self.publish_status();
        match result {
            Ok(_) => info!(target: "derivation", ?signal, "[SIGNAL] Executed Successfully"),
            Err(e) => {
                error!(target: "derivation", ?e, ?signal, "Failed to signal derivation pipeline")
            }
        }
    }

    /// Attempts to step the derivation pipeline forward as much as possible in order to produce the
    /// next safe payload.
    async fn produce_next_attributes(&mut self) -> Result<OpAttributesWithParent, DerivationError> {
        // As we start the safe head at the disputed block's parent, we step the pipeline until the
        // first attributes are produced. All batches at and before the safe head will be
        // dropped, so the first payload will always be the disputed one.
        loop {
            let result =
                self.pipeline.step(self.derivation_state_machine.last_confirmed_safe_head()).await;
            self.publish_status();
            match result {
                StepResult::PreparedAttributes => { /* continue; attributes will be sent off. */ }
                StepResult::AdvancedOrigin => {
                    let origin =
                        self.pipeline.origin().ok_or(PipelineError::MissingOrigin.crit())?.number;

                    metrics::counter!(Metrics::DERIVATION_L1_ORIGIN).absolute(origin);
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
                                self.publish_status();
                            } else {
                                if let ResetError::ReorgDetected(expected, new) = e {
                                    warn!(
                                        target: "derivation",
                                        "L1 reorg detected! Expected: {expected} | New: {new}"
                                    );

                                    metrics::counter!(Metrics::L1_REORG_COUNT).increment(1);
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
                            metrics::counter!(Metrics::DERIVATION_CRITICAL_ERROR).increment(1);
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
                self.signal(*signal).await;
                self.derivation_state_machine.update(&DerivationStateUpdate::SignalProcessed)?;
            }
            DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(safe_head) => {
                info!(target: "derivation", safe_head = ?*safe_head, "Received safe head from engine.");
                self.derivation_state_machine
                    .update(&DerivationStateUpdate::NewAttributesConfirmed(safe_head))?;

                self.attempt_derivation().await?;
            }
            DerivationActorRequest::ProcessEngineSyncCompletionRequest(safe_head) => {
                info!(target: "derivation", "Engine finished syncing, starting derivation.");
                self.derivation_state_machine
                    .update(&DerivationStateUpdate::ELSyncCompleted(safe_head))?;

                self.attempt_derivation().await?;
            }
        }

        Ok(())
    }

    async fn process_finalized_l1_block(
        &mut self,
        block: BlockInfo,
    ) -> Result<(), DerivationError> {
        if let Some(l2_block_number) = self.finalizer.try_finalize_next(block) {
            // Finalize by number against the engine's canonical chain.
            self.engine_client
                .send_finalized_l2_block(FinalizeBlockId::ByNumber(l2_block_number))
                .await
                .map_err(|error| DerivationError::Sender(Box::new(error)))?;
        }
        Ok(())
    }

    async fn process_l1_head(&mut self, head: BlockInfo) -> Result<(), DerivationError> {
        info!(target: "derivation", l1_head = ?head, "Processing l1 head update");
        self.derivation_state_machine.update(&DerivationStateUpdate::L1DataReceived)?;
        self.attempt_derivation().await
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

        // Enqueue the payload attributes for finalization tracking.
        self.finalizer.enqueue_for_finalization(&payload_attributes);

        // Send payload attributes out for processing.
        self.engine_client
            .send_safe_l2_signal(payload_attributes.into())
            .await
            .map_err(|e| DerivationError::Sender(Box::new(e)))?;

        Ok(())
    }
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
        tokio::select! {
            request = self.inbound_request_rx.recv() => {
                let request = request.ok_or(DerivationError::RequestReceiveFailed)?;
                self.handle_derivation_actor_request(request).await
            }
            result = self.head.changed() => {
                result.map_err(|_| DerivationError::L1ReceiveFailed)?;
                let head = *self.head.borrow_and_update();
                self.process_l1_head(head).await
            }
            result = self.finalized.changed() => {
                result.map_err(|_| DerivationError::L1ReceiveFailed)?;
                let finalized = *self.finalized.borrow_and_update();
                self.process_finalized_l1_block(finalized).await
            }
        }
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
    /// An L1 observation publisher closed.
    #[error("l1 observation channel closed")]
    L1ReceiveFailed,
    /// An invalid state transition occurred.
    #[error(transparent)]
    StateTransitionError(#[from] DerivationStateTransitionError),
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::derivation::engine_client::MockDerivationEngineClient;
    use alloy_primitives::B256;
    use kona_derive::PipelineResult;
    use kona_genesis::{HardForkConfig, RollupConfig, SystemConfig};
    use kona_protocol::{BlockInfo, L2BlockInfo};
    use rstest::rstest;
    use std::sync::Arc;

    /// A pipeline stub that can advance, yield, reset, or report an L1 reorg.
    #[derive(Debug)]
    struct TestPipeline {
        rollup_config: Arc<RollupConfig>,
        origin: Option<BlockInfo>,
        advance_to: Option<BlockInfo>,
        reorg: bool,
    }

    impl Iterator for TestPipeline {
        type Item = OpAttributesWithParent;

        fn next(&mut self) -> Option<Self::Item> {
            None
        }
    }

    impl kona_derive::OriginProvider for TestPipeline {
        fn origin(&self) -> Option<BlockInfo> {
            self.origin
        }
    }

    #[async_trait]
    impl SignalReceiver for TestPipeline {
        async fn signal(&mut self, signal: Signal) -> PipelineResult<()> {
            if let Signal::Reset(reset) = signal {
                self.origin = Some(BlockInfo {
                    number: reset.l2_safe_head.l1_origin.number,
                    hash: reset.l2_safe_head.l1_origin.hash,
                    ..Default::default()
                });
            }
            Ok(())
        }
    }

    #[async_trait]
    impl Pipeline for TestPipeline {
        fn peek(&self) -> Option<&OpAttributesWithParent> {
            None
        }

        async fn step(&mut self, _: L2BlockInfo) -> StepResult {
            if let Some(origin) = self.advance_to.take() {
                self.origin = Some(origin);
                StepResult::AdvancedOrigin
            } else if self.reorg {
                StepResult::StepFailed(PipelineErrorKind::Reset(ResetError::ReorgDetected(
                    B256::ZERO,
                    B256::repeat_byte(1),
                )))
            } else {
                StepResult::StepFailed(PipelineError::Eof.temp())
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
        let (_head_tx, head_rx) = watch::channel(BlockInfo::default());
        let (_finalized_tx, finalized_rx) = watch::channel(BlockInfo::default());
        let mut actor = DerivationActor::new(
            engine_client,
            request_rx,
            TestPipeline {
                rollup_config: rollup_config.clone(),
                origin: Some(BlockInfo::default()),
                advance_to: None,
                reorg: true,
            },
            head_rx,
            finalized_rx,
        );

        // Complete EL sync so the actor starts deriving, then let it hit the reorg.
        request_tx
            .send(DerivationActorRequest::ProcessEngineSyncCompletionRequest(Box::default()))
            .await
            .unwrap();
        actor.step().await.unwrap();

        assert_eq!(actor.derivation_state_machine.current_state(), DerivationState::AwaitingSignal);
    }

    #[tokio::test]
    async fn publishes_pipeline_origin_after_advancing_and_resetting() {
        let initial = BlockInfo { number: 5, ..Default::default() };
        let advanced = BlockInfo { number: 7, ..Default::default() };
        let (tx, rx) = mpsc::channel(1);
        let (head_tx, head_rx) = watch::channel(BlockInfo::default());
        let (_finalized_tx, finalized_rx) = watch::channel(BlockInfo::default());
        let mut actor = DerivationActor::new(
            MockDerivationEngineClient::new(),
            rx,
            TestPipeline {
                rollup_config: Arc::new(RollupConfig::default()),
                origin: Some(initial),
                advance_to: Some(advanced),
                reorg: false,
            },
            head_rx,
            finalized_rx,
        );
        let status = actor.state_receiver();
        assert_eq!(status.borrow().current_l1, Some(initial));
        tx.send(DerivationActorRequest::ProcessEngineSyncCompletionRequest(Box::default()))
            .await
            .unwrap();
        actor.step().await.unwrap();
        assert_eq!(status.borrow().current_l1, Some(advanced));
        // Observing a newer head does not mean derivation has processed it.
        head_tx.send_replace(BlockInfo { number: 100, ..Default::default() });
        actor.step().await.unwrap();
        assert_eq!(status.borrow().current_l1, Some(advanced));
        let reset_origin = BlockInfo { number: 4, ..Default::default() };
        tx.send(DerivationActorRequest::ProcessEngineSignalRequest(Box::new(Signal::Reset(
            kona_derive::ResetSignal {
                l2_safe_head: L2BlockInfo { l1_origin: reset_origin.id(), ..Default::default() },
            },
        ))))
        .await
        .unwrap();
        actor.step().await.unwrap();
        assert_eq!(status.borrow().current_l1, Some(reset_origin));
    }
    #[tokio::test]
    async fn head_changes_wake_derivation_and_coalesce_reorgs() {
        let (_requests, requests) = mpsc::channel(1);
        let (heads, head_rx) = watch::channel(BlockInfo::default());
        let (_finalized, finalized_rx) = watch::channel(BlockInfo::default());
        let mut actor = DerivationActor::new(
            MockDerivationEngineClient::new(),
            requests,
            TestPipeline {
                rollup_config: Arc::new(RollupConfig::default()),
                origin: Some(BlockInfo::default()),
                advance_to: None,
                reorg: false,
            },
            head_rx,
            finalized_rx,
        );
        actor
            .derivation_state_machine
            .update(&DerivationStateUpdate::ELSyncCompleted(Box::default()))
            .unwrap();
        actor.attempt_derivation().await.unwrap();
        assert_eq!(actor.derivation_state_machine.current_state(), DerivationState::AwaitingL1Data);
        for (number, hash) in [(7, 1), (7, 2), (6, 3)] {
            let block = BlockInfo { number, hash: B256::repeat_byte(hash), ..Default::default() };
            actor.pipeline.advance_to = Some(block);
            heads.send_replace(block);
            actor.step().await.unwrap();
            assert_eq!(actor.status.borrow().current_l1, Some(block));
            assert_eq!(
                actor.derivation_state_machine.current_state(),
                DerivationState::AwaitingL1Data
            );
        }
        heads.send_replace(BlockInfo { number: 8, ..Default::default() });
        let latest = BlockInfo { number: 9, ..Default::default() };
        heads.send_replace(latest);
        actor.pipeline.advance_to = Some(latest);
        actor.step().await.unwrap();
        assert_eq!(actor.status.borrow().current_l1, Some(latest));
        assert!(!actor.head.has_changed().unwrap());
    }

    #[tokio::test]
    async fn finality_changes_finalize_without_new_heads() {
        let (_requests, requests) = mpsc::channel(1);
        let (_heads, head_rx) = watch::channel(BlockInfo::default());
        let (finalized, finalized_rx) =
            watch::channel(BlockInfo { number: 99, ..Default::default() });
        let mut engine = MockDerivationEngineClient::new();
        engine
            .expect_send_finalized_l2_block()
            .withf(|id| matches!(id, FinalizeBlockId::ByNumber(17)))
            .once()
            .returning(|_| Ok(()));
        let mut actor = DerivationActor::new(
            engine,
            requests,
            TestPipeline {
                rollup_config: Arc::new(RollupConfig::default()),
                origin: None,
                advance_to: None,
                reorg: false,
            },
            head_rx,
            finalized_rx,
        );
        actor.finalizer.enqueue_for_finalization(&OpAttributesWithParent {
            attributes: Default::default(),
            parent: L2BlockInfo {
                block_info: BlockInfo { number: 16, ..Default::default() },
                ..Default::default()
            },
            derived_from: Some(BlockInfo { number: 7, ..Default::default() }),
            is_last_in_span: false,
        });
        // Initial snapshots do not act as block events.
        tokio::select! {
            biased;
            result = actor.step() => panic!("processed an initial snapshot: {result:?}"),
            _ = std::future::ready(()) => {}
        }
        finalized.send_replace(BlockInfo { number: 6, ..Default::default() });
        actor.step().await.unwrap();
        finalized.send_replace(BlockInfo { number: 7, ..Default::default() });
        finalized.send_replace(BlockInfo { number: 8, ..Default::default() });
        actor.step().await.unwrap();
    }

    #[rstest]
    #[case::head(true)]
    #[case::finalized(false)]
    #[tokio::test]
    async fn closed_l1_publisher_is_fatal(#[case] close_head: bool) {
        let (_requests, requests) = mpsc::channel(1);
        let (heads, head_rx) = watch::channel(BlockInfo::default());
        let (finalized, finalized_rx) = watch::channel(BlockInfo::default());
        let mut actor = DerivationActor::new(
            MockDerivationEngineClient::new(),
            requests,
            TestPipeline {
                rollup_config: Arc::new(RollupConfig::default()),
                origin: None,
                advance_to: None,
                reorg: false,
            },
            head_rx,
            finalized_rx,
        );
        let mut publishers = vec![heads, finalized];
        drop(publishers.remove(usize::from(!close_head)));
        assert!(matches!(actor.step().await, Err(DerivationError::L1ReceiveFailed)));
    }
}

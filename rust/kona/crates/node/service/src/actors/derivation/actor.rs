//! [`NodeActor`] implementation for the derivation sub-routine.

use crate::{
    DerivationActorRequest, DerivationEngineClient, DerivationState, DerivationStateMachine,
    DerivationStateTransitionError, DerivationStateUpdate, Metrics, NodeActor,
    actors::derivation::L2Finalizer,
};
use async_trait::async_trait;
use kona_derive::{
    ActivationSignal, Pipeline, PipelineError, PipelineErrorKind, PipelineResult, ResetError,
    ResetSignal, Signal, SignalReceiver, StepResult,
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
    /// The Engine client used to interact with the engine.
    engine_client: DerivationEngineClient_,

    /// The derivation pipeline.
    pipeline: PipelineSignalReceiver,
    /// Publishes the pipeline's origin: the L1 block that derivation has reached.
    derivation_origin_tx: watch::Sender<Option<BlockInfo>>,
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
    pub fn new(
        engine_client: DerivationEngineClient_,
        inbound_request_rx: mpsc::Receiver<DerivationActorRequest>,
        pipeline: PipelineSignalReceiver,
        derivation_origin_tx: watch::Sender<Option<BlockInfo>>,
    ) -> Self {
        Self {
            pipeline,
            derivation_origin_tx,
            inbound_request_rx,
            engine_client,
            derivation_state_machine: DerivationStateMachine::default(),
            finalizer: L2Finalizer::default(),
        }
    }

    /// Handles a [`Signal`] received over the derivation signal receiver channel.
    async fn signal(&mut self, signal: Signal) {
        let result = match signal {
            Signal::Reset(reset) => self.signal_reset(reset).await,
            Signal::Activation(activation) => self.signal_activation(activation).await,
            Signal::FlushChannel => self.signal_flush_channel().await,
        };
        match result {
            Ok(()) => info!(target: "derivation", ?signal, "[SIGNAL] Executed Successfully"),
            Err(e) => {
                error!(target: "derivation", ?e, ?signal, "Failed to signal derivation pipeline")
            }
        }
        self.publish_origin();
    }

    /// Resets the pipeline and clears finalization and origin state before the reset starts.
    async fn signal_reset(&mut self, reset: ResetSignal) -> PipelineResult<()> {
        self.derivation_origin_tx.send_replace(None);
        self.finalizer.clear();
        self.pipeline.signal(Signal::Reset(reset)).await?;
        self.derivation_origin_tx.send_replace(self.pipeline.origin());
        Ok(())
    }

    /// Activates a hardfork while preserving the pipeline's derivation state.
    async fn signal_activation(&mut self, activation: ActivationSignal) -> PipelineResult<()> {
        self.pipeline.signal(Signal::Activation(activation)).await
    }

    /// Flushes the pipeline's currently active channel.
    async fn signal_flush_channel(&mut self) -> PipelineResult<()> {
        self.pipeline.signal(Signal::FlushChannel).await
    }

    /// Refreshes an established origin without republishing one invalidated by a reset.
    fn publish_origin(&self) {
        let pipeline_origin = self.pipeline.origin();
        self.derivation_origin_tx.send_if_modified(|origin| {
            if origin.is_some() && *origin != pipeline_origin {
                *origin = pipeline_origin;
                true
            } else {
                false
            }
        });
    }

    /// Attempts to step the derivation pipeline forward as much as possible in order to produce the
    /// next safe payload.
    async fn produce_next_attributes(&mut self) -> Result<OpAttributesWithParent, DerivationError> {
        // As we start the safe head at the disputed block's parent, we step the pipeline until the
        // first attributes are produced. All batches at and before the safe head will be
        // dropped, so the first payload will always be the disputed one.
        loop {
            let step =
                self.pipeline.step(self.derivation_state_machine.last_confirmed_safe_head()).await;
            // A failed step can also move the origin, as a Holocene activation does.
            self.publish_origin();
            match step {
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
                                // Like op-node, report no origin until the reset sets a new one.
                                self.derivation_origin_tx.send_replace(None);
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
                self.signal(*signal).await;
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
    /// An invalid state transition occurred.
    #[error(transparent)]
    StateTransitionError(#[from] DerivationStateTransitionError),
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::actors::derivation::engine_client::MockDerivationEngineClient;
    use alloy_primitives::B256;
    use kona_genesis::{HardForkConfig, RollupConfig, SystemConfig};
    use kona_protocol::{BlockInfo, L2BlockInfo};
    use rstest::rstest;
    use std::sync::Arc;

    /// A pipeline stub that returns `step_result` once, then yields. Origin advances increment
    /// its L1 block number, and a reset moves it to the safe head's L1 origin.
    #[derive(Debug)]
    struct StubPipeline {
        rollup_config: Arc<RollupConfig>,
        origin: BlockInfo,
        step_result: StepResult,
        reset_error: Option<PipelineErrorKind>,
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
            if let Signal::Reset(reset) = signal {
                if let Some(error) = self.reset_error.take() {
                    return Err(error);
                }
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
                reset_error: None,
                step_result: StepResult::StepFailed(
                    ResetError::ReorgDetected(B256::ZERO, B256::repeat_byte(1)).reset(),
                ),
            },
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
                reset_error: None,
            },
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
        assert_eq!(
            derivation_origin_rx.borrow().map(|origin| origin.id()),
            Some(safe_head.l1_origin)
        );

        request_tx
            .send(DerivationActorRequest::ProcessEngineSyncCompletionRequest(Box::new(safe_head)))
            .await
            .unwrap();
        actor.step().await.unwrap();
        assert_eq!(derivation_origin_rx.borrow().unwrap().number, 11);

        request_tx.send(reset()).await.unwrap();
        actor.step().await.unwrap();
        assert_eq!(
            derivation_origin_rx.borrow().map(|origin| origin.id()),
            Some(safe_head.l1_origin)
        );
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
                reset_error: None,
            },
            derivation_origin_tx,
        );

        actor.signal(signal).await;
        assert!(derivation_origin_rx.borrow().is_none());
        assert!(matches!(actor.produce_next_attributes().await, Err(DerivationError::Yield)));
        assert_eq!(actor.pipeline.origin.number, 1);
        assert!(derivation_origin_rx.borrow().is_none());
    }

    #[rstest]
    #[case::activation(Signal::Activation(ActivationSignal::default()))]
    #[case::flush_channel(Signal::FlushChannel)]
    #[tokio::test]
    async fn test_failed_reset_keeps_origin_unpublished_until_successful_reset(
        #[case] signal: Signal,
    ) {
        let (derivation_origin_tx, derivation_origin_rx) = watch::channel(None);
        let (request_tx, request_rx) = mpsc::channel(1);
        let mut actor = DerivationActor::new(
            MockDerivationEngineClient::new(),
            request_rx,
            StubPipeline {
                rollup_config: Arc::new(RollupConfig::default()),
                origin: BlockInfo::default(),
                reset_error: None,
                step_result: StepResult::AdvancedOrigin,
            },
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
        assert_eq!(
            derivation_origin_rx.borrow().map(|origin| origin.id()),
            Some(safe_head.l1_origin)
        );

        // A reset can fail before it replaces the old origin, for example on an L2 RPC error.
        actor.pipeline.reset_error = Some(PipelineError::Eof.temp());
        request_tx.send(reset()).await.unwrap();
        actor.step().await.unwrap();
        assert_eq!(actor.pipeline.origin.id(), safe_head.l1_origin);
        assert!(derivation_origin_rx.borrow().is_none());

        // Neither a successful non-reset signal nor a step makes that origin valid again.
        request_tx
            .send(DerivationActorRequest::ProcessEngineSignalRequest(Box::new(signal)))
            .await
            .unwrap();
        actor.step().await.unwrap();
        assert!(derivation_origin_rx.borrow().is_none());
        request_tx
            .send(DerivationActorRequest::ProcessEngineSyncCompletionRequest(Box::new(safe_head)))
            .await
            .unwrap();
        actor.step().await.unwrap();
        assert_eq!(actor.pipeline.origin.number, 11);
        assert!(derivation_origin_rx.borrow().is_none());

        request_tx.send(reset()).await.unwrap();
        actor.step().await.unwrap();
        assert_eq!(
            derivation_origin_rx.borrow().map(|origin| origin.id()),
            Some(safe_head.l1_origin)
        );
    }
}

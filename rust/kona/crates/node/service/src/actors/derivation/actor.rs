//! [`NodeActor`] implementation for the derivation sub-routine.

use crate::{
    DerivationActorRequest, DerivationEngineClient, DerivationState, DerivationStateMachine,
    DerivationStateTransitionError, DerivationStateUpdate, Metrics, NodeActor,
    actors::derivation::L2Finalizer,
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
    safe_db: Arc<dyn SafeDb>,
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
            safe_db,
            derivation_state_machine: DerivationStateMachine::default(),
            finalizer: L2Finalizer::default(),
        }
    }

    /// Handles a [`Signal`] received over the derivation signal receiver channel.
    async fn signal(&mut self, signal: Signal) {
        if matches!(signal, Signal::Reset(_)) {
            // Clear the finalization queue on reset.
            self.finalizer.clear();
        }

        match self.pipeline.signal(signal).await {
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
                                // send the `reset` signal to the engine actor only when interop is
                                // not active.
                                if !self.pipeline.rollup_config().is_interop_active(
                                    self.derivation_state_machine
                                        .last_confirmed_safe_head()
                                        .block_info
                                        .timestamp,
                                ) {
                                    self.engine_client.reset_engine_forkchoice().await.map_err(|e| {
                                        error!(target: "derivation", ?e, "Failed to send reset request");
                                        DerivationError::Sender(Box::new(e))
                                    })?;
                                }
                                self.safe_db.safe_head_reset(
                                    self.derivation_state_machine.last_confirmed_safe_head(),
                                )?;
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
                self.record_safe_head(*safe_head)?;
                self.derivation_state_machine
                    .update(&DerivationStateUpdate::NewAttributesConfirmed(safe_head))?;

                self.attempt_derivation().await?;
            }
            DerivationActorRequest::ProcessEngineSyncCompletionRequest(safe_head) => {
                info!(target: "derivation", "Engine finished syncing, starting derivation.");
                // EL sync can land on a chain that diverges from what was recorded, so drop
                // anything at or above the synced head and keep the contiguous history below it.
                self.safe_db.safe_head_reset(*safe_head)?;
                self.derivation_state_machine
                    .update(&DerivationStateUpdate::ELSyncCompleted(safe_head))?;

                self.attempt_derivation().await?;
            }
        }

        Ok(())
    }

    /// Records that `safe_head` became safe as of the pipeline's current L1 origin.
    ///
    /// The origin is read here rather than carried from where the attributes were derived
    /// because the state machine blocks derivation between `NewAttributesDerived` and this
    /// confirmation, so the pipeline cannot have advanced its origin in between.
    fn record_safe_head(&self, safe_head: L2BlockInfo) -> Result<(), DerivationError> {
        if !self.safe_db.enabled() {
            return Ok(());
        }
        let origin = self.pipeline.origin().ok_or(PipelineError::MissingOrigin.crit())?;
        self.safe_db.safe_head_updated(safe_head, origin.id())?;
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
    /// The safe-head database rejected an update. Derivation stops rather than continuing with
    /// a recorded history that no longer matches the chain.
    #[error(transparent)]
    SafeDb(#[from] SafeDbError),
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
    use alloy_eips::BlockNumHash;
    use alloy_primitives::B256;
    use kona_derive::{
        PipelineBuilder,
        test_utils::{TestAttributesBuilder, TestChainProvider, TestDAP, TestL2ChainProvider},
    };
    use kona_genesis::RollupConfig;
    use kona_protocol::BlockInfo;
    use kona_safedb::SafeDatabase;
    use tempfile::TempDir;

    const PIPELINE_ORIGIN: u64 = 100;
    /// Deliberately different from [`PIPELINE_ORIGIN`] so a test that passes cannot be recording
    /// the safe head's own L1 origin by mistake.
    const SAFE_HEAD_L1_ORIGIN: u64 = 42;

    fn l1_origin_block() -> BlockInfo {
        BlockInfo {
            hash: B256::repeat_byte(1),
            number: PIPELINE_ORIGIN,
            parent_hash: B256::ZERO,
            timestamp: 0,
        }
    }

    fn safe_head() -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo {
                hash: B256::repeat_byte(2),
                number: 20,
                parent_hash: B256::ZERO,
                timestamp: 0,
            },
            l1_origin: BlockNumHash { hash: B256::ZERO, number: SAFE_HEAD_L1_ORIGIN },
            seq_num: 0,
        }
    }

    fn actor(
        safe_db: Arc<dyn SafeDb>,
    ) -> DerivationActor<MockDerivationEngineClient, impl Pipeline + SignalReceiver> {
        let pipeline = PipelineBuilder::new()
            .rollup_config(Arc::new(RollupConfig::default()))
            .origin(l1_origin_block())
            .dap_source(TestDAP::default())
            .builder(TestAttributesBuilder::default())
            .chain_provider(TestChainProvider::default())
            .l2_chain_provider(TestL2ChainProvider::default())
            .build_polled();
        let (_tx, rx) = mpsc::channel(1);
        DerivationActor::new(MockDerivationEngineClient::new(), rx, pipeline, safe_db)
    }

    #[tokio::test]
    async fn records_the_safe_head_against_the_pipeline_origin() {
        let dir = TempDir::new().unwrap();
        let db = Arc::new(SafeDatabase::new(dir.path()).unwrap());
        let mut actor = actor(db.clone());

        actor
            .handle_derivation_actor_request(
                DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(Box::new(safe_head())),
            )
            .await
            .unwrap();

        let record = db.last_entry().unwrap();
        assert_eq!(
            record.l1.number, PIPELINE_ORIGIN,
            "must record the L1 block derivation reached, not the safe head's own origin"
        );
        assert_eq!(record.l1.hash, l1_origin_block().hash);
        assert_eq!(record.safe_head, safe_head().block_info.id());
    }

    #[tokio::test]
    async fn records_nothing_when_the_database_is_disabled() {
        let mut actor = actor(Arc::new(kona_safedb::DisabledDatabase));

        actor
            .handle_derivation_actor_request(
                DerivationActorRequest::ProcessEngineSafeHeadUpdateRequest(Box::new(safe_head())),
            )
            .await
            .expect("a disabled database must not fail the request");
    }
}

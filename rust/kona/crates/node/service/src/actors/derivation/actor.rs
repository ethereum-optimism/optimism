//! [`NodeActor`] implementation for the derivation sub-routine.

use crate::{
    DerivationActorRequest, DerivationEngineClient, DerivationState, DerivationStateMachine,
    DerivationStateTransitionError, DerivationStateUpdate, Metrics, NodeActor,
    actors::derivation::{L2Finalizer, SafeHeadHistory},
};
use async_trait::async_trait;
use kona_derive::{
    ActivationSignal, Pipeline, PipelineError, PipelineErrorKind, PipelineResult, ResetError,
    Signal, SignalReceiver, StepResult,
};
use kona_engine::FinalizeBlockId;
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use kona_safedb::{SafeDb, SafeDbError};
use std::sync::Arc;
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
    /// Records the safe head reached as of each locally derived L1 block.
    safe_db: Option<Arc<dyn SafeDb>>,
    /// The attributes sent to the engine and not yet confirmed as a safe head.
    attributes_awaiting_confirmation: Option<OpAttributesWithParent>,
    /// The recent safe heads an L1 reorg may keep, with the L1 data they depend on.
    safe_head_history: SafeHeadHistory,
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
        safe_db: Option<Arc<dyn SafeDb>>,
        derivation_origin_tx: watch::Sender<Option<BlockInfo>>,
    ) -> Self {
        Self {
            pipeline,
            derivation_origin_tx,
            inbound_request_rx,
            engine_client,
            derivation_state_machine: DerivationStateMachine::default(),
            finalizer: L2Finalizer::default(),
            safe_db,
            attributes_awaiting_confirmation: None,
            safe_head_history: SafeHeadHistory::default(),
        }
    }

    /// Handles a [`Signal`] received over the derivation signal receiver channel.
    async fn signal(&mut self, signal: Signal) -> Result<(), DerivationError> {
        let result = match signal {
            Signal::Reset(reset) => {
                self.derivation_origin_tx.send_replace(None);
                self.finalizer.clear();
                self.attributes_awaiting_confirmation = None;
                self.reset_safe_head_history(reset.l2_safe_head);
                self.truncate_safe_db(reset.l2_safe_head, "engine reset")?;
                let result = self.pipeline.signal(Signal::Reset(reset)).await;
                if result.is_ok() {
                    self.derivation_origin_tx.send_replace(self.pipeline.origin());
                }
                result
            }
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
        Ok(())
    }

    /// Restarts the [`SafeHeadHistory`] from the safe head derivation restarts from.
    fn reset_safe_head_history(&mut self, safe_head: L2BlockInfo) {
        let seq_window_size = self.pipeline.rollup_config().seq_window_size;
        self.safe_head_history.reset_to(safe_head, seq_window_size);
    }

    /// Asks the engine to reset for a pipeline [`ResetError`]. An L1 reorg lets the engine keep a
    /// safe head the reorg left derivable, rather than rewind a full sequencing window.
    async fn reset_engine(&self, e: &ResetError) -> Result<(), DerivationError> {
        let reset = if let ResetError::ReorgDetected(expected, new) = e {
            warn!(target: "derivation", "L1 reorg detected! Expected: {expected} | New: {new}");
            metrics::counter!(Metrics::L1_REORG_COUNT).increment(1);

            // The pipeline's origin has no canonical child, so it was reorged out, and so was every
            // L1 block at or above it. Drop the safe heads that depend on one without asking the
            // engine, whose L1 node may not have seen the reorg yet.
            let reorged_from = self.pipeline.origin().map_or(u64::MAX, |origin| origin.number);
            let recent_safe_heads = self
                .safe_head_history
                .newest_first()
                .into_iter()
                .filter(|derived| derived.derived_from.number < reorged_from)
                .collect();
            self.engine_client.reset_engine_after_l1_reorg(recent_safe_heads).await
        } else {
            self.engine_client.reset_engine_forkchoice().await
        };

        reset.map_err(|e| {
            error!(target: "derivation", ?e, "Failed to send reset request");
            DerivationError::Sender(Box::new(e))
        })
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
                            } else {
                                // Like op-node, report no origin until the reset sets a new one.
                                self.derivation_origin_tx.send_replace(None);
                                self.reset_engine(&e).await?;
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
                self.signal(*signal).await?;
                // A channel flush leaves the outstanding engine confirmation pending.
                if !matches!(*signal, Signal::FlushChannel) {
                    self.derivation_state_machine
                        .update(&DerivationStateUpdate::SignalProcessed)?;
                }
                // The engine reports its safe head only when it changes, so a reset that keeps
                // the confirmed safe head, as one after a shallow L1 reorg usually does, gets no
                // report. Resume from it instead of waiting for the next L1 head.
                if let Signal::Reset(reset) = *signal &&
                    reset.l2_safe_head ==
                        self.derivation_state_machine.last_confirmed_safe_head() &&
                    self.derivation_state_machine.current_state() ==
                        DerivationState::AwaitingUpdateAfterSignal
                {
                    self.derivation_state_machine.update(
                        &DerivationStateUpdate::NewAttributesConfirmed(Box::new(
                            reset.l2_safe_head,
                        )),
                    )?;
                    self.attempt_derivation().await?;
                }
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
                    .update(&DerivationStateUpdate::NewAttributesConfirmed(safe_head.clone()))?;
                self.settle_derived_attributes(*safe_head)?;

                self.attempt_derivation().await?;
            }
            DerivationActorRequest::ProcessEngineSyncCompletionRequest(safe_head) => {
                info!(target: "derivation", "Engine finished syncing, starting derivation.");
                self.derivation_state_machine
                    .update(&DerivationStateUpdate::ELSyncCompleted(safe_head.clone()))?;
                self.reset_safe_head_history(*safe_head);
                self.truncate_safe_db(*safe_head, "EL sync completion")?;

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
    fn settle_derived_attributes(&mut self, safe_head: L2BlockInfo) -> Result<(), DerivationError> {
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
        if let Some(derived_from) = attributes.derived_from.map(|block| block.id()) {
            // Both records take the L1 block the history works out: the derivation source,
            // raised to the L1 block the parent was recorded at, since a block cannot become safe
            // before its parent. Below a floor known only as a bound, the history records nothing
            // and the database keeps the derivation source.
            let safe_at =
                self.safe_head_history.record(safe_head, derived_from).unwrap_or(derived_from);
            if let Some(database) = &self.safe_db {
                debug!(target: "derivation", l1_block = safe_at.number, safe_head = safe_head.block_info.number, "Recording safe head");
                database.safe_head_updated(safe_head, safe_at).inspect_err(|error| {
                    error!(target: "derivation", %error, l1_block = safe_at.number, safe_head = safe_head.block_info.number, "Failed to record safe head");
                })?;
                debug!(target: "derivation", l1_block = safe_at.number, safe_head = safe_head.block_info.number, "Recorded safe head");
            }
        }
        self.finalizer.enqueue_for_finalization(&attributes);
        Ok(())
    }

    fn truncate_safe_db(&self, resume_head: L2BlockInfo, reason: &str) -> Result<(), SafeDbError> {
        let Some(database) = &self.safe_db else {
            return Ok(());
        };
        debug!(target: "derivation", reason, resume_head = resume_head.block_info.number, "Reading safe head database tip before truncation");
        match database.last_entry() {
            Ok(tip) if tip.safe_head.number < resume_head.block_info.number => {
                warn!(target: "derivation", reason, resume_head = resume_head.block_info.number, tip = tip.safe_head.number, "Safe head database tip is below the resume point; clearing history to avoid a gap");
                database.clear().inspect_err(|error| {
                    error!(target: "derivation", %error, reason, resume_head = resume_head.block_info.number, "Failed to clear safe head database");
                })?;
                info!(target: "derivation", reason, resume_head = resume_head.block_info.number, "Cleared safe head database");
                return Ok(());
            }
            Ok(tip) => {
                debug!(target: "derivation", reason, tip = tip.safe_head.number, "Read safe head database tip")
            }
            Err(SafeDbError::NotFound) => {
                debug!(target: "derivation", reason, "Safe head database has no recorded history")
            }
            Err(error) => {
                warn!(target: "derivation", %error, reason, resume_head = resume_head.block_info.number, "Failed to read safe head database tip before truncation")
            }
        }
        info!(target: "derivation", reason, resume_head = resume_head.block_info.number, "Truncating safe head database");
        database.safe_head_reset(resume_head).inspect_err(|error| {
            error!(target: "derivation", %error, reason, resume_head = resume_head.block_info.number, "Failed to truncate safe head database");
        })?;
        info!(target: "derivation", reason, resume_head = resume_head.block_info.number, "Truncated safe head database");
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
    SafeDbError(#[from] SafeDbError),
    /// An invalid state transition occurred.
    #[error(transparent)]
    StateTransitionError(#[from] DerivationStateTransitionError),
}

#[cfg(test)]
mod tests;

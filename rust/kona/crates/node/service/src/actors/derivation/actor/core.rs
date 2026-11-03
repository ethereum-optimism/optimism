//! [`NodeActor`] implementation for the derivation sub-routine.

use crate::{
    DerivationActorRequest, DerivationEngineClient, DerivationState, DerivationStateMachine,
    DerivationStateTransitionError, DerivationStateUpdate, Metrics, NodeActor,
    actors::derivation::L2Finalizer,
};
use alloy_eips::BlockNumHash;
use async_trait::async_trait;
use kona_derive::{
    ActivationSignal, Pipeline, PipelineError, PipelineErrorKind, ResetError, ResetSignal, Signal,
    SignalReceiver, StepResult,
};
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use kona_safedb::{SafeDb, SafeDbError};
use std::{sync::Arc, time::Duration};
use thiserror::Error;
use tokio::{
    sync::{mpsc, watch},
    time::Instant,
};

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
    pub(super) pipeline: PipelineSignalReceiver,
    /// Publishes the pipeline's origin: the L1 block that derivation has reached.
    derivation_origin_tx: watch::Sender<Option<BlockInfo>>,
    /// The state machine controlling when derivation can occur.
    pub(super) derivation_state_machine: DerivationStateMachine,
    /// The [`L2Finalizer`] tracks derived L2 blocks awaiting finalization.
    pub(crate) finalizer: L2Finalizer,
    /// Records the safe head reached as of each locally derived L1 block.
    safe_db: Option<Arc<dyn SafeDb>>,
    /// Genesis is safe from L1 block zero, even when the configured L1 genesis is later.
    l1_block_zero: BlockNumHash,
    /// The attributes sent to the engine and not yet confirmed as a safe head.
    pub(super) attributes_awaiting_confirmation: Option<OpAttributesWithParent>,
    /// A pipeline reset that has not yet succeeded. Derivation must not resume before it does.
    pending_reset: Option<PendingReset>,
    /// How many engine resets this actor has handled. Attributes and finalizations carry it so
    /// that the engine can drop those derived before its latest reset.
    reset_epoch: u64,
}

/// A pipeline reset that has not yet succeeded.
#[derive(Debug)]
enum PendingReset {
    /// The reset head was reorged out of the L2 chain, so the engine was asked for a fresh one.
    AwaitingFreshHead,
    /// The reset signal failed temporarily and is retried with backoff.
    Retry(ScheduledRetry),
}

/// A reset signal retried after a temporary failure.
#[derive(Debug)]
struct ScheduledRetry {
    attempts: u32,
    retry_at: Instant,
    signal: ResetSignal,
}

impl ScheduledRetry {
    /// op-node retries a temporarily failed reset through its exponential step backoff, capped
    /// at ten seconds.
    const MAX_RETRY_DELAY: Duration = Duration::from_secs(10);
    const MIN_RETRY_DELAY: Duration = Duration::from_millis(100);

    fn first(signal: ResetSignal) -> Self {
        Self::with_attempts(signal, 1)
    }

    fn retried(&self) -> Self {
        Self::with_attempts(self.signal, self.attempts + 1)
    }

    fn with_attempts(signal: ResetSignal, attempts: u32) -> Self {
        Self { attempts, retry_at: Instant::now() + Self::retry_delay(attempts), signal }
    }

    fn retry_delay(attempts: u32) -> Duration {
        let doublings = attempts.saturating_sub(1).min(16);
        Self::MIN_RETRY_DELAY.saturating_mul(1 << doublings).min(Self::MAX_RETRY_DELAY)
    }
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
        l1_block_zero: BlockNumHash,
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
            l1_block_zero,
            attributes_awaiting_confirmation: None,
            pending_reset: None,
            reset_epoch: 0,
        }
    }

    /// Forwards an activation or channel flush to the pipeline. Either failure leaves the
    /// derivation state intact, so it is logged and the actor continues. A reset is handled by
    /// [`Self::reset_pipeline_and_resume`] instead.
    pub(super) async fn forward_non_reset_signal(&mut self, signal: Signal) {
        match self.pipeline.signal(signal).await {
            Ok(()) => info!(target: "derivation", ?signal, "[SIGNAL] Executed Successfully"),
            Err(e) => {
                error!(target: "derivation", ?e, ?signal, "Failed to signal derivation pipeline")
            }
        }
        self.publish_origin();
    }

    /// Resets the pipeline to the engine's safe head, adopts that head, and resumes derivation.
    ///
    /// `SafeDB` is truncated before the pipeline resets, so derivation must not resume until the
    /// pipeline reset has succeeded; [`Self::attempt_derivation`] holds while a reset is pending,
    /// whichever actor requested it. Like op-node, a temporary pipeline failure, such as an L2
    /// RPC error while the pipeline walks back to its L1 origin, schedules a retry with backoff,
    /// and a reset head that was reorged out asks the engine for a fresh one. Only a critical
    /// failure stops the actor.
    pub(super) async fn reset_pipeline_and_resume(
        &mut self,
        reset: ResetSignal,
    ) -> Result<(), DerivationError> {
        self.derivation_origin_tx.send_replace(None);
        self.finalizer.clear();
        self.attributes_awaiting_confirmation = None;
        self.truncate_safe_db(reset.l2_safe_head, "engine reset")?;
        match self.pipeline.signal(Signal::Reset(reset)).await {
            Ok(()) => info!(target: "derivation", ?reset, "[SIGNAL] Executed Successfully"),
            Err(PipelineErrorKind::Temporary(error)) => {
                self.schedule_reset_retry(reset, error);
                return Ok(());
            }
            Err(PipelineErrorKind::Reset(error)) => {
                return self.request_fresh_reset_head(reset, error).await;
            }
            Err(error @ PipelineErrorKind::Critical(_)) => {
                error!(target: "derivation", ?error, ?reset, "Failed to reset derivation pipeline");
                metrics::counter!(Metrics::DERIVATION_CRITICAL_ERROR).increment(1);
                return Err(error.into());
            }
        }
        self.pending_reset = None;
        metrics::gauge!(Metrics::DERIVATION_PENDING_RESET_ATTEMPTS).set(0.0);
        self.derivation_origin_tx.send_replace(self.pipeline.origin());
        self.derivation_state_machine.update(&DerivationStateUpdate::SignalProcessed)?;
        // Adopt the reset head and resume in this request. A separate confirmation could arrive
        // after an L1 update has already resumed derivation.
        self.derivation_state_machine
            .update(&DerivationStateUpdate::NewAttributesConfirmed(Box::new(reset.l2_safe_head)))?;
        self.attempt_derivation().await
    }

    /// Schedules `reset` to be signaled again after a backoff that grows with each failed
    /// attempt of that same signal.
    fn schedule_reset_retry(&mut self, reset: ResetSignal, error: PipelineError) {
        let retry = match self.pending_reset.take() {
            Some(PendingReset::Retry(retry)) if retry.signal == reset => retry.retried(),
            _ => ScheduledRetry::first(reset),
        };
        warn!(
            target: "derivation",
            ?error,
            ?reset,
            attempt = retry.attempts,
            retry_delay = ?ScheduledRetry::retry_delay(retry.attempts),
            "Pipeline reset failed temporarily; scheduling a retry"
        );
        metrics::gauge!(Metrics::DERIVATION_PENDING_RESET_ATTEMPTS).set(f64::from(retry.attempts));
        self.pending_reset = Some(PendingReset::Retry(retry));
    }

    /// Asks the engine for a fresh reset head. The pipeline could not walk back from the head in
    /// `reset`, so signaling it again cannot succeed; like op-node, that failure starts a new
    /// reset rather than a retry.
    async fn request_fresh_reset_head(
        &mut self,
        reset: ResetSignal,
        error: ResetError,
    ) -> Result<(), DerivationError> {
        warn!(
            target: "derivation",
            ?error,
            ?reset,
            "Pipeline reset head is no longer canonical; requesting a fresh engine reset"
        );
        self.pending_reset = Some(PendingReset::AwaitingFreshHead);
        metrics::gauge!(Metrics::DERIVATION_PENDING_RESET_ATTEMPTS).set(0.0);
        self.engine_client.reset_engine_forkchoice().await.map_err(|e| {
            error!(target: "derivation", ?e, "Failed to send reset request");
            DerivationError::Sender(Box::new(e))
        })
    }

    async fn retry_pending_reset(&mut self) -> Result<(), DerivationError> {
        let Some(PendingReset::Retry(retry)) = &self.pending_reset else {
            return Ok(());
        };
        let reset = retry.signal;
        info!(target: "derivation", ?reset, attempt = retry.attempts + 1, "Retrying pipeline reset");
        self.reset_pipeline_and_resume(reset).await
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
    pub(super) async fn produce_next_attributes(
        &mut self,
    ) -> Result<OpAttributesWithParent, DerivationError> {
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
                                if let ResetError::ReorgDetected(expected, new) = e {
                                    warn!(
                                        target: "derivation",
                                        "L1 reorg detected! Expected: {expected} | New: {new}"
                                    );

                                    metrics::counter!(Metrics::L1_REORG_COUNT).increment(1);
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

    pub(super) async fn handle_derivation_actor_request(
        &mut self,
        request_type: DerivationActorRequest,
    ) -> Result<(), DerivationError> {
        match request_type {
            DerivationActorRequest::ProcessEngineSignalRequest(signal) => match *signal {
                Signal::Reset(reset) => {
                    // Count each engine reset once, not each retry of its pipeline reset.
                    self.reset_epoch += 1;
                    self.reset_pipeline_and_resume(reset).await?
                }
                Signal::Activation(_) => {
                    self.forward_non_reset_signal(*signal).await;
                    self.derivation_state_machine
                        .update(&DerivationStateUpdate::SignalProcessed)?;
                }
                // A channel flush leaves the outstanding engine confirmation pending.
                Signal::FlushChannel => self.forward_non_reset_signal(*signal).await,
            },
            DerivationActorRequest::ProcessFinalizedL1Block(finalized_l1_block) => {
                // Attempt to finalize the block. If successful, notify engine.
                if let Some(l2_block_number) = self.finalizer.try_finalize_next(*finalized_l1_block)
                {
                    // Local L1-finality: the engine's own canonical chain is the authoritative
                    // source at this height, so finalize by number.
                    self.engine_client
                        .send_derived_finalization(l2_block_number, self.reset_epoch)
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
                // Only consolidating the attributes derived here moves the engine's safe head, and
                // each consolidation advances it. Adopting any other head would skip blocks that
                // were never derived and leave a gap in the safe head database.
                let Some(attributes) = self
                    .attributes_awaiting_confirmation
                    .take_if(|attributes| attributes_produce(attributes, &safe_head))
                else {
                    warn!(
                        target: "derivation",
                        safe_head = safe_head.block_info.number,
                        awaiting = ?self.attributes_awaiting_confirmation.as_ref().map(|attributes| attributes.block_number()),
                        "Ignoring a safe head that does not confirm the derived attributes"
                    );
                    return Ok(());
                };
                self.derivation_state_machine
                    .update(&DerivationStateUpdate::NewAttributesConfirmed(safe_head.clone()))?;
                self.record_confirmed_attributes(&attributes, *safe_head)?;

                self.attempt_derivation().await?;
            }
            DerivationActorRequest::ProcessEngineSyncCompletionRequest(safe_head) => {
                info!(target: "derivation", "Engine finished syncing, starting derivation.");
                self.derivation_state_machine
                    .update(&DerivationStateUpdate::ELSyncCompleted(safe_head.clone()))?;
                self.truncate_safe_db(*safe_head, "EL sync completion")?;

                self.attempt_derivation().await?;
            }
        }

        Ok(())
    }

    /// Records and queues for finalization the derived `attributes` that `safe_head` confirms.
    ///
    /// Finalization is queued only once the record is written, so the engine's finalized head can
    /// never run ahead of the recorded history.
    pub(super) fn record_confirmed_attributes(
        &mut self,
        attributes: &OpAttributesWithParent,
        safe_head: L2BlockInfo,
    ) -> Result<(), DerivationError> {
        if let (Some(database), Some(derived_from)) = (&self.safe_db, attributes.derived_from) {
            debug!(target: "derivation", l1_block = derived_from.number, safe_head = safe_head.block_info.number, "Recording safe head");
            database.safe_head_updated(safe_head, derived_from.id()).inspect_err(|error| {
                error!(target: "derivation", %error, l1_block = derived_from.number, safe_head = safe_head.block_info.number, "Failed to record safe head");
            })?;
            debug!(target: "derivation", l1_block = derived_from.number, safe_head = safe_head.block_info.number, "Recorded safe head");
        }
        self.finalizer.enqueue_for_finalization(attributes);
        Ok(())
    }

    pub(super) fn truncate_safe_db(
        &self,
        resume_head: L2BlockInfo,
        reason: &str,
    ) -> Result<(), SafeDbError> {
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
        if resume_head.block_info.id() == self.pipeline.rollup_config().genesis.l2 {
            debug!(target: "derivation", reason, l1_block = 0, safe_head = resume_head.block_info.number, "Recording genesis safe head");
            database.safe_head_updated(resume_head, self.l1_block_zero).inspect_err(|error| {
                error!(target: "derivation", %error, reason, safe_head = resume_head.block_info.number, "Failed to record genesis safe head");
            })?;
            info!(target: "derivation", reason, safe_head = resume_head.block_info.number, "Recorded genesis safe head at L1 block zero");
        }
        Ok(())
    }

    /// Attempts to process the next payload attributes.
    async fn attempt_derivation(&mut self) -> Result<(), DerivationError> {
        if self.derivation_state_machine.current_state() != DerivationState::Deriving {
            info!(target: "derivation", derivation_state=?self.derivation_state_machine, "Skipping derivation.");
            return Ok(());
        }
        if let Some(pending_reset) = &self.pending_reset {
            // SafeDB is already truncated to the reset head, so the pipeline must reset before
            // it is stepped. Wait for that reset the same way a drained data source is awaited.
            info!(target: "derivation", ?pending_reset, "Holding derivation until the pipeline reset succeeds.");
            self.derivation_state_machine.update(&DerivationStateUpdate::MoreDataNeeded)?;
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
            .send_derived_attributes(payload_attributes, self.reset_epoch)
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
        let retry_at = match &self.pending_reset {
            Some(PendingReset::Retry(retry)) => Some(retry.retry_at),
            Some(PendingReset::AwaitingFreshHead) | None => None,
        };
        let pending_reset_due = async move {
            match retry_at {
                Some(retry_at) => tokio::time::sleep_until(retry_at).await,
                None => std::future::pending().await,
            }
        };
        tokio::select! {
            // Inbound requests win, so a busy queue can hold a due retry past `retry_at`.
            // Requests are infrequent, so that delay is accepted.
            biased;
            request = self.inbound_request_rx.recv() => {
                let request = request.ok_or_else(|| {
                    error!(
                        target: "derivation",
                        "DerivationActor inbound request receiver closed unexpectedly",
                    );
                    DerivationError::RequestReceiveFailed
                })?;
                self.handle_derivation_actor_request(request).await
            }
            () = pending_reset_due => self.retry_pending_reset().await,
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
    /// A safe head database write failed. The actor stops rather than leave a gap that later
    /// queries would serve as an older safe head.
    #[error("safe head db write failed")]
    SafeDbError(#[from] SafeDbError),
    /// An invalid state transition occurred.
    #[error(transparent)]
    StateTransitionError(#[from] DerivationStateTransitionError),
}

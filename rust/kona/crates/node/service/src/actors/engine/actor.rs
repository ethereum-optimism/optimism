use crate::{
    BuildRequest, CanonicalizeRequest, EngineDerivationClient, EngineError, NodeActor, SealRequest,
};
use async_trait::async_trait;
use kona_derive::{ResetSignal, Signal};
use kona_engine::{
    BuildTask, CanonicalizeTask, ConsolidateTask, DEFAULT_EL_SYNC_STALL_TIMEOUT, DrainScope,
    Engine, EngineActorRequest, EngineClient, EngineTask, EngineTaskError, EngineTaskErrorSeverity,
    EngineTaskErrors, FinalizeTask, ImportedBlockSink, InsertTask, SealTask,
};
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use std::sync::Arc;
use tokio::{
    sync::{mpsc, watch},
    time::{self, Duration, Instant},
};

#[cfg(test)]
#[path = "queue_tests.rs"]
mod queue_tests;

/// The delay before the first retry after a dependency failure; a clean drain resets the backoff
/// to it.
pub(crate) const INITIAL_RETRY_DELAY: Duration = Duration::from_millis(100);

/// The cap on the doubling retry delay, keeping recovery responsive.
pub(crate) const MAX_RETRY_DELAY: Duration = Duration::from_secs(5);

/// The interval between retries of a task waiting for the execution layer to sync. Each retry costs
/// RPCs on an execution layer busy syncing, and a wait is not a failure, so it leaves the failure
/// backoff alone.
pub(crate) const WAIT_RETRY_INTERVAL: Duration = Duration::from_secs(2);

/// Responsible for managing the operations sent to the execution layer's Engine API. To accomplish
/// this, it uses the [`Engine`] task queue to order Engine API  interactions based off of
/// the [`Ord`] implementation of [`EngineTask`].
#[derive(Debug)]
pub struct EngineActor<DerivationClient>
where
    DerivationClient: EngineDerivationClient,
{
    /// The client used to send messages to the [`crate::DerivationActor`].
    derivation_client: DerivationClient,
    /// Whether the EL sync is complete. This should only ever go from false to true.
    el_sync_complete: bool,
    /// How long a consolidation waits for an execution layer whose sync reports no progress.
    el_sync_stall_timeout: Duration,
    /// The last local-safe head sent to the derivation actor.
    last_engine_safe_head_sent: L2BlockInfo,
    /// The error of the last skipped engine task, so an identical repeat logs no new warning
    /// until a reset changes the state the task was judged against.
    last_skipped_task_error: Option<String>,
    /// A channel to use to relay the current unsafe head.
    /// ## Note
    /// This is `Some` when the node is in sequencer mode, and `None` when the node is in validator
    /// mode.
    unsafe_head_tx: Option<watch::Sender<L2BlockInfo>>,

    /// Where to hand every imported block, so the derivation providers can read it locally
    /// instead of fetching it back from the execution layer.
    block_sink: Arc<dyn ImportedBlockSink>,
    /// An [`EngineClient`] used for creating engine tasks.
    client: Arc<EngineClient>,
    /// The [`Engine`] task queue.
    engine: Engine,
    /// The inbound request channel.
    inbound_request_rx: mpsc::Receiver<EngineActorRequest>,
    /// The [`RollupConfig`] used to build tasks.
    rollup: Arc<RollupConfig>,

    /// A reset is retained until successful; RPC failure must not lose the request.
    reset_pending: bool,
    /// Callers waiting for the retained reset to finish.
    reset_waiters: Vec<mpsc::Sender<crate::EngineClientResult<()>>>,
    /// Earliest next attempt after a dependency failure.
    retry_at: Instant,
    /// Exponentially increasing retry delay, capped to keep recovery responsive.
    retry_delay: Duration,
    /// Earliest retry of the task parked while it waits for the execution layer. Kept apart from
    /// `retry_at`, so a failure of a task that runs ahead of it cannot bring its retry forward.
    waiting_retry_at: Instant,
}

impl<DerivationClient> EngineActor<DerivationClient>
where
    DerivationClient: EngineDerivationClient + 'static,
{
    /// Constructs a new [`EngineActor`] from the params.
    pub fn new(
        client: Arc<EngineClient>,
        config: Arc<RollupConfig>,
        derivation_client: DerivationClient,
        engine: Engine,
        unsafe_head_tx: Option<watch::Sender<L2BlockInfo>>,
        inbound_request_rx: mpsc::Receiver<EngineActorRequest>,
        block_sink: Arc<dyn ImportedBlockSink>,
    ) -> Self {
        Self {
            block_sink,
            client,
            derivation_client,
            el_sync_complete: false,
            el_sync_stall_timeout: DEFAULT_EL_SYNC_STALL_TIMEOUT,
            engine,
            inbound_request_rx,
            last_engine_safe_head_sent: L2BlockInfo::default(),
            last_skipped_task_error: None,
            reset_pending: false,
            reset_waiters: Vec::new(),
            retry_at: Instant::now(),
            retry_delay: INITIAL_RETRY_DELAY,
            rollup: config,
            unsafe_head_tx,
            waiting_retry_at: Instant::now(),
        }
    }

    /// Bounds how long a consolidation waits for an execution layer whose sync reports no
    /// progress.
    pub const fn with_el_sync_stall_timeout(mut self, el_sync_stall_timeout: Duration) -> Self {
        self.el_sync_stall_timeout = el_sync_stall_timeout;
        self
    }

    /// Logs a skipped task at `warn!` when its error differs from the last skip, and at `debug!`
    /// while the identical error repeats, so a target resubmitted on every poll warns once.
    fn log_skipped_task(&mut self, err: &EngineTaskErrors) {
        let message = err.to_string();
        if self.last_skipped_task_error.as_ref() == Some(&message) {
            debug!(target: "engine", %err, "Skipped the same engine task again");
            return;
        }
        warn!(target: "engine", %err, "Skipped an engine task that no longer applies");
        self.last_skipped_task_error = Some(message);
    }

    fn defer_retry(&mut self) {
        self.retry_at = Instant::now() + self.retry_delay;
        self.retry_delay = (self.retry_delay * 2).min(MAX_RETRY_DELAY);
    }

    /// Schedules the retry of a task waiting for the execution layer, leaving the failure backoff
    /// as it is.
    fn defer_waiting_retry(&mut self) {
        self.waiting_retry_at = Instant::now() + WAIT_RETRY_INTERVAL;
    }

    /// Which tasks a drain may run now: the waiting task only once its retry is due.
    fn drain_scope(&self, now: Instant) -> DrainScope {
        if now >= self.waiting_retry_at {
            DrainScope::AllTasks
        } else {
            DrainScope::AheadOfWaitingTask
        }
    }

    /// When the actor next has work to retry without a new request, if any: the failure backoff
    /// while a reset or a runnable task is pending, and otherwise the waiting task's retry.
    fn next_retry_at(&self) -> Option<Instant> {
        if self.reset_pending || self.engine.has_work_ahead_of_waiting_task() {
            return Some(self.retry_at);
        }
        self.engine.has_waiting_task().then(|| self.retry_at.max(self.waiting_retry_at))
    }

    fn begin_reset(&mut self) {
        if !self.reset_pending {
            self.engine.clear();
            self.last_skipped_task_error = None;
            self.reset_pending = true;
            self.retry_at = Instant::now();
        }
    }

    async fn attempt_reset(&mut self) -> Result<(), EngineError> {
        match self.reset().await {
            Ok(()) => {
                self.reset_pending = false;
                self.retry_delay = INITIAL_RETRY_DELAY;
                if !self.el_sync_complete && self.engine.state().el_sync_finished {
                    self.mark_el_sync_complete_and_notify_derivation_actor().await?;
                }
            }
            Err(EngineError::EngineReset(err)) if err.is_temporary() => {
                warn!(target: "engine", ?err, "Reset dependency unavailable; retaining reset");
                self.defer_retry();
            }
            Err(err) => return Err(err),
        }
        Ok(())
    }

    /// Resets the inner [`Engine`] and propagates the reset to the derivation actor.
    async fn reset(&mut self) -> Result<(), EngineError> {
        // Reset the engine.
        let l2_safe_head = self.engine.reset(self.client.clone(), self.rollup.clone()).await?;
        // The sequencer must observe the same reset head as RPC readers before its reset
        // acknowledgement can release a queued start command.
        self.publish_unsafe_head();

        // Derivation may be awaiting this reply with its inbound queue full of L1 updates.
        // Release those callers before awaiting delivery to that same queue. They transition
        // to AwaitingSignal before consuming the queued reset signal below.
        for waiter in self.reset_waiters.drain(..) {
            let _ = waiter.try_send(Ok(()));
        }

        // Signal the derivation actor to reset.
        let signal = Signal::Reset(ResetSignal { l2_safe_head });
        match self.derivation_client.send_signal(signal).await {
            Ok(_) => info!(target: "engine", "Sent reset signal to derivation actor"),
            Err(err) => {
                error!(target: "engine", ?err, "Failed to send reset signal to the derivation actor");
                return Err(EngineError::ChannelClosed);
            }
        }

        // The reset signal includes the recovered safe head. Derivation adopts it and resumes
        // while handling that message, so an intervening L1 update cannot overtake confirmation.
        self.last_engine_safe_head_sent = l2_safe_head;

        Ok(())
    }

    /// Publishes the current unsafe head to the sequencer without waiting on derivation.
    fn publish_unsafe_head(&self) {
        if let Some(unsafe_head_tx) = self.unsafe_head_tx.as_ref() {
            unsafe_head_tx.send_if_modified(|current_head| {
                let new_head = self.engine.state().sync_state.unsafe_head();
                if *current_head == new_head {
                    false
                } else {
                    *current_head = new_head;
                    true
                }
            });
        }
    }

    /// Drains the inner [`Engine`] task queue and attempts to update the safe head.
    async fn drain(&mut self, scope: DrainScope) -> Result<(), EngineError> {
        match self.engine.drain(scope).await {
            Ok(_) => {
                trace!(target: "engine", ?scope, "[ENGINE] tasks drained");
                self.retry_delay = INITIAL_RETRY_DELAY;
            }
            Err(err) => {
                match err.severity() {
                    EngineTaskErrorSeverity::Critical => {
                        error!(target: "engine", ?err, "Critical error draining engine tasks");
                        return Err(err.into());
                    }
                    EngineTaskErrorSeverity::Reset => {
                        warn!(target: "engine", ?err, "Received reset request");
                        self.begin_reset();
                        return Ok(());
                    }
                    EngineTaskErrorSeverity::Flush => {
                        // This error is encountered when the engine API rejects a derived
                        // payload: its attributes at forkchoiceUpdated, or the payload at
                        // newPayload. Post-holocene, the payload is replaced by a
                        // "deposits-only" block. At the same time, the channel and any
                        // remaining buffered batches are flushed.
                        warn!(target: "engine", ?err, "Invalid payload, Flushing derivation pipeline.");
                        match self.derivation_client.send_signal(Signal::FlushChannel).await {
                            Ok(_) => {
                                debug!(target: "engine", "Sent flush signal to derivation actor")
                            }
                            Err(err) => {
                                error!(target: "engine", ?err, "Failed to send flush signal to the derivation actor.");
                                return Err(EngineError::ChannelClosed);
                            }
                        }
                    }
                    EngineTaskErrorSeverity::Temporary => {
                        trace!(target: "engine", ?err, "Temporary error draining engine tasks");
                        self.defer_retry();
                        return Ok(());
                    }
                    EngineTaskErrorSeverity::Skip => self.log_skipped_task(&err),
                    // A waiting task changed nothing, but tasks that ran before it in this drain
                    // did, so their safe head and sync completion still reach derivation below.
                    // They all succeeded, so the failure backoff starts over too.
                    EngineTaskErrorSeverity::Wait => {
                        trace!(target: "engine", %err, "Engine task waits before its retry");
                        self.retry_delay = INITIAL_RETRY_DELAY;
                        self.defer_waiting_retry();
                    }
                }
            }
        }

        self.send_derivation_actor_safe_head_if_updated().await?;

        if !self.el_sync_complete && self.engine.state().el_sync_finished {
            if self.engine.state().sync_state.established_finalized_head().is_none() {
                self.begin_reset();
            } else {
                self.mark_el_sync_complete_and_notify_derivation_actor().await?;
            }
        }

        Ok(())
    }

    async fn mark_el_sync_complete_and_notify_derivation_actor(
        &mut self,
    ) -> Result<(), EngineError> {
        self.el_sync_complete = true;

        self.derivation_client
            .notify_sync_completed(self.engine.state().sync_state)
            .await
            .map(|_| Ok(()))
            .map_err(|e| {
                error!(target: "engine", ?e, "Failed to notify sync completed");
                EngineError::ChannelClosed
            })?
    }

    /// Sends the [`crate::DerivationActor`] the local-safe head if it changed since the last send.
    ///
    /// Derivation continues from, and records, the head that L1 data made safe. That is the
    /// local-safe label; the safe label can lag it by cross-chain verification.
    async fn send_derivation_actor_safe_head_if_updated(&mut self) -> Result<(), EngineError> {
        let local_safe_head = self.engine.state().sync_state.local_safe_head();
        if local_safe_head == self.last_engine_safe_head_sent {
            debug!(target: "engine", safe_head = ?local_safe_head, "Local-safe head unchanged");
            return Ok(());
        }

        self.derivation_client.send_new_engine_safe_head(local_safe_head).await.map_err(|e| {
            error!(target: "engine", ?e, "Failed to send new engine safe head");
            EngineError::ChannelClosed
        })?;

        info!(target: "engine", safe_head = ?local_safe_head, "Sent local-safe head to derivation");
        self.last_engine_safe_head_sent = local_safe_head;

        Ok(())
    }
}

#[async_trait]
impl<DerivationClient> NodeActor for EngineActor<DerivationClient>
where
    DerivationClient: EngineDerivationClient + 'static,
{
    type Error = EngineError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        let now = Instant::now();
        if now >= self.retry_at {
            if self.reset_pending {
                self.attempt_reset().await?;
            } else {
                // Work that outranks a task waiting for the execution layer to sync, such as an
                // unsafe import, runs as it arrives; the waiting task runs once its retry is due.
                self.drain(self.drain_scope(now)).await?;
            }
        }

        // If the unsafe head has updated, propagate it to the outbound channels.
        self.publish_unsafe_head();

        // Wait for the next processing request.
        // Retry without needing a new inbound message, while still accepting requests during
        // backoff. Bound queued work so a prolonged outage cannot grow memory without limit.
        let next_retry_at = self.next_retry_at();
        let request = tokio::select! {
            request = self.inbound_request_rx.recv(),
                if self.engine.len() < 1024 && self.reset_waiters.len() < 1024 => {
                request.ok_or(EngineError::ChannelClosed)?
            }
            _ = time::sleep_until(next_retry_at.unwrap_or(self.retry_at)),
                if next_retry_at.is_some() => return Ok(()),
        };

        match request {
            EngineActorRequest::Build(build_request) => {
                let BuildRequest { attributes, result_tx } = *build_request;
                let task = EngineTask::Build(Box::new(BuildTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    attributes,
                    result_tx,
                )));
                self.engine.enqueue(task);
            }
            EngineActorRequest::ProcessSafeL2Signal(safe_signal) => {
                let task = EngineTask::Consolidate(Box::new(ConsolidateTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    safe_signal,
                    Arc::clone(&self.block_sink),
                    self.el_sync_stall_timeout,
                )));
                self.engine.enqueue(task);
            }
            EngineActorRequest::ProcessFinalizedL2Block(finalized_l2_block_id) => {
                // Finalize the L2 block identified by the provided [`FinalizeBlockId`].
                let task = EngineTask::Finalize(Box::new(FinalizeTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    *finalized_l2_block_id,
                )));
                self.engine.enqueue(task);
            }
            EngineActorRequest::ProcessUnsafeL2Block(envelope) => {
                let task = EngineTask::Insert(Box::new(InsertTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    *envelope,
                    false, /* The payload is not derived in this case. This is an unsafe
                            * block. */
                    Arc::clone(&self.block_sink),
                )));
                self.engine.enqueue(task);
            }
            EngineActorRequest::Reset(reset_request) => {
                warn!(target: "engine", "Received reset request");

                self.begin_reset();
                self.reset_waiters.push(reset_request.result_tx);
            }
            EngineActorRequest::Canonicalize(request) => {
                let CanonicalizeRequest { payload, parent, result_tx } = *request;
                self.engine.enqueue(EngineTask::Canonicalize(Box::new(CanonicalizeTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    payload,
                    parent,
                    result_tx,
                    self.unsafe_head_tx.clone(),
                    Arc::clone(&self.block_sink),
                ))));
            }
            EngineActorRequest::Seal(seal_request) => {
                let SealRequest { payload_id, attributes, result_tx } = *seal_request;
                let task = EngineTask::Seal(Box::new(SealTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    payload_id,
                    attributes,
                    Some(result_tx),
                )));
                self.engine.enqueue(task);
            }
        }

        Ok(())
    }
}

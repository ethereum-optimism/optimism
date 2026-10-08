use crate::{BuildRequest, EngineDerivationClient, EngineError, NodeActor, SealRequest};
use async_trait::async_trait;
use kona_derive::{ResetSignal, Signal};
use kona_engine::{
    BuildSealCoupling, BuildTask, ConsolidateTask, Engine, EngineActorRequest, EngineClient,
    EngineTask, EngineTaskError, EngineTaskErrorSeverity, FinalizeTask, ImportedBlockSink,
    InsertTask, SealTask,
};
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use std::sync::Arc;
use tokio::{
    sync::{mpsc, watch},
    time::{self, Duration, Instant},
};

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
    /// The last safe head update sent.
    last_safe_head_sent: L2BlockInfo,
    /// A channel to use to relay the current unsafe head.
    /// ## Note
    /// This is `Some` when the node is in sequencer mode, and `None` when the node is in validator
    /// mode.
    unsafe_head_tx: Option<watch::Sender<L2BlockInfo>>,

    /// The [`RollupConfig`] used to build tasks.
    rollup: Arc<RollupConfig>,
    /// An [`EngineClient`] used for creating engine tasks.
    client: Arc<EngineClient>,
    /// The [`Engine`] task queue.
    engine: Engine,
    /// The inbound request channel.
    inbound_request_rx: mpsc::Receiver<EngineActorRequest>,
    /// Where to hand every imported block, so the derivation providers can read it locally
    /// instead of fetching it back from the execution layer.
    block_sink: Arc<dyn ImportedBlockSink>,
    /// A reset is retained until successful; RPC failure must not lose the request.
    reset_pending: bool,
    /// Callers waiting for the retained reset to finish.
    reset_waiters: Vec<mpsc::Sender<crate::EngineClientResult<()>>>,
    /// Earliest next attempt after a dependency failure.
    retry_at: Instant,
    /// Exponentially increasing retry delay, capped to keep recovery responsive.
    retry_delay: Duration,
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
            client,
            derivation_client,
            el_sync_complete: false,
            engine,
            last_safe_head_sent: L2BlockInfo::default(),
            rollup: config,
            unsafe_head_tx,
            inbound_request_rx,
            block_sink,
            reset_pending: false,
            reset_waiters: Vec::new(),
            retry_at: Instant::now(),
            retry_delay: Duration::from_millis(100),
        }
    }

    fn defer_retry(&mut self) {
        self.retry_at = Instant::now() + self.retry_delay;
        self.retry_delay = (self.retry_delay * 2).min(Duration::from_secs(5));
    }

    fn begin_reset(&mut self) {
        if !self.reset_pending {
            self.engine.clear();
            self.reset_pending = true;
            self.retry_at = Instant::now();
        }
    }

    async fn attempt_reset(&mut self) -> Result<(), EngineError> {
        match self.reset().await {
            Ok(()) => {
                self.reset_pending = false;
                self.retry_delay = Duration::from_millis(100);
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

        self.send_derivation_actor_safe_head_if_updated().await?;

        Ok(())
    }

    /// Drains the inner [`Engine`] task queue and attempts to update the safe head.
    async fn drain(&mut self) -> Result<(), EngineError> {
        match self.engine.drain().await {
            Ok(_) => {
                trace!(target: "engine", "[ENGINE] tasks drained");
                self.retry_delay = Duration::from_millis(100);
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
                        // This error is encountered when the payload is marked INVALID
                        // by the engine api. Post-holocene, the payload is replaced by
                        // a "deposits-only" block and re-executed. At the same time,
                        // the channel and any remaining buffered batches are flushed.
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
                }
            }
        }

        self.send_derivation_actor_safe_head_if_updated().await?;

        if !self.el_sync_complete && self.engine.state().el_sync_finished {
            if self.engine.state().sync_state.finalized_head() == L2BlockInfo::default() {
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
            .notify_sync_completed(self.engine.state().sync_state.safe_head())
            .await
            .map(|_| Ok(()))
            .map_err(|e| {
                error!(target: "engine", ?e, "Failed to notify sync completed");
                EngineError::ChannelClosed
            })?
    }

    /// Attempts to send the [`crate::DerivationActor`] the safe head if updated.
    async fn send_derivation_actor_safe_head_if_updated(&mut self) -> Result<(), EngineError> {
        let engine_safe_head = self.engine.state().sync_state.safe_head();
        if engine_safe_head == self.last_safe_head_sent {
            info!(target: "engine", safe_head = ?engine_safe_head, "Safe head unchanged");
            // This was already sent, so do not send it.
            return Ok(());
        }

        self.derivation_client.send_new_engine_safe_head(engine_safe_head).await.map_err(|e| {
            error!(target: "engine", ?e, "Failed to send new engine safe head");
            EngineError::ChannelClosed
        })?;

        info!(target: "engine", safe_head = ?engine_safe_head, "Attempted L2 Safe Head Update");
        self.last_safe_head_sent = engine_safe_head;

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
        if Instant::now() >= self.retry_at {
            if self.reset_pending {
                self.attempt_reset().await?;
            } else {
                self.drain().await?;
            }
        }

        // If the unsafe head has updated, propagate it to the outbound channels.
        if let Some(unsafe_head_tx) = self.unsafe_head_tx.as_ref() {
            unsafe_head_tx.send_if_modified(|val| {
                let new_head = self.engine.state().sync_state.unsafe_head();
                (*val != new_head).then(|| *val = new_head).is_some()
            });
        }

        // Wait for the next processing request.
        // Retry without needing a new inbound message, while still accepting requests during
        // backoff. Bound queued work so a prolonged outage cannot grow memory without limit.
        let request = tokio::select! {
            request = self.inbound_request_rx.recv(),
                if self.engine.len() < 1024 && self.reset_waiters.len() < 1024 => {
                request.ok_or(EngineError::ChannelClosed)?
            }
            _ = time::sleep_until(self.retry_at),
                if self.reset_pending || !self.engine.is_empty() => return Ok(()),
        };

        match request {
            EngineActorRequest::Build(build_request) => {
                let BuildRequest { attributes, result_tx } = *build_request;
                let task = EngineTask::Build(Box::new(BuildTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    attributes,
                    Some(result_tx),
                )));
                self.engine.enqueue(task);
            }
            EngineActorRequest::ProcessSafeL2Signal(safe_signal) => {
                let task = EngineTask::Consolidate(Box::new(ConsolidateTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    safe_signal,
                    Arc::clone(&self.block_sink),
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
            EngineActorRequest::Seal(seal_request) => {
                let SealRequest { payload_id, attributes, result_tx } = *seal_request;
                let task = EngineTask::Seal(Box::new(SealTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    payload_id,
                    attributes,
                    // The payload is not derived in this case.
                    false,
                    BuildSealCoupling::Detached,
                    Some(result_tx),
                    Arc::clone(&self.block_sink),
                )));
                self.engine.enqueue(task);
            }
        }

        Ok(())
    }
}

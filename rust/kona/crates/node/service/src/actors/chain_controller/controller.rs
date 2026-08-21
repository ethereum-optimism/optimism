use crate::{
    BuildRequest, ChainControllerClientError, ChainControllerDerivationClient,
    ChainControllerError, NodeActor, ResetRequest, SealRequest,
};
use async_trait::async_trait;
use kona_derive::{ResetSignal, Signal};
use kona_engine::{
    BuildTask, ConsolidateInput, ConsolidateTask, Engine, EngineClient, EngineTask,
    EngineTaskError, EngineTaskErrorSeverity, FinalizeBlockId, FinalizeTask, InsertTask, SealTask,
};
use kona_genesis::RollupConfig;
use kona_protocol::L2BlockInfo;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::sync::Arc;
use tokio::sync::{mpsc, watch};

/// A request handled by the [`ChainController`].
#[derive(Debug)]
pub enum ChainControllerRequest {
    /// Request to start building a block.
    Build(Box<BuildRequest>),
    /// Request to process a local-safe signal, which can be derived attributes or delegated block
    /// info.
    ProcessLocalSafeL2Signal(ConsolidateInput),
    /// Request to process the finalized L2 block identified by the provided [`FinalizeBlockId`].
    ProcessFinalizedL2Block(Box<FinalizeBlockId>),
    /// Request to process a received unsafe L2 block.
    ProcessUnsafeL2Block(Box<OpExecutionPayloadEnvelope>),
    /// Request to reset the forkchoice.
    Reset(Box<ResetRequest>),
    /// Request to seal a block.
    Seal(Box<SealRequest>),
}

/// Owns this chain's head state: the sole writer of the [`Engine`]'s heads and the only actor that
/// drives the execution layer's Engine API.
///
/// Every state-mutating input — derived attributes from derivation, unsafe payloads from gossip,
/// finalization, resets, and sequencer block building — arrives on one inbound request channel as a
/// [`ChainControllerRequest`] and is ordered through the [`Engine`] task queue by the [`Ord`]
/// implementation of [`EngineTask`]. In the other direction the controller mediates every
/// derivation-bound signal (reset, channel flush, sync-completed, and the local-safe lockstep
/// confirmation) and owns reset initiation, so derivation never reaches the execution layer itself.
///
/// Read-only queries are served by its peer [`ChainControllerRpcActor`], which shares a watch over
/// the engine state and queue length but holds a read-only client.
///
/// [`ChainControllerRpcActor`]: crate::ChainControllerRpcActor
#[derive(Debug)]
pub struct ChainController<EngineClient_, DerivationClient>
where
    EngineClient_: EngineClient,
    DerivationClient: ChainControllerDerivationClient,
{
    /// The client used to send messages to the [`crate::DerivationActor`].
    derivation_client: DerivationClient,
    /// Whether the EL sync is complete. This should only ever go from false to true.
    el_sync_complete: bool,
    /// The last local-safe head update sent to the derivation actor.
    last_local_safe_head_sent: L2BlockInfo,
    /// A channel to use to relay the current unsafe head.
    /// ## Note
    /// This is `Some` when the node is in sequencer mode, and `None` when the node is in validator
    /// mode.
    unsafe_head_tx: Option<watch::Sender<L2BlockInfo>>,

    /// The [`RollupConfig`] used to build tasks.
    rollup: Arc<RollupConfig>,
    /// An [`EngineClient`] used for creating engine tasks.
    client: Arc<EngineClient_>,
    /// The [`Engine`] task queue.
    engine: Engine<EngineClient_>,
    /// The inbound request channel.
    inbound_request_rx: mpsc::Receiver<ChainControllerRequest>,
}

impl<EngineClient_, DerivationClient> ChainController<EngineClient_, DerivationClient>
where
    EngineClient_: EngineClient + 'static,
    DerivationClient: ChainControllerDerivationClient + 'static,
{
    /// Constructs a new [`ChainController`] from the params.
    pub fn new(
        client: Arc<EngineClient_>,
        config: Arc<RollupConfig>,
        derivation_client: DerivationClient,
        engine: Engine<EngineClient_>,
        unsafe_head_tx: Option<watch::Sender<L2BlockInfo>>,
        inbound_request_rx: mpsc::Receiver<ChainControllerRequest>,
    ) -> Self {
        Self {
            client,
            derivation_client,
            el_sync_complete: false,
            engine,
            last_local_safe_head_sent: L2BlockInfo::default(),
            rollup: config,
            unsafe_head_tx,
            inbound_request_rx,
        }
    }

    /// Resets the inner [`Engine`] and propagates the reset to the derivation actor.
    async fn reset(&mut self) -> Result<(), ChainControllerError> {
        // Reset the engine. Resets re-derive local safety only; the cross-safe head is not
        // touched.
        let l2_safe_head = self.engine.reset(self.client.clone(), self.rollup.clone()).await?;

        // Signal the derivation actor to reset.
        let signal = Signal::Reset(ResetSignal { l2_safe_head });
        match self.derivation_client.send_signal(signal).await {
            Ok(_) => info!(target: "engine", "Sent reset signal to derivation actor"),
            Err(err) => {
                error!(target: "engine", ?err, "Failed to send reset signal to the derivation actor");
                return Err(ChainControllerError::ChannelClosed);
            }
        }

        self.send_derivation_actor_local_safe_head_if_updated().await?;

        Ok(())
    }

    /// Drains the inner [`Engine`] task queue and attempts to update the safe head.
    async fn drain(&mut self) -> Result<(), ChainControllerError> {
        match self.engine.drain().await {
            Ok(_) => {
                trace!(target: "engine", "[ENGINE] tasks drained");
            }
            Err(err) => {
                match err.severity() {
                    EngineTaskErrorSeverity::Critical => {
                        error!(target: "engine", ?err, "Critical error draining engine tasks");
                        return Err(err.into());
                    }
                    EngineTaskErrorSeverity::Reset => {
                        warn!(target: "engine", ?err, "Received reset request");
                        self.reset().await?;
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
                                return Err(ChainControllerError::ChannelClosed);
                            }
                        }
                    }
                    EngineTaskErrorSeverity::Temporary => {
                        trace!(target: "engine", ?err, "Temporary error draining engine tasks");
                    }
                }
            }
        }

        self.send_derivation_actor_local_safe_head_if_updated().await?;

        if !self.el_sync_complete && self.engine.state().el_sync_finished {
            self.mark_el_sync_complete_and_notify_derivation_actor().await?;
        }

        Ok(())
    }

    async fn mark_el_sync_complete_and_notify_derivation_actor(
        &mut self,
    ) -> Result<(), ChainControllerError> {
        self.el_sync_complete = true;

        // Reset the engine if the sync state does not already know about a finalized block.
        if self.engine.state().sync_state.finalized_head() == L2BlockInfo::default() {
            // If the sync status is finished, we can reset the engine and start derivation.
            info!(target: "engine", "Performing initial engine reset");
            self.reset().await?;
        } else {
            info!(target: "engine", "finalized head is not default, so not resetting");
        }

        self.derivation_client
            .notify_sync_completed(self.engine.state().sync_state.local_safe_head())
            .await
            .map(|_| Ok(()))
            .map_err(|e| {
                error!(target: "engine", ?e, "Failed to notify sync completed");
                ChainControllerError::ChannelClosed
            })?
    }

    /// Attempts to send the [`crate::DerivationActor`] the local-safe head if updated.
    ///
    /// This is the depth-1 lockstep confirmation that unblocks derivation's next set of payload
    /// attributes, so it must be driven by local-safe. Driving it from cross-safe deadlocks under
    /// interop: derivation would wait on a promotion that waits on every chain's derivation.
    async fn send_derivation_actor_local_safe_head_if_updated(
        &mut self,
    ) -> Result<(), ChainControllerError> {
        let engine_local_safe_head = self.engine.state().sync_state.local_safe_head();
        if engine_local_safe_head == self.last_local_safe_head_sent {
            info!(target: "engine", local_safe_head = ?engine_local_safe_head, "Local-safe head unchanged");
            // This was already sent, so do not send it.
            return Ok(());
        }

        self.derivation_client
            .send_new_engine_local_safe_head(engine_local_safe_head)
            .await
            .map_err(|e| {
                error!(target: "engine", ?e, "Failed to send new engine local-safe head");
                ChainControllerError::ChannelClosed
            })?;

        info!(target: "engine", local_safe_head = ?engine_local_safe_head, "Attempted L2 local-safe head update");
        self.last_local_safe_head_sent = engine_local_safe_head;

        Ok(())
    }
}

#[async_trait]
impl<EngineClient_, DerivationClient> NodeActor for ChainController<EngineClient_, DerivationClient>
where
    EngineClient_: EngineClient + 'static,
    DerivationClient: ChainControllerDerivationClient + 'static,
{
    type Error = ChainControllerError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        // Attempt to drain all outstanding tasks from the engine queue before adding new ones.
        self.drain()
            .await
            .inspect_err(|err| error!(target: "engine", ?err, "Failed to drain engine tasks"))?;

        // If the unsafe head has updated, propagate it to the outbound channels.
        if let Some(unsafe_head_tx) = self.unsafe_head_tx.as_ref() {
            unsafe_head_tx.send_if_modified(|val| {
                let new_head = self.engine.state().sync_state.unsafe_head();
                (*val != new_head).then(|| *val = new_head).is_some()
            });
        }

        // Wait for the next processing request.
        let request = self.inbound_request_rx.recv().await.ok_or_else(|| {
            error!(target: "engine", "Engine processing request receiver closed unexpectedly");
            ChainControllerError::ChannelClosed
        })?;

        match request {
            ChainControllerRequest::Build(build_request) => {
                let BuildRequest { attributes, result_tx } = *build_request;
                let task = EngineTask::Build(Box::new(BuildTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    attributes,
                    Some(result_tx),
                )));
                self.engine.enqueue(task);
            }
            ChainControllerRequest::ProcessLocalSafeL2Signal(local_safe_signal) => {
                let task = EngineTask::Consolidate(Box::new(ConsolidateTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    local_safe_signal,
                )));
                self.engine.enqueue(task);
            }
            ChainControllerRequest::ProcessFinalizedL2Block(finalized_l2_block_id) => {
                // Finalize the L2 block identified by the provided [`FinalizeBlockId`].
                let task = EngineTask::Finalize(Box::new(FinalizeTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    *finalized_l2_block_id,
                )));
                self.engine.enqueue(task);
            }
            ChainControllerRequest::ProcessUnsafeL2Block(envelope) => {
                let task = EngineTask::Insert(Box::new(InsertTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    *envelope,
                    false, /* The payload is not derived in this case. This is an unsafe
                            * block. */
                )));
                self.engine.enqueue(task);
            }
            ChainControllerRequest::Reset(reset_request) => {
                warn!(target: "engine", "Received reset request");

                let reset_res = self.reset().await;

                // Send the result.
                let response_payload = reset_res
                    .as_ref()
                    .map(|_| ())
                    .map_err(|e| ChainControllerClientError::ResetForkchoiceError(e.to_string()));
                if reset_request.result_tx.send(response_payload).await.is_err() {
                    warn!(target: "engine", "Sending reset response failed");
                    // If there was an error and we couldn't notify the caller to handle it,
                    // return the error.
                    reset_res?;
                }
            }
            ChainControllerRequest::Seal(seal_request) => {
                let SealRequest { payload_id, attributes, result_tx } = *seal_request;
                let task = EngineTask::Seal(Box::new(SealTask::new(
                    self.client.clone(),
                    self.rollup.clone(),
                    payload_id,
                    attributes,
                    // The payload is not derived in this case.
                    false,
                    Some(result_tx),
                )));
                self.engine.enqueue(task);
            }
        }

        Ok(())
    }
}

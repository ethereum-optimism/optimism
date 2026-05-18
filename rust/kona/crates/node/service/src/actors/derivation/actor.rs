//! [`NodeActor`] implementation for the derivation sub-routine.
//!
//! Phase 4 of the pure-derivation migration replaced the inner `OnlinePipeline`
//! with [`NodeDeriver`], an async driver around the IO-free `kona_derive::Deriver`.
//! The actor's job here is purely cross-actor coordination: it relays the L1
//! head + safe-head + flush/reset signals coming from neighbouring actors into
//! `NodeDeriver` method calls, and it forwards built attributes back to the
//! engine actor.

use crate::{
    CancellableContext, DerivationActorRequest, DerivationEngineClient, DerivationState,
    DerivationStateMachine, DerivationStateTransitionError, DerivationStateUpdate, Metrics,
    NodeActor,
    actors::derivation::{L2Finalizer, NodeDeriver, NodeDeriverError, NodeStep},
};
use async_trait::async_trait;
use kona_derive::Signal;
use kona_protocol::OpAttributesWithParent;
use thiserror::Error;
use tokio::{select, sync::mpsc};
use tokio_util::sync::{CancellationToken, WaitForCancellationFuture};

/// The [`NodeActor`] for the derivation sub-routine.
///
/// Receives messages from neighbour actors and drives [`NodeDeriver`] forward
/// to produce the next safe payload. Forwards built attributes to the engine
/// actor through `engine_client`.
#[derive(Debug)]
pub struct DerivationActor<DerivationEngineClient_> {
    /// The cancellation token, shared between all tasks.
    cancellation_token: CancellationToken,
    /// The channel on which all inbound requests are received by the [`DerivationActor`].
    inbound_request_rx: mpsc::Receiver<DerivationActorRequest>,
    /// The Engine client used to interact with the engine.
    engine_client: DerivationEngineClient_,

    /// The async wrapper around the pure derivation engine.
    deriver: NodeDeriver,
    /// The state machine controlling when derivation can occur.
    derivation_state_machine: DerivationStateMachine,
    /// The [`L2Finalizer`] tracks derived L2 blocks awaiting finalization.
    pub(crate) finalizer: L2Finalizer,
}

impl<DerivationEngineClient_> CancellableContext for DerivationActor<DerivationEngineClient_>
where
    DerivationEngineClient_: DerivationEngineClient,
{
    fn cancelled(&self) -> WaitForCancellationFuture<'_> {
        self.cancellation_token.cancelled()
    }
}

impl<DerivationEngineClient_> DerivationActor<DerivationEngineClient_>
where
    DerivationEngineClient_: DerivationEngineClient,
{
    /// Creates a new instance of the [`DerivationActor`].
    pub fn new(
        engine_client: DerivationEngineClient_,
        cancellation_token: CancellationToken,
        inbound_request_rx: mpsc::Receiver<DerivationActorRequest>,
        deriver: NodeDeriver,
    ) -> Self {
        Self {
            cancellation_token,
            deriver,
            inbound_request_rx,
            engine_client,
            derivation_state_machine: DerivationStateMachine::default(),
            finalizer: L2Finalizer::default(),
        }
    }

    /// Handles a [`Signal`] received over the derivation request channel.
    ///
    /// The engine actor sends [`Signal::Reset`] and [`Signal::FlushChannel`]
    /// here when an L2 reorg or invalid-payload condition is detected. The
    /// pure deriver collapses both into the same shape — re-anchor at the
    /// safe head — so this method's only fanout is logging.
    async fn handle_signal(&mut self, signal: Signal) -> Result<(), DerivationError> {
        match signal {
            Signal::Reset(reset) => {
                info!(target: "derivation", safe_head = ?reset.l2_safe_head, "Handling Signal::Reset");
                // Clear the finalization queue on reset.
                self.finalizer.clear();
                self.deriver.reset(reset.l2_safe_head).await.map_err(DerivationError::Deriver)?;
            }
            Signal::FlushChannel => {
                let safe_head = self.derivation_state_machine.last_confirmed_safe_head();
                info!(target: "derivation", safe_head = ?safe_head, "Handling Signal::FlushChannel");
                self.deriver.flush_channel(safe_head).await.map_err(DerivationError::Deriver)?;
            }
            Signal::Activation(_) => {
                // Holocene+ subsumes the soft-reset activation path. The pure deriver
                // never produces a `ResetError::HoloceneActivation`, and the engine
                // actor never sends `Signal::Activation` on this code path. If it
                // ever arrives anyway, treat it as a no-op rather than reaching for
                // a behavior that no longer exists.
                debug!(target: "derivation", "Ignoring Signal::Activation (Holocene+ subsumes soft reset)");
            }
            Signal::ProvideBlock(_) => {
                // The async indexed-traversal pipeline uses `ProvideBlock` to push L1 blocks
                // through the stage chain. With the pure deriver, the actor fetches L1 data
                // itself in `NodeDeriver::step` based on the L1 head height; there's no
                // out-of-band block-push surface that produces this signal here.
                debug!(
                    target: "derivation",
                    "Ignoring Signal::ProvideBlock (handled internally by NodeDeriver)"
                );
            }
        }
        Ok(())
    }

    /// Drive [`NodeDeriver`] forward until it either produces attributes or
    /// signals that it needs more L1 data (which the caller will reschedule
    /// via the next [`DerivationActorRequest::ProcessL1HeadUpdateRequest`]).
    async fn produce_next_attributes(&mut self) -> Result<OpAttributesWithParent, DerivationError> {
        let safe_head = self.derivation_state_machine.last_confirmed_safe_head();
        match self.deriver.step(safe_head).await {
            NodeStep::Attributes(attrs) => Ok(*attrs),
            NodeStep::Yield => Err(DerivationError::Yield),
            NodeStep::NeedsReset(err) => {
                error!(target: "derivation", ?err, "Derivation reported critical error — requesting reset");
                kona_macros::inc!(counter, Metrics::DERIVATION_CRITICAL_ERROR);
                // Tell the engine actor to reset; on success it'll send us a `Signal::Reset`
                // back through the inbound channel.
                self.engine_client.reset_engine_forkchoice().await.map_err(|e| {
                    error!(target: "derivation", ?e, "Failed to send reset request");
                    DerivationError::Sender(Box::new(e))
                })?;
                Err(DerivationError::Yield)
            }
        }
    }

    async fn handle_derivation_actor_request(
        &mut self,
        request_type: DerivationActorRequest,
    ) -> Result<(), DerivationError> {
        match request_type {
            DerivationActorRequest::ProcessEngineSignalRequest(signal) => {
                self.handle_signal(*signal).await?;
            }
            DerivationActorRequest::ProcessFinalizedL1Block(finalized_l1_block) => {
                // Attempt to finalize the block. If successful, notify engine.
                if let Some(l2_block_number) = self.finalizer.try_finalize_next(*finalized_l1_block)
                {
                    self.engine_client
                        .send_finalized_l2_block(l2_block_number)
                        .await
                        .map_err(|e| DerivationError::Sender(Box::new(e)))?;
                }
            }
            DerivationActorRequest::ProcessL1HeadUpdateRequest(l1_head) => {
                info!(target: "derivation", l1_head = ?*l1_head, "Processing l1 head update");

                self.deriver.set_l1_head(*l1_head);
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
impl<DerivationEngineClient_> NodeActor for DerivationActor<DerivationEngineClient_>
where
    DerivationEngineClient_: DerivationEngineClient + 'static,
{
    type Error = DerivationError;
    type StartData = ();

    async fn start(mut self, _: Self::StartData) -> Result<(), Self::Error> {
        info!(target: "derivation", "Starting derivation");
        loop {
            select! {
                biased;

                _ = self.cancellation_token.cancelled() => {
                    info!(
                        target: "derivation",
                        "Received shutdown signal. Exiting derivation task."
                    );
                    return Ok(());
                }
                req = self.inbound_request_rx.recv() => {
                    let Some(request_type) = req else {
                        error!(target: "derivation", "DerivationActor inbound request receiver closed unexpectedly");
                        self.cancellation_token.cancel();
                        return Err(DerivationError::RequestReceiveFailed);
                    };

                    self.handle_derivation_actor_request(request_type).await?;
                }
            }
        }
    }
}

/// An error from the [`DerivationActor`].
#[derive(Error, Debug)]
pub enum DerivationError {
    /// An error originating from the derivation engine wrapper.
    #[error(transparent)]
    Deriver(#[from] NodeDeriverError),
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

//! The long-running [`SequencerActor`] service.

use crate::{
    Conductor, SequencerEngineClient, UnsafePayloadGossipClient,
    actors::sequencer::{
        error::SequencerActorError,
        handle::{SequencerCommand, SequencerHandle},
        origin_selector::OriginSelector,
        workflow::{BlockSequenceOutcome, SequencingWorkflow},
    },
};
use kona_derive::AttributesBuilder;
use kona_genesis::RollupConfig;
use std::sync::Arc;
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;

/// Result of checking control requests at a block boundary.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(super) enum BoundaryAction {
    /// Continue draining queued control requests at the current boundary.
    Continue,
    /// Build or retry one block without checking control requests again.
    Build,
    /// Transition from the active session to the stopped session.
    Stop,
    /// Shut down the sequencer service.
    Shutdown,
}

/// The sequencer's control plane and linear block-production workflow.
///
/// Unlike the node's step-driven actors, the sequencer owns one long-running async event loop.
/// Control requests are checked once before each block action; after an action starts it runs to a
/// completion without cancellation from the control plane.
#[derive(Debug)]
pub struct SequencerActor<
    AttributesBuilder_,
    Conductor_,
    OriginSelector_,
    SequencerEngineClient_,
    UnsafePayloadGossipClient_,
> where
    AttributesBuilder_: AttributesBuilder,
    Conductor_: Conductor,
    OriginSelector_: OriginSelector,
    SequencerEngineClient_: SequencerEngineClient,
    UnsafePayloadGossipClient_: UnsafePayloadGossipClient,
{
    command_rx: mpsc::Receiver<SequencerCommand>,
    pub(super) conductor: Option<Arc<Conductor_>>,
    pub(super) engine_client: Arc<SequencerEngineClient_>,
    initially_active: bool,
    pub(super) in_recovery_mode: bool,
    pub(super) workflow: SequencingWorkflow<
        AttributesBuilder_,
        Conductor_,
        OriginSelector_,
        SequencerEngineClient_,
        UnsafePayloadGossipClient_,
    >,
}

impl<
    AttributesBuilder_,
    Conductor_,
    OriginSelector_,
    SequencerEngineClient_,
    UnsafePayloadGossipClient_,
>
    SequencerActor<
        AttributesBuilder_,
        Conductor_,
        OriginSelector_,
        SequencerEngineClient_,
        UnsafePayloadGossipClient_,
    >
where
    AttributesBuilder_: AttributesBuilder,
    Conductor_: Conductor,
    OriginSelector_: OriginSelector,
    SequencerEngineClient_: SequencerEngineClient,
    UnsafePayloadGossipClient_: UnsafePayloadGossipClient,
{
    /// Instantiates the sequencer service and its cloneable control handle.
    #[allow(clippy::too_many_arguments)]
    pub fn new(
        attributes_builder: AttributesBuilder_,
        conductor: Option<Conductor_>,
        engine_client: SequencerEngineClient_,
        initially_active: bool,
        in_recovery_mode: bool,
        origin_selector: OriginSelector_,
        rollup_config: Arc<RollupConfig>,
        unsafe_payload_gossip_client: UnsafePayloadGossipClient_,
    ) -> (Self, SequencerHandle) {
        let (command_tx, command_rx) = mpsc::channel(1024);
        let conductor = conductor.map(Arc::new);
        let engine_client = Arc::new(engine_client);
        let workflow = SequencingWorkflow::new(
            attributes_builder,
            conductor.clone(),
            engine_client.clone(),
            origin_selector,
            rollup_config,
            unsafe_payload_gossip_client,
        );

        (
            Self {
                command_rx,
                conductor,
                engine_client,
                initially_active,
                in_recovery_mode,
                workflow,
            },
            SequencerHandle::new(command_tx),
        )
    }

    /// Runs the sequencer until node shutdown or a critical sequencing error.
    pub async fn run(mut self, shutdown: CancellationToken) -> Result<(), SequencerActorError> {
        tokio::select! {
            biased;
            _ = shutdown.cancelled() => return Ok(()),
            result = self.engine_client.reset_engine_forkchoice() => {
                result.map_err(|err| {
                    error!(target: "sequencer", ?err, "Failed to perform initial engine reset");
                    err
                })?;
            }
        }

        if self.initially_active &&
            self.run_active_session(&shutdown).await? == BoundaryAction::Shutdown
        {
            return Ok(());
        }

        loop {
            if self.wait_for_start(&shutdown).await? == BoundaryAction::Shutdown {
                return Ok(());
            }
            if self.run_active_session(&shutdown).await? == BoundaryAction::Shutdown {
                return Ok(());
            }
        }
    }

    /// Runs complete block actions until a stop or shutdown request is observed at a boundary.
    async fn run_active_session(
        &mut self,
        shutdown: &CancellationToken,
    ) -> Result<BoundaryAction, SequencerActorError> {
        let mut retry_candidate = None;

        loop {
            match self.before_next_block(shutdown).await? {
                BoundaryAction::Build => {}
                BoundaryAction::Stop => return Ok(BoundaryAction::Stop),
                BoundaryAction::Shutdown => return Ok(BoundaryAction::Shutdown),
                BoundaryAction::Continue => unreachable!("boundary helper drains continuations"),
            }

            match self
                .workflow
                .sequence_one_block(self.in_recovery_mode, retry_candidate.take())
                .await?
            {
                BlockSequenceOutcome::Canonicalized(head) => {
                    debug!(target: "sequencer", head = ?head.block_info, "Sequencer advanced unsafe head");
                }
                BlockSequenceOutcome::Replan => {}
                BlockSequenceOutcome::Retry(candidate) => retry_candidate = Some(candidate),
            }
        }
    }

    /// Gives queued control requests priority once, immediately before the next block action.
    async fn before_next_block(
        &mut self,
        shutdown: &CancellationToken,
    ) -> Result<BoundaryAction, SequencerActorError> {
        loop {
            tokio::select! {
                biased;
                _ = shutdown.cancelled() => return Ok(BoundaryAction::Shutdown),
                command = self.command_rx.recv() => {
                    match command {
                        Some(command) => match self.handle_command(command, true).await {
                            BoundaryAction::Continue => {}
                            action => return Ok(action),
                        },
                        None => return Ok(BoundaryAction::Build),
                    }
                }
                () = std::future::ready(()) => return Ok(BoundaryAction::Build),
            }
        }
    }

    /// Waits in the stopped session until a start or shutdown request arrives.
    async fn wait_for_start(
        &mut self,
        shutdown: &CancellationToken,
    ) -> Result<BoundaryAction, SequencerActorError> {
        loop {
            tokio::select! {
                biased;
                _ = shutdown.cancelled() => return Ok(BoundaryAction::Shutdown),
                command = self.command_rx.recv() => {
                    let Some(command) = command else {
                        shutdown.cancelled().await;
                        return Ok(BoundaryAction::Shutdown);
                    };
                    match self.handle_command(command, false).await {
                        BoundaryAction::Build => return Ok(BoundaryAction::Build),
                        BoundaryAction::Continue | BoundaryAction::Stop => {}
                        BoundaryAction::Shutdown => return Ok(BoundaryAction::Shutdown),
                    }
                }
            }
        }
    }
}

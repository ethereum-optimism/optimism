//! Long-running unsafe-chain acquisition service.

use crate::{
    engine::{ENGINE_RETRY_DELAY, EngineClient, EngineError},
    unsafe_chain::{
        SequencerHandle, SequencerStatus, SequencingWorkflow, SequencingWorkflowFactory,
        control::SequencerCommand,
    },
};
use kona_rpc::SequencerAdminAPIError;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use thiserror::Error;
use tokio::sync::{mpsc, watch};
use tokio_util::sync::CancellationToken;

#[cfg(feature = "metrics")]
use crate::Metrics;

const CONTROL_CAPACITY: usize = 64;

#[derive(Debug)]
enum LocalProducer {
    Disabled,
    Configured {
        factory: SequencingWorkflowFactory,
        workflow: Option<SequencingWorkflow>,
        command_rx: mpsc::Receiver<SequencerCommand>,
        status_tx: watch::Sender<SequencerStatus>,
        active: bool,
        recovery_mode: bool,
    },
}

/// Owns network following and optional local production of the unsafe chain.
#[derive(Debug)]
pub struct UnsafeChainService {
    engine: EngineClient,
    payload_rx: mpsc::Receiver<OpExecutionPayloadEnvelope>,
    producer: LocalProducer,
}

impl UnsafeChainService {
    /// Creates a follower-only unsafe-chain service.
    pub const fn follower(
        engine: EngineClient,
        payload_rx: mpsc::Receiver<OpExecutionPayloadEnvelope>,
    ) -> Self {
        Self { engine, payload_rx, producer: LocalProducer::Disabled }
    }

    /// Creates an unsafe-chain service with restartable local-production capability.
    pub fn sequencer(
        engine: EngineClient,
        payload_rx: mpsc::Receiver<OpExecutionPayloadEnvelope>,
        factory: SequencingWorkflowFactory,
        initially_active: bool,
        recovery_mode: bool,
    ) -> (Self, SequencerHandle) {
        let (command_tx, command_rx) = mpsc::channel(CONTROL_CAPACITY);
        let status = SequencerStatus {
            active: initially_active,
            recovery_mode,
            conductor_enabled: factory.conductor().is_some(),
        };
        let (status_tx, status_rx) = watch::channel(status);
        Self::update_status_metrics(initially_active, recovery_mode);
        let workflow = initially_active.then(|| factory.create());
        (
            Self {
                engine,
                payload_rx,
                producer: LocalProducer::Configured {
                    factory,
                    workflow,
                    command_rx,
                    status_tx,
                    active: initially_active,
                    recovery_mode,
                },
            },
            SequencerHandle::new(command_tx, status_rx),
        )
    }

    /// Runs until shutdown or a terminal unsafe-chain failure.
    pub async fn run(self, shutdown: CancellationToken) -> Result<(), UnsafeChainServiceError> {
        self.engine.wait_ready().await?;
        let Self { engine, payload_rx, producer } = self;
        match producer {
            LocalProducer::Disabled => Self::run_follower(engine, payload_rx, shutdown).await,
            LocalProducer::Configured {
                factory,
                workflow,
                command_rx,
                status_tx,
                active,
                recovery_mode,
            } => {
                Self::run_with_producer(
                    engine,
                    payload_rx,
                    factory,
                    workflow,
                    command_rx,
                    status_tx,
                    active,
                    recovery_mode,
                    shutdown,
                )
                .await
            }
        }
    }

    async fn run_follower(
        engine: EngineClient,
        mut payload_rx: mpsc::Receiver<OpExecutionPayloadEnvelope>,
        shutdown: CancellationToken,
    ) -> Result<(), UnsafeChainServiceError> {
        loop {
            tokio::select! {
                biased;
                _ = shutdown.cancelled() => return Ok(()),
                payload = payload_rx.recv() => {
                    let payload = payload.ok_or(UnsafeChainServiceError::PayloadChannelClosed)?;
                    Self::import_network_payload(&engine, payload).await?;
                }
            }
        }
    }

    #[allow(clippy::too_many_arguments)]
    async fn run_with_producer(
        engine: EngineClient,
        mut payload_rx: mpsc::Receiver<OpExecutionPayloadEnvelope>,
        factory: SequencingWorkflowFactory,
        mut workflow: Option<SequencingWorkflow>,
        mut command_rx: mpsc::Receiver<SequencerCommand>,
        status_tx: watch::Sender<SequencerStatus>,
        mut active: bool,
        mut recovery_mode: bool,
        shutdown: CancellationToken,
    ) -> Result<(), UnsafeChainServiceError> {
        let conductor = factory.conductor().cloned();
        engine.set_local_sequencing_active(active).await?;
        loop {
            if active {
                if shutdown.is_cancelled() {
                    let _ = engine.set_local_sequencing_active(false).await;
                    Self::publish_status(&status_tx, conductor.is_some(), false, recovery_mode);
                    return Ok(());
                }

                while let Ok(command) = command_rx.try_recv() {
                    Self::handle_command(
                        command,
                        &engine,
                        &factory,
                        &mut workflow,
                        conductor.as_ref(),
                        &status_tx,
                        &mut active,
                        &mut recovery_mode,
                    )
                    .await;
                    if !active {
                        break;
                    }
                }
                if !active {
                    continue;
                }

                // A local producer is authoritative while active. Payloads already queued from
                // gossip are discarded at the boundary rather than racing the local build parent.
                while let Ok(payload) = payload_rx.try_recv() {
                    debug!(target: "unsafe_chain", hash = %payload.block_hash(), "Dropping network unsafe payload while local sequencing is active");
                }

                workflow
                    .as_mut()
                    .expect("active local production always owns a workflow")
                    .sequence_one(recovery_mode)
                    .await?;
                continue;
            }

            tokio::select! {
                biased;
                _ = shutdown.cancelled() => {
                    let _ = engine.set_local_sequencing_active(false).await;
                    return Ok(());
                }
                command = command_rx.recv() => {
                    let command = command.ok_or(UnsafeChainServiceError::ControlChannelClosed)?;
                    Self::handle_command(
                        command,
                        &engine,
                        &factory,
                        &mut workflow,
                        conductor.as_ref(),
                        &status_tx,
                        &mut active,
                        &mut recovery_mode,
                    ).await;
                }
                payload = payload_rx.recv() => {
                    let payload = payload.ok_or(UnsafeChainServiceError::PayloadChannelClosed)?;
                    Self::import_network_payload(&engine, payload).await?;
                }
            }
        }
    }

    #[allow(clippy::too_many_arguments)]
    async fn handle_command(
        command: SequencerCommand,
        engine: &EngineClient,
        factory: &SequencingWorkflowFactory,
        workflow: &mut Option<SequencingWorkflow>,
        conductor: Option<&std::sync::Arc<dyn super::Conductor>>,
        status_tx: &watch::Sender<SequencerStatus>,
        active: &mut bool,
        recovery_mode: &mut bool,
    ) {
        match command {
            SequencerCommand::Start(response) => {
                let result = if *active {
                    Ok(())
                } else {
                    engine
                        .set_local_sequencing_active(true)
                        .await
                        .map_err(|error| SequencerAdminAPIError::RequestError(error.to_string()))
                        .map(|()| {
                            *workflow = Some(factory.create());
                            *active = true;
                        })
                };
                Self::publish_status(status_tx, conductor.is_some(), *active, *recovery_mode);
                let _ = response.send(result);
            }
            SequencerCommand::Stop(response) => {
                let deactivation = engine.set_local_sequencing_active(false).await;
                *active = false;
                *workflow = None;
                Self::publish_status(status_tx, conductor.is_some(), *active, *recovery_mode);
                let result = match deactivation {
                    Ok(()) => engine.state().await.map(|state| state.unsafe_head().hash()),
                    Err(error) => Err(error),
                }
                .map_err(|error| {
                    SequencerAdminAPIError::ErrorAfterSequencerWasStopped(error.to_string())
                });
                let _ = response.send(result);
            }
            SequencerCommand::SetRecoveryMode(mode, response) => {
                *recovery_mode = mode;
                Self::publish_status(status_tx, conductor.is_some(), *active, *recovery_mode);
                let _ = response.send(Ok(()));
            }
            SequencerCommand::OverrideLeader(response) => {
                let result = match conductor {
                    Some(conductor) => conductor.override_leader().await.map_err(|error| {
                        SequencerAdminAPIError::LeaderOverrideError(error.to_string())
                    }),
                    None => Err(SequencerAdminAPIError::LeaderOverrideError(
                        "No conductor configured".to_string(),
                    )),
                };
                let _ = response.send(result);
            }
        }
    }

    fn publish_status(
        status_tx: &watch::Sender<SequencerStatus>,
        conductor_enabled: bool,
        active: bool,
        recovery_mode: bool,
    ) {
        status_tx.send_replace(SequencerStatus { active, recovery_mode, conductor_enabled });
        Self::update_status_metrics(active, recovery_mode);
    }

    fn update_status_metrics(active: bool, recovery_mode: bool) {
        #[cfg(feature = "metrics")]
        metrics::gauge!(
            Metrics::SEQUENCER_STATE,
            "active" => active.to_string(),
            "recovery" => recovery_mode.to_string()
        )
        .set(1);
        #[cfg(not(feature = "metrics"))]
        let _ = (active, recovery_mode);
    }

    async fn import_network_payload(
        engine: &EngineClient,
        payload: OpExecutionPayloadEnvelope,
    ) -> Result<(), UnsafeChainServiceError> {
        loop {
            match engine.import_unsafe(payload.clone()).await {
                Ok(_) => return Ok(()),
                Err(EngineError::InvalidUnsafePayload(error)) => {
                    warn!(target: "unsafe_chain", %error, "Dropping invalid network unsafe payload");
                    return Ok(());
                }
                Err(EngineError::Temporary(_) | EngineError::ResponseDropped) => {
                    tokio::time::sleep(ENGINE_RETRY_DELAY).await;
                }
                Err(error) => return Err(UnsafeChainServiceError::Engine(error)),
            }
        }
    }
}

/// Terminal unsafe-chain service failure.
#[derive(Debug, Error)]
pub enum UnsafeChainServiceError {
    /// Semantic engine operation failed.
    #[error(transparent)]
    Engine(#[from] EngineError),
    /// Local producer failed.
    #[error(transparent)]
    Sequencing(#[from] super::SequencingError),
    /// Network payload producer stopped unexpectedly.
    #[error("unsafe payload channel closed")]
    PayloadChannelClosed,
    /// Every local-producer control handle was dropped.
    #[error("sequencer control channel closed")]
    ControlChannelClosed,
}

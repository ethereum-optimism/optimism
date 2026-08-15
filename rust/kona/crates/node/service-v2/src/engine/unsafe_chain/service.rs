//! Engine-private unsafe-chain acquisition workflow.

use crate::engine::{
    ENGINE_RETRY_DELAY, EngineError,
    api::EngineInternalHandle,
    unsafe_chain::{
        SequencerHandle, SequencerStatus, SequencingWorkflow, SequencingWorkflowFactory,
        control::{SequencerCommand, UnsafeLifecycleCommand},
    },
};
use kona_rpc::SequencerAdminAPIError;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use thiserror::Error;
use tokio::sync::{mpsc, watch};

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

/// Owns network following and optional local production inside Engine.
#[derive(Debug)]
pub(crate) struct UnsafeChainService {
    engine: EngineInternalHandle,
    payload_rx: mpsc::Receiver<OpExecutionPayloadEnvelope>,
    producer: LocalProducer,
    lifecycle_rx: mpsc::Receiver<UnsafeLifecycleCommand>,
}

impl UnsafeChainService {
    /// Creates a follower-only workflow.
    pub(crate) fn follower(
        engine: EngineInternalHandle,
        payload_rx: mpsc::Receiver<OpExecutionPayloadEnvelope>,
    ) -> (Self, SequencerHandle, mpsc::Sender<UnsafeLifecycleCommand>) {
        let (lifecycle_tx, lifecycle_rx) = mpsc::channel(CONTROL_CAPACITY);
        let (_status_tx, status) = watch::channel(SequencerStatus::default());
        (
            Self { engine, payload_rx, producer: LocalProducer::Disabled, lifecycle_rx },
            SequencerHandle::new(None, status),
            lifecycle_tx,
        )
    }

    /// Creates a workflow with restartable local-production capability.
    pub(crate) fn sequencer(
        engine: EngineInternalHandle,
        payload_rx: mpsc::Receiver<OpExecutionPayloadEnvelope>,
        factory: SequencingWorkflowFactory,
        initially_active: bool,
        recovery_mode: bool,
    ) -> (Self, SequencerHandle, mpsc::Sender<UnsafeLifecycleCommand>) {
        let (command_tx, command_rx) = mpsc::channel(CONTROL_CAPACITY);
        let (lifecycle_tx, lifecycle_rx) = mpsc::channel(CONTROL_CAPACITY);
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
                lifecycle_rx,
            },
            SequencerHandle::new(Some(command_tx), status_rx),
            lifecycle_tx,
        )
    }

    /// Runs until its owner explicitly shuts it down or a terminal unsafe failure occurs.
    pub(crate) async fn run(self) -> Result<(), UnsafeChainServiceError> {
        let Self { engine, payload_rx, producer, lifecycle_rx } = self;
        match producer {
            LocalProducer::Disabled => Self::run_follower(engine, payload_rx, lifecycle_rx).await,
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
                    lifecycle_rx,
                )
                .await
            }
        }
    }

    async fn run_follower(
        engine: EngineInternalHandle,
        mut payload_rx: mpsc::Receiver<OpExecutionPayloadEnvelope>,
        mut lifecycle_rx: mpsc::Receiver<UnsafeLifecycleCommand>,
    ) -> Result<(), UnsafeChainServiceError> {
        let mut quiesced = false;
        loop {
            tokio::select! {
                biased;
                command = lifecycle_rx.recv() => {
                    match command {
                        Some(UnsafeLifecycleCommand::Quiesce(done)) => {
                            quiesced = true;
                            let _ = done.send(());
                        }
                        Some(UnsafeLifecycleCommand::Shutdown(done)) => {
                            let _ = done.send(());
                            return Ok(());
                        }
                        None => return Ok(()),
                    }
                }
                payload = payload_rx.recv(), if !quiesced => {
                    let payload = payload.ok_or(UnsafeChainServiceError::PayloadChannelClosed)?;
                    Self::import_network_payload(&engine, payload).await?;
                }
            }
        }
    }

    #[allow(clippy::too_many_arguments)]
    async fn run_with_producer(
        engine: EngineInternalHandle,
        mut payload_rx: mpsc::Receiver<OpExecutionPayloadEnvelope>,
        factory: SequencingWorkflowFactory,
        mut workflow: Option<SequencingWorkflow>,
        mut command_rx: mpsc::Receiver<SequencerCommand>,
        status_tx: watch::Sender<SequencerStatus>,
        mut active: bool,
        mut recovery_mode: bool,
        mut lifecycle_rx: mpsc::Receiver<UnsafeLifecycleCommand>,
    ) -> Result<(), UnsafeChainServiceError> {
        let conductor = factory.conductor().cloned();
        let mut quiesced = false;
        engine.set_local_sequencing_active(active).await?;

        loop {
            if active {
                while let Ok(command) = lifecycle_rx.try_recv() {
                    if Self::handle_lifecycle(
                        command,
                        &engine,
                        &status_tx,
                        conductor.is_some(),
                        &mut workflow,
                        &mut active,
                        recovery_mode,
                        &mut quiesced,
                    )
                    .await?
                    {
                        return Ok(());
                    }
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
                        quiesced,
                    )
                    .await;
                    if !active {
                        break;
                    }
                }
                if !active {
                    continue;
                }

                // Local production is authoritative while active. Drop gossip payloads only at a
                // block boundary, never by racing an accepted publication action.
                while let Ok(payload) = payload_rx.try_recv() {
                    debug!(target: "engine::unsafe", hash = %payload.block_hash(), "Dropping network unsafe payload while local sequencing is active");
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
                command = lifecycle_rx.recv() => {
                    let Some(command) = command else { return Ok(()) };
                    if Self::handle_lifecycle(
                        command,
                        &engine,
                        &status_tx,
                        conductor.is_some(),
                        &mut workflow,
                        &mut active,
                        recovery_mode,
                        &mut quiesced,
                    ).await? {
                        return Ok(());
                    }
                }
                command = command_rx.recv() => {
                    if let Some(command) = command {
                        Self::handle_command(
                            command,
                            &engine,
                            &factory,
                            &mut workflow,
                            conductor.as_ref(),
                            &status_tx,
                            &mut active,
                            &mut recovery_mode,
                            quiesced,
                        ).await;
                    }
                }
                payload = payload_rx.recv(), if !quiesced => {
                    let payload = payload.ok_or(UnsafeChainServiceError::PayloadChannelClosed)?;
                    Self::import_network_payload(&engine, payload).await?;
                }
            }
        }
    }

    #[allow(clippy::too_many_arguments)]
    async fn handle_lifecycle(
        command: UnsafeLifecycleCommand,
        engine: &EngineInternalHandle,
        status_tx: &watch::Sender<SequencerStatus>,
        conductor_enabled: bool,
        workflow: &mut Option<SequencingWorkflow>,
        active: &mut bool,
        recovery_mode: bool,
        quiesced: &mut bool,
    ) -> Result<bool, UnsafeChainServiceError> {
        let (shutdown, done) = match command {
            UnsafeLifecycleCommand::Quiesce(done) => (false, done),
            UnsafeLifecycleCommand::Shutdown(done) => (true, done),
        };
        if *active {
            engine.set_local_sequencing_active(false).await?;
        }
        *active = false;
        *workflow = None;
        *quiesced = true;
        Self::publish_status(status_tx, conductor_enabled, false, recovery_mode);
        let _ = done.send(());
        Ok(shutdown)
    }

    #[allow(clippy::too_many_arguments)]
    async fn handle_command(
        command: SequencerCommand,
        engine: &EngineInternalHandle,
        factory: &SequencingWorkflowFactory,
        workflow: &mut Option<SequencingWorkflow>,
        conductor: Option<&std::sync::Arc<dyn super::Conductor>>,
        status_tx: &watch::Sender<SequencerStatus>,
        active: &mut bool,
        recovery_mode: &mut bool,
        quiesced: bool,
    ) {
        match command {
            SequencerCommand::Start(response) => {
                let result = if quiesced {
                    Err(SequencerAdminAPIError::RequestError(
                        "Engine unsafe processing is shutting down".to_string(),
                    ))
                } else if *active {
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
        engine: &EngineInternalHandle,
        payload: OpExecutionPayloadEnvelope,
    ) -> Result<(), UnsafeChainServiceError> {
        loop {
            match engine.import_unsafe(payload.clone()).await {
                Ok(_) => return Ok(()),
                Err(EngineError::InvalidUnsafePayload(error)) => {
                    warn!(target: "engine::unsafe", %error, "Dropping invalid network unsafe payload");
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

/// Terminal Engine-private unsafe-chain failure.
#[derive(Debug, Error)]
pub(crate) enum UnsafeChainServiceError {
    /// Semantic engine operation failed.
    #[error(transparent)]
    Engine(#[from] EngineError),
    /// Local producer failed.
    #[error(transparent)]
    Sequencing(#[from] super::sequencer::SequencingError),
    /// Network payload producer stopped unexpectedly.
    #[error("unsafe payload channel closed")]
    PayloadChannelClosed,
}

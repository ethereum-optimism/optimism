//! Public control handle for the sequencer service.

use alloy_primitives::B256;
use async_trait::async_trait;
use kona_rpc::{SequencerAdminAPIClient, SequencerAdminAPIError};
use tokio::sync::{mpsc, oneshot};

/// Private command protocol between [`SequencerHandle`] and the sequencer service.
#[derive(Debug)]
pub(super) enum SequencerCommand {
    /// Reports whether the sequencer is active.
    Active(oneshot::Sender<Result<bool, SequencerAdminAPIError>>),
    /// Starts sequencing.
    Start(oneshot::Sender<Result<(), SequencerAdminAPIError>>),
    /// Stops sequencing and returns the final unsafe head.
    Stop(oneshot::Sender<Result<B256, SequencerAdminAPIError>>),
    /// Reports whether conductor integration is configured.
    ConductorEnabled(oneshot::Sender<Result<bool, SequencerAdminAPIError>>),
    /// Reports whether recovery mode is enabled.
    RecoveryMode(oneshot::Sender<Result<bool, SequencerAdminAPIError>>),
    /// Updates recovery mode.
    SetRecoveryMode(bool, oneshot::Sender<Result<(), SequencerAdminAPIError>>),
    /// Overrides conductor leadership.
    OverrideLeader(oneshot::Sender<Result<(), SequencerAdminAPIError>>),
    /// Resets the derivation pipeline's engine forkchoice.
    ResetDerivationPipeline(oneshot::Sender<Result<(), SequencerAdminAPIError>>),
}

/// Cloneable control capability for the sequencer service.
///
/// The command channel is an implementation detail. Callers interact only through methods on this
/// handle, while the sequencer service remains the sole owner of sequencing state and workflow.
#[derive(Debug, Clone)]
pub struct SequencerHandle {
    command_tx: mpsc::Sender<SequencerCommand>,
}

impl SequencerHandle {
    pub(super) const fn new(command_tx: mpsc::Sender<SequencerCommand>) -> Self {
        Self { command_tx }
    }

    async fn request<T>(
        &self,
        command: impl FnOnce(oneshot::Sender<Result<T, SequencerAdminAPIError>>) -> SequencerCommand,
    ) -> Result<T, SequencerAdminAPIError> {
        let (response_tx, response_rx) = oneshot::channel();
        self.command_tx.send(command(response_tx)).await.map_err(|_| {
            SequencerAdminAPIError::RequestError("sequencer command channel closed".to_string())
        })?;
        response_rx.await.map_err(|_| SequencerAdminAPIError::ResponseError)?
    }

    /// Returns whether the sequencer is active.
    pub async fn is_active(&self) -> Result<bool, SequencerAdminAPIError> {
        self.request(SequencerCommand::Active).await
    }

    /// Starts sequencing in an idempotent fashion.
    pub async fn start(&self) -> Result<(), SequencerAdminAPIError> {
        self.request(SequencerCommand::Start).await
    }

    /// Stops sequencing at the next block boundary and returns the final unsafe head.
    pub async fn stop(&self) -> Result<B256, SequencerAdminAPIError> {
        self.request(SequencerCommand::Stop).await
    }

    /// Returns whether conductor integration is configured.
    pub async fn conductor_enabled(&self) -> Result<bool, SequencerAdminAPIError> {
        self.request(SequencerCommand::ConductorEnabled).await
    }

    /// Returns whether recovery mode is enabled.
    pub async fn recovery_mode(&self) -> Result<bool, SequencerAdminAPIError> {
        self.request(SequencerCommand::RecoveryMode).await
    }

    /// Enables or disables recovery mode.
    pub async fn set_recovery_mode(&self, mode: bool) -> Result<(), SequencerAdminAPIError> {
        self.request(|response| SequencerCommand::SetRecoveryMode(mode, response)).await
    }

    /// Overrides conductor leadership.
    pub async fn override_leader(&self) -> Result<(), SequencerAdminAPIError> {
        self.request(SequencerCommand::OverrideLeader).await
    }

    /// Resets the derivation pipeline's engine forkchoice.
    pub async fn reset_derivation_pipeline(&self) -> Result<(), SequencerAdminAPIError> {
        self.request(SequencerCommand::ResetDerivationPipeline).await
    }
}

#[async_trait]
impl SequencerAdminAPIClient for SequencerHandle {
    async fn is_sequencer_active(&self) -> Result<bool, SequencerAdminAPIError> {
        self.is_active().await
    }

    async fn is_conductor_enabled(&self) -> Result<bool, SequencerAdminAPIError> {
        self.conductor_enabled().await
    }

    async fn is_recovery_mode(&self) -> Result<bool, SequencerAdminAPIError> {
        self.recovery_mode().await
    }

    async fn start_sequencer(&self) -> Result<(), SequencerAdminAPIError> {
        self.start().await
    }

    async fn stop_sequencer(&self) -> Result<B256, SequencerAdminAPIError> {
        self.stop().await
    }

    async fn set_recovery_mode(&self, mode: bool) -> Result<(), SequencerAdminAPIError> {
        self.set_recovery_mode(mode).await
    }

    async fn override_leader(&self) -> Result<(), SequencerAdminAPIError> {
        self.override_leader().await
    }

    async fn reset_derivation_pipeline(&self) -> Result<(), SequencerAdminAPIError> {
        self.reset_derivation_pipeline().await
    }
}

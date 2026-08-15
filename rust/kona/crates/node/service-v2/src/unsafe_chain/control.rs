//! Narrow local-sequencer lifecycle control.

use alloy_primitives::B256;
use kona_rpc::SequencerAdminAPIError;
use tokio::sync::{mpsc, oneshot, watch};

/// Private control protocol consumed only at unsafe-chain block boundaries.
#[derive(Debug)]
pub(super) enum SequencerCommand {
    Start(oneshot::Sender<Result<(), SequencerAdminAPIError>>),
    Stop(oneshot::Sender<Result<B256, SequencerAdminAPIError>>),
    SetRecoveryMode(bool, oneshot::Sender<Result<(), SequencerAdminAPIError>>),
    OverrideLeader(oneshot::Sender<Result<(), SequencerAdminAPIError>>),
}

/// Observable local-production status used for administration and recovery gating.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct SequencerStatus {
    /// Whether local block production is active.
    pub active: bool,
    /// Whether transaction-pool exclusion recovery mode is active.
    pub recovery_mode: bool,
    /// Whether an HA conductor is configured.
    pub conductor_enabled: bool,
}

/// Cloneable control capability for the local producer.
#[derive(Debug, Clone)]
pub struct SequencerHandle {
    command_tx: mpsc::Sender<SequencerCommand>,
    status: watch::Receiver<SequencerStatus>,
}

impl SequencerHandle {
    pub(super) const fn new(
        command_tx: mpsc::Sender<SequencerCommand>,
        status: watch::Receiver<SequencerStatus>,
    ) -> Self {
        Self { command_tx, status }
    }

    /// Returns the latest status snapshot.
    pub fn status(&self) -> SequencerStatus {
        *self.status.borrow()
    }

    /// Returns whether local production is active.
    pub fn is_active(&self) -> bool {
        self.status().active
    }

    /// Starts local production at the next unsafe-chain boundary.
    pub async fn start(&self) -> Result<(), SequencerAdminAPIError> {
        self.request(SequencerCommand::Start).await
    }

    /// Stops local production after the accepted block action completes.
    pub async fn stop(&self) -> Result<B256, SequencerAdminAPIError> {
        self.request(SequencerCommand::Stop).await
    }

    /// Enables or disables transaction-pool exclusion recovery mode.
    pub async fn set_recovery_mode(&self, mode: bool) -> Result<(), SequencerAdminAPIError> {
        self.request(|response| SequencerCommand::SetRecoveryMode(mode, response)).await
    }

    /// Overrides conductor leadership.
    pub async fn override_leader(&self) -> Result<(), SequencerAdminAPIError> {
        self.request(SequencerCommand::OverrideLeader).await
    }

    async fn request<T>(
        &self,
        command: impl FnOnce(oneshot::Sender<Result<T, SequencerAdminAPIError>>) -> SequencerCommand,
    ) -> Result<T, SequencerAdminAPIError> {
        let (response, result) = oneshot::channel();
        self.command_tx.send(command(response)).await.map_err(|_| {
            SequencerAdminAPIError::RequestError(
                "unsafe-chain control service is unavailable".to_string(),
            )
        })?;
        result.await.map_err(|_| SequencerAdminAPIError::ResponseError)?
    }
}

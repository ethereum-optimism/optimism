//! Engine-private local-sequencer and unsafe-workflow control.

use alloy_primitives::B256;
use kona_rpc::SequencerAdminAPIError;
use tokio::sync::{mpsc, oneshot, watch};

/// Administration consumed only at unsafe-chain block boundaries.
#[derive(Debug)]
pub(super) enum SequencerCommand {
    Start(oneshot::Sender<Result<(), SequencerAdminAPIError>>),
    Stop(oneshot::Sender<Result<B256, SequencerAdminAPIError>>),
    SetRecoveryMode(bool, oneshot::Sender<Result<(), SequencerAdminAPIError>>),
    OverrideLeader(oneshot::Sender<Result<(), SequencerAdminAPIError>>),
}

/// Node lifecycle commands, separate from externally reachable administration.
#[derive(Debug)]
pub(crate) enum UnsafeLifecycleCommand {
    Quiesce(oneshot::Sender<()>),
    Shutdown(oneshot::Sender<()>),
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

/// Cloneable Engine-private control capability for the local producer.
#[derive(Debug, Clone)]
pub(crate) struct SequencerHandle {
    command_tx: Option<mpsc::Sender<SequencerCommand>>,
    status: watch::Receiver<SequencerStatus>,
}

impl SequencerHandle {
    pub(super) const fn new(
        command_tx: Option<mpsc::Sender<SequencerCommand>>,
        status: watch::Receiver<SequencerStatus>,
    ) -> Self {
        Self { command_tx, status }
    }

    pub(crate) fn configured_status(&self) -> Result<SequencerStatus, SequencerAdminAPIError> {
        self.sender()?;
        Ok(*self.status.borrow())
    }

    fn sender(&self) -> Result<&mpsc::Sender<SequencerCommand>, SequencerAdminAPIError> {
        self.command_tx.as_ref().ok_or_else(|| {
            SequencerAdminAPIError::RequestError("local sequencing is not configured".to_string())
        })
    }

    pub(crate) async fn start(&self) -> Result<(), SequencerAdminAPIError> {
        self.request(SequencerCommand::Start).await
    }

    pub(crate) async fn stop(&self) -> Result<B256, SequencerAdminAPIError> {
        self.request(SequencerCommand::Stop).await
    }

    pub(crate) async fn set_recovery_mode(&self, mode: bool) -> Result<(), SequencerAdminAPIError> {
        self.request(|response| SequencerCommand::SetRecoveryMode(mode, response)).await
    }

    pub(crate) async fn override_leader(&self) -> Result<(), SequencerAdminAPIError> {
        self.request(SequencerCommand::OverrideLeader).await
    }

    async fn request<T>(
        &self,
        command: impl FnOnce(oneshot::Sender<Result<T, SequencerAdminAPIError>>) -> SequencerCommand,
    ) -> Result<T, SequencerAdminAPIError> {
        let (response, result) = oneshot::channel();
        self.sender()?.send(command(response)).await.map_err(|_| {
            SequencerAdminAPIError::RequestError(
                "Engine unsafe-chain workflow is unavailable".to_string(),
            )
        })?;
        result.await.map_err(|_| SequencerAdminAPIError::ResponseError)?
    }
}

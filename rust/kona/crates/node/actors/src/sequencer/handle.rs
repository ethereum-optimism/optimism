use alloy_primitives::B256;
use thiserror::Error;
use tokio::sync::{mpsc, oneshot, watch};

/// Sequencer state shared by the actor and admin RPC readers.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct SequencerState {
    /// Whether the sequencer is active.
    pub active: bool,
    /// Whether a conductor is configured.
    pub conductor_enabled: bool,
    /// Whether the sequencer is in recovery mode.
    pub recovery_mode: bool,
}

/// State-changing admin commands executed by the sequencer actor.
#[derive(Debug)]
pub enum SequencerAdminCommand {
    /// Start sequencing.
    StartSequencer(oneshot::Sender<Result<(), SequencerAdminAPIError>>),
    /// Stop sequencing and return the current unsafe block hash.
    StopSequencer(oneshot::Sender<Result<B256, SequencerAdminAPIError>>),
    /// Set recovery mode.
    SetRecoveryMode(bool, oneshot::Sender<Result<(), SequencerAdminAPIError>>),
    /// Override the conductor leader.
    OverrideLeader(oneshot::Sender<Result<(), SequencerAdminAPIError>>),
}

/// Published sequencer state and its admin command queue.
#[derive(Debug, Clone)]
pub struct Handle {
    state: watch::Receiver<SequencerState>,
    commands: mpsc::Sender<SequencerAdminCommand>,
}

impl Handle {
    /// Construct a handle from the sequencer's published state and command sender.
    pub const fn new(
        state: watch::Receiver<SequencerState>,
        commands: mpsc::Sender<SequencerAdminCommand>,
    ) -> Self {
        Self { state, commands }
    }

    /// Returns the latest published state.
    pub fn snapshot(&self) -> Result<SequencerState, SequencerAdminAPIError> {
        // Fail if the actor exited, even if its final update has not been read yet.
        self.state
            .has_changed()
            .map_err(|error| SequencerAdminAPIError::RequestError(error.to_string()))?;
        Ok(*self.state.borrow())
    }

    /// Starts sequencing.
    pub async fn start(&self) -> Result<(), SequencerAdminAPIError> {
        self.command(SequencerAdminCommand::StartSequencer).await
    }

    /// Stops sequencing and returns the unsafe head hash.
    pub async fn stop(&self) -> Result<B256, SequencerAdminAPIError> {
        self.command(SequencerAdminCommand::StopSequencer).await
    }

    /// Sets recovery mode.
    pub async fn set_recovery_mode(&self, enabled: bool) -> Result<(), SequencerAdminAPIError> {
        self.command(|tx| SequencerAdminCommand::SetRecoveryMode(enabled, tx)).await
    }

    /// Overrides the conductor leader.
    pub async fn override_leader(&self) -> Result<(), SequencerAdminAPIError> {
        self.command(SequencerAdminCommand::OverrideLeader).await
    }

    async fn command<T>(
        &self,
        command: impl FnOnce(
            oneshot::Sender<Result<T, SequencerAdminAPIError>>,
        ) -> SequencerAdminCommand,
    ) -> Result<T, SequencerAdminAPIError> {
        let (tx, rx) = oneshot::channel();
        self.commands
            .send(command(tx))
            .await
            .map_err(|error| SequencerAdminAPIError::RequestError(error.to_string()))?;
        rx.await.map_err(|error| SequencerAdminAPIError::RequestError(error.to_string()))?
    }
}

/// Errors that can occur when using the sequencer admin API.
#[derive(Debug, Error)]
pub enum SequencerAdminAPIError {
    /// Error sending request.
    #[error("Error sending request: {0}.")]
    RequestError(String),

    /// Sequencer stopped successfully, followed by some error.
    #[error("Sequencer stopped successfully, followed by error: {0}.")]
    ErrorAfterSequencerWasStopped(String),

    /// Error overriding leader.
    #[error("Error overriding leader: {0}.")]
    LeaderOverrideError(String),
}

//! The sequencer's private command protocol and published administrative state.

use alloy_primitives::B256;
use async_trait::async_trait;
use kona_rpc::{SequencerAdminAPIClient, SequencerAdminAPIError};
use tokio::sync::{mpsc, oneshot, watch};

/// Administrative state published by the sequencer owner.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct SequencerState {
    /// Whether the sequencer is active.
    pub active: bool,
    /// Whether the sequencer uses recovery-mode origin selection.
    pub recovery_mode: bool,
}

pub(super) type Reply<T> = oneshot::Sender<Result<T, SequencerAdminAPIError>>;

#[derive(Debug)]
pub(crate) enum Message {
    Start(Reply<()>),
    Stop(Reply<B256>),
    SetRecoveryMode(bool, Reply<()>),
    OverrideLeader(Reply<()>),
}

/// Access to the sequencer owner; reads do not wait for its command queue or dependencies.
#[derive(Debug, Clone)]
pub struct SequencerHandle {
    commands: mpsc::Sender<Message>,
    state: watch::Receiver<SequencerState>,
    conductor_enabled: bool,
}

impl SequencerHandle {
    pub(super) const fn new(
        commands: mpsc::Sender<Message>,
        state: watch::Receiver<SequencerState>,
        conductor_enabled: bool,
    ) -> Self {
        Self { commands, state, conductor_enabled }
    }

    fn snapshot(&self) -> Result<SequencerState, SequencerAdminAPIError> {
        // A closed owner must fail even when its final publication has not yet been consumed.
        self.state.has_changed().map_err(|_| SequencerAdminAPIError::ResponseError)?;
        Ok(*self.state.borrow())
    }

    async fn command<T>(
        &self,
        message: impl FnOnce(Reply<T>) -> Message,
    ) -> Result<T, SequencerAdminAPIError> {
        let (reply, result) = oneshot::channel();
        self.commands
            .send(message(reply))
            .await
            .map_err(|_| SequencerAdminAPIError::RequestError("request channel closed".into()))?;
        result.await.map_err(|_| SequencerAdminAPIError::ResponseError)?
    }

    #[cfg(test)]
    pub(crate) fn test_channel(
        state: SequencerState,
        conductor_enabled: bool,
    ) -> (Self, TestSequencer) {
        let (commands, command_rx) = mpsc::channel(32);
        let (state_tx, state_rx) = watch::channel(state);
        (
            Self::new(commands, state_rx, conductor_enabled),
            TestSequencer { commands: command_rx, state: state_tx },
        )
    }
}

#[async_trait]
impl SequencerAdminAPIClient for SequencerHandle {
    async fn is_sequencer_active(&self) -> Result<bool, SequencerAdminAPIError> {
        Ok(self.snapshot()?.active)
    }

    async fn is_conductor_enabled(&self) -> Result<bool, SequencerAdminAPIError> {
        self.snapshot()?;
        Ok(self.conductor_enabled)
    }

    async fn is_recovery_mode(&self) -> Result<bool, SequencerAdminAPIError> {
        Ok(self.snapshot()?.recovery_mode)
    }

    async fn start_sequencer(&self) -> Result<(), SequencerAdminAPIError> {
        self.command(Message::Start).await
    }

    async fn stop_sequencer(&self) -> Result<B256, SequencerAdminAPIError> {
        self.command(Message::Stop).await
    }

    async fn set_recovery_mode(&self, mode: bool) -> Result<(), SequencerAdminAPIError> {
        self.command(|reply| Message::SetRecoveryMode(mode, reply)).await
    }

    async fn override_leader(&self) -> Result<(), SequencerAdminAPIError> {
        self.command(Message::OverrideLeader).await
    }
}

#[cfg(test)]
#[derive(Debug)]
pub(crate) struct TestSequencer {
    pub(crate) commands: mpsc::Receiver<Message>,
    pub(crate) state: watch::Sender<SequencerState>,
}

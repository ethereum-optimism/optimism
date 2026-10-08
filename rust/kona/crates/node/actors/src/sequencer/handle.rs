use alloy_primitives::B256;
use thiserror::Error;
use tokio::sync::{mpsc, oneshot, watch};

/// Sequencer state shared by the actor and admin RPC readers.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct State {
    /// Whether the sequencer is active.
    pub active: bool,
    /// Whether a conductor is configured.
    pub conductor_enabled: bool,
}

/// State-changing admin commands executed by the sequencer actor.
#[derive(Debug)]
pub(super) enum Message {
    /// Start sequencing.
    StartSequencer(oneshot::Sender<Result<(), HandleError>>),
    /// Stop sequencing and return the current unsafe block hash.
    StopSequencer(oneshot::Sender<Result<B256, HandleError>>),
    /// Override the conductor leader.
    OverrideLeader(oneshot::Sender<Result<(), HandleError>>),
}

/// Published sequencer state and its admin command queue.
#[derive(Debug, Clone)]
pub struct Handle {
    state: watch::Receiver<State>,
    commands: mpsc::Sender<Message>,
}

impl Handle {
    /// Construct a handle from the sequencer's published state and command sender.
    pub(super) const fn new(
        state: watch::Receiver<State>,
        commands: mpsc::Sender<Message>,
    ) -> Self {
        Self { state, commands }
    }

    /// Returns the latest published state.
    ///
    /// Fails after the publisher closes.
    pub fn snapshot(&self) -> Result<State, HandleError> {
        // Fail if the actor exited, even if its final update has not been read yet.
        self.state.has_changed().map_err(|error| HandleError::RequestError(error.to_string()))?;
        Ok(*self.state.borrow())
    }

    /// Starts sequencing.
    pub async fn start(&self) -> Result<(), HandleError> {
        self.command(Message::StartSequencer).await
    }

    /// Stops sequencing and returns the unsafe head hash.
    pub async fn stop(&self) -> Result<B256, HandleError> {
        self.command(Message::StopSequencer).await
    }

    /// Overrides the conductor leader.
    pub async fn override_leader(&self) -> Result<(), HandleError> {
        self.command(Message::OverrideLeader).await
    }

    async fn command<T>(
        &self,
        command: impl FnOnce(oneshot::Sender<Result<T, HandleError>>) -> Message,
    ) -> Result<T, HandleError> {
        let (tx, rx) = oneshot::channel();
        self.commands
            .send(command(tx))
            .await
            .map_err(|error| HandleError::RequestError(error.to_string()))?;
        rx.await.map_err(|error| HandleError::RequestError(error.to_string()))?
    }
}

/// Errors that can occur when using the sequencer admin API.
#[derive(Debug, Error)]
pub enum HandleError {
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

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn snapshot_reads_state_with_a_full_mailbox_and_rejects_closed_publisher() {
        let initial = State { active: false, conductor_enabled: true };
        let (published, state) = watch::channel(initial);
        let (commands, _receiver) = mpsc::channel(1);
        let (reply, _) = oneshot::channel();
        commands.send(Message::StartSequencer(reply)).await.unwrap();
        let handle = Handle::new(state, commands);
        assert_eq!(handle.snapshot().unwrap(), initial);

        let updated = State { active: true, ..initial };
        published.send_replace(updated);
        assert_eq!(handle.snapshot().unwrap(), updated);
        // Reject closed publishers even when their final update is unread.
        drop(published);
        assert!(matches!(handle.snapshot(), Err(HandleError::RequestError(_))));
    }

    #[tokio::test]
    async fn commands_return_actor_results_and_channel_errors() {
        let (_published, state) = watch::channel(State { active: true, conductor_enabled: false });
        let (commands, mut receiver) = mpsc::channel(1);
        let handle = Handle::new(state, commands);
        let hash = B256::repeat_byte(42);
        let (result, ()) = tokio::join!(handle.stop(), async {
            let Message::StopSequencer(reply) = receiver.recv().await.unwrap() else {
                panic!("expected stop");
            };
            reply.send(Ok(hash)).unwrap();
        });
        assert_eq!(result.unwrap(), hash);

        let (result, ()) = tokio::join!(handle.override_leader(), async {
            let Message::OverrideLeader(reply) = receiver.recv().await.unwrap() else {
                panic!("expected leader override");
            };
            reply.send(Err(HandleError::LeaderOverrideError("unavailable".to_owned()))).unwrap();
        });
        assert!(matches!(result, Err(HandleError::LeaderOverrideError(_))));
        let (result, ()) = tokio::join!(handle.start(), async {
            drop(receiver.recv().await.unwrap());
        });
        assert!(matches!(result, Err(HandleError::RequestError(_))));
        drop(receiver);
        assert!(matches!(handle.start().await, Err(HandleError::RequestError(_))));
    }
}

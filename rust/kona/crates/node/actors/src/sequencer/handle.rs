use alloy_primitives::B256;
use thiserror::Error;
use tokio::sync::{mpsc, oneshot, watch};

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
    is_active: watch::Receiver<bool>,
    commands: mpsc::Sender<Message>,
}

impl Handle {
    /// Construct a handle from the sequencer's published state and command sender.
    pub(super) const fn new(
        is_active: watch::Receiver<bool>,
        commands: mpsc::Sender<Message>,
    ) -> Self {
        Self { is_active, commands }
    }

    /// Returns whether sequencing is active.
    ///
    /// Fails after the publisher closes.
    pub fn is_active(&self) -> Result<bool, HandleError> {
        // Fail if the actor exited, even if its final update has not been read yet.
        self.is_active
            .has_changed()
            .map_err(|error| HandleError::RequestError(error.to_string()))?;
        Ok(*self.is_active.borrow())
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
    async fn is_active_reads_with_a_full_mailbox_and_rejects_closed_publisher() {
        let initial = false;
        let (published, state) = watch::channel(initial);
        let (commands, _receiver) = mpsc::channel(1);
        let (reply, _) = oneshot::channel();
        commands.send(Message::StartSequencer(reply)).await.unwrap();
        let handle = Handle::new(state, commands);
        assert_eq!(handle.is_active().unwrap(), initial);

        let updated = true;
        published.send_replace(updated);
        assert_eq!(handle.is_active().unwrap(), updated);
        // Reject closed publishers even when their final update is unread.
        drop(published);
        assert!(matches!(handle.is_active(), Err(HandleError::RequestError(_))));
    }

    #[tokio::test]
    async fn commands_return_actor_results_and_channel_errors() {
        let (_published, state) = watch::channel(true);
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

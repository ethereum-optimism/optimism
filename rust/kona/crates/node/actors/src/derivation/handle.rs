//! Requests to the derivation actor.

use super::{DerivationStatus, Handle, Message};
use kona_derive::Signal;
use kona_protocol::L2BlockInfo;
use thiserror::Error;
use tokio::sync::watch;

/// The result of a request to the derivation actor.
pub type DerivationClientResult<T> = Result<T, DerivationClientError>;

/// Error making requests to the derivation actor.
#[derive(Debug, Error)]
pub enum DerivationClientError {
    /// Error making a request to the derivation actor. The request never made it there.
    #[error("Error making a request to the derivation actor: {0}.")]
    RequestError(String),
}

impl Handle {
    /// Subscribes to the derivation progress the actor publishes.
    pub fn status(&self) -> watch::Receiver<DerivationStatus> {
        self.status.clone()
    }

    /// Queues notice that the execution layer finished syncing, with the safe head derivation
    /// starts from.
    pub async fn notify_el_sync_completed(
        &self,
        safe_head: L2BlockInfo,
    ) -> DerivationClientResult<()> {
        self.send(Message::ElSyncCompleted(Box::new(safe_head))).await
    }

    /// Queues the engine's new safe head.
    pub async fn send_safe_head(&self, safe_head: L2BlockInfo) -> DerivationClientResult<()> {
        self.send(Message::SafeHead(Box::new(safe_head))).await
    }

    /// Queues a signal for the derivation pipeline.
    pub async fn send_signal(&self, signal: Signal) -> DerivationClientResult<()> {
        self.send(Message::Signal(Box::new(signal))).await
    }

    async fn send(&self, message: Message) -> DerivationClientResult<()> {
        self.messages
            .send(message)
            .await
            .map_err(|_| DerivationClientError::RequestError("request channel closed.".to_string()))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tokio::sync::mpsc;

    fn handle() -> (Handle, mpsc::Receiver<Message>) {
        let (messages, receiver) = mpsc::channel(3);
        let (_, status) = watch::channel(DerivationStatus::default());
        (Handle { messages, status }, receiver)
    }

    #[tokio::test]
    async fn requests_send_their_message() {
        let (handle, mut receiver) = handle();
        let head = L2BlockInfo::default();
        handle.notify_el_sync_completed(head).await.unwrap();
        handle.send_safe_head(head).await.unwrap();
        handle.send_signal(Signal::FlushChannel).await.unwrap();
        assert!(matches!(
            receiver.recv().await.unwrap(),
            Message::ElSyncCompleted(block) if *block == head
        ));
        assert!(
            matches!(receiver.recv().await.unwrap(), Message::SafeHead(block) if *block == head)
        );
        assert!(matches!(
            receiver.recv().await.unwrap(),
            Message::Signal(signal) if matches!(*signal, Signal::FlushChannel)
        ));
    }

    #[tokio::test]
    async fn requests_report_a_closed_channel() {
        let (handle, receiver) = handle();
        drop(receiver);
        assert!(matches!(
            handle.send_signal(Signal::FlushChannel).await,
            Err(DerivationClientError::RequestError(_))
        ));
    }
}

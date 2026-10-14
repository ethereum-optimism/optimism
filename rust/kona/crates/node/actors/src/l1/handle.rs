//! Requests to the L1 actor.

use kona_protocol::BlockInfo;
use std::num::NonZeroUsize;
use thiserror::Error;
use tokio::sync::{mpsc, oneshot};

/// What changed on L1 since a consumer's cursor.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Update {
    /// Canonical blocks that extend the cursor, oldest first.
    Advanced(Vec<BlockInfo>),
    /// The cursor was reorged out. The consumer rolls back to `ancestor`, the newest canonical
    /// block on the cursor's chain, then applies `blocks`, oldest first.
    Reorged {
        /// The newest block both on the cursor's chain and on the canonical chain.
        ancestor: BlockInfo,
        /// Canonical blocks that extend the ancestor, oldest first.
        blocks: Vec<BlockInfo>,
    },
    /// The cursor is older than `base`, the latest finalized block, so the actor cannot tell
    /// whether it is canonical. Blocks up to `base` cannot reorg: the consumer catches up by
    /// number through `base`, checking parent hashes, and asks again from there.
    Behind {
        /// The latest finalized block, the oldest block the actor holds.
        base: BlockInfo,
    },
    /// The cursor is on no chain the actor knows of, so the consumer resets.
    Unknown,
}

/// The latest L1 blocks the actor observed.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct Status {
    /// The canonical head.
    pub head: BlockInfo,
    /// The safe block.
    pub safe: BlockInfo,
    /// The finalized block.
    pub finalized: BlockInfo,
}

/// Requests to the L1 actor.
#[derive(Debug, Clone)]
pub struct Handle {
    messages: mpsc::Sender<Message>,
}

/// The L1 actor stopped.
#[derive(Debug, Clone, Copy, Error)]
#[error("the L1 actor stopped")]
pub struct HandleError;

impl Handle {
    /// Creates a handle that sends requests to `messages`.
    pub(super) const fn new(messages: mpsc::Sender<Message>) -> Self {
        Self { messages }
    }

    /// Waits until there is something to report after `cursor`, then returns it, with at most
    /// `max` blocks.
    pub async fn updates_since(
        &self,
        cursor: BlockInfo,
        max: NonZeroUsize,
    ) -> Result<Update, HandleError> {
        self.request(|reply| Message::UpdatesSince { cursor, max, reply }).await
    }

    /// Waits until the canonical head is a block other than `seen`, and returns it.
    pub async fn next_head(&self, seen: BlockInfo) -> Result<BlockInfo, HandleError> {
        self.request(|reply| Message::NextHead(ChangeWaiter { seen, reply })).await
    }

    /// Waits until the finalized block is a block other than `seen`, and returns it.
    pub async fn next_finalized(&self, seen: BlockInfo) -> Result<BlockInfo, HandleError> {
        self.request(|reply| Message::NextFinalized(ChangeWaiter { seen, reply })).await
    }

    /// Returns the latest L1 blocks the actor observed.
    pub async fn status(&self) -> Result<Status, HandleError> {
        self.request(Message::Status).await
    }

    async fn request<T>(
        &self,
        message: impl FnOnce(oneshot::Sender<T>) -> Message,
    ) -> Result<T, HandleError> {
        let (reply, response) = oneshot::channel();
        self.messages.send(message(reply)).await.map_err(|_| HandleError)?;
        response.await.map_err(|_| HandleError)
    }
}

#[cfg(test)]
impl Handle {
    /// Returns a handle whose actor reports `status`, and drops every other request.
    pub(crate) fn reporting(status: tokio::sync::watch::Receiver<Status>) -> Self {
        let (messages, mut requests) = mpsc::channel(1);
        tokio::spawn(async move {
            while let Some(message) = requests.recv().await {
                if let Message::Status(reply) = message {
                    let _ = reply.send(*status.borrow());
                }
            }
        });
        Self { messages }
    }
}

/// A request to the L1 actor.
#[derive(Debug)]
pub(super) enum Message {
    UpdatesSince { cursor: BlockInfo, max: NonZeroUsize, reply: oneshot::Sender<Update> },
    NextHead(ChangeWaiter),
    NextFinalized(ChangeWaiter),
    Status(oneshot::Sender<Status>),
}

/// A request waiting for a block other than `seen`.
#[derive(Debug)]
pub(super) struct ChangeWaiter {
    pub(super) seen: BlockInfo,
    pub(super) reply: oneshot::Sender<BlockInfo>,
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::l1::test_utils::block;

    #[tokio::test]
    async fn a_reporting_handle_serves_the_status() {
        let status = Status { head: block(3, 1, 1), ..Default::default() };
        let (_published, receiver) = tokio::sync::watch::channel(status);
        assert_eq!(Handle::reporting(receiver).status().await.unwrap(), status);
    }
}

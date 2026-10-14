use discv5::{Enr, kbucket::NodeStatus};
use enr::NodeId;
use libp2p::Multiaddr;
use std::{collections::HashSet, time::Duration};
use thiserror::Error;
use tokio::sync::{
    mpsc::{self, error::TrySendError},
    oneshot,
};

/// Queries and bans executed by the discovery actor.
#[derive(Debug)]
pub(super) enum Message {
    /// Read the local ENR.
    LocalEnr(oneshot::Sender<Enr>),
    /// Count the connected discv5 peers.
    PeerCount(oneshot::Sender<usize>),
    /// Read the ENRs in the routing table.
    TableEnrs(oneshot::Sender<Vec<Enr>>),
    /// Read the routing table entries.
    TableInfos(oneshot::Sender<Vec<(NodeId, Enr, NodeStatus)>>),
    /// Ban the routing table's nodes at any of these addresses.
    Ban {
        /// The addresses to ban.
        addrs: HashSet<Multiaddr>,
        /// How long to ban them for.
        duration: Duration,
    },
}

/// Sends queries and bans to the discovery actor.
///
/// Requests wait in the actor's queue until it has started the discv5 service.
#[derive(Debug, Clone)]
pub struct Handle {
    messages: mpsc::Sender<Message>,
}

impl Handle {
    /// Constructs a handle from the actor's message sender.
    pub(super) const fn new(messages: mpsc::Sender<Message>) -> Self {
        Self { messages }
    }

    /// Returns the local ENR.
    pub async fn local_enr(&self) -> Result<Enr, HandleError> {
        self.request(Message::LocalEnr).await
    }

    /// Returns the number of connected discv5 peers.
    pub async fn peer_count(&self) -> Result<usize, HandleError> {
        self.request(Message::PeerCount).await
    }

    /// Returns the ENRs in the routing table.
    pub async fn table_enrs(&self) -> Result<Vec<Enr>, HandleError> {
        self.request(Message::TableEnrs).await
    }

    /// Returns the routing table entries.
    pub async fn table_infos(&self) -> Result<Vec<(NodeId, Enr, NodeStatus)>, HandleError> {
        self.request(Message::TableInfos).await
    }

    /// Queues a ban of the routing table's nodes at any of `addrs` for `duration`, without
    /// waiting for room in the queue.
    ///
    /// Success means the ban was queued, not that it was applied.
    pub fn try_ban(
        &self,
        addrs: HashSet<Multiaddr>,
        duration: Duration,
    ) -> Result<(), TrySendError<()>> {
        self.messages.try_send(Message::Ban { addrs, duration }).map_err(crate::without_value)
    }

    async fn request<T>(
        &self,
        message: impl FnOnce(oneshot::Sender<T>) -> Message,
    ) -> Result<T, HandleError> {
        let (tx, rx) = oneshot::channel();
        self.messages.send(message(tx)).await.map_err(|_| HandleError)?;
        rx.await.map_err(|_| HandleError)
    }
}

/// The discovery actor stopped before answering.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Error)]
#[error("The discovery actor stopped")]
pub struct HandleError;

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn requests_return_replies_and_report_a_stopped_actor() {
        let (messages, mut receiver) = mpsc::channel(1);
        let handle = Handle::new(messages);

        let (count, ()) = tokio::join!(handle.peer_count(), async {
            let Message::PeerCount(reply) = receiver.recv().await.unwrap() else {
                panic!("expected a peer count query");
            };
            reply.send(3).unwrap();
        });
        assert_eq!(count, Ok(3));

        let addrs = HashSet::from(["/ip4/192.0.2.1/tcp/9000".parse().unwrap()]);
        handle.try_ban(addrs.clone(), Duration::from_secs(5)).unwrap();
        // The queue holds one message, so a second ban finds it full instead of waiting.
        assert_eq!(handle.try_ban(addrs.clone(), Duration::ZERO), Err(TrySendError::Full(())));
        assert!(matches!(
            receiver.recv().await.unwrap(),
            Message::Ban { addrs: banned, duration } if banned == addrs && duration == Duration::from_secs(5)
        ));

        // The actor dropped the query without replying.
        let (enrs, ()) = tokio::join!(handle.table_enrs(), async {
            drop(receiver.recv().await.unwrap());
        });
        assert_eq!(enrs, Err(HandleError));

        drop(receiver);
        assert_eq!(handle.local_enr().await.map(|_| ()), Err(HandleError));
        assert_eq!(handle.try_ban(HashSet::new(), Duration::ZERO), Err(TrySendError::Closed(())));
    }
}

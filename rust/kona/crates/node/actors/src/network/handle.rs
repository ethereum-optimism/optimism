use discv5::Enr;
use ipnet::IpNet;
use kona_gossip::GossipState;
use libp2p::{Multiaddr, PeerId};
use std::{net::IpAddr, sync::Arc};
use thiserror::Error;
use tokio::sync::{
    mpsc::{self, error::TrySendError},
    oneshot, watch,
};

/// Replies once the actor has published the state that follows a change.
pub(super) type Applied = oneshot::Sender<()>;

/// Changes the network actor makes to its gossip driver.
#[derive(Debug)]
pub(super) enum Message {
    /// Block a peer.
    BlockPeer(PeerId, Applied),
    /// Unblock a peer.
    UnblockPeer(PeerId, Applied),
    /// Block an IP address.
    BlockAddr(IpAddr, Applied),
    /// Unblock an IP address.
    UnblockAddr(IpAddr, Applied),
    /// Block a subnet.
    BlockSubnet(IpNet, Applied),
    /// Unblock a subnet.
    UnblockSubnet(IpNet, Applied),
    /// Dial a peer.
    ConnectPeer(Multiaddr, Applied),
    /// Disconnect a peer.
    DisconnectPeer(PeerId, Applied),
    /// Protect a peer from disconnection.
    ProtectPeer(PeerId, Applied),
    /// Stop protecting a peer from disconnection.
    UnprotectPeer(PeerId, Applied),
}

/// The network actor's published gossip state, its message queue, and its queue of peers to dial.
#[derive(Debug, Clone)]
pub struct Handle {
    state: watch::Receiver<Arc<GossipState>>,
    messages: mpsc::Sender<Message>,
    peers: mpsc::Sender<Enr>,
}

impl Handle {
    /// Constructs a handle from the actor's published state and its senders.
    pub(super) const fn new(
        state: watch::Receiver<Arc<GossipState>>,
        messages: mpsc::Sender<Message>,
        peers: mpsc::Sender<Enr>,
    ) -> Self {
        Self { state, messages, peers }
    }

    /// Subscribes to the gossip state the actor publishes after each event and change.
    pub fn state(&self) -> watch::Receiver<Arc<GossipState>> {
        self.state.clone()
    }

    /// Blocks a peer.
    pub async fn block_peer(&self, id: PeerId) -> Result<(), HandleError> {
        self.apply(|applied| Message::BlockPeer(id, applied)).await
    }

    /// Unblocks a peer.
    pub async fn unblock_peer(&self, id: PeerId) -> Result<(), HandleError> {
        self.apply(|applied| Message::UnblockPeer(id, applied)).await
    }

    /// Blocks an IP address.
    pub async fn block_addr(&self, address: IpAddr) -> Result<(), HandleError> {
        self.apply(|applied| Message::BlockAddr(address, applied)).await
    }

    /// Unblocks an IP address.
    pub async fn unblock_addr(&self, address: IpAddr) -> Result<(), HandleError> {
        self.apply(|applied| Message::UnblockAddr(address, applied)).await
    }

    /// Blocks a subnet.
    pub async fn block_subnet(&self, subnet: IpNet) -> Result<(), HandleError> {
        self.apply(|applied| Message::BlockSubnet(subnet, applied)).await
    }

    /// Unblocks a subnet.
    pub async fn unblock_subnet(&self, subnet: IpNet) -> Result<(), HandleError> {
        self.apply(|applied| Message::UnblockSubnet(subnet, applied)).await
    }

    /// Dials a peer. Success means the actor handled the request, not that it dialed: the
    /// connection gate may refuse the address, and a connected peer is not dialed again.
    pub async fn connect_peer(&self, address: Multiaddr) -> Result<(), HandleError> {
        self.apply(|applied| Message::ConnectPeer(address, applied)).await
    }

    /// Disconnects a peer.
    pub async fn disconnect_peer(&self, id: PeerId) -> Result<(), HandleError> {
        self.apply(|applied| Message::DisconnectPeer(id, applied)).await
    }

    /// Protects a peer from disconnection.
    pub async fn protect_peer(&self, id: PeerId) -> Result<(), HandleError> {
        self.apply(|applied| Message::ProtectPeer(id, applied)).await
    }

    /// Stops protecting a peer from disconnection.
    pub async fn unprotect_peer(&self, id: PeerId) -> Result<(), HandleError> {
        self.apply(|applied| Message::UnprotectPeer(id, applied)).await
    }

    /// Queues a peer to dial, without waiting for room in the queue.
    pub fn dial(&self, enr: Enr) -> Result<(), TrySendError<Enr>> {
        self.peers.try_send(enr)
    }

    /// Sends a change and waits until the actor has published the state that follows it.
    async fn apply(&self, message: impl FnOnce(Applied) -> Message) -> Result<(), HandleError> {
        let (applied_tx, applied_rx) = oneshot::channel();
        self.messages.send(message(applied_tx)).await.map_err(|_| HandleError)?;
        applied_rx.await.map_err(|_| HandleError)
    }
}

/// The network actor stopped before applying a change.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Error)]
#[error("The network actor stopped")]
pub struct HandleError;

#[cfg(test)]
mod tests {
    use super::*;

    fn enr() -> Enr {
        let key = enr::CombinedKey::generate_secp256k1();
        Enr::builder().build(&key).unwrap()
    }

    #[tokio::test]
    async fn changes_wait_for_the_actor_and_report_a_stopped_actor() {
        let (_state_tx, state) = watch::channel(Arc::new(crate::network::actor::tests::state()));
        let (messages, mut receiver) = mpsc::channel(1);
        let (peers, _peers) = mpsc::channel(1);
        let handle = Handle::new(state, messages, peers);
        let id = PeerId::random();

        let (result, ()) = tokio::join!(handle.block_peer(id), async {
            let Message::BlockPeer(blocked, applied) = receiver.recv().await.unwrap() else {
                panic!("expected a peer block");
            };
            assert_eq!(blocked, id);
            applied.send(()).unwrap();
        });
        assert_eq!(result, Ok(()));

        // The actor dropped the change without applying it.
        let (result, ()) = tokio::join!(handle.protect_peer(id), async {
            drop(receiver.recv().await.unwrap());
        });
        assert_eq!(result, Err(HandleError));

        drop(receiver);
        assert_eq!(handle.disconnect_peer(id).await, Err(HandleError));
    }

    #[tokio::test]
    async fn dials_without_waiting_for_room() {
        let (_state_tx, state) = watch::channel(Arc::new(crate::network::actor::tests::state()));
        let (messages, _messages) = mpsc::channel(1);
        let (peers, mut receiver) = mpsc::channel(1);
        let handle = Handle::new(state, messages, peers);
        let (first, second) = (enr(), enr());

        handle.dial(first.clone()).unwrap();
        assert!(
            matches!(handle.dial(second.clone()), Err(TrySendError::Full(enr)) if enr == second)
        );
        assert_eq!(receiver.recv().await.unwrap(), first);
        drop(receiver);
        assert!(matches!(handle.dial(second), Err(TrySendError::Closed(_))));
    }
}

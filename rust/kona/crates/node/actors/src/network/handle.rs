use crate::signer;
use discv5::Enr;
use ipnet::IpNet;
use kona_gossip::GossipState;
use libp2p::{Multiaddr, PeerId};
use std::{net::IpAddr, sync::Arc};
use thiserror::Error;
use tokio::sync::{
    mpsc::{
        self,
        error::{SendError, TrySendError},
    },
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

/// The network actor's published gossip state, and its queues of messages, peers to dial, and
/// signed payloads to publish.
#[derive(Debug, Clone)]
pub struct Handle {
    state: watch::Receiver<Arc<GossipState>>,
    messages: mpsc::Sender<Message>,
    peers: mpsc::Sender<Enr>,
    payloads: mpsc::Sender<signer::Payload>,
}

impl Handle {
    /// Constructs a handle from the actor's published state and its senders.
    pub(super) const fn new(
        state: watch::Receiver<Arc<GossipState>>,
        messages: mpsc::Sender<Message>,
        peers: mpsc::Sender<Enr>,
        payloads: mpsc::Sender<signer::Payload>,
    ) -> Self {
        Self { state, messages, peers, payloads }
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

    /// Queues a signed payload to publish, waiting for room in the queue.
    ///
    /// Success means the payload was queued, not that it was published.
    pub async fn publish(
        &self,
        payload: signer::Payload,
    ) -> Result<(), SendError<signer::Payload>> {
        self.payloads.send(payload).await
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
        let (payloads, _payloads) = mpsc::channel(1);
        let handle = Handle::new(state, messages, peers, payloads);
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
        let (payloads, _payloads) = mpsc::channel(1);
        let handle = Handle::new(state, messages, peers, payloads);
        let (first, second) = (enr(), enr());

        handle.dial(first.clone()).unwrap();
        assert!(
            matches!(handle.dial(second.clone()), Err(TrySendError::Full(enr)) if enr == second)
        );
        assert_eq!(receiver.recv().await.unwrap(), first);
        drop(receiver);
        assert!(matches!(handle.dial(second), Err(TrySendError::Closed(_))));
    }

    #[tokio::test]
    async fn publishing_waits_for_room_and_returns_the_payload_once_the_actor_stops() {
        let (_state_tx, state) = watch::channel(Arc::new(crate::network::actor::tests::state()));
        let (messages, _messages) = mpsc::channel(1);
        let (peers, _peers) = mpsc::channel(1);
        let (payloads, mut receiver) = mpsc::channel(1);
        let handle = Handle::new(state, messages, peers, payloads);
        let payload = |number| {
            let mut block = alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope>::default();
            block.header.number = number;
            signer::Payload {
                payload: op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope::V1(
                    alloy_rpc_types_engine::ExecutionPayloadV1::from_block_slow(&block),
                ),
                signature: alloy_primitives::Signature::test_signature(),
            }
        };

        handle.publish(payload(1)).await.unwrap();
        let second = handle.publish(payload(2));
        tokio::pin!(second);
        tokio::select! {
            biased;
            result = &mut second => panic!("a full queue accepted a payload: {result:?}"),
            _ = std::future::ready(()) => {}
        }
        assert_eq!(receiver.recv().await.unwrap(), payload(1));
        second.await.unwrap();
        drop(receiver);
        assert_eq!(handle.publish(payload(3)).await.unwrap_err().0, payload(3));
    }
}

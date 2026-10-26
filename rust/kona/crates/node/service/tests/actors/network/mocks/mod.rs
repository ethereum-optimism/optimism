//! Tests interactions with sequencer actor's inputs channels.

use std::{str::FromStr, time::Duration};

use backon::{ExponentialBuilder, Retryable};
use discv5::Enr;
use kona_gossip::{P2pRpcRequest, PeerDump, PeerInfo};
use kona_node_service::NetworkActorError;
use kona_rpc::NetworkAdminQuery;
use libp2p::Multiaddr;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use tokio::{
    sync::{mpsc, oneshot},
    task::JoinHandle,
};

pub(crate) mod builder;

/// One node's network stack: a [`NetworkActor`](kona_node_service::NetworkActor) and a
/// [`SignerActor`](kona_node_service::SignerActor) running on their own tasks, plus the channels a
/// test uses to drive and observe them.
///
/// It is one peer in the test's p2p network, not the network itself.
pub(crate) struct TestNetwork {
    /// Sends p2p RPC requests, the ones behind `opp2p_*`, to the network actor.
    pub(super) p2p_rpc_tx: mpsc::Sender<P2pRpcRequest>,
    /// Sends admin queries, such as posting an unsafe payload, to the network actor.
    #[allow(dead_code)]
    pub(super) admin_rpc_tx: mpsc::Sender<NetworkAdminQuery>,
    /// Payloads sent here are signed with this node's key and gossiped.
    pub(super) gossip_payload_tx: mpsc::Sender<OpExecutionPayloadEnvelope>,
    /// Receives the unsafe blocks this node accepts from gossip.
    pub(super) blocks_rx: mpsc::Receiver<OpExecutionPayloadEnvelope>,
    /// The network actor's task.
    #[allow(dead_code)]
    handle: JoinHandle<Result<(), NetworkActorError>>,
}

/// An error from querying a [`TestNetwork`] through its p2p RPC channel.
#[derive(Debug, thiserror::Error)]
pub(crate) enum TestNetworkError {
    #[error("P2p receiver closed")]
    P2pReceiverClosed,
    #[error("P2p receiver closed before sending response: {0}")]
    OneshotError(#[from] oneshot::error::RecvError),
    #[error("Peer info missing ENR")]
    PeerInfoMissingEnr,
    #[error("Invalid ENR: {0}")]
    InvalidEnr(String),
    #[error("Peer info missing listen address")]
    PeerInfoMissingAddress,
    #[error("Invalid multiaddr: {0}")]
    InvalidMultiaddr(String),
    /// The other node is not among this node's connected peers yet.
    #[error("Peer not connected")]
    PeerNotConnected,
}

impl TestNetwork {
    /// Returns this node's own peer info, as `opp2p_self` does.
    pub(super) async fn peer_info(&self) -> Result<PeerInfo, TestNetworkError> {
        // Try to get the peer info. Send a peer info request to the network actor.
        let (peer_info_tx, peer_info_rx) = oneshot::channel();
        let peer_info_request = P2pRpcRequest::PeerInfo(peer_info_tx);
        self.p2p_rpc_tx
            .send(peer_info_request)
            .await
            .map_err(|_| TestNetworkError::P2pReceiverClosed)?;

        let info = peer_info_rx.await?;

        Ok(info)
    }

    /// Returns this node's connected peers, as `opp2p_peers` does.
    pub(super) async fn peers(&self) -> Result<PeerDump, TestNetworkError> {
        let (peers_tx, peers_rx) = oneshot::channel();
        let peers_request = P2pRpcRequest::Peers { out: peers_tx, connected: true };
        self.p2p_rpc_tx
            .send(peers_request)
            .await
            .map_err(|_| TestNetworkError::P2pReceiverClosed)?;
        let peers = peers_rx.await?;
        Ok(peers)
    }

    /// Checks whether `other` is among this node's connected peers.
    pub(super) async fn is_connected_to(&self, other: &Self) -> Result<(), TestNetworkError> {
        let other_peer_id = other.peer_id().await?;
        let peers = self.peers().await?;
        if !peers.peers.contains_key(&other_peer_id) {
            return Err(TestNetworkError::PeerNotConnected);
        }
        Ok(())
    }

    /// Like `is_connected_to`, but retries a couple of times until the connection is established.
    pub(super) async fn is_connected_to_with_retries(
        &self,
        other: &Self,
    ) -> Result<(), TestNetworkError> {
        (async || self.is_connected_to(other).await)
            .retry(ExponentialBuilder::default().with_total_delay(Some(Duration::from_secs(360))))
            // When to retry
            .when(|e| matches!(e, TestNetworkError::PeerNotConnected))
            .notify(|e, duration| tracing::info!(target: "network", "Retrying connection. Error: {e:?}, duration: {duration:?}"))
            .await
    }

    /// Returns the first address this node listens on, ending in its `/p2p/<peer id>`.
    pub(super) async fn listen_addr(&self) -> Result<Multiaddr, TestNetworkError> {
        let info = self.peer_info().await?;
        let addr = info.addresses.first().ok_or(TestNetworkError::PeerInfoMissingAddress)?;
        addr.parse().map_err(|e: libp2p::multiaddr::Error| {
            TestNetworkError::InvalidMultiaddr(e.to_string())
        })
    }

    /// Asks this node to dial `other` at its listen address, the request `opp2p_connectPeer` sends.
    /// Use [`Self::is_connected_to_with_retries`] to wait for the connection.
    pub(super) async fn connect_to(&self, other: &Self) -> Result<(), TestNetworkError> {
        let address = other.listen_addr().await?;
        self.p2p_rpc_tx
            .send(P2pRpcRequest::ConnectPeer { address })
            .await
            .map_err(|_| TestNetworkError::P2pReceiverClosed)
    }

    /// Returns the ENRs in this node's discovery table, as `opp2p_discoveryTable` does.
    pub(super) async fn discovery_table(&self) -> Result<Vec<String>, TestNetworkError> {
        let (table_tx, table_rx) = oneshot::channel();
        self.p2p_rpc_tx
            .send(P2pRpcRequest::DiscoveryTable(table_tx))
            .await
            .map_err(|_| TestNetworkError::P2pReceiverClosed)?;
        Ok(table_rx.await?)
    }

    /// Returns this node's own ENR.
    pub(super) async fn peer_enr(&self) -> Result<Enr, TestNetworkError> {
        let enr = self.peer_info().await?.enr.ok_or(TestNetworkError::PeerInfoMissingEnr)?;
        // Parse the ENR
        let enr = Enr::from_str(&enr).map_err(TestNetworkError::InvalidEnr)?;
        Ok(enr)
    }

    /// Returns this node's own peer ID.
    pub(super) async fn peer_id(&self) -> Result<String, TestNetworkError> {
        Ok(self.peer_info().await?.peer_id)
    }
}

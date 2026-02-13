//! Tests interactions with sequencer actor's inputs channels.

use std::{str::FromStr, time::Duration};

use backon::{ExponentialBuilder, Retryable};
use discv5::Enr;
use kona_gossip::{P2pRpcRequest, PeerDump, PeerInfo};
use kona_node_service::{NetworkActorError, NetworkInboundData};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use tokio::{
    sync::{mpsc, oneshot},
    task::JoinHandle,
};

pub(crate) mod builder;

pub(crate) struct TestNetwork {
    pub(super) inbound_data: NetworkInboundData,
    pub(super) blocks_rx: mpsc::Receiver<OpExecutionPayloadEnvelope>,
    #[allow(dead_code)]
    handle: JoinHandle<Result<(), NetworkActorError>>,
}

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
    #[error("Peer not connected")]
    PeerNotConnected,
}

impl TestNetwork {
    pub(super) async fn peer_info(&self) -> Result<PeerInfo, TestNetworkError> {
        // Try to get the peer info. Send a peer info request to the network actor.
        let (peer_info_tx, peer_info_rx) = oneshot::channel();
        let peer_info_request = P2pRpcRequest::PeerInfo(peer_info_tx);
        self.inbound_data
            .p2p_rpc
            .send(peer_info_request)
            .await
            .map_err(|_| TestNetworkError::P2pReceiverClosed)?;

        let info = peer_info_rx.await?;

        Ok(info)
    }

    pub(super) async fn peers(&self) -> Result<PeerDump, TestNetworkError> {
        let (peers_tx, peers_rx) = oneshot::channel();
        let peers_request = P2pRpcRequest::Peers { out: peers_tx, connected: true };
        self.inbound_data
            .p2p_rpc
            .send(peers_request)
            .await
            .map_err(|_| TestNetworkError::P2pReceiverClosed)?;
        let peers = peers_rx.await?;
        Ok(peers)
    }

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

    pub(super) async fn peer_enr(&self) -> Result<Enr, TestNetworkError> {
        let enr = self.peer_info().await?.enr.ok_or(TestNetworkError::PeerInfoMissingEnr)?;
        // Parse the ENR
        let enr = Enr::from_str(&enr).map_err(TestNetworkError::InvalidEnr)?;
        Ok(enr)
    }

    pub(super) async fn peer_id(&self) -> Result<String, TestNetworkError> {
        Ok(self.peer_info().await?.peer_id)
    }
}

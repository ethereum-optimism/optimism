//! Tests interactions with sequencer actor's inputs channels.

use std::{str::FromStr, time::Duration};

use backon::{ExponentialBuilder, Retryable};
use discv5::Enr;
use kona_gossip::{GossipCommand, GossipCommandSender, PeerDump, PeerInfo};
use kona_node_service::NetworkActorError;
use kona_rpc::{OpP2PApiServer, P2pRpc};
use libp2p::Multiaddr;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use tokio::{sync::mpsc, task::JoinHandle};

pub(crate) mod builder;

/// One node's network stack: a [`NetworkActor`](kona_node_service::NetworkActor) and a
/// [`SignerActor`](kona_node_service::SignerActor) running on their own tasks, plus the channels a
/// test uses to drive and observe them.
///
/// It is one peer in the test's p2p network, not the network itself.
pub(crate) struct TestNetwork {
    /// The node's p2p RPC, the one behind the `opp2p_*` methods.
    pub(super) p2p_rpc: P2pRpc,
    /// Sends gossip commands, such as dialing a peer, to the network actor.
    pub(super) gossip_command_tx: GossipCommandSender,
    /// Sends unsafe payloads to the network actor to gossip, as the admin API does.
    #[allow(dead_code)]
    pub(super) admin_rpc_tx: mpsc::Sender<OpExecutionPayloadEnvelope>,
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
    #[error("P2p RPC failed: {0}")]
    Rpc(#[from] jsonrpsee::types::ErrorObjectOwned),
    #[error("Peer info missing ENR")]
    PeerInfoMissingEnr,
    #[error("Invalid ENR: {0}")]
    InvalidEnr(String),
    #[error("Peer info missing listen address")]
    PeerInfoMissingAddress,
    #[error("Invalid multiaddr: {0}")]
    InvalidMultiaddr(String),
    /// The network actor stopped before applying a gossip command.
    #[error("Gossip command not applied")]
    CommandNotApplied,
    /// The other node is not among this node's connected peers yet.
    #[error("Peer not connected")]
    PeerNotConnected,
    /// The other node is connected, but not subscribed to the block topics yet.
    #[error("Peer not gossiping blocks")]
    PeerNotGossipingBlocks,
}

/// Polls every 10ms at first, so in-memory nodes are ready without waiting, then backs off to once
/// a second for nodes that find each other through discovery, for up to 30 seconds in all.
fn retry_backoff() -> ExponentialBuilder {
    ExponentialBuilder::default()
        .with_min_delay(Duration::from_millis(10))
        .with_max_delay(Duration::from_secs(1))
        .without_max_times()
        .with_total_delay(Some(Duration::from_secs(30)))
}

impl TestNetwork {
    /// Returns this node's own peer info, as `opp2p_self` does.
    pub(super) async fn peer_info(&self) -> Result<PeerInfo, TestNetworkError> {
        Ok(self.p2p_rpc.opp2p_self().await?)
    }

    /// Returns this node's connected peers, as `opp2p_peers` does.
    pub(super) async fn peers(&self) -> Result<PeerDump, TestNetworkError> {
        Ok(self.p2p_rpc.opp2p_peers(true).await?)
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

    /// Like `is_connected_to`, but retries until the connection is established, see
    /// [`retry_backoff`].
    pub(super) async fn is_connected_to_with_retries(
        &self,
        other: &Self,
    ) -> Result<(), TestNetworkError> {
        (async || self.is_connected_to(other).await)
            .retry(retry_backoff())
            // When to retry
            .when(|e| matches!(e, TestNetworkError::PeerNotConnected))
            .notify(|e, duration| tracing::info!(target: "network", "Retrying connection. Error: {e:?}, duration: {duration:?}"))
            .await
    }

    /// Checks whether `other` is a connected peer subscribed to the block topics, so that blocks
    /// this node publishes or relays reach it.
    pub(super) async fn gossips_blocks_with(&self, other: &Self) -> Result<(), TestNetworkError> {
        let other_peer_id = other.peer_id().await?;
        match self.peers().await?.peers.get(&other_peer_id) {
            None => Err(TestNetworkError::PeerNotConnected),
            Some(info) if !info.gossip_blocks => Err(TestNetworkError::PeerNotGossipingBlocks),
            Some(_) => Ok(()),
        }
    }

    /// Like `gossips_blocks_with`, but retries until it holds, see [`retry_backoff`].
    pub(super) async fn gossips_blocks_with_retries(
        &self,
        other: &Self,
    ) -> Result<(), TestNetworkError> {
        (async || self.gossips_blocks_with(other).await)
            .retry(retry_backoff())
            .when(|e| {
                matches!(
                    e,
                    TestNetworkError::PeerNotConnected | TestNetworkError::PeerNotGossipingBlocks
                )
            })
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

    /// Asks this node to dial `other` at its listen address, the command `opp2p_connectPeer`
    /// sends. Use [`Self::is_connected_to_with_retries`] to wait for the connection: unlike
    /// `opp2p_connectPeer`, this does not wait for it.
    pub(super) async fn connect_to(&self, other: &Self) -> Result<(), TestNetworkError> {
        let address = other.listen_addr().await?;
        let (applied_tx, applied_rx) = tokio::sync::oneshot::channel();
        self.gossip_command_tx
            .send((GossipCommand::ConnectPeer { address }, applied_tx))
            .await
            .map_err(|_| TestNetworkError::CommandNotApplied)?;
        applied_rx.await.map_err(|_| TestNetworkError::CommandNotApplied)
    }

    /// Returns the ENRs in this node's discovery table, as `opp2p_discoveryTable` does.
    pub(super) async fn discovery_table(&self) -> Result<Vec<String>, TestNetworkError> {
        Ok(self.p2p_rpc.opp2p_discovery_table().await?)
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

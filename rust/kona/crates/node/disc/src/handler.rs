//! Handler to the [`discv5::Discv5`] service spawned in a thread.

use discv5::{Enr, RequestError, kbucket::NodeStatus, metrics::Metrics};
use enr::NodeId;
use libp2p::{Multiaddr, PeerId};
use std::{collections::HashSet, string::String, time::Duration};
use tokio::sync::{mpsc::Sender, oneshot};

/// Request message for communicating with the Discv5 discovery service.
///
/// These requests are sent from the main application thread to the discovery
/// service running in a separate task, enabling asynchronous operations on
/// the discovery table and peer management.
#[derive(Debug)]
pub enum HandlerRequest {
    /// Request current metrics from the discovery service.
    ///
    /// Returns performance and operational statistics including query counts,
    /// success rates, and table population metrics.
    Metrics(oneshot::Sender<Metrics>),

    /// Get the current number of connected peers in the discovery table.
    ///
    /// Returns the count of peers currently maintained in the routing table,
    /// which indicates the health and connectivity of the discovery service.
    PeerCount(oneshot::Sender<usize>),

    /// Add an ENR to the discovery service's routing table.
    ///
    /// Manually inserts a peer record into the table, typically used for
    /// adding bootstrap nodes or peers discovered through other channels.
    AddEnr(Enr),

    /// Request an ENR from a specific network address.
    ///
    /// Initiates a discovery query to retrieve the ENR for a peer at the
    /// given address. Used for peer verification and metadata retrieval.
    RequestEnr {
        /// Channel to receive the result of the ENR request.
        out: oneshot::Sender<Result<Enr, RequestError>>,
        /// Network address to query for the ENR.
        addr: String,
    },

    /// Get the local node's ENR.
    ///
    /// Returns the ENR that represents this node in the discovery network,
    /// including its network address, capabilities, and cryptographic identity.
    LocalEnr(oneshot::Sender<Enr>),

    /// Get all ENRs currently stored in the routing table.
    ///
    /// Returns a complete dump of peer records known to the discovery service,
    /// useful for debugging and network analysis.
    TableEnrs(oneshot::Sender<Vec<Enr>>),

    /// Get detailed information about nodes in the routing table.
    ///
    /// Returns comprehensive information including node IDs, ENRs, and status
    /// for all peers in the discovery table.
    TableInfos(oneshot::Sender<Vec<(NodeId, Enr, NodeStatus)>>),

    /// Ban peers for a duration, by their libp2p peer IDs.
    ///
    /// Prevents the discovery service from interacting with the peers' nodes, whether or not they
    /// are in its table, useful for blocking malicious or problematic peers.
    BanPeers {
        /// The peers to ban.
        peer_ids: HashSet<PeerId>,
        /// Duration for which the peers should be banned.
        ban_duration: Duration,
    },
}

/// Handler to the spawned [`discv5::Discv5`] service.
///
/// Provides a lock-free way to access the spawned `discv5::Discv5` service
/// by using message-passing to relay requests and responses through
/// a channel.
#[derive(Debug, Clone)]
pub struct Discv5Handler {
    /// Sends [`HandlerRequest`]s to the spawned [`discv5::Discv5`] service.
    pub sender: Sender<HandlerRequest>,
    /// The chain id.
    pub chain_id: u64,
}

/// The discovery service stopped, so it did not answer a request.
///
/// See [`Discv5Driver::start`](crate::Discv5Driver::start) for when it stops.
#[derive(Debug, Clone, Copy, PartialEq, Eq, thiserror::Error)]
#[error("discovery service stopped")]
pub struct DiscoveryStopped;

impl Discv5Handler {
    /// Creates a new [`Discv5Handler`] service.
    pub const fn new(chain_id: u64, sender: Sender<HandlerRequest>) -> Self {
        Self { sender, chain_id }
    }

    /// Returns the ENRs in the discovery table.
    pub async fn table_enrs(&self) -> Result<Vec<Enr>, DiscoveryStopped> {
        self.request(HandlerRequest::TableEnrs).await
    }

    /// Returns the node ID, ENR and status of each node in the discovery table.
    pub async fn table_infos(&self) -> Result<Vec<(NodeId, Enr, NodeStatus)>, DiscoveryStopped> {
        self.request(HandlerRequest::TableInfos).await
    }

    /// Returns the local ENR of the node.
    pub async fn local_enr(&self) -> Result<Enr, DiscoveryStopped> {
        self.request(HandlerRequest::LocalEnr).await
    }

    /// Requests an [`Enr`] from the node at a [`Multiaddr`].
    pub async fn request_enr(
        &self,
        addr: Multiaddr,
    ) -> Result<Result<Enr, RequestError>, DiscoveryStopped> {
        self.request(|out| HandlerRequest::RequestEnr { out, addr: addr.to_string() }).await
    }

    /// Returns the metrics of the discovery service.
    pub async fn metrics(&self) -> Result<Metrics, DiscoveryStopped> {
        self.request(HandlerRequest::Metrics).await
    }

    /// Returns the number of connected peers in the discovery table.
    pub async fn peer_count(&self) -> Result<usize, DiscoveryStopped> {
        self.request(HandlerRequest::PeerCount).await
    }

    /// Sends the request that `make` builds around a response channel, and waits for the response.
    async fn request<T>(
        &self,
        make: impl FnOnce(oneshot::Sender<T>) -> HandlerRequest,
    ) -> Result<T, DiscoveryStopped> {
        let (tx, rx) = oneshot::channel();
        self.sender.send(make(tx)).await.map_err(|_| DiscoveryStopped)?;
        rx.await.map_err(|_| DiscoveryStopped)
    }
}

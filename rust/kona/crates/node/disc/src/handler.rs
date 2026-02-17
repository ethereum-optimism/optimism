//! Handler to the [`discv5::Discv5`] service spawned in a thread.

use discv5::{Enr, RequestError, enr::NodeId, kbucket::NodeStatus, metrics::Metrics};
use libp2p::Multiaddr;
use std::{collections::HashSet, string::String, sync::Arc, time::Duration};
use tokio::sync::mpsc::Sender;

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
    Metrics(tokio::sync::oneshot::Sender<Metrics>),

    /// Get the current number of connected peers in the discovery table.
    ///
    /// Returns the count of peers currently maintained in the routing table,
    /// which indicates the health and connectivity of the discovery service.
    PeerCount(tokio::sync::oneshot::Sender<usize>),

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
        out: tokio::sync::oneshot::Sender<Result<Enr, RequestError>>,
        /// Network address to query for the ENR.
        addr: String,
    },

    /// Get the local node's ENR.
    ///
    /// Returns the ENR that represents this node in the discovery network,
    /// including its network address, capabilities, and cryptographic identity.
    LocalEnr(tokio::sync::oneshot::Sender<Enr>),

    /// Get all ENRs currently stored in the routing table.
    ///
    /// Returns a complete dump of peer records known to the discovery service,
    /// useful for debugging and network analysis.
    TableEnrs(tokio::sync::oneshot::Sender<Vec<Enr>>),

    /// Get detailed information about nodes in the routing table.
    ///
    /// Returns comprehensive information including node IDs, ENRs, and status
    /// for all peers in the discovery table.
    TableInfos(tokio::sync::oneshot::Sender<Vec<(NodeId, Enr, NodeStatus)>>),

    /// Ban specific network addresses for a duration.
    ///
    /// Prevents the discovery service from interacting with the specified
    /// addresses, useful for blocking malicious or problematic peers.
    BanAddrs {
        /// Set of network addresses to ban.
        addrs_to_ban: Arc<HashSet<Multiaddr>>,
        /// Duration for which the addresses should be banned.
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

impl Discv5Handler {
    /// Creates a new [`Discv5Handler`] service.
    pub const fn new(chain_id: u64, sender: Sender<HandlerRequest>) -> Self {
        Self { sender, chain_id }
    }

    /// Blocking request for the ENRs of the discovery service.
    ///
    /// Returns `None` if the request could not be sent or received.
    pub fn table_enrs(&self) -> tokio::sync::oneshot::Receiver<Vec<Enr>> {
        let (tx, rx) = tokio::sync::oneshot::channel();
        let sender = self.sender.clone();
        tokio::spawn(async move {
            if let Err(e) = sender.send(HandlerRequest::TableEnrs(tx)).await {
                warn!(target: "discovery", err = ?e, "Failed to send table ENRs request");
            }
        });
        rx
    }

    /// Returns a [`tokio::sync::oneshot::Receiver`] that contains a vector of information about
    /// the nodes in the discv5 table.
    pub fn table_infos(&self) -> tokio::sync::oneshot::Receiver<Vec<(NodeId, Enr, NodeStatus)>> {
        let (tx, rx) = tokio::sync::oneshot::channel();
        let sender = self.sender.clone();
        tokio::spawn(async move {
            if let Err(e) = sender.send(HandlerRequest::TableInfos(tx)).await {
                warn!(target: "discv5_handler", "Failed to send table infos request: {:?}", e);
            }
        });
        rx
    }

    /// Blocking request for the local ENR of the node.
    ///
    /// Returns `None` if the request could not be sent or received.
    pub fn local_enr(&self) -> tokio::sync::oneshot::Receiver<Enr> {
        let (tx, rx) = tokio::sync::oneshot::channel();
        let sender = self.sender.clone();
        tokio::spawn(async move {
            if let Err(e) = sender.send(HandlerRequest::LocalEnr(tx)).await {
                warn!(target: "discovery", err = ?e, "Failed to send local ENR request");
            }
        });
        rx
    }

    /// Requests an [`Enr`] from the discv5 service given a [`Multiaddr`].
    pub fn request_enr(
        &self,
        addr: Multiaddr,
    ) -> tokio::sync::oneshot::Receiver<Result<Enr, RequestError>> {
        let (tx, rx) = tokio::sync::oneshot::channel();
        let sender = self.sender.clone();
        tokio::spawn(async move {
            if let Err(e) =
                sender.send(HandlerRequest::RequestEnr { out: tx, addr: addr.to_string() }).await
            {
                warn!(target: "discv5_handler", "Failed to send request ENR request: {:?}", e);
            }
        });
        rx
    }

    /// Blocking request for the metrics of the discovery service.
    ///
    /// Returns `None` if the request could not be sent or received.
    pub fn metrics(&self) -> tokio::sync::oneshot::Receiver<Metrics> {
        let (tx, rx) = tokio::sync::oneshot::channel();
        let sender = self.sender.clone();
        tokio::spawn(async move {
            if let Err(e) = sender.send(HandlerRequest::Metrics(tx)).await {
                warn!(target: "discovery", err = ?e, "Failed to send metrics request");
            }
        });
        rx
    }

    /// Blocking request for the discovery service peer count.
    ///
    /// Returns `None` if the request could not be sent or received.
    pub fn peer_count(&self) -> tokio::sync::oneshot::Receiver<usize> {
        let (tx, rx) = tokio::sync::oneshot::channel();
        let sender = self.sender.clone();
        tokio::spawn(async move {
            if let Err(e) = sender.send(HandlerRequest::PeerCount(tx)).await {
                warn!(target: "discovery", err = ?e, "Failed to send peer count request");
            }
        });
        rx
    }
}

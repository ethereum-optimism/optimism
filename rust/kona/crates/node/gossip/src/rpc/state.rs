//! Published gossip state for queries outside the network actor.

use crate::{ConnectionGate, GossipDriver};
use ipnet::IpNet;
use libp2p::{Multiaddr, PeerId, gossipsub::TopicHash, identify::Info};
use std::{
    collections::{HashMap, HashSet},
    net::IpAddr,
    sync::Arc,
    time::Duration,
};
use tokio::sync::{Mutex, watch};

/// Gossip data sampled by the network actor after processing events and commands.
#[derive(Debug)]
pub struct GossipState {
    /// The local gossip peer ID.
    pub local_peer_id: PeerId,
    /// Local listening addresses.
    pub listen_addresses: Vec<Multiaddr>,
    /// Advertised external addresses.
    pub external_addresses: Vec<Multiaddr>,
    /// Known peer identification information.
    pub peerstore: HashMap<PeerId, Info>,
    /// Peers with an established connection.
    pub connected_peers: HashSet<PeerId>,
    /// Current gossip scores of known peers.
    pub peer_scores: HashMap<PeerId, f64>,
    /// Peers subscribed to one of the block topics.
    pub gossip_peers: HashSet<PeerId>,
    /// Number of peers subscribed to each locally subscribed topic.
    pub topic_peer_counts: HashMap<TopicHash, usize>,
    /// The four block topic hashes.
    pub block_topics: [TopicHash; 4],
    /// Blocked peers.
    pub blocked_peers: Vec<PeerId>,
    /// Blocked IP addresses.
    pub blocked_addresses: Vec<IpAddr>,
    /// Blocked subnets.
    pub blocked_subnets: Vec<IpNet>,
    /// Peers protected from disconnection.
    pub protected_peers: Vec<PeerId>,
    /// Ping observations updated by the driver's asynchronous ping handlers.
    pings: Arc<Mutex<HashMap<PeerId, Duration>>>,
}

impl GossipState {
    /// Copies current ping observations without exposing mutable shared state.
    pub async fn pings(&self) -> HashMap<PeerId, Duration> {
        self.pings.lock().await.clone()
    }
}

impl<G: ConnectionGate> GossipDriver<G> {
    /// Samples gossip data without awaiting or sharing ownership of the swarm.
    pub fn snapshot(&self) -> GossipState {
        let gossipsub = &self.swarm.behaviour().gossipsub;
        let block_topics = [
            self.handler.blocks_v1_topic.hash(),
            self.handler.blocks_v2_topic.hash(),
            self.handler.blocks_v3_topic.hash(),
            self.handler.blocks_v4_topic.hash(),
        ];
        let mut topic_peer_counts =
            gossipsub.topics().map(|topic| (topic.clone(), 0)).collect::<HashMap<_, _>>();
        let mut gossip_peers = HashSet::new();
        for (peer, topics) in gossipsub.all_peers() {
            for topic in topics {
                if let Some(count) = topic_peer_counts.get_mut(topic) {
                    *count += 1;
                }
                if block_topics.contains(topic) {
                    gossip_peers.insert(*peer);
                }
            }
        }
        GossipState {
            local_peer_id: *self.local_peer_id(),
            listen_addresses: self.swarm.listeners().cloned().collect(),
            external_addresses: self.swarm.external_addresses().cloned().collect(),
            peerstore: self.peerstore.clone(),
            connected_peers: self.swarm.connected_peers().copied().collect(),
            peer_scores: self
                .peerstore
                .keys()
                .map(|peer| (*peer, gossipsub.peer_score(peer).unwrap_or_default()))
                .collect(),
            gossip_peers,
            topic_peer_counts,
            block_topics,
            blocked_peers: self.connection_gate.list_blocked_peers(),
            blocked_addresses: self.connection_gate.list_blocked_addrs(),
            blocked_subnets: self.connection_gate.list_blocked_subnets(),
            protected_peers: self.connection_gate.list_protected_peers(),
            pings: Arc::clone(&self.ping),
        }
    }
}

/// A cloneable read-only view of the network actor's published gossip state.
#[derive(Debug, Clone)]
pub struct GossipQueryHandle {
    state: watch::Receiver<Arc<GossipState>>,
}

impl GossipQueryHandle {
    /// Wraps the receiver of the network actor's published state.
    pub const fn new(state: watch::Receiver<Arc<GossipState>>) -> Self {
        Self { state }
    }

    /// Returns the latest snapshot, or an error if the network actor has exited.
    pub fn snapshot(&self) -> Result<Arc<GossipState>, watch::error::RecvError> {
        self.state.has_changed()?;
        Ok(Arc::clone(&self.state.borrow()))
    }
}

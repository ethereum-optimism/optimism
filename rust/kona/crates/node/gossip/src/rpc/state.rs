//! Published gossip state for queries outside the network actor.

use crate::{ConnectionGate, Event, GossipDriver};
use ipnet::IpNet;
use libp2p::{
    Multiaddr, PeerId,
    gossipsub::{self, TopicHash},
    identify::{self, Info},
    swarm::SwarmEvent,
};
use std::{
    collections::{HashMap, HashSet},
    net::IpAddr,
    sync::Arc,
    time::Duration,
};
use tokio::{
    sync::{
        mpsc::{self, error::TrySendError},
        watch,
    },
    time::Instant,
};
use tracing::debug;

/// How long a reader waits for the network actor to publish a snapshot it asked for, before it
/// answers from the last one. The actor publishes after the step it is in, which a slow command,
/// such as a dial that resolves a DNS name, can hold up.
const REFRESH_TIMEOUT: Duration = Duration::from_secs(2);

/// Gossip data the network actor samples from its driver, after each command and when a
/// [`GossipQueryHandle`] asks for fresher data.
#[derive(Debug)]
pub struct GossipState {
    /// When the network actor built this snapshot.
    pub built_at: Instant,
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
    /// The latest ping round-trip time to each connected peer.
    pub pings: HashMap<PeerId, Duration>,
}

impl GossipState {
    /// Returns whether `event` changes what a snapshot shows: the connected peers, the peerstore,
    /// the topic subscriptions, or the node's addresses. The network actor then publishes a new
    /// snapshot soon, rather than leaving readers to find theirs stale.
    ///
    /// An event missing here only costs freshness, since readers ask for a new snapshot once
    /// theirs is older than they accept. Scores and pings change without an event, so readers see
    /// them up to that age anyway.
    pub fn changed_by(event: &SwarmEvent<Event>) -> bool {
        match event {
            SwarmEvent::Behaviour(Event::Identify(event)) => {
                matches!(**event, identify::Event::Received { .. })
            }
            SwarmEvent::Behaviour(Event::Gossipsub(event)) => matches!(
                **event,
                gossipsub::Event::Subscribed { .. } | gossipsub::Event::Unsubscribed { .. }
            ),
            event => matches!(
                event,
                SwarmEvent::ConnectionEstablished { .. } |
                    SwarmEvent::ConnectionClosed { .. } |
                    SwarmEvent::NewListenAddr { .. } |
                    SwarmEvent::ExpiredListenAddr { .. } |
                    SwarmEvent::ListenerClosed { .. } |
                    SwarmEvent::ExternalAddrConfirmed { .. } |
                    SwarmEvent::ExternalAddrExpired { .. }
            ),
        }
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
            built_at: Instant::now(),
            local_peer_id: *self.local_peer_id(),
            listen_addresses: self.swarm.listeners().cloned().collect(),
            external_addresses: self.swarm.external_addresses().cloned().collect(),
            peerstore: self.peerstore.clone(),
            connected_peers: self.swarm.connected_peers().copied().collect(),
            peer_scores: self.peerstore.keys().map(|peer| (*peer, self.peer_score(peer))).collect(),
            gossip_peers,
            topic_peer_counts,
            block_topics,
            blocked_peers: self.list_blocked_peers(),
            blocked_addresses: self.list_blocked_addrs(),
            blocked_subnets: self.list_blocked_subnets(),
            protected_peers: self.connection_gate.list_protected_peers(),
            pings: self.ping.clone(),
        }
    }
}

/// A cloneable read-only view of the network actor's gossip state.
///
/// The network actor doesn't rebuild the [`GossipState`] for every network event. It publishes
/// one after each command, so that a read after a command sees its effect, and when a handle
/// asks for one fresher than the last. Handles that ask at the same time share one build.
#[derive(Debug, Clone)]
pub struct GossipQueryHandle {
    /// The latest snapshot the network actor published.
    state: watch::Receiver<Arc<GossipState>>,
    /// Asks the network actor to publish a new snapshot. Its capacity is one, so a request
    /// already waiting stands for every handle that asks before the actor takes it.
    refresh: mpsc::Sender<()>,
}

/// The network actor has stopped, so there is no gossip state to read.
#[derive(Debug, Clone, Copy, thiserror::Error)]
#[error("the network actor has stopped")]
pub struct NetworkActorStopped;

impl GossipQueryHandle {
    /// Wraps the receiver of the network actor's published state, and the sender that asks it
    /// to publish a new snapshot.
    pub const fn new(state: watch::Receiver<Arc<GossipState>>, refresh: mpsc::Sender<()>) -> Self {
        Self { state, refresh }
    }

    /// Returns a snapshot built at most `max_age` ago. If the latest is older, asks the network
    /// actor for a new one and waits for it, for up to two seconds. If none comes in that
    /// time, returns the latest after all, so that a busy actor delays readers but can't block
    /// them.
    ///
    /// With a zero `max_age`, it waits for a snapshot built after this call.
    pub async fn snapshot(
        &self,
        max_age: Duration,
    ) -> Result<Arc<GossipState>, NetworkActorStopped> {
        let mut state = self.state.clone();
        // An error means the network actor dropped its sender.
        state.has_changed().map_err(|_| NetworkActorStopped)?;
        let latest = Arc::clone(&state.borrow_and_update());
        if !max_age.is_zero() && latest.built_at.elapsed() <= max_age {
            return Ok(latest);
        }

        // A full channel holds a request the actor hasn't taken yet. It publishes after taking
        // it, so after the snapshot above, which answers this call too.
        if self.refresh.try_send(()) == Err(TrySendError::Closed(())) {
            return Err(NetworkActorStopped);
        }
        match tokio::time::timeout(REFRESH_TIMEOUT, state.changed()).await {
            Ok(changed) => changed.map_err(|_| NetworkActorStopped)?,
            Err(_) => debug!(
                target: "gossip",
                age = ?latest.built_at.elapsed(),
                "The network actor didn't publish a new snapshot in time, answering from the last one"
            ),
        }
        Ok(Arc::clone(&state.borrow()))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::builder::tests::memory_driver;

    /// A handle on a published snapshot, and the actor's ends of the channels.
    fn handle() -> (GossipQueryHandle, watch::Sender<Arc<GossipState>>, mpsc::Receiver<()>) {
        let (state_tx, state_rx) = watch::channel(Arc::new(memory_driver().snapshot()));
        let (refresh_tx, refresh_rx) = mpsc::channel(1);
        (GossipQueryHandle::new(state_rx, refresh_tx), state_tx, refresh_rx)
    }

    /// Publishes a new snapshot, as the actor does when asked.
    fn publish(state_tx: &watch::Sender<Arc<GossipState>>) -> Arc<GossipState> {
        let state = Arc::new(memory_driver().snapshot());
        state_tx.send_replace(Arc::clone(&state));
        state
    }

    /// A snapshot younger than `max_age` is returned as it is, without asking the actor.
    #[tokio::test(start_paused = true)]
    async fn test_fresh_snapshot_is_read_without_the_actor() {
        let (handle, state_tx, mut refresh_rx) = handle();
        let published = Arc::clone(&state_tx.borrow());
        tokio::time::advance(Duration::from_millis(500)).await;

        let snapshot = handle.snapshot(Duration::from_secs(1)).await.expect("snapshot");
        assert!(Arc::ptr_eq(&snapshot, &published));
        assert!(refresh_rx.try_recv().is_err());
    }

    /// A snapshot older than `max_age` makes the reader ask for a new one, and wait for it.
    #[tokio::test(start_paused = true)]
    async fn test_stale_snapshot_waits_for_a_new_one() {
        let (handle, state_tx, mut refresh_rx) = handle();
        tokio::time::advance(Duration::from_secs(2)).await;

        let read = tokio::spawn(async move { handle.snapshot(Duration::from_secs(1)).await });
        refresh_rx.recv().await.expect("a refresh request");
        let published = publish(&state_tx);

        let snapshot = read.await.expect("reader").expect("snapshot");
        assert!(Arc::ptr_eq(&snapshot, &published));
    }

    /// Readers that find a stale snapshot at the same time send one request, and share the
    /// snapshot built for it.
    #[tokio::test(start_paused = true)]
    async fn test_concurrent_stale_reads_share_one_snapshot() {
        let (handle, state_tx, mut refresh_rx) = handle();
        tokio::time::advance(Duration::from_secs(2)).await;

        // Poll each read once: after that, each has asked for a refresh, or found one already
        // asked for, and waits for the snapshot.
        let mut reads: Vec<_> =
            (0..10).map(|_| Box::pin(handle.snapshot(Duration::from_secs(1)))).collect();
        for read in &mut reads {
            assert!(futures::poll!(read.as_mut()).is_pending());
        }
        refresh_rx.try_recv().expect("one refresh request");
        assert!(refresh_rx.try_recv().is_err(), "only one refresh request");
        let published = publish(&state_tx);

        for read in reads {
            assert!(Arc::ptr_eq(&read.await.expect("snapshot"), &published));
        }
    }

    /// With a zero `max_age`, even a snapshot from this instant is not enough: the reader waits
    /// for one published after it asked.
    #[tokio::test(start_paused = true)]
    async fn test_zero_max_age_waits_for_a_snapshot_after_the_call() {
        let (handle, state_tx, mut refresh_rx) = handle();
        let published = Arc::clone(&state_tx.borrow());

        let read = tokio::spawn(async move { handle.snapshot(Duration::ZERO).await });
        refresh_rx.recv().await.expect("a refresh request");
        let newer = publish(&state_tx);

        let snapshot = read.await.expect("reader").expect("snapshot");
        assert!(!Arc::ptr_eq(&snapshot, &published));
        assert!(Arc::ptr_eq(&snapshot, &newer));
    }

    /// If the actor doesn't answer a refresh request in time, the reader gets the last snapshot.
    #[tokio::test(start_paused = true)]
    async fn test_busy_actor_delays_reads_by_the_refresh_timeout() {
        let (handle, state_tx, _refresh_rx) = handle();
        let published = Arc::clone(&state_tx.borrow());
        let start = Instant::now();

        let snapshot = handle.snapshot(Duration::ZERO).await.expect("snapshot");
        assert!(Arc::ptr_eq(&snapshot, &published));
        assert_eq!(start.elapsed(), REFRESH_TIMEOUT);
    }

    /// Once the actor has stopped, reads fail rather than return its last snapshot.
    #[tokio::test(start_paused = true)]
    async fn test_reads_fail_after_the_actor_stops() {
        let (handle, state_tx, refresh_rx) = handle();
        drop(state_tx);
        drop(refresh_rx);
        assert!(handle.snapshot(Duration::from_secs(1)).await.is_err());
        assert!(handle.snapshot(Duration::ZERO).await.is_err());
    }
}

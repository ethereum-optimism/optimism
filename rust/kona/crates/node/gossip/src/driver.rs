//! Consensus-layer gossipsub driver for Optimism.

use alloy_primitives::{Address, Signature};
use derive_more::Debug;
use discv5::Enr;
use futures::stream::StreamExt;
use ipnet::IpNet;
use kona_genesis::RollupConfig;
use kona_peers::{EnrValidation, PeerMonitoring, enr_to_multiaddr};
use libp2p::{
    Multiaddr, PeerId, Swarm,
    core::transport::ListenerId,
    gossipsub::{IdentTopic, MessageId},
    swarm::{ConnectionId, SwarmEvent, dial_opts::DialOpts},
};
use libp2p_identity::Keypair;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{collections::HashMap, net::IpAddr, time::Duration};
use tokio::{sync::oneshot, time::Instant};

use crate::{
    Behaviour, BlockHandler, CanonicalIp, CanonicalNet, ConnectPeerError, ConnectionGate,
    ConnectionGater, DialError, Event, GossipDriverBuilder, Handler, PeerAddr, PublishError,
};

mod starter;
pub use starter::GossipDriverStarter;

/// A driver for a [`Swarm`] instance.
///
/// Connects the swarm to the given [`Multiaddr`]
/// and handles events using the [`BlockHandler`].
#[derive(Debug)]
pub struct GossipDriver<G: ConnectionGate> {
    /// The [`Swarm`] instance.
    #[debug(skip)]
    pub(crate) swarm: Swarm<Behaviour>,
    /// The address the swarm listens on.
    addr: Multiaddr,
    /// The settings the driver was built with.
    config: GossipDriverConfig,
    /// The [`BlockHandler`].
    pub(crate) handler: BlockHandler,
    /// A mapping from [`PeerId`] to [`Multiaddr`].
    pub(crate) peerstore: HashMap<PeerId, libp2p::identify::Info>,
    /// Tracks connection start time for peers
    peer_connection_start: HashMap<PeerId, Instant>,
    /// The connection gate.
    connection_gate: G,
    /// The addresses of each peer to dial at startup and keep connected, without the node's own
    /// ID.
    static_peers: HashMap<PeerId, Vec<Multiaddr>>,
    /// When static peers that aren't connected are dialed again.
    ///
    /// Set when the driver starts with static peers, and after each check to one
    /// [`GossipDriverConfig::static_peer_redial_interval`] later. Brought forward to
    /// [`GossipDriverConfig::static_peer_reconnect_delay`] from now when the connection to a
    /// static peer closes.
    next_static_peer_check: Option<Instant>,
    /// The latest ping round-trip time to each connected peer.
    pub(crate) ping: HashMap<PeerId, Duration>,
    /// The replies to `ConnectPeer` commands, told once their peer connects or the dial fails.
    pending_connects: HashMap<PeerId, Vec<oneshot::Sender<Result<(), ConnectPeerError>>>>,
    /// The replies to `DisconnectPeer` commands, told once their peer's last connection closes.
    pending_disconnects: HashMap<PeerId, Vec<oneshot::Sender<()>>>,
    /// The open connections to each peer, used to close duplicate connections.
    connections: HashMap<PeerId, HashMap<ConnectionId, PeerConnection>>,
    /// When the next duplicate connection may be closed, if there is one.
    ///
    /// Set when a duplicate connection is established, and after each check to the earliest
    /// duplicate still in its grace period. `None` until the next duplicate is established.
    next_duplicate_check: Option<Instant>,
    /// The listener [`GossipDriverStarter::start`] added, removed again when the driver is
    /// dropped.
    listener_id: ListenerId,
}

/// Removes the listener, so another driver can listen on the same address.
///
/// Dropping a TCP listener frees its port anyway, but the memory transport only frees one when its
/// listener is removed, so a restarted in-memory node couldn't listen on its address again.
impl<G: ConnectionGate> Drop for GossipDriver<G> {
    fn drop(&mut self) {
        // Does nothing if the listener has already closed.
        self.swarm.remove_listener(self.listener_id);
    }
}

/// How a [`GossipDriver`] behaves once it runs, fixed when it is built.
#[derive(Debug, Clone)]
pub struct GossipDriverConfig {
    /// If set, peers whose score drops below the threshold are banned.
    pub peer_monitoring: Option<PeerMonitoring>,
    /// How often static peers that aren't connected are dialed again.
    pub static_peer_redial_interval: Duration,
    /// How soon a static peer is dialed again after its connection closes.
    pub static_peer_reconnect_delay: Duration,
    /// How long a duplicate connection stays open before it is closed.
    ///
    /// gossipsub sends its subscriptions to a peer only once, on the first connection, through a
    /// queue that all connections to the peer share (`on_connection_established` in
    /// libp2p-gossipsub). Closing a connection while it still carries them loses them, and the
    /// peers then never mesh. Waiting gives them time to arrive.
    pub duplicate_connection_grace: Duration,
}

/// No peer monitoring, and the default intervals and grace period.
impl Default for GossipDriverConfig {
    fn default() -> Self {
        Self {
            peer_monitoring: None,
            static_peer_redial_interval: STATIC_PEER_REDIAL_INTERVAL,
            static_peer_reconnect_delay: STATIC_PEER_RECONNECT_DELAY,
            duplicate_connection_grace: DUPLICATE_CONNECTION_GRACE,
        }
    }
}

/// How long a duplicate connection stays open by default, see
/// [`GossipDriverConfig::duplicate_connection_grace`].
const DUPLICATE_CONNECTION_GRACE: Duration = Duration::from_secs(10);

/// The longest a ban can last: about a century, which every platform's clock can add to now.
const LONGEST_BAN: Duration = Duration::from_secs(100 * 365 * 24 * 60 * 60);

/// How often static peers that aren't connected are dialed again by default, as in op-node.
const STATIC_PEER_REDIAL_INTERVAL: Duration = Duration::from_secs(60);

/// How soon a static peer is dialed again after its connection closes, by default.
///
/// Short, so a peer that restarts is back quickly. Not zero, so a peer that keeps closing the
/// connection right away isn't redialed in a tight loop.
const STATIC_PEER_RECONNECT_DELAY: Duration = Duration::from_secs(1);

/// Adds a reply to tell about `peer_id`, first dropping the ones whose caller has given up, so
/// that callers that keep timing out can't pile replies up while a dial runs.
fn add_reply<T>(
    replies: &mut HashMap<PeerId, Vec<oneshot::Sender<T>>>,
    peer_id: PeerId,
    reply: oneshot::Sender<T>,
) {
    let replies = replies.entry(peer_id).or_default();
    replies.retain(|reply| !reply.is_closed());
    replies.push(reply);
}

/// An open connection to a peer.
#[derive(Debug, Clone)]
pub(crate) struct PeerConnection {
    /// The peer that dialed the connection, this node or the other one.
    dialer: PeerId,
    /// When the connection was established.
    established: Instant,
    /// The peer's address on this connection: the one we dialed, or the one it dialed us from.
    /// Blocking its IP address or subnet closes the connection.
    remote_addr: Multiaddr,
}

impl<G> GossipDriver<G>
where
    G: ConnectionGate,
{
    /// Returns the [`GossipDriverBuilder`] that can be used to construct the [`GossipDriver`].
    pub fn builder(
        rollup_config: RollupConfig,
        signer: Address,
        gossip_addr: Multiaddr,
        keypair: Keypair,
    ) -> GossipDriverBuilder {
        GossipDriverBuilder::new(rollup_config, signer, gossip_addr, keypair)
    }

    /// Returns the settings the driver was built with.
    pub const fn config(&self) -> &GossipDriverConfig {
        &self.config
    }

    /// Publishes an unsafe block to gossip.
    ///
    /// ## Arguments
    ///
    /// * `topic_selector` - A function that selects the topic for the block. This is expected to be
    ///   a closure that takes the [`BlockHandler`] and returns the [`IdentTopic`] for the block.
    /// * `payload` - The payload to be published.
    ///
    /// ## Returns
    ///
    /// Returns the [`MessageId`] of the published message or a [`PublishError`]
    /// if the message could not be published.
    pub fn publish(
        &mut self,
        selector: impl FnOnce(&BlockHandler) -> IdentTopic,
        payload: OpExecutionPayloadEnvelope,
        signature: Signature,
    ) -> Result<MessageId, PublishError> {
        let topic = selector(&self.handler);
        let topic_hash = topic.hash();
        let data = self.handler.encode(topic, payload, signature)?;
        let id = self.swarm.behaviour_mut().gossipsub.publish(topic_hash, data)?;
        metrics::gauge!(crate::Metrics::UNSAFE_BLOCK_PUBLISHED).increment(1);
        Ok(id)
    }

    /// Dials each static peer that isn't connected, at all of its addresses, then schedules the
    /// next check.
    ///
    /// A static peer the operator blocked is skipped quietly, and dialed again once unblocked.
    fn dial_static_peers(&mut self) {
        self.next_static_peer_check = None;
        if self.static_peers.is_empty() {
            return;
        }
        let gate = &self.swarm.behaviour().gate;
        let due: Vec<_> = self
            .static_peers
            .iter()
            .filter(|(peer_id, _)| !self.is_connected(peer_id) && !gate.is_blocked_peer(peer_id))
            .map(|(peer_id, addrs)| (*peer_id, addrs.clone()))
            .collect();
        for (peer_id, addrs) in due {
            debug!(target: "gossip", %peer_id, ?addrs, "Dialing static peer");
            // A refusal is logged in `dial_peer`, and the next check tries again.
            let _ = self.dial_peer(peer_id, addrs);
        }
        self.update_next_static_peer_check(
            Instant::now() + self.config.static_peer_redial_interval,
        );
    }

    /// Makes the next static peer check happen by `at`, keeping an earlier one.
    fn update_next_static_peer_check(&mut self, at: Instant) {
        self.next_static_peer_check =
            Some(self.next_static_peer_check.map_or(at, |check| check.min(at)));
    }

    /// Returns the local peer id.
    pub fn local_peer_id(&self) -> &libp2p::PeerId {
        self.swarm.local_peer_id()
    }

    /// Returns the address the driver listens on, as the swarm reported it.
    pub const fn addr(&self) -> &Multiaddr {
        &self.addr
    }

    /// Returns where peers can dial this node: the address the swarm listens on, ending in this
    /// node's peer ID.
    pub fn peer_addr(&self) -> PeerAddr {
        // The swarm reports listen addresses without a `/p2p`.
        PeerAddr::new(self.addr.clone(), *self.local_peer_id())
            .expect("a listen address is a peer address once it has the peer ID")
    }

    /// Returns the [`BlockHandler`].
    pub const fn handler(&self) -> &BlockHandler {
        &self.handler
    }

    /// Attempts to select the next event from the Swarm.
    ///
    /// When duplicate connections may be closed, it returns [`Event::DuplicateConnectionsDue`]
    /// instead, when static peers may be dialed again, [`Event::StaticPeersDue`], and when the
    /// unsafe block signer changed, [`Event::SignerChanged`], for [`Self::handle_event`] to act on.
    pub async fn next(&mut self) -> Option<SwarmEvent<Event>> {
        let duplicates = self.next_duplicate_check;
        let static_peers = self.next_static_peer_check;
        // `select!` builds the futures of disabled branches too, so each needs some deadline.
        let now = Instant::now;
        tokio::select! {
            // Observe rotations even when no gossip messages arrive. Fixed-address builders
            // drop the sender, so stop polling a closed channel, which would otherwise report a
            // change on every call.
            Ok(()) = self.handler.signer_recv.changed(), if self.handler.signer_recv.has_changed().is_ok() => {
                Some(SwarmEvent::Behaviour(Event::SignerChanged))
            }
            event = self.swarm.next() => event,
            () = tokio::time::sleep_until(duplicates.unwrap_or_else(now)), if duplicates.is_some() => {
                Some(SwarmEvent::Behaviour(Event::DuplicateConnectionsDue))
            }
            () = tokio::time::sleep_until(static_peers.unwrap_or_else(now)), if static_peers.is_some() => {
                Some(SwarmEvent::Behaviour(Event::StaticPeersDue))
            }
        }
    }

    /// Returns the peers the swarm has a connection to.
    pub fn connected_peers(&self) -> impl Iterator<Item = &PeerId> {
        self.swarm.connected_peers()
    }

    /// Returns whether the swarm has a connection to `peer_id`.
    pub fn is_connected(&self, peer_id: &PeerId) -> bool {
        self.swarm.is_connected(peer_id)
    }

    /// Returns the gossipsub score of `peer_id`, or 0 if peer scoring is off or the peer has no
    /// score.
    pub fn peer_score(&self, peer_id: &PeerId) -> f64 {
        self.swarm.behaviour().gossipsub.peer_score(peer_id).unwrap_or_default()
    }

    /// Closes every connection to `peer_id`.
    ///
    /// The swarm reports each closed connection, and [`Self::handle_event`] records the connection
    /// duration once the last one closes.
    pub fn disconnect_peer(&mut self, peer_id: PeerId) {
        if self.swarm.disconnect_peer_id(peer_id).is_err() {
            warn!(target: "gossip", %peer_id, "Cannot disconnect a peer that isn't connected");
        }
    }

    /// Dials `peer`, and tells `reply` once the peer connects or the dial fails.
    ///
    /// A peer that is already connected is told at once, and so is one the gate refuses to dial,
    /// with the reason. A peer that is already being dialed is told how that dial ends.
    ///
    /// A peer counts as connected until its connections have closed, so connecting a peer just
    /// blocked, while its connections close, is told it is connected.
    pub fn connect_peer(
        &mut self,
        peer: PeerAddr,
        reply: oneshot::Sender<Result<(), ConnectPeerError>>,
    ) {
        let peer_id = peer.peer_id();
        match self.dial_peer(peer_id, [peer.into()]) {
            Ok(()) | Err(DialError::AlreadyDialing { .. }) => {
                add_reply(&mut self.pending_connects, peer_id, reply);
            }
            Err(DialError::AlreadyConnected { .. }) => {
                let _ = reply.send(Ok(()));
            }
            Err(refusal) => {
                let _ = reply.send(Err(refusal.into()));
            }
        }
    }

    /// Disconnects `peer_id`, and tells `reply` once its last connection closes. A peer that
    /// isn't connected is told at once.
    pub fn disconnect_peer_with_reply(&mut self, peer_id: PeerId, reply: oneshot::Sender<()>) {
        if !self.is_connected(&peer_id) {
            let _ = reply.send(());
            return;
        }
        add_reply(&mut self.pending_disconnects, peer_id, reply);
        self.disconnect_peer(peer_id);
    }

    /// Tells the `ConnectPeer` commands waiting for `peer_id` the outcome of its dial.
    fn reply_to_connects(&mut self, peer_id: &PeerId, outcome: Result<(), ConnectPeerError>) {
        for reply in self.pending_connects.remove(peer_id).into_iter().flatten() {
            let _ = reply.send(outcome.clone());
        }
    }

    /// Bans a peer for its score: disconnects it, and refuses its connections in both directions
    /// for [`PeerMonitoring::ban_duration`]. Discovery bans the peer separately, by its ID.
    ///
    /// What identify told about the peer stays in the peerstore, as for any disconnected peer, so
    /// that the RPC still lists it among the known peers, as op-node does.
    pub fn ban_peer(&mut self, peer_id: PeerId) {
        // The score that got the peer banned, read before the peer is disconnected.
        let score = self.peer_score(&peer_id);
        metrics::gauge!(
            crate::Metrics::BANNED_PEERS,
            "peer_id" => peer_id.to_string(),
            "score" => score.to_string()
        )
        .increment(1);
        if let Some(monitoring) = &self.config.peer_monitoring {
            // A duration too long for the clock bans the peer for as long as the node runs.
            let now = Instant::now();
            let until = now.checked_add(monitoring.ban_duration).unwrap_or(now + LONGEST_BAN);
            self.swarm.behaviour_mut().gate.ban_peer(peer_id, until);
        }
        self.disconnect_peer(peer_id);
    }

    /// Protects `peer_id` from the redial limit. It can still be banned for its score, unless it is
    /// a static peer.
    pub fn protect_peer(&mut self, peer_id: PeerId) {
        self.connection_gate.protect_peer(peer_id);
    }

    /// Lists the protected peers, the static peers among them.
    pub fn list_protected_peers(&self) -> Vec<PeerId> {
        self.connection_gate.list_protected_peers()
    }

    /// Removes the protection of `peer_id`, unless it is a static peer, which stays protected.
    ///
    /// As in op-node, where unprotecting through the API removes only the API's protection, and
    /// static peers are protected for being static.
    pub fn unprotect_peer(&mut self, peer_id: PeerId) {
        if self.is_static_peer(&peer_id) {
            info!(target: "gossip", %peer_id, "Static peer stays protected");
            return;
        }
        self.connection_gate.unprotect_peer(peer_id);
    }

    /// Blocks `peer_id`: closes its connections and aborts our dials to it, the gate refuses to
    /// dial it and refuses its incoming connections, and gossipsub ignores its messages.
    ///
    /// Its connections are closed even if it is protected or a static peer: protection only
    /// exempts a peer from the redial limit. op-node leaves a blocked peer's open connections in
    /// place.
    pub fn block_peer(&mut self, peer_id: PeerId) {
        let behaviour = self.swarm.behaviour_mut();
        behaviour.gate.block_peer(peer_id);
        behaviour.gossipsub.blacklist_peer(&peer_id);
        // Also aborts our dials still in flight. An error only means the peer isn't connected.
        let _ = self.swarm.disconnect_peer_id(peer_id);
        if self.is_static_peer(&peer_id) {
            warn!(target: "gossip", %peer_id, "Blocked a static peer: it isn't dialed until unblocked");
        }
    }

    /// Unblocks `peer_id`, undoing [`Self::block_peer`]. A static peer is dialed again at the next
    /// static peer check.
    pub fn unblock_peer(&mut self, peer_id: PeerId) {
        let behaviour = self.swarm.behaviour_mut();
        behaviour.gate.unblock_peer(&peer_id);
        behaviour.gossipsub.remove_blacklisted_peer(&peer_id);
    }

    /// Lists the blocked peers.
    pub fn list_blocked_peers(&self) -> Vec<PeerId> {
        self.swarm.behaviour().gate.list_blocked_peers()
    }

    /// Blocks an IP address: closes the connections to and from it, and the gate refuses to dial
    /// it and refuses incoming connections from it. An IPv4-mapped IPv6 address blocks the IPv4
    /// address in it.
    pub fn block_addr(&mut self, ip: IpAddr) {
        let ip = CanonicalIp::from(ip);
        self.swarm.behaviour_mut().gate.block_addr(ip);
        self.close_blocked_connections();
        self.warn_about_blocked_static_peers(self.static_peer_addrs_at(ip));
    }

    /// Unblocks an IP address.
    pub fn unblock_addr(&mut self, ip: IpAddr) {
        self.swarm.behaviour_mut().gate.unblock_addr(&ip.into());
    }

    /// Lists the blocked IP addresses.
    pub fn list_blocked_addrs(&self) -> Vec<CanonicalIp> {
        self.swarm.behaviour().gate.list_blocked_addrs()
    }

    /// Blocks a subnet: closes the connections to and from the IP addresses in it, and the gate
    /// refuses to dial them and refuses incoming connections from them. An IPv4-mapped IPv6
    /// subnet blocks the IPv4 subnet in it.
    pub fn block_subnet(&mut self, subnet: IpNet) {
        let subnet = CanonicalNet::from(subnet);
        self.swarm.behaviour_mut().gate.block_subnet(subnet);
        self.close_blocked_connections();
        self.warn_about_blocked_static_peers(self.static_peer_addrs_in(subnet));
    }

    /// Warns about each static peer address that a new block covers, so that an operator sees
    /// when they block a static peer by accident.
    fn warn_about_blocked_static_peers(&self, covered: Vec<(PeerId, Multiaddr)>) {
        for (peer_id, addr) in covered {
            warn!(target: "gossip", %peer_id, %addr, "Blocked an address of a static peer: it isn't dialed there until unblocked");
        }
    }

    /// Returns the static peers' addresses that blocking `ip` covers.
    fn static_peer_addrs_at(&self, ip: CanonicalIp) -> Vec<(PeerId, Multiaddr)> {
        self.static_peer_addrs_covered_by(|addr| *addr == ip)
    }

    /// Returns the static peers' addresses that blocking `subnet` covers.
    fn static_peer_addrs_in(&self, subnet: CanonicalNet) -> Vec<(PeerId, Multiaddr)> {
        self.static_peer_addrs_covered_by(|ip| subnet.contains(ip))
    }

    /// Returns the static peers' addresses whose IP `covers`: the IP addresses they are configured
    /// with, and the addresses of the connections open to them.
    ///
    /// A static peer configured with a DNS name isn't found unless it is connected, since finding
    /// it would need a DNS lookup.
    fn static_peer_addrs_covered_by(
        &self,
        covers: impl Fn(&CanonicalIp) -> bool,
    ) -> Vec<(PeerId, Multiaddr)> {
        let mut covered = Vec::new();
        for (peer_id, addrs) in &self.static_peers {
            let connections = self.connections.get(peer_id).into_iter().flat_map(HashMap::values);
            let open = connections.map(|connection| &connection.remote_addr);
            for addr in addrs.iter().chain(open) {
                let ip = ConnectionGater::ip_from_addr(addr);
                if ip.is_some_and(|ip| covers(&ip)) && !covered.contains(&(*peer_id, addr.clone()))
                {
                    covered.push((*peer_id, addr.clone()));
                }
            }
        }
        covered
    }

    /// Closes the connections whose remote IP address the gate now refuses, whichever side
    /// dialed them. Connections without an IP, such as in-memory ones, stay open.
    fn close_blocked_connections(&mut self) {
        let gate = &self.swarm.behaviour().gate;
        let blocked: Vec<_> = self
            .connections
            .values()
            .flatten()
            .filter(|(_, connection)| {
                ConnectionGater::ip_from_addr(&connection.remote_addr)
                    .is_some_and(|ip| gate.check_ip(ip).is_err())
            })
            .map(|(connection_id, connection)| (*connection_id, connection.remote_addr.clone()))
            .collect();
        for (connection_id, remote_addr) in blocked {
            debug!(target: "gossip", ?connection_id, %remote_addr, "Closing a connection to a blocked address");
            self.swarm.close_connection(connection_id);
        }
    }

    /// Unblocks a subnet.
    pub fn unblock_subnet(&mut self, subnet: IpNet) {
        self.swarm.behaviour_mut().gate.unblock_subnet(&subnet.into());
    }

    /// Lists the blocked subnets.
    pub fn list_blocked_subnets(&self) -> Vec<CanonicalNet> {
        self.swarm.behaviour().gate.list_blocked_subnets()
    }

    /// Returns whether `peer_id` is one of the static peers.
    pub fn is_static_peer(&self, peer_id: &PeerId) -> bool {
        self.static_peers.contains_key(peer_id)
    }

    /// Returns the connected peers whose gossipsub score is below the ban threshold of
    /// [`GossipDriverConfig::peer_monitoring`], or none if it isn't set.
    ///
    /// Static peers are never banned, as in op-node: they would be dialed again right away.
    pub fn peers_to_ban(&self) -> Vec<PeerId> {
        let Some(monitoring) = self.config.peer_monitoring.as_ref() else {
            return Vec::new();
        };
        self.swarm
            .connected_peers()
            .filter(|peer_id| {
                self.peer_score(peer_id) < monitoring.ban_threshold && !self.is_static_peer(peer_id)
            })
            .copied()
            .collect()
    }

    /// Returns the address to dial the peer that an [`Enr`] from discovery describes, for
    /// [`Self::dial_peer`].
    ///
    /// Returns `None` for an ENR of another OP Stack chain, or one without a TCP socket.
    pub fn enr_to_peer_addr(&self, enr: &Enr) -> Option<PeerAddr> {
        let chain_id = self.handler.rollup_config.l2_chain_id.id();
        let validation = EnrValidation::validate(enr, chain_id);
        if validation.is_invalid() {
            trace!(target: "gossip", "Invalid OP Stack ENR for chain id {chain_id}: {validation}");
            return None;
        }
        let Some(addr) = enr_to_multiaddr(enr) else {
            debug!(target: "gossip", "Failed to extract tcp socket from enr: {:?}", enr);
            metrics::counter!(crate::Metrics::DIAL_PEER_ERROR, "type" => "invalid_enr")
                .increment(1);
            return None;
        };
        // `enr_to_multiaddr` ends the address in the peer ID.
        PeerAddr::try_from(addr).ok()
    }

    /// Dials the given [`Multiaddr`], which must end in the peer's `/p2p/<peer id>`. See
    /// [`Self::dial_peer`] for the result.
    pub fn dial_multiaddr(&mut self, addr: impl Into<Multiaddr>) -> Result<(), DialError> {
        let addr = addr.into();
        let Some(peer_id) = ConnectionGater::peer_id_from_addr(&addr) else {
            warn!(target: "gossip", peer=?addr, "Failed to extract PeerId from Multiaddr");
            metrics::counter!(crate::Metrics::DIAL_PEER_ERROR, "type" => "invalid_multiaddr")
                .increment(1);
            return Err(DialError::InvalidMultiaddr { addr });
        };
        self.dial_peer(peer_id, [addr])
    }

    /// Dials `peer_id` at whichever of `addrs` connects first.
    ///
    /// The gate checks the peer first, and then each address, which must end in `/p2p/<peer_id>`:
    /// the addresses it refuses are left out, and nothing is dialed if it refuses them all. The
    /// rest make a single dial, in which libp2p tries several addresses at once and keeps the
    /// first connection.
    ///
    /// Returns `Ok` once the dial has started, and [`DialError::AlreadyConnected`] if the peer is
    /// already connected. If no address is left, returns the gate's first refusal, or
    /// [`DialError::NoAddresses`] if the gate refused none. Whether the dial then connects comes
    /// later, in [`SwarmEvent::ConnectionEstablished`]
    /// or [`SwarmEvent::OutgoingConnectionError`].
    pub fn dial_peer(
        &mut self,
        peer_id: PeerId,
        addrs: impl IntoIterator<Item = Multiaddr>,
    ) -> Result<(), DialError> {
        // A connected peer is counted as a dial error: a caller, or an operator through
        // `opp2p_connectPeer`, asked to connect a peer we are already connected to. It is checked
        // before the gate, which would refuse it as already dialing or count it as a dial.
        if self.is_connected(&peer_id) {
            debug!(target: "gossip", ?peer_id, "Already connected to peer, not dialing");
            metrics::counter!(crate::Metrics::DIAL_PEER_ERROR, "type" => "already_connected")
                .increment(1);
            return Err(DialError::AlreadyConnected { peer_id });
        }

        let gate = &self.swarm.behaviour().gate;
        self.connection_gate.can_dial_peer(&peer_id, gate)?;

        // A DNS name is resolved here, so that the gate checks the IPs libp2p then dials, rather
        // than libp2p resolving it again, maybe to an IP the gate didn't see.
        let mut refusal = None;
        let addrs: Vec<Multiaddr> = addrs
            .into_iter()
            .flat_map(|addr| ConnectionGater::resolve_dns_addrs(addr, ConnectionGater::system_resolve))
            .filter(|addr| {
                if ConnectionGater::peer_id_from_addr(addr) != Some(peer_id) {
                    warn!(target: "gossip", ?peer_id, peer=?addr, "Address is not for the peer being dialed");
                    return false;
                }
                if let Err(dial_error) = self.connection_gate.can_dial_addr(&peer_id, addr, gate) {
                    refusal.get_or_insert(dial_error);
                    return false;
                }
                true
            })
            .collect();
        if addrs.is_empty() {
            return Err(refusal.unwrap_or(DialError::NoAddresses { peer_id }));
        }

        // Let the gate know we are dialing the peer.
        self.connection_gate.dialing(&peer_id);

        // Dial from a new port. From the listen port, two peers dialing each other at the same
        // time would share one TCP connection, which fails the handshake, and on macOS could not
        // redial until it leaves `TIME_WAIT`. With new ports, each dial gets its own connection,
        // and the duplicate is closed once both are established.
        let opts = DialOpts::peer_id(peer_id).addresses(addrs.clone()).allocate_new_port().build();
        match self.swarm.dial(opts) {
            Ok(_) => {
                trace!(target: "gossip", ?peer_id, ?addrs, "Dialed peer");
                for addr in &addrs {
                    self.connection_gate.dialed(addr);
                }
                metrics::counter!(crate::Metrics::DIAL_PEER).increment(1);
                Ok(())
            }
            Err(e) => {
                error!(target: "gossip", "Failed to connect to peer: {:?}", e);
                self.connection_gate.remove_dial(&peer_id);
                metrics::counter!(crate::Metrics::DIAL_PEER_ERROR, "type" => "connection_error")
                    .increment(1);
                Err(DialError::SwarmRefused { peer_id, reason: e.to_string() })
            }
        }
    }

    /// Closes duplicate connections whose grace period has passed, so that both peers keep the
    /// same one, and schedules the next check for those still in theirs.
    ///
    /// Two peers that dial each other at the same time, or one that dials while the other is
    /// already connected, get two connections. Both peers close the connection dialed by the one
    /// with the larger [`PeerId`], once a connection dialed by the smaller one exists. That only
    /// depends on what both peers see, so they close the same connection and keep the other.
    /// Connections dialed by the same peer are left open: there is no choice between those that
    /// both peers would agree on.
    ///
    /// A peer's duplicates are closed once its youngest connection has been open for
    /// [`GossipDriverConfig::duplicate_connection_grace`].
    fn close_duplicate_connections(&mut self) {
        self.next_duplicate_check = None;
        let local_peer_id = *self.local_peer_id();
        let now = Instant::now();
        let mut pending = Vec::new();
        let mut duplicates = Vec::new();
        for (peer_id, connections) in &self.connections {
            let preferred_dialer = local_peer_id.min(*peer_id);
            if connections.len() < 2 ||
                !connections.values().any(|connection| connection.dialer == preferred_dialer)
            {
                continue;
            }
            let youngest = connections.values().map(|connection| connection.established).max();
            let Some(closable_at) = youngest.map(|at| at + self.config.duplicate_connection_grace)
            else {
                continue;
            };
            if closable_at > now {
                pending.push(closable_at);
                continue;
            }
            duplicates.extend(
                connections
                    .iter()
                    .filter(|(_, connection)| connection.dialer != preferred_dialer)
                    .map(|(connection_id, _)| (*peer_id, *connection_id)),
            );
        }
        for (peer_id, connection_id) in duplicates {
            debug!(target: "gossip", ?peer_id, ?connection_id, "Closing duplicate connection");
            self.swarm.close_connection(connection_id);
        }
        for at in pending {
            self.update_next_duplicate_check(at);
        }
    }

    /// Moves the next duplicate connection check to `at`, if that is earlier.
    fn update_next_duplicate_check(&mut self, at: Instant) {
        self.next_duplicate_check =
            Some(self.next_duplicate_check.map_or(at, |check| check.min(at)));
    }

    fn handle_gossip_event(&mut self, event: Event) -> Option<OpExecutionPayloadEnvelope> {
        match event {
            Event::Gossipsub(e) => return self.handle_gossipsub_event(*e),
            Event::DuplicateConnectionsDue => self.close_duplicate_connections(),
            Event::StaticPeersDue => self.dial_static_peers(),
            Event::SignerChanged => self.handler.update_signer(),
            Event::Ping(libp2p::ping::Event { peer, result, .. }) => {
                trace!(target: "gossip", ?peer, ?result, "Ping received");

                // If the peer is connected to gossip, record the connection duration.
                if let Some(start_time) = self.peer_connection_start.get(&peer) {
                    let _ping_duration = start_time.elapsed();
                    metrics::histogram!(crate::Metrics::GOSSIP_PEER_CONNECTION_DURATION_SECONDS)
                        .record(_ping_duration.as_secs_f64());
                }

                // Record the peer score in the metrics if available.
                if let Some(_peer_score) = self.swarm.behaviour().gossipsub.peer_score(&peer) {
                    metrics::histogram!(crate::Metrics::PEER_SCORES).record(_peer_score);
                }

                if let Ok(time) = result {
                    self.ping.insert(peer, time);
                }
            }
            Event::Identify(e) => self.handle_identify_event(*e),
        };

        None
    }

    fn handle_identify_event(&mut self, event: libp2p::identify::Event) {
        match event {
            libp2p::identify::Event::Received { connection_id, peer_id, info } => {
                debug!(target: "gossip", ?connection_id, ?peer_id, ?info, "Received identify info from peer");
                self.peerstore.insert(peer_id, info);
            }
            libp2p::identify::Event::Sent { connection_id, peer_id } => {
                debug!(target: "gossip", ?connection_id, ?peer_id, "Sent identify info to peer");
            }
            libp2p::identify::Event::Pushed { connection_id, peer_id, info } => {
                debug!(target: "gossip", ?connection_id, ?peer_id, ?info, "Pushed identify info to peer");
            }
            libp2p::identify::Event::Error { connection_id, peer_id, error } => {
                error!(target: "gossip", ?connection_id, ?peer_id, ?error, "Error raised while attempting to identify remote");
            }
        }
    }

    /// Handles a [`libp2p::gossipsub::Event`].
    fn handle_gossipsub_event(
        &mut self,
        event: libp2p::gossipsub::Event,
    ) -> Option<OpExecutionPayloadEnvelope> {
        match event {
            libp2p::gossipsub::Event::Message {
                propagation_source: src,
                message_id: id,
                message,
            } => {
                trace!(target: "gossip", "Received message with topic: {}", message.topic);
                metrics::gauge!(crate::Metrics::GOSSIP_EVENT, "type" => "message", "topic" => message.topic.to_string()).increment(1);
                if self.handler.topics().contains(&message.topic) {
                    let (status, payload) = self.handler.handle(message);
                    _ = self
                        .swarm
                        .behaviour_mut()
                        .gossipsub
                        .report_message_validation_result(&id, &src, status);
                    return payload;
                }
            }
            libp2p::gossipsub::Event::Subscribed { peer_id, topic } => {
                trace!(target: "gossip", "Peer: {:?} subscribed to topic: {:?}", peer_id, topic);
                metrics::gauge!(crate::Metrics::GOSSIP_EVENT, "type" => "subscribed", "topic" => topic.to_string()).increment(1);
            }
            libp2p::gossipsub::Event::Unsubscribed { peer_id, topic } => {
                trace!(target: "gossip", "Peer: {:?} unsubscribed from topic: {:?}", peer_id, topic);
                metrics::gauge!(crate::Metrics::GOSSIP_EVENT, "type" => "unsubscribed", "topic" => topic.to_string()).increment(1);
            }
            libp2p::gossipsub::Event::SlowPeer { peer_id, .. } => {
                trace!(target: "gossip", "Slow peer: {:?}", peer_id);
                metrics::gauge!(crate::Metrics::GOSSIP_EVENT, "type" => "slow_peer").increment(1);
            }
            libp2p::gossipsub::Event::GossipsubNotSupported { peer_id } => {
                trace!(target: "gossip", "Peer: {:?} does not support gossipsub", peer_id);
                metrics::gauge!(crate::Metrics::GOSSIP_EVENT, "type" => "not_supported")
                    .increment(1);
            }
        }
        None
    }

    /// Handles the [`SwarmEvent<Event>`].
    pub fn handle_event(&mut self, event: SwarmEvent<Event>) -> Option<OpExecutionPayloadEnvelope> {
        match event {
            SwarmEvent::Behaviour(behavior_event) => {
                return self.handle_gossip_event(behavior_event);
            }
            SwarmEvent::ConnectionEstablished {
                peer_id,
                connection_id,
                endpoint,
                num_established,
                ..
            } => {
                let dialer = if endpoint.is_dialer() {
                    // Our dial succeeded, so it is no longer in flight. An inbound connection
                    // leaves a dial of ours alone: it still ends in its own event.
                    self.connection_gate.remove_dial(&peer_id);
                    *self.local_peer_id()
                } else {
                    peer_id
                };
                let established = Instant::now();
                let remote_addr = endpoint.get_remote_address().clone();
                let connection = PeerConnection { dialer, established, remote_addr };
                self.connections.entry(peer_id).or_default().insert(connection_id, connection);
                if num_established.get() > 1 {
                    self.update_next_duplicate_check(
                        established + self.config.duplicate_connection_grace,
                    );
                }

                let peer_count = self.swarm.connected_peers().count();
                debug!(target: "gossip", "Connection established: {:?} | Peer Count: {}", peer_id, peer_count);
                metrics::counter!(crate::Metrics::GOSSIPSUB_CONNECTION, "type" => "connected")
                    .increment(1);
                metrics::gauge!(crate::Metrics::GOSSIP_PEER_COUNT).set(peer_count as f64);

                self.peer_connection_start.entry(peer_id).or_insert_with(Instant::now);
                self.reply_to_connects(&peer_id, Ok(()));
            }
            SwarmEvent::OutgoingConnectionError { peer_id: _peer_id, error, .. } => {
                debug!(target: "gossip", "Outgoing connection error: {:?}", error);
                // Remove the peer from current_dials so it can be dialed again
                if let Some(peer_id) = _peer_id {
                    self.connection_gate.remove_dial(&peer_id);
                    // Our dial can fail after the peer has connected to us, for example when
                    // both dialed at once.
                    let outcome = if self.is_connected(&peer_id) {
                        Ok(())
                    } else if let libp2p::swarm::DialError::Denied { cause } = &error &&
                        let Some(refusal) = cause.downcast_ref::<DialError>()
                    {
                        Err(refusal.clone().into())
                    } else {
                        Err(ConnectPeerError::DialFailed(error.to_string()))
                    };
                    self.reply_to_connects(&peer_id, outcome);
                }
                metrics::counter!(crate::Metrics::GOSSIPSUB_CONNECTION, "type" => "outgoing_error")
                    .increment(1);
            }
            SwarmEvent::IncomingConnectionError { error, .. } => {
                debug!(target: "gossip", "Incoming connection error: {:?}", error);
                metrics::counter!(crate::Metrics::GOSSIPSUB_CONNECTION, "type" => "incoming_error")
                    .increment(1);
            }
            SwarmEvent::ConnectionClosed {
                peer_id, connection_id, num_established, cause, ..
            } => {
                if let Some(connections) = self.connections.get_mut(&peer_id) {
                    connections.remove(&connection_id);
                    if connections.is_empty() {
                        self.connections.remove(&peer_id);
                    }
                }
                metrics::counter!(crate::Metrics::GOSSIPSUB_CONNECTION, "type" => "closed")
                    .increment(1);

                // The peer is still connected, for example after closing a duplicate connection.
                if num_established > 0 {
                    debug!(target: "gossip", ?peer_id, ?cause, num_established, "Connection closed, peer still connected");
                    return None;
                }

                if self.static_peers.contains_key(&peer_id) {
                    self.update_next_static_peer_check(
                        Instant::now() + self.config.static_peer_reconnect_delay,
                    );
                }

                let peer_count = self.swarm.connected_peers().count();
                debug!(target: "gossip", ?peer_id, ?cause, peer_count, "Connection closed");
                metrics::gauge!(crate::Metrics::GOSSIP_PEER_COUNT).set(peer_count as f64);

                // Record the total connection duration.
                if let Some(start_time) = self.peer_connection_start.remove(&peer_id) {
                    let _peer_duration = start_time.elapsed();
                    metrics::histogram!(crate::Metrics::GOSSIP_PEER_CONNECTION_DURATION_SECONDS)
                        .record(_peer_duration.as_secs_f64());
                }

                // Record the peer score in the metrics if available.
                if let Some(_peer_score) = self.swarm.behaviour().gossipsub.peer_score(&peer_id) {
                    metrics::histogram!(crate::Metrics::PEER_SCORES).record(_peer_score);
                }

                self.ping.remove(&peer_id);
                for reply in self.pending_disconnects.remove(&peer_id).into_iter().flatten() {
                    let _ = reply.send(());
                }
            }
            SwarmEvent::NewListenAddr { listener_id, address } => {
                debug!(target: "gossip", reporter_id = ?listener_id, new_address = ?address, "New listen address");
            }
            // The node no longer accepts connections.
            SwarmEvent::ListenerClosed { listener_id, addresses, reason }
                if listener_id == self.listener_id =>
            {
                error!(target: "gossip", ?addresses, ?reason, "Gossip listener closed, not accepting connections");
            }
            SwarmEvent::Dialing { peer_id, connection_id } => {
                debug!(target: "gossip", ?peer_id, ?connection_id, "Dialing peer");
            }
            SwarmEvent::NewExternalAddrOfPeer { peer_id, address } => {
                debug!(target: "gossip", ?peer_id, ?address, "New external address of peer");
            }
            _ => {
                debug!(target: "gossip", ?event, "Ignoring non-behaviour in event handler");
            }
        };

        None
    }
}

#[cfg(test)]
mod tests;

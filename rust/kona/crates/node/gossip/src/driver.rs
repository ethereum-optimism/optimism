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
use tokio::time::Instant;

use crate::{
    Behaviour, BlockHandler, ConnectionGate, ConnectionGater, Event, GossipDriverBuilder, Handler,
    PeerAddr, PublishError, StartError,
};

/// A driver for a [`Swarm`] instance.
///
/// Connects the swarm to the given [`Multiaddr`]
/// and handles events using the [`BlockHandler`].
#[derive(Debug)]
pub struct GossipDriver<G: ConnectionGate> {
    /// The [`Swarm`] instance.
    #[debug(skip)]
    pub(crate) swarm: Swarm<Behaviour>,
    /// A [`Multiaddr`] to listen on.
    pub(crate) addr: Multiaddr,
    /// The [`BlockHandler`].
    pub(crate) handler: BlockHandler,
    /// A mapping from [`PeerId`] to [`Multiaddr`].
    pub(crate) peerstore: HashMap<PeerId, libp2p::identify::Info>,
    /// If set, the gossip layer will monitor peer scores and ban peers that are below a given
    /// threshold.
    pub peer_monitoring: Option<PeerMonitoring>,
    /// Tracks connection start time for peers
    pub(crate) peer_connection_start: HashMap<PeerId, Instant>,
    /// The connection gate.
    pub(crate) connection_gate: G,
    /// The addresses of each peer to dial at startup and keep connected, without the node's own
    /// ID.
    pub(crate) static_peers: HashMap<PeerId, Vec<Multiaddr>>,
    /// How often static peers that aren't connected are dialed again.
    pub(crate) static_peer_redial_interval: Duration,
    /// How soon a static peer is dialed again after its connection closes.
    pub(crate) static_peer_reconnect_delay: Duration,
    /// When static peers that aren't connected are dialed again.
    ///
    /// Set when the driver starts with static peers, and after each check to one
    /// [`Self::static_peer_redial_interval`] later. Brought forward to
    /// [`Self::static_peer_reconnect_delay`] from now when the connection to a static peer closes.
    next_static_peer_check: Option<Instant>,
    /// The latest ping round-trip time to each connected peer.
    pub(crate) ping: HashMap<PeerId, Duration>,
    /// The open connections to each peer, used to close duplicate connections.
    pub(crate) connections: HashMap<PeerId, HashMap<ConnectionId, PeerConnection>>,
    /// How long a duplicate connection stays open before it is closed.
    ///
    /// gossipsub sends its subscriptions to a peer only once, on the first connection, through a
    /// queue that all connections to the peer share (`on_connection_established` in
    /// libp2p-gossipsub). Closing a connection while it still carries them loses them, and the
    /// peers then never mesh. Waiting gives them time to arrive.
    pub(crate) duplicate_connection_grace: Duration,
    /// When the next duplicate connection may be closed, if there is one.
    ///
    /// Set when a duplicate connection is established, and after each check to the earliest
    /// duplicate still in its grace period. `None` until the next duplicate is established.
    next_duplicate_check: Option<Instant>,
    /// The listener [`Self::start`] added, removed again when the driver is dropped.
    listener_id: Option<ListenerId>,
}

/// Removes the listener, so another driver can listen on the same address.
///
/// Dropping a TCP listener frees its port anyway, but the memory transport only frees one when its
/// listener is removed, so a restarted in-memory node couldn't listen on its address again.
impl<G: ConnectionGate> Drop for GossipDriver<G> {
    fn drop(&mut self) {
        if let Some(listener_id) = self.listener_id.take() {
            self.swarm.remove_listener(listener_id);
        }
    }
}

/// How long a duplicate connection stays open by default, see
/// [`GossipDriver::duplicate_connection_grace`].
const DUPLICATE_CONNECTION_GRACE: Duration = Duration::from_secs(10);

/// How often static peers that aren't connected are dialed again by default, as in op-node.
const STATIC_PEER_REDIAL_INTERVAL: Duration = Duration::from_secs(60);

/// How soon a static peer is dialed again after its connection closes, by default.
///
/// Short, so a peer that restarts is back quickly. Not zero, so a peer that keeps closing the
/// connection right away isn't redialed in a tight loop.
const STATIC_PEER_RECONNECT_DELAY: Duration = Duration::from_secs(1);

/// An open connection to a peer.
#[derive(Debug, Clone, Copy)]
pub(crate) struct PeerConnection {
    /// The peer that dialed the connection, this node or the other one.
    dialer: PeerId,
    /// When the connection was established.
    established: Instant,
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

    /// Creates a new [`GossipDriver`] instance.
    pub fn new(swarm: Swarm<Behaviour>, addr: Multiaddr, handler: BlockHandler, gate: G) -> Self {
        Self {
            swarm,
            addr,
            handler,
            peerstore: Default::default(),
            peer_monitoring: None,
            peer_connection_start: Default::default(),
            connection_gate: gate,
            static_peers: HashMap::new(),
            static_peer_redial_interval: STATIC_PEER_REDIAL_INTERVAL,
            static_peer_reconnect_delay: STATIC_PEER_RECONNECT_DELAY,
            next_static_peer_check: None,
            listener_id: None,
            ping: HashMap::new(),
            connections: Default::default(),
            duplicate_connection_grace: DUPLICATE_CONNECTION_GRACE,
            next_duplicate_check: None,
        }
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

    /// Starts the libp2p Swarm.
    ///
    /// - Tells the swarm to listen on the given [`Multiaddr`], and waits until it does.
    /// - Dials the static peers.
    ///
    /// The dials only start once the swarm listens, so a start that fails leaves nothing running.
    ///
    /// A driver starts once: calling this again returns [`StartError::AlreadyStarted`], and the
    /// driver keeps listening on the address the first call returned.
    pub async fn start(&mut self) -> Result<Multiaddr, StartError> {
        if self.listener_id.is_some() {
            return Err(StartError::AlreadyStarted);
        }

        let id = self.swarm.listen_on(self.addr.clone()).inspect_err(|err| {
            error!(target: "gossip", "Fail to listen on {}: {err}", self.addr);
        })?;
        self.listener_id = Some(id);
        loop {
            match self.swarm.select_next_some().await {
                SwarmEvent::NewListenAddr { address, listener_id } if listener_id == id => {
                    info!(target: "gossip", peer_id = %self.local_peer_id(), "Swarm now listening on: {address}");

                    self.addr = address.clone();
                    self.dial_static_peers();

                    return Ok(address);
                }
                SwarmEvent::ListenerClosed { listener_id, reason, .. } if listener_id == id => {
                    error!(target: "gossip", ?reason, "Listener on {} closed before it reported an address", self.addr);
                    self.listener_id = None;
                    return Err(StartError::ListenerClosed(reason.err()));
                }
                // Keep the driver's state in step with the swarm. A block arriving here would be
                // dropped, but blocks need a connection, and the driver dials its static peers only
                // once it listens.
                event => {
                    self.handle_event(event);
                }
            }
        }
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
            info!(target: "gossip", %peer_id, ?addrs, "Dialing static peer");
            self.dial_peer(peer_id, addrs);
        }
        self.update_next_static_peer_check(Instant::now() + self.static_peer_redial_interval);
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

    /// Returns the address the driver listens on, the one it was built with until
    /// [`Self::start`] replaces it with the address the swarm reports.
    pub const fn addr(&self) -> &Multiaddr {
        &self.addr
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

    /// Disconnects a peer banned for its score, and forgets what identify told about it. Discovery
    /// bans the peer separately, by its ID.
    pub fn ban_peer(&mut self, peer_id: PeerId) {
        // The score that got the peer banned, read before the peer is disconnected.
        let score = self.peer_score(&peer_id);
        metrics::gauge!(
            crate::Metrics::BANNED_PEERS,
            "peer_id" => peer_id.to_string(),
            "score" => score.to_string()
        )
        .increment(1);
        self.disconnect_peer(peer_id);
        self.peerstore.remove(&peer_id);
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

    /// Blocks `peer_id`: the gate refuses to dial it, and gossipsub ignores its messages.
    pub fn block_peer(&mut self, peer_id: PeerId) {
        let behaviour = self.swarm.behaviour_mut();
        behaviour.gate.block_peer(peer_id);
        behaviour.gossipsub.blacklist_peer(&peer_id);
    }

    /// Unblocks `peer_id`, undoing [`Self::block_peer`].
    pub fn unblock_peer(&mut self, peer_id: PeerId) {
        let behaviour = self.swarm.behaviour_mut();
        behaviour.gate.unblock_peer(&peer_id);
        behaviour.gossipsub.remove_blacklisted_peer(&peer_id);
    }

    /// Lists the blocked peers.
    pub fn list_blocked_peers(&self) -> Vec<PeerId> {
        self.swarm.behaviour().gate.list_blocked_peers()
    }

    /// Blocks an IP address: the gate refuses to dial it.
    pub fn block_addr(&mut self, ip: IpAddr) {
        self.swarm.behaviour_mut().gate.block_addr(ip);
    }

    /// Unblocks an IP address.
    pub fn unblock_addr(&mut self, ip: IpAddr) {
        self.swarm.behaviour_mut().gate.unblock_addr(&ip);
    }

    /// Lists the blocked IP addresses.
    pub fn list_blocked_addrs(&self) -> Vec<IpAddr> {
        self.swarm.behaviour().gate.list_blocked_addrs()
    }

    /// Blocks a subnet: the gate refuses to dial any IP address in it.
    pub fn block_subnet(&mut self, subnet: IpNet) {
        self.swarm.behaviour_mut().gate.block_subnet(subnet);
    }

    /// Unblocks a subnet.
    pub fn unblock_subnet(&mut self, subnet: IpNet) {
        self.swarm.behaviour_mut().gate.unblock_subnet(&subnet);
    }

    /// Lists the blocked subnets.
    pub fn list_blocked_subnets(&self) -> Vec<IpNet> {
        self.swarm.behaviour().gate.list_blocked_subnets()
    }

    /// Returns whether `peer_id` is one of the static peers.
    pub fn is_static_peer(&self, peer_id: &PeerId) -> bool {
        self.static_peers.contains_key(peer_id)
    }

    /// Returns the connected peers whose gossipsub score is below the ban threshold of
    /// [`Self::peer_monitoring`], or none if it isn't set.
    ///
    /// Static peers are never banned, as in op-node: they would be dialed again right away.
    pub fn peers_to_ban(&self) -> Vec<PeerId> {
        let Some(monitoring) = self.peer_monitoring.as_ref() else {
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
            metrics::gauge!(crate::Metrics::DIAL_PEER_ERROR, "type" => "invalid_enr").increment(1);
            return None;
        };
        // `enr_to_multiaddr` ends the address in the peer ID.
        PeerAddr::try_from(addr).ok()
    }

    /// Dials the given [`Multiaddr`], which must end in the peer's `/p2p/<peer id>`.
    pub fn dial_multiaddr(&mut self, addr: Multiaddr) {
        let Some(peer_id) = ConnectionGater::peer_id_from_addr(&addr) else {
            warn!(target: "gossip", peer=?addr, "Failed to extract PeerId from Multiaddr");
            metrics::gauge!(crate::Metrics::DIAL_PEER_ERROR, "type" => "invalid_multiaddr")
                .increment(1);
            return;
        };
        self.dial_peer(peer_id, [addr]);
    }

    /// Dials `peer_id` at whichever of `addrs` connects first.
    ///
    /// Each address must end in `/p2p/<peer_id>`, and the gate checks each one: the addresses it
    /// refuses are left out, and nothing is dialed if it refuses them all. The rest make a single
    /// dial, in which libp2p tries several addresses at once and keeps the first connection.
    pub fn dial_peer(&mut self, peer_id: PeerId, addrs: impl IntoIterator<Item = Multiaddr>) {
        // Callers should not dial connected peers, so this is counted as a dial error. It is
        // checked before the gate, which would refuse it as already dialing or count it as a dial.
        if self.is_connected(&peer_id) {
            debug!(target: "gossip", ?peer_id, "Already connected to peer, not dialing");
            metrics::gauge!(crate::Metrics::DIAL_PEER_ERROR, "type" => "already_connected")
                .increment(1);
            return;
        }

        // A DNS name is resolved here, so that the gate checks the IPs libp2p then dials, rather
        // than libp2p resolving it again, maybe to an IP the gate didn't see.
        //
        // Every address is checked before the peer is marked as being dialed, which would make
        // the gate refuse all but the first.
        let gate = &self.swarm.behaviour().gate;
        let addrs: Vec<Multiaddr> = addrs
            .into_iter()
            .flat_map(|addr| ConnectionGater::resolve_dns_addrs(addr, ConnectionGater::system_resolve))
            .filter(|addr| {
                if ConnectionGater::peer_id_from_addr(addr) != Some(peer_id) {
                    warn!(target: "gossip", ?peer_id, peer=?addr, "Address is not for the peer being dialed");
                    return false;
                }
                if let Err(dial_error) = self.connection_gate.can_dial(addr, gate) {
                    debug!(target: "gossip", ?dial_error, "unable to dial peer");
                    return false;
                }
                true
            })
            .collect();
        if addrs.is_empty() {
            return;
        }

        // Let the gate know we are dialing the peer.
        // Note: libp2p-dns will automatically resolve DNS multiaddrs at the transport layer.
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
                metrics::gauge!(crate::Metrics::DIAL_PEER).increment(1);
            }
            Err(e) => {
                error!(target: "gossip", "Failed to connect to peer: {:?}", e);
                self.connection_gate.remove_dial(&peer_id);
                metrics::gauge!(crate::Metrics::DIAL_PEER_ERROR, "type" => "connection_error")
                    .increment(1);
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
    /// [`Self::duplicate_connection_grace`].
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
            let Some(closable_at) = youngest.map(|at| at + self.duplicate_connection_grace) else {
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
                let connection = PeerConnection { dialer, established: Instant::now() };
                self.connections.entry(peer_id).or_default().insert(connection_id, connection);
                if num_established.get() > 1 {
                    self.update_next_duplicate_check(
                        connection.established + self.duplicate_connection_grace,
                    );
                }

                let peer_count = self.swarm.connected_peers().count();
                info!(target: "gossip", "Connection established: {:?} | Peer Count: {}", peer_id, peer_count);
                metrics::gauge!(crate::Metrics::GOSSIPSUB_CONNECTION, "type" => "connected")
                    .increment(1);
                metrics::gauge!(crate::Metrics::GOSSIP_PEER_COUNT).set(peer_count as f64);

                self.peer_connection_start.entry(peer_id).or_insert_with(Instant::now);
            }
            SwarmEvent::OutgoingConnectionError { peer_id: _peer_id, error, .. } => {
                debug!(target: "gossip", "Outgoing connection error: {:?}", error);
                // Remove the peer from current_dials so it can be dialed again
                if let Some(peer_id) = _peer_id {
                    self.connection_gate.remove_dial(&peer_id);
                }
                metrics::gauge!(crate::Metrics::GOSSIPSUB_CONNECTION, "type" => "outgoing_error")
                    .increment(1);
            }
            SwarmEvent::IncomingConnectionError { error, .. } => {
                debug!(target: "gossip", "Incoming connection error: {:?}", error);
                metrics::gauge!(crate::Metrics::GOSSIPSUB_CONNECTION, "type" => "incoming_error")
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
                metrics::gauge!(crate::Metrics::GOSSIPSUB_CONNECTION, "type" => "closed")
                    .increment(1);

                // The peer is still connected, for example after closing a duplicate connection.
                if num_established > 0 {
                    debug!(target: "gossip", ?peer_id, ?cause, num_established, "Connection closed, peer still connected");
                    return None;
                }

                if self.static_peers.contains_key(&peer_id) {
                    self.update_next_static_peer_check(
                        Instant::now() + self.static_peer_reconnect_delay,
                    );
                }

                let peer_count = self.swarm.connected_peers().count();
                warn!(target: "gossip", ?peer_id, ?cause, peer_count, "Connection closed");
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
            }
            SwarmEvent::NewListenAddr { listener_id, address } => {
                debug!(target: "gossip", reporter_id = ?listener_id, new_address = ?address, "New listen address");
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
mod tests {
    use super::*;
    use crate::{
        GossipTransport, SIGNER_GRACE_PERIOD,
        builder::tests::{
            memory_driver, memory_driver_builder, memory_driver_builder_at, memory_transport,
            poll_until_connected,
        },
        signer::tests::{address, payload},
    };
    use futures::future;
    use libp2p::{
        StreamProtocol, Transport, TransportError,
        core::{
            muxing::StreamMuxerBox,
            transport::{self, TransportEvent},
        },
        multiaddr::Protocol,
    };
    use std::{
        pin::Pin,
        task::{Context, Poll},
    };
    use tokio::sync::watch;

    /// Polls both drivers' swarms until `done` holds for them, for at most 10 seconds.
    async fn poll_until(
        driver_1: &mut GossipDriver<ConnectionGater>,
        driver_2: &mut GossipDriver<ConnectionGater>,
        done: impl Fn(&GossipDriver<ConnectionGater>, &GossipDriver<ConnectionGater>) -> bool,
    ) {
        let poll = async {
            while !done(driver_1, driver_2) {
                tokio::select! {
                    Some(event) = driver_1.next() => { driver_1.handle_event(event); }
                    Some(event) = driver_2.next() => { driver_2.handle_event(event); }
                }
            }
        };
        tokio::time::timeout(Duration::from_secs(10), poll).await.expect("condition reached");
    }

    /// Polls both drivers' swarms and never returns, for a test that runs something else
    /// alongside, or waits a while to see that nothing happens.
    ///
    /// It sets no timeout, so a test on a paused clock can wait past [`poll_until`]'s.
    async fn poll_forever(
        driver_1: &mut GossipDriver<ConnectionGater>,
        driver_2: &mut GossipDriver<ConnectionGater>,
    ) {
        loop {
            tokio::select! {
                Some(event) = driver_1.next() => { driver_1.handle_event(event); }
                Some(event) = driver_2.next() => { driver_2.handle_event(event); }
            }
        }
    }

    /// Polls one driver's swarm until `done` holds for it, for at most 10 seconds.
    async fn poll_one_until(
        driver: &mut GossipDriver<ConnectionGater>,
        done: impl Fn(&GossipDriver<ConnectionGater>) -> bool,
    ) {
        let poll = async {
            while !done(driver) {
                if let Some(event) = driver.next().await {
                    driver.handle_event(event);
                }
            }
        };
        tokio::time::timeout(Duration::from_secs(10), poll).await.expect("condition reached");
    }

    /// Returns the number of connections the driver has open.
    fn num_connections(driver: &GossipDriver<ConnectionGater>) -> u32 {
        driver.swarm.network_info().connection_counters().num_established()
    }

    /// Returns whether the driver knows that `peer` is subscribed to some topic.
    fn knows_subscriptions(driver: &GossipDriver<ConnectionGater>, peer: PeerId) -> bool {
        driver
            .swarm
            .behaviour()
            .gossipsub
            .all_peers()
            .any(|(id, topics)| *id == peer && !topics.is_empty())
    }

    /// Starts both drivers and has them dial each other at the same time, returning their peer IDs.
    async fn cross_dial(
        driver_1: &mut GossipDriver<ConnectionGater>,
        driver_2: &mut GossipDriver<ConnectionGater>,
    ) -> (PeerId, PeerId) {
        let peer_1 = *driver_1.local_peer_id();
        let peer_2 = *driver_2.local_peer_id();
        let addr_1 = driver_1.start().await.expect("listen").with(Protocol::P2p(peer_1));
        let addr_2 = driver_2.start().await.expect("listen").with(Protocol::P2p(peer_2));
        driver_1.dial_multiaddr(addr_2);
        driver_2.dial_multiaddr(addr_1);
        (peer_1, peer_2)
    }

    /// Starting a driver again fails, and the driver keeps listening where it first started.
    #[tokio::test]
    async fn test_start_twice_fails() {
        let mut driver_1 = memory_driver();
        let mut driver_2 = memory_driver();
        let peer_1 = *driver_1.local_peer_id();
        let addr_1 = driver_1.start().await.expect("listen");

        assert!(matches!(driver_1.start().await, Err(StartError::AlreadyStarted)));
        assert_eq!(driver_1.addr(), &addr_1);

        driver_2.start().await.expect("listen");
        driver_2.dial_multiaddr(addr_1.with(Protocol::P2p(peer_1)));
        poll_until_connected(&mut driver_1, &mut driver_2).await;
    }

    /// A transport whose listeners close with an error before they report an address.
    ///
    /// The memory transport always reports the address first, so it can't test this.
    #[derive(Default)]
    struct ClosingTransport {
        /// The listener to close on the next poll.
        listener_id: Option<ListenerId>,
    }

    impl Transport for ClosingTransport {
        type Output = (PeerId, StreamMuxerBox);
        type Error = std::io::Error;
        type ListenerUpgrade = future::Pending<Result<Self::Output, Self::Error>>;
        type Dial = future::Pending<Result<Self::Output, Self::Error>>;

        fn listen_on(
            &mut self,
            id: ListenerId,
            _addr: Multiaddr,
        ) -> Result<(), TransportError<Self::Error>> {
            self.listener_id = Some(id);
            Ok(())
        }

        fn remove_listener(&mut self, _id: ListenerId) -> bool {
            false
        }

        fn dial(
            &mut self,
            addr: Multiaddr,
            _opts: transport::DialOpts,
        ) -> Result<Self::Dial, TransportError<Self::Error>> {
            Err(TransportError::MultiaddrNotSupported(addr))
        }

        fn poll(
            mut self: Pin<&mut Self>,
            _cx: &mut Context<'_>,
        ) -> Poll<TransportEvent<Self::ListenerUpgrade, Self::Error>> {
            self.listener_id.take().map_or(Poll::Pending, |listener_id| {
                Poll::Ready(TransportEvent::ListenerClosed {
                    listener_id,
                    reason: Err(std::io::Error::other("listener failed")),
                })
            })
        }
    }

    /// A listener that closes before it reports an address fails the start, instead of leaving
    /// it waiting for an address that never comes.
    #[tokio::test]
    async fn test_start_fails_when_listener_closes() {
        let mut driver = memory_driver_builder()
            .with_transport(GossipTransport::Other(Box::new(|_| {
                Ok(ClosingTransport::default().boxed())
            })))
            .build()
            .expect("build");

        let start = tokio::time::timeout(Duration::from_secs(10), driver.start()).await;
        let Ok(Err(StartError::ListenerClosed(Some(reason)))) = start else {
            panic!("expected the listener to close, got {start:?}");
        };
        assert_eq!(reason.to_string(), "listener failed");
    }

    /// kona doesn't serve op-node's retired sync request/response protocol, `payload_by_number`.
    /// A peer that still asks for it is told it isn't supported, so it can't send the node a
    /// single byte on it, let alone the 128 MiB it once could.
    #[tokio::test]
    async fn test_sync_request_protocol_is_unsupported() {
        let mut driver = memory_driver();
        let peer_id = *driver.local_peer_id();
        let addr = driver.start().await.expect("listen");

        let GossipTransport::Other(transport) = memory_transport() else {
            unreachable!("the memory transport is a custom one");
        };
        let mut peer = libp2p::SwarmBuilder::with_new_identity()
            .with_tokio()
            .with_other_transport(transport)
            .expect("transport")
            .with_behaviour(|_| libp2p_stream::Behaviour::new())
            .expect("behaviour")
            .build();
        peer.dial(addr.with(Protocol::P2p(peer_id))).expect("dial");

        let chain_id = driver.handler().rollup_config.l2_chain_id;
        let protocol =
            StreamProtocol::try_from_owned(format!("/opstack/req/payload_by_number/{chain_id}/0/"))
                .expect("protocol");
        let mut control = peer.behaviour().new_control();
        let mut open = std::pin::pin!(control.open_stream(peer_id, protocol));
        let opened = tokio::time::timeout(Duration::from_secs(10), async {
            loop {
                tokio::select! {
                    opened = &mut open => break opened,
                    Some(event) = driver.next() => { driver.handle_event(event); }
                    _ = peer.select_next_some() => {}
                }
            }
        })
        .await
        .expect("the peer gets an answer in time");

        assert!(
            matches!(opened, Err(libp2p_stream::OpenStreamError::UnsupportedProtocol(_))),
            "{opened:?}"
        );
    }

    /// Two peers that dial each other at the same time end up with one connection, the one dialed
    /// by the peer with the smaller ID, and each knows the other's subscriptions.
    #[tokio::test]
    async fn test_crossing_dials_keep_one_connection() {
        let mut driver_1 = memory_driver();
        let mut driver_2 = memory_driver();
        driver_1.duplicate_connection_grace = Duration::from_millis(100);
        driver_2.duplicate_connection_grace = Duration::from_millis(100);

        let (peer_1, peer_2) = cross_dial(&mut driver_1, &mut driver_2).await;
        poll_until(&mut driver_1, &mut driver_2, |driver_1, driver_2| {
            num_connections(driver_1) == 1 &&
                num_connections(driver_2) == 1 &&
                knows_subscriptions(driver_1, peer_2) &&
                knows_subscriptions(driver_2, peer_1)
        })
        .await;

        let preferred_dialer = peer_1.min(peer_2);
        for (driver, peer) in [(&driver_1, peer_2), (&driver_2, peer_1)] {
            let dialers: Vec<_> =
                driver.connections[&peer].values().map(|connection| connection.dialer).collect();
            assert_eq!(dialers, vec![preferred_dialer]);
        }
    }

    /// A duplicate connection stays open for its grace period, so that gossipsub's subscriptions,
    /// which it sends only on the first connection, are not lost with it.
    #[tokio::test]
    async fn test_duplicate_connection_stays_open_during_grace() {
        let mut driver_1 = memory_driver();
        let mut driver_2 = memory_driver();
        driver_1.duplicate_connection_grace = Duration::from_secs(60);
        driver_2.duplicate_connection_grace = Duration::from_secs(60);

        cross_dial(&mut driver_1, &mut driver_2).await;
        poll_until(&mut driver_1, &mut driver_2, |driver_1, driver_2| {
            num_connections(driver_1) == 2 && num_connections(driver_2) == 2
        })
        .await;

        // Well within the grace period, nothing is closed.
        let poll = poll_forever(&mut driver_1, &mut driver_2);
        let _ = tokio::time::timeout(Duration::from_millis(200), poll).await;
        assert_eq!(num_connections(&driver_1), 2);
        assert_eq!(num_connections(&driver_2), 2);
    }

    /// Builds a gossip driver on TCP, listening on a free port on localhost once started.
    fn tcp_driver() -> GossipDriver<ConnectionGater> {
        let mut driver = crate::GossipDriverBuilder::new(
            kona_genesis::RollupConfig::default(),
            alloy_primitives::Address::ZERO,
            "/ip4/127.0.0.1/tcp/0".parse().expect("valid multiaddr"),
            libp2p::identity::Keypair::generate_secp256k1(),
        )
        .build()
        .expect("build gossip driver");
        driver.duplicate_connection_grace = Duration::from_millis(100);
        driver
    }

    /// `dial_peer` resolves a DNS name before the gate, so it checks and dials the IP the name
    /// resolves to, rather than leaving libp2p to resolve the name again.
    #[tokio::test]
    async fn test_dial_peer_resolves_dns_names_before_the_gate() {
        let mut driver_1 = tcp_driver();
        let mut driver_2 = tcp_driver();
        let peer_2 = *driver_2.local_peer_id();
        let listen_addr_2 = driver_2.start().await.expect("listen");
        let port = listen_addr_2
            .iter()
            .find_map(|protocol| match protocol {
                Protocol::Tcp(port) => Some(port),
                _ => None,
            })
            .unwrap();
        let dns_addr: Multiaddr =
            format!("/dns4/localhost/tcp/{port}/p2p/{peer_2}").parse().unwrap();
        let ip_addr: Multiaddr = format!("/ip4/127.0.0.1/tcp/{port}/p2p/{peer_2}").parse().unwrap();
        driver_1.start().await.expect("listen");

        // `localhost` resolves to a blocked IP, so nothing is dialed.
        let localhost = "127.0.0.1".parse().unwrap();
        driver_1.block_addr(localhost);
        driver_1.dial_peer(peer_2, [dns_addr.clone()]);
        assert!(!driver_1.connection_gate.current_dials.contains(&peer_2));

        // Unblocked, the resolved address is what's dialed and counted.
        driver_1.unblock_addr(localhost);
        driver_1.dial_peer(peer_2, [dns_addr]);
        let dialed: Vec<_> = driver_1.connection_gate.dialed_addrs.keys().cloned().collect();
        assert_eq!(dialed, vec![ip_addr]);
        poll_until_connected(&mut driver_1, &mut driver_2).await;
    }

    /// Peers dialing each other over TCP at the same time connect. Dialing from the listen port,
    /// both dials would share one TCP connection and fail the handshake, which happens to about
    /// one crossing in five, so ten crossings catch it most of the time.
    #[tokio::test]
    async fn test_tcp_crossing_dials_connect() {
        for _ in 0..10 {
            let mut driver_1 = tcp_driver();
            let mut driver_2 = tcp_driver();
            cross_dial(&mut driver_1, &mut driver_2).await;
            poll_until(&mut driver_1, &mut driver_2, |driver_1, driver_2| {
                num_connections(driver_1) == 1 && num_connections(driver_2) == 1
            })
            .await;
        }
    }

    /// Runs `f` and returns the dial errors it recorded: for each `type` label of
    /// `kona_node_dial_peer_error`, how many times it went up, in no particular order.
    fn dial_errors(f: impl FnOnce()) -> Vec<(String, usize)> {
        let recorder = metrics_util::debugging::DebuggingRecorder::new();
        let snapshotter = recorder.snapshotter();
        metrics::with_local_recorder(&recorder, f);
        snapshotter
            .snapshot()
            .into_vec()
            .into_iter()
            .filter(|(key, ..)| key.key().name() == crate::Metrics::DIAL_PEER_ERROR)
            .filter_map(|(key, _, _, value)| {
                let kind = key.key().labels().find(|l| l.key() == "type")?.value().to_owned();
                // Recorded as a gauge that only goes up (#23699), so it holds a whole count.
                let metrics_util::debugging::DebugValue::Gauge(count) = value else {
                    return None;
                };
                Some((kind, count.into_inner() as usize))
            })
            .collect()
    }

    /// Dialing a connected peer is skipped before the gate, and recorded as an `already_connected`
    /// dial error without counting as a dial.
    #[tokio::test]
    async fn test_dial_skips_connected_peer() {
        let mut driver_1 = memory_driver();
        let mut driver_2 = memory_driver();

        let peer_2 = *driver_2.local_peer_id();
        let addr_2 = driver_2.start().await.expect("listen").with(Protocol::P2p(peer_2));
        driver_1.start().await.expect("listen");

        // The one dial that connects the peers.
        driver_1.dial_multiaddr(addr_2.clone());
        poll_until_connected(&mut driver_1, &mut driver_2).await;

        // Dial the connected peer again, as a caller that doesn't check `is_connected` would, for
        // example `opp2p_connectPeer` for a peer that is already connected. Discovery handing
        // over the peer's ENR again used to do this too, before the network actor checked.
        let redials = 3;
        let dial_errors = dial_errors(|| {
            for _ in 0..redials {
                driver_1.dial_multiaddr(addr_2.clone());
            }
        });

        // Each redial is turned away by the connected check in `dial_peer`: one `already_connected`
        // error per redial. Without the check, the gate would let them through, and libp2p would
        // refuse to dial a connected peer, which shows up as a `connection_error` instead.
        assert_eq!(dial_errors, vec![("already_connected".to_owned(), redials)]);
        // Only the first dial counts against the address's redial limit.
        assert_eq!(driver_1.connection_gate.dialed_addrs[&addr_2].num_dials, 1);
    }

    /// An address no one listens on, so a dial to it fails.
    fn unused_memory_addr(peer: PeerId) -> Multiaddr {
        Multiaddr::empty().with(Protocol::Memory(u64::MAX)).with(Protocol::P2p(peer))
    }

    /// A peer dialed at several addresses connects through the one that works, in one dial that
    /// counts against each address.
    #[tokio::test]
    async fn test_dial_peer_connects_at_any_address() {
        let mut driver_1 = memory_driver();
        let mut driver_2 = memory_driver();

        let peer_2 = *driver_2.local_peer_id();
        let addr_2 = driver_2.start().await.expect("listen").with(Protocol::P2p(peer_2));
        driver_1.start().await.expect("listen");

        driver_1.dial_peer(peer_2, [unused_memory_addr(peer_2), addr_2.clone()]);
        poll_until_connected(&mut driver_1, &mut driver_2).await;

        assert!(driver_1.is_connected(&peer_2));
        let gate = &driver_1.connection_gate;
        assert_eq!(gate.dialed_addrs[&unused_memory_addr(peer_2)].num_dials, 1);
        assert_eq!(gate.dialed_addrs[&addr_2].num_dials, 1);
    }

    /// Addresses the gate refuses, or that are for another peer, are left out of the dial.
    #[tokio::test]
    async fn test_dial_peer_leaves_out_refused_addresses() {
        let mut driver_1 = memory_driver();
        let mut driver_2 = memory_driver();

        let peer_2 = *driver_2.local_peer_id();
        let addr_2 = driver_2.start().await.expect("listen").with(Protocol::P2p(peer_2));
        driver_1.start().await.expect("listen");

        let blocked: Multiaddr = format!("/ip4/10.0.0.1/tcp/9222/p2p/{peer_2}").parse().unwrap();
        driver_1.block_addr("10.0.0.1".parse().unwrap());
        let other_peer = unused_memory_addr(PeerId::random());

        driver_1.dial_peer(peer_2, [blocked, other_peer, addr_2.clone()]);

        let dialed: Vec<_> = driver_1.connection_gate.dialed_addrs.keys().cloned().collect();
        assert_eq!(dialed, vec![addr_2]);
        poll_until_connected(&mut driver_1, &mut driver_2).await;
    }

    /// Nothing is dialed when the gate refuses every address.
    #[tokio::test]
    async fn test_dial_peer_skips_peer_with_no_allowed_address() {
        let mut driver = memory_driver();
        let peer = PeerId::random();
        driver.block_peer(peer);

        driver.dial_peer(peer, [unused_memory_addr(peer)]);

        assert!(!driver.connection_gate.current_dials.contains(&peer));
        assert!(driver.connection_gate.dialed_addrs.is_empty());
    }

    /// When every address fails, the dial ends once, so the peer can be dialed again.
    #[tokio::test]
    async fn test_dial_peer_failing_at_every_address_is_no_longer_in_flight() {
        let mut driver = memory_driver();
        driver.start().await.expect("listen");
        let peer = PeerId::random();
        let unused_2 =
            Multiaddr::empty().with(Protocol::Memory(u64::MAX - 1)).with(Protocol::P2p(peer));

        driver.dial_peer(peer, [unused_memory_addr(peer), unused_2]);
        assert!(driver.connection_gate.current_dials.contains(&peer));

        poll_one_until(&mut driver, |driver| !driver.connection_gate.current_dials.contains(&peer))
            .await;
    }

    /// Starting a driver dials its static peers, at all of their addresses.
    #[tokio::test]
    async fn test_start_dials_static_peers() {
        let mut driver_2 = memory_driver();
        let peer_2 = *driver_2.local_peer_id();
        let addr_2 = driver_2.start().await.expect("listen").with(Protocol::P2p(peer_2));

        let static_peers = [unused_memory_addr(peer_2), addr_2]
            .map(|addr| PeerAddr::try_from(addr).unwrap())
            .to_vec();
        let mut driver_1 =
            memory_driver_builder().with_static_peers(static_peers).build().expect("build");
        driver_1.start().await.expect("listen");

        poll_until_connected(&mut driver_1, &mut driver_2).await;
    }

    /// An ENR for `chain_id` signed with `keypair`, with an IPv4 socket that has a TCP port only
    /// if `tcp` is set.
    fn enr(keypair: &libp2p_identity::Keypair, chain_id: u64, tcp: bool) -> discv5::Enr {
        use alloy_rlp::Encodable;
        let mut secret = keypair.clone().try_into_secp256k1().unwrap().secret().to_bytes();
        let key = enr::CombinedKey::secp256k1_from_bytes(&mut secret).unwrap();
        let mut opstack = Vec::new();
        kona_peers::OpStackEnr::from_chain_id(chain_id).encode(&mut opstack);
        let mut builder = discv5::Enr::builder();
        builder.ip4(std::net::Ipv4Addr::LOCALHOST).udp4(9223);
        if tcp {
            builder.tcp4(9222);
        }
        builder.add_value_rlp(kona_peers::OpStackEnr::OP_CL_KEY, opstack.into());
        builder.build(&key).unwrap()
    }

    /// An ENR from discovery gives the peer ID and TCP address to dial, if it is for this chain
    /// and has a TCP port.
    #[test]
    fn test_enr_to_peer_addr() {
        let driver = memory_driver();
        let chain_id = driver.handler.rollup_config.l2_chain_id.id();
        let keypair = libp2p_identity::Keypair::generate_secp256k1();
        let peer_id = keypair.public().to_peer_id();

        let peer_addr = driver.enr_to_peer_addr(&enr(&keypair, chain_id, true)).unwrap();
        assert_eq!(peer_addr.peer_id(), peer_id);
        assert_eq!(peer_addr.to_string(), format!("/ip4/127.0.0.1/tcp/9222/p2p/{peer_id}"));
        assert_eq!(driver.enr_to_peer_addr(&enr(&keypair, chain_id + 1, true)), None);
        assert_eq!(driver.enr_to_peer_addr(&enr(&keypair, chain_id, false)), None);
    }

    /// How often static peers are dialed again in tests.
    const REDIAL_INTERVAL: Duration = Duration::from_millis(100);

    /// A random identity and memory address for a peer, with the static peer address they make,
    /// chosen up front so the peer can be listed as a static peer before it starts, or restart at
    /// the same address.
    fn random_identity() -> (libp2p_identity::Keypair, Multiaddr, PeerAddr) {
        let keypair = libp2p_identity::Keypair::generate_secp256k1();
        let addr = Multiaddr::empty().with(Protocol::Memory(rand::random::<u64>()));
        let static_peer =
            PeerAddr::try_from(addr.clone().with(Protocol::P2p(keypair.public().to_peer_id())))
                .unwrap();
        (keypair, addr, static_peer)
    }

    /// A driver whose only static peer is `static_peer`, dialed again every [`REDIAL_INTERVAL`].
    fn driver_with_static_peer(static_peer: PeerAddr) -> GossipDriver<ConnectionGater> {
        memory_driver_builder()
            .with_static_peers(vec![static_peer])
            .with_static_peer_redial_interval(REDIAL_INTERVAL)
            .build()
            .expect("build")
    }

    /// A static peer that starts after the node is dialed on the next check, and once it is
    /// connected, the checks leave it alone.
    #[tokio::test]
    async fn test_static_peer_started_later_is_dialed() {
        let (keypair_2, addr_2, static_peer) = random_identity();
        let peer_2 = static_peer.peer_id();
        let mut driver_1 = driver_with_static_peer(static_peer.clone());
        driver_1.start().await.expect("listen");

        // The dial at startup fails, as nothing listens at the address yet.
        assert!(driver_1.connection_gate.current_dials.contains(&peer_2));
        poll_one_until(&mut driver_1, |driver_1| {
            !driver_1.connection_gate.current_dials.contains(&peer_2)
        })
        .await;
        assert!(!driver_1.is_connected(&peer_2));

        // Once the peer starts, a later check dials it again.
        let mut driver_2 = memory_driver_builder_at(keypair_2, addr_2).build().expect("build");
        driver_2.start().await.expect("listen");
        poll_until_connected(&mut driver_1, &mut driver_2).await;

        // Skipped before `dial_peer`, which would count it as an `already_connected` error.
        let num_dials = driver_1.connection_gate.dialed_addrs[static_peer.addr()].num_dials;
        assert!(dial_errors(|| driver_1.dial_static_peers()).is_empty());
        assert_eq!(driver_1.connection_gate.dialed_addrs[static_peer.addr()].num_dials, num_dials);
    }

    /// A static peer that restarts is dialed again soon after its connection closes, well before
    /// the next regular check, which is a minute away.
    #[tokio::test]
    async fn test_static_peer_reconnects_after_restart() {
        let (keypair_2, addr_2, static_peer) = random_identity();
        let peer_2 = static_peer.peer_id();
        let mut driver_2 =
            memory_driver_builder_at(keypair_2.clone(), addr_2.clone()).build().expect("build");
        driver_2.start().await.expect("listen");
        // A short reconnect delay, but the default minute between regular checks: only the
        // reconnect after the connection closes can bring the peer back within the test.
        let mut driver_1 = memory_driver_builder()
            .with_static_peers(vec![static_peer])
            .with_static_peer_reconnect_delay(REDIAL_INTERVAL)
            .build()
            .unwrap();
        driver_1.start().await.expect("listen");
        poll_until_connected(&mut driver_1, &mut driver_2).await;

        // Stop the peer, and wait until the node notices.
        drop(driver_2);
        poll_one_until(&mut driver_1, |driver_1| !driver_1.is_connected(&peer_2)).await;

        // Start it again, with the same identity and address.
        let mut driver_2 = memory_driver_builder_at(keypair_2, addr_2).build().expect("build");
        driver_2.start().await.expect("listen on the same address");
        poll_until_connected(&mut driver_1, &mut driver_2).await;
    }

    /// A blocked static peer is skipped without a dial, but checks continue, so it is dialed again
    /// once unblocked.
    #[tokio::test]
    async fn test_blocked_static_peer_is_skipped() {
        let (_, _, static_peer) = random_identity();
        let peer = static_peer.peer_id();
        let mut driver = driver_with_static_peer(static_peer);
        driver.block_peer(peer);

        // Skipped before `dial_peer`, which would count it as a `blocked_peer` error.
        assert!(dial_errors(|| driver.dial_static_peers()).is_empty());

        assert!(!driver.connection_gate.current_dials.contains(&peer));
        assert!(driver.connection_gate.dialed_addrs.is_empty());
        assert!(driver.next_static_peer_check.is_some());

        // Once unblocked, the next check dials it.
        driver.unblock_peer(peer);
        driver.dial_static_peers();
        assert!(driver.connection_gate.current_dials.contains(&peer));
    }

    /// Peers scoring below the ban threshold are banned, but a static peer never is.
    #[tokio::test]
    async fn test_peers_to_ban_skips_static_peers() {
        let (keypair_2, addr_2, static_peer) = random_identity();
        let static_peer_id = static_peer.peer_id();
        let mut driver_2 = memory_driver_builder_at(keypair_2, addr_2).build().expect("build");
        driver_2.start().await.expect("listen");
        let mut driver_3 = memory_driver();
        let peer_3 = *driver_3.local_peer_id();
        let addr_3 = driver_3.start().await.expect("listen").with(Protocol::P2p(peer_3));

        // Light scoring weighs the application score, which the test can set.
        let mut driver_1 = memory_driver_builder()
            .with_peer_scoring(kona_peers::PeerScoreLevel::Light)
            .with_peer_monitoring(Some(kona_peers::PeerMonitoring {
                ban_threshold: -100.0,
                ban_duration: Duration::from_secs(60),
            }))
            .with_static_peers(vec![static_peer])
            .build()
            .expect("build");
        driver_1.start().await.expect("listen");
        poll_until_connected(&mut driver_1, &mut driver_2).await;
        driver_1.dial_multiaddr(addr_3);
        poll_until_connected(&mut driver_1, &mut driver_3).await;
        assert!(driver_1.peers_to_ban().is_empty());

        let gossipsub = &mut driver_1.swarm.behaviour_mut().gossipsub;
        for peer_id in [static_peer_id, peer_3] {
            assert!(gossipsub.set_application_score(&peer_id, -1000.0), "{peer_id} is scored");
        }

        assert_eq!(driver_1.peers_to_ban(), vec![peer_3]);
    }

    /// Banning a peer disconnects it and forgets its identify info, and the connection is cleaned
    /// up once it closes.
    #[tokio::test]
    async fn test_ban_peer() {
        let mut driver_1 = memory_driver();
        let mut driver_2 = memory_driver();
        let peer_2 = *driver_2.local_peer_id();
        let listen_addr_2 = driver_2.start().await.expect("listen");
        driver_1.start().await.expect("listen");

        driver_1.dial_multiaddr(listen_addr_2.clone().with(Protocol::P2p(peer_2)));
        // Wait for identify, so there is identify info to forget.
        poll_until(&mut driver_1, &mut driver_2, |driver_1, _| {
            driver_1.peerstore.contains_key(&peer_2)
        })
        .await;

        driver_1.ban_peer(peer_2);
        assert!(!driver_1.peerstore.contains_key(&peer_2));

        poll_one_until(&mut driver_1, |driver_1| !driver_1.is_connected(&peer_2)).await;
        assert!(!driver_1.peer_connection_start.contains_key(&peer_2));
    }

    /// Disconnecting a peer with the `DisconnectPeer` command records the connection's duration
    /// when the connection closes, not when the command runs.
    ///
    /// Pings record the duration so far too, so the test doesn't count samples. It checks the start
    /// time instead: recording the duration at close also removes it.
    #[tokio::test]
    async fn test_disconnect_peer_records_connection_duration_at_close() {
        let recorder = metrics_util::debugging::DebuggingRecorder::new();
        let snapshotter = recorder.snapshotter();
        // For the whole test thread, so that it sees what polling the swarm records too.
        let _recorder = metrics::set_default_local_recorder(&recorder);
        // Each snapshot empties the histograms, so this returns the durations since the last one.
        let durations = || -> usize {
            snapshotter
                .snapshot()
                .into_vec()
                .into_iter()
                .filter(|(key, ..)| {
                    key.key().name() == crate::Metrics::GOSSIP_PEER_CONNECTION_DURATION_SECONDS
                })
                .map(|(.., value)| match value {
                    metrics_util::debugging::DebugValue::Histogram(values) => values.len(),
                    _ => 0,
                })
                .sum()
        };

        let mut driver_1 = memory_driver();
        let mut driver_2 = memory_driver();
        let peer_2 = *driver_2.local_peer_id();
        let addr_2 = driver_2.start().await.expect("listen").with(Protocol::P2p(peer_2));
        driver_1.start().await.expect("listen");
        driver_1.dial_multiaddr(addr_2);
        poll_until_connected(&mut driver_1, &mut driver_2).await;
        durations();

        // The command only starts the disconnect: the duration isn't recorded yet.
        crate::GossipCommand::DisconnectPeer { peer_id: peer_2 }.handle(&mut driver_1);
        assert!(driver_1.peer_connection_start.contains_key(&peer_2));

        // The close records it, and removes the start time, after which pings record nothing.
        poll_one_until(&mut driver_1, |driver_1| !driver_1.is_connected(&peer_2)).await;
        assert!(!driver_1.peer_connection_start.contains_key(&peer_2));
        assert!(durations() >= 1);
    }

    /// The `UnprotectPeer` command leaves a static peer protected, but unprotects others.
    #[test]
    fn test_unprotect_peer_keeps_static_peers_protected() {
        use crate::ConnectionGate;
        use std::collections::HashSet;

        let (_, _, static_peer) = random_identity();
        let static_peer_id = static_peer.peer_id();
        let mut driver = driver_with_static_peer(static_peer);
        let other_peer = PeerId::random();
        crate::GossipCommand::ProtectPeer { peer_id: other_peer }.handle(&mut driver);
        // Both are protected to begin with, so unprotecting has something to remove.
        let protected: HashSet<_> =
            driver.connection_gate.list_protected_peers().into_iter().collect();
        assert_eq!(protected, HashSet::from([static_peer_id, other_peer]));

        for peer_id in [static_peer_id, other_peer] {
            crate::GossipCommand::UnprotectPeer { peer_id }.handle(&mut driver);
        }

        assert_eq!(driver.connection_gate.list_protected_peers(), vec![static_peer_id]);
    }

    /// A dial that connects is no longer in flight in the gate.
    #[tokio::test]
    async fn test_connected_dial_is_no_longer_in_flight() {
        let mut driver_1 = memory_driver();
        let mut driver_2 = memory_driver();

        let peer_2 = *driver_2.local_peer_id();
        let addr_2 = driver_2.start().await.expect("listen").with(Protocol::P2p(peer_2));
        driver_1.start().await.expect("listen");

        driver_1.dial_multiaddr(addr_2);
        assert!(driver_1.connection_gate.current_dials.contains(&peer_2));
        poll_until_connected(&mut driver_1, &mut driver_2).await;

        assert!(!driver_1.connection_gate.current_dials.contains(&peer_2));
    }

    fn driver(signer: watch::Receiver<Address>) -> GossipDriver<ConnectionGater> {
        GossipDriverBuilder::new(
            RollupConfig {
                l2_chain_id: alloy_chains::Chain::optimism_mainnet(),
                block_time: 2,
                ..Default::default()
            },
            address(1),
            "/ip4/127.0.0.1/tcp/0".parse().unwrap(),
            Keypair::generate_secp256k1(),
        )
        .with_unsafe_block_signer_receiver(signer)
        .build()
        .unwrap()
    }

    #[tokio::test(start_paused = true)]
    async fn idle_gossip_observes_rotation_before_next_payload() {
        let (sender, receiver) = watch::channel(address(1));
        let mut driver = driver(receiver);
        sender.send(address(2)).unwrap();
        let event = driver.next().await.expect("signer change");
        assert!(matches!(event, SwarmEvent::Behaviour(Event::SignerChanged)));
        driver.handle_event(event);

        // No gossip arrived during grace. Expiry is measured from the observed update,
        // not from this first old-key payload.
        tokio::time::advance(SIGNER_GRACE_PERIOD + Duration::from_nanos(1)).await;
        assert!(driver.handler.validate_signature(&payload(1)).is_err());
        assert!(driver.handler.validate_signature(&payload(2)).is_ok());
    }

    #[tokio::test]
    async fn closed_signer_channel_retains_its_final_value_without_spinning() {
        let (sender, receiver) = watch::channel(address(1));
        let mut driver = driver(receiver);
        sender.send(address(2)).unwrap();
        drop(sender);
        // A closed channel has nothing more to report, rather than a change on every call.
        let mut next = Box::pin(driver.next());
        assert!(futures::poll!(&mut next).is_pending());
        drop(next);
        // Validation still reads the final value.
        assert!(driver.handler.validate_signature(&payload(2)).is_ok());
        assert!(driver.handler.validate_signature(&payload(1)).is_err());
    }

    #[tokio::test(start_paused = true)]
    async fn signer_update_preserves_duplicate_connection_deadline() {
        let (sender, receiver) = watch::channel(address(1));
        let mut driver = driver(receiver);
        driver.next_duplicate_check = Some(Instant::now() + Duration::from_secs(1));
        sender.send(address(2)).unwrap();
        let event = driver.next().await.expect("signer change");
        assert!(matches!(event, SwarmEvent::Behaviour(Event::SignerChanged)));
        driver.handle_event(event);

        tokio::time::advance(Duration::from_secs(1)).await;
        let event = driver.next().await;
        assert!(matches!(event, Some(SwarmEvent::Behaviour(Event::DuplicateConnectionsDue))));
        assert!(driver.handler.validate_signature(&payload(2)).is_ok());
    }
}

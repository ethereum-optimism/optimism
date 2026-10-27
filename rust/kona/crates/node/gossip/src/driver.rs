//! Consensus-layer gossipsub driver for Optimism.

use alloy_primitives::{Address, Signature, hex};
use derive_more::Debug;
use discv5::Enr;
use futures::{AsyncReadExt, AsyncWriteExt, stream::StreamExt};
use kona_genesis::RollupConfig;
use kona_peers::{EnrValidation, PeerMonitoring, enr_to_multiaddr};
use libp2p::{
    Multiaddr, PeerId, Swarm, TransportError,
    core::transport::ListenerId,
    gossipsub::{IdentTopic, MessageId},
    swarm::{ConnectionId, SwarmEvent, dial_opts::DialOpts},
};
use libp2p_identity::Keypair;
use libp2p_stream::IncomingStreams;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{
    collections::HashMap,
    sync::Arc,
    time::{Duration, Instant},
};
use tokio::sync::Mutex;

use crate::{
    Behaviour, BlockHandler, ConnectionGate, ConnectionGater, Event, GossipDriverBuilder, Handler,
    PublishError,
};

/// A driver for a [`Swarm`] instance.
///
/// Connects the swarm to the given [`Multiaddr`]
/// and handles events using the [`BlockHandler`].
#[derive(Debug)]
pub struct GossipDriver<G: ConnectionGate> {
    /// The [`Swarm`] instance.
    #[debug(skip)]
    pub swarm: Swarm<Behaviour>,
    /// A [`Multiaddr`] to listen on.
    pub addr: Multiaddr,
    /// The [`BlockHandler`].
    pub handler: BlockHandler,
    /// A [`libp2p_stream::Control`] instance. Can be used to control the sync request/response
    #[debug(skip)]
    pub sync_handler: libp2p_stream::Control,
    /// The inbound streams for the sync request/response protocol.
    ///
    /// This is an option to allow to take the underlying value when the gossip driver gets
    /// activated.
    ///
    /// TODO(op-rs/kona#2141): remove the sync-req-resp protocol once the `op-node`
    /// phases it out.
    #[debug(skip)]
    pub sync_protocol: Option<IncomingStreams>,
    /// A mapping from [`PeerId`] to [`Multiaddr`].
    pub peerstore: HashMap<PeerId, libp2p::identify::Info>,
    /// If set, the gossip layer will monitor peer scores and ban peers that are below a given
    /// threshold.
    pub peer_monitoring: Option<PeerMonitoring>,
    /// Tracks connection start time for peers
    pub peer_connection_start: HashMap<PeerId, Instant>,
    /// The connection gate.
    pub connection_gate: G,
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
    next_static_peer_check: Option<tokio::time::Instant>,
    /// Tracks ping times for peers.
    pub ping: Arc<Mutex<HashMap<PeerId, Duration>>>,
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
    next_duplicate_check: Option<tokio::time::Instant>,
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
    established: tokio::time::Instant,
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
    pub fn new(
        swarm: Swarm<Behaviour>,
        addr: Multiaddr,
        handler: BlockHandler,
        sync_handler: libp2p_stream::Control,
        sync_protocol: IncomingStreams,
        gate: G,
    ) -> Self {
        Self {
            swarm,
            addr,
            handler,
            peerstore: Default::default(),
            peer_monitoring: None,
            peer_connection_start: Default::default(),
            sync_handler,
            sync_protocol: Some(sync_protocol),
            connection_gate: gate,
            static_peers: HashMap::new(),
            static_peer_redial_interval: STATIC_PEER_REDIAL_INTERVAL,
            static_peer_reconnect_delay: STATIC_PEER_RECONNECT_DELAY,
            next_static_peer_check: None,
            listener_id: None,
            ping: Arc::new(Mutex::new(Default::default())),
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

    /// Handles the sync request/response protocol.
    ///
    /// This is a mock handler that supports the `payload_by_number` protocol.
    /// It always returns: not found (1), version (0). `<https://specs.optimism.io/protocol/rollup-node-p2p.html#payload_by_number>`
    ///
    /// ## Note
    ///
    /// This is used to ensure op-nodes are not penalizing kona-nodes for not supporting it.
    /// This feature is being deprecated by the op-node team. Once it is fully removed from the
    /// op-node's implementation we will remove this handler.
    pub(super) fn sync_protocol_handler(&mut self) {
        let Some(mut sync_protocol) = self.sync_protocol.take() else {
            return;
        };

        // Spawn a new task to handle the sync request/response protocol.
        tokio::spawn(async move {
            loop {
                let Some((peer_id, mut inbound_stream)) = sync_protocol.next().await else {
                    warn!(target: "gossip", "The sync protocol stream has ended");
                    return;
                };

                info!(target: "gossip", "Received a sync request from {peer_id}, spawning a new task to handle it");

                tokio::spawn(async move {
                    let mut buffer = Vec::new();
                    let Ok(bytes_received) = inbound_stream.read_to_end(&mut buffer).await else {
                        error!(target: "gossip", "Failed to read the sync request from {peer_id}");
                        return;
                    };

                    debug!(target: "gossip", bytes_received = bytes_received, peer_id = ?peer_id, payload = ?buffer, "Received inbound sync request");

                    // We return: not found (1), version (0). `<https://specs.optimism.io/protocol/rollup-node-p2p.html#payload_by_number>`
                    // Response format: <response> = <res><version><payload>
                    // No payload is returned.
                    const OUTPUT: [u8; 2] = hex!("0100");

                    // We only write that we're not supporting the sync request.
                    if let Err(e) = inbound_stream.write_all(&OUTPUT).await {
                        error!(target: "gossip", err = ?e, "Failed to write the sync response to {peer_id}");
                        return;
                    };

                    debug!(target: "gossip", bytes_sent = OUTPUT.len(), peer_id = ?peer_id, "Sent outbound sync response");
                });
            }
        });
    }

    /// Starts the libp2p Swarm.
    ///
    /// - Starts the sync request/response protocol handler.
    /// - Tells the swarm to listen on the given [`Multiaddr`].
    ///
    /// Waits for the swarm to start listen before returning and connecting to peers.
    pub async fn start(&mut self) -> Result<Multiaddr, TransportError<std::io::Error>> {
        // Start the sync request/response protocol handler.
        self.sync_protocol_handler();

        match self.swarm.listen_on(self.addr.clone()) {
            Ok(id) => loop {
                self.listener_id = Some(id);
                if let SwarmEvent::NewListenAddr { address, listener_id } =
                    self.swarm.select_next_some().await &&
                    id == listener_id
                {
                    info!(target: "gossip", peer_id = %self.local_peer_id(), "Swarm now listening on: {address}");

                    self.addr = address.clone();
                    self.dial_static_peers();

                    return Ok(address);
                }
            },
            Err(err) => {
                error!(target: "gossip", "Fail to listen on {}: {err}", self.addr);
                Err(err)
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
        let due: Vec<_> = self
            .static_peers
            .iter()
            .filter(|(peer_id, _)| {
                !self.is_connected(peer_id) && !self.connection_gate.is_blocked_peer(peer_id)
            })
            .map(|(peer_id, addrs)| (*peer_id, addrs.clone()))
            .collect();
        for (peer_id, addrs) in due {
            info!(target: "gossip", %peer_id, ?addrs, "Dialing static peer");
            self.dial_peer(peer_id, addrs);
        }
        self.update_next_static_peer_check(
            tokio::time::Instant::now() + self.static_peer_redial_interval,
        );
    }

    /// Makes the next static peer check happen by `at`, keeping an earlier one.
    fn update_next_static_peer_check(&mut self, at: tokio::time::Instant) {
        self.next_static_peer_check =
            Some(self.next_static_peer_check.map_or(at, |check| check.min(at)));
    }

    /// Returns the local peer id.
    pub fn local_peer_id(&self) -> &libp2p::PeerId {
        self.swarm.local_peer_id()
    }

    /// Returns a mutable reference to the Swarm's behaviour.
    pub fn behaviour_mut(&mut self) -> &mut Behaviour {
        self.swarm.behaviour_mut()
    }

    /// Attempts to select the next event from the Swarm.
    ///
    /// When duplicate connections may be closed, it returns [`Event::DuplicateConnectionsDue`]
    /// instead, and when static peers may be dialed again, [`Event::StaticPeersDue`], for
    /// [`Self::handle_event`] to act on.
    pub async fn next(&mut self) -> Option<SwarmEvent<Event>> {
        let duplicates = self.next_duplicate_check;
        let static_peers = self.next_static_peer_check;
        // `select!` builds the futures of disabled branches too, so each needs some deadline.
        let now = tokio::time::Instant::now;
        tokio::select! {
            event = self.swarm.next() => event,
            () = tokio::time::sleep_until(duplicates.unwrap_or_else(now)), if duplicates.is_some() => {
                Some(SwarmEvent::Behaviour(Event::DuplicateConnectionsDue))
            }
            () = tokio::time::sleep_until(static_peers.unwrap_or_else(now)), if static_peers.is_some() => {
                Some(SwarmEvent::Behaviour(Event::StaticPeersDue))
            }
        }
    }

    /// Returns the number of connected peers.
    pub fn connected_peers(&self) -> usize {
        self.swarm.connected_peers().count()
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

    /// Returns the peer ID and gossip address that an [`Enr`] from discovery gives, for
    /// [`Self::dial_peer`].
    ///
    /// Returns `None` for an ENR of another OP Stack chain, or one without a TCP socket.
    pub fn enr_to_multiaddr(&self, enr: &Enr) -> Option<(PeerId, Multiaddr)> {
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
        let peer_id = ConnectionGater::peer_id_from_addr(&addr)?;
        Some((peer_id, addr))
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

        // Every address is checked before the peer is marked as being dialed, which would make
        // the gate refuse all but the first.
        let addrs: Vec<Multiaddr> = addrs
            .into_iter()
            .filter(|addr| {
                if ConnectionGater::peer_id_from_addr(addr) != Some(peer_id) {
                    warn!(target: "gossip", ?peer_id, peer=?addr, "Address is not for the peer being dialed");
                    return false;
                }
                if let Err(dial_error) = self.connection_gate.can_dial(addr) {
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
        let now = tokio::time::Instant::now();
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
    fn update_next_duplicate_check(&mut self, at: tokio::time::Instant) {
        self.next_duplicate_check =
            Some(self.next_duplicate_check.map_or(at, |check| check.min(at)));
    }

    fn handle_gossip_event(&mut self, event: Event) -> Option<OpExecutionPayloadEnvelope> {
        match event {
            Event::Gossipsub(e) => return self.handle_gossipsub_event(*e),
            Event::DuplicateConnectionsDue => self.close_duplicate_connections(),
            Event::StaticPeersDue => self.dial_static_peers(),
            Event::Ping(libp2p::ping::Event { peer, result, .. }) => {
                trace!(target: "gossip", ?peer, ?result, "Ping received");

                // If the peer is connected to gossip, record the connection duration.
                if let Some(start_time) = self.peer_connection_start.get(&peer) {
                    let _ping_duration = start_time.elapsed();
                    metrics::histogram!(crate::Metrics::GOSSIP_PEER_CONNECTION_DURATION_SECONDS)
                        .record(_ping_duration.as_secs_f64());
                }

                // Record the peer score in the metrics if available.
                if let Some(_peer_score) = self.behaviour_mut().gossipsub.peer_score(&peer) {
                    metrics::histogram!(crate::Metrics::PEER_SCORES).record(_peer_score);
                }

                let pings = Arc::clone(&self.ping);
                tokio::spawn(async move {
                    if let Ok(time) = result {
                        pings.lock().await.insert(peer, time);
                    }
                });
            }
            Event::Identify(e) => self.handle_identify_event(*e),
            // Don't do anything with stream events as this should be unreachable code.
            Event::Stream => {
                error!(target: "gossip", "Stream events should not be emitted!");
            }
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
                let connection =
                    PeerConnection { dialer, established: tokio::time::Instant::now() };
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
                        tokio::time::Instant::now() + self.static_peer_reconnect_delay,
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
                if let Some(_peer_score) = self.behaviour_mut().gossipsub.peer_score(&peer_id) {
                    metrics::histogram!(crate::Metrics::PEER_SCORES).record(_peer_score);
                }

                let pings = Arc::clone(&self.ping);
                tokio::spawn(async move {
                    pings.lock().await.remove(&peer_id);
                });

                // If the connection was initiated by us, remove the peer from the current dials
                // set so that we can dial it again.
                self.connection_gate.remove_dial(&peer_id);
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
    use crate::{
        ConnectionGater, GossipDriver, StaticPeer,
        builder::tests::{
            memory_driver, memory_driver_builder, memory_driver_builder_at, poll_until_connected,
        },
    };
    use libp2p::{Multiaddr, PeerId, multiaddr::Protocol};
    use std::time::Duration;

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
        let _ = tokio::time::timeout(Duration::from_millis(200), async {
            loop {
                tokio::select! {
                    Some(event) = driver_1.next() => { driver_1.handle_event(event); }
                    Some(event) = driver_2.next() => { driver_2.handle_event(event); }
                }
            }
        })
        .await;
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
        use crate::ConnectionGate;

        let mut driver_1 = memory_driver();
        let mut driver_2 = memory_driver();

        let peer_2 = *driver_2.local_peer_id();
        let addr_2 = driver_2.start().await.expect("listen").with(Protocol::P2p(peer_2));
        driver_1.start().await.expect("listen");

        let blocked: Multiaddr = format!("/ip4/10.0.0.1/tcp/9222/p2p/{peer_2}").parse().unwrap();
        driver_1.connection_gate.block_addr("10.0.0.1".parse().unwrap());
        let other_peer = unused_memory_addr(PeerId::random());

        driver_1.dial_peer(peer_2, [blocked, other_peer, addr_2.clone()]);

        let dialed: Vec<_> = driver_1.connection_gate.dialed_addrs.keys().cloned().collect();
        assert_eq!(dialed, vec![addr_2]);
        poll_until_connected(&mut driver_1, &mut driver_2).await;
    }

    /// Nothing is dialed when the gate refuses every address.
    #[tokio::test]
    async fn test_dial_peer_skips_peer_with_no_allowed_address() {
        use crate::ConnectionGate;

        let mut driver = memory_driver();
        let peer = PeerId::random();
        driver.connection_gate.block_peer(&peer);

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
            .map(|addr| StaticPeer::try_from(addr).unwrap())
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
    fn test_enr_to_multiaddr() {
        let driver = memory_driver();
        let chain_id = driver.handler.rollup_config.l2_chain_id.id();
        let keypair = libp2p_identity::Keypair::generate_secp256k1();
        let peer_id = keypair.public().to_peer_id();

        let expected: Multiaddr = format!("/ip4/127.0.0.1/tcp/9222/p2p/{peer_id}").parse().unwrap();
        assert_eq!(
            driver.enr_to_multiaddr(&enr(&keypair, chain_id, true)),
            Some((peer_id, expected))
        );
        assert_eq!(driver.enr_to_multiaddr(&enr(&keypair, chain_id + 1, true)), None);
        assert_eq!(driver.enr_to_multiaddr(&enr(&keypair, chain_id, false)), None);
    }

    /// How often static peers are dialed again in tests.
    const REDIAL_INTERVAL: Duration = Duration::from_millis(100);

    /// A random identity and memory address for a peer, with the static peer address they make,
    /// chosen up front so the peer can be listed as a static peer before it starts, or restart at
    /// the same address.
    fn random_identity() -> (libp2p_identity::Keypair, Multiaddr, StaticPeer) {
        let keypair = libp2p_identity::Keypair::generate_secp256k1();
        let addr = Multiaddr::empty().with(Protocol::Memory(rand::random::<u64>()));
        let static_peer =
            StaticPeer::try_from(addr.clone().with(Protocol::P2p(keypair.public().to_peer_id())))
                .unwrap();
        (keypair, addr, static_peer)
    }

    /// A driver whose only static peer is `static_peer`, dialed again every [`REDIAL_INTERVAL`].
    fn driver_with_static_peer(static_peer: StaticPeer) -> GossipDriver<ConnectionGater> {
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
        use crate::ConnectionGate;

        let (_, _, static_peer) = random_identity();
        let peer = static_peer.peer_id();
        let mut driver = driver_with_static_peer(static_peer);
        driver.connection_gate.block_peer(&peer);

        // Skipped before `dial_peer`, which would count it as a `blocked_peer` error.
        assert!(dial_errors(|| driver.dial_static_peers()).is_empty());

        assert!(!driver.connection_gate.current_dials.contains(&peer));
        assert!(driver.connection_gate.dialed_addrs.is_empty());
        assert!(driver.next_static_peer_check.is_some());
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
}

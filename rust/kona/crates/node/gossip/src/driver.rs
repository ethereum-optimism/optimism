//! Consensus-layer gossipsub driver for Optimism.

use alloy_primitives::{Address, Signature, hex};
use derive_more::Debug;
use discv5::Enr;
use futures::{AsyncReadExt, AsyncWriteExt, stream::StreamExt};
use kona_genesis::RollupConfig;
use kona_peers::{EnrValidation, PeerMonitoring, enr_to_multiaddr};
use libp2p::{
    Multiaddr, PeerId, Swarm, TransportError,
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
}

/// How long a duplicate connection stays open by default, see
/// [`GossipDriver::duplicate_connection_grace`].
const DUPLICATE_CONNECTION_GRACE: Duration = Duration::from_secs(10);

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
                if let SwarmEvent::NewListenAddr { address, listener_id } =
                    self.swarm.select_next_some().await &&
                    id == listener_id
                {
                    info!(target: "gossip", "Swarm now listening on: {address}");

                    self.addr = address.clone();

                    return Ok(address);
                }
            },
            Err(err) => {
                error!(target: "gossip", "Fail to listen on {}: {err}", self.addr);
                Err(err)
            }
        }
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
    /// instead, for [`Self::handle_event`] to close them.
    pub async fn next(&mut self) -> Option<SwarmEvent<Event>> {
        loop {
            self.handler.update_signer();
            let check = self.next_duplicate_check;
            let deadline = check.unwrap_or_else(tokio::time::Instant::now);
            tokio::select! {
                // Observe rotations even when no gossip messages arrive. Fixed-address builders
                // drop the sender, so stop polling a closed channel and retain its last value.
                _ = self.handler.signer_recv.changed(), if self.handler.signer_recv.has_changed().is_ok() => {}
                event = self.swarm.next() => return event,
                () = tokio::time::sleep_until(deadline), if check.is_some() => {
                    return Some(SwarmEvent::Behaviour(Event::DuplicateConnectionsDue));
                }
            }
        }
    }

    /// Returns the number of connected peers.
    pub fn connected_peers(&self) -> usize {
        self.swarm.connected_peers().count()
    }

    /// Dials the given [`Enr`].
    pub fn dial(&mut self, enr: Enr) {
        let validation = EnrValidation::validate(&enr, self.handler.rollup_config.l2_chain_id.id());
        if validation.is_invalid() {
            trace!(target: "gossip", "Invalid OP Stack ENR for chain id {}: {}", self.handler.rollup_config.l2_chain_id.id(), validation);
            return;
        }
        let Some(multiaddr) = enr_to_multiaddr(&enr) else {
            debug!(target: "gossip", "Failed to extract tcp socket from enr: {:?}", enr);
            metrics::gauge!(crate::Metrics::DIAL_PEER_ERROR, "type" => "invalid_enr").increment(1);
            return;
        };
        self.dial_multiaddr(multiaddr);
    }

    /// Dials the given [`Multiaddr`].
    pub fn dial_multiaddr(&mut self, addr: Multiaddr) {
        let peer_id = ConnectionGater::peer_id_from_addr(&addr);

        // Discovery hands over the ENRs of connected peers again and again. So this is checked
        // before the gate, which would refuse it as already dialing or count it as a dial.
        if peer_id.is_some_and(|peer_id| self.swarm.is_connected(&peer_id)) {
            debug!(target: "gossip", peer=?addr, "Already connected to peer, not dialing");
            metrics::gauge!(crate::Metrics::DIAL_PEER_ERROR, "type" => "already_connected")
                .increment(1);
            return;
        }

        // Check if we're allowed to dial the address.
        if let Err(dial_error) = self.connection_gate.can_dial(&addr) {
            debug!(target: "gossip", ?dial_error, "unable to dial peer");
            return;
        }

        // `ConnectionGater` rejects an address without a peer ID, but another gate may not.
        let Some(peer_id) = peer_id else {
            warn!(target: "gossip", peer=?addr, "Failed to extract PeerId from Multiaddr");
            return;
        };

        // Let the gate know we are dialing the address.
        // Note: libp2p-dns will automatically resolve DNS multiaddrs at the transport layer.
        self.connection_gate.dialing(&addr);

        // Dial from a new port. From the listen port, two peers dialing each other at the same
        // time would share one TCP connection, which fails the handshake, and on macOS could not
        // redial until it leaves `TIME_WAIT`. With new ports, each dial gets its own connection,
        // and the duplicate is closed once both are established.
        let opts = DialOpts::unknown_peer_id().address(addr.clone()).allocate_new_port().build();
        match self.swarm.dial(opts) {
            Ok(_) => {
                trace!(target: "gossip", peer=?addr, "Dialed peer");
                self.connection_gate.dialed(&addr);
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
                let dialer = if endpoint.is_dialer() { *self.local_peer_id() } else { peer_id };
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
    use super::*;
    use crate::{
        SIGNER_GRACE_PERIOD,
        builder::tests::{memory_driver, poll_until_connected},
        signer::tests::{address, payload},
    };
    use libp2p::multiaddr::Protocol;
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

    /// Dials `addr` a few times, as discovery does whenever it sees the peer again.
    fn redial(driver: &mut GossipDriver<ConnectionGater>, addr: &Multiaddr) {
        for _ in 0..3 {
            driver.dial_multiaddr(addr.clone());
        }
    }

    /// Dialing a connected peer is skipped before the gate: it is recorded as an
    /// `already_connected` dial error, not refused as already dialing, and not counted as a dial.
    #[tokio::test]
    async fn test_dial_skips_connected_peer() {
        let mut driver_1 = memory_driver();
        let mut driver_2 = memory_driver();

        let peer_2 = *driver_2.local_peer_id();
        let addr_2 = driver_2.start().await.expect("listen").with(Protocol::P2p(peer_2));
        driver_1.start().await.expect("listen");

        driver_1.dial_multiaddr(addr_2.clone());
        poll_until_connected(&mut driver_1, &mut driver_2).await;

        let recorder = metrics_util::debugging::DebuggingRecorder::new();
        let snapshotter = recorder.snapshotter();
        metrics::with_local_recorder(&recorder, || redial(&mut driver_1, &addr_2));
        let dial_error_types: Vec<(String, f64)> = snapshotter
            .snapshot()
            .into_vec()
            .into_iter()
            .filter(|(key, ..)| key.key().name() == crate::Metrics::DIAL_PEER_ERROR)
            .filter_map(|(key, _, _, value)| {
                let kind = key.key().labels().find(|l| l.key() == "type")?.value().to_owned();
                let metrics_util::debugging::DebugValue::Gauge(count) = value else {
                    return None;
                };
                Some((kind, count.into_inner()))
            })
            .collect();
        assert_eq!(dial_error_types, vec![("already_connected".to_owned(), 3.0)]);

        assert_eq!(driver_1.connection_gate.dialed_peers[&addr_2].num_dials, 1);
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
        let mut next = Box::pin(driver.next());
        assert!(futures::poll!(&mut next).is_pending());
        sender.send(address(2)).unwrap();
        assert!(futures::poll!(&mut next).is_pending());
        drop(next);

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
        let mut next = Box::pin(driver.next());
        assert!(futures::poll!(&mut next).is_pending());
        drop(next);
        assert!(driver.handler.validate_signature(&payload(2)).is_ok());
        assert!(driver.handler.validate_signature(&payload(1)).is_err());
    }

    #[tokio::test(start_paused = true)]
    async fn signer_update_preserves_duplicate_connection_deadline() {
        let (sender, receiver) = watch::channel(address(1));
        let mut driver = driver(receiver);
        driver.next_duplicate_check = Some(tokio::time::Instant::now() + Duration::from_secs(1));
        let mut next = Box::pin(driver.next());
        assert!(futures::poll!(&mut next).is_pending());
        sender.send(address(2)).unwrap();
        assert!(futures::poll!(&mut next).is_pending());

        tokio::time::advance(Duration::from_secs(1)).await;
        assert!(matches!(next.await, Some(SwarmEvent::Behaviour(Event::DuplicateConnectionsDue))));
        assert!(driver.handler.validate_signature(&payload(2)).is_ok());
    }
}

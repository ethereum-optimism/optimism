//! A builder for the [`GossipDriver`].

use alloy_primitives::Address;
use kona_genesis::RollupConfig;
use kona_peers::{PeerMonitoring, PeerScoreLevel};
use libp2p::{
    Multiaddr, PeerId, SwarmBuilder,
    core::{muxing::StreamMuxerBox, transport::Boxed},
    gossipsub::Config,
    identity::Keypair,
    noise::Config as NoiseConfig,
    tcp::Config as TcpConfig,
    yamux::Config as YamuxConfig,
};
use std::{convert::Infallible, error::Error, time::Duration};
use tokio::sync::watch;

use crate::{
    Behaviour, BlockHandler, ConnectionGate, GaterConfig, GossipDriver, GossipDriverBuilderError,
    PeerAddr,
};

/// How long a connection with no open streams stays open, unless set with
/// [`GossipDriverBuilder::with_timeout`].
const DEFAULT_IDLE_CONNECTION_TIMEOUT: Duration = Duration::from_secs(60);

/// Builds the swarm's authenticated, multiplexed transport from the node's [`Keypair`].
pub type TransportBuilder = Box<
    dyn FnOnce(&Keypair) -> Result<Boxed<(PeerId, StreamMuxerBox)>, Box<dyn Error + Send + Sync>>
        + Send,
>;

/// The transport the gossip swarm listens and dials on.
#[derive(Default, derive_more::Debug)]
pub enum GossipTransport {
    /// TCP, with DNS resolution of `/dns`, `/dns4` and `/dns6` addresses.
    #[default]
    Tcp,
    /// A transport built by the caller, such as an in-memory one in tests.
    ///
    /// The connection gater cannot apply IP and subnet blocks to addresses without an IP, such as
    /// `/memory` ones. With this transport it dials them after its other checks, such as blocked
    /// peers, leaving the rest to the transport.
    Other(#[debug(skip)] TransportBuilder),
}

/// A builder for the [`GossipDriver`].
#[derive(Debug)]
pub struct GossipDriverBuilder {
    /// The [`RollupConfig`] for the network.
    rollup_config: RollupConfig,
    /// The [`Keypair`] for the node.
    keypair: Keypair,
    /// The [`Multiaddr`] for the gossip driver to listen on.
    gossip_addr: Multiaddr,
    /// The unsafe block signer [`Address`] that gossiped blocks must be signed by.
    signer: watch::Receiver<Address>,
    /// The idle connection timeout, [`DEFAULT_IDLE_CONNECTION_TIMEOUT`] if not set.
    timeout: Option<Duration>,
    /// Sets the [`PeerScoreLevel`] for the [`Behaviour`].
    scoring: Option<PeerScoreLevel>,
    /// The [`Config`] for the [`Behaviour`].
    config: Option<Config>,
    /// If set, the gossip layer will monitor peer scores and ban peers that are below a given
    /// threshold.
    peer_monitoring: Option<PeerMonitoring>,
    /// The configuration for the connection gater.
    gater_config: Option<GaterConfig>,
    /// Topic scoring. Disabled by default.
    topic_scoring: bool,
    /// The transport the swarm listens and dials on. TCP by default.
    transport: GossipTransport,
    /// The peers to dial at startup and keep connected.
    static_peers: Vec<PeerAddr>,
    /// How often static peers that aren't connected are dialed again, once a minute if not set.
    static_peer_redial_interval: Option<Duration>,
    /// How soon a static peer is dialed again after its connection closes, a second if not set.
    static_peer_reconnect_delay: Option<Duration>,
}

impl GossipDriverBuilder {
    /// Creates a new [`GossipDriverBuilder`] that accepts blocks signed by `signer`.
    ///
    /// To follow a signer that can change, such as one read from L1, use
    /// [`Self::with_unsafe_block_signer_receiver`].
    pub fn new(
        rollup_config: RollupConfig,
        signer: Address,
        gossip_addr: Multiaddr,
        keypair: Keypair,
    ) -> Self {
        Self {
            timeout: None,
            keypair,
            gossip_addr,
            signer: watch::channel(signer).1,
            scoring: None,
            config: None,
            peer_monitoring: None,
            gater_config: None,
            rollup_config,
            topic_scoring: false,
            transport: GossipTransport::default(),
            static_peers: Vec::new(),
            static_peer_redial_interval: None,
            static_peer_reconnect_delay: None,
        }
    }

    /// Sets the [`GossipTransport`] the swarm listens and dials on.
    pub fn with_transport(mut self, transport: GossipTransport) -> Self {
        self.transport = transport;
        self
    }

    /// Sets the peers to dial at startup and keep connected.
    pub fn with_static_peers(mut self, static_peers: Vec<PeerAddr>) -> Self {
        self.static_peers = static_peers;
        self
    }

    /// Sets how often static peers that aren't connected are dialed again.
    pub const fn with_static_peer_redial_interval(mut self, interval: Duration) -> Self {
        self.static_peer_redial_interval = Some(interval);
        self
    }

    /// Sets how soon a static peer is dialed again after its connection closes.
    pub const fn with_static_peer_reconnect_delay(mut self, delay: Duration) -> Self {
        self.static_peer_reconnect_delay = Some(delay);
        self
    }

    /// Sets the configuration for the connection gater.
    pub const fn with_gater_config(mut self, config: GaterConfig) -> Self {
        self.gater_config = Some(config);
        self
    }

    /// Sets the [`RollupConfig`] for the network.
    /// This is used to determine the topic to publish to.
    pub fn with_rollup_config(mut self, rollup_config: RollupConfig) -> Self {
        self.rollup_config = rollup_config;
        self
    }

    /// Sets topic scoring.
    /// This is disabled by default.
    pub const fn with_topic_scoring(mut self, topic_scoring: bool) -> Self {
        self.topic_scoring = topic_scoring;
        self
    }

    /// Sets the [`PeerScoreLevel`] for the [`Behaviour`].
    pub const fn with_peer_scoring(mut self, level: PeerScoreLevel) -> Self {
        self.scoring = Some(level);
        self
    }

    /// Sets the [`PeerMonitoring`] configuration for the gossip driver.
    pub const fn with_peer_monitoring(mut self, peer_monitoring: Option<PeerMonitoring>) -> Self {
        self.peer_monitoring = peer_monitoring;
        self
    }

    /// Sets the receiver of the unsafe block signer [`Address`] that gossiped blocks must be
    /// signed by, so that validation follows its current value.
    pub fn with_unsafe_block_signer_receiver(mut self, signer: watch::Receiver<Address>) -> Self {
        self.signer = signer;
        self
    }

    /// Sets the [`Keypair`] for the node.
    pub fn with_keypair(mut self, keypair: Keypair) -> Self {
        self.keypair = keypair;
        self
    }

    /// Sets the swarm's idle connection timeout.
    pub const fn with_timeout(mut self, timeout: Duration) -> Self {
        self.timeout = Some(timeout);
        self
    }

    /// Sets the [`Multiaddr`] for the gossip driver to listen on.
    pub fn with_address(mut self, addr: Multiaddr) -> Self {
        self.gossip_addr = addr;
        self
    }

    /// Sets the [`Config`] for the [`Behaviour`].
    pub fn with_config(mut self, config: Config) -> Self {
        self.config = Some(config);
        self
    }

    /// Builds the [`GossipDriver`].
    pub fn build(self) -> Result<GossipDriver<crate::ConnectionGater>, GossipDriverBuilderError> {
        // Unpacked without `..`, so a new builder field that is never used fails to compile.
        let Self {
            rollup_config,
            keypair,
            gossip_addr: addr,
            signer: signer_rx,
            timeout,
            scoring,
            config,
            peer_monitoring,
            gater_config,
            topic_scoring,
            transport,
            static_peers,
            static_peer_redial_interval,
            static_peer_reconnect_delay,
        } = self;

        let timeout = timeout.unwrap_or(DEFAULT_IDLE_CONNECTION_TIMEOUT);
        let mut static_peers = PeerAddr::group_by_peer(static_peers);
        // Lets a cluster share one list.
        if let Some(addrs) = static_peers.remove(&keypair.public().to_peer_id()) {
            info!(target: "gossip", ?addrs, "Static peer list contains this node's own ID, ignoring it");
        }
        let block_time = rollup_config.block_time;

        // Block Handler setup
        let handler = BlockHandler::new(rollup_config, signer_rx);

        // Construct the gossip behaviour
        let config = config.unwrap_or_else(crate::default_config);
        info!(
            target: "gossip",
            "CONFIG: [Mesh D: {}] [Mesh L: {}] [Mesh H: {}] [Gossip Lazy: {}] [Flood Publish: {}]",
            config.mesh_n(),
            config.mesh_n_low(),
            config.mesh_n_high(),
            config.gossip_lazy(),
            config.flood_publish()
        );
        info!(
            target: "gossip",
            "CONFIG: [Heartbeat: {}] [Floodsub: {}] [Validation: {:?}] [Max Transmit: {} bytes]",
            config.heartbeat_interval().as_secs(),
            config.support_floodsub(),
            config.validation_mode(),
            config.max_transmit_size()
        );
        let mut behaviour = Behaviour::new(keypair.public(), config, &[Box::new(handler.clone())])?;

        // If peer scoring is configured, set it on the behaviour.
        match scoring {
            None => info!(target: "scoring", "Peer scoring not enabled"),
            Some(PeerScoreLevel::Off) => {
                info!(target: "scoring", level = ?PeerScoreLevel::Off, "Peer scoring explicitly disabled")
            }
            Some(level) => {
                use crate::handler::Handler;
                let params = level
                    .to_params(handler.topics(), topic_scoring, block_time)
                    .unwrap_or_default();
                // op-node doesn't start with invalid parameters either.
                behaviour
                    .gossipsub
                    .with_peer_score(params, PeerScoreLevel::thresholds())
                    .map_err(GossipDriverBuilderError::InvalidPeerScoring)?;
                debug!(target: "scoring", "Peer scoring enabled successfully");
            }
        }

        debug!(target: "gossip", "Building Swarm with Peer ID: {}", keypair.public().to_peer_id());
        let custom_transport = matches!(transport, GossipTransport::Other(_));
        let builder = SwarmBuilder::with_existing_identity(keypair).with_tokio();
        let swarm = match transport {
            // Note: with_dns() must be called after with_tcp() to wrap TCP with DNS resolution.
            GossipTransport::Tcp => builder
                .with_tcp(
                    TcpConfig::default().nodelay(true),
                    |i: &Keypair| {
                        debug!(target: "gossip", "Noise Config Peer ID: {}", i.public().to_peer_id());
                        NoiseConfig::new(i)
                    },
                    YamuxConfig::default,
                )
                .map_err(GossipDriverBuilderError::TcpError)?
                .with_dns()
                .map_err(GossipDriverBuilderError::DnsError)?
                .with_behaviour(|_| behaviour)
                .unwrap_or_else(|never: Infallible| match never {})
                .with_swarm_config(|c| c.with_idle_connection_timeout(timeout))
                .build(),
            GossipTransport::Other(build_transport) => builder
                .with_other_transport(build_transport)
                .map_err(GossipDriverBuilderError::OtherTransportError)?
                .with_behaviour(|_| behaviour)
                .unwrap_or_else(|never: Infallible| match never {})
                .with_swarm_config(|c| c.with_idle_connection_timeout(timeout))
                .build(),
        };

        let mut gate = crate::ConnectionGater::new(gater_config.unwrap_or_default());
        // Static peers are exempt from the redial limit.
        for peer_id in static_peers.keys() {
            gate.protect_peer(*peer_id);
        }
        if custom_transport {
            gate = gate.allowing_addrs_without_ip();
        }

        let mut driver = GossipDriver::new(swarm, addr, handler, gate);
        driver.peer_monitoring = peer_monitoring;
        driver.static_peers = static_peers;
        if let Some(interval) = static_peer_redial_interval {
            driver.static_peer_redial_interval = interval;
        }
        if let Some(delay) = static_peer_reconnect_delay {
            driver.static_peer_reconnect_delay = delay;
        }
        Ok(driver)
    }
}

#[cfg(test)]
pub(crate) mod tests {
    use super::*;
    use libp2p::{
        Transport,
        core::{transport::MemoryTransport, upgrade::Version},
        multiaddr::Protocol,
    };
    use std::collections::HashMap;

    /// An in-process transport with the upgrades `with_tcp` applies: lazy protocol negotiation,
    /// Noise, then Yamux.
    pub(crate) fn memory_transport() -> GossipTransport {
        GossipTransport::Other(Box::new(|keypair| {
            Ok(MemoryTransport::default()
                .upgrade(Version::V1Lazy)
                .authenticate(NoiseConfig::new(keypair)?)
                .multiplex(YamuxConfig::default())
                .boxed())
        }))
    }

    /// Builds a gossip driver whose swarm uses the in-memory transport, listening on a free
    /// `/memory` port once started.
    pub(crate) fn memory_driver() -> GossipDriver<crate::ConnectionGater> {
        memory_driver_builder().build().expect("build gossip driver")
    }

    /// A builder for [`memory_driver`], for tests that need more settings.
    pub(crate) fn memory_driver_builder() -> GossipDriverBuilder {
        memory_driver_builder_at(
            Keypair::generate_secp256k1(),
            Multiaddr::empty().with(Protocol::Memory(0)),
        )
    }

    /// Like [`memory_driver_builder`], with a given identity and listen address, for tests that
    /// need to know them before the driver starts, or to restart it.
    pub(crate) fn memory_driver_builder_at(
        keypair: Keypair,
        addr: Multiaddr,
    ) -> GossipDriverBuilder {
        // Peer scoring decays scores once a block, and gossipsub rejects a decay interval under a
        // second, so a driver with peer scoring can't be built with the default block time of 0.
        let rollup_config = RollupConfig { block_time: 2, ..Default::default() };
        GossipDriverBuilder::new(rollup_config, Address::ZERO, addr, keypair)
            .with_transport(memory_transport())
    }

    /// Polls both drivers' swarms until they are connected to each other, for at most 10 seconds.
    /// Both need polling, as the handshake runs on both sides.
    pub(crate) async fn poll_until_connected(
        driver_1: &mut GossipDriver<crate::ConnectionGater>,
        driver_2: &mut GossipDriver<crate::ConnectionGater>,
    ) {
        let (peer_1, peer_2) = (*driver_1.local_peer_id(), *driver_2.local_peer_id());
        let connected = async {
            while !driver_1.is_connected(&peer_2) || !driver_2.is_connected(&peer_1) {
                tokio::select! {
                    Some(event) = driver_1.next() => { driver_1.handle_event(event); }
                    Some(event) = driver_2.next() => { driver_2.handle_event(event); }
                }
            }
        };
        tokio::time::timeout(Duration::from_secs(10), connected).await.expect("swarms connect");
    }

    /// The settings the driver reads at runtime reach it, not just the builder.
    #[test]
    fn test_build_passes_settings_to_driver() {
        let static_peer: PeerAddr =
            format!("/ip4/127.0.0.1/tcp/9222/p2p/{}", PeerId::random()).parse().unwrap();
        let driver = memory_driver_builder()
            .with_peer_monitoring(Some(PeerMonitoring {
                ban_threshold: -10.0,
                ban_duration: Duration::from_secs(60),
            }))
            .with_static_peers(vec![static_peer.clone()])
            .build()
            .expect("build gossip driver");

        let monitoring = driver.peer_monitoring.as_ref().expect("peer monitoring is set");
        assert_eq!(monitoring.ban_threshold, -10.0);
        assert_eq!(
            driver.static_peers,
            HashMap::from([(static_peer.peer_id(), vec![static_peer.addr().clone()])])
        );
    }

    /// Peer scoring parameters that gossipsub rejects fail the build, rather than leave the node
    /// running without scoring.
    #[test]
    fn test_build_rejects_invalid_peer_scoring() {
        // A block time of 0 makes the scores decay every 0 s.
        let err = GossipDriverBuilder::new(
            RollupConfig::default(),
            Address::ZERO,
            Multiaddr::empty().with(Protocol::Memory(0)),
            Keypair::generate_secp256k1(),
        )
        .with_peer_scoring(PeerScoreLevel::Light)
        .build()
        .unwrap_err();
        assert!(matches!(err, GossipDriverBuilderError::InvalidPeerScoring(_)), "{err:?}");
    }

    /// The node's own ID is dropped from the static peers, and the others are protected from the
    /// moment the driver is built.
    #[test]
    fn test_build_protects_static_peers() {
        let keypair = Keypair::generate_secp256k1();
        let local = keypair.public().to_peer_id();
        let other = PeerId::random();
        let static_peers = [other, local]
            .map(|id| format!("/ip4/127.0.0.1/tcp/9222/p2p/{id}").parse().unwrap())
            .to_vec();

        let driver =
            memory_driver_builder_at(keypair, Multiaddr::empty().with(Protocol::Memory(0)))
                .with_static_peers(static_peers)
                .build()
                .expect("build gossip driver");

        assert_eq!(driver.static_peers.keys().collect::<Vec<_>>(), vec![&other]);
        assert_eq!(driver.connection_gate.list_protected_peers(), vec![other]);
    }

    /// A dropped driver frees its memory address, so a new one can listen there, as on TCP.
    #[tokio::test]
    async fn test_dropped_driver_frees_its_address() {
        let addr = Multiaddr::empty().with(Protocol::Memory(rand::random::<u64>()));
        let builder = || memory_driver_builder_at(Keypair::generate_secp256k1(), addr.clone());

        let mut driver = builder().build().expect("build");
        driver.start().await.expect("listen");
        drop(driver);

        builder().build().expect("build").start().await.expect("listen on the same address");
    }

    /// [`GossipTransport::Other`] replaces TCP: swarms built with a memory transport listen on
    /// and connect over `/memory` addresses.
    #[tokio::test]
    async fn test_other_transport_connects_swarms() {
        let mut driver_1 = memory_driver();
        let mut driver_2 = memory_driver();

        // Port 0 binds a free memory port, the way it binds a free TCP port.
        let addr_2 = driver_2.start().await.expect("listen");
        assert!(matches!(addr_2.addr().iter().next(), Some(Protocol::Memory(port)) if port != 0));
        driver_1.start().await.expect("listen");

        driver_1.swarm.dial(Multiaddr::from(addr_2)).expect("dial");

        poll_until_connected(&mut driver_1, &mut driver_2).await;
    }
}

//! A builder for the [`GossipDriver`].

use alloy_primitives::Address;
use kona_genesis::RollupConfig;
use kona_peers::{PeerMonitoring, PeerScoreLevel};
use libp2p::{
    Multiaddr, PeerId, StreamProtocol, SwarmBuilder,
    core::{muxing::StreamMuxerBox, transport::Boxed},
    gossipsub::Config,
    identity::Keypair,
    noise::Config as NoiseConfig,
    tcp::Config as TcpConfig,
    yamux::Config as YamuxConfig,
};
use std::{convert::Infallible, error::Error, fmt, time::Duration};
use tokio::sync::watch;

use crate::{Behaviour, BlockHandler, GaterConfig, GossipDriver, GossipDriverBuilderError};

/// Builds the swarm's authenticated, multiplexed transport from the node's [`Keypair`].
pub type TransportBuilder = Box<
    dyn FnOnce(&Keypair) -> Result<Boxed<(PeerId, StreamMuxerBox)>, Box<dyn Error + Send + Sync>>
        + Send,
>;

/// The transport the gossip swarm listens and dials on.
#[derive(Default)]
pub enum GossipTransport {
    /// TCP, with DNS resolution of `/dns`, `/dns4` and `/dns6` addresses.
    #[default]
    Tcp,
    /// A transport built by the caller, such as an in-memory one in tests.
    Other(TransportBuilder),
}

impl fmt::Debug for GossipTransport {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Tcp => f.write_str("Tcp"),
            Self::Other(_) => f.write_str("Other(..)"),
        }
    }
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
    /// The idle connection timeout as a [`Duration`].
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
        }
    }

    /// Sets the [`GossipTransport`] the swarm listens and dials on.
    pub fn with_transport(mut self, transport: GossipTransport) -> Self {
        self.transport = transport;
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
    pub fn build(
        mut self,
    ) -> Result<GossipDriver<crate::ConnectionGater>, GossipDriverBuilderError> {
        // Extract builder arguments
        let timeout = self.timeout.take().unwrap_or(Duration::from_secs(60));
        let keypair = self.keypair;
        let addr = self.gossip_addr;
        let signer_rx = self.signer;
        let rollup_config = self.rollup_config;
        let l2_chain_id = rollup_config.l2_chain_id;
        let block_time = rollup_config.block_time;

        // Block Handler setup
        let handler = BlockHandler::new(rollup_config, signer_rx);

        // Construct the gossip behaviour
        let config = self.config.unwrap_or_else(crate::default_config);
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
        match self.scoring {
            None => info!(target: "scoring", "Peer scoring not enabled"),
            Some(PeerScoreLevel::Off) => {
                info!(target: "scoring", level = ?PeerScoreLevel::Off, "Peer scoring explicitly disabled")
            }
            Some(level) => {
                use crate::handler::Handler;
                let params = level
                    .to_params(handler.topics(), self.topic_scoring, block_time)
                    .unwrap_or_default();
                match behaviour.gossipsub.with_peer_score(params, PeerScoreLevel::thresholds()) {
                    Ok(_) => debug!(target: "scoring", "Peer scoring enabled successfully"),
                    Err(e) => warn!(target: "scoring", "Peer scoring failed: {}", e),
                }
            }
        }

        // Let's setup the sync request/response protocol stream.
        let mut sync_handler = behaviour.sync_req_resp.new_control();

        let protocol = format!("/opstack/req/payload_by_number/{l2_chain_id}/0/");
        let sync_protocol_name = StreamProtocol::try_from_owned(protocol)
            .map_err(GossipDriverBuilderError::SetupSyncReqRespError)?;
        let sync_protocol = sync_handler
            .accept(sync_protocol_name)
            .map_err(GossipDriverBuilderError::SyncReqRespAlreadyAccepted)?;

        debug!(target: "gossip", "Building Swarm with Peer ID: {}", keypair.public().to_peer_id());
        let builder = SwarmBuilder::with_existing_identity(keypair).with_tokio();
        let swarm = match self.transport {
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

        let gater_config = self.gater_config.take().unwrap_or_default();
        let gate = crate::ConnectionGater::new(gater_config);

        Ok(GossipDriver::new(swarm, addr, handler, sync_handler, sync_protocol, gate))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use libp2p::{
        Transport,
        core::{transport::MemoryTransport, upgrade::Version},
        multiaddr::Protocol,
    };

    /// An in-process transport with the upgrades `with_tcp` applies: lazy protocol negotiation,
    /// Noise, then Yamux.
    fn memory_transport() -> GossipTransport {
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
    fn memory_driver() -> GossipDriver<crate::ConnectionGater> {
        GossipDriverBuilder::new(
            RollupConfig::default(),
            Address::ZERO,
            Multiaddr::empty().with(Protocol::Memory(0)),
            Keypair::generate_secp256k1(),
        )
        .with_transport(memory_transport())
        .build()
        .expect("build gossip driver")
    }

    /// [`GossipTransport::Other`] replaces TCP: swarms built with a memory transport listen on
    /// and connect over `/memory` addresses.
    #[tokio::test]
    async fn test_other_transport_connects_swarms() {
        let mut driver_1 = memory_driver();
        let mut driver_2 = memory_driver();

        // Port 0 binds a free memory port, the way it binds a free TCP port.
        let addr_2 = driver_2.start().await.expect("listen");
        assert!(matches!(addr_2.iter().next(), Some(Protocol::Memory(port)) if port != 0));
        driver_1.start().await.expect("listen");

        let peer_2 = *driver_2.local_peer_id();
        driver_1.swarm.dial(addr_2.with(Protocol::P2p(peer_2))).expect("dial");

        // Both swarms have to be polled for the handshake to complete.
        let connected = async {
            while driver_1.connected_peers() == 0 || driver_2.connected_peers() == 0 {
                tokio::select! {
                    Some(event) = driver_1.next() => { driver_1.handle_event(event); }
                    Some(event) = driver_2.next() => { driver_2.handle_event(event); }
                }
            }
        };
        tokio::time::timeout(Duration::from_secs(10), connected).await.expect("swarms connect");

        assert!(driver_1.swarm.is_connected(&peer_2));
        assert!(driver_2.swarm.is_connected(driver_1.local_peer_id()));
    }
}

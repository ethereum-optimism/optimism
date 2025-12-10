//! A builder for the [`GossipDriver`].

use alloy_primitives::Address;
use kona_genesis::RollupConfig;
use kona_peers::{PeerMonitoring, PeerScoreLevel};
use libp2p::{
    Multiaddr, StreamProtocol, SwarmBuilder, gossipsub::Config, identity::Keypair,
    noise::Config as NoiseConfig, tcp::Config as TcpConfig, yamux::Config as YamuxConfig,
};
use std::time::Duration;
use tokio::sync::watch::{self};

use crate::{Behaviour, BlockHandler, GaterConfig, GossipDriver, GossipDriverBuilderError};

/// A builder for the [`GossipDriver`].
#[derive(Debug)]
pub struct GossipDriverBuilder {
    /// The [`RollupConfig`] for the network.
    rollup_config: RollupConfig,
    /// The [`Keypair`] for the node.
    keypair: Keypair,
    /// The [`Multiaddr`] for the gossip driver to listen on.
    gossip_addr: Multiaddr,
    /// Unsafe block signer [`Address`].
    signer: Address,
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
}

impl GossipDriverBuilder {
    /// Creates a new [`GossipDriverBuilder`].
    pub const fn new(
        rollup_config: RollupConfig,
        signer: Address,
        gossip_addr: Multiaddr,
        keypair: Keypair,
    ) -> Self {
        Self {
            timeout: None,
            keypair,
            gossip_addr,
            signer,
            scoring: None,
            config: None,
            peer_monitoring: None,
            gater_config: None,
            rollup_config,
            topic_scoring: false,
        }
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

    /// Sets the unsafe block signer [`Address`].
    pub const fn with_unsafe_block_signer_receiver(mut self, signer: Address) -> Self {
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
    ) -> Result<
        (GossipDriver<crate::ConnectionGater>, watch::Sender<Address>),
        GossipDriverBuilderError,
    > {
        // Extract builder arguments
        let timeout = self.timeout.take().unwrap_or(Duration::from_secs(60));
        let keypair = self.keypair;
        let addr = self.gossip_addr;
        let signer_recv = self.signer;
        let rollup_config = self.rollup_config;
        let l2_chain_id = rollup_config.l2_chain_id;
        let block_time = rollup_config.block_time;

        let (signer_tx, signer_rx) = watch::channel(signer_recv);

        // Block Handler setup
        let handler = BlockHandler::new(rollup_config, signer_rx);

        // Construct the gossip behaviour
        let config = self.config.unwrap_or(crate::default_config());
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
            .map_err(|_| GossipDriverBuilderError::SetupSyncReqRespError)?;
        let sync_protocol = sync_handler
            .accept(sync_protocol_name)
            .map_err(|_| GossipDriverBuilderError::SyncReqRespAlreadyAccepted)?;

        // Build the swarm.
        debug!(target: "gossip", "Building Swarm with Peer ID: {}", keypair.public().to_peer_id());
        let swarm = SwarmBuilder::with_existing_identity(keypair)
            .with_tokio()
            .with_tcp(
                TcpConfig::default().nodelay(true),
                |i: &Keypair| {
                    debug!(target: "gossip", "Noise Config Peer ID: {}", i.public().to_peer_id());
                    NoiseConfig::new(i)
                },
                YamuxConfig::default,
            )
            .map_err(|_| GossipDriverBuilderError::TcpError)?
            .with_behaviour(|_| behaviour)
            .map_err(|_| GossipDriverBuilderError::WithBehaviourError)?
            .with_swarm_config(|c| c.with_idle_connection_timeout(timeout))
            .build();

        let gater_config = self.gater_config.take().unwrap_or_default();
        let gate = crate::ConnectionGater::new(gater_config);

        Ok((GossipDriver::new(swarm, addr, handler, sync_handler, sync_protocol, gate), signer_tx))
    }
}

//! A builder for the [`GossipDriver`].

use alloy_primitives::Address;
use kona_genesis::RollupConfig;
use kona_peers::{PeerMonitoring, PeerScoreLevel};
use libp2p::{
    Multiaddr, StreamProtocol, SwarmBuilder, gossipsub::Config, identity::Keypair,
    noise::Config as NoiseConfig, tcp::Config as TcpConfig, yamux::Config as YamuxConfig,
};
use std::time::Duration;
use tokio::sync::watch;

use crate::{Behaviour, BlockHandler, GaterConfig, GossipDriver, GossipDriverBuilderError, Handler};

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

    /// Returns the rollup config — accessor for downstream callers that need
    /// to construct a default [`BlockHandler`] without consuming the builder.
    pub const fn rollup_config(&self) -> &RollupConfig {
        &self.rollup_config
    }

    /// Returns the configured unsafe-block-signer address.
    pub const fn signer(&self) -> Address {
        self.signer
    }

    /// Sets the [`Config`] for the [`Behaviour`].
    pub fn with_config(mut self, config: Config) -> Self {
        self.config = Some(config);
        self
    }

    /// Builds the [`GossipDriver`] with the default `BlockHandler`.
    ///
    /// Returns the driver plus a `watch::Sender<Address>` for hot-swapping
    /// the unsafe-block-signer address (driven by `SystemConfig` updates).
    /// To plug in a custom `Handler` impl instead — e.g. for SRA-based
    /// rotation where the valid-set varies per epoch — use
    /// [`Self::build_with_handler`].
    pub fn build(
        mut self,
    ) -> Result<
        (GossipDriver<crate::ConnectionGater, BlockHandler>, watch::Sender<Address>),
        GossipDriverBuilderError,
    > {
        let signer_recv = self.signer;
        let rollup_config = self.rollup_config.clone();
        let block_time = rollup_config.block_time;
        let (signer_tx, signer_rx) = watch::channel(signer_recv);
        let handler = BlockHandler::new(rollup_config, signer_rx);
        let driver = self.build_with_handler_inner(handler, Some(block_time))?;
        Ok((driver, signer_tx))
    }

    /// Builds the [`GossipDriver`] with a caller-supplied `Handler` impl.
    ///
    /// This is the override seam used by binaries that need a non-default
    /// validation rule (e.g. PSO Chain's SRA-set membership check). The
    /// returned driver does NOT come with a `watch::Sender<Address>` — the
    /// caller's handler is expected to source its valid-set itself.
    ///
    /// Caveat: peer scoring uses `block_time` from the rollup config to size
    /// the scoring params. If you want peer scoring with a custom handler,
    /// the same `RollupConfig` set on the builder is used for the timing.
    pub fn build_with_handler<H>(
        mut self,
        handler: H,
    ) -> Result<GossipDriver<crate::ConnectionGater, H>, GossipDriverBuilderError>
    where
        H: Handler + Clone + 'static,
    {
        let block_time = self.rollup_config.block_time;
        self.build_with_handler_inner(handler, Some(block_time))
    }

    /// Shared assembly path used by both [`Self::build`] and
    /// [`Self::build_with_handler`]. `block_time` is only consumed when
    /// peer-scoring is enabled.
    fn build_with_handler_inner<H>(
        &mut self,
        handler: H,
        block_time: Option<u64>,
    ) -> Result<GossipDriver<crate::ConnectionGater, H>, GossipDriverBuilderError>
    where
        H: Handler + Clone + 'static,
    {
        let timeout = self.timeout.take().unwrap_or(Duration::from_secs(60));
        let keypair = self.keypair.clone();
        let addr = self.gossip_addr.clone();
        let l2_chain_id = self.rollup_config.l2_chain_id;

        // Construct the gossip behaviour
        let config = self.config.take().unwrap_or_else(crate::default_config);
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
                let params = level
                    .to_params(handler.topics(), self.topic_scoring, block_time.unwrap_or(2))
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

        // Build the swarm with DNS+TCP transport.
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
            .with_dns()
            .map_err(|_| GossipDriverBuilderError::TcpError)?
            .with_behaviour(|_| behaviour)
            .map_err(|_| GossipDriverBuilderError::WithBehaviourError)?
            .with_swarm_config(|c| c.with_idle_connection_timeout(timeout))
            .build();

        let gater_config = self.gater_config.take().unwrap_or_default();
        let gate = crate::ConnectionGater::new(gater_config);

        Ok(GossipDriver::new(swarm, addr, handler, sync_handler, sync_protocol, gate))
    }
}

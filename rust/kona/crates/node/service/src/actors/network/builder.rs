//! Network Builder Module.

use alloy_primitives::Address;
use discv5::Config as Discv5Config;
use kona_disc::{Discv5Builder, LocalNode};
use kona_genesis::RollupConfig;
use kona_gossip::{GaterConfig, GossipDriverBuilder};
use kona_peers::{BootNodes, BootStoreFile, PeerMonitoring, PeerScoreLevel};
use kona_sources::BlockSigner;
use libp2p::{Multiaddr, identity::Keypair};
use std::time::Duration;

use crate::{
    NetworkBuilderError,
    actors::network::{NetworkConfig, NetworkDriver},
};

/// Constructs a [`NetworkDriver`] for the OP Stack Consensus Layer.
#[derive(Debug)]
pub struct NetworkBuilder {
    /// The discovery driver.
    pub(super) discovery: Discv5Builder,
    /// The gossip driver.
    pub(super) gossip: GossipDriverBuilder,
    /// A signer for payloads.
    pub(super) signer: Option<BlockSigner>,
    /// Whether to update the ENR socket after the libp2p Swarm is started.
    /// This is set to true by default.
    /// This may be set to false if the node is configured to use a static advertised address (when
    /// used with a nat for example).
    pub(super) enr_update: bool,
}

impl From<NetworkConfig> for NetworkBuilder {
    fn from(config: NetworkConfig) -> Self {
        Self::new(
            config.rollup_config,
            config.unsafe_block_signer,
            config.gossip_address,
            config.keypair,
            config.discovery_address,
            config.discovery_config,
            config.gossip_signer,
        )
        .with_enr_update(config.enr_update)
        .with_discovery_randomize(config.discovery_randomize)
        .with_bootstore(config.bootstore)
        .with_bootnodes(config.bootnodes)
        .with_discovery_interval(config.discovery_interval)
        .with_gossip_config(config.gossip_config)
        .with_peer_scoring(config.scoring)
        .with_peer_monitoring(config.monitor_peers)
        .with_topic_scoring(config.topic_scoring)
        .with_gater_config(config.gater_config)
    }
}

impl NetworkBuilder {
    /// Creates a new [`NetworkBuilder`].
    pub fn new(
        rollup_config: RollupConfig,
        unsafe_block_signer: Address,
        gossip_addr: Multiaddr,
        keypair: Keypair,
        discovery_address: LocalNode,
        discovery_config: discv5::Config,
        signer: Option<BlockSigner>,
    ) -> Self {
        Self {
            discovery: Discv5Builder::new(
                discovery_address,
                rollup_config.l2_chain_id.id(),
                discovery_config,
            ),
            gossip: GossipDriverBuilder::new(
                rollup_config,
                unsafe_block_signer,
                gossip_addr,
                keypair,
            ),
            signer,
            enr_update: true,
        }
    }

    /// Sets the ENR update flag for the [`NetworkBuilder`].
    pub fn with_enr_update(self, enr_update: bool) -> Self {
        Self { enr_update, ..self }
    }

    /// Sets the configuration for the connection gater.
    pub fn with_gater_config(self, config: GaterConfig) -> Self {
        Self { gossip: self.gossip.with_gater_config(config), ..self }
    }

    /// Sets the signer for the [`NetworkBuilder`].
    pub fn with_signer(self, signer: Option<BlockSigner>) -> Self {
        Self { signer, ..self }
    }

    /// Sets the bootstore path for the [`Discv5Builder`].
    pub fn with_bootstore(self, bootstore: Option<BootStoreFile>) -> Self {
        Self { discovery: self.discovery.with_bootstore_file(bootstore), ..self }
    }

    /// Sets the interval at which to randomize discovery peers.
    pub fn with_discovery_randomize(self, randomize: Option<Duration>) -> Self {
        Self { discovery: self.discovery.with_discovery_randomize(randomize), ..self }
    }

    /// Sets the initial bootnodes to add to the bootstore.
    pub fn with_bootnodes(self, bootnodes: BootNodes) -> Self {
        Self { discovery: self.discovery.with_bootnodes(bootnodes), ..self }
    }

    /// Sets the peer scoring based on the given [`PeerScoreLevel`].
    pub fn with_peer_scoring(self, level: PeerScoreLevel) -> Self {
        Self { gossip: self.gossip.with_peer_scoring(level), ..self }
    }

    /// Sets topic scoring for the [`GossipDriverBuilder`].
    pub fn with_topic_scoring(self, topic_scoring: bool) -> Self {
        Self { gossip: self.gossip.with_topic_scoring(topic_scoring), ..self }
    }

    /// Sets the peer monitoring for the [`GossipDriverBuilder`].
    pub fn with_peer_monitoring(self, peer_monitoring: Option<PeerMonitoring>) -> Self {
        Self { gossip: self.gossip.with_peer_monitoring(peer_monitoring), ..self }
    }

    /// Sets the discovery interval for the [`Discv5Builder`].
    pub fn with_discovery_interval(self, interval: tokio::time::Duration) -> Self {
        Self { discovery: self.discovery.with_interval(interval), ..self }
    }

    /// Sets the address for the [`Discv5Builder`].
    pub fn with_discovery_address(self, address: LocalNode) -> Self {
        Self { discovery: self.discovery.with_local_node(address), ..self }
    }

    /// Sets the gossipsub config for the [`GossipDriverBuilder`].
    pub fn with_gossip_config(self, config: libp2p::gossipsub::Config) -> Self {
        Self { gossip: self.gossip.with_config(config), ..self }
    }

    /// Sets the [`Discv5Config`] for the [`Discv5Builder`].
    pub fn with_discovery_config(self, config: Discv5Config) -> Self {
        Self { discovery: self.discovery.with_discovery_config(config), ..self }
    }

    /// Sets the gossip address for the [`GossipDriverBuilder`].
    pub fn with_gossip_address(self, addr: Multiaddr) -> Self {
        Self { gossip: self.gossip.with_address(addr), ..self }
    }

    /// Sets the timeout for the [`GossipDriverBuilder`].
    pub fn with_timeout(self, timeout: Duration) -> Self {
        Self { gossip: self.gossip.with_timeout(timeout), ..self }
    }

    /// Builds the [`NetworkDriver`].
    pub fn build(self) -> Result<NetworkDriver, NetworkBuilderError> {
        let (gossip, unsafe_block_signer_sender) = self.gossip.build()?;
        let discovery = self.discovery.build()?;

        Ok(NetworkDriver {
            gossip,
            discovery,
            unsafe_block_signer_sender,
            signer: self.signer,
            enr_update: self.enr_update,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_chains::Chain;
    use discv5::{ConfigBuilder, ListenConfig, enr::CombinedKey};
    use libp2p::gossipsub::IdentTopic;
    use std::net::{IpAddr, Ipv4Addr, SocketAddr};

    #[derive(Debug)]
    struct NetworkBuilderParams {
        rollup_config: RollupConfig,
        signer: Address,
    }

    impl Default for NetworkBuilderParams {
        fn default() -> Self {
            Self { rollup_config: RollupConfig::default(), signer: Address::random() }
        }
    }

    fn network_builder(params: NetworkBuilderParams) -> NetworkBuilder {
        let keypair = Keypair::generate_secp256k1();
        let signer = params.signer;
        let gossip = SocketAddr::new(IpAddr::V4(Ipv4Addr::UNSPECIFIED), 9099);
        let mut gossip_addr = Multiaddr::from(gossip.ip());
        gossip_addr.push(libp2p::multiaddr::Protocol::Tcp(gossip.port()));

        let CombinedKey::Secp256k1(secret_key) = CombinedKey::generate_secp256k1() else {
            unreachable!()
        };

        let discovery_address =
            LocalNode::new(secret_key, IpAddr::V4(Ipv4Addr::UNSPECIFIED), 9098, 9098);

        let discovery_config =
            ConfigBuilder::new(ListenConfig::from_ip(IpAddr::V4(Ipv4Addr::UNSPECIFIED), 9098))
                .build();

        NetworkBuilder::new(
            params.rollup_config,
            signer,
            gossip_addr,
            keypair,
            discovery_address,
            discovery_config,
            None,
        )
    }

    #[test]
    fn test_build_simple_succeeds() {
        let signer = Address::random();
        let CombinedKey::Secp256k1(secret_key) = CombinedKey::generate_secp256k1() else {
            unreachable!()
        };
        let disc_listen = SocketAddr::new(IpAddr::V4(Ipv4Addr::UNSPECIFIED), 9097);
        let disc_enr = LocalNode::new(secret_key, IpAddr::V4(Ipv4Addr::UNSPECIFIED), 9098, 9098);
        let gossip = SocketAddr::new(IpAddr::V4(Ipv4Addr::UNSPECIFIED), 9099);
        let mut gossip_addr = Multiaddr::from(gossip.ip());
        gossip_addr.push(libp2p::multiaddr::Protocol::Tcp(gossip.port()));

        let driver = network_builder(NetworkBuilderParams {
            rollup_config: RollupConfig {
                l2_chain_id: Chain::optimism_mainnet(),
                ..Default::default()
            },
            signer,
        })
        .with_gossip_address(gossip_addr.clone())
        .with_discovery_address(disc_enr)
        .with_discovery_config(ConfigBuilder::new(disc_listen.into()).build())
        .build()
        .unwrap();

        // Driver Assertions
        let id = 10;
        assert_eq!(driver.gossip.addr, gossip_addr);
        assert_eq!(driver.discovery.chain_id, id);
        assert_eq!(driver.discovery.disc.local_enr().tcp4().unwrap(), 9098);

        // Block Handler Assertions
        assert_eq!(driver.gossip.handler.rollup_config.l2_chain_id, id);
        let v1 = IdentTopic::new(format!("/optimism/{id}/0/blocks"));
        println!("{:?}", driver.gossip.handler.blocks_v1_topic);
        assert_eq!(driver.gossip.handler.blocks_v1_topic.hash(), v1.hash());
        let v2 = IdentTopic::new(format!("/optimism/{id}/1/blocks"));
        assert_eq!(driver.gossip.handler.blocks_v2_topic.hash(), v2.hash());
        let v3 = IdentTopic::new(format!("/optimism/{id}/2/blocks"));
        assert_eq!(driver.gossip.handler.blocks_v3_topic.hash(), v3.hash());
        let v4 = IdentTopic::new(format!("/optimism/{id}/3/blocks"));
        assert_eq!(driver.gossip.handler.blocks_v4_topic.hash(), v4.hash());
    }

    #[test]
    fn test_build_network_custom_configs() {
        let gossip = SocketAddr::new(IpAddr::V4(Ipv4Addr::UNSPECIFIED), 9099);
        let mut gossip_addr = Multiaddr::from(gossip.ip());
        gossip_addr.push(libp2p::multiaddr::Protocol::Tcp(gossip.port()));

        let CombinedKey::Secp256k1(secret_key) = CombinedKey::generate_secp256k1() else {
            unreachable!()
        };

        let disc = LocalNode::new(secret_key, IpAddr::V4(Ipv4Addr::UNSPECIFIED), 9097, 9097);
        let discovery_config =
            ConfigBuilder::new(ListenConfig::from_ip(IpAddr::V4(Ipv4Addr::UNSPECIFIED), 9098))
                .build();
        let driver = network_builder(Default::default())
            .with_gossip_address(gossip_addr)
            .with_discovery_address(disc)
            .with_discovery_config(discovery_config)
            .build()
            .unwrap();

        assert_eq!(driver.discovery.disc.local_enr().tcp4().unwrap(), 9097);
    }
}

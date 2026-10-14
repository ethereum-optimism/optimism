//! Configuration for the `Network`.

use super::StartError;
use alloy_primitives::Address;
use kona_disc::{Discv5Builder, Discv5Driver, LocalNode};
use kona_genesis::RollupConfig;
use kona_gossip::{ConnectionGater, GaterConfig, GossipDriver, GossipDriverBuilder};
use kona_peers::{BootNodes, BootStoreFile, PeerMonitoring, PeerScoreLevel};
use libp2p::{Multiaddr, identity::Keypair, multiaddr::Protocol};
use std::net::{IpAddr, SocketAddr};
use tokio::{sync::watch, time::Duration};

/// Configuration for kona's P2P stack.
#[derive(Debug, Clone)]
pub struct NetworkConfig {
    /// Discovery Config.
    pub discovery_config: discv5::Config,
    /// The local node's advertised address to external peers.
    /// Note: This may be different from the node's discovery listen address.
    pub discovery_address: LocalNode,
    /// The interval to find peers.
    pub discovery_interval: Duration,
    /// The interval to remove peers from the discovery service.
    pub discovery_randomize: Option<Duration>,
    /// Whether to run discv5 discovery. Without it, the node finds no peers by itself: it keeps
    /// the peers it is told to connect to, and those that connect to it, and the discovery
    /// settings, the bootnodes and the bootstore are ignored.
    pub discovery_enabled: bool,
    /// Whether to update the ENR socket when the gossip listen address changes.
    pub enr_update: bool,
    /// The gossip address.
    pub gossip_address: libp2p::Multiaddr,
    /// The keypair.
    pub keypair: Keypair,
    /// The gossip config.
    pub gossip_config: libp2p::gossipsub::Config,
    /// The peer score level.
    pub scoring: PeerScoreLevel,
    /// Whether to enable topic scoring.
    pub topic_scoring: bool,
    /// Peer score monitoring config.
    pub monitor_peers: Option<PeerMonitoring>,
    /// An optional path to the bootstore.
    pub bootstore: Option<BootStoreFile>,
    /// The configuration for the connection gater.
    pub gater_config: GaterConfig,
    /// An optional list of bootnode ENRs to start the node with.
    pub bootnodes: BootNodes,
    /// The [`RollupConfig`].
    pub rollup_config: RollupConfig,
}

impl NetworkConfig {
    const DEFAULT_DISCOVERY_INTERVAL: Duration = Duration::from_secs(5);
    const DEFAULT_DISCOVERY_RANDOMIZE: Option<Duration> = None;

    /// Returns the [`discv5::Config`] from the CLI arguments.
    pub fn discv5_config(listen_config: discv5::ListenConfig, static_ip: bool) -> discv5::Config {
        // We can use a default listen config here since it
        // will be overridden by the discovery service builder.
        let mut builder = discv5::ConfigBuilder::new(listen_config);

        // When a node sees too few new inbound sessions, discv5's auto-NAT strips `ip`/`udp` from
        // the ENR, leaving `tcp` without an address, and ignores IP votes for 6h, which takes
        // reachable nodes off the network. op-node has no equivalent, and discv5 before 0.12
        // never ran it.
        builder.auto_nat_listen_duration(None);

        if static_ip {
            builder.disable_enr_update();
        }

        builder.build()
    }

    /// Creates a new [`NetworkConfig`] with the given [`RollupConfig`] with the minimum required
    /// fields. Generates a random keypair for the node.
    pub fn new(
        rollup_config: RollupConfig,
        discovery_listen: LocalNode,
        gossip_address: Multiaddr,
    ) -> Self {
        Self {
            rollup_config,
            discovery_config: Self::discv5_config((&discovery_listen).into(), false),
            discovery_address: discovery_listen,
            discovery_interval: Self::DEFAULT_DISCOVERY_INTERVAL,
            discovery_randomize: Self::DEFAULT_DISCOVERY_RANDOMIZE,
            discovery_enabled: true,
            gossip_address,
            enr_update: true,
            keypair: Keypair::generate_secp256k1(),
            bootnodes: Default::default(),
            bootstore: Default::default(),
            gater_config: Default::default(),
            gossip_config: kona_gossip::default_config(),
            scoring: Default::default(),
            topic_scoring: Default::default(),
            monitor_peers: Default::default(),
        }
    }

    /// Builds the gossip driver, and the discovery driver if [`Self::discovery_enabled`], and
    /// starts the gossip swarm listening.
    ///
    /// Gossiped blocks must be signed by the current value of `unsafe_block_signer`. When
    /// [`Self::enr_update`] is set, discovery advertises the address the swarm listens on, so the
    /// listen port may be `0`. Discovery is built but not started.
    pub async fn start(
        self,
        unsafe_block_signer: watch::Receiver<Address>,
    ) -> Result<(GossipDriver<ConnectionGater>, Option<Discv5Driver>), StartError> {
        let initial_signer = *unsafe_block_signer.borrow();
        let mut gossip = GossipDriverBuilder::new(
            self.rollup_config.clone(),
            initial_signer,
            self.gossip_address,
            self.keypair,
        )
        .with_unsafe_block_signer_receiver(unsafe_block_signer)
        .with_config(self.gossip_config)
        .with_peer_scoring(self.scoring)
        .with_peer_monitoring(self.monitor_peers)
        .with_topic_scoring(self.topic_scoring)
        .with_gater_config(self.gater_config)
        .build()?;
        let discovery = if self.discovery_enabled {
            Some(
                Discv5Builder::new(
                    self.discovery_address,
                    self.rollup_config.l2_chain_id.id(),
                    self.discovery_config,
                )
                .with_interval(self.discovery_interval)
                .with_discovery_randomize(self.discovery_randomize)
                .with_bootstore_file(self.bootstore)
                .with_bootnodes(self.bootnodes)
                .build()?,
            )
        } else {
            None
        };

        let listen_addr = gossip.start().await?;
        if let Some(discovery) = &discovery &&
            self.enr_update
        {
            let socket =
                tcp_socket(&listen_addr).ok_or(StartError::InvalidListenAddr(listen_addr))?;
            discovery.disc.update_local_enr_socket(socket, true);
        }
        Ok((gossip, discovery))
    }
}

/// Returns the IP address and TCP port of `addr`.
fn tcp_socket(addr: &Multiaddr) -> Option<SocketAddr> {
    let ip = addr.iter().find_map(|protocol| match protocol {
        Protocol::Ip4(ip) => Some(IpAddr::V4(ip)),
        Protocol::Ip6(ip) => Some(IpAddr::V6(ip)),
        _ => None,
    })?;
    let port = addr.iter().find_map(|protocol| match protocol {
        Protocol::Tcp(port) => Some(port),
        _ => None,
    })?;
    Some(SocketAddr::new(ip, port))
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::net::{IpAddr, Ipv4Addr};

    fn listen() -> discv5::ListenConfig {
        discv5::ListenConfig::from_ip(IpAddr::V4(Ipv4Addr::LOCALHOST), 0)
    }

    #[test]
    fn discv5_config_disables_auto_nat() {
        for static_ip in [false, true] {
            let config = NetworkConfig::discv5_config(listen(), static_ip);
            assert_eq!(config.auto_nat_listen_duration, None, "static_ip = {static_ip}");
        }
    }

    #[test]
    fn new_disables_auto_nat() {
        let local_node = LocalNode::new(
            enr::k256::ecdsa::SigningKey::from_bytes(&[1u8; 32].into()).unwrap(),
            IpAddr::V4(Ipv4Addr::LOCALHOST),
            0,
            0,
        );
        let config = NetworkConfig::new(RollupConfig::default(), local_node, Multiaddr::empty());
        assert_eq!(config.discovery_config.auto_nat_listen_duration, None);
    }

    /// A config for OP Mainnet whose gossip listens on a free loopback port and whose discovery
    /// advertises `tcp_port`.
    fn loopback(tcp_port: u16, enr_update: bool) -> NetworkConfig {
        let local_node = LocalNode::new(
            enr::k256::ecdsa::SigningKey::from_bytes(&[1u8; 32].into()).unwrap(),
            IpAddr::V4(Ipv4Addr::LOCALHOST),
            tcp_port,
            0,
        );
        let rollup_config = RollupConfig {
            l2_chain_id: alloy_chains::Chain::optimism_mainnet(),
            ..Default::default()
        };
        NetworkConfig {
            enr_update,
            ..NetworkConfig::new(rollup_config, local_node, "/ip4/127.0.0.1/tcp/0".parse().unwrap())
        }
    }

    #[tokio::test]
    async fn start_listens_on_the_block_topics_and_advertises_the_listen_port() {
        let (gossip, discovery) =
            loopback(9098, true).start(watch::channel(Address::ZERO).1).await.unwrap();
        let discovery = discovery.unwrap();

        let port = tcp_socket(&gossip.addr).unwrap().port();
        assert_ne!(port, 0);
        assert_eq!(discovery.disc.local_enr().tcp4(), Some(port));
        assert_eq!(discovery.chain_id, 10);
        let handler = &gossip.handler;
        let topics = [
            &handler.blocks_v1_topic,
            &handler.blocks_v2_topic,
            &handler.blocks_v3_topic,
            &handler.blocks_v4_topic,
        ];
        for (version, topic) in topics.into_iter().enumerate() {
            let expected =
                libp2p::gossipsub::IdentTopic::new(format!("/optimism/10/{version}/blocks"));
            assert_eq!(topic.hash(), expected.hash());
        }
    }

    #[tokio::test]
    async fn start_keeps_the_configured_enr_without_enr_update() {
        let (_gossip, discovery) =
            loopback(9098, false).start(watch::channel(Address::ZERO).1).await.unwrap();
        let discovery = discovery.unwrap();
        assert_eq!(discovery.disc.local_enr().tcp4(), Some(9098));
    }

    #[tokio::test]
    async fn start_builds_no_discovery_without_it() {
        let config = NetworkConfig { discovery_enabled: false, ..loopback(9098, true) };
        let (gossip, discovery) = config.start(watch::channel(Address::ZERO).1).await.unwrap();

        assert_ne!(tcp_socket(&gossip.addr).unwrap().port(), 0);
        assert!(discovery.is_none());
    }

    #[test]
    fn tcp_socket_needs_an_ip_and_a_tcp_port() {
        let socket = |addr: &str| tcp_socket(&addr.parse().unwrap());
        assert_eq!(socket("/ip4/127.0.0.1/tcp/9000"), Some("127.0.0.1:9000".parse().unwrap()));
        assert_eq!(socket("/ip6/::1/tcp/9000"), Some("[::1]:9000".parse().unwrap()));
        assert_eq!(socket("/ip4/127.0.0.1/udp/9000"), None);
        assert_eq!(socket("/dns4/localhost/tcp/9000"), None);
    }
}

//! Configuration for the `Network`.

use alloy_primitives::Address;
use kona_disc::LocalNode;
use kona_genesis::RollupConfig;
use kona_gossip::GaterConfig;
use kona_peers::{BootNodes, BootStoreFile, PeerMonitoring, PeerScoreLevel};
use kona_sources::BlockSigner;
use libp2p::{Multiaddr, identity::Keypair};
use tokio::time::Duration;

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
    /// Whether to update the ENR socket when the gossip listen address changes.
    pub enr_update: bool,
    /// The gossip address.
    pub gossip_address: libp2p::Multiaddr,
    /// The unsafe block signer.
    pub unsafe_block_signer: Address,
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
    /// Signs the sequencer's blocks for gossip. Required in sequencer mode. It is used by the
    /// [`SignerActor`](crate::SignerActor), not by the network itself.
    pub gossip_signer: Option<BlockSigner>,
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
        unsafe_block_signer: Address,
    ) -> Self {
        Self {
            rollup_config,
            discovery_config: Self::discv5_config((&discovery_listen).into(), false),
            discovery_address: discovery_listen,
            discovery_interval: Self::DEFAULT_DISCOVERY_INTERVAL,
            discovery_randomize: Self::DEFAULT_DISCOVERY_RANDOMIZE,
            gossip_address,
            unsafe_block_signer,
            enr_update: true,
            keypair: Keypair::generate_secp256k1(),
            bootnodes: Default::default(),
            bootstore: Default::default(),
            gater_config: Default::default(),
            gossip_config: Default::default(),
            scoring: Default::default(),
            topic_scoring: Default::default(),
            monitor_peers: Default::default(),
            gossip_signer: Default::default(),
        }
    }
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
        let config = NetworkConfig::new(
            RollupConfig::default(),
            local_node,
            Multiaddr::empty(),
            Address::ZERO,
        );
        assert_eq!(config.discovery_config.auto_nat_listen_duration, None);
    }
}

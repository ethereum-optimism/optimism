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
    /// A signer for gossip payloads.
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

        if static_ip {
            builder.disable_enr_update();

            // If we have a static IP, we don't want to use any kind of NAT discovery mechanism.
            builder.auto_nat_listen_duration(None);
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
            discovery_config: discv5::ConfigBuilder::new((&discovery_listen).into()).build(),
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

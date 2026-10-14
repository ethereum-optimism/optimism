use std::{
    collections::HashSet,
    net::{IpAddr, Ipv4Addr},
    time::Duration,
};

use alloy_chains::Chain;
use alloy_signer::k256;
use alloy_signer_local::PrivateKeySigner;
use discv5::{ConfigBuilder, Enr, ListenConfig};

use crate::actors::network::TestNetwork;
use alloy_primitives::Address;
use async_trait::async_trait;
use kona_disc::LocalNode;
use kona_genesis::RollupConfig;
use kona_node_service::{NetworkConfig, discovery, network, signer};
use kona_peers::BootNode;
use kona_rpc::P2pRpc;
use kona_sources::BlockSignerHandler;
use libp2p::{Multiaddr, identity::Keypair, multiaddr::Protocol};
use rand::RngCore;
use tokio::sync::{
    mpsc::{
        self,
        error::{SendError, TrySendError},
    },
    watch,
};
use tracing::error;

pub(crate) struct TestNetworkBuilder {
    chain_id: u64,
    unsafe_block_signer: Address,
    custom_keypair: Option<Keypair>,
}

impl TestNetworkBuilder {
    fn rollup_config(&self) -> RollupConfig {
        RollupConfig { l2_chain_id: Chain::from_id(self.chain_id), ..Default::default() }
    }

    pub(crate) fn new() -> Self {
        let chain_id = rand::rng().next_u64();

        Self { chain_id, unsafe_block_signer: Address::ZERO, custom_keypair: None }
    }

    /// Sets a sequencer keypair for the network.
    /// The next network built will be the sequencer's network. This will set the unsafe block
    /// signer to the sequencer's address and the custom keypair to the sequencer's keypair.
    /// This amounts to calling [`Self::with_unsafe_block_signer`] and [`Self::with_custom_keypair`]
    /// sequentially.
    pub(crate) fn set_sequencer(mut self) -> Self {
        let sequencer_keypair = Keypair::generate_secp256k1();
        let secp256k1_key = sequencer_keypair.clone().try_into_secp256k1()
        .map_err(|e| anyhow::anyhow!("Impossible to convert keypair to secp256k1. This is a bug since we only support secp256k1 keys: {e}")).unwrap()
        .secret().to_bytes();
        let local_node_key = k256::ecdsa::SigningKey::from_bytes(&secp256k1_key.into())
        .map_err(|e| anyhow::anyhow!("Impossible to convert keypair to k256 signing key. This is a bug since we only support secp256k1 keys: {e}")).unwrap();

        self.custom_keypair = Some(sequencer_keypair);
        self.unsafe_block_signer = Address::from_private_key(&local_node_key);

        self
    }

    /// Minimal network configuration.
    /// Only allows loopback addresses in the discovery table.
    pub(crate) async fn build(&mut self, bootnodes: Vec<Enr>) -> TestNetwork {
        let keypair = self.custom_keypair.take().unwrap_or_else(Keypair::generate_secp256k1);

        let secp256k1_key = keypair.clone().try_into_secp256k1()
        .map_err(|e| anyhow::anyhow!("Impossible to convert keypair to secp256k1. This is a bug since we only support secp256k1 keys: {e}")).unwrap()
        .secret().to_bytes();
        // LocalNode takes enr's k256 key and BlockSignerHandler alloy's signer; both share one
        // secret.
        let local_node_key = enr::k256::ecdsa::SigningKey::from_bytes(&secp256k1_key.into())
            .expect("libp2p secp256k1 secret is a valid k256 key");
        let block_signer = PrivateKeySigner::from_bytes(&secp256k1_key.into())
            .expect("libp2p secp256k1 secret is a valid k256 key");

        let node_addr = IpAddr::V4(Ipv4Addr::UNSPECIFIED);

        let discovery_config = ConfigBuilder::new(ListenConfig::from_ip(node_addr, 0))
            // Only allow loopback addresses.
            .table_filter(|enr| {
                let Some(ip) = enr.ip4() else {
                    return false;
                };

                ip.is_loopback()
            })
            .build();

        let mut gossip_multiaddr = Multiaddr::from(node_addr);
        gossip_multiaddr.push(Protocol::Tcp(0));

        // Create a new network actor. No external connections
        let config = NetworkConfig {
            keypair,
            discovery_config,
            bootnodes: bootnodes.into_iter().map(Into::into).collect::<Vec<BootNode>>().into(),
            ..NetworkConfig::new(
                // Create a new rollup config. We don't need to specify any of the fields.
                self.rollup_config(),
                LocalNode::new(local_node_key, node_addr, 0, 0),
                gossip_multiaddr,
            )
        };
        let (gossip, discovery_driver) =
            config.start(watch::channel(self.unsafe_block_signer).1).await.expect("start network");
        let discovery_builder = discovery::Builder::new();
        let discovery = discovery_builder.handle();
        let network_builder = network::Builder::new(gossip);
        let network = network_builder.handle();
        let discovery_actor = discovery_builder.build(discovery_driver, Network(network.clone()));
        tokio::spawn(async move {
            if let Err(err) = discovery_actor.await {
                error!(target: "net", ?err, "Discovery actor failed");
            }
        });

        let (blocks_tx, blocks_rx) = mpsc::channel(1024);
        let signer_builder = signer::Builder::new();
        let signer_handle = signer_builder.handle();

        let network_actor = network_builder
            // The network passes the blocks it receives to the test, and stops once the test drops
            // the receiver.
            .build(blocks_tx, BanningDiscovery(discovery.clone()));

        let p2p_rpc = P2pRpc::new(network.state(), network.clone(), discovery, self.chain_id);

        // Every test network can sign: payloads sent to `signer_handle` are signed with this
        // node's key and then gossiped.
        let signer = signer_builder.build(
            BlockSignerHandler::Local(block_signer),
            self.chain_id,
            watch::channel(self.unsafe_block_signer).1,
            Network(network),
        );
        tokio::spawn(async move {
            if let Err(err) = signer.await {
                error!(target: "net", ?err, "Signer actor failed");
            }
        });

        let handle = tokio::spawn(network_actor);

        TestNetwork { p2p_rpc, signer: signer_handle, blocks_rx, handle }
    }
}

/// Passes the peers discovery finds and the payloads the signer signs to the network actor.
#[derive(Debug, Clone)]
struct Network(network::Handle);

impl discovery::Network for Network {
    fn dial(&self, enr: Enr) -> Result<(), TrySendError<Enr>> {
        self.0.dial(enr)
    }
}

#[async_trait]
impl signer::Network for Network {
    type Error = SendError<signer::Payload>;

    async fn publish(&self, payload: signer::Payload) -> Result<(), Self::Error> {
        self.0.publish(payload).await
    }
}

/// Passes the network actor's bans to the discovery actor.
#[derive(Debug)]
struct BanningDiscovery(discovery::Handle);

impl network::Discovery for BanningDiscovery {
    fn try_ban(
        &self,
        addrs: HashSet<Multiaddr>,
        duration: Duration,
    ) -> Result<(), TrySendError<()>> {
        self.0.try_ban(addrs, duration)
    }
}

use std::net::{IpAddr, Ipv4Addr};

use alloy_chains::Chain;
use alloy_signer::k256;
use discv5::{ConfigBuilder, Enr, ListenConfig};

use crate::actors::network::TestNetwork;
use alloy_primitives::Address;
use async_trait::async_trait;
use kona_disc::LocalNode;
use kona_genesis::RollupConfig;
use kona_node_service::{
    EngineClientResult, NetworkActor, NetworkBuilder, NetworkEngineClient, NodeActor,
};
use kona_peers::BootNode;
use kona_sources::BlockSigner;
use libp2p::{Multiaddr, identity::Keypair, multiaddr::Protocol};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use rand::RngCore;
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;
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
    pub(crate) fn build(&mut self, bootnodes: Vec<Enr>) -> TestNetwork {
        let keypair = self.custom_keypair.take().unwrap_or(Keypair::generate_secp256k1());

        let secp256k1_key = keypair.clone().try_into_secp256k1()
        .map_err(|e| anyhow::anyhow!("Impossible to convert keypair to secp256k1. This is a bug since we only support secp256k1 keys: {e}")).unwrap()
        .secret().to_bytes();
        let local_node_key = k256::ecdsa::SigningKey::from_bytes(&secp256k1_key.into())
        .map_err(|e| anyhow::anyhow!("Impossible to convert keypair to k256 signing key. This is a bug since we only support secp256k1 keys: {e}")).unwrap();

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
        let builder = NetworkBuilder::new(
            // Create a new rollup config. We don't need to specify any of the fields.
            self.rollup_config(),
            self.unsafe_block_signer,
            gossip_multiaddr,
            keypair,
            LocalNode::new(local_node_key.clone(), node_addr, 0, 0),
            discovery_config,
            Some(BlockSigner::Local(local_node_key.into())),
        )
        .with_bootnodes(bootnodes.into_iter().map(Into::into).collect::<Vec<BootNode>>().into());

        let (blocks_tx, blocks_rx) = mpsc::channel(1024);
        let (inbound_data, actor) = NetworkActor::new(
            ForwardingNetworkEngineClient { blocks_tx },
            CancellationToken::new(),
            builder,
        );

        let handle = tokio::spawn(async move { actor.start(()).await });

        TestNetwork { inbound_data, blocks_rx, handle }
    }
}

#[derive(Debug)]
struct ForwardingNetworkEngineClient {
    blocks_tx: mpsc::Sender<OpExecutionPayloadEnvelope>,
}

#[async_trait]
impl NetworkEngineClient for ForwardingNetworkEngineClient {
    async fn send_unsafe_block(&self, block: OpExecutionPayloadEnvelope) -> EngineClientResult<()> {
        let _ = self
            .blocks_tx
            .send(block)
            .await
            .inspect_err(|e| error!(target: "net", "Failed to send block: {:?}", e));
        Ok(())
    }
}

use std::{
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
use kona_gossip::{GossipTransport, PeerAddr};
use kona_node_service::{
    EngineClientResult, NetworkActor, NetworkBuilder, NetworkEngineClient, NodeActor, SignerActor,
};
use kona_peers::BootNode;
use kona_rpc::P2pRpc;
use kona_sources::BlockSignerHandler;
use libp2p::{
    Multiaddr, Transport,
    core::{transport::MemoryTransport, upgrade::Version},
    identity::Keypair,
    multiaddr::Protocol,
    noise, yamux,
};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use rand::RngCore;
use tokio::sync::{mpsc, watch};
use tracing::{Instrument, error, info_span};

/// Builds [`TestNetwork`]s: nodes that share a random chain ID, so they get no registry bootnodes
/// and find each other only through the bootnodes a test passes in, unless a test sets a chain ID.
pub(crate) struct TestNetworkBuilder {
    /// The chain ID every node built by this builder uses.
    chain_id: u64,
    /// Whether the nodes built from now on run discovery.
    discovery_enabled: bool,
    /// The UDP port the next node built listens on for discovery, if not a free one.
    udp_port: Option<u16>,
    /// The block signer of the nodes built from now on: they only accept blocks it signed.
    unsafe_block_signer: Address,
    /// The keypair for the next node built, if it must not be random.
    custom_keypair: Option<Keypair>,
    /// Whether nodes gossip over the in-memory transport instead of TCP.
    ///
    /// In-memory nodes run without discovery, since it could not connect them: their ENRs would
    /// only advertise IP and TCP, which the memory transport cannot dial. So connect them
    /// explicitly. That makes them fast to test manually configured networks with.
    in_memory: bool,
    /// How often nodes look up random nodes, if not the default.
    discovery_interval: Option<Duration>,
    /// The ID of the next node built, which prefixes its logs.
    next_node_id: usize,
    /// The static peers of the next node built.
    static_peers: Vec<PeerAddr>,
}

impl TestNetworkBuilder {
    fn rollup_config(&self) -> RollupConfig {
        RollupConfig { l2_chain_id: Chain::from_id(self.chain_id), ..Default::default() }
    }

    pub(crate) fn new() -> Self {
        init_test_tracing();
        let chain_id = rand::rng().next_u64();

        Self {
            chain_id,
            discovery_enabled: true,
            udp_port: None,
            unsafe_block_signer: Address::ZERO,
            custom_keypair: None,
            in_memory: false,
            discovery_interval: None,
            next_node_id: 0,
            static_peers: Vec::new(),
        }
    }

    /// Makes the nodes built from now on gossip over the in-memory transport, listening on a free
    /// `/memory` port, and run without discovery. See the `in_memory` field for why.
    pub(crate) const fn in_memory(mut self) -> Self {
        self.in_memory = true;
        self.without_discovery()
    }

    /// Makes the nodes built from now on run without discovery.
    pub(crate) const fn without_discovery(mut self) -> Self {
        self.discovery_enabled = false;
        self
    }

    /// Makes the nodes built from now on use `chain_id`, and so the bootnodes the registry has for
    /// it, instead of a random chain ID.
    pub(crate) const fn with_chain_id(mut self, chain_id: u64) -> Self {
        self.chain_id = chain_id;
        self
    }

    /// Makes the next node built, and only that one, listen for discovery on `port`.
    pub(crate) const fn with_udp_port(mut self, port: u16) -> Self {
        self.udp_port = Some(port);
        self
    }

    /// Makes the nodes built from now on look up random nodes every `interval`, instead of the
    /// default 5s.
    ///
    /// Only for tests that wait on discovery to connect nodes: frequent lookups flood debug logs.
    pub(crate) const fn with_discovery_interval(mut self, interval: Duration) -> Self {
        self.discovery_interval = Some(interval);
        self
    }

    /// Gives the next node built, and only that one, these static peers, which it dials at
    /// startup. The nodes built after it have none, unless this is called again.
    pub(crate) fn with_static_peers(mut self, static_peers: Vec<PeerAddr>) -> Self {
        self.static_peers = static_peers;
        self
    }

    /// Makes every node built from now on accept, and relay, only blocks signed by `signer`.
    /// Nodes built before keep the signer they were built with.
    pub(crate) const fn with_unsafe_block_signer(mut self, signer: Address) -> Self {
        self.unsafe_block_signer = signer;
        self
    }

    /// Gives the next node built, and only that one, this keypair instead of a random one. The
    /// nodes built after it get random keypairs again.
    pub(crate) fn with_custom_keypair(mut self, keypair: Keypair) -> Self {
        self.custom_keypair = Some(keypair);
        self
    }

    /// Makes the next node built the sequencer: it gets a new keypair to sign blocks with, and
    /// every node built from now on, the sequencer included, accepts only blocks signed with it.
    ///
    /// So build the sequencer next, then the nodes that should accept its blocks. A node that must
    /// be built before the sequencer, such as a hub it lists as a static peer, needs
    /// [`Self::with_unsafe_block_signer`] and [`Self::with_custom_keypair`] separately, which is
    /// what this calls.
    pub(crate) fn set_sequencer(self) -> Self {
        let sequencer_keypair = Keypair::generate_secp256k1();
        self.with_unsafe_block_signer(signer_address(&sequencer_keypair))
            .with_custom_keypair(sequencer_keypair)
    }

    /// Builds and starts one node, which bootstraps discovery from `bootnodes`.
    ///
    /// Its discovery table only accepts loopback addresses.
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

        let udp_port = self.udp_port.take().unwrap_or(0);
        let discovery_config = ConfigBuilder::new(ListenConfig::from_ip(node_addr, udp_port))
            // Only allow loopback addresses.
            .table_filter(|enr| {
                let Some(ip) = enr.ip4() else {
                    return false;
                };

                ip.is_loopback()
            })
            .build();

        let gossip_multiaddr = if self.in_memory {
            Multiaddr::empty().with(Protocol::Memory(0))
        } else {
            Multiaddr::from(node_addr).with(Protocol::Tcp(0))
        };

        // Create a new network actor. No external connections
        let mut builder = NetworkBuilder::new(
            // Create a new rollup config. We don't need to specify any of the fields.
            self.rollup_config(),
            self.unsafe_block_signer,
            gossip_multiaddr,
            keypair,
            LocalNode::new(local_node_key, node_addr, 0, udp_port),
            discovery_config,
        )
        .with_discovery_enabled(self.discovery_enabled)
        .with_bootnodes(bootnodes.into_iter().map(Into::into).collect::<Vec<BootNode>>().into())
        .with_static_peers(std::mem::take(&mut self.static_peers));
        if let Some(interval) = self.discovery_interval {
            builder = builder.with_discovery_interval(interval);
        }
        if self.in_memory {
            // A `/memory` address has no IP or TCP port to put in the ENR.
            builder = builder.with_enr_update(false).with_gossip_transport(memory_transport());
        }

        // Logs from this node's tasks are prefixed with `node{id=..}`.
        let span = info_span!(target: NODE_SPAN_TARGET, "node", id = self.next_node_id);
        self.next_node_id += 1;

        let handler = builder
            .build()
            .expect("build network")
            .start()
            .instrument(span.clone())
            .await
            .expect("start network");

        let (blocks_tx, blocks_rx) = mpsc::channel(1024);
        let (gossip_command_tx, gossip_command_rx) = mpsc::channel(1024);
        let (admin_rpc_tx, admin_rpc_rx) = mpsc::channel(1024);
        let (gossip_payload_tx, gossip_payload_rx) = mpsc::channel(256);
        let (signed_payload_tx, signed_payload_rx) = mpsc::channel(16);

        let discovery = handler.discovery.as_ref().map(|discovery| discovery.handler.clone());
        let mut actor = NetworkActor::new(
            ForwardingNetworkEngineClient { blocks_tx },
            handler,
            gossip_command_rx,
            admin_rpc_rx,
            signed_payload_rx,
        );

        let p2p_rpc = P2pRpc::new(
            actor.gossip_query_handle(),
            discovery,
            self.chain_id,
            gossip_command_tx.clone(),
        );

        // Every test network can sign: payloads sent to `gossip_payload_tx` are signed with this
        // node's key and then gossiped.
        let mut signer = SignerActor::new(
            BlockSignerHandler::Local(block_signer),
            self.chain_id,
            watch::channel(self.unsafe_block_signer).1,
            gossip_payload_rx,
            signed_payload_tx,
        );
        tokio::spawn(
            async move {
                let err = loop {
                    if let Err(err) = signer.step().await {
                        break err;
                    }
                };
                error!(target: "net", ?err, "Signer actor failed");
            }
            .instrument(span.clone()),
        );

        let handle = tokio::spawn(
            async move {
                loop {
                    actor.step().await?;
                }
                // Unreachable: the loop above only exits via Err.
                #[allow(unreachable_code)]
                Ok(())
            }
            .instrument(span),
        );

        TestNetwork {
            p2p_rpc,
            gossip_command_tx,
            admin_rpc_tx,
            gossip_payload_tx,
            blocks_rx,
            handle,
        }
    }
}

/// The address a node with `keypair` signs blocks as.
pub(crate) fn signer_address(keypair: &Keypair) -> Address {
    let secp256k1_key = keypair.clone().try_into_secp256k1()
        .map_err(|e| anyhow::anyhow!("Impossible to convert keypair to secp256k1. This is a bug since we only support secp256k1 keys: {e}")).unwrap()
        .secret().to_bytes();
    let signing_key = k256::ecdsa::SigningKey::from_bytes(&secp256k1_key.into())
        .map_err(|e| anyhow::anyhow!("Impossible to convert keypair to k256 signing key. This is a bug since we only support secp256k1 keys: {e}")).unwrap();
    Address::from_private_key(&signing_key)
}

/// The target of the span around each test node's tasks.
const NODE_SPAN_TARGET: &str = "test_node";

/// Logs to the test output, filtered by `RUST_LOG`. Without it, only errors are logged.
///
/// Each line a node's tasks log is prefixed with `node{id=..}`, the order the node was built in.
/// For example, to see warnings from everything and dialing details from the gossip driver:
///
/// ```sh
/// RUST_LOG=warn,gossip=trace cargo nextest run -p kona-node-service test_large_network_conn \
///     --no-capture
/// ```
///
/// `--no-capture` prints the logs of passing tests too. Setting `RUST_LOG` replaces the default of
/// logging errors, so start it with a base level such as `warn`.
fn init_test_tracing() {
    // Spans are filtered like events, so the node spans are always enabled, to prefix whatever
    // `RUST_LOG` lets through. They log nothing themselves.
    let filter = tracing_subscriber::EnvFilter::from_default_env()
        .add_directive(format!("{NODE_SPAN_TARGET}=info").parse().expect("valid directive"));

    // Fails if another test in this binary already installed a subscriber, which is fine.
    let _ = tracing_subscriber::fmt().with_env_filter(filter).with_test_writer().try_init();
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

/// An in-process transport with the upgrades `with_tcp` applies: lazy protocol negotiation, Noise,
/// then Yamux.
fn memory_transport() -> GossipTransport {
    GossipTransport::Other(Box::new(|keypair| {
        Ok(MemoryTransport::default()
            .upgrade(Version::V1Lazy)
            .authenticate(noise::Config::new(keypair)?)
            .multiplex(yamux::Config::default())
            .boxed())
    }))
}

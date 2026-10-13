//! Example of how to use kona's libp2p gossip service as a standalone component
//!
//! ## Usage
//!
//! ```sh
//! cargo run --release -p example-gossip
//! ```
//!
//! ## Inputs
//!
//! The gossip service takes the following inputs:
//!
//! - `-v` or `--verbosity`: Verbosity level (0-2)
//! - `-c` or `--l2-chain-id`: The L2 chain ID to use
//! - `-l` or `--gossip-port`: Port to listen for gossip on
//! - `-i` or `--interval`: Interval to send discovery packets

#![warn(unused_crate_dependencies)]

use async_trait::async_trait;
use clap::Parser;
use enr::CombinedKey;
use kona_cli::{LogArgs, LogConfig};
use kona_disc::LocalNode;
use kona_node_service::{
    EngineClientResult, NetworkActor, NetworkConfig, NetworkEngineClient, NodeActor, discovery,
    network,
};
use kona_registry::ROLLUP_CONFIGS;
use libp2p::Multiaddr;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{
    collections::HashSet,
    net::{IpAddr, Ipv4Addr, SocketAddr},
    time::Duration,
};
use tokio::sync::{mpsc, watch};
use tracing::error;
use tracing_subscriber::EnvFilter;

#[derive(Debug)]
struct ForwardingNetworkEngineClient {
    block_tx: mpsc::Sender<OpExecutionPayloadEnvelope>,
}

#[async_trait]
impl NetworkEngineClient for ForwardingNetworkEngineClient {
    async fn send_unsafe_block(&self, block: OpExecutionPayloadEnvelope) -> EngineClientResult<()> {
        let _ = self
            .block_tx
            .send(block)
            .await
            .inspect_err(|e| error!(target: "net", "Failed to send block: {:?}", e));

        Ok(())
    }
}

/// Passes the network actor's bans to the discovery actor.
#[derive(Debug)]
struct BanningDiscovery(discovery::Handle);

#[async_trait]
impl network::Discovery for BanningDiscovery {
    type Error = discovery::HandleError;

    async fn ban(
        &self,
        addrs: HashSet<Multiaddr>,
        duration: Duration,
    ) -> Result<(), Self::Error> {
        self.0.ban(addrs, duration).await
    }
}

/// The gossip command.
#[derive(Parser, Debug, Clone)]
#[command(about = "Runs the gossip service")]
pub struct GossipCommand {
    #[command(flatten)]
    pub v: LogArgs,
    /// The L2 chain ID to use.
    #[arg(long, short = 'c', default_value = "10", help = "The L2 chain ID to use")]
    pub l2_chain_id: u64,
    /// Port to listen for gossip on.
    #[arg(long, short = 'l', default_value = "9099", help = "Port to listen for gossip on")]
    pub gossip_port: u16,
    /// Port to listen for discovery on.
    #[arg(long, short = 'd', default_value = "9098", help = "Port to listen for discovery on")]
    pub disc_port: u16,
    /// Interval to send discovery packets.
    #[arg(long, short = 'i', default_value = "1", help = "Interval to send discovery packets")]
    pub interval: u64,
}

impl GossipCommand {
    /// Run the gossip subcommand.
    pub async fn run(self) -> anyhow::Result<()> {
        LogConfig::new(self.v).init_tracing_subscriber(None::<EnvFilter>)?;

        let rollup_config = ROLLUP_CONFIGS
            .get(&self.l2_chain_id)
            .ok_or(anyhow::anyhow!("No rollup config found for chain ID"))?;
        let signer = rollup_config
            .genesis
            .system_config
            .as_ref()
            .ok_or(anyhow::anyhow!("No system config found for chain ID"))?
            .batcher_address;
        tracing::debug!(target: "gossip", "Gossip configured with signer: {:?}", signer);

        let gossip = SocketAddr::new(IpAddr::V4(Ipv4Addr::UNSPECIFIED), self.gossip_port);
        tracing::info!(target: "gossip", "Starting gossip driver on {:?}", gossip);

        let mut gossip_addr = Multiaddr::from(gossip.ip());
        gossip_addr.push(libp2p::multiaddr::Protocol::Tcp(gossip.port()));

        let CombinedKey::Secp256k1(secret_key) = CombinedKey::generate_secp256k1() else {
            unreachable!()
        };

        let disc_ip = Ipv4Addr::UNSPECIFIED;
        let disc_addr =
            LocalNode::new(secret_key, IpAddr::V4(disc_ip), self.disc_port, self.disc_port);

        let (unsafe_blocks_tx, mut unsafe_blocks_rx) = mpsc::channel(1024);
        let (gossip_command_tx, gossip_command_rx) = mpsc::channel(1024);
        let (gossip_payload_tx, gossip_payload_rx) = mpsc::channel(256);
        // This example only consumes inbound gossip blocks; the other channels exist solely to
        // satisfy NetworkActor::new and are held to keep them open.
        let _unused_senders = (gossip_command_tx, gossip_payload_tx);

        let network_config = NetworkConfig {
            discovery_interval: Duration::from_secs(self.interval),
            ..NetworkConfig::new(rollup_config.clone(), disc_addr, gossip_addr)
        };
        let (gossip, discovery_driver) = network_config.start(watch::channel(signer).1).await?;
        let discovery_builder = discovery::Builder::new(discovery::Capacity::try_from(1024)?);
        let discovery = discovery_builder.handle();
        let (enrs_tx, enrs) = mpsc::channel(1024);
        let discovery_actor = discovery_builder.build(discovery_driver, enrs_tx);
        tokio::spawn(async move {
            if let Err(e) = discovery_actor.await {
                error!(target: "gossip", "Discovery actor error: {e:?}");
            }
        });

        let mut network = NetworkActor::new(
            ForwardingNetworkEngineClient { block_tx: unsafe_blocks_tx },
            gossip,
            BanningDiscovery(discovery),
            enrs,
            gossip_command_rx,
            gossip_payload_rx,
        );

        tokio::spawn(async move {
            loop {
                if let Err(e) = network.step().await {
                    error!(target: "gossip", "Network actor error: {e:?}");
                    return;
                }
            }
        });

        tracing::info!(target: "gossip", "Gossip driver started, receiving blocks.");
        loop {
            match unsafe_blocks_rx.recv().await {
                Some(block) => {
                    tracing::info!(target: "gossip", "Received unsafe block: {:?}", block);
                }
                None => {
                    tracing::warn!(target: "gossip", "unsafe block gossip channel closed");
                }
            }
        }
    }
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    if let Err(err) = GossipCommand::parse().run().await {
        eprintln!("Error: {err:?}");
        std::process::exit(1);
    }
    Ok(())
}

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

use clap::Parser;
use discv5::enr::CombinedKey;
use kona_cli::{LogArgs, LogConfig};
use kona_disc::LocalNode;
use kona_node_service::{NetworkActor, NetworkConfig, NetworkContext, NodeActor};
use kona_registry::ROLLUP_CONFIGS;
use libp2p::{Multiaddr, identity::Keypair};
use std::{
    net::{IpAddr, Ipv4Addr, SocketAddr},
    time::Duration,
};
use tokio_util::sync::CancellationToken;
use tracing_subscriber::EnvFilter;

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

        let (_, network) = NetworkActor::new(
            NetworkConfig {
                discovery_address: disc_addr,
                gossip_address: gossip_addr,
                unsafe_block_signer: signer,
                discovery_config: discv5::ConfigBuilder::new(discv5::ListenConfig::Ipv4 {
                    ip: disc_ip,
                    port: self.disc_port,
                })
                .build(),
                discovery_interval: Duration::from_secs(self.interval),
                discovery_randomize: None,
                keypair: Keypair::generate_secp256k1(),
                gossip_config: Default::default(),
                scoring: Default::default(),
                topic_scoring: Default::default(),
                monitor_peers: Default::default(),
                bootstore: None,
                gater_config: Default::default(),
                bootnodes: Default::default(),
                rollup_config: rollup_config.clone(),
                gossip_signer: None,
                enr_update: true,
            }
            .into(),
        );

        let (unsafe_blocks_tx, mut unsafe_blocks_rx) = tokio::sync::mpsc::channel(1024);

        network
            .start(NetworkContext {
                blocks: unsafe_blocks_tx,
                cancellation: CancellationToken::new(),
            })
            .await?;

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

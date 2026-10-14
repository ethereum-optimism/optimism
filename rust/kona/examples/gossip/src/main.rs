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
use discv5::Enr;
use enr::CombinedKey;
use kona_cli::{LogArgs, LogConfig};
use kona_disc::LocalNode;
use kona_node_service::{NetworkConfig, discovery, network};
use kona_registry::ROLLUP_CONFIGS;
use libp2p::Multiaddr;
use std::{
    collections::HashSet,
    net::{IpAddr, Ipv4Addr, SocketAddr},
    time::Duration,
};
use tokio::sync::{
    mpsc::{self, error::TrySendError},
    watch,
};
use tracing::error;
use tracing_subscriber::EnvFilter;

/// Passes the peers discovery finds to the network actor.
#[derive(Debug)]
struct DialingNetwork(network::Handle);

impl discovery::Network for DialingNetwork {
    fn dial(&self, enr: Enr) -> Result<(), TrySendError<Enr>> {
        self.0.dial(enr)
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

        let network_config = NetworkConfig {
            discovery_interval: Duration::from_secs(self.interval),
            ..NetworkConfig::new(rollup_config.clone(), disc_addr, gossip_addr)
        };
        let (gossip, discovery_driver) = network_config.start(watch::channel(signer).1).await?;
        let discovery_builder = discovery::Builder::new();
        let discovery = discovery_builder.handle();
        let network_builder = network::Builder::new(gossip);
        let discovery_actor =
            discovery_builder.build(discovery_driver, DialingNetwork(network_builder.handle()));
        tokio::spawn(async move {
            if let Err(e) = discovery_actor.await {
                error!(target: "gossip", "Discovery actor error: {e:?}");
            }
        });

        let network_actor = network_builder.build(unsafe_blocks_tx, BanningDiscovery(discovery));
        tokio::spawn(async move {
            if let Err(e) = network_actor.await {
                error!(target: "gossip", "Network actor error: {e:?}");
            }
        });

        tracing::info!(target: "gossip", "Gossip driver started, receiving blocks.");
        while let Some(block) = unsafe_blocks_rx.recv().await {
            tracing::info!(target: "gossip", "Received unsafe block: {:?}", block);
        }
        tracing::warn!(target: "gossip", "unsafe block gossip channel closed");
        Ok(())
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

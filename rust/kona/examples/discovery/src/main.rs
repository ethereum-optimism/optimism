//! Example of how to use kona's discv5 discovery service as a standalone component
//!
//! ## Usage
//!
//! ```sh
//! cargo run --release -p example-discovery
//! ```
//!
//! ## Inputs
//!
//! The discovery service takes the following inputs:
//!
//! - `-v` or `--verbosity`: Verbosity level (0-2)
//! - `-c` or `--l2-chain-id`: The L2 chain ID to use
//! - `-l` or `--disc-port`: Port to listen for discovery on
//! - `-i` or `--interval`: Interval to send discovery packets

#![warn(unused_crate_dependencies)]

use clap::Parser;
use enr::CombinedKey;
use kona_cli::{LogArgs, LogConfig};
use kona_disc::{Discv5Builder, LocalNode};
use kona_node_actors::discovery;
use std::{
    net::{IpAddr, Ipv4Addr},
    time::Duration,
};
use tokio::sync::mpsc;

/// The discovery command.
#[derive(Parser, Debug, Clone)]
#[command(about = "Runs the discovery service")]
pub struct DiscCommand {
    /// Verbosity level (0-5).
    /// If set to 0, no logs are printed.
    /// By default, the verbosity level is set to 3 (info level).
    #[command(flatten)]
    pub v: LogArgs,
    /// The L2 chain ID to use.
    #[arg(long, short = 'c', default_value = "10", help = "The L2 chain ID to use")]
    pub l2_chain_id: u64,
    /// Discovery port to listen on.
    #[arg(long, short = 'l', default_value = "9099", help = "Port to listen to discovery")]
    pub disc_port: u16,
    /// Interval to send discovery packets.
    #[arg(long, short = 'i', default_value = "3", help = "Interval to send discovery packets")]
    pub interval: u64,
}

impl DiscCommand {
    /// Run the discovery subcommand.
    pub async fn run(self) -> anyhow::Result<()> {
        let filter = tracing_subscriber::EnvFilter::from_default_env()
            .add_directive("discv5=error".parse()?);
        LogConfig::new(self.v).init_tracing_subscriber(Some(filter))?;

        let CombinedKey::Secp256k1(secret_key) = CombinedKey::generate_secp256k1() else {
            unreachable!()
        };

        let socket = LocalNode::new(
            secret_key,
            IpAddr::V4(Ipv4Addr::UNSPECIFIED),
            self.disc_port,
            self.disc_port,
        );
        tracing::info!(target: "discovery", "Starting discovery service on {:?}", socket);

        let driver = Discv5Builder::new(
            socket,
            self.l2_chain_id,
            discv5::ConfigBuilder::new(discv5::ListenConfig::Ipv4 {
                ip: Ipv4Addr::UNSPECIFIED,
                port: self.disc_port,
            })
            .build(),
        )
        .with_interval(Duration::from_secs(self.interval))
        .build()?;
        let builder = discovery::Builder::new();
        let handle = builder.handle();
        let (peers_tx, mut peers) = mpsc::channel(1024);
        let mut actor = tokio::spawn(builder.build(driver, peers_tx));
        tracing::info!(target: "discovery", "Discovery service started, receiving peers.");

        let mut interval = tokio::time::interval(Duration::from_secs(5));
        loop {
            tokio::select! {
                result = &mut actor => {
                    result??;
                    return Ok(());
                }
                Some(enr) = peers.recv() => {
                    tracing::debug!(target: "discovery", "Received peer: {:?}", enr);
                }
                _ = interval.tick() => {
                    match handle.peer_count().await {
                        Ok(count) => tracing::debug!(target: "discovery", "Discovery peer count: {count}"),
                        Err(err) => tracing::warn!(target: "discovery", ?err, "Failed to read the peer count"),
                    }
                }
            }
        }
    }
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    if let Err(err) = DiscCommand::parse().run().await {
        eprintln!("Error: {err:?}");
        std::process::exit(1);
    }
    Ok(())
}

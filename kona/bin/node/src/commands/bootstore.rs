//! Bootstore Subcommand

use crate::flags::GlobalArgs;
use clap::Parser;
use kona_cli::LogConfig;
use kona_peers::{BootStore, BootStoreFile};
use std::path::PathBuf;

/// The `bootstore` Subcommand
///
/// The `bootstore` subcommand can be used to interact with local bootstores.
///
/// # Usage
///
/// ```sh
/// kona-node bootstore [FLAGS] [OPTIONS]
/// ```
#[derive(Parser, Default, PartialEq, Debug, Clone)]
#[command(about = "Utility tool to interact with local bootstores")]
pub struct BootstoreCommand {
    /// Optionally prints all bootstores.
    /// This option overrides the chain ID configured with `--l2-chain-id`.
    #[arg(long = "all")]
    pub all: bool,
    /// The directory to store the bootstore.
    #[arg(long = "p2p.bootstore", env = "KONA_NODE_P2P_BOOTSTORE")]
    pub bootstore: Option<PathBuf>,
}

impl BootstoreCommand {
    /// Initializes the logging system based on global arguments.
    pub fn init_logs(&self, args: &GlobalArgs) -> anyhow::Result<()> {
        LogConfig::new(args.log_args.clone()).init_tracing_subscriber(None)?;
        Ok(())
    }

    /// Runs the subcommand.
    pub fn run(self, args: &GlobalArgs) -> anyhow::Result<()> {
        println!("--------------------------");
        if self.all {
            self.all()?;
        } else {
            self.info(args.l2_chain_id.into())?;
        }
        Ok(())
    }

    /// Prints all bootstores.
    pub fn all(&self) -> anyhow::Result<()> {
        for available in BootStore::available(self.bootstore.clone()) {
            self.info(available)?;
        }
        Ok(())
    }

    /// Prints information for the bootstore with the given chain ID.
    pub fn info(&self, chain_id: u64) -> anyhow::Result<()> {
        let chain = kona_registry::OPCHAINS
            .get(&chain_id)
            .ok_or(anyhow::anyhow!("Chain ID {chain_id} not found in the registry"))?;
        println!("{} Bootstore (Chain ID: {chain_id})", chain.name);
        let bootstore: BootStoreFile = self
            .bootstore
            .clone()
            .map_or(BootStoreFile::Default { chain_id }, BootStoreFile::Custom);
        let bootstore: BootStore = bootstore.try_into()?;
        println!("Path: {}", self.bootstore.clone().unwrap_or_default().display());
        println!("Peer Count: {}", bootstore.peers.len());
        println!("Valid peers: {}", bootstore.valid_peers_with_chain_id(chain_id).len());
        println!("--------------------------");
        Ok(())
    }
}

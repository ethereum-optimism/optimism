//! The `lokahi` CLI.

use crate::{
    config::Config, rpc::AuthorityRpc, source::ChainSource, state::Verified, verifier::Verifier,
    version,
};
use clap::Parser;
use eyre::Result;
use jsonrpsee::server::Server;
use kona_rpc::InteropAuthorityApiServer;
use std::{
    path::PathBuf,
    sync::{Arc, RwLock},
};
use tracing::info;
use tracing_subscriber::EnvFilter;

/// The `lokahi` CLI: verifies cross-chain messages over a set of chains and serves the verified
/// heads their nodes follow.
#[derive(Parser, Clone, Debug)]
#[command(
    author,
    version = version::short_version(),
    long_version = version::long_version(),
    about,
    long_about = None
)]
pub(crate) struct Cli {
    /// Path of the JSON configuration file.
    #[arg(long, env = "LOKAHI_CONFIG")]
    config: PathBuf,
}

impl Cli {
    /// Runs the verifier and its RPC until interrupted.
    pub(crate) async fn run(self) -> Result<()> {
        tracing_subscriber::fmt()
            .json()
            .with_env_filter(
                EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new("info")),
            )
            .init();

        let config = Config::load(&self.config)?;
        let genesis_times = config
            .chains
            .iter()
            .map(|chain| {
                (chain.rollup_config.l2_chain_id.id(), chain.rollup_config.genesis.l2_time)
            })
            .collect();
        let state =
            Arc::new(RwLock::new(Verified::new(config.activation_timestamp, genesis_times)));
        let mut sources = Vec::with_capacity(config.chains.len());
        for chain in config.chains {
            sources.push(
                ChainSource::connect(chain.rollup_config, &chain.cl_rpc, &chain.el_rpc).await?,
            );
        }
        let chain_ids: Vec<_> = sources.iter().map(ChainSource::chain_id).collect();

        let server = Server::builder().build(config.rpc_addr).await?;
        let addr = server.local_addr()?;
        let handle = server.start(AuthorityRpc { state: state.clone() }.into_rpc());
        info!(addr = %addr, ?chain_ids, activation = config.activation_timestamp, "RPC server bound to address");

        let verifier = Verifier::new(
            sources,
            config.dependency_set,
            kona_sources::rpc::l1(config.l1_rpc),
            state,
        );
        tokio::select! {
            () = verifier.run() => {}
            result = tokio::signal::ctrl_c() => {
                result?;
                info!("Received SIGINT, shutting down");
            }
        }
        handle.stop()?;
        Ok(())
    }
}

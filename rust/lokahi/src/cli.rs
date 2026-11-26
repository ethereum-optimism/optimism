//! The `lokahi` CLI.

use crate::{
    config::Config,
    rpc::Rpc,
    source::{self, ChainSource},
    state::Verified,
    store::Store,
    verifier::Verifier,
    version,
};
use clap::Parser;
use eyre::Result;
use jsonrpsee::server::Server;
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
    /// Directory of the verifier's store, created if it does not exist. A restart resumes from
    /// it.
    #[arg(long, env = "LOKAHI_DATADIR")]
    datadir: PathBuf,
    /// Serves the `lokahitest` RPC namespace, which pauses and resumes verification. For tests
    /// only: anyone who reaches the RPC can then hold verification back.
    #[arg(long, env = "LOKAHI_TEST_CONTROL")]
    test_control: bool,
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
        let store = Store::open(&self.datadir)?;
        let state = Verified::open(store, config.activation_timestamp, genesis_times)?;
        match state.activation_timestamp() {
            Some(_) => {
                info!(datadir = %self.datadir.display(), next_timestamp = state.next_timestamp(), "Opened the store")
            }
            None => info!(datadir = %self.datadir.display(), "Opened the store"),
        }
        let state = Arc::new(RwLock::new(state));
        let mut sources = Vec::with_capacity(config.chains.len());
        for chain in config.chains {
            sources.push(
                ChainSource::connect(chain.rollup_config, &chain.cl_rpc, &chain.el_rpc).await?,
            );
        }
        let sources = Arc::new(source::by_chain(sources));
        let chain_ids: Vec<_> = sources.keys().copied().collect();

        let server = Server::builder().build(config.rpc_addr).await?;
        let addr = server.local_addr()?;
        let rpc = Rpc { state: state.clone(), sources: sources.clone() };
        let handle = server.start(rpc.into_module(self.test_control)?);
        info!(addr = %addr, ?chain_ids, activation = ?config.activation_timestamp, test_control = self.test_control, "RPC server bound to address");

        let verify = async {
            if config.activation_timestamp.is_none() {
                info!(
                    "No chain schedules interop: verifying nothing, serving the sources' outputs"
                );
                return std::future::pending().await;
            }
            Verifier::new(
                sources,
                config.dependency_set,
                kona_sources::rpc::l1(config.l1_rpc),
                state,
            )
            .run()
            .await
        };
        tokio::select! {
            result = verify => result?,
            result = tokio::signal::ctrl_c() => {
                result?;
                info!("Received SIGINT, shutting down");
            }
        }
        handle.stop()?;
        Ok(())
    }
}

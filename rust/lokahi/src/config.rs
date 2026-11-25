//! The verifier's configuration file.

use eyre::{Context, Result, bail, ensure};
use kona_genesis::RollupConfig;
use kona_interop::DependencySet;
use serde::Deserialize;
use std::{fs::File, net::SocketAddr, path::Path};
use url::Url;

/// The configuration file, JSON.
#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct ConfigFile {
    /// The L1 execution RPC.
    pub l1_rpc: Url,
    /// Where the authority RPC listens.
    pub rpc_addr: SocketAddr,
    /// Path of the interop dependency set, in the JSON form op-node and kona-node read.
    pub dependency_set: std::path::PathBuf,
    /// The chains to verify.
    pub chains: Vec<ChainFile>,
}

/// One chain of the [`ConfigFile`].
#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
pub(crate) struct ChainFile {
    /// Path of the chain's rollup config.
    pub rollup_config: std::path::PathBuf,
    /// The consensus RPC of the chain's source node: local-safe blocks and their L1 inclusion are
    /// read from its `superroot_atTimestamp`.
    pub cl_rpc: Url,
    /// The execution RPC of the chain's source node: headers and receipts.
    pub el_rpc: Url,
}

/// The loaded configuration.
#[derive(Debug)]
pub(crate) struct Config {
    pub l1_rpc: Url,
    pub rpc_addr: SocketAddr,
    pub dependency_set: DependencySet,
    pub chains: Vec<Chain>,
    /// The first timestamp the verifier covers: interop is active on every chain from it.
    pub activation_timestamp: u64,
}

/// A loaded chain.
#[derive(Debug)]
pub(crate) struct Chain {
    pub rollup_config: RollupConfig,
    pub cl_rpc: Url,
    pub el_rpc: Url,
}

impl Config {
    /// Loads the configuration file at `path` and the files it names.
    pub(crate) fn load(path: &Path) -> Result<Self> {
        let file: ConfigFile = read_json(path)?;
        let dependency_set: DependencySet = read_json(&file.dependency_set)?;
        let chains = file
            .chains
            .into_iter()
            .map(|chain| {
                Ok(Chain {
                    rollup_config: read_json(&chain.rollup_config)?,
                    cl_rpc: chain.cl_rpc,
                    el_rpc: chain.el_rpc,
                })
            })
            .collect::<Result<Vec<_>>>()?;
        let activation_timestamp = activation_timestamp(&chains)?;
        for chain in &chains {
            let chain_id = chain.rollup_config.l2_chain_id.id();
            ensure!(
                dependency_set.dependencies.contains_key(&chain_id),
                "chain {chain_id} is not in the dependency set"
            );
        }
        Ok(Self {
            l1_rpc: file.l1_rpc,
            rpc_addr: file.rpc_addr,
            dependency_set,
            chains,
            activation_timestamp,
        })
    }
}

/// The timestamp from which interop is active on every chain: the latest `lagoon_time`, where a
/// chain activating before its genesis, as at genesis, activates at its genesis.
fn activation_timestamp(chains: &[Chain]) -> Result<u64> {
    if chains.is_empty() {
        bail!("the configuration names no chains");
    }
    chains
        .iter()
        .map(|chain| {
            let config = &chain.rollup_config;
            let lagoon_time = config.hardforks.lagoon_time.ok_or_else(|| {
                eyre::eyre!("chain {} schedules no interop (lagoon_time)", config.l2_chain_id.id())
            })?;
            Ok(lagoon_time.max(config.genesis.l2_time))
        })
        .try_fold(0, |latest, time: Result<u64>| Ok(latest.max(time?)))
}

fn read_json<T: serde::de::DeserializeOwned>(path: &Path) -> Result<T> {
    let file = File::open(path).wrap_err_with(|| format!("open {}", path.display()))?;
    serde_json::from_reader(file).wrap_err_with(|| format!("parse {}", path.display()))
}

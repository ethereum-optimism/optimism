//! The verifier's configuration file.

use eyre::{Context, Result, ensure};
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

/// The timestamp from which interop is active on every chain. Every chain must schedule interop
/// (`lagoon_time`) at the same time, since verification moves over all of them in lockstep; a
/// chain activating before its genesis, as at genesis, activates at its genesis.
fn activation_timestamp(chains: &[Chain]) -> Result<u64> {
    let Some(first) = chains.first() else {
        return Err(eyre::eyre!("the configuration names no chains"));
    };
    let lagoon_time = |config: &RollupConfig| {
        config.hardforks.lagoon_time.ok_or_else(|| {
            eyre::eyre!("chain {} schedules no interop (lagoon_time)", config.l2_chain_id.id())
        })
    };
    let first_time = lagoon_time(&first.rollup_config)?;
    let mut activation = 0;
    for chain in chains {
        let config = &chain.rollup_config;
        let time = lagoon_time(config)?;
        ensure!(
            time == first_time,
            "chain {} activates interop at {time}, chain {} at {first_time}: all chains must activate it at the same time",
            config.l2_chain_id.id(),
            first.rollup_config.l2_chain_id.id()
        );
        activation = activation.max(time.max(config.genesis.l2_time));
    }
    Ok(activation)
}

fn read_json<T: serde::de::DeserializeOwned>(path: &Path) -> Result<T> {
    let file = File::open(path).wrap_err_with(|| format!("open {}", path.display()))?;
    serde_json::from_reader(file).wrap_err_with(|| format!("parse {}", path.display()))
}

#[cfg(test)]
mod tests {
    use super::*;
    use kona_genesis::ChainGenesis;

    fn chain(chain_id: u64, lagoon_time: Option<u64>) -> Chain {
        let mut rollup_config = RollupConfig {
            l2_chain_id: chain_id.into(),
            genesis: ChainGenesis { l2_time: 100, ..Default::default() },
            ..Default::default()
        };
        rollup_config.hardforks.lagoon_time = lagoon_time;
        Chain {
            rollup_config,
            cl_rpc: "http://127.0.0.1:1".parse().unwrap(),
            el_rpc: "http://127.0.0.1:2".parse().unwrap(),
        }
    }

    #[test]
    fn chains_activating_interop_together_activate_at_that_time_or_genesis() {
        assert_eq!(activation_timestamp(&[chain(1, Some(150)), chain(2, Some(150))]).unwrap(), 150);
        assert_eq!(activation_timestamp(&[chain(1, Some(0)), chain(2, Some(0))]).unwrap(), 100);
    }

    #[rstest::rstest]
    #[case::other_times(Some(150), Some(160))]
    #[case::one_without_interop(Some(150), None)]
    fn chains_activating_interop_apart_are_refused(
        #[case] first: Option<u64>,
        #[case] second: Option<u64>,
    ) {
        assert!(activation_timestamp(&[chain(1, first), chain(2, second)]).is_err());
        assert!(activation_timestamp(&[]).is_err());
    }
}

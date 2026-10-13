//! Chain configuration flags.

use crate::flags::OverrideArgs;
use anyhow::{Result, bail};
use clap::{ArgGroup, Parser};
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_interop::{ChainDependency, DependencySet};
use kona_node_service::ChainSpec;
use kona_registry::{DEPENDENCY_SETS, L1Config, scr_rollup_config_by_alloy_ident};
use serde_json::{Value, from_reader, from_value};
use std::{collections::BTreeMap, fs::File, io::Read, path::PathBuf, sync::Arc};
use tracing::debug;

/// Flags that select the chain the node follows and load its configuration.
///
/// The rollup config comes from exactly one of `--chain`, which looks it up in the superchain
/// registry, and `--l2-config-file`.
#[derive(Parser, Debug, Default, Clone, PartialEq, Eq)]
#[command(group(
    ArgGroup::new("rollup_config").required(true).args(["l2_chain_id", "l2_config_file"])
))]
#[command(next_help_heading = "Chain")]
pub struct ChainArgs {
    /// The L2 chain to load the rollup config for from the superchain registry, by chain ID or
    /// name.
    #[arg(long = "chain", env = "KONA_NODE_L2_CHAIN_ID")]
    pub l2_chain_id: Option<alloy_chains::Chain>,
    /// Path to an L2 rollup configuration file, for a chain that is not in the superchain
    /// registry.
    #[arg(long, env = "KONA_NODE_ROLLUP_CONFIG")]
    pub l2_config_file: Option<PathBuf>,
    /// Path to a custom L1 chain configuration or genesis file
    /// (overrides the default L1 configuration from the registry)
    #[arg(long, env = "KONA_NODE_L1_CHAIN_CONFIG")]
    pub l1_config_file: Option<PathBuf>,
    /// Path to a JSON file describing the interop dependency set for this
    /// chain. Mirrors op-node's `--interop.dependency-set`. Required when the
    /// rollup config schedules the Lagoon hardfork; the inner
    /// `StatefulAttributesBuilder` constructor panics otherwise; turning a
    /// silent state-divergence bug into a startup crash.
    #[arg(long = "interop.dependency-set", env = "KONA_NODE_INTEROP_DEPENDENCY_SET")]
    pub interop_dependency_set: Option<PathBuf>,
    /// Hardfork override flags.
    #[command(flatten)]
    pub overrides: OverrideArgs,
}

impl ChainArgs {
    /// Get the L2 rollup config, either from a file or the superchain registry, with the
    /// hardfork overrides applied.
    pub fn rollup_config(&self) -> Result<RollupConfig> {
        let cfg = match (&self.l2_config_file, &self.l2_chain_id) {
            (Some(path), None) => {
                debug!("Loading l2 config from file: {:?}", path);
                let file = File::open(path)
                    .map_err(|e| anyhow::anyhow!("Failed to open l2 config file: {e}"))?;
                from_reader(file).map_err(|e| anyhow::anyhow!("Failed to parse l2 config: {e}"))?
            }
            (None, Some(chain)) => {
                debug!("Loading l2 config from superchain registry");
                let Some(cfg) = scr_rollup_config_by_alloy_ident(chain) else {
                    bail!(
                        "Chain {chain} is not in the superchain registry; \
                         pass its rollup config with --l2-config-file instead"
                    );
                };
                cfg.clone()
            }
            _ => bail!("Exactly one of --chain and --l2-config-file must be set"),
        };
        Ok(self.overrides.apply(cfg))
    }

    /// Resolves the rules of the chain the node follows: the rollup config with the hardfork
    /// overrides applied, the config of the L1 chain it names, and its dependency set.
    pub fn chain_spec(&self) -> Result<ChainSpec> {
        let rollup_config = self.rollup_config()?;
        let l1_config = self.l1_config(rollup_config.l1_chain_id)?;
        let dependency_set = self.dependency_set(&rollup_config)?;
        Ok(ChainSpec {
            rollup_config: Arc::new(rollup_config),
            l1_config: Arc::new(l1_config),
            dependency_set,
        })
    }

    /// Get the L1 config, either from a file or the known chains.
    pub fn l1_config(&self, l1_chain_id: u64) -> Result<L1ChainConfig> {
        match &self.l1_config_file {
            Some(path) => {
                debug!("Loading l1 config from file: {:?}", path);
                let file = File::open(path)
                    .map_err(|e| anyhow::anyhow!("Failed to open l1 config file: {e}"))?;
                Self::parse_l1_config(file)
                    .map_err(|e| anyhow::anyhow!("Failed to parse l1 config: {e}"))
            }
            None => {
                debug!("Loading l1 config from known chains");
                let cfg = L1Config::get_l1_genesis(l1_chain_id).map_err(|e| {
                    anyhow::anyhow!("Failed to find l1 config for chain ID {l1_chain_id}: {e}")
                })?;
                Ok(cfg.into())
            }
        }
    }

    /// Parses an L1 chain config from either a direct config or a genesis document.
    fn parse_l1_config(reader: impl Read) -> serde_json::Result<L1ChainConfig> {
        let mut value: Value = from_reader(reader)?;
        if let Value::Object(object) = &mut value &&
            let Some(config) = object.remove("config")
        {
            return from_value(config);
        }
        from_value(value)
    }

    /// Returns the chain's interop [`DependencySet`], as op-node resolves it: from
    /// `--interop.dependency-set` when given, and otherwise from the superchain registry, by the
    /// rollup config's L2 chain ID.
    ///
    /// For a chain outside the registry, a Lagoon hardfork scheduled without the flag is an error
    /// rather than a guess, since the set decides which upgrade transactions Lagoon's activation
    /// block carries. Any other such chain gets a set containing only itself, which is what the
    /// registry gives every chain outside an interop cluster, and which nothing reads before
    /// Lagoon.
    #[allow(clippy::zero_sized_map_values)] // `DependencySet` is a map to `ChainDependency {}`.
    pub fn dependency_set(&self, cfg: &RollupConfig) -> Result<Arc<DependencySet>> {
        match &self.interop_dependency_set {
            Some(path) => {
                let file = File::open(path).map_err(|e| {
                    anyhow::anyhow!("Failed to open interop dependency-set file {path:?}: {e}")
                })?;
                let dep_set: DependencySet = from_reader(file).map_err(|e| {
                    anyhow::anyhow!("Failed to parse interop dependency-set {path:?}: {e}")
                })?;
                Ok(Arc::new(dep_set))
            }
            None if let Some(dependency_set) = DEPENDENCY_SETS.get(&cfg.l2_chain_id.id()) => {
                Ok(Arc::new(dependency_set.clone()))
            }
            None if cfg.hardforks.lagoon_time.is_some() => bail!(
                "Lagoon is scheduled for this chain (lagoon_time = {:?}), but \
                 --interop.dependency-set was not provided. Supply the dependency-set \
                 JSON file matching op-node's --interop.dependency-set to avoid silent \
                 state divergence on Lagoon activation.",
                cfg.hardforks.lagoon_time,
            ),
            None => Ok(Arc::new(DependencySet {
                dependencies: BTreeMap::from([(cfg.l2_chain_id.id(), ChainDependency {})]),
                override_message_expiry_window: None,
            })),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use rstest::rstest;

    #[rstest]
    #[case::numeric_optimism("10", 10)]
    #[case::numeric_ethereum("1", 1)]
    #[case::numeric_base("8453", 8453)]
    #[case::numeric_unknown("999999", 999999)]
    #[case::string_optimism("optimism", 10)]
    #[case::string_mainnet("mainnet", 1)]
    #[case::string_base("base", 8453)]
    fn test_l2_chain_id_parse_valid(#[case] value: &str, #[case] expected_id: u64) {
        let args = ChainArgs::try_parse_from(["test", "--chain", value]).unwrap();
        assert_eq!(args.l2_chain_id.unwrap().id(), expected_id);
    }

    #[rstest]
    #[case::invalid_string("invalid_chain")]
    #[case::empty_string("")]
    fn test_l2_chain_id_parse_invalid(#[case] invalid_value: &str) {
        let result = ChainArgs::try_parse_from(["test", "--chain", invalid_value]);
        assert!(result.is_err());
    }

    #[rstest]
    #[case::numeric("10", 10)]
    #[case::string("optimism", 10)]
    fn test_l2_chain_id_env_var(#[case] env_value: &str, #[case] expected_id: u64) {
        unsafe {
            std::env::set_var("KONA_NODE_L2_CHAIN_ID", env_value);
        }
        let args = ChainArgs::try_parse_from(["test"]).unwrap();
        assert_eq!(args.l2_chain_id.unwrap().id(), expected_id);
        unsafe {
            std::env::remove_var("KONA_NODE_L2_CHAIN_ID");
        }
    }

    #[test]
    fn test_l2_config_file_env_var() {
        unsafe {
            std::env::set_var("KONA_NODE_ROLLUP_CONFIG", "rollup.json");
        }
        let args = ChainArgs::try_parse_from(["test"]).unwrap();
        assert_eq!(args.l2_config_file, Some(PathBuf::from("rollup.json")));
        assert_eq!(args.l2_chain_id, None);

        // A chain from the command line conflicts with a rollup config file from the environment.
        let err = ChainArgs::try_parse_from(["test", "--chain", "10"]).unwrap_err();
        assert_eq!(err.kind(), clap::error::ErrorKind::ArgumentConflict);
        unsafe {
            std::env::remove_var("KONA_NODE_ROLLUP_CONFIG");
        }
    }

    #[test]
    fn test_rollup_config_source_required() {
        let err = ChainArgs::try_parse_from(["test"]).unwrap_err();
        assert_eq!(err.kind(), clap::error::ErrorKind::MissingRequiredArgument);
    }

    #[test]
    fn test_rollup_config_sources_conflict() {
        let err =
            ChainArgs::try_parse_from(["test", "--chain", "10", "--l2-config-file", "rollup.json"])
                .unwrap_err();
        assert_eq!(err.kind(), clap::error::ErrorKind::ArgumentConflict);
    }

    #[test]
    fn test_rollup_config_applies_overrides() {
        let mut file = tempfile::NamedTempFile::new().unwrap();
        serde_json::to_writer(&mut file, &RollupConfig::default()).unwrap();
        let path = file.path().to_str().unwrap();

        for args in [
            ChainArgs::try_parse_from(["test", "--karst-override", "123", "--chain", "10"])
                .unwrap(),
            ChainArgs::try_parse_from([
                "test",
                "--karst-override",
                "123",
                "--l2-config-file",
                path,
            ])
            .unwrap(),
        ] {
            assert_eq!(args.rollup_config().unwrap().hardforks.karst_time, Some(123));
        }
    }

    #[test]
    fn test_l1_config_from_direct_and_genesis_files() {
        let documents = [
            serde_json::json!({"chainId": 123}),
            serde_json::json!({"config": {"chainId": 123}, "alloc": {}}),
        ];

        for document in documents {
            let mut file = tempfile::NamedTempFile::new().unwrap();
            serde_json::to_writer(&mut file, &document).unwrap();

            let args = ChainArgs { l1_config_file: Some(file.path().into()), ..Default::default() };
            assert_eq!(args.l1_config(123).unwrap().chain_id, 123);
        }
    }

    #[test]
    fn dependency_set_comes_from_the_registry() {
        let args = ChainArgs::try_parse_from(["test", "--chain", "10"]).unwrap();
        let mut cfg =
            RollupConfig { l2_chain_id: alloy_chains::Chain::from_id(10), ..Default::default() };
        // A registry chain needs no flag, even with Lagoon scheduled.
        cfg.hardforks.lagoon_time = Some(0);

        assert_eq!(*args.dependency_set(&cfg).unwrap(), DEPENDENCY_SETS[&10]);
    }

    #[test]
    fn dependency_set_defaults_to_the_chain_alone_outside_the_registry() {
        let args = ChainArgs::try_parse_from(["test", "--chain", "10"]).unwrap();
        let cfg = RollupConfig {
            l2_chain_id: alloy_chains::Chain::from_id(999_999),
            ..Default::default()
        };
        assert!(!DEPENDENCY_SETS.contains_key(&999_999));

        let dependency_set = args.dependency_set(&cfg).unwrap();

        assert_eq!(dependency_set.dependencies.keys().copied().collect::<Vec<_>>(), [999_999]);
        assert_eq!(dependency_set.override_message_expiry_window, None);
    }

    #[test]
    fn dependency_set_is_required_when_lagoon_is_scheduled_outside_the_registry() {
        let args = ChainArgs::try_parse_from(["test", "--chain", "10"]).unwrap();
        let mut cfg = RollupConfig::default();
        assert!(!DEPENDENCY_SETS.contains_key(&cfg.l2_chain_id.id()));
        cfg.hardforks.lagoon_time = Some(0);

        assert!(args.dependency_set(&cfg).is_err());
    }

    #[test]
    #[allow(clippy::zero_sized_map_values)]
    fn dependency_set_is_loaded_from_the_flag() {
        let set = DependencySet {
            dependencies: BTreeMap::from([(10, ChainDependency {}), (11, ChainDependency {})]),
            override_message_expiry_window: Some(60),
        };
        let mut file = tempfile::NamedTempFile::new().unwrap();
        serde_json::to_writer(&mut file, &set).unwrap();
        let path = file.path().to_str().unwrap();
        let args =
            ChainArgs::try_parse_from(["test", "--chain", "10", "--interop.dependency-set", path])
                .unwrap();
        let mut cfg = RollupConfig::default();
        cfg.hardforks.lagoon_time = Some(0);

        assert_eq!(*args.dependency_set(&cfg).unwrap(), set);
    }
}

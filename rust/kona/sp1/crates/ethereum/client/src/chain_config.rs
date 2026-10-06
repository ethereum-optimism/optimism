//! Trusted chain configurations shared by range and consolidation execution.

use std::collections::BTreeMap;

use alloy_primitives::U256;
use anyhow::{anyhow, bail, ensure};
use kona_genesis::{L1ChainConfig, RollupConfig};
use kona_interop::DependencySet;
use kona_registry::{DEPENDENCY_SETS, L1_CONFIGS, ROLLUP_CONFIGS};

/// Resolved chain rules supplied by the caller, independent of the preimage witness.
///
/// Overrides must come from a trusted caller; production guests must resolve their own registry.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ChainConfigs {
    /// Interop dependency set for the input chains.
    pub dependency_set: DependencySet,
    /// Rollup configurations keyed by L2 chain ID.
    pub rollup_configs: BTreeMap<u64, RollupConfig>,
    /// L1 chain configuration shared by all input chains.
    pub l1_config: L1ChainConfig,
}

impl ChainConfigs {
    /// Resolves chain rules exclusively from the registry compiled into this binary.
    pub fn from_registry(input_chain_ids: &[U256]) -> anyhow::Result<Self> {
        let chain_ids = input_chain_ids_as_u64(input_chain_ids)?;
        let dependency_set = chain_ids
            .first()
            .and_then(|chain_id| DEPENDENCY_SETS.get(chain_id))
            .cloned()
            .ok_or_else(|| {
                anyhow!("no embedded dependency set for super-range chain IDs {chain_ids:?}")
            })?;
        let rollup_configs = chain_ids
            .iter()
            .map(|chain_id| {
                let config = ROLLUP_CONFIGS.get(chain_id).cloned().ok_or_else(|| {
                    anyhow!("no embedded rollup config for super-range chain ID {chain_id}")
                })?;
                Ok((*chain_id, config))
            })
            .collect::<anyhow::Result<BTreeMap<_, _>>>()?;
        let l1_chain_id = rollup_configs
            .values()
            .next()
            .ok_or_else(|| anyhow!("super-range rollup config set is empty"))?
            .l1_chain_id;
        let l1_config = L1_CONFIGS
            .get(&l1_chain_id)
            .cloned()
            .ok_or_else(|| anyhow!("no embedded L1 config for chain ID {l1_chain_id}"))?;
        let configs = Self { dependency_set, rollup_configs, l1_config };
        configs.validate(input_chain_ids)?;
        Ok(configs)
    }

    /// Checks exact chain coverage and consistency of the supplied chain rules.
    pub fn validate(&self, input_chain_ids: &[U256]) -> anyhow::Result<()> {
        input_chain_ids_as_u64(input_chain_ids)?;
        ensure_dependency_set_matches_inputs(input_chain_ids, &self.dependency_set)?;
        ensure_rollup_configs_match_inputs(input_chain_ids, &self.rollup_configs)?;
        let first_l1_chain_id = self
            .rollup_configs
            .values()
            .next()
            .ok_or_else(|| anyhow!("super-range rollup config set is empty"))?
            .l1_chain_id;
        for config in self.rollup_configs.values() {
            ensure!(
                config.l1_chain_id == first_l1_chain_id,
                "super-range rollup configs must share one L1 chain ID, got {first_l1_chain_id} and {}",
                config.l1_chain_id
            );
        }
        ensure!(
            self.l1_config.chain_id == first_l1_chain_id,
            "L1 config chain ID {} does not match rollup L1 chain ID {first_l1_chain_id}",
            self.l1_config.chain_id
        );
        Ok(())
    }
}

fn ensure_rollup_configs_match_inputs(
    input_chain_ids: &[U256],
    rollup_configs: &BTreeMap<u64, RollupConfig>,
) -> anyhow::Result<()> {
    let config_chain_ids = rollup_configs.keys().copied().map(U256::from).collect::<Vec<_>>();
    ensure!(
        input_chain_ids == config_chain_ids,
        "super-range chain IDs {input_chain_ids:?} must exactly match rollup config chain IDs {config_chain_ids:?}",
    );

    for (chain_id, config) in rollup_configs {
        ensure!(
            config.l2_chain_id.id() == *chain_id,
            "rollup config key {chain_id} does not match config L2 chain ID {actual}",
            actual = config.l2_chain_id.id(),
        );
    }

    Ok(())
}

fn input_chain_ids_as_u64(input_chain_ids: &[U256]) -> anyhow::Result<Vec<u64>> {
    input_chain_ids
        .iter()
        .map(|chain_id| {
            if *chain_id > U256::from(u64::MAX) {
                bail!("super-range chain ID {chain_id} does not fit in dependency set keys");
            }
            Ok(chain_id.saturating_to::<u64>())
        })
        .collect()
}

pub(crate) fn ensure_dependency_set_matches_inputs(
    input_chain_ids: &[U256],
    dependency_set: &DependencySet,
) -> anyhow::Result<()> {
    let dependency_set_chain_ids =
        dependency_set.dependencies.keys().copied().map(U256::from).collect::<Vec<_>>();
    ensure!(
        input_chain_ids == dependency_set_chain_ids,
        "super-range chain IDs {input_chain_ids:?} must exactly match dependency set chain IDs {dependency_set_chain_ids:?}",
    );

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_utils::chain_configs;

    #[test]
    fn explicit_configs_reject_inconsistent_chain_rules() {
        let chain_ids = [U256::from(10), U256::from(20)];
        let valid = chain_configs(&[10, 20]);
        valid.validate(&chain_ids).unwrap();
        let mut invalid = valid.clone();
        invalid.rollup_configs.remove(&20);
        assert!(
            invalid.validate(&chain_ids).unwrap_err().to_string().contains("must exactly match")
        );
        let mut invalid = valid.clone();
        invalid.rollup_configs.get_mut(&20).unwrap().l2_chain_id = 30.into();
        assert!(
            invalid
                .validate(&chain_ids)
                .unwrap_err()
                .to_string()
                .contains("does not match config L2 chain ID")
        );
        let mut invalid = valid.clone();
        invalid.rollup_configs.get_mut(&20).unwrap().l1_chain_id = 2;
        assert!(
            invalid
                .validate(&chain_ids)
                .unwrap_err()
                .to_string()
                .contains("must share one L1 chain ID")
        );
        let mut invalid = valid;
        invalid.l1_config.chain_id = 2;
        assert!(
            invalid
                .validate(&chain_ids)
                .unwrap_err()
                .to_string()
                .contains("does not match rollup L1 chain ID")
        );
    }

    #[test]
    fn registry_configs_reject_partial_cluster_and_oversized_chain_ids() {
        assert!(
            ChainConfigs::from_registry(&[U256::from(10), U256::from(u64::MAX)])
                .unwrap_err()
                .to_string()
                .contains("no embedded rollup config")
        );
        assert!(
            ChainConfigs::from_registry(&[U256::MAX])
                .unwrap_err()
                .to_string()
                .contains("does not fit in dependency set keys")
        );
    }
}

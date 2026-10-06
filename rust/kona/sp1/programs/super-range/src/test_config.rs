//! Unverified chain configs for synthetic-chain test guests only.

use anyhow::Context;
use kona_preimage::{
    DEPENDENCY_SET_KEY, L1_CONFIG_KEY, L2_ROLLUP_CONFIG_KEY, PreimageKey, PreimageOracleClient,
};
use kona_sp1_client_utils::{
    super_root::SuperInteropInputs, witness::preimage_store::PreimageStore,
};
use kona_sp1_ethereum_client_utils::chain_config::ChainConfigs;

/// Present only in test-config-fallback ELFs; prestate builds reject any ELF that contains it.
/// Scanned by build-prestates.sh, rust-ci.yml, and rust-e2e.yml; change all together.
pub(super) const MARKER: &str =
    "KONA_SP1_UNSAFE_TEST_CONFIG_FALLBACK{fd6d88e711058eef5eff1512237c8ad3}";

pub(super) async fn load(
    inputs: &SuperInteropInputs,
    oracle: &PreimageStore,
) -> anyhow::Result<ChainConfigs> {
    let chain_ids = match inputs {
        SuperInteropInputs::Range(inputs) => inputs.chain_ids.clone(),
        SuperInteropInputs::Consolidation(inputs) => inputs
            .transitions
            .first()
            .map(|transition| {
                transition.optimistic_blocks.iter().map(|block| block.chain_id).collect()
            })
            .unwrap_or_default(),
    };
    if let Ok(configs) = ChainConfigs::from_registry(&chain_ids) {
        return Ok(configs);
    }
    let dependency_set = oracle.get(PreimageKey::new_local(DEPENDENCY_SET_KEY.to())).await?;
    let rollup_configs = oracle.get(PreimageKey::new_local(L2_ROLLUP_CONFIG_KEY.to())).await?;
    let l1_config = oracle.get(PreimageKey::new_local(L1_CONFIG_KEY.to())).await?;
    let configs = ChainConfigs {
        dependency_set: serde_json::from_slice(&dependency_set)
            .context("failed to decode test dependency set")?,
        rollup_configs: serde_json::from_slice(&rollup_configs)
            .context("failed to decode test rollup configs")?,
        l1_config: serde_json::from_slice(&l1_config).context("failed to decode test L1 config")?,
    };
    configs.validate(&chain_ids)?;
    Ok(configs)
}

#[cfg(test)]
mod tests {
    use super::*;
    use kona_sp1_client_utils::super_root::{SuperRangeInputs, TimestampSpan};

    #[test]
    fn test_guest_loads_complete_synthetic_configs_and_rejects_mismatched_l1() {
        let mut configs = ChainConfigs::from_registry(&["0xa".parse().unwrap()]).unwrap();
        configs.l1_config = Default::default();
        let mut rollup = configs.rollup_configs.remove(&10).unwrap();
        rollup.l2_chain_id = u64::MAX.into();
        configs.rollup_configs.insert(u64::MAX, rollup);
        let dependency = configs.dependency_set.dependencies.remove(&10).unwrap();
        configs.dependency_set.dependencies.insert(u64::MAX, dependency);
        let inputs = SuperInteropInputs::Range(SuperRangeInputs {
            span: TimestampSpan::new(101, 101).unwrap(),
            l1_head: Default::default(),
            chain_ids: vec![u64::MAX.to_string().parse().unwrap()],
            previous_super_root_proofs: Vec::new(),
            claimed_transitions: Vec::new(),
        });
        for mismatched_l1 in [false, true] {
            let mut supplied = configs.clone();
            if mismatched_l1 {
                supplied.l1_config.chain_id = 2;
            }
            let mut oracle = PreimageStore::default();
            for (key, bytes) in [
                (DEPENDENCY_SET_KEY, serde_json::to_vec(&supplied.dependency_set).unwrap()),
                (L2_ROLLUP_CONFIG_KEY, serde_json::to_vec(&supplied.rollup_configs).unwrap()),
                (L1_CONFIG_KEY, serde_json::to_vec(&supplied.l1_config).unwrap()),
            ] {
                oracle.save_preimage(PreimageKey::new_local(key.to()), bytes).unwrap();
            }
            let result = kona_proof::block_on(load(&inputs, &oracle));
            if mismatched_l1 {
                assert!(
                    result.unwrap_err().to_string().contains("does not match rollup L1 chain ID")
                );
            } else {
                assert_eq!(result.unwrap(), configs);
            }
        }
    }
}

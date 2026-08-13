#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(feature = "std"), no_std)]

extern crate alloc;

use alloc::vec::Vec;

pub use alloy_primitives::map::HashMap;
use kona_genesis::L1ChainConfig;
pub use kona_genesis::{
    Chain, ChainConfig, ChainDependency, ChainList, DependencySet, MESSAGE_EXPIRY_WINDOW,
    RollupConfig,
};

pub mod superchain;
pub use superchain::Registry;

/// L1 chain configurations.
pub mod l1;
pub use l1::L1Config;

#[cfg(test)]
pub mod test_utils;

lazy_static::lazy_static! {
    /// Private initializer that loads the superchain configurations.
    static ref _INIT: Registry = Registry::from_chain_list();

    /// Chain configurations exported from the registry
    pub static ref CHAINS: ChainList = _INIT.chain_list.clone();

    /// OP Chain configurations exported from the registry
    pub static ref OPCHAINS: HashMap<u64, ChainConfig> = _INIT.op_chains.clone();

    /// Rollup configurations exported from the registry
    pub static ref ROLLUP_CONFIGS: HashMap<u64, RollupConfig> = _INIT.rollup_configs.clone();

    /// L1 chain configurations exported from the registry
    /// Note: the l1 chain configurations are not exported from the superchain registry but rather from a genesis dump file.
    pub static ref L1_CONFIGS: HashMap<u64, L1ChainConfig> = _INIT.l1_configs.clone();

    /// All interop dependency sets embedded into this binary, keyed by L2 chain id.
    /// Each chain that belongs to an interop cluster maps to that cluster's
    /// [`DependencySet`]; chains in disjoint clusters map to **different** values.
    /// Cross-cluster proofs must be rejected by the consumer (see `BootInfo::load`).
    pub static ref DEPENDENCY_SETS: HashMap<u64, DependencySet> = {
        let raw = include_str!(concat!(env!("KONA_REGISTRY_DIR"), "/depsets.json"));
        let depsets: Vec<DependencySet> = serde_json::from_str(raw)
            .expect("parse embedded etc/depsets.json");
        let mut by_chain: HashMap<u64, DependencySet> = HashMap::default();
        for ds in depsets {
            for chain_id in ds.dependencies.keys().copied() {
                if let Some(existing) = by_chain.insert(chain_id, ds.clone()) {
                    assert_eq!(
                        existing, ds,
                        "embedded depsets contain overlapping clusters; build script bug"
                    );
                }
            }
        }
        by_chain
    };
}

/// Returns a [`RollupConfig`] by its identifier.
pub fn scr_rollup_config_by_ident(ident: &str) -> Option<&RollupConfig> {
    let chain_id = CHAINS.get_chain_by_ident(ident)?.chain_id;
    ROLLUP_CONFIGS.get(&chain_id)
}

/// Returns a [`RollupConfig`] by its identifier.
pub fn scr_rollup_config_by_alloy_ident(chain: &alloy_chains::Chain) -> Option<&RollupConfig> {
    ROLLUP_CONFIGS.get(&chain.id())
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_chains::Chain as AlloyChain;
    use alloy_hardforks::{
        holesky::{HOLESKY_BPO1_TIMESTAMP, HOLESKY_BPO2_TIMESTAMP},
        sepolia::{SEPOLIA_BPO1_TIMESTAMP, SEPOLIA_BPO2_TIMESTAMP},
    };
    use alloy_op_hardforks::{OP_MAINNET_JOVIAN_TIMESTAMP, OP_SEPOLIA_JOVIAN_TIMESTAMP};

    #[test]
    fn test_hardcoded_rollup_configs() {
        let test_cases =
            [(10, test_utils::OP_MAINNET_CONFIG), (11155420, test_utils::OP_SEPOLIA_CONFIG)]
                .to_vec();

        for (chain_id, expected) in test_cases {
            let derived = super::ROLLUP_CONFIGS.get(&chain_id).unwrap();
            assert_eq!(expected, *derived);
        }
    }

    #[test]
    fn test_chain_by_ident() {
        const ALLOY_OP: AlloyChain = AlloyChain::optimism_mainnet();

        let chain_by_ident = CHAINS.get_chain_by_ident("mainnet/op").unwrap();
        let chain_by_alloy_ident = CHAINS.get_chain_by_alloy_ident(&ALLOY_OP).unwrap();
        let chain_by_id = CHAINS.get_chain_by_id(10).unwrap();

        assert_eq!(chain_by_ident, chain_by_id);
        assert_eq!(chain_by_alloy_ident, chain_by_id);
    }

    #[test]
    fn test_rollup_config_by_ident() {
        const ALLOY_OP: AlloyChain = AlloyChain::optimism_mainnet();

        let rollup_config_by_ident = scr_rollup_config_by_ident("mainnet/op").unwrap();
        let rollup_config_by_alloy_ident = scr_rollup_config_by_alloy_ident(&ALLOY_OP).unwrap();
        let rollup_config_by_id = ROLLUP_CONFIGS.get(&10).unwrap();

        assert_eq!(rollup_config_by_ident, rollup_config_by_id);
        assert_eq!(rollup_config_by_alloy_ident, rollup_config_by_id);
    }

    #[test]
    fn test_jovian_timestamps() {
        let op_mainnet_config_by_ident = scr_rollup_config_by_ident("mainnet/op").unwrap();
        assert_eq!(
            op_mainnet_config_by_ident.hardforks.jovian_time,
            Some(OP_MAINNET_JOVIAN_TIMESTAMP)
        );

        let op_sepolia_config_by_ident = scr_rollup_config_by_ident("sepolia/op").unwrap();
        assert_eq!(
            op_sepolia_config_by_ident.hardforks.jovian_time,
            Some(OP_SEPOLIA_JOVIAN_TIMESTAMP)
        );
    }

    #[test]
    fn test_bpo_timestamps() {
        let sepolia_config = L1_CONFIGS.get(&11155111).unwrap();
        assert_eq!(sepolia_config.bpo1_time, Some(SEPOLIA_BPO1_TIMESTAMP));
        assert_eq!(sepolia_config.bpo2_time, Some(SEPOLIA_BPO2_TIMESTAMP));

        let holesky_config = L1_CONFIGS.get(&17000).unwrap();
        assert_eq!(holesky_config.bpo1_time, Some(HOLESKY_BPO1_TIMESTAMP));
        assert_eq!(holesky_config.bpo2_time, Some(HOLESKY_BPO2_TIMESTAMP));
    }

    const CUSTOM_CONFIGS_TEST_ENABLED: Option<&str> = option_env!("KONA_CUSTOM_CONFIGS_TEST");
    const CUSTOM_CONFIGS: Option<&str> = option_env!("KONA_CUSTOM_CONFIGS");
    const CUSTOM_CONFIGS_DIR: Option<&str> = option_env!("KONA_CUSTOM_CONFIGS_DIR");
    const CUSTOM_CONFIGS_CFG: bool = cfg!(kona_custom_configs = "true");

    #[test]
    fn custom_chain_is_loaded_when_enabled() {
        if CUSTOM_CONFIGS_TEST_ENABLED != Some("true") {
            return;
        };
        assert!(
            CUSTOM_CONFIGS == Some("true") || CUSTOM_CONFIGS_CFG,
            "KONA_CUSTOM_CONFIGS=true or --cfg kona_custom_configs=\"true\" is required when \
             KONA_CUSTOM_CONFIGS_TEST is set"
        );
        assert!(
            CUSTOM_CONFIGS_DIR.is_some() || CUSTOM_CONFIGS_CFG,
            "KONA_CUSTOM_CONFIGS_DIR or --cfg kona_custom_configs_dir=\"...\" is required when \
             KONA_CUSTOM_CONFIGS_TEST is set"
        );
        assert_eq!(
            env!("KONA_REGISTRY_DIR").strip_prefix(env!("OUT_DIR")),
            Some("/registry-etc"),
            "custom configs must be merged outside the committed registry snapshot"
        );

        let test1_chain_id = 123999119;
        let test2_chain_id = 223999119;
        let test1_ident = "test1/testnet";
        let test2_ident = "test2/testnet";

        let chain1 = CHAINS
            .get_chain_by_ident(test1_ident)
            .unwrap_or_else(|| panic!("custom chain `{test1_ident}` missing"));
        assert_eq!(chain1.chain_id, test1_chain_id);
        let chain2 = CHAINS
            .get_chain_by_ident(test2_ident)
            .unwrap_or_else(|| panic!("custom chain `{test2_ident}` missing"));
        assert_eq!(chain2.chain_id, test2_chain_id);

        assert!(
            OPCHAINS.contains_key(&test1_chain_id),
            "chain config missing for {test1_chain_id}"
        );
        assert!(
            ROLLUP_CONFIGS.contains_key(&test1_chain_id),
            "rollup config missing for {test1_chain_id}"
        );
        assert!(
            OPCHAINS.contains_key(&test2_chain_id),
            "chain config missing for {test2_chain_id}"
        );
        assert!(
            ROLLUP_CONFIGS.contains_key(&test2_chain_id),
            "rollup config missing for {test2_chain_id}"
        );

        let depset = DEPENDENCY_SETS
            .get(&test1_chain_id)
            .expect("test1 chain id present in embedded depsets");
        assert!(depset.dependencies.contains_key(&test1_chain_id));
        assert!(depset.dependencies.contains_key(&test2_chain_id));
        // Both chain ids must map to the SAME depset value (cluster identity).
        assert_eq!(DEPENDENCY_SETS.get(&test1_chain_id), DEPENDENCY_SETS.get(&test2_chain_id));
    }

    /// Custom-config chains keep the preimage-oracle fallback, so they are exempt from the
    /// registry depset invariants below.
    fn custom_configs_baked_in() -> bool {
        CUSTOM_CONFIGS == Some("true") || CUSTOM_CONFIGS_CFG
    }

    #[test]
    fn test_every_registry_chain_has_a_depset() {
        if custom_configs_baked_in() {
            return;
        }
        for chain in &CHAINS.chains {
            let depset = DEPENDENCY_SETS.get(&chain.chain_id).unwrap_or_else(|| {
                panic!(
                    "no embedded depset for `{}` (chain id {})",
                    chain.identifier, chain.chain_id
                )
            });
            assert!(
                depset.dependencies.contains_key(&chain.chain_id),
                "depset for `{}` does not contain the chain itself",
                chain.identifier
            );
        }
    }

    #[test]
    fn test_chains_without_interop_config_get_self_only_depsets() {
        if custom_configs_baked_in() {
            return;
        }
        for (chain_id, config) in OPCHAINS.iter().filter(|(_, c)| c.interop.is_none()) {
            let depset = DEPENDENCY_SETS
                .get(chain_id)
                .unwrap_or_else(|| panic!("no embedded depset for chain id {chain_id}"));
            assert_eq!(
                depset.dependencies.keys().copied().collect::<Vec<_>>(),
                alloc::vec![*chain_id],
                "`{}` declares no interop dependencies but is not a single-chain cluster",
                config.name
            );
            assert!(depset.override_message_expiry_window.is_none());
        }
    }

    // TODO(#21760): Add the registry-derived interop cluster test back when
    // there are interop-staging networks in the superchain registry. @jelias2
}

//! Contains the full superchain data.

use crate::L1Config;

use super::ChainList;
use alloy_primitives::map::HashMap;
use kona_genesis::{ChainConfig, L1ChainConfig, RollupConfig, Superchains};

/// The registry containing all the superchain configurations.
#[derive(Debug, Clone, Default, Eq, PartialEq)]
pub struct Registry {
    /// The list of chains.
    pub chain_list: ChainList,
    /// Map of chain IDs to their chain configuration.
    pub op_chains: HashMap<u64, ChainConfig>,
    /// Map of chain IDs to their rollup configurations.
    pub rollup_configs: HashMap<u64, RollupConfig>,
    /// Map of l1 chain IDs to their l1 configurations.
    pub l1_configs: HashMap<u64, L1ChainConfig>,
}

impl Registry {
    /// Read the chain list.
    pub fn read_chain_list() -> ChainList {
        let chain_list = include_str!("../etc/chainList.json");
        serde_json::from_str(chain_list).expect("Failed to read chain list")
    }

    /// Read superchain configs.
    pub fn read_superchain_configs() -> Superchains {
        let superchain_configs = include_str!("../etc/configs.json");
        serde_json::from_str(superchain_configs).expect("Failed to read superchain configs")
    }

    /// Initialize the superchain configurations from the chain list.
    pub fn from_chain_list() -> Self {
        let chain_list = Self::read_chain_list();
        let superchains = Self::read_superchain_configs();
        let mut op_chains = HashMap::default();
        let mut rollup_configs = HashMap::default();

        for superchain in superchains.superchains {
            for mut chain_config in superchain.chains {
                chain_config.l1_chain_id = superchain.config.l1.chain_id;
                if let Some(a) = &mut chain_config.addresses {
                    a.zero_proof_addresses();
                }
                let mut rollup = chain_config.as_rollup_config();
                rollup.superchain_config_address = superchain.config.superchain_config_addr;
                rollup_configs.insert(chain_config.chain_id, rollup);
                op_chains.insert(chain_config.chain_id, chain_config);
            }
        }

        Self { chain_list, op_chains, rollup_configs, l1_configs: L1Config::build_l1_configs() }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_op_hardforks::{
        OP_MAINNET_ISTHMUS_TIMESTAMP, OP_MAINNET_JOVIAN_TIMESTAMP, OP_SEPOLIA_ISTHMUS_TIMESTAMP,
        OP_SEPOLIA_JOVIAN_TIMESTAMP,
    };
    use alloy_primitives::address;

    #[test]
    fn test_read_chain_configs() {
        let superchains = Registry::from_chain_list();
        assert!(superchains.chain_list.len() > 1);
        let op_mainnet = superchains.op_chains.get(&10).expect("OP Mainnet config missing");
        assert_eq!(op_mainnet.name, "OP Mainnet");
        assert_eq!(op_mainnet.chain_id, 10);
        assert_eq!(op_mainnet.l1_chain_id, 1);
        assert_eq!(
            op_mainnet.batch_inbox_addr,
            address!("ff00000000000000000000000000000000000010")
        );
        assert!(op_mainnet.governed_by_optimism);
    }

    #[test]
    fn test_read_rollup_configs() {
        let superchains = Registry::from_chain_list();
        assert_eq!(
            *superchains.rollup_configs.get(&10).unwrap(),
            crate::test_utils::OP_MAINNET_CONFIG
        );
    }

    #[test]
    fn test_isthmus_timestamps() {
        let superchains = Registry::from_chain_list();
        let op_mainnet_config = superchains.rollup_configs.get(&10).unwrap();
        assert_eq!(op_mainnet_config.hardforks.isthmus_time, Some(OP_MAINNET_ISTHMUS_TIMESTAMP));

        let op_sepolia_config = superchains.rollup_configs.get(&11155420).unwrap();
        assert_eq!(op_sepolia_config.hardforks.isthmus_time, Some(OP_SEPOLIA_ISTHMUS_TIMESTAMP));
    }

    #[test]
    fn test_jovian_timestamps() {
        let superchains = Registry::from_chain_list();
        let op_mainnet_config = superchains.rollup_configs.get(&10).unwrap();
        assert_eq!(op_mainnet_config.hardforks.jovian_time, Some(OP_MAINNET_JOVIAN_TIMESTAMP));

        let op_sepolia_config = superchains.rollup_configs.get(&11155420).unwrap();
        assert_eq!(op_sepolia_config.hardforks.jovian_time, Some(OP_SEPOLIA_JOVIAN_TIMESTAMP));
    }
}

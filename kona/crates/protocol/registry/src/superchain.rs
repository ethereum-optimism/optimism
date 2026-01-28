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
                rollup.protocol_versions_address = superchain
                    .config
                    .protocol_versions_addr
                    .expect("Missing protocol versions address");
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
    use alloc::string::{String, ToString};
    use alloy_op_hardforks::{
        BASE_MAINNET_ISTHMUS_TIMESTAMP, BASE_MAINNET_JOVIAN_TIMESTAMP,
        BASE_SEPOLIA_ISTHMUS_TIMESTAMP, BASE_SEPOLIA_JOVIAN_TIMESTAMP,
        OP_MAINNET_ISTHMUS_TIMESTAMP, OP_MAINNET_JOVIAN_TIMESTAMP, OP_SEPOLIA_ISTHMUS_TIMESTAMP,
        OP_SEPOLIA_JOVIAN_TIMESTAMP,
    };
    use alloy_primitives::address;
    use kona_genesis::{AddressList, OP_MAINNET_BASE_FEE_CONFIG, Roles, SuperchainLevel};

    #[test]
    fn test_read_chain_configs() {
        let superchains = Registry::from_chain_list();
        assert!(superchains.chain_list.len() > 1);
        let base_config = ChainConfig {
            name: String::from("Base"),
            chain_id: 8453,
            l1_chain_id: 1,
            public_rpc: String::from("https://mainnet.base.org"),
            sequencer_rpc: String::from("https://mainnet-sequencer.base.org"),
            explorer: String::from("https://explorer.base.org"),
            superchain_level: SuperchainLevel::StandardCandidate,
            governed_by_optimism: false,
            superchain_time: Some(0),
            batch_inbox_addr: address!("ff00000000000000000000000000000000008453"),
            hardfork_config: crate::test_utils::BASE_MAINNET_CONFIG.hardforks,
            block_time: 2,
            seq_window_size: 3600,
            max_sequencer_drift: 600,
            data_availability_type: "eth-da".to_string(),
            optimism: Some(OP_MAINNET_BASE_FEE_CONFIG),
            alt_da: None,
            genesis: crate::test_utils::BASE_MAINNET_CONFIG.genesis,
            roles: Some(Roles {
                proxy_admin_owner: Some(
                    "7bB41C3008B3f03FE483B28b8DB90e19Cf07595c".parse().unwrap(),
                ),
                ..Default::default()
            }),
            addresses: Some(AddressList {
                l1_standard_bridge_proxy: Some(address!(
                    "3154Cf16ccdb4C6d922629664174b904d80F2C35"
                )),
                optimism_portal_proxy: Some(address!("49048044D57e1C92A77f79988d21Fa8fAF74E97e")),
                system_config_proxy: Some(address!("73a79Fab69143498Ed3712e519A88a918e1f4072")),
                dispute_game_factory_proxy: Some(address!(
                    "43edb88c4b80fdd2adff2412a7bebf9df42cb40e"
                )),
                ..Default::default()
            }),
            gas_paying_token: None,
        };
        assert_eq!(*superchains.op_chains.get(&8453).unwrap(), base_config);
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

        let base_mainnet_config = superchains.rollup_configs.get(&8453).unwrap();
        assert_eq!(
            base_mainnet_config.hardforks.isthmus_time,
            Some(BASE_MAINNET_ISTHMUS_TIMESTAMP)
        );

        let base_sepolia_config = superchains.rollup_configs.get(&84532).unwrap();
        assert_eq!(
            base_sepolia_config.hardforks.isthmus_time,
            Some(BASE_SEPOLIA_ISTHMUS_TIMESTAMP)
        );
    }

    #[test]
    fn test_jovian_timestamps() {
        let superchains = Registry::from_chain_list();
        let op_mainnet_config = superchains.rollup_configs.get(&10).unwrap();
        assert_eq!(op_mainnet_config.hardforks.jovian_time, Some(OP_MAINNET_JOVIAN_TIMESTAMP));

        let op_sepolia_config = superchains.rollup_configs.get(&11155420).unwrap();
        assert_eq!(op_sepolia_config.hardforks.jovian_time, Some(OP_SEPOLIA_JOVIAN_TIMESTAMP));

        let base_mainnet_config = superchains.rollup_configs.get(&8453).unwrap();
        assert_eq!(base_mainnet_config.hardforks.jovian_time, Some(BASE_MAINNET_JOVIAN_TIMESTAMP));

        let base_sepolia_config = superchains.rollup_configs.get(&84532).unwrap();
        assert_eq!(base_sepolia_config.hardforks.jovian_time, Some(BASE_SEPOLIA_JOVIAN_TIMESTAMP));
    }
}

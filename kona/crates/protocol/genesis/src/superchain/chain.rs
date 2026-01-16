//! Contains the `Superchain` type.

use crate::{ChainConfig, SuperchainConfig};
use alloc::{string::String, vec::Vec};

/// A superchain configuration.
#[derive(Debug, Clone, Default, Eq, PartialEq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(deny_unknown_fields))]
pub struct Superchain {
    /// Superchain identifier, without capitalization or display changes.
    pub name: String,
    /// Superchain configuration file contents.
    pub config: SuperchainConfig,
    /// Chain IDs of chains that are part of this superchain.
    pub chains: Vec<ChainConfig>,
}

#[cfg(test)]
#[cfg(feature = "serde")]
mod tests {
    use super::*;
    use crate::{HardForkConfig, SuperchainConfig, SuperchainL1Info};
    use alloc::{string::ToString, vec};

    #[test]
    fn test_deny_unknown_fields_superchain() {
        let raw: &str = r#"
        {
            "name": "Mainnet",
            "config": {
                "name": "Mainnet",
                "l1": {
                    "chainId": 10,
                    "publicRPC": "https://mainnet.rpc",
                    "explorer": "https://mainnet.explorer"
                },
                "hardforks": {
                    "canyon_time": 1699981200,
                    "delta_time": 1703203200,
                    "ecotone_time": 1708534800,
                    "fjord_time": 1716998400,
                    "granite_time": 1723478400,
                    "holocene_time": 1732633200
                }
            },
            "chains": [],
            "other": "test"
        }
        "#;

        let err = serde_json::from_str::<Superchain>(raw).unwrap_err();
        assert_eq!(err.classify(), serde_json::error::Category::Data);
    }

    #[test]
    fn test_superchain_serde() {
        let raw: &str = r#"
        {
            "name": "Mainnet",
            "config": {
                "name": "Mainnet",
                "l1": {
                    "chainId": 10,
                    "publicRPC": "https://mainnet.rpc",
                    "explorer": "https://mainnet.explorer"
                },
                "hardforks": {
                    "canyon_time": 1699981200,
                    "delta_time": 1703203200,
                    "ecotone_time": 1708534800,
                    "fjord_time": 1716998400,
                    "granite_time": 1723478400,
                    "holocene_time": 1732633200
                }
            },
            "chains": []
        }
        "#;

        let superchain = Superchain {
            name: "Mainnet".to_string(),
            config: SuperchainConfig {
                name: "Mainnet".to_string(),
                l1: SuperchainL1Info {
                    chain_id: 10,
                    public_rpc: "https://mainnet.rpc".to_string(),
                    explorer: "https://mainnet.explorer".to_string(),
                },
                hardforks: HardForkConfig {
                    regolith_time: None,
                    canyon_time: Some(1699981200),
                    delta_time: Some(1703203200),
                    ecotone_time: Some(1708534800),
                    fjord_time: Some(1716998400),
                    granite_time: Some(1723478400),
                    holocene_time: Some(1732633200),
                    pectra_blob_schedule_time: None,
                    isthmus_time: None,
                    jovian_time: None,
                    interop_time: None,
                },
                protocol_versions_addr: None,
                superchain_config_addr: None,
                op_contracts_manager_proxy_addr: None,
            },
            chains: vec![],
        };

        let deserialized: Superchain = serde_json::from_str(raw).unwrap();
        assert_eq!(superchain, deserialized);
    }
}

use alloy_chains::NamedChain;
use alloy_genesis::ChainConfig;
use alloy_primitives::{ChainId, U256};
use serde::{Deserialize, Serialize};

/// The chain metadata stored in a superchain toml config file.
/// Referring here as `ChainMetadata` to avoid confusion with `ChainConfig`.
/// Find configs here: `<https://github.com/ethereum-optimism/superchain-registry/tree/main/superchain/configs>`
/// This struct is stripped down to only include the necessary fields. We use JSON instead of
/// TOML to make it easier to work in a no-std environment.
#[derive(Clone, Debug, Deserialize)]
#[serde(rename_all = "snake_case")]
pub(crate) struct ChainMetadata {
    pub chain_id: ChainId,
    pub hardforks: HardforkConfig,
    pub optimism: Option<OptimismConfig>,
}

#[derive(Clone, Debug, Deserialize)]
#[serde(rename_all = "snake_case")]
pub(crate) struct HardforkConfig {
    pub canyon_time: Option<u64>,
    pub delta_time: Option<u64>,
    pub ecotone_time: Option<u64>,
    pub fjord_time: Option<u64>,
    pub granite_time: Option<u64>,
    pub holocene_time: Option<u64>,
    pub isthmus_time: Option<u64>,
    pub jovian_time: Option<u64>,
}

#[derive(Clone, Debug, Deserialize)]
#[serde(rename_all = "snake_case")]
pub(crate) struct OptimismConfig {
    pub eip1559_elasticity: u64,
    pub eip1559_denominator: u64,
    pub eip1559_denominator_canyon: Option<u64>,
}

#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub(crate) struct ChainConfigExtraFields {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub bedrock_block: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub regolith_time: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub canyon_time: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub delta_time: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub ecotone_time: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub fjord_time: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub granite_time: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub holocene_time: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub isthmus_time: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub jovian_time: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub optimism: Option<ChainConfigExtraFieldsOptimism>,
}

// Helper struct to serialize field for extra fields in ChainConfig
#[derive(Clone, Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub(crate) struct ChainConfigExtraFieldsOptimism {
    pub eip1559_elasticity: u64,
    pub eip1559_denominator: u64,
    pub eip1559_denominator_canyon: Option<u64>,
}

impl From<&OptimismConfig> for ChainConfigExtraFieldsOptimism {
    fn from(value: &OptimismConfig) -> Self {
        Self {
            eip1559_elasticity: value.eip1559_elasticity,
            eip1559_denominator: value.eip1559_denominator,
            eip1559_denominator_canyon: value.eip1559_denominator_canyon,
        }
    }
}

/// Returns a [`ChainConfig`] filled from [`ChainMetadata`] with extra fields and handling
/// special case for Optimism chain.
// Mimic the behavior from https://github.com/ethereum-optimism/op-geth/blob/35e2c852/params/superchain.go#L26
pub(crate) fn to_genesis_chain_config(chain_config: &ChainMetadata) -> ChainConfig {
    let mut res = ChainConfig {
        chain_id: chain_config.chain_id,
        homestead_block: Some(0),
        dao_fork_block: None,
        dao_fork_support: false,
        eip150_block: Some(0),
        eip155_block: Some(0),
        eip158_block: Some(0),
        byzantium_block: Some(0),
        constantinople_block: Some(0),
        petersburg_block: Some(0),
        istanbul_block: Some(0),
        muir_glacier_block: Some(0),
        berlin_block: Some(0),
        london_block: Some(0),
        arrow_glacier_block: Some(0),
        gray_glacier_block: Some(0),
        merge_netsplit_block: Some(0),
        shanghai_time: chain_config.hardforks.canyon_time, // Shanghai activates with Canyon
        cancun_time: chain_config.hardforks.ecotone_time,  // Cancun activates with Ecotone
        prague_time: chain_config.hardforks.isthmus_time,  // Prague activates with Isthmus
        osaka_time: None,
        terminal_total_difficulty: Some(U256::ZERO),
        terminal_total_difficulty_passed: true,
        ethash: None,
        clique: None,
        ..Default::default()
    };

    // Special case for Optimism chain
    if chain_config.chain_id == NamedChain::Optimism as ChainId {
        res.berlin_block = Some(3950000);
        res.london_block = Some(105235063);
        res.arrow_glacier_block = Some(105235063);
        res.gray_glacier_block = Some(105235063);
        res.merge_netsplit_block = Some(105235063);
    }

    // Add extra fields for ChainConfig from Genesis
    let extra_fields = ChainConfigExtraFields {
        bedrock_block: if chain_config.chain_id == NamedChain::Optimism as ChainId {
            Some(105235063)
        } else {
            Some(0)
        },
        regolith_time: Some(0),
        canyon_time: chain_config.hardforks.canyon_time,
        delta_time: chain_config.hardforks.delta_time,
        ecotone_time: chain_config.hardforks.ecotone_time,
        fjord_time: chain_config.hardforks.fjord_time,
        granite_time: chain_config.hardforks.granite_time,
        holocene_time: chain_config.hardforks.holocene_time,
        isthmus_time: chain_config.hardforks.isthmus_time,
        jovian_time: chain_config.hardforks.jovian_time,
        optimism: chain_config.optimism.as_ref().map(|o| o.into()),
    };
    res.extra_fields =
        serde_json::to_value(extra_fields).unwrap_or_default().try_into().unwrap_or_default();

    res
}

#[cfg(test)]
mod tests {
    use super::*;

    const BASE_CHAIN_METADATA: &str = r#"
    {
      "chain_id": 8453,
      "hardforks": {
        "canyon_time": 1704992401,
        "delta_time": 1708560000,
        "ecotone_time": 1710374401,
        "fjord_time": 1720627201,
        "granite_time": 1726070401,
        "holocene_time": 1736445601,
        "isthmus_time": 1746806401
      },
      "optimism": {
        "eip1559_elasticity": 6,
        "eip1559_denominator": 50,
        "eip1559_denominator_canyon": 250
      }
    }
    "#;

    #[test]
    fn test_deserialize_chain_config() {
        let config: ChainMetadata = serde_json::from_str(BASE_CHAIN_METADATA).unwrap();
        assert_eq!(config.chain_id, 8453);
        // hardforks
        assert_eq!(config.hardforks.canyon_time, Some(1704992401));
        assert_eq!(config.hardforks.delta_time, Some(1708560000));
        assert_eq!(config.hardforks.ecotone_time, Some(1710374401));
        assert_eq!(config.hardforks.fjord_time, Some(1720627201));
        assert_eq!(config.hardforks.granite_time, Some(1726070401));
        assert_eq!(config.hardforks.holocene_time, Some(1736445601));
        assert_eq!(config.hardforks.isthmus_time, Some(1746806401));
        // optimism
        assert_eq!(config.optimism.as_ref().unwrap().eip1559_elasticity, 6);
        assert_eq!(config.optimism.as_ref().unwrap().eip1559_denominator, 50);
        assert_eq!(config.optimism.as_ref().unwrap().eip1559_denominator_canyon, Some(250));
    }

    #[test]
    fn test_chain_config_extra_fields() {
        let extra_fields = ChainConfigExtraFields {
            bedrock_block: Some(105235063),
            regolith_time: Some(0),
            canyon_time: Some(1704992401),
            delta_time: Some(1708560000),
            ecotone_time: Some(1710374401),
            fjord_time: Some(1720627201),
            granite_time: Some(1726070401),
            holocene_time: Some(1736445601),
            isthmus_time: Some(1746806401),
            jovian_time: None,
            optimism: Option::from(ChainConfigExtraFieldsOptimism {
                eip1559_elasticity: 6,
                eip1559_denominator: 50,
                eip1559_denominator_canyon: Some(250),
            }),
        };
        let value = serde_json::to_value(extra_fields).unwrap();
        assert_eq!(value.get("bedrockBlock").unwrap(), 105235063);
        assert_eq!(value.get("regolithTime").unwrap(), 0);
        assert_eq!(value.get("canyonTime").unwrap(), 1704992401);
        assert_eq!(value.get("deltaTime").unwrap(), 1708560000);
        assert_eq!(value.get("ecotoneTime").unwrap(), 1710374401);
        assert_eq!(value.get("fjordTime").unwrap(), 1720627201);
        assert_eq!(value.get("graniteTime").unwrap(), 1726070401);
        assert_eq!(value.get("holoceneTime").unwrap(), 1736445601);
        assert_eq!(value.get("isthmusTime").unwrap(), 1746806401);
        assert_eq!(value.get("jovianTime"), None);
        let optimism = value.get("optimism").unwrap();
        assert_eq!(optimism.get("eip1559Elasticity").unwrap(), 6);
        assert_eq!(optimism.get("eip1559Denominator").unwrap(), 50);
        assert_eq!(optimism.get("eip1559DenominatorCanyon").unwrap(), 250);
    }

    #[test]
    fn test_convert_to_genesis_chain_config() {
        let config: ChainMetadata = serde_json::from_str(BASE_CHAIN_METADATA).unwrap();
        let chain_config = to_genesis_chain_config(&config);
        assert_eq!(chain_config.chain_id, 8453);
        assert_eq!(chain_config.homestead_block, Some(0));
        assert_eq!(chain_config.dao_fork_block, None);
        assert!(!chain_config.dao_fork_support);
        assert_eq!(chain_config.eip150_block, Some(0));
        assert_eq!(chain_config.eip155_block, Some(0));
        assert_eq!(chain_config.eip158_block, Some(0));
        assert_eq!(chain_config.byzantium_block, Some(0));
        assert_eq!(chain_config.constantinople_block, Some(0));
        assert_eq!(chain_config.petersburg_block, Some(0));
        assert_eq!(chain_config.istanbul_block, Some(0));
        assert_eq!(chain_config.muir_glacier_block, Some(0));
        assert_eq!(chain_config.berlin_block, Some(0));
        assert_eq!(chain_config.london_block, Some(0));
        assert_eq!(chain_config.arrow_glacier_block, Some(0));
        assert_eq!(chain_config.gray_glacier_block, Some(0));
        assert_eq!(chain_config.merge_netsplit_block, Some(0));
        assert_eq!(chain_config.shanghai_time, Some(1704992401));
        assert_eq!(chain_config.cancun_time, Some(1710374401));
        assert_eq!(chain_config.prague_time, Some(1746806401));
        assert_eq!(chain_config.osaka_time, None);
        assert_eq!(chain_config.terminal_total_difficulty, Some(U256::ZERO));
        assert!(chain_config.terminal_total_difficulty_passed);
        assert_eq!(chain_config.ethash, None);
        assert_eq!(chain_config.clique, None);
        assert_eq!(chain_config.extra_fields.get("bedrockBlock").unwrap(), 0);
        assert_eq!(chain_config.extra_fields.get("regolithTime").unwrap(), 0);
        assert_eq!(chain_config.extra_fields.get("canyonTime").unwrap(), 1704992401);
        assert_eq!(chain_config.extra_fields.get("deltaTime").unwrap(), 1708560000);
        assert_eq!(chain_config.extra_fields.get("ecotoneTime").unwrap(), 1710374401);
        assert_eq!(chain_config.extra_fields.get("fjordTime").unwrap(), 1720627201);
        assert_eq!(chain_config.extra_fields.get("graniteTime").unwrap(), 1726070401);
        assert_eq!(chain_config.extra_fields.get("holoceneTime").unwrap(), 1736445601);
        assert_eq!(chain_config.extra_fields.get("isthmusTime").unwrap(), 1746806401);
        assert_eq!(chain_config.extra_fields.get("jovianTime"), None);
        let optimism = chain_config.extra_fields.get("optimism").unwrap();
        assert_eq!(optimism.get("eip1559Elasticity").unwrap(), 6);
        assert_eq!(optimism.get("eip1559Denominator").unwrap(), 50);
        assert_eq!(optimism.get("eip1559DenominatorCanyon").unwrap(), 250);
    }

    #[test]
    fn test_convert_to_genesis_chain_config_op() {
        const OP_CHAIN_METADATA: &str = r#"
        {
          "chain_id": 10,
          "hardforks": {
            "canyon_time": 1704992401,
            "delta_time": 1708560000,
            "ecotone_time": 1710374401,
            "fjord_time": 1720627201,
            "granite_time": 1726070401,
            "holocene_time": 1736445601,
            "isthmus_time": 1746806401
          },
          "optimism": {
            "eip1559_elasticity": 6,
            "eip1559_denominator": 50,
            "eip1559_denominator_canyon": 250
          }
        }
        "#;
        let config: ChainMetadata = serde_json::from_str(OP_CHAIN_METADATA).unwrap();
        assert_eq!(config.hardforks.canyon_time, Some(1704992401));
        let chain_config = to_genesis_chain_config(&config);
        assert_eq!(chain_config.chain_id, 10);
        assert_eq!(chain_config.shanghai_time, Some(1704992401));
        assert_eq!(chain_config.cancun_time, Some(1710374401));
        assert_eq!(chain_config.prague_time, Some(1746806401));
        assert_eq!(chain_config.berlin_block, Some(3950000));
        assert_eq!(chain_config.london_block, Some(105235063));
        assert_eq!(chain_config.arrow_glacier_block, Some(105235063));
        assert_eq!(chain_config.gray_glacier_block, Some(105235063));
        assert_eq!(chain_config.merge_netsplit_block, Some(105235063));
        assert_eq!(chain_config.extra_fields.get("bedrockBlock").unwrap(), 105235063);
        assert_eq!(chain_config.extra_fields.get("regolithTime").unwrap(), 0);
        assert_eq!(chain_config.extra_fields.get("canyonTime").unwrap(), 1704992401);
        assert_eq!(chain_config.extra_fields.get("deltaTime").unwrap(), 1708560000);
        assert_eq!(chain_config.extra_fields.get("ecotoneTime").unwrap(), 1710374401);
        assert_eq!(chain_config.extra_fields.get("fjordTime").unwrap(), 1720627201);
        assert_eq!(chain_config.extra_fields.get("graniteTime").unwrap(), 1726070401);
        assert_eq!(chain_config.extra_fields.get("holoceneTime").unwrap(), 1736445601);
        assert_eq!(chain_config.extra_fields.get("isthmusTime").unwrap(), 1746806401);
        assert_eq!(chain_config.extra_fields.get("jovianTime"), None);

        let optimism = chain_config.extra_fields.get("optimism").unwrap();
        assert_eq!(optimism.get("eip1559Elasticity").unwrap(), 6);
        assert_eq!(optimism.get("eip1559Denominator").unwrap(), 50);
        assert_eq!(optimism.get("eip1559DenominatorCanyon").unwrap(), 250);
    }
}

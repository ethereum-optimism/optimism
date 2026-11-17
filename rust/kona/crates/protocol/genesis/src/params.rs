//! Module containing fee parameters.

use alloy_eips::eip1559::BaseFeeParams;

/// Base fee max change denominator for Optimism Mainnet as defined in the Optimism
/// [transaction costs](https://docs.optimism.io/app-developers/transactions/fees) doc.
pub const OP_MAINNET_EIP1559_DEFAULT_BASE_FEE_MAX_CHANGE_DENOMINATOR: u64 = 50;

/// Base fee max change denominator for Optimism Mainnet as defined in the Optimism Canyon
/// hardfork.
pub const OP_MAINNET_EIP1559_BASE_FEE_MAX_CHANGE_DENOMINATOR_CANYON: u64 = 250;

/// Base fee max change denominator for Optimism Mainnet as defined in the Optimism
/// [transaction costs](https://docs.optimism.io/app-developers/transactions/fees) doc.
pub const OP_MAINNET_EIP1559_DEFAULT_ELASTICITY_MULTIPLIER: u64 = 6;

/// Get the base fee parameters for Optimism Mainnet.
pub const OP_MAINNET_BASE_FEE_PARAMS: BaseFeeParams = BaseFeeParams {
    max_change_denominator: OP_MAINNET_EIP1559_DEFAULT_BASE_FEE_MAX_CHANGE_DENOMINATOR as u128,
    elasticity_multiplier: OP_MAINNET_EIP1559_DEFAULT_ELASTICITY_MULTIPLIER as u128,
};

/// Get the base fee parameters for Optimism Mainnet.
pub const OP_MAINNET_BASE_FEE_PARAMS_CANYON: BaseFeeParams = BaseFeeParams {
    max_change_denominator: OP_MAINNET_EIP1559_BASE_FEE_MAX_CHANGE_DENOMINATOR_CANYON as u128,
    elasticity_multiplier: OP_MAINNET_EIP1559_DEFAULT_ELASTICITY_MULTIPLIER as u128,
};

/// Get the base fee parameters for Optimism Mainnet.
pub const OP_MAINNET_BASE_FEE_CONFIG: BaseFeeConfig = BaseFeeConfig {
    eip1559_elasticity: OP_MAINNET_EIP1559_DEFAULT_ELASTICITY_MULTIPLIER,
    eip1559_denominator: OP_MAINNET_EIP1559_DEFAULT_BASE_FEE_MAX_CHANGE_DENOMINATOR,
    eip1559_denominator_canyon: OP_MAINNET_EIP1559_BASE_FEE_MAX_CHANGE_DENOMINATOR_CANYON,
};

/// Optimism Base Fee Config.
#[derive(Debug, Copy, Clone, Eq, PartialEq)]
#[cfg_attr(feature = "arbitrary", derive(arbitrary::Arbitrary))]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct BaseFeeConfig {
    /// EIP 1559 Elasticity Parameter
    #[cfg_attr(
        feature = "serde",
        serde(rename = "eip1559Elasticity", alias = "eip1559_elasticity")
    )]
    pub eip1559_elasticity: u64,
    /// EIP 1559 Denominator
    #[cfg_attr(
        feature = "serde",
        serde(rename = "eip1559Denominator", alias = "eip1559_denominator")
    )]
    pub eip1559_denominator: u64,
    /// EIP 1559 Denominator for the Canyon hardfork
    #[cfg_attr(
        feature = "serde",
        serde(rename = "eip1559DenominatorCanyon", alias = "eip1559_denominator_canyon")
    )]
    pub eip1559_denominator_canyon: u64,
}

impl BaseFeeConfig {
    /// Get the base fee parameters for Optimism Mainnet
    pub const fn optimism() -> Self {
        Self {
            eip1559_elasticity: OP_MAINNET_EIP1559_DEFAULT_ELASTICITY_MULTIPLIER,
            eip1559_denominator: OP_MAINNET_EIP1559_DEFAULT_BASE_FEE_MAX_CHANGE_DENOMINATOR,
            eip1559_denominator_canyon: OP_MAINNET_EIP1559_BASE_FEE_MAX_CHANGE_DENOMINATOR_CANYON,
        }
    }

    /// Returns the [`BaseFeeParams`] before Canyon hardfork.
    pub const fn pre_canyon_params(&self) -> BaseFeeParams {
        BaseFeeParams {
            max_change_denominator: self.eip1559_denominator as u128,
            elasticity_multiplier: self.eip1559_elasticity as u128,
        }
    }

    /// Returns the [`BaseFeeParams`] since Canyon hardfork.
    pub const fn post_canyon_params(&self) -> BaseFeeParams {
        BaseFeeParams {
            max_change_denominator: self.eip1559_denominator_canyon as u128,
            elasticity_multiplier: self.eip1559_elasticity as u128,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    #[cfg(feature = "serde")]
    fn test_base_fee_config_ser() {
        let config = OP_MAINNET_BASE_FEE_CONFIG;
        let raw_str = serde_json::to_string(&config).unwrap();
        assert_eq!(
            raw_str,
            r#"{"eip1559Elasticity":6,"eip1559Denominator":50,"eip1559DenominatorCanyon":250}"#
        );
    }

    #[test]
    #[cfg(feature = "serde")]
    fn test_base_fee_config_deser() {
        let raw_str: &'static str =
            r#"{"eip1559Elasticity":6,"eip1559Denominator":50,"eip1559DenominatorCanyon":250}"#;
        let config: BaseFeeConfig = serde_json::from_str(raw_str).unwrap();
        assert_eq!(config, OP_MAINNET_BASE_FEE_CONFIG);
    }
}

//! Contains the superchain L1 information.

use alloc::string::String;

/// Superchain L1 anchor information
#[derive(Debug, Clone, Default, Hash, Eq, PartialEq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(deny_unknown_fields))]
pub struct SuperchainL1Info {
    /// L1 chain ID
    #[cfg_attr(feature = "serde", serde(alias = "chainId"))]
    pub chain_id: u64,
    /// L1 chain public RPC endpoint
    #[cfg_attr(feature = "serde", serde(alias = "publicRPC"))]
    pub public_rpc: String,
    /// L1 chain explorer RPC endpoint
    pub explorer: String,
}

#[cfg(test)]
#[cfg(feature = "serde")]
mod tests {
    use super::*;
    use crate::alloc::string::ToString;

    #[test]
    fn test_deny_unknown_fields_sc_l1_info() {
        let raw: &str = r#"
        {
            "chainId": 10,
            "publicRPC": "https://mainnet.rpc",
            "explorer": "https://mainnet.explorer",
            "unknown_field": "unknown"
        }
        "#;

        let err = serde_json::from_str::<SuperchainL1Info>(raw).unwrap_err();
        assert_eq!(err.classify(), serde_json::error::Category::Data);
    }

    #[test]
    fn test_sc_l1_info_serde() {
        let raw: &str = r#"
        {
            "chainId": 10,
            "publicRPC": "https://mainnet.rpc",
            "explorer": "https://mainnet.explorer"
        }
        "#;

        let sc_l1_info = SuperchainL1Info {
            chain_id: 10,
            public_rpc: "https://mainnet.rpc".to_string(),
            explorer: "https://mainnet.explorer".to_string(),
        };

        let sc_l1_info_serde = serde_json::from_str::<SuperchainL1Info>(raw).unwrap();
        assert_eq!(sc_l1_info, sc_l1_info_serde);
    }
}

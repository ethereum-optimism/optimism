//! Contains the AltDA config type.

use alloc::string::String;
use alloy_primitives::Address;

/// AltDA configuration.
///
/// See: <https://github.com/ethereum-optimism/superchain-registry/blob/8ff62ada16e14dd59d0fb94ffb47761c7fa96e01/ops/internal/config/chain.go#L133-L138>
#[derive(Debug, Clone, Default, Hash, Eq, PartialEq)]
#[cfg_attr(feature = "arbitrary", derive(arbitrary::Arbitrary))]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(deny_unknown_fields))]
pub struct AltDAConfig {
    /// AltDA challenge address
    #[cfg_attr(feature = "serde", serde(alias = "da_challenge_contract_address"))]
    pub da_challenge_address: Option<Address>,
    /// AltDA challenge window time (in seconds)
    pub da_challenge_window: Option<u64>,
    /// AltDA resolution window time (in seconds)
    pub da_resolve_window: Option<u64>,
    /// AltDA commitment type
    pub da_commitment_type: Option<String>,
}

#[cfg(test)]
#[cfg(feature = "serde")]
mod tests {
    use super::*;
    use alloc::string::ToString;
    use alloy_primitives::address;

    #[test]
    fn test_altda_deserialize_json() {
        let raw: &str = r#"
        {
            "da_challenge_address": "0x12c6a7db25b20347ca6f5d47e56d5e8219871c6d",
            "da_challenge_window": 1,
            "da_resolve_window": 1,
            "da_commitment_type": "KeccakCommitment"
        }
        "#;

        let altda = AltDAConfig {
            da_challenge_address: Some(address!("12c6a7db25b20347ca6f5d47e56d5e8219871c6d")),
            da_challenge_window: Some(1),
            da_resolve_window: Some(1),
            da_commitment_type: Some("KeccakCommitment".to_string()),
        };

        let deserialized: AltDAConfig = serde_json::from_str(raw).unwrap();
        assert_eq!(altda, deserialized);
    }

    #[test]
    fn test_altda_unknown_fields_json() {
        let raw: &str = r#"
        {
            "da_challenge_address": "0x12c6a7db25b20347ca6f5d47e56d5e8219871c6d",
            "da_challenge_window": 1,
            "da_resolve_window": 1,
            "da_commitment_type": "KeccakCommitment",
            "unknown_field": "unknown"
        }
        "#;

        let err = serde_json::from_str::<AltDAConfig>(raw).unwrap_err();
        assert_eq!(err.classify(), serde_json::error::Category::Data);
    }
}

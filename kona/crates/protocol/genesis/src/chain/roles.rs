//! OP Stack component roles.

use alloy_primitives::Address;

/// Roles in the OP Stack.
///
/// See: <https://github.com/ethereum-optimism/superchain-registry/blob/8ff62ada16e14dd59d0fb94ffb47761c7fa96e01/ops/internal/config/chain.go#L146-L154>
#[derive(Debug, Clone, Hash, PartialEq, Eq, Default)]
#[cfg_attr(feature = "arbitrary", derive(arbitrary::Arbitrary))]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "PascalCase"))]
#[cfg_attr(feature = "serde", serde(deny_unknown_fields))]
pub struct Roles {
    /// The system config owner
    pub system_config_owner: Option<Address>,
    /// The owner of the Proxy Admin
    pub proxy_admin_owner: Option<Address>,
    /// The guardian address
    pub guardian: Option<Address>,
    /// The challenger's address
    pub challenger: Option<Address>,
    /// The proposer's address
    pub proposer: Option<Address>,
    /// Unsafe block signer.
    pub unsafe_block_signer: Option<Address>,
    /// The batch submitter's address
    pub batch_submitter: Option<Address>,
}

#[cfg(test)]
#[cfg(feature = "serde")]
mod tests {
    use super::*;

    #[test]
    fn test_roles_serde() {
        let json_roles: &str = r#"{
            "SystemConfigOwner": "0x8c20c40180751d93E939DDDee3517AE0d1EBeAd2",
            "ProxyAdminOwner": "0x4377BB0F0103992b31eC12b4d796a8687B8dC8E9",
            "Guardian": "0x8c20c40180751d93E939DDDee3517AE0d1EBeAd2",
            "Challenger": "0x8c20c40180751d93E939DDDee3517AE0d1EBeAd2",
            "Proposer": "0x95014c45078354Ff839f14192228108Eac82E00A",
            "UnsafeBlockSigner": "0xa95B83e39AA78B00F12fe431865B563793D97AF5",
            "BatchSubmitter": "0x19CC7073150D9f5888f09E0e9016d2a39667df14"
        }"#;

        let expected: Roles = Roles {
            system_config_owner: Some(
                "0x8c20c40180751d93E939DDDee3517AE0d1EBeAd2".parse().unwrap(),
            ),
            proxy_admin_owner: Some("0x4377BB0F0103992b31eC12b4d796a8687B8dC8E9".parse().unwrap()),
            guardian: Some("0x8c20c40180751d93E939DDDee3517AE0d1EBeAd2".parse().unwrap()),
            challenger: Some("0x8c20c40180751d93E939DDDee3517AE0d1EBeAd2".parse().unwrap()),
            proposer: Some("0x95014c45078354Ff839f14192228108Eac82E00A".parse().unwrap()),
            unsafe_block_signer: Some(
                "0xa95B83e39AA78B00F12fe431865B563793D97AF5".parse().unwrap(),
            ),
            batch_submitter: Some("0x19CC7073150D9f5888f09E0e9016d2a39667df14".parse().unwrap()),
        };

        let deserialized: Roles = serde_json::from_str(json_roles).unwrap();

        assert_eq!(expected, deserialized);
    }

    #[test]
    fn test_roles_unknown_field_json() {
        let json_roles: &str = r#"{
            "SystemConfigOwner": "0x8c20c40180751d93E939DDDee3517AE0d1EBeAd2",
            "ProxyAdminOwner": "0x4377BB0F0103992b31eC12b4d796a8687B8dC8E9",
            "Guardian": "0x8c20c40180751d93E939DDDee3517AE0d1EBeAd2",
            "Challenger": "0x8c20c40180751d93E939DDDee3517AE0d1EBeAd2",
            "Proposer": "0x95014c45078354Ff839f14192228108Eac82E00A",
            "UnsafeBlockSigner": "0xa95B83e39AA78B00F12fe431865B563793D97AF5",
            "BatchSubmitter": "0x19CC7073150D9f5888f09E0e9016d2a39667df14",
            "UnknownField": "unknown"
        }"#;

        let err = serde_json::from_str::<Roles>(json_roles).unwrap_err();
        assert_eq!(err.classify(), serde_json::error::Category::Data);
    }
}

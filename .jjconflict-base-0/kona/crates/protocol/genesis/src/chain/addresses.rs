//! Address Types

use alloy_primitives::Address;

/// The set of network-specific contracts for a given chain.
///
/// See: <https://github.com/ethereum-optimism/superchain-registry/blob/8ff62ada16e14dd59d0fb94ffb47761c7fa96e01/ops/internal/config/chain.go#L156-L175>
#[derive(Debug, Clone, Hash, PartialEq, Eq, Default)]
#[cfg_attr(feature = "arbitrary", derive(arbitrary::Arbitrary))]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "PascalCase"))]
pub struct AddressList {
    /// The address manager
    pub address_manager: Option<Address>,
    /// L1 Cross Domain Messenger proxy address
    pub l1_cross_domain_messenger_proxy: Option<Address>,
    /// L1 ERC721 Bridge proxy address
    #[cfg_attr(feature = "serde", serde(alias = "L1ERC721BridgeProxy"))]
    pub l1_erc721_bridge_proxy: Option<Address>,
    /// L1 Standard Bridge proxy address
    pub l1_standard_bridge_proxy: Option<Address>,
    /// L2 Output Oracle Proxy address
    pub l2_output_oracle_proxy: Option<Address>,
    /// Optimism Mintable ERC20 Factory Proxy address
    #[cfg_attr(feature = "serde", serde(alias = "OptimismMintableERC20FactoryProxy"))]
    pub optimism_mintable_erc20_factory_proxy: Option<Address>,
    /// Optimism Portal Proxy address
    pub optimism_portal_proxy: Option<Address>,
    /// System Config Proxy address
    pub system_config_proxy: Option<Address>,
    /// Proxy Admin address
    pub proxy_admin: Option<Address>,
    /// The superchain config address
    pub superchain_config: Option<Address>,

    // Fault Proof Contract Addresses
    /// Anchor State Registry Proxy address
    pub anchor_state_registry_proxy: Option<Address>,
    /// Delayed WETH Proxy address
    #[cfg_attr(feature = "serde", serde(alias = "DelayedWETHProxy"))]
    pub delayed_weth_proxy: Option<Address>,
    /// Dispute Game Factory Proxy address
    pub dispute_game_factory_proxy: Option<Address>,
    /// Fault Dispute Game Proxy address
    pub fault_dispute_game: Option<Address>,
    /// MIPS Proxy address
    #[cfg_attr(feature = "serde", serde(alias = "MIPS"))]
    pub mips: Option<Address>,
    /// Permissioned Dispute Game Proxy address
    pub permissioned_dispute_game: Option<Address>,
    /// Preimage Oracle Proxy address
    pub preimage_oracle: Option<Address>,
    /// The data availability challenge contract address
    #[cfg_attr(feature = "serde", serde(alias = "DAChallengeAddress"))]
    pub data_availability_challenge: Option<Address>,
}

impl AddressList {
    /// Sets zeroed addresses to [`Option::None`].
    pub fn zero_proof_addresses(&mut self) {
        if self.anchor_state_registry_proxy == Some(Address::ZERO) {
            self.anchor_state_registry_proxy = None;
        }
        if self.delayed_weth_proxy == Some(Address::ZERO) {
            self.delayed_weth_proxy = None;
        }
        if self.dispute_game_factory_proxy == Some(Address::ZERO) {
            self.dispute_game_factory_proxy = None;
        }
        if self.fault_dispute_game == Some(Address::ZERO) {
            self.fault_dispute_game = None;
        }
        if self.mips == Some(Address::ZERO) {
            self.mips = None;
        }
        if self.permissioned_dispute_game == Some(Address::ZERO) {
            self.permissioned_dispute_game = None;
        }
        if self.preimage_oracle == Some(Address::ZERO) {
            self.preimage_oracle = None;
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::address;

    #[test]
    fn zero_proof_addresses() {
        let mut addresses = AddressList {
            anchor_state_registry_proxy: Some(Address::ZERO),
            delayed_weth_proxy: Some(Address::ZERO),
            dispute_game_factory_proxy: Some(Address::ZERO),
            fault_dispute_game: Some(Address::ZERO),
            mips: Some(Address::ZERO),
            permissioned_dispute_game: Some(Address::ZERO),
            preimage_oracle: Some(Address::ZERO),
            ..Default::default()
        };

        addresses.zero_proof_addresses();

        assert_eq!(addresses.anchor_state_registry_proxy, None);
        assert_eq!(addresses.delayed_weth_proxy, None);
        assert_eq!(addresses.dispute_game_factory_proxy, None);
        assert_eq!(addresses.fault_dispute_game, None);
        assert_eq!(addresses.mips, None);
        assert_eq!(addresses.permissioned_dispute_game, None);
        assert_eq!(addresses.preimage_oracle, None);
    }

    #[test]
    fn test_addresses_deserialize() {
        let raw: &str = r#"
        {
            "AddressManager": "0x8efb6b5c4767b09dc9aa6af4eaa89f749522bae2",
            "L1CrossDomainMessengerProxy": "0x866e82a600a1414e583f7f13623f1ac5d58b0afa",
            "L1Erc721BridgeProxy": "0x608d94945a64503e642e6370ec598e519a2c1e53",
            "L1StandardBridgeProxy": "0x3154cf16ccdb4c6d922629664174b904d80f2c35",
            "L2OutputOracleProxy": "0x56315b90c40730925ec5485cf004d835058518a0",
            "OptimismMintableErc20FactoryProxy": "0x05cc379ebd9b30bba19c6fa282ab29218ec61d84",
            "OptimismPortalProxy": "0x49048044d57e1c92a77f79988d21fa8faf74e97e",
            "SystemConfigProxy": "0x73a79fab69143498ed3712e519a88a918e1f4072",
            "ProxyAdmin": "0x0475cbcaebd9ce8afa5025828d5b98dfb67e059e",
            "AnchorStateRegistryProxy": "0xdb9091e48b1c42992a1213e6916184f9ebdbfedf",
            "DelayedWethProxy": "0xa2f2ac6f5af72e494a227d79db20473cf7a1ffe8",
            "DisputeGameFactoryProxy": "0x43edb88c4b80fdd2adff2412a7bebf9df42cb40e",
            "FaultDisputeGame": "0xcd3c0194db74c23807d4b90a5181e1b28cf7007c",
            "Mips": "0x16e83ce5ce29bf90ad9da06d2fe6a15d5f344ce4",
            "PermissionedDisputeGame": "0x19009debf8954b610f207d5925eede827805986e",
            "PreimageOracle": "0x9c065e11870b891d214bc2da7ef1f9ddfa1be277"
        }
        "#;

        let addresses = AddressList {
            address_manager: Some(address!("8EfB6B5c4767B09Dc9AA6Af4eAA89F749522BaE2")),
            l1_cross_domain_messenger_proxy: Some(address!(
                "866E82a600A1414e583f7F13623F1aC5d58b0Afa"
            )),
            l1_erc721_bridge_proxy: Some(address!("608d94945A64503E642E6370Ec598e519a2C1E53")),
            l1_standard_bridge_proxy: Some(address!("3154Cf16ccdb4C6d922629664174b904d80F2C35")),
            l2_output_oracle_proxy: Some(address!("56315b90c40730925ec5485cf004d835058518A0")),
            optimism_mintable_erc20_factory_proxy: Some(address!(
                "05cc379EBD9B30BbA19C6fA282AB29218EC61D84"
            )),
            optimism_portal_proxy: Some(address!("49048044D57e1C92A77f79988d21Fa8fAF74E97e")),
            system_config_proxy: Some(address!("73a79Fab69143498Ed3712e519A88a918e1f4072")),
            proxy_admin: Some(address!("0475cBCAebd9CE8AfA5025828d5b98DFb67E059E")),
            superchain_config: None,
            anchor_state_registry_proxy: Some(address!("db9091e48b1c42992a1213e6916184f9ebdbfedf")),
            delayed_weth_proxy: Some(address!("a2f2ac6f5af72e494a227d79db20473cf7a1ffe8")),
            dispute_game_factory_proxy: Some(address!("43edb88c4b80fdd2adff2412a7bebf9df42cb40e")),
            fault_dispute_game: Some(address!("cd3c0194db74c23807d4b90a5181e1b28cf7007c")),
            mips: Some(address!("16e83ce5ce29bf90ad9da06d2fe6a15d5f344ce4")),
            permissioned_dispute_game: Some(address!("19009debf8954b610f207d5925eede827805986e")),
            preimage_oracle: Some(address!("9c065e11870b891d214bc2da7ef1f9ddfa1be277")),
            data_availability_challenge: None,
        };

        let deserialized: AddressList = serde_json::from_str(raw).unwrap();
        assert_eq!(addresses, deserialized);
    }

    #[test]
    fn test_addresses_unknown_field_json() {
        let raw: &str = r#"
        {
            "AddressManager": "0x8efb6b5c4767b09dc9aa6af4eaa89f749522bae2",
            "L1CrossDomainMessengerProxy": "0x866e82a600a1414e583f7f13623f1ac5d58b0afa",
            "L1Erc721BridgeProxy": "0x608d94945a64503e642e6370ec598e519a2c1e53",
            "L1StandardBridgeProxy": "0x3154cf16ccdb4c6d922629664174b904d80f2c35",
            "L2OutputOracleProxy": "0x56315b90c40730925ec5485cf004d835058518a0",
            "OptimismMintableErc20FactoryProxy": "0x05cc379ebd9b30bba19c6fa282ab29218ec61d84",
            "OptimismPortalProxy": "0x49048044d57e1c92a77f79988d21fa8faf74e97e",
            "SystemConfigProxy": "0x73a79fab69143498ed3712e519a88a918e1f4072",
            "ProxyAdmin": "0x0475cbcaebd9ce8afa5025828d5b98dfb67e059e",
            "AnchorStateRegistryProxy": "0xdb9091e48b1c42992a1213e6916184f9ebdbfedf",
            "DelayedWethProxy": "0xa2f2ac6f5af72e494a227d79db20473cf7a1ffe8",
            "DisputeGameFactoryProxy": "0x43edb88c4b80fdd2adff2412a7bebf9df42cb40e",
            "FaultDisputeGame": "0xcd3c0194db74c23807d4b90a5181e1b28cf7007c",
            "Mips": "0x16e83ce5ce29bf90ad9da06d2fe6a15d5f344ce4",
            "PermissionedDisputeGame": "0x19009debf8954b610f207d5925eede827805986e",
            "PreimageOracle": "0x9c065e11870b891d214bc2da7ef1f9ddfa1be277",
            "unknown_field": "unknown"
        }
        "#;

        let addresses = AddressList {
            address_manager: Some(address!("8EfB6B5c4767B09Dc9AA6Af4eAA89F749522BaE2")),
            l1_cross_domain_messenger_proxy: Some(address!(
                "866E82a600A1414e583f7F13623F1aC5d58b0Afa"
            )),
            l1_erc721_bridge_proxy: Some(address!("608d94945A64503E642E6370Ec598e519a2C1E53")),
            l1_standard_bridge_proxy: Some(address!("3154Cf16ccdb4C6d922629664174b904d80F2C35")),
            l2_output_oracle_proxy: Some(address!("56315b90c40730925ec5485cf004d835058518A0")),
            optimism_mintable_erc20_factory_proxy: Some(address!(
                "05cc379EBD9B30BbA19C6fA282AB29218EC61D84"
            )),
            optimism_portal_proxy: Some(address!("49048044D57e1C92A77f79988d21Fa8fAF74E97e")),
            system_config_proxy: Some(address!("73a79Fab69143498Ed3712e519A88a918e1f4072")),
            proxy_admin: Some(address!("0475cBCAebd9CE8AfA5025828d5b98DFb67E059E")),
            superchain_config: None,
            anchor_state_registry_proxy: Some(address!("db9091e48b1c42992a1213e6916184f9ebdbfedf")),
            delayed_weth_proxy: Some(address!("a2f2ac6f5af72e494a227d79db20473cf7a1ffe8")),
            dispute_game_factory_proxy: Some(address!("43edb88c4b80fdd2adff2412a7bebf9df42cb40e")),
            fault_dispute_game: Some(address!("cd3c0194db74c23807d4b90a5181e1b28cf7007c")),
            mips: Some(address!("16e83ce5ce29bf90ad9da06d2fe6a15d5f344ce4")),
            permissioned_dispute_game: Some(address!("19009debf8954b610f207d5925eede827805986e")),
            preimage_oracle: Some(address!("9c065e11870b891d214bc2da7ef1f9ddfa1be277")),
            data_availability_challenge: None,
        };

        let deserialized: AddressList = serde_json::from_str(raw).unwrap();
        assert_eq!(addresses, deserialized);
    }
}

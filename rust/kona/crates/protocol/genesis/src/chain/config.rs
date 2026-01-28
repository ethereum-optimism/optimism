//! Contains the chain config type.

use alloc::string::String;
use alloy_chains::Chain;
use alloy_eips::eip1559::BaseFeeParams;
use alloy_primitives::Address;

use crate::{
    AddressList, AltDAConfig, BaseFeeConfig, ChainGenesis, GRANITE_CHANNEL_TIMEOUT, HardForkConfig,
    Roles, RollupConfig, SuperchainLevel, base_fee_params, base_fee_params_canyon,
    params::base_fee_config, rollup::DEFAULT_INTEROP_MESSAGE_EXPIRY_WINDOW,
};

/// L1 chain configuration from the `alloy-genesis` crate.
pub type L1ChainConfig = alloy_genesis::ChainConfig;

/// Defines core blockchain settings per block.
///
/// Tailors unique settings for each network based on
/// its genesis block and superchain configuration.
///
/// This struct bridges the interface between the [`ChainConfig`][ccr]
/// defined in the [`superchain-registry`][scr] and the [`ChainConfig`][ccg]
/// defined in [`op-geth`][opg].
///
/// [opg]: https://github.com/ethereum-optimism/op-geth
/// [scr]: https://github.com/ethereum-optimism/superchain-registry
/// [ccg]: https://github.com/ethereum-optimism/op-geth/blob/optimism/params/config.go#L342
/// [ccr]: https://github.com/ethereum-optimism/superchain-registry/blob/main/ops/internal/config/superchain.go#L70
#[derive(Debug, Clone, Default, Eq, PartialEq)]
#[cfg_attr(feature = "arbitrary", derive(arbitrary::Arbitrary))]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub struct ChainConfig {
    /// Chain name (e.g. "Base")
    #[cfg_attr(feature = "serde", serde(rename = "Name", alias = "name"))]
    pub name: String,
    /// L1 chain ID
    #[cfg_attr(feature = "serde", serde(skip))]
    pub l1_chain_id: u64,
    /// Chain public RPC endpoint
    #[cfg_attr(feature = "serde", serde(rename = "PublicRPC", alias = "public_rpc"))]
    pub public_rpc: String,
    /// Chain sequencer RPC endpoint
    #[cfg_attr(feature = "serde", serde(rename = "SequencerRPC", alias = "sequencer_rpc"))]
    pub sequencer_rpc: String,
    /// Chain explorer HTTP endpoint
    #[cfg_attr(feature = "serde", serde(rename = "Explorer", alias = "explorer"))]
    pub explorer: String,
    /// Level of integration with the superchain.
    #[cfg_attr(feature = "serde", serde(rename = "SuperchainLevel", alias = "superchain_level"))]
    pub superchain_level: SuperchainLevel,
    /// Whether the chain is governed by optimism.
    #[cfg_attr(
        feature = "serde",
        serde(rename = "GovernedByOptimism", alias = "governed_by_optimism")
    )]
    #[cfg_attr(feature = "serde", serde(default))]
    pub governed_by_optimism: bool,
    /// Time of when a given chain is opted in to the Superchain.
    /// If set, hardforks times after the superchain time
    /// will be inherited from the superchain-wide config.
    #[cfg_attr(feature = "serde", serde(rename = "SuperchainTime", alias = "superchain_time"))]
    pub superchain_time: Option<u64>,
    /// Data availability type.
    #[cfg_attr(
        feature = "serde",
        serde(rename = "DataAvailabilityType", alias = "data_availability_type")
    )]
    pub data_availability_type: String,
    /// Chain ID
    #[cfg_attr(feature = "serde", serde(rename = "l2_chain_id", alias = "chain_id"))]
    pub chain_id: u64,
    /// Chain-specific batch inbox address
    #[cfg_attr(
        feature = "serde",
        serde(rename = "batch_inbox_address", alias = "batch_inbox_addr")
    )]
    #[cfg_attr(feature = "serde", serde(default))]
    pub batch_inbox_addr: Address,
    /// The block time in seconds.
    #[cfg_attr(feature = "serde", serde(rename = "block_time"))]
    pub block_time: u64,
    /// The sequencer window size in seconds.
    #[cfg_attr(feature = "serde", serde(rename = "seq_window_size"))]
    pub seq_window_size: u64,
    /// The maximum sequencer drift in seconds.
    #[cfg_attr(feature = "serde", serde(rename = "max_sequencer_drift"))]
    pub max_sequencer_drift: u64,
    /// Gas paying token metadata. Not consumed by downstream OPStack components.
    #[cfg_attr(feature = "serde", serde(rename = "GasPayingToken", alias = "gas_paying_token"))]
    pub gas_paying_token: Option<Address>,
    /// Hardfork Config. These values may override the superchain-wide defaults.
    #[cfg_attr(feature = "serde", serde(rename = "hardfork_configuration", alias = "hardforks"))]
    pub hardfork_config: HardForkConfig,
    /// Optimism configuration
    #[cfg_attr(feature = "serde", serde(rename = "optimism"))]
    pub optimism: Option<BaseFeeConfig>,
    /// Alternative DA configuration
    #[cfg_attr(feature = "serde", serde(rename = "alt_da"))]
    pub alt_da: Option<AltDAConfig>,
    /// Chain-specific genesis information
    pub genesis: ChainGenesis,
    /// Roles
    #[cfg_attr(feature = "serde", serde(rename = "Roles", alias = "roles"))]
    pub roles: Option<Roles>,
    /// Addresses
    #[cfg_attr(feature = "serde", serde(rename = "Addresses", alias = "addresses"))]
    pub addresses: Option<AddressList>,
}

impl ChainConfig {
    /// Returns the base fee params for the chain.
    pub fn base_fee_params(&self) -> BaseFeeParams {
        self.optimism
            .as_ref()
            .map(|op| op.pre_canyon_params())
            .unwrap_or_else(|| base_fee_params(self.chain_id))
    }

    /// Returns the canyon base fee params for the chain.
    pub fn canyon_base_fee_params(&self) -> BaseFeeParams {
        self.optimism
            .as_ref()
            .map(|op| op.post_canyon_params())
            .unwrap_or_else(|| base_fee_params_canyon(self.chain_id))
    }

    /// Returns the base fee config for the chain.
    pub fn base_fee_config(&self) -> BaseFeeConfig {
        self.optimism.as_ref().map(|op| *op).unwrap_or_else(|| base_fee_config(self.chain_id))
    }

    /// Loads the rollup config for the OP-Stack chain given the chain config and address list.
    #[deprecated(since = "0.2.1", note = "please use `as_rollup_config` instead")]
    pub fn load_op_stack_rollup_config(&self) -> RollupConfig {
        self.as_rollup_config()
    }

    /// Loads the rollup config for the OP-Stack chain given the chain config and address list.
    pub fn as_rollup_config(&self) -> RollupConfig {
        RollupConfig {
            genesis: self.genesis,
            l1_chain_id: self.l1_chain_id,
            l2_chain_id: Chain::from(self.chain_id),
            block_time: self.block_time,
            seq_window_size: self.seq_window_size,
            max_sequencer_drift: self.max_sequencer_drift,
            hardforks: self.hardfork_config,
            batch_inbox_address: self.batch_inbox_addr,
            deposit_contract_address: self
                .addresses
                .as_ref()
                .and_then(|a| a.optimism_portal_proxy)
                .unwrap_or_default(),
            l1_system_config_address: self
                .addresses
                .as_ref()
                .and_then(|a| a.system_config_proxy)
                .unwrap_or_default(),
            protocol_versions_address: self
                .addresses
                .as_ref()
                .and_then(|a| a.address_manager)
                .unwrap_or_default(),
            superchain_config_address: None,
            blobs_enabled_l1_timestamp: None,
            da_challenge_address: self
                .alt_da
                .as_ref()
                .and_then(|alt_da| alt_da.da_challenge_address),

            // The below chain parameters can be different per OP-Stack chain,
            // but since none of the superchain chains differ, it's not represented in the
            // superchain-registry yet. This restriction on superchain-chains may change in the
            // future. Test/Alt configurations can still load custom rollup-configs when
            // necessary.
            channel_timeout: 300,
            granite_channel_timeout: GRANITE_CHANNEL_TIMEOUT,
            interop_message_expiry_window: DEFAULT_INTEROP_MESSAGE_EXPIRY_WINDOW,
            chain_op_config: self.base_fee_config(),
            alt_da_config: self.alt_da.clone(),
        }
    }
}

#[cfg(test)]
#[cfg(feature = "serde")]
mod tests {
    use super::*;

    #[test]
    fn test_chain_config_json() {
        let raw: &str = r#"
        {
            "Name": "Base",
            "PublicRPC": "https://mainnet.base.org",
            "SequencerRPC": "https://mainnet-sequencer.base.org",
            "Explorer": "https://explorer.base.org",
            "SuperchainLevel": 1,
            "GovernedByOptimism": false,
            "SuperchainTime": 0,
            "DataAvailabilityType": "eth-da",
            "l2_chain_id": 8453,
            "batch_inbox_address": "0xff00000000000000000000000000000000008453",
            "block_time": 2,
            "seq_window_size": 3600,
            "max_sequencer_drift": 600,
            "GasPayingToken": null,
            "hardfork_configuration": {
                "canyon_time": 1704992401,
                "delta_time": 1708560000,
                "ecotone_time": 1710374401,
                "fjord_time": 1720627201,
                "granite_time": 1726070401,
                "holocene_time": 1736445601
            },
            "optimism": {
                "eip1559Elasticity": 6,
                "eip1559Denominator": 50,
                "eip1559DenominatorCanyon": 250
            },
            "alt_da": null,
            "genesis": {
                "l1": {
                  "number": 17481768,
                  "hash": "0x5c13d307623a926cd31415036c8b7fa14572f9dac64528e857a470511fc30771"
                },
                "l2": {
                  "number": 0,
                  "hash": "0xf712aa9241cc24369b143cf6dce85f0902a9731e70d66818a3a5845b296c73dd"
                },
                "l2_time": 1686789347,
                "system_config": {
                  "batcherAddress": "0x5050f69a9786f081509234f1a7f4684b5e5b76c9",
                  "overhead": "0xbc",
                  "scalar": "0xa6fe0",
                  "gasLimit": 30000000
                }
            },
            "Roles": {
                "SystemConfigOwner": "0x14536667cd30e52c0b458baaccb9fada7046e056",
                "ProxyAdminOwner": "0x7bb41c3008b3f03fe483b28b8db90e19cf07595c",
                "Guardian": "0x09f7150d8c019bef34450d6920f6b3608cefdaf2",
                "Challenger": "0x6f8c5ba3f59ea3e76300e3becdc231d656017824",
                "Proposer": "0x642229f238fb9de03374be34b0ed8d9de80752c5",
                "UnsafeBlockSigner": "0xaf6e19be0f9ce7f8afd49a1824851023a8249e8a",
                "BatchSubmitter": "0x5050f69a9786f081509234f1a7f4684b5e5b76c9"
            },
            "Addresses": {
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
        }
        "#;

        let deserialized: ChainConfig = serde_json::from_str(raw).unwrap();
        assert_eq!(deserialized.name, "Base");
    }

    #[test]
    fn test_chain_config_unknown_field_json() {
        let raw: &str = r#"
        {
            "Name": "Base",
            "PublicRPC": "https://mainnet.base.org",
            "SequencerRPC": "https://mainnet-sequencer.base.org",
            "Explorer": "https://explorer.base.org",
            "SuperchainLevel": 1,
            "GovernedByOptimism": false,
            "SuperchainTime": 0,
            "DataAvailabilityType": "eth-da",
            "l2_chain_id": 8453,
            "batch_inbox_address": "0xff00000000000000000000000000000000008453",
            "block_time": 2,
            "seq_window_size": 3600,
            "max_sequencer_drift": 600,
            "GasPayingToken": null,
            "hardfork_configuration": {
                "canyon_time": 1704992401,
                "delta_time": 1708560000,
                "ecotone_time": 1710374401,
                "fjord_time": 1720627201,
                "granite_time": 1726070401,
                "holocene_time": 1736445601
            },
            "optimism": {
            "eip1559Elasticity": "0x6",
            "eip1559Denominator": "0x32",
            "eip1559DenominatorCanyon": "0xfa"
            },
            "alt_da": null,
            "genesis": {
                "l1": {
                  "number": 17481768,
                  "hash": "0x5c13d307623a926cd31415036c8b7fa14572f9dac64528e857a470511fc30771"
                },
                "l2": {
                  "number": 0,
                  "hash": "0xf712aa9241cc24369b143cf6dce85f0902a9731e70d66818a3a5845b296c73dd"
                },
                "l2_time": 1686789347,
                "system_config": {
                  "batcherAddress": "0x5050f69a9786f081509234f1a7f4684b5e5b76c9",
                  "overhead": "0xbc",
                  "scalar": "0xa6fe0",
                  "gasLimit": 30000000
                }
            },
            "Roles": {
                "SystemConfigOwner": "0x14536667cd30e52c0b458baaccb9fada7046e056",
                "ProxyAdminOwner": "0x7bb41c3008b3f03fe483b28b8db90e19cf07595c",
                "Guardian": "0x09f7150d8c019bef34450d6920f6b3608cefdaf2",
                "Challenger": "0x6f8c5ba3f59ea3e76300e3becdc231d656017824",
                "Proposer": "0x642229f238fb9de03374be34b0ed8d9de80752c5",
                "UnsafeBlockSigner": "0xaf6e19be0f9ce7f8afd49a1824851023a8249e8a",
                "BatchSubmitter": "0x5050f69a9786f081509234f1a7f4684b5e5b76c9"
            },
            "Addresses": {
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
            },
            "unknown_field": "unknown"
        }
        "#;

        let err = serde_json::from_str::<ChainConfig>(raw).unwrap_err();
        assert_eq!(err.classify(), serde_json::error::Category::Data);
    }
}

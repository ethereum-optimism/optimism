//! Addresses of OP pre-deploys.
//!
//! This module contains the addresses of various predeploy contracts in the OP Stack.
//! See the complete set of predeploys at <https://specs.optimism.io/protocol/predeploys.html#predeploys>

use alloy_primitives::{Address, address};

/// Container for all predeploy contract addresses
#[derive(Debug, Copy, Clone, PartialEq, Eq)]
#[non_exhaustive]
pub struct Predeploys;

impl Predeploys {
    /// List of all predeploys.
    pub const ALL: [Address; 24] = [
        Self::LEGACY_MESSAGE_PASSER,
        Self::DEPLOYER_WHITELIST,
        Self::LEGACY_ERC20_ETH,
        Self::WETH9,
        Self::L2_CROSS_DOMAIN_MESSENGER,
        Self::L2_STANDARD_BRIDGE,
        Self::SEQUENCER_FEE_VAULT,
        Self::OP_MINTABLE_ERC20_FACTORY,
        Self::L1_BLOCK_NUMBER,
        Self::GAS_PRICE_ORACLE,
        Self::GOVERNANCE_TOKEN,
        Self::L1_BLOCK_INFO,
        Self::L2_TO_L1_MESSAGE_PASSER,
        Self::L2_ERC721_BRIDGE,
        Self::OP_MINTABLE_ERC721_FACTORY,
        Self::PROXY_ADMIN,
        Self::BASE_FEE_VAULT,
        Self::L1_FEE_VAULT,
        Self::SCHEMA_REGISTRY,
        Self::EAS,
        Self::BEACON_BLOCK_ROOT,
        Self::OPERATOR_FEE_VAULT,
        Self::CROSS_L2_INBOX,
        Self::L2_TO_L2_XDM,
    ];

    /// The LegacyMessagePasser contract stores commitments to withdrawal transactions before the
    /// Bedrock upgrade.
    /// <https://specs.optimism.io/protocol/predeploys.html#legacymessagepasser>
    pub const LEGACY_MESSAGE_PASSER: Address =
        address!("0x4200000000000000000000000000000000000000");

    /// The DeployerWhitelist was used to provide additional safety during initial phases of
    /// Optimism.
    /// <https://specs.optimism.io/protocol/predeploys.html#deployerwhitelist>
    pub const DEPLOYER_WHITELIST: Address = address!("0x4200000000000000000000000000000000000002");

    /// The LegacyERC20ETH predeploy represented all ether in the system before the Bedrock upgrade.
    /// <https://specs.optimism.io/protocol/predeploys.html#legacyerc20eth>
    pub const LEGACY_ERC20_ETH: Address = address!("0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000");

    /// The WETH9 predeploy address.
    /// <https://specs.optimism.io/protocol/predeploys.html#weth9>
    pub const WETH9: Address = address!("0x4200000000000000000000000000000000000006");

    /// Higher level API for sending cross domain messages.
    /// <https://specs.optimism.io/protocol/predeploys.html#l2crossdomainmessenger>
    pub const L2_CROSS_DOMAIN_MESSENGER: Address =
        address!("0x4200000000000000000000000000000000000007");

    /// The L2 cross-domain messenger proxy address.
    /// <https://specs.optimism.io/protocol/predeploys.html#l2standardbridge>
    pub const L2_STANDARD_BRIDGE: Address = address!("0x4200000000000000000000000000000000000010");

    /// The sequencer fee vault proxy address.
    /// <https://specs.optimism.io/protocol/predeploys.html#sequencerfeevault>
    pub const SEQUENCER_FEE_VAULT: Address = address!("0x4200000000000000000000000000000000000011");

    /// The Optimism mintable ERC20 factory proxy address.
    /// <https://specs.optimism.io/protocol/predeploys.html#optimismmintableerc20factory>
    pub const OP_MINTABLE_ERC20_FACTORY: Address =
        address!("0x4200000000000000000000000000000000000012");

    /// Returns the last known L1 block number (legacy system).
    /// <https://specs.optimism.io/protocol/predeploys.html#l1blocknumber>
    pub const L1_BLOCK_NUMBER: Address = address!("0x4200000000000000000000000000000000000013");

    /// The gas price oracle proxy address.
    /// <https://specs.optimism.io/protocol/predeploys.html#gaspriceoracle>
    pub const GAS_PRICE_ORACLE: Address = address!("0x420000000000000000000000000000000000000F");

    /// The governance token proxy address.
    /// <https://specs.optimism.io/governance/gov-token.html>
    pub const GOVERNANCE_TOKEN: Address = address!("0x4200000000000000000000000000000000000042");

    /// The L1 block information proxy address.
    /// <https://specs.optimism.io/protocol/predeploys.html#l1block>
    pub const L1_BLOCK_INFO: Address = address!("0x4200000000000000000000000000000000000015");

    /// The L2 contract `L2ToL1MessagePasser`, stores commitments to withdrawal transactions.
    /// <https://specs.optimism.io/protocol/predeploys.html#l2tol1messagepasser>
    pub const L2_TO_L1_MESSAGE_PASSER: Address =
        address!("0x4200000000000000000000000000000000000016");

    /// The L2 ERC721 bridge proxy address.
    /// <https://specs.optimism.io/protocol/predeploys.html>
    pub const L2_ERC721_BRIDGE: Address = address!("0x4200000000000000000000000000000000000014");

    /// The Optimism mintable ERC721 proxy address.
    /// <https://specs.optimism.io/protocol/predeploys.html#optimismmintableerc721factory>
    pub const OP_MINTABLE_ERC721_FACTORY: Address =
        address!("0x4200000000000000000000000000000000000017");

    /// The L2 proxy admin address.
    /// <https://specs.optimism.io/protocol/predeploys.html#proxyadmin>
    pub const PROXY_ADMIN: Address = address!("0x4200000000000000000000000000000000000018");

    /// The base fee vault address.
    /// <https://specs.optimism.io/protocol/predeploys.html#basefeevault>
    pub const BASE_FEE_VAULT: Address = address!("0x4200000000000000000000000000000000000019");

    /// The L1 fee vault address.
    /// <https://specs.optimism.io/protocol/predeploys.html#l1feevault>
    pub const L1_FEE_VAULT: Address = address!("0x420000000000000000000000000000000000001a");

    /// The schema registry proxy address.
    /// <https://specs.optimism.io/protocol/predeploys.html#schemaregistry>
    pub const SCHEMA_REGISTRY: Address = address!("0x4200000000000000000000000000000000000020");

    /// The EAS proxy address.
    /// <https://specs.optimism.io/protocol/predeploys.html#eas>
    pub const EAS: Address = address!("0x4200000000000000000000000000000000000021");

    /// Provides access to L1 beacon block roots (EIP-4788).
    /// <https://specs.optimism.io/protocol/predeploys.html#beacon-block-root>
    pub const BEACON_BLOCK_ROOT: Address = address!("0x000F3df6D732807Ef1319fB7B8bB8522d0Beac02");

    /// The Operator Fee Vault proxy address.
    pub const OPERATOR_FEE_VAULT: Address = address!("0x420000000000000000000000000000000000001B");

    /// The CrossL2Inbox proxy address.
    pub const CROSS_L2_INBOX: Address = address!("0x4200000000000000000000000000000000000022");

    /// The L2ToL2CrossDomainMessenger proxy address.
    pub const L2_TO_L2_XDM: Address = address!("0x4200000000000000000000000000000000000023");
}

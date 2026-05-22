// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Libraries
import { Fork } from "scripts/libraries/Config.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";

/// @title Predeploys
/// @notice Contains constant addresses for protocol contracts that are pre-deployed to the L2 system.
//          This excludes the preinstalls (non-protocol contracts).
library Predeploys {
    /// @notice Number of predeploy-namespace addresses reserved for protocol usage.
    uint256 internal constant PREDEPLOY_COUNT = 2048;

    /// @custom:legacy
    /// @notice Address of the LegacyMessagePasser predeploy. Deprecate. Use the updated
    ///         L2ToL1MessagePasser contract instead.
    address internal constant LEGACY_MESSAGE_PASSER = 0x4200000000000000000000000000000000000000;

    /// @custom:legacy
    /// @notice Address of the L1MessageSender predeploy. Deprecated. Use L2CrossDomainMessenger
    ///         or access tx.origin (or msg.sender) in a L1 to L2 transaction instead.
    ///         Not embedded into new OP-Stack chains.
    address internal constant L1_MESSAGE_SENDER = 0x4200000000000000000000000000000000000001;

    /// @custom:legacy
    /// @notice Address of the DeployerWhitelist predeploy. No longer active.
    address internal constant DEPLOYER_WHITELIST = 0x4200000000000000000000000000000000000002;

    /// @notice Address of the canonical WETH contract.
    address internal constant WETH = 0x4200000000000000000000000000000000000006;

    /// @notice Address of the L2CrossDomainMessenger predeploy.
    address internal constant L2_CROSS_DOMAIN_MESSENGER = 0x4200000000000000000000000000000000000007;

    /// @notice Address of the GasPriceOracle predeploy. Includes fee information
    ///         and helpers for computing the L1 portion of the transaction fee.
    address internal constant GAS_PRICE_ORACLE = 0x420000000000000000000000000000000000000F;

    /// @notice Address of the L2StandardBridge predeploy.
    address internal constant L2_STANDARD_BRIDGE = 0x4200000000000000000000000000000000000010;

    //// @notice Address of the SequencerFeeWallet predeploy.
    address internal constant SEQUENCER_FEE_WALLET = 0x4200000000000000000000000000000000000011;

    /// @notice Address of the OptimismMintableERC20Factory predeploy.
    address internal constant OPTIMISM_MINTABLE_ERC20_FACTORY = 0x4200000000000000000000000000000000000012;

    /// @custom:legacy
    /// @notice Address of the L1BlockNumber predeploy. Deprecated. Use the L1Block predeploy
    ///         instead, which exposes more information about the L1 state.
    address internal constant L1_BLOCK_NUMBER = 0x4200000000000000000000000000000000000013;

    /// @notice Address of the L2ERC721Bridge predeploy.
    address internal constant L2_ERC721_BRIDGE = 0x4200000000000000000000000000000000000014;

    /// @notice Address of the L1Block predeploy.
    address internal constant L1_BLOCK_ATTRIBUTES = 0x4200000000000000000000000000000000000015;

    /// @notice Address of the L2ToL1MessagePasser predeploy.
    address internal constant L2_TO_L1_MESSAGE_PASSER = 0x4200000000000000000000000000000000000016;

    /// @notice Address of the OptimismMintableERC721Factory predeploy.
    address internal constant OPTIMISM_MINTABLE_ERC721_FACTORY = 0x4200000000000000000000000000000000000017;

    /// @notice Address of the L2ProxyAdmin predeploy.
    address internal constant PROXY_ADMIN = 0x4200000000000000000000000000000000000018;

    /// @notice Address of the BaseFeeVault predeploy.
    address internal constant BASE_FEE_VAULT = 0x4200000000000000000000000000000000000019;

    /// @notice Address of the L1FeeVault predeploy.
    address internal constant L1_FEE_VAULT = 0x420000000000000000000000000000000000001A;

    /// @notice Address of the OperatorFeeVault predeploy.
    address internal constant OPERATOR_FEE_VAULT = 0x420000000000000000000000000000000000001b;

    /// @notice Address of the SchemaRegistry predeploy.
    address internal constant SCHEMA_REGISTRY = 0x4200000000000000000000000000000000000020;

    /// @notice Address of the EAS predeploy.
    address internal constant EAS = 0x4200000000000000000000000000000000000021;

    /// @notice Address of the GovernanceToken predeploy.
    address internal constant GOVERNANCE_TOKEN = 0x4200000000000000000000000000000000000042;

    /// @custom:legacy
    /// @notice Address of the LegacyERC20ETH predeploy. Deprecated. Balances are migrated to the
    ///         state trie as of the Bedrock upgrade. Contract has been locked and write functions
    ///         can no longer be accessed.
    address internal constant LEGACY_ERC20_ETH = 0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000;

    /// @notice Address of the CrossL2Inbox predeploy.
    address internal constant CROSS_L2_INBOX = 0x4200000000000000000000000000000000000022;

    /// @notice Address of the L2ToL2CrossDomainMessenger predeploy.
    address internal constant L2_TO_L2_CROSS_DOMAIN_MESSENGER = 0x4200000000000000000000000000000000000023;

    /// @notice Address of the SuperchainETHBridge predeploy.
    address internal constant SUPERCHAIN_ETH_BRIDGE = 0x4200000000000000000000000000000000000024;

    /// @notice Address of the ETHLiquidity predeploy.
    address internal constant ETH_LIQUIDITY = 0x4200000000000000000000000000000000000025;

    /// @notice Address of the NativeAssetLiquidity predeploy.
    address internal constant NATIVE_ASSET_LIQUIDITY = 0x4200000000000000000000000000000000000029;

    /// @notice Address of the LiquidityController predeploy.
    address internal constant LIQUIDITY_CONTROLLER = 0x420000000000000000000000000000000000002a;

    /// @notice Address of the ConditionalDeployer predeploy.
    address internal constant CONDITIONAL_DEPLOYER = 0x420000000000000000000000000000000000002C;

    /// @notice Address of the L2DevFeatureFlags predeploy.
    address internal constant L2_DEV_FEATURE_FLAGS = 0x420000000000000000000000000000000000002d;

    /// @notice Configuration record for a single predeploy implementation.
    /// @param proxy             Canonical proxy address (0x4200...). CGT variants share the same
    ///                          proxy as their standard counterpart.
    /// @param name              Implementation contract name (e.g. "L1Block", "L1BlockCGT").
    /// @param artifactPath      Forge artifact path ("Contract.sol:Contract").
    /// @param deployGasLimit    Gas limit for deployments in NUT bundles.
    ///                          Based on gas profiling with a safety margin.
    /// @param devFeatureGate    DevFeatures constant that gates this predeploy on L2 genesis.
    /// @param isCustomGasToken  True if this predeploy is only deployed on custom gas token chains.
    /// @param isInterop         True if this predeploy is only deployed on interop-enabled chains.
    /// @param isProxied         True if the predeploy uses a Proxy. Non-proxied predeploys
    ///                          (WETH, GovernanceToken) are etched directly without proxy or
    ///                          implementation slot setup, and are excluded from NUT bundles.
    /// @param isDeprecated      True if the predeploy is deprecated. Deprecated predeploys are
    ///                          present on-chain for backwards compatibility but are excluded from
    ///                          proxy setup loops, NUT bundles, and upgrade checks.
    /// @param isVariant         True for records that share a proxy address with another (primary) record.
    struct PredeployRecord {
        address proxy;
        string name;
        string artifactPath;
        uint64 deployGasLimit;
        bytes32 devFeatureGate;
        bool isCustomGasToken;
        bool isInterop;
        bool isProxied;
        bool isDeprecated;
        bool isVariant;
    }

    /// @notice Returns the name of the predeploy at the given address.
    ///         Always returns the primary, non-variant, record name for a given proxy address.
    ///         e.g. getName(L1_BLOCK_ATTRIBUTES) always returns "L1Block", not "L1BlockCGT".
    function getName(address _addr) internal pure returns (string memory out_) {
        require(isPredeployNamespace(_addr), "Predeploys: address must be a predeploy");
        PredeployRecord[] memory records = getAllRecords();
        for (uint256 i = 0; i < records.length; i++) {
            if (records[i].proxy == _addr && !records[i].isVariant) {
                return records[i].name;
            }
        }
        revert("Predeploys: unnamed predeploy");
    }

    /// @notice Returns true if the predeploy is not proxied.
    function notProxied(address _addr) internal pure returns (bool) {
        PredeployRecord[] memory records = getAllRecords();
        for (uint256 i = 0; i < records.length; i++) {
            if (records[i].proxy == _addr) {
                return !records[i].isProxied;
            }
        }
        return false;
    }

    /// @notice Returns true if the address is in the predeploy namespace.
    /// @param _addr The address to check.
    /// @return True if the address is in range 0x4200...0000 to 0x4200...07FF.
    function isPredeployNamespace(address _addr) internal pure returns (bool) {
        return uint160(_addr) >> 11 == uint160(0x4200000000000000000000000000000000000000) >> 11;
    }

    /// @notice Function to compute the expected address of the predeploy implementation
    ///         in the genesis state.
    function predeployToCodeNamespace(address _addr) internal pure returns (address) {
        require(
            isPredeployNamespace(_addr), "Predeploys: can only derive code-namespace address for predeploy addresses"
        );
        return address(
            uint160(uint256(uint160(_addr)) & 0xffff | uint256(uint160(0xc0D3C0d3C0d3C0D3c0d3C0d3c0D3C0d3c0d30000)))
        );
    }

    /// @notice Returns true if the predeploy is upgradeable. In this context, upgradeable means that the predeploy
    ///         is in the predeploy namespace and it is proxied.
    /// @param _proxy The address of the predeploy.
    /// @return isUpgradeable_ True if the predeploy is upgradeable, false otherwise.
    function isUpgradeable(address _proxy) internal pure returns (bool isUpgradeable_) {
        isUpgradeable_ = isPredeployNamespace(_proxy) && !notProxied(_proxy);
    }

    /// @notice Returns all predeploy implementation records.
    /// @dev THE SINGLE SOURCE OF TRUTH for predeploy configuration.
    ///      When adding a new predeploy, update ALL of the following:
    ///        1. Predeploys.sol (this file)
    ///           - Add an `address internal constant` for the proxy address above.
    ///           - Add a `PredeployRecord` entry here and bump the array size.
    ///        2. src/L2/L2ContractsManager.sol
    ///           - Add an `address internal immutable` declaration.
    ///           - Assign it via `findImpl()` in the constructor.
    ///           - Call `upgradeTo` or `upgradeToAndCall` for it in `_apply()`.
    ///           - Add an entry to `getImplementations()` and bump the array count.
    ///        3. scripts/L2Genesis.s.sol
    ///           - Add a `setXxx()` setter function.
    ///           - Call it from `setPredeployImplementations()`.
    ///      Non-proxied records (isProxied = false) are appended at the end and must be skipped
    ///      by consumers that operate on proxy/implementation slots (NUT bundle, setPredeployProxies).
    ///      Deprecated records (isDeprecated = true) are appended after non-proxied records and must
    ///      be skipped by consumers that perform proxy setup, NUT bundles, or upgrade checks.
    function getAllRecords() internal pure returns (PredeployRecord[] memory records_) {
        records_ = new PredeployRecord[](30);

        // ── Core predeploys ────────────────────────────────────────────────────────────────
        records_[0] = PredeployRecord({
            proxy: L2_CROSS_DOMAIN_MESSENGER,
            name: "L2CrossDomainMessenger",
            artifactPath: "L2CrossDomainMessenger.sol:L2CrossDomainMessenger",
            deployGasLimit: 3_129_000,
            devFeatureGate: bytes32(0),
            isCustomGasToken: false,
            isInterop: false,
            isProxied: true,
            isDeprecated: false,
            isVariant: false
        });
        records_[1] = PredeployRecord({
            proxy: GAS_PRICE_ORACLE,
            name: "GasPriceOracle",
            artifactPath: "GasPriceOracle.sol:GasPriceOracle",
            deployGasLimit: 2_762_000,
            devFeatureGate: bytes32(0),
            isCustomGasToken: false,
            isInterop: false,
            isProxied: true,
            isDeprecated: false,
            isVariant: false
        });
        records_[2] = PredeployRecord({
            proxy: L2_STANDARD_BRIDGE,
            name: "L2StandardBridge",
            artifactPath: "L2StandardBridge.sol:L2StandardBridge",
            deployGasLimit: 4_193_000,
            devFeatureGate: bytes32(0),
            isCustomGasToken: false,
            isInterop: false,
            isProxied: true,
            isDeprecated: false,
            isVariant: false
        });
        records_[3] = PredeployRecord({
            proxy: SEQUENCER_FEE_WALLET,
            name: "SequencerFeeVault",
            artifactPath: "SequencerFeeVault.sol:SequencerFeeVault",
            deployGasLimit: 1_506_000,
            devFeatureGate: bytes32(0),
            isCustomGasToken: false,
            isInterop: false,
            isProxied: true,
            isDeprecated: false,
            isVariant: false
        });
        records_[4] = PredeployRecord({
            proxy: OPTIMISM_MINTABLE_ERC20_FACTORY,
            name: "OptimismMintableERC20Factory",
            artifactPath: "OptimismMintableERC20Factory.sol:OptimismMintableERC20Factory",
            deployGasLimit: 4_193_000,
            devFeatureGate: bytes32(0),
            isCustomGasToken: false,
            isInterop: false,
            isProxied: true,
            isDeprecated: false,
            isVariant: false
        });
        records_[5] = PredeployRecord({
            proxy: L2_ERC721_BRIDGE,
            name: "L2ERC721Bridge",
            artifactPath: "L2ERC721Bridge.sol:L2ERC721Bridge",
            deployGasLimit: 2_367_000,
            devFeatureGate: bytes32(0),
            isCustomGasToken: false,
            isInterop: false,
            isProxied: true,
            isDeprecated: false,
            isVariant: false
        });
        records_[6] = PredeployRecord({
            proxy: L1_BLOCK_ATTRIBUTES,
            name: "L1Block",
            artifactPath: "L1Block.sol:L1Block",
            deployGasLimit: 1_191_000,
            devFeatureGate: bytes32(0),
            isCustomGasToken: false,
            isInterop: false,
            isProxied: true,
            isDeprecated: false,
            isVariant: false
        });
        records_[7] = PredeployRecord({
            proxy: L1_BLOCK_ATTRIBUTES,
            name: "L1BlockCGT",
            artifactPath: "L1BlockCGT.sol:L1BlockCGT",
            deployGasLimit: 1_568_000,
            devFeatureGate: bytes32(0),
            isCustomGasToken: true,
            isInterop: false,
            isProxied: true,
            isDeprecated: false,
            isVariant: true
        });
        records_[8] = PredeployRecord({
            proxy: L2_TO_L1_MESSAGE_PASSER,
            name: "L2ToL1MessagePasser",
            artifactPath: "L2ToL1MessagePasser.sol:L2ToL1MessagePasser",
            deployGasLimit: 694_000,
            devFeatureGate: bytes32(0),
            isCustomGasToken: false,
            isInterop: false,
            isProxied: true,
            isDeprecated: false,
            isVariant: false
        });
        records_[9] = PredeployRecord({
            proxy: L2_TO_L1_MESSAGE_PASSER,
            name: "L2ToL1MessagePasserCGT",
            artifactPath: "L2ToL1MessagePasserCGT.sol:L2ToL1MessagePasserCGT",
            deployGasLimit: 827_394,
            devFeatureGate: bytes32(0),
            isCustomGasToken: true,
            isInterop: false,
            isProxied: true,
            isDeprecated: false,
            isVariant: true
        });
        records_[10] = PredeployRecord({
            proxy: OPTIMISM_MINTABLE_ERC721_FACTORY,
            name: "OptimismMintableERC721Factory",
            artifactPath: "OptimismMintableERC721Factory.sol:OptimismMintableERC721Factory",
            deployGasLimit: 5_661_000,
            devFeatureGate: bytes32(0),
            isCustomGasToken: false,
            isInterop: false,
            isProxied: true,
            isDeprecated: false,
            isVariant: false
        });
        records_[11] = PredeployRecord({
            proxy: PROXY_ADMIN,
            name: "L2ProxyAdmin",
            artifactPath: "L2ProxyAdmin.sol:L2ProxyAdmin",
            deployGasLimit: 2_541_000,
            devFeatureGate: bytes32(0),
            isCustomGasToken: false,
            isInterop: false,
            isProxied: true,
            isDeprecated: false,
            isVariant: false
        });
        records_[12] = PredeployRecord({
            proxy: BASE_FEE_VAULT,
            name: "BaseFeeVault",
            artifactPath: "BaseFeeVault.sol:BaseFeeVault",
            deployGasLimit: 1_503_000,
            devFeatureGate: bytes32(0),
            isCustomGasToken: false,
            isInterop: false,
            isProxied: true,
            isDeprecated: false,
            isVariant: false
        });
        records_[13] = PredeployRecord({
            proxy: L1_FEE_VAULT,
            name: "L1FeeVault",
            artifactPath: "L1FeeVault.sol:L1FeeVault",
            deployGasLimit: 260_550,
            devFeatureGate: bytes32(0),
            isCustomGasToken: false,
            isInterop: false,
            isProxied: true,
            isDeprecated: false,
            isVariant: false
        });
        records_[14] = PredeployRecord({
            proxy: OPERATOR_FEE_VAULT,
            name: "OperatorFeeVault",
            artifactPath: "OperatorFeeVault.sol:OperatorFeeVault",
            deployGasLimit: 1_504_000,
            devFeatureGate: bytes32(0),
            isCustomGasToken: false,
            isInterop: false,
            isProxied: true,
            isDeprecated: false,
            isVariant: false
        });
        records_[15] = PredeployRecord({
            proxy: SCHEMA_REGISTRY,
            name: "SchemaRegistry",
            artifactPath: "SchemaRegistry.sol:SchemaRegistry",
            deployGasLimit: 805_000,
            devFeatureGate: bytes32(0),
            isCustomGasToken: false,
            isInterop: false,
            isProxied: true,
            isDeprecated: false,
            isVariant: false
        });
        records_[16] = PredeployRecord({
            proxy: EAS,
            name: "EAS",
            artifactPath: "EAS.sol:EAS",
            deployGasLimit: 6_251_000,
            devFeatureGate: bytes32(0),
            isCustomGasToken: false,
            isInterop: false,
            isProxied: true,
            isDeprecated: false,
            isVariant: false
        });

        // ── Interop predeploys ─────────────────────────────────────────────────────────────
        // Interop requires both the INTEROP sys feature and the OPTIMISM_PORTAL_INTEROP dev
        // feature. Both gates mirror the full condition checked in L2Genesis.
        records_[17] = PredeployRecord({
            proxy: CROSS_L2_INBOX,
            name: "CrossL2Inbox",
            artifactPath: "CrossL2Inbox.sol:CrossL2Inbox",
            deployGasLimit: 668_000,
            devFeatureGate: DevFeatures.OPTIMISM_PORTAL_INTEROP,
            isCustomGasToken: false,
            isInterop: true,
            isProxied: true,
            isDeprecated: false,
            isVariant: false
        });
        records_[18] = PredeployRecord({
            proxy: L2_TO_L2_CROSS_DOMAIN_MESSENGER,
            name: "L2ToL2CrossDomainMessenger",
            artifactPath: "L2ToL2CrossDomainMessenger.sol:L2ToL2CrossDomainMessenger",
            deployGasLimit: 1_611_000,
            devFeatureGate: DevFeatures.OPTIMISM_PORTAL_INTEROP,
            isCustomGasToken: false,
            isInterop: true,
            isProxied: true,
            isDeprecated: false,
            isVariant: false
        });
        records_[19] = PredeployRecord({
            proxy: SUPERCHAIN_ETH_BRIDGE,
            name: "SuperchainETHBridge",
            artifactPath: "SuperchainETHBridge.sol:SuperchainETHBridge",
            deployGasLimit: 757_000,
            devFeatureGate: DevFeatures.OPTIMISM_PORTAL_INTEROP,
            isCustomGasToken: false,
            isInterop: true,
            isProxied: true,
            isDeprecated: false,
            isVariant: false
        });
        records_[20] = PredeployRecord({
            proxy: ETH_LIQUIDITY,
            name: "ETHLiquidity",
            artifactPath: "ETHLiquidity.sol:ETHLiquidity",
            deployGasLimit: 423_000,
            devFeatureGate: DevFeatures.OPTIMISM_PORTAL_INTEROP,
            isCustomGasToken: false,
            isInterop: true,
            isProxied: true,
            isDeprecated: false,
            isVariant: false
        });

        // ── CGT predeploys ─────────────────────────────────────────────────────────────────
        records_[21] = PredeployRecord({
            proxy: NATIVE_ASSET_LIQUIDITY,
            name: "NativeAssetLiquidity",
            artifactPath: "NativeAssetLiquidity.sol:NativeAssetLiquidity",
            deployGasLimit: 392_000,
            devFeatureGate: bytes32(0),
            isCustomGasToken: true,
            isInterop: false,
            isProxied: true,
            isDeprecated: false,
            isVariant: false
        });
        records_[22] = PredeployRecord({
            proxy: LIQUIDITY_CONTROLLER,
            name: "LiquidityController",
            artifactPath: "LiquidityController.sol:LiquidityController",
            deployGasLimit: 1_870_000,
            devFeatureGate: bytes32(0),
            isCustomGasToken: true,
            isInterop: false,
            isProxied: true,
            isDeprecated: false,
            isVariant: false
        });

        records_[23] = PredeployRecord({
            proxy: CONDITIONAL_DEPLOYER,
            name: "ConditionalDeployer",
            artifactPath: "ConditionalDeployer.sol:ConditionalDeployer",
            deployGasLimit: 116_400,
            devFeatureGate: DevFeatures.L2CM,
            isCustomGasToken: false,
            isInterop: false,
            isProxied: true,
            isDeprecated: false,
            isVariant: false
        });
        records_[24] = PredeployRecord({
            proxy: L2_DEV_FEATURE_FLAGS,
            name: "L2DevFeatureFlags",
            artifactPath: "L2DevFeatureFlags.sol:L2DevFeatureFlags",
            deployGasLimit: 328_228,
            devFeatureGate: DevFeatures.L2CM,
            isCustomGasToken: false,
            isInterop: false,
            isProxied: true,
            isDeprecated: false,
            isVariant: false
        });

        // ── Non-proxied predeploys ─────────────────────────────────────────────────────────
        // These are etched directly (no Proxy wrapper, no implementation slot).
        // Excluded from NUT bundles and proxy setup. deployGasLimit is unused.
        records_[25] = PredeployRecord({
            proxy: WETH,
            name: "WETH",
            artifactPath: "WETH.sol:WETH",
            deployGasLimit: 0,
            devFeatureGate: bytes32(0),
            isCustomGasToken: false,
            isInterop: false,
            isProxied: false,
            isDeprecated: false,
            isVariant: false
        });
        records_[26] = PredeployRecord({
            proxy: GOVERNANCE_TOKEN,
            name: "GovernanceToken",
            artifactPath: "GovernanceToken.sol:GovernanceToken",
            deployGasLimit: 0,
            devFeatureGate: bytes32(0),
            isCustomGasToken: false,
            isInterop: false,
            isProxied: false,
            isDeprecated: false,
            isVariant: false
        });

        // ── Deprecated predeploys ──────────────────────────────────────────────────────────
        // Present on-chain for backwards compatibility but excluded from proxy setup loops,
        // NUT bundles, and upgrade checks. Handled by individual setters in L2Genesis.
        records_[27] = PredeployRecord({
            proxy: LEGACY_MESSAGE_PASSER,
            name: "LegacyMessagePasser",
            artifactPath: "LegacyMessagePasser.sol:LegacyMessagePasser",
            deployGasLimit: 0,
            devFeatureGate: bytes32(0),
            isCustomGasToken: false,
            isInterop: false,
            isProxied: true,
            isDeprecated: true,
            isVariant: false
        });
        records_[28] = PredeployRecord({
            proxy: DEPLOYER_WHITELIST,
            name: "DeployerWhitelist",
            artifactPath: "DeployerWhitelist.sol:DeployerWhitelist",
            deployGasLimit: 0,
            devFeatureGate: bytes32(0),
            isCustomGasToken: false,
            isInterop: false,
            isProxied: true,
            isDeprecated: true,
            isVariant: false
        });
        records_[29] = PredeployRecord({
            proxy: L1_BLOCK_NUMBER,
            name: "L1BlockNumber",
            artifactPath: "L1BlockNumber.sol:L1BlockNumber",
            deployGasLimit: 0,
            devFeatureGate: bytes32(0),
            isCustomGasToken: false,
            isInterop: false,
            isProxied: true,
            isDeprecated: true,
            isVariant: false
        });
    }

    /// @notice Returns all proxied, non-deprecated predeploy records, including variant records.
    /// @dev Variant records (isVariant = true) share a proxy with a primary record (e.g. L1BlockCGT
    ///      shares the L1Block proxy). Callers that need one entry per proxy should skip variants.
    function getUpgradeableRecords() internal pure returns (PredeployRecord[] memory records_) {
        PredeployRecord[] memory all = getAllRecords();
        uint256 count = 0;
        for (uint256 i = 0; i < all.length; i++) {
            if (!all[i].isProxied || all[i].isDeprecated) continue;
            count++;
        }
        // Create a new array with the proxied, non-deprecated records
        records_ = new PredeployRecord[](count);
        uint256 j = 0;
        for (uint256 i = 0; i < all.length; i++) {
            if (!all[i].isProxied || all[i].isDeprecated) continue;
            records_[j++] = all[i];
        }
    }

    /// @notice Asserts that the registry record for `_proxy` has the expected gate fields.
    ///         Reverts if no matching record is found or if the gates differ.
    ///         Called by L2Genesis setters to self-verify their own registry configuration,
    ///         catching any drift between a setter's assumed gates and the registry.
    function assertGates(address _proxy, bytes32 _devGate, bool _isCustomGasToken, bool _isInterop) internal pure {
        PredeployRecord[] memory records = getAllRecords();
        for (uint256 i = 0; i < records.length; i++) {
            if (records[i].proxy == _proxy) {
                require(
                    records[i].devFeatureGate == _devGate && records[i].isCustomGasToken == _isCustomGasToken
                        && records[i].isInterop == _isInterop,
                    "Predeploys: gate mismatch"
                );
                return;
            }
        }
        revert("Predeploys: proxy not found");
    }

    function isSupportedPredeploy(
        address _addr,
        uint256 _fork,
        bool _isCustomGasToken,
        bool _useInterop,
        bytes32 _devFeatureBitmap
    )
        internal
        pure
        returns (bool)
    {
        // iterate over all records and check if the predeploy is supported based on the arguments
        PredeployRecord[] memory records = getAllRecords();
        for (uint256 i = 0; i < records.length; i++) {
            if (records[i].proxy == _addr) {
                if (records[i].devFeatureGate != 0) {
                    // If the feature on the gate is not present on the bitmap, the predeploy is not supported.
                    if (!DevFeatures.isDevFeatureEnabled(_devFeatureBitmap, records[i].devFeatureGate)) {
                        return false;
                    }
                    // Additional conditions for interop
                    if (DevFeatures.isDevFeatureEnabled(records[i].devFeatureGate, DevFeatures.OPTIMISM_PORTAL_INTEROP))
                    {
                        if (_fork < uint256(Fork.INTEROP) || !_useInterop) {
                            return false;
                        }
                    }
                }

                // If the predeploy has a system feature gate, check if it is supported based on the arguments.
                if ((records[i].isCustomGasToken && !_isCustomGasToken) || (records[i].isInterop && !_useInterop)) {
                    return false;
                }

                return true;
            }
        }
        return false;
    }
}

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

    /// @notice Address of the OptimismSuperchainERC20Factory predeploy.
    address internal constant OPTIMISM_SUPERCHAIN_ERC20_FACTORY = 0x4200000000000000000000000000000000000026;

    /// @notice Address of the OptimismSuperchainERC20Beacon predeploy.
    address internal constant OPTIMISM_SUPERCHAIN_ERC20_BEACON = 0x4200000000000000000000000000000000000027;

    // TODO: Precalculate the address of the implementation contract
    /// @notice Arbitrary address of the OptimismSuperchainERC20 implementation contract.
    address internal constant OPTIMISM_SUPERCHAIN_ERC20 = 0xB9415c6cA93bdC545D4c5177512FCC22EFa38F28;

    /// @notice Address of the SuperchainTokenBridge predeploy.
    address internal constant SUPERCHAIN_TOKEN_BRIDGE = 0x4200000000000000000000000000000000000028;

    /// @notice Address of the NativeAssetLiquidity predeploy.
    address internal constant NATIVE_ASSET_LIQUIDITY = 0x4200000000000000000000000000000000000029;

    /// @notice Address of the LiquidityController predeploy.
    address internal constant LIQUIDITY_CONTROLLER = 0x420000000000000000000000000000000000002a;

    /// @notice Address of the FeeSplitter predeploy.
    address internal constant FEE_SPLITTER = 0x420000000000000000000000000000000000002B;

    /// @notice Address of the ConditionalDeployer predeploy.
    address internal constant CONDITIONAL_DEPLOYER = 0x420000000000000000000000000000000000002C;

    /// @notice Address of the L2DevFeatureFlags predeploy.
    address internal constant L2_DEV_FEATURE_FLAGS = 0x420000000000000000000000000000000000002d;

    /// @notice Returns the name of the predeploy at the given address.
    function getName(address _addr) internal pure returns (string memory out_) {
        require(isPredeployNamespace(_addr), "Predeploys: address must be a predeploy");
        if (_addr == LEGACY_MESSAGE_PASSER) return "LegacyMessagePasser";
        if (_addr == L1_MESSAGE_SENDER) return "L1MessageSender";
        if (_addr == DEPLOYER_WHITELIST) return "DeployerWhitelist";
        if (_addr == WETH) return "WETH";
        if (_addr == L2_CROSS_DOMAIN_MESSENGER) return "L2CrossDomainMessenger";
        if (_addr == GAS_PRICE_ORACLE) return "GasPriceOracle";
        if (_addr == L2_STANDARD_BRIDGE) return "L2StandardBridge";
        if (_addr == SEQUENCER_FEE_WALLET) return "SequencerFeeVault";
        if (_addr == OPTIMISM_MINTABLE_ERC20_FACTORY) return "OptimismMintableERC20Factory";
        if (_addr == L1_BLOCK_NUMBER) return "L1BlockNumber";
        if (_addr == L2_ERC721_BRIDGE) return "L2ERC721Bridge";
        if (_addr == L1_BLOCK_ATTRIBUTES) return "L1Block";
        if (_addr == L2_TO_L1_MESSAGE_PASSER) return "L2ToL1MessagePasser";
        if (_addr == OPTIMISM_MINTABLE_ERC721_FACTORY) return "OptimismMintableERC721Factory";
        if (_addr == PROXY_ADMIN) return "L2ProxyAdmin";
        if (_addr == BASE_FEE_VAULT) return "BaseFeeVault";
        if (_addr == L1_FEE_VAULT) return "L1FeeVault";
        if (_addr == OPERATOR_FEE_VAULT) return "OperatorFeeVault";
        if (_addr == SCHEMA_REGISTRY) return "SchemaRegistry";
        if (_addr == EAS) return "EAS";
        if (_addr == GOVERNANCE_TOKEN) return "GovernanceToken";
        if (_addr == LEGACY_ERC20_ETH) return "LegacyERC20ETH";
        if (_addr == CROSS_L2_INBOX) return "CrossL2Inbox";
        if (_addr == L2_TO_L2_CROSS_DOMAIN_MESSENGER) return "L2ToL2CrossDomainMessenger";
        if (_addr == SUPERCHAIN_ETH_BRIDGE) return "SuperchainETHBridge";
        if (_addr == ETH_LIQUIDITY) return "ETHLiquidity";
        if (_addr == OPTIMISM_SUPERCHAIN_ERC20_FACTORY) return "OptimismSuperchainERC20Factory";
        if (_addr == OPTIMISM_SUPERCHAIN_ERC20_BEACON) return "OptimismSuperchainERC20Beacon";
        if (_addr == SUPERCHAIN_TOKEN_BRIDGE) return "SuperchainTokenBridge";
        if (_addr == LIQUIDITY_CONTROLLER) return "LiquidityController";
        if (_addr == NATIVE_ASSET_LIQUIDITY) return "NativeAssetLiquidity";
        if (_addr == FEE_SPLITTER) return "FeeSplitter";
        if (_addr == CONDITIONAL_DEPLOYER) return "ConditionalDeployer";
        if (_addr == L2_DEV_FEATURE_FLAGS) return "L2DevFeatureFlags";
        revert("Predeploys: unnamed predeploy");
    }

    /// @notice Returns true if the predeploy is not proxied.
    function notProxied(address _addr) internal pure returns (bool) {
        return _addr == GOVERNANCE_TOKEN || _addr == WETH;
    }

    /// @notice Metadata for a predeploy in the registry. Each entry is the single source of truth
    ///         for whether a predeploy is supported (deployed at genesis) and upgradeable (managed by L2CM).
    struct PredeployEntry {
        address addr;
        bool upgradeable;
        bool requiresInterop;
        bool requiresCGT;
        bytes32 requiredDevFeature;
    }

    /// @notice Returns the full predeploy registry. This is the SINGLE SOURCE OF TRUTH for predeploy
    ///         support and upgrade coverage. Adding a new predeploy means adding ONE entry here.
    ///
    /// @dev The registry includes all predeploys that should be deployed at genesis AND/OR upgraded
    ///      by L2CM. Legacy predeploys that are deployed but never upgraded have upgradeable=false.
    ///      Non-proxied predeploys (WETH, GOVERNANCE_TOKEN) are excluded — they are handled separately.
    function _registry() internal pure returns (PredeployEntry[] memory entries_) {
        entries_ = new PredeployEntry[](27);

        // Core predeploys — always supported, upgradeable
        entries_[0] = PredeployEntry(L2_CROSS_DOMAIN_MESSENGER, true, false, false, bytes32(0));
        entries_[1] = PredeployEntry(GAS_PRICE_ORACLE, true, false, false, bytes32(0));
        entries_[2] = PredeployEntry(L2_STANDARD_BRIDGE, true, false, false, bytes32(0));
        entries_[3] = PredeployEntry(SEQUENCER_FEE_WALLET, true, false, false, bytes32(0));
        entries_[4] = PredeployEntry(OPTIMISM_MINTABLE_ERC20_FACTORY, true, false, false, bytes32(0));
        entries_[5] = PredeployEntry(L2_ERC721_BRIDGE, true, false, false, bytes32(0));
        entries_[6] = PredeployEntry(L1_BLOCK_ATTRIBUTES, true, false, false, bytes32(0));
        entries_[7] = PredeployEntry(L2_TO_L1_MESSAGE_PASSER, true, false, false, bytes32(0));
        entries_[8] = PredeployEntry(OPTIMISM_MINTABLE_ERC721_FACTORY, true, false, false, bytes32(0));
        entries_[9] = PredeployEntry(PROXY_ADMIN, true, false, false, bytes32(0));
        entries_[10] = PredeployEntry(BASE_FEE_VAULT, true, false, false, bytes32(0));
        entries_[11] = PredeployEntry(L1_FEE_VAULT, true, false, false, bytes32(0));
        entries_[12] = PredeployEntry(OPERATOR_FEE_VAULT, true, false, false, bytes32(0));
        entries_[13] = PredeployEntry(SCHEMA_REGISTRY, true, false, false, bytes32(0));
        entries_[14] = PredeployEntry(EAS, true, false, false, bytes32(0));
        entries_[15] = PredeployEntry(FEE_SPLITTER, true, false, false, bytes32(0));

        // Legacy predeploys — supported at genesis, but NOT upgradeable (no L2CM management)
        entries_[16] = PredeployEntry(LEGACY_MESSAGE_PASSER, false, false, false, bytes32(0));
        entries_[17] = PredeployEntry(DEPLOYER_WHITELIST, false, false, false, bytes32(0));
        entries_[18] = PredeployEntry(L1_BLOCK_NUMBER, false, false, false, bytes32(0));

        // Interop predeploys — require fork >= INTEROP, interop dev feature, and useInterop
        entries_[19] = PredeployEntry(CROSS_L2_INBOX, true, true, false, bytes32(0));
        entries_[20] = PredeployEntry(L2_TO_L2_CROSS_DOMAIN_MESSENGER, true, true, false, bytes32(0));
        entries_[21] = PredeployEntry(SUPERCHAIN_ETH_BRIDGE, true, true, false, bytes32(0));
        entries_[22] = PredeployEntry(ETH_LIQUIDITY, true, true, false, bytes32(0));

        // CGT predeploys — require custom gas token
        entries_[23] = PredeployEntry(NATIVE_ASSET_LIQUIDITY, true, false, true, bytes32(0));
        entries_[24] = PredeployEntry(LIQUIDITY_CONTROLLER, true, false, true, bytes32(0));

        // L2CM dev feature predeploys
        entries_[25] = PredeployEntry(CONDITIONAL_DEPLOYER, true, false, false, DevFeatures.L2CM);
        entries_[26] = PredeployEntry(L2_DEV_FEATURE_FLAGS, true, false, false, DevFeatures.L2CM);
    }

    /// @notice Returns true if the address is a supported predeploy on this chain.
    ///         Derived from the registry — no separate list to maintain.
    /// @param _addr             The address of the predeploy to check.
    /// @param _fork             The fork number for which support is being checked.
    /// @param _isCustomGasToken Whether the chain uses a custom gas token.
    /// @param _useInterop       Whether interop is enabled as a system configuration on this chain.
    /// @param _devFeatureBitmap Per-chain dev feature bitmap stored in L2DevFeatureFlags.
    ///
    /// @return True if the predeploy is supported on this fork with the given feature flags.
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
        // Non-proxied predeploys are always supported but not in the registry
        if (_addr == WETH || _addr == GOVERNANCE_TOKEN) return true;

        bool _isInteropDevFeatureEnabled =
            DevFeatures.isDevFeatureEnabled(_devFeatureBitmap, DevFeatures.OPTIMISM_PORTAL_INTEROP);

        PredeployEntry[] memory entries = _registry();
        for (uint256 i = 0; i < entries.length; i++) {
            if (entries[i].addr != _addr) continue;

            // Check interop condition
            if (entries[i].requiresInterop) {
                if (!(_fork >= uint256(Fork.INTEROP) && _isInteropDevFeatureEnabled && _useInterop)) return false;
            }

            // Check CGT condition
            if (entries[i].requiresCGT) {
                if (!_isCustomGasToken) return false;
            }

            // Check dev feature condition
            if (entries[i].requiredDevFeature != bytes32(0)) {
                if (!DevFeatures.isDevFeatureEnabled(_devFeatureBitmap, entries[i].requiredDevFeature)) return false;
            }

            return true;
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

    /// @notice Returns all proxied predeploys that should be upgraded by L2CM.
    ///         Derived from the registry — no separate list to maintain.
    /// @dev Excludes: WETH, GOVERNANCE_TOKEN (not proxied), legacy predeploys (not upgraded).
    function getUpgradeablePredeploys() internal pure returns (address[] memory predeploys_) {
        PredeployEntry[] memory entries = _registry();

        // First pass: count upgradeable entries
        uint256 count = 0;
        for (uint256 i = 0; i < entries.length; i++) {
            if (entries[i].upgradeable) count++;
        }

        // Second pass: populate the array
        predeploys_ = new address[](count);
        uint256 idx = 0;
        for (uint256 i = 0; i < entries.length; i++) {
            if (entries[i].upgradeable) {
                predeploys_[idx] = entries[i].addr;
                idx++;
            }
        }
    }

    /// @notice Returns the full registry of predeploy entries with their metadata.
    ///         Useful for tests and tooling that need to know predeploy conditions.
    function getPredeployRegistry() internal pure returns (PredeployEntry[] memory) {
        return _registry();
    }
}

// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IStandardBridge } from "interfaces/universal/IStandardBridge.sol";
import { IERC721Bridge } from "interfaces/universal/IERC721Bridge.sol";
import { IOptimismMintableERC20Factory } from "interfaces/universal/IOptimismMintableERC20Factory.sol";
import { IOptimismMintableERC721Factory } from "interfaces/L2/IOptimismMintableERC721Factory.sol";
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";
import { ILiquidityController } from "interfaces/L2/ILiquidityController.sol";
import { IFeeSplitter } from "interfaces/L2/IFeeSplitter.sol";
import { ISharesCalculator } from "interfaces/L2/ISharesCalculator.sol";
import { IL2CrossDomainMessenger } from "interfaces/L2/IL2CrossDomainMessenger.sol";
import { IL2StandardBridge } from "interfaces/L2/IL2StandardBridge.sol";
import { IL2ERC721Bridge } from "interfaces/L2/IL2ERC721Bridge.sol";
import { IL1Block } from "interfaces/L2/IL1Block.sol";

import { IL2ProxyAdmin } from "interfaces/L2/IL2ProxyAdmin.sol";

// Libraries
import { Features } from "src/libraries/Features.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { IL2DevFeatureFlags } from "interfaces/L2/IL2DevFeatureFlags.sol";
import { L2ContractsManagerTypes } from "src/libraries/L2ContractsManagerTypes.sol";
import { L2ContractsManagerUtils } from "src/libraries/L2ContractsManagerUtils.sol";

/// @title L2ContractsManager
/// @notice Manages the upgrade of the L2 predeploys.
contract L2ContractsManager is ISemver {
    /// @notice Thrown when the upgrade function is called outside of a DELEGATECALL context.
    error L2ContractsManager_OnlyDelegatecall();
    error L2ContractsManager_FeatureFlagMismatch();

    /// @notice The semantic version of the L2ContractsManager contract.
    /// @custom:semver 1.3.0
    string public constant version = "1.3.0";

    /// @notice The address of this contract. Used to enforce that the upgrade function is only
    ///         called via DELEGATECALL.
    address internal immutable THIS_L2CM;

    /// @notice Storage slot for OpenZeppelin v4 Initializable contracts.
    bytes32 internal constant INITIALIZABLE_SLOT_OZ_V4 = bytes32(0);

    /// @notice Storage slot for OpenZeppelin v5 Initializable contracts.
    /// @dev Equal to keccak256(abi.encode(uint256(keccak256("openzeppelin.storage.Initializable")) - 1)) &
    /// ~bytes32(uint256(0xff))
    bytes32 internal constant INITIALIZABLE_SLOT_OZ_V5 =
        0xf0c57e16840df040f15088dc2f81fe391c3923bec73e23a9662efc9c229c6a00;

    /// @notice The implementation address of the StorageSetter contract.
    address internal immutable STORAGE_SETTER_IMPL;

    /// @notice Each of the implementation addresses for each predeploy that exists in this upgrade.
    /// @notice GasPriceOracle implementation.
    address internal immutable GAS_PRICE_ORACLE_IMPL;
    /// @notice L2CrossDomainMessenger implementation.
    address internal immutable L2_CROSS_DOMAIN_MESSENGER_IMPL;
    /// @notice L2StandardBridge implementation.
    address internal immutable L2_STANDARD_BRIDGE_IMPL;
    /// @notice SequencerFeeWallet implementation.
    address internal immutable SEQUENCER_FEE_WALLET_IMPL;
    /// @notice OptimismMintableERC20Factory implementation.
    address internal immutable OPTIMISM_MINTABLE_ERC20_FACTORY_IMPL;
    /// @notice L2ERC721Bridge implementation.
    address internal immutable L2_ERC721_BRIDGE_IMPL;
    /// @notice L1Block implementation.
    address internal immutable L1_BLOCK_IMPL;
    /// @notice L1Block implementation for custom gas token networks.
    address internal immutable L1_BLOCK_CGT_IMPL;
    /// @notice L2ToL1MessagePasser implementation.
    address internal immutable L2_TO_L1_MESSAGE_PASSER_IMPL;
    /// @notice L2ToL1MessagePasser implementation for custom gas token networks.
    address internal immutable L2_TO_L1_MESSAGE_PASSER_CGT_IMPL;
    /// @notice OptimismMintableERC721Factory implementation.
    address internal immutable OPTIMISM_MINTABLE_ERC721_FACTORY_IMPL;
    /// @notice ProxyAdmin implementation.
    address internal immutable PROXY_ADMIN_IMPL;
    /// @notice BaseFeeVault implementation.
    address internal immutable BASE_FEE_VAULT_IMPL;
    /// @notice L1FeeVault implementation.
    address internal immutable L1_FEE_VAULT_IMPL;
    /// @notice OperatorFeeVault implementation.
    address internal immutable OPERATOR_FEE_VAULT_IMPL;
    /// @notice SchemaRegistry implementation.
    address internal immutable SCHEMA_REGISTRY_IMPL;
    /// @notice EAS implementation.
    address internal immutable EAS_IMPL;
    /// @notice CrossL2Inbox implementation.
    address internal immutable CROSS_L2_INBOX_IMPL;
    /// @notice L2ToL2CrossDomainMessenger implementation.
    address internal immutable L2_TO_L2_CROSS_DOMAIN_MESSENGER_IMPL;
    /// @notice SuperchainETHBridge implementation.
    address internal immutable SUPERCHAIN_ETH_BRIDGE_IMPL;
    /// @notice ETHLiquidity implementation.
    address internal immutable ETH_LIQUIDITY_IMPL;
    /// @notice NativeAssetLiquidity implementation.
    address internal immutable NATIVE_ASSET_LIQUIDITY_IMPL;
    /// @notice LiquidityController implementation.
    address internal immutable LIQUIDITY_CONTROLLER_IMPL;
    // TODO(#19600): Remove FEE_SPLITTER_IMPL as part of revenue sharing deprecation.
    /// @notice FeeSplitter implementation.
    address internal immutable FEE_SPLITTER_IMPL;
    /// @notice CONDITIONAL_DEPLOYER implementation.
    address internal immutable CONDITIONAL_DEPLOYER_IMPL;
    /// @notice L2DevFeatureFlags implementation.
    address internal immutable L2_DEV_FEATURE_FLAGS_IMPL;

    /// @notice Constructor for the L2ContractsManager contract.
    /// @param _implementations The implementation struct containing the new implementation addresses for the L2
    /// predeploys.
    constructor(L2ContractsManagerTypes.Implementations memory _implementations) {
        // Store the address of this contract for DELEGATECALL enforcement.
        THIS_L2CM = address(this);

        // Utility address for upgrading initializable contracts.
        STORAGE_SETTER_IMPL = _implementations.storageSetterImpl;
        // Predeploy implementations.
        L2_CROSS_DOMAIN_MESSENGER_IMPL = _implementations.l2CrossDomainMessengerImpl;
        GAS_PRICE_ORACLE_IMPL = _implementations.gasPriceOracleImpl;
        L2_STANDARD_BRIDGE_IMPL = _implementations.l2StandardBridgeImpl;
        SEQUENCER_FEE_WALLET_IMPL = _implementations.sequencerFeeWalletImpl;
        OPTIMISM_MINTABLE_ERC20_FACTORY_IMPL = _implementations.optimismMintableERC20FactoryImpl;
        L2_ERC721_BRIDGE_IMPL = _implementations.l2ERC721BridgeImpl;
        L1_BLOCK_IMPL = _implementations.l1BlockImpl;
        L1_BLOCK_CGT_IMPL = _implementations.l1BlockCGTImpl;
        L2_TO_L1_MESSAGE_PASSER_IMPL = _implementations.l2ToL1MessagePasserImpl;
        L2_TO_L1_MESSAGE_PASSER_CGT_IMPL = _implementations.l2ToL1MessagePasserCGTImpl;
        OPTIMISM_MINTABLE_ERC721_FACTORY_IMPL = _implementations.optimismMintableERC721FactoryImpl;
        PROXY_ADMIN_IMPL = _implementations.proxyAdminImpl;
        BASE_FEE_VAULT_IMPL = _implementations.baseFeeVaultImpl;
        L1_FEE_VAULT_IMPL = _implementations.l1FeeVaultImpl;
        OPERATOR_FEE_VAULT_IMPL = _implementations.operatorFeeVaultImpl;
        SCHEMA_REGISTRY_IMPL = _implementations.schemaRegistryImpl;
        EAS_IMPL = _implementations.easImpl;
        CROSS_L2_INBOX_IMPL = _implementations.crossL2InboxImpl;
        L2_TO_L2_CROSS_DOMAIN_MESSENGER_IMPL = _implementations.l2ToL2CrossDomainMessengerImpl;
        SUPERCHAIN_ETH_BRIDGE_IMPL = _implementations.superchainETHBridgeImpl;
        ETH_LIQUIDITY_IMPL = _implementations.ethLiquidityImpl;
        NATIVE_ASSET_LIQUIDITY_IMPL = _implementations.nativeAssetLiquidityImpl;
        LIQUIDITY_CONTROLLER_IMPL = _implementations.liquidityControllerImpl;
        // TODO(#19600): Remove FEE_SPLITTER_IMPL as part of revenue sharing deprecation.
        FEE_SPLITTER_IMPL = _implementations.feeSplitterImpl;
        CONDITIONAL_DEPLOYER_IMPL = _implementations.conditionalDeployerImpl;
        L2_DEV_FEATURE_FLAGS_IMPL = _implementations.l2DevFeatureFlagsImpl;
    }

    /// @notice Executes the upgrade for all predeploys.
    /// @dev This function MUST be called via DELEGATECALL from the L2ProxyAdmin.
    function upgrade() external {
        if (address(this) == THIS_L2CM) revert L2ContractsManager_OnlyDelegatecall();

        L2ContractsManagerTypes.FullConfig memory fullConfig = _loadFullConfig();
        _apply(fullConfig);
    }

    /// @notice Loads the full configuration for the L2 Predeploys.
    /// @return fullConfig_ The full configuration.
    function _loadFullConfig() internal view returns (L2ContractsManagerTypes.FullConfig memory fullConfig_) {
        // First we read the system customization and dev feature flags from the state.
        // Because the L2CM's upgrade function does not accept arguments, these values must be set from outside of the
        // Network Upgrade Transactions bundle. The expectation is that they will be set at the start of a
        // hard fork block, within the consensus client's code.

        // Read system customization flags from L1Block.
        // Uses the legacy isCustomGasToken() getter which has existed since custom gas token shipped.
        fullConfig_.isCustomGasToken = IL1Block(Predeploys.L1_BLOCK_ATTRIBUTES).isCustomGasToken();

        // Uses try/catch because isFeatureEnabled() may not exist on pre-upgrade L1Block contracts.
        // The INTEROP feature is enabled after genesis via a Network Upgrade Transaction (NUT) issued
        // by the consensus client at the start of the hard fork block.
        // eip150-safe
        try IL1Block(Predeploys.L1_BLOCK_ATTRIBUTES).isFeatureEnabled(Features.INTEROP) returns (bool isInterop_) {
            fullConfig_.isInterop = isInterop_;
        } catch {
            fullConfig_.isInterop = false;
        }
        // The INTEROP system customization can only be enabled if the dev feature is also enabled.
        // The dev feature gates whether interop code was deployed; the system customization controls activation.
        if (fullConfig_.isInterop && !_isDevFeatureEnabled(DevFeatures.OPTIMISM_PORTAL_INTEROP)) {
            revert L2ContractsManager_FeatureFlagMismatch();
        }

        // L2CrossDomainMessenger
        fullConfig_.crossDomainMessenger = L2ContractsManagerTypes.CrossDomainMessengerConfig({
            // TODO(#19468): Remove legacy getter after Karst upgrade.
            otherMessenger: ICrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).OTHER_MESSENGER()
        });

        // L2StandardBridge
        fullConfig_.standardBridge = L2ContractsManagerTypes.StandardBridgeConfig({
            // TODO(#19468): Remove legacy getter after Karst upgrade.
            otherBridge: IStandardBridge(payable(Predeploys.L2_STANDARD_BRIDGE)).OTHER_BRIDGE()
        });

        // L2ERC721Bridge
        fullConfig_.erc721Bridge = L2ContractsManagerTypes.ERC721BridgeConfig({
            // TODO(#19468): Remove legacy getter after Karst upgrade.
            otherBridge: IERC721Bridge(Predeploys.L2_ERC721_BRIDGE).OTHER_BRIDGE()
        });

        // OptimismMintableERC20Factory
        fullConfig_.mintableERC20Factory = L2ContractsManagerTypes.MintableERC20FactoryConfig({
            // TODO(#19468): Remove legacy getter after Karst upgrade.
            bridge: IOptimismMintableERC20Factory(Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY).BRIDGE()
        });

        // OptimismMintableERC721Factory
        fullConfig_.mintableERC721Factory = L2ContractsManagerTypes.MintableERC721FactoryConfig({
            // TODO(#19468): Remove legacy getter after Karst upgrade.
            bridge: IOptimismMintableERC721Factory(Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY).BRIDGE(),
            // TODO(#19468): Remove legacy getter after Karst upgrade.
            remoteChainID: IOptimismMintableERC721Factory(Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY).REMOTE_CHAIN_ID()
        });

        // SequencerFeeVault
        fullConfig_.sequencerFeeVault = L2ContractsManagerUtils.readFeeVaultConfig(Predeploys.SEQUENCER_FEE_WALLET);

        // BaseFeeVault
        fullConfig_.baseFeeVault = L2ContractsManagerUtils.readFeeVaultConfig(Predeploys.BASE_FEE_VAULT);

        // L1FeeVault
        fullConfig_.l1FeeVault = L2ContractsManagerUtils.readFeeVaultConfig(Predeploys.L1_FEE_VAULT);

        // OperatorFeeVault
        fullConfig_.operatorFeeVault = L2ContractsManagerUtils.readFeeVaultConfig(Predeploys.OPERATOR_FEE_VAULT);

        // LiquidityController
        if (fullConfig_.isCustomGasToken) {
            ILiquidityController liquidityController = ILiquidityController(Predeploys.LIQUIDITY_CONTROLLER);
            fullConfig_.liquidityController = L2ContractsManagerTypes.LiquidityControllerConfig({
                owner: liquidityController.owner(),
                gasPayingTokenName: liquidityController.gasPayingTokenName(),
                gasPayingTokenSymbol: liquidityController.gasPayingTokenSymbol()
            });
        }

        // TODO(#19600): Remove FeeSplitter loading config as part of revenue sharing deprecation.
        // FeeSplitter
        ISharesCalculator sharesCalculator;

        // FeeSplitter may not be deployed at the predeploy address, since fee vaults on
        // earlier contract versions only support L1 withdrawals. We initialize with sharesCalculator as address(0)
        // to preserve this behavior.
        // eip150-safe
        try IFeeSplitter(payable(Predeploys.FEE_SPLITTER)).sharesCalculator() returns (
            ISharesCalculator sharesCalculator_
        ) {
            sharesCalculator = sharesCalculator_;
        } catch {
            sharesCalculator = ISharesCalculator(address(0));
        }

        fullConfig_.feeSplitter = L2ContractsManagerTypes.FeeSplitterConfig({ sharesCalculator: sharesCalculator });
    }

    /// @notice Upgrades each of the predeploys to its corresponding new implementation. Applies the appropriate
    ///         configuration to each predeploy.
    /// @param _config The full configuration for the L2 Predeploys.
    function _apply(L2ContractsManagerTypes.FullConfig memory _config) internal {
        // Initializable predeploys.

        // L2CrossDomainMessenger
        L2ContractsManagerUtils.upgradeToAndCall(
            Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            L2_CROSS_DOMAIN_MESSENGER_IMPL,
            STORAGE_SETTER_IMPL,
            abi.encodeCall(IL2CrossDomainMessenger.initialize, (_config.crossDomainMessenger.otherMessenger)),
            INITIALIZABLE_SLOT_OZ_V4,
            20 // Account for CrossDomainMessengerLegacySpacer0
        );

        // L2StandardBridge
        L2ContractsManagerUtils.upgradeToAndCall(
            Predeploys.L2_STANDARD_BRIDGE,
            L2_STANDARD_BRIDGE_IMPL,
            STORAGE_SETTER_IMPL,
            abi.encodeCall(IL2StandardBridge.initialize, (_config.standardBridge.otherBridge)),
            INITIALIZABLE_SLOT_OZ_V4,
            0
        );

        // L2ERC721Bridge
        L2ContractsManagerUtils.upgradeToAndCall(
            Predeploys.L2_ERC721_BRIDGE,
            L2_ERC721_BRIDGE_IMPL,
            STORAGE_SETTER_IMPL,
            abi.encodeCall(IL2ERC721Bridge.initialize, payable(address(_config.erc721Bridge.otherBridge))),
            INITIALIZABLE_SLOT_OZ_V4,
            0
        );

        // OptimismMintableERC20Factory
        L2ContractsManagerUtils.upgradeToAndCall(
            Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY,
            OPTIMISM_MINTABLE_ERC20_FACTORY_IMPL,
            STORAGE_SETTER_IMPL,
            abi.encodeCall(IOptimismMintableERC20Factory.initialize, (_config.mintableERC20Factory.bridge)),
            INITIALIZABLE_SLOT_OZ_V4,
            0
        );

        // OptimismMintableERC721Factory
        L2ContractsManagerUtils.upgradeToAndCall(
            Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY,
            OPTIMISM_MINTABLE_ERC721_FACTORY_IMPL,
            STORAGE_SETTER_IMPL,
            abi.encodeCall(
                IOptimismMintableERC721Factory.initialize,
                (_config.mintableERC721Factory.bridge, _config.mintableERC721Factory.remoteChainID)
            ),
            bytes32(uint256(1)), // Initializable storage is at slot 1 due to mapping at slot 0
            0
        );

        // LiquidityController (only on custom gas token networks)
        if (_config.isCustomGasToken) {
            L2ContractsManagerUtils.upgradeToAndCall(
                Predeploys.LIQUIDITY_CONTROLLER,
                LIQUIDITY_CONTROLLER_IMPL,
                STORAGE_SETTER_IMPL,
                abi.encodeCall(
                    ILiquidityController.initialize,
                    (
                        _config.liquidityController.owner,
                        _config.liquidityController.gasPayingTokenName,
                        _config.liquidityController.gasPayingTokenSymbol
                    )
                ),
                INITIALIZABLE_SLOT_OZ_V4,
                0
            );

            // NativeAssetLiquidity
            L2ContractsManagerUtils.upgradeTo(Predeploys.NATIVE_ASSET_LIQUIDITY, NATIVE_ASSET_LIQUIDITY_IMPL);
        }

        // TODO(#19600): Remove FeeSplitter upgrade as part of revenue sharing deprecation.
        // FeeSplitter
        L2ContractsManagerUtils.upgradeToAndCall(
            Predeploys.FEE_SPLITTER,
            FEE_SPLITTER_IMPL,
            STORAGE_SETTER_IMPL,
            abi.encodeCall(IFeeSplitter.initialize, (ISharesCalculator(_config.feeSplitter.sharesCalculator))),
            INITIALIZABLE_SLOT_OZ_V4,
            0
        );

        // TODO(#19600): Remove withdrawalNetwork arg from fee vault initializers as part of revenue sharing
        // deprecation.
        // SequencerFeeVault
        L2ContractsManagerUtils.upgradeToAndCall(
            Predeploys.SEQUENCER_FEE_WALLET,
            SEQUENCER_FEE_WALLET_IMPL,
            STORAGE_SETTER_IMPL,
            abi.encodeCall(
                IFeeVault.initialize,
                (
                    _config.sequencerFeeVault.recipient,
                    _config.sequencerFeeVault.minWithdrawalAmount,
                    _config.sequencerFeeVault.withdrawalNetwork
                )
            ),
            INITIALIZABLE_SLOT_OZ_V5,
            0
        );

        // BaseFeeVault
        L2ContractsManagerUtils.upgradeToAndCall(
            Predeploys.BASE_FEE_VAULT,
            BASE_FEE_VAULT_IMPL,
            STORAGE_SETTER_IMPL,
            abi.encodeCall(
                IFeeVault.initialize,
                (
                    _config.baseFeeVault.recipient,
                    _config.baseFeeVault.minWithdrawalAmount,
                    _config.baseFeeVault.withdrawalNetwork
                )
            ),
            INITIALIZABLE_SLOT_OZ_V5,
            0
        );

        // L1FeeVault
        L2ContractsManagerUtils.upgradeToAndCall(
            Predeploys.L1_FEE_VAULT,
            L1_FEE_VAULT_IMPL,
            STORAGE_SETTER_IMPL,
            abi.encodeCall(
                IFeeVault.initialize,
                (
                    _config.l1FeeVault.recipient,
                    _config.l1FeeVault.minWithdrawalAmount,
                    _config.l1FeeVault.withdrawalNetwork
                )
            ),
            INITIALIZABLE_SLOT_OZ_V5,
            0
        );

        // OperatorFeeVault
        L2ContractsManagerUtils.upgradeToAndCall(
            Predeploys.OPERATOR_FEE_VAULT,
            OPERATOR_FEE_VAULT_IMPL,
            STORAGE_SETTER_IMPL,
            abi.encodeCall(
                IFeeVault.initialize,
                (
                    _config.operatorFeeVault.recipient,
                    _config.operatorFeeVault.minWithdrawalAmount,
                    _config.operatorFeeVault.withdrawalNetwork
                )
            ),
            INITIALIZABLE_SLOT_OZ_V5,
            0
        );

        // Non-initializable predeploys.
        L2ContractsManagerUtils.upgradeTo(Predeploys.GAS_PRICE_ORACLE, GAS_PRICE_ORACLE_IMPL);
        // L1BlockAttributes and L2ToL1MessagePasser have different implementations for custom gas token networks.
        L2ContractsManagerUtils.upgradeTo(
            Predeploys.L1_BLOCK_ATTRIBUTES, _config.isCustomGasToken ? L1_BLOCK_CGT_IMPL : L1_BLOCK_IMPL
        );
        // TODO(#19468): Remove this migration step after Karst. Post-Karst, the feature
        // mapping will already be populated from the upgrade, making this call unnecessary.
        // After upgrading L1Block to the CGT impl, populate the feature mapping so that
        // isCustomGasToken() continues to return true. The new impl reads from the mapping
        // rather than the legacy storage slot.
        if (_config.isCustomGasToken) {
            IL1Block(Predeploys.L1_BLOCK_ATTRIBUTES).setFeature(Features.CUSTOM_GAS_TOKEN);
        }
        L2ContractsManagerUtils.upgradeTo(
            Predeploys.L2_TO_L1_MESSAGE_PASSER,
            _config.isCustomGasToken ? L2_TO_L1_MESSAGE_PASSER_CGT_IMPL : L2_TO_L1_MESSAGE_PASSER_IMPL
        );
        L2ContractsManagerUtils.upgradeTo(Predeploys.PROXY_ADMIN, PROXY_ADMIN_IMPL);
        L2ContractsManagerUtils.upgradeTo(Predeploys.L2_DEV_FEATURE_FLAGS, L2_DEV_FEATURE_FLAGS_IMPL);

        // Interop predeploys are gated behind the OPTIMISM_PORTAL_INTEROP dev feature flag.
        if (_config.isInterop) {
            L2ContractsManagerUtils.upgradeTo(Predeploys.CROSS_L2_INBOX, CROSS_L2_INBOX_IMPL);
            L2ContractsManagerUtils.upgradeTo(
                Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER, L2_TO_L2_CROSS_DOMAIN_MESSENGER_IMPL
            );
            L2ContractsManagerUtils.upgradeTo(Predeploys.SUPERCHAIN_ETH_BRIDGE, SUPERCHAIN_ETH_BRIDGE_IMPL);
            L2ContractsManagerUtils.upgradeTo(Predeploys.ETH_LIQUIDITY, ETH_LIQUIDITY_IMPL);
        }
        L2ContractsManagerUtils.upgradeTo(Predeploys.SCHEMA_REGISTRY, SCHEMA_REGISTRY_IMPL);
        L2ContractsManagerUtils.upgradeTo(Predeploys.EAS, EAS_IMPL);
        L2ContractsManagerUtils.upgradeTo(Predeploys.CONDITIONAL_DEPLOYER, CONDITIONAL_DEPLOYER_IMPL);
    }

    /// @notice Checks if a development feature is enabled by reading from the L2DevFeatureFlags predeploy.
    ///         If the L2DevFeatureFlags Predeploy is not available on-chain, i.e. it has no implementation,
    ///         it defaults to false.
    /// @param _feature The feature to check.
    /// @return True if the L2DevFeatureFlags is available and _feature is enabled, false otherwise.
    function _isDevFeatureEnabled(bytes32 _feature) internal view returns (bool) {
        address flagsImpl =
            IL2ProxyAdmin(Predeploys.PROXY_ADMIN).getProxyImplementation(Predeploys.L2_DEV_FEATURE_FLAGS);
        if (flagsImpl.code.length == 0) return false;
        return IL2DevFeatureFlags(Predeploys.L2_DEV_FEATURE_FLAGS).isDevFeatureEnabled(_feature);
    }

    /// @notice Returns the implementation addresses for each predeploy upgraded by the L2ContractsManager.
    /// @return implementations_ The implementation addresses for each predeploy upgraded by the L2ContractsManager.
    function getImplementations()
        external
        view
        returns (L2ContractsManagerTypes.Implementations memory implementations_)
    {
        implementations_.storageSetterImpl = STORAGE_SETTER_IMPL;
        implementations_.l2CrossDomainMessengerImpl = L2_CROSS_DOMAIN_MESSENGER_IMPL;
        implementations_.gasPriceOracleImpl = GAS_PRICE_ORACLE_IMPL;
        implementations_.l2StandardBridgeImpl = L2_STANDARD_BRIDGE_IMPL;
        implementations_.sequencerFeeWalletImpl = SEQUENCER_FEE_WALLET_IMPL;
        implementations_.optimismMintableERC20FactoryImpl = OPTIMISM_MINTABLE_ERC20_FACTORY_IMPL;
        implementations_.l2ERC721BridgeImpl = L2_ERC721_BRIDGE_IMPL;
        implementations_.l1BlockImpl = L1_BLOCK_IMPL;
        implementations_.l1BlockCGTImpl = L1_BLOCK_CGT_IMPL;
        implementations_.l2ToL1MessagePasserImpl = L2_TO_L1_MESSAGE_PASSER_IMPL;
        implementations_.l2ToL1MessagePasserCGTImpl = L2_TO_L1_MESSAGE_PASSER_CGT_IMPL;
        implementations_.optimismMintableERC721FactoryImpl = OPTIMISM_MINTABLE_ERC721_FACTORY_IMPL;
        implementations_.proxyAdminImpl = PROXY_ADMIN_IMPL;
        implementations_.baseFeeVaultImpl = BASE_FEE_VAULT_IMPL;
        implementations_.l1FeeVaultImpl = L1_FEE_VAULT_IMPL;
        implementations_.operatorFeeVaultImpl = OPERATOR_FEE_VAULT_IMPL;
        implementations_.schemaRegistryImpl = SCHEMA_REGISTRY_IMPL;
        implementations_.easImpl = EAS_IMPL;
        implementations_.crossL2InboxImpl = CROSS_L2_INBOX_IMPL;
        implementations_.l2ToL2CrossDomainMessengerImpl = L2_TO_L2_CROSS_DOMAIN_MESSENGER_IMPL;
        implementations_.superchainETHBridgeImpl = SUPERCHAIN_ETH_BRIDGE_IMPL;
        implementations_.ethLiquidityImpl = ETH_LIQUIDITY_IMPL;
        implementations_.nativeAssetLiquidityImpl = NATIVE_ASSET_LIQUIDITY_IMPL;
        implementations_.liquidityControllerImpl = LIQUIDITY_CONTROLLER_IMPL;
        implementations_.feeSplitterImpl = FEE_SPLITTER_IMPL;
        implementations_.conditionalDeployerImpl = CONDITIONAL_DEPLOYER_IMPL;
        implementations_.l2DevFeatureFlagsImpl = L2_DEV_FEATURE_FLAGS_IMPL;
    }
}

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

    /// @notice Thrown when the fullConfig input and dev feature flags are not compatible.
    error L2ContractsManager_FeatureFlagMismatch();

    /// @notice The semantic version of the L2ContractsManager contract.
    /// @custom:semver 1.9.1
    string public constant version = "1.9.1";

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
    /// @notice CONDITIONAL_DEPLOYER implementation.
    address internal immutable CONDITIONAL_DEPLOYER_IMPL;
    /// @notice L2DevFeatureFlags implementation.
    address internal immutable L2_DEV_FEATURE_FLAGS_IMPL;

    /// @notice Constructor for the L2ContractsManager contract.
    /// @dev Loads the implementation addresses for all predeploys.
    /// @param _implementations Array of name + implementation records for all predeploys.
    constructor(L2ContractsManagerTypes.ImplRecord[] memory _implementations) {
        // Store the address of this contract for DELEGATECALL enforcement.
        THIS_L2CM = address(this);

        // Utility address for upgrading initializable contracts.
        STORAGE_SETTER_IMPL = L2ContractsManagerUtils.findImpl(_implementations, "StorageSetter");
        // Predeploy implementations.
        L2_CROSS_DOMAIN_MESSENGER_IMPL = L2ContractsManagerUtils.findImpl(_implementations, "L2CrossDomainMessenger");
        GAS_PRICE_ORACLE_IMPL = L2ContractsManagerUtils.findImpl(_implementations, "GasPriceOracle");
        L2_STANDARD_BRIDGE_IMPL = L2ContractsManagerUtils.findImpl(_implementations, "L2StandardBridge");
        SEQUENCER_FEE_WALLET_IMPL = L2ContractsManagerUtils.findImpl(_implementations, "SequencerFeeVault");
        OPTIMISM_MINTABLE_ERC20_FACTORY_IMPL =
            L2ContractsManagerUtils.findImpl(_implementations, "OptimismMintableERC20Factory");
        L2_ERC721_BRIDGE_IMPL = L2ContractsManagerUtils.findImpl(_implementations, "L2ERC721Bridge");
        L1_BLOCK_IMPL = L2ContractsManagerUtils.findImpl(_implementations, "L1Block");
        L1_BLOCK_CGT_IMPL = L2ContractsManagerUtils.findImpl(_implementations, "L1BlockCGT");
        L2_TO_L1_MESSAGE_PASSER_IMPL = L2ContractsManagerUtils.findImpl(_implementations, "L2ToL1MessagePasser");
        L2_TO_L1_MESSAGE_PASSER_CGT_IMPL = L2ContractsManagerUtils.findImpl(_implementations, "L2ToL1MessagePasserCGT");
        OPTIMISM_MINTABLE_ERC721_FACTORY_IMPL =
            L2ContractsManagerUtils.findImpl(_implementations, "OptimismMintableERC721Factory");
        PROXY_ADMIN_IMPL = L2ContractsManagerUtils.findImpl(_implementations, "L2ProxyAdmin");
        BASE_FEE_VAULT_IMPL = L2ContractsManagerUtils.findImpl(_implementations, "BaseFeeVault");
        L1_FEE_VAULT_IMPL = L2ContractsManagerUtils.findImpl(_implementations, "L1FeeVault");
        OPERATOR_FEE_VAULT_IMPL = L2ContractsManagerUtils.findImpl(_implementations, "OperatorFeeVault");
        SCHEMA_REGISTRY_IMPL = L2ContractsManagerUtils.findImpl(_implementations, "SchemaRegistry");
        EAS_IMPL = L2ContractsManagerUtils.findImpl(_implementations, "EAS");
        CROSS_L2_INBOX_IMPL = L2ContractsManagerUtils.findImpl(_implementations, "CrossL2Inbox");
        L2_TO_L2_CROSS_DOMAIN_MESSENGER_IMPL =
            L2ContractsManagerUtils.findImpl(_implementations, "L2ToL2CrossDomainMessenger");
        SUPERCHAIN_ETH_BRIDGE_IMPL = L2ContractsManagerUtils.findImpl(_implementations, "SuperchainETHBridge");
        ETH_LIQUIDITY_IMPL = L2ContractsManagerUtils.findImpl(_implementations, "ETHLiquidity");
        NATIVE_ASSET_LIQUIDITY_IMPL = L2ContractsManagerUtils.findImpl(_implementations, "NativeAssetLiquidity");
        LIQUIDITY_CONTROLLER_IMPL = L2ContractsManagerUtils.findImpl(_implementations, "LiquidityController");
        CONDITIONAL_DEPLOYER_IMPL = L2ContractsManagerUtils.findImpl(_implementations, "ConditionalDeployer");
        L2_DEV_FEATURE_FLAGS_IMPL = L2ContractsManagerUtils.findImpl(_implementations, "L2DevFeatureFlags");
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
            otherMessenger: ICrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).otherMessenger()
        });

        // L2StandardBridge
        fullConfig_.standardBridge = L2ContractsManagerTypes.StandardBridgeConfig({
            otherBridge: IStandardBridge(payable(Predeploys.L2_STANDARD_BRIDGE)).otherBridge()
        });

        // L2ERC721Bridge
        fullConfig_.erc721Bridge = L2ContractsManagerTypes.ERC721BridgeConfig({
            otherBridge: IERC721Bridge(Predeploys.L2_ERC721_BRIDGE).otherBridge()
        });

        // OptimismMintableERC20Factory
        fullConfig_.mintableERC20Factory = L2ContractsManagerTypes.MintableERC20FactoryConfig({
            bridge: IOptimismMintableERC20Factory(Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY).bridge()
        });

        // OptimismMintableERC721Factory
        fullConfig_.mintableERC721Factory = L2ContractsManagerTypes.MintableERC721FactoryConfig({
            bridge: IOptimismMintableERC721Factory(Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY).bridge(),
            remoteChainID: IOptimismMintableERC721Factory(Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY).remoteChainID()
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
        }

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
        L2ContractsManagerUtils.upgradeTo(
            Predeploys.L2_TO_L1_MESSAGE_PASSER,
            _config.isCustomGasToken ? L2_TO_L1_MESSAGE_PASSER_CGT_IMPL : L2_TO_L1_MESSAGE_PASSER_IMPL
        );
        L2ContractsManagerUtils.upgradeTo(Predeploys.PROXY_ADMIN, PROXY_ADMIN_IMPL);
        L2ContractsManagerUtils.upgradeTo(Predeploys.L2_DEV_FEATURE_FLAGS, L2_DEV_FEATURE_FLAGS_IMPL);
        if (_config.isCustomGasToken) {
            L2ContractsManagerUtils.upgradeTo(Predeploys.NATIVE_ASSET_LIQUIDITY, NATIVE_ASSET_LIQUIDITY_IMPL);
        }

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
        returns (L2ContractsManagerTypes.ImplRecord[] memory implementations_)
    {
        implementations_ = new L2ContractsManagerTypes.ImplRecord[](26);
        implementations_[0] = L2ContractsManagerTypes.ImplRecord({ name: "StorageSetter", impl: STORAGE_SETTER_IMPL });
        implementations_[1] =
            L2ContractsManagerTypes.ImplRecord({ name: "L2CrossDomainMessenger", impl: L2_CROSS_DOMAIN_MESSENGER_IMPL });
        implementations_[2] =
            L2ContractsManagerTypes.ImplRecord({ name: "GasPriceOracle", impl: GAS_PRICE_ORACLE_IMPL });
        implementations_[3] =
            L2ContractsManagerTypes.ImplRecord({ name: "L2StandardBridge", impl: L2_STANDARD_BRIDGE_IMPL });
        implementations_[4] =
            L2ContractsManagerTypes.ImplRecord({ name: "SequencerFeeVault", impl: SEQUENCER_FEE_WALLET_IMPL });
        implementations_[5] = L2ContractsManagerTypes.ImplRecord({
            name: "OptimismMintableERC20Factory",
            impl: OPTIMISM_MINTABLE_ERC20_FACTORY_IMPL
        });
        implementations_[6] =
            L2ContractsManagerTypes.ImplRecord({ name: "L2ERC721Bridge", impl: L2_ERC721_BRIDGE_IMPL });
        implementations_[7] = L2ContractsManagerTypes.ImplRecord({ name: "L1Block", impl: L1_BLOCK_IMPL });
        implementations_[8] = L2ContractsManagerTypes.ImplRecord({ name: "L1BlockCGT", impl: L1_BLOCK_CGT_IMPL });
        implementations_[9] =
            L2ContractsManagerTypes.ImplRecord({ name: "L2ToL1MessagePasser", impl: L2_TO_L1_MESSAGE_PASSER_IMPL });
        implementations_[10] = L2ContractsManagerTypes.ImplRecord({
            name: "L2ToL1MessagePasserCGT",
            impl: L2_TO_L1_MESSAGE_PASSER_CGT_IMPL
        });
        implementations_[11] = L2ContractsManagerTypes.ImplRecord({
            name: "OptimismMintableERC721Factory",
            impl: OPTIMISM_MINTABLE_ERC721_FACTORY_IMPL
        });
        implementations_[12] = L2ContractsManagerTypes.ImplRecord({ name: "L2ProxyAdmin", impl: PROXY_ADMIN_IMPL });
        implementations_[13] = L2ContractsManagerTypes.ImplRecord({ name: "BaseFeeVault", impl: BASE_FEE_VAULT_IMPL });
        implementations_[14] = L2ContractsManagerTypes.ImplRecord({ name: "L1FeeVault", impl: L1_FEE_VAULT_IMPL });
        implementations_[15] =
            L2ContractsManagerTypes.ImplRecord({ name: "OperatorFeeVault", impl: OPERATOR_FEE_VAULT_IMPL });
        implementations_[16] =
            L2ContractsManagerTypes.ImplRecord({ name: "SchemaRegistry", impl: SCHEMA_REGISTRY_IMPL });
        implementations_[17] = L2ContractsManagerTypes.ImplRecord({ name: "EAS", impl: EAS_IMPL });
        implementations_[18] = L2ContractsManagerTypes.ImplRecord({ name: "CrossL2Inbox", impl: CROSS_L2_INBOX_IMPL });
        implementations_[19] = L2ContractsManagerTypes.ImplRecord({
            name: "L2ToL2CrossDomainMessenger",
            impl: L2_TO_L2_CROSS_DOMAIN_MESSENGER_IMPL
        });
        implementations_[20] =
            L2ContractsManagerTypes.ImplRecord({ name: "SuperchainETHBridge", impl: SUPERCHAIN_ETH_BRIDGE_IMPL });
        implementations_[21] = L2ContractsManagerTypes.ImplRecord({ name: "ETHLiquidity", impl: ETH_LIQUIDITY_IMPL });
        implementations_[22] =
            L2ContractsManagerTypes.ImplRecord({ name: "NativeAssetLiquidity", impl: NATIVE_ASSET_LIQUIDITY_IMPL });
        implementations_[23] =
            L2ContractsManagerTypes.ImplRecord({ name: "LiquidityController", impl: LIQUIDITY_CONTROLLER_IMPL });
        implementations_[24] =
            L2ContractsManagerTypes.ImplRecord({ name: "ConditionalDeployer", impl: CONDITIONAL_DEPLOYER_IMPL });
        implementations_[25] =
            L2ContractsManagerTypes.ImplRecord({ name: "L2DevFeatureFlags", impl: L2_DEV_FEATURE_FLAGS_IMPL });
    }
}

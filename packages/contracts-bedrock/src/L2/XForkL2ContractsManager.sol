// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";
import { IStorageSetter } from "interfaces/universal/IStorageSetter.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IStandardBridge } from "interfaces/universal/IStandardBridge.sol";
import { IERC721Bridge } from "interfaces/universal/IERC721Bridge.sol";
import { IOptimismMintableERC20Factory } from "interfaces/universal/IOptimismMintableERC20Factory.sol";
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";
import { ILiquidityController } from "interfaces/L2/ILiquidityController.sol";
import { IFeeSplitter } from "interfaces/L2/IFeeSplitter.sol";
import { ISharesCalculator } from "interfaces/L2/ISharesCalculator.sol";
import { IL2CrossDomainMessenger } from "interfaces/L2/IL2CrossDomainMessenger.sol";
import { IL2StandardBridge } from "interfaces/L2/IL2StandardBridge.sol";
import { IL2ERC721Bridge } from "interfaces/L2/IL2ERC721Bridge.sol";
import { IL1Block } from "interfaces/L2/IL1Block.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Types } from "src/libraries/Types.sol";
import { XForkL2CMTypes } from "src/libraries/XForkL2CMTypes.sol";

/// @title XForkL2ContractsManager
/// @notice Manages the upgrade of the L2 predeploys for the XFork upgrade.
contract XForkL2ContractsManager is ISemver {
    /// @notice Thrown when the upgrade function is called outside of a DELEGATECALL context.
    error XForkL2ContractsManager_OnlyDelegatecall();

    /// @notice The semantic version of the L2ContractsManager contract.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

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
    /// @notice L1BlockAttributes implementation.
    address internal immutable L1_BLOCK_ATTRIBUTES_IMPL;
    /// @notice L1BlockAttributes implementation for custom gas token networks.
    address internal immutable L1_BLOCK_ATTRIBUTES_CGT_IMPL;
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
    /// @notice OptimismSuperchainERC20Factory implementation.
    address internal immutable OPTIMISM_SUPERCHAIN_ERC20_FACTORY_IMPL;
    /// @notice OptimismSuperchainERC20Beacon implementation.
    address internal immutable OPTIMISM_SUPERCHAIN_ERC20_BEACON_IMPL;
    /// @notice SuperchainTokenBridge implementation.
    address internal immutable SUPERCHAIN_TOKEN_BRIDGE_IMPL;
    /// @notice NativeAssetLiquidity implementation.
    address internal immutable NATIVE_ASSET_LIQUIDITY_IMPL;
    /// @notice LiquidityController implementation.
    address internal immutable LIQUIDITY_CONTROLLER_IMPL;
    /// @notice FeeSplitter implementation.
    address internal immutable FEE_SPLITTER_IMPL;

    constructor(XForkL2CMTypes.Implementations memory _implementations) {
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
        L1_BLOCK_ATTRIBUTES_IMPL = _implementations.l1BlockAttributesImpl;
        L1_BLOCK_ATTRIBUTES_CGT_IMPL = _implementations.l1BlockAttributesCGTImpl;
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
        OPTIMISM_SUPERCHAIN_ERC20_FACTORY_IMPL = _implementations.optimismSuperchainERC20FactoryImpl;
        OPTIMISM_SUPERCHAIN_ERC20_BEACON_IMPL = _implementations.optimismSuperchainERC20BeaconImpl;
        SUPERCHAIN_TOKEN_BRIDGE_IMPL = _implementations.superchainTokenBridgeImpl;
        NATIVE_ASSET_LIQUIDITY_IMPL = _implementations.nativeAssetLiquidityImpl;
        LIQUIDITY_CONTROLLER_IMPL = _implementations.liquidityControllerImpl;
        FEE_SPLITTER_IMPL = _implementations.feeSplitterImpl;
    }

    /// @notice Executes the upgrade for all predeploys.
    /// @dev This function MUST be called via DELEGATECALL from the L2ProxyAdmin.
    function upgrade() external {
        if (address(this) == THIS_L2CM) revert XForkL2ContractsManager_OnlyDelegatecall();

        XForkL2CMTypes.FullConfig memory fullConfig = _fullConfig();
        _apply(fullConfig);
    }

    /// @notice Loads the full configuration for the L2 Predeploys.
    /// @return fullConfig_ The full configuration.
    function _fullConfig() internal view returns (XForkL2CMTypes.FullConfig memory fullConfig_) {
        bool isCustomGasToken = IL1Block(Predeploys.L1_BLOCK_ATTRIBUTES).isCustomGasToken();

        // L2CrossDomainMessenger
        fullConfig_.crossDomainMessenger = XForkL2CMTypes.CrossDomainMessengerConfig({
            otherMessenger: address(ICrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).otherMessenger())
        });

        // L2StandardBridge
        fullConfig_.standardBridge = XForkL2CMTypes.StandardBridgeConfig({
            otherBridge: address(IStandardBridge(payable(Predeploys.L2_STANDARD_BRIDGE)).otherBridge())
        });

        // L2ERC721Bridge
        fullConfig_.erc721Bridge = XForkL2CMTypes.ERC721BridgeConfig({
            otherBridge: address(IERC721Bridge(Predeploys.L2_ERC721_BRIDGE).otherBridge())
        });

        // OptimismMintableERC20Factory
        fullConfig_.mintableERC20Factory = XForkL2CMTypes.MintableERC20FactoryConfig({
            bridge: IOptimismMintableERC20Factory(Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY).bridge()
        });

        // SequencerFeeVault
        fullConfig_.sequencerFeeVault = _readFeeVaultConfig(Predeploys.SEQUENCER_FEE_WALLET);

        // BaseFeeVault
        fullConfig_.baseFeeVault = _readFeeVaultConfig(Predeploys.BASE_FEE_VAULT);

        // L1FeeVault
        fullConfig_.l1FeeVault = _readFeeVaultConfig(Predeploys.L1_FEE_VAULT);

        // OperatorFeeVault
        fullConfig_.operatorFeeVault = _readFeeVaultConfig(Predeploys.OPERATOR_FEE_VAULT);

        // LiquidityController
        if (isCustomGasToken) {
            ILiquidityController liquidityController = ILiquidityController(Predeploys.LIQUIDITY_CONTROLLER);
            fullConfig_.liquidityController = XForkL2CMTypes.LiquidityControllerConfig({
                owner: liquidityController.owner(),
                gasPayingTokenName: liquidityController.gasPayingTokenName(),
                gasPayingTokenSymbol: liquidityController.gasPayingTokenSymbol()
            });
        }

        // FeeSplitter
        fullConfig_.feeSplitter = XForkL2CMTypes.FeeSplitterConfig({
            sharesCalculator: address(IFeeSplitter(payable(Predeploys.FEE_SPLITTER)).sharesCalculator())
        });
    }

    /// @notice Reads the configuration from a FeeVault predeploy.
    /// @param _feeVault The address of the FeeVault predeploy.
    /// @return config_ The FeeVault configuration.
    function _readFeeVaultConfig(address _feeVault)
        internal
        view
        returns (XForkL2CMTypes.FeeVaultConfig memory config_)
    {
        IFeeVault feeVault = IFeeVault(payable(_feeVault));
        config_ = XForkL2CMTypes.FeeVaultConfig({
            recipient: feeVault.recipient(),
            minWithdrawalAmount: feeVault.minWithdrawalAmount(),
            withdrawalNetwork: feeVault.withdrawalNetwork()
        });
    }

    /// @notice Upgrades each of the predeploys to its corresponding new implementation. Applies the appropriate
    ///         configuration to each predeploy.
    /// @param _config The full configuration for the L2 Predeploys.
    function _apply(XForkL2CMTypes.FullConfig memory _config) internal {
        bool isCustomGasToken = IL1Block(Predeploys.L1_BLOCK_ATTRIBUTES).isCustomGasToken();

        // Initializable predeploys.

        // L2CrossDomainMessenger
        _upgradeToAndCall(
            Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            L2_CROSS_DOMAIN_MESSENGER_IMPL,
            abi.encodeCall(
                IL2CrossDomainMessenger.initialize, (ICrossDomainMessenger(_config.crossDomainMessenger.otherMessenger))
            ),
            INITIALIZABLE_SLOT_OZ_V4,
            20 // Account for CrossDomainMessengerLegacySpacer0
        );

        // L2StandardBridge
        _upgradeToAndCall(
            Predeploys.L2_STANDARD_BRIDGE,
            L2_STANDARD_BRIDGE_IMPL,
            abi.encodeCall(IL2StandardBridge.initialize, (IStandardBridge(payable(_config.standardBridge.otherBridge)))),
            INITIALIZABLE_SLOT_OZ_V4,
            0
        );

        // L2ERC721Bridge
        _upgradeToAndCall(
            Predeploys.L2_ERC721_BRIDGE,
            L2_ERC721_BRIDGE_IMPL,
            abi.encodeCall(IL2ERC721Bridge.initialize, (payable(_config.erc721Bridge.otherBridge))),
            INITIALIZABLE_SLOT_OZ_V4,
            0
        );

        // OptimismMintableERC20Factory
        _upgradeToAndCall(
            Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY,
            OPTIMISM_MINTABLE_ERC20_FACTORY_IMPL,
            abi.encodeCall(IOptimismMintableERC20Factory.initialize, (_config.mintableERC20Factory.bridge)),
            INITIALIZABLE_SLOT_OZ_V4,
            0
        );

        // LiquidityController (only on custom gas token networks)
        if (isCustomGasToken) {
            _upgradeToAndCall(
                Predeploys.LIQUIDITY_CONTROLLER,
                LIQUIDITY_CONTROLLER_IMPL,
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

        // FeeSplitter
        _upgradeToAndCall(
            Predeploys.FEE_SPLITTER,
            FEE_SPLITTER_IMPL,
            abi.encodeCall(IFeeSplitter.initialize, (ISharesCalculator(_config.feeSplitter.sharesCalculator))),
            INITIALIZABLE_SLOT_OZ_V4,
            0
        );

        // SequencerFeeVault
        _upgradeToAndCall(
            Predeploys.SEQUENCER_FEE_WALLET,
            SEQUENCER_FEE_WALLET_IMPL,
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
        _upgradeToAndCall(
            Predeploys.BASE_FEE_VAULT,
            BASE_FEE_VAULT_IMPL,
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
        _upgradeToAndCall(
            Predeploys.L1_FEE_VAULT,
            L1_FEE_VAULT_IMPL,
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
        _upgradeToAndCall(
            Predeploys.OPERATOR_FEE_VAULT,
            OPERATOR_FEE_VAULT_IMPL,
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
        _upgradeTo(Predeploys.GAS_PRICE_ORACLE, GAS_PRICE_ORACLE_IMPL);
        // L1BlockAttributes and L2ToL1MessagePasser have different implementations for custom gas token networks.
        _upgradeTo(
            Predeploys.L1_BLOCK_ATTRIBUTES, isCustomGasToken ? L1_BLOCK_ATTRIBUTES_CGT_IMPL : L1_BLOCK_ATTRIBUTES_IMPL
        );
        _upgradeTo(
            Predeploys.L2_TO_L1_MESSAGE_PASSER,
            isCustomGasToken ? L2_TO_L1_MESSAGE_PASSER_CGT_IMPL : L2_TO_L1_MESSAGE_PASSER_IMPL
        );
        _upgradeTo(Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY, OPTIMISM_MINTABLE_ERC721_FACTORY_IMPL);
        _upgradeTo(Predeploys.PROXY_ADMIN, PROXY_ADMIN_IMPL);
        _upgradeTo(Predeploys.CROSS_L2_INBOX, CROSS_L2_INBOX_IMPL);
        _upgradeTo(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER, L2_TO_L2_CROSS_DOMAIN_MESSENGER_IMPL);
        _upgradeTo(Predeploys.SUPERCHAIN_ETH_BRIDGE, SUPERCHAIN_ETH_BRIDGE_IMPL);
        _upgradeTo(Predeploys.ETH_LIQUIDITY, ETH_LIQUIDITY_IMPL);
        _upgradeTo(Predeploys.OPTIMISM_SUPERCHAIN_ERC20_FACTORY, OPTIMISM_SUPERCHAIN_ERC20_FACTORY_IMPL);
        _upgradeTo(Predeploys.OPTIMISM_SUPERCHAIN_ERC20_BEACON, OPTIMISM_SUPERCHAIN_ERC20_BEACON_IMPL);
        _upgradeTo(Predeploys.SUPERCHAIN_TOKEN_BRIDGE, SUPERCHAIN_TOKEN_BRIDGE_IMPL);
        // NativeAssetLiquidity
        if (isCustomGasToken) {
            _upgradeTo(Predeploys.NATIVE_ASSET_LIQUIDITY, NATIVE_ASSET_LIQUIDITY_IMPL);
        }
        _upgradeTo(Predeploys.SCHEMA_REGISTRY, SCHEMA_REGISTRY_IMPL);
        _upgradeTo(Predeploys.EAS, EAS_IMPL);
    }

    /// @notice Upgrades a predeploy to a new implementation without calling an initializer.
    /// @param _proxy The proxy address of the predeploy.
    /// @param _implementation The new implementation address.
    function _upgradeTo(address _proxy, address _implementation) internal {
        IProxy(payable(_proxy)).upgradeTo(_implementation);
    }

    /// @notice Upgrades an initializable Predeploy's implementation to _implementation by resetting the initialized
    ///         slot and calling upgradeToAndCall with _data.
    /// @dev It's important to make sure that only initializable Predeploys are upgraded to this way.
    /// @param _proxy The proxy of the contract.
    /// @param _implementation The new implementation of the contract.
    /// @param _data The data to call upgradeToAndCall with.
    /// @param _slot The slot where the initialized value is located.
    /// @param _offset The offset of the initializer value in the slot.
    function _upgradeToAndCall(
        address _proxy,
        address _implementation,
        bytes memory _data,
        bytes32 _slot,
        uint8 _offset
    )
        internal
    {
        // Upgrade to StorageSetter.
        IProxy(payable(_proxy)).upgradeTo(STORAGE_SETTER_IMPL);

        // Reset the initialized slot by zeroing the single byte at `_offset` (from the right).
        bytes32 current = IStorageSetter(_proxy).getBytes32(_slot);
        uint256 mask = ~(uint256(0xff) << (uint256(_offset) * 8));
        IStorageSetter(_proxy).setBytes32(_slot, bytes32(uint256(current) & mask));

        // Upgrade to the implementation and call the initializer.
        IProxy(payable(_proxy)).upgradeToAndCall(_implementation, _data);
    }
}

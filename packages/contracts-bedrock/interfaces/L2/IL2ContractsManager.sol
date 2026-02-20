// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

// Libraries
import { L2ContractsManagerTypes } from "src/libraries/L2ContractsManagerTypes.sol";

/// @title IL2ContractsManager
/// @notice Interface for the L2ContractsManager contract.
interface IL2ContractsManager is ISemver {
    /// @notice Thrown when the upgrade function is called outside of a DELEGATECALL context.
    error L2ContractsManager_OnlyDelegatecall();

    /// @notice Thrown when a user attempts to downgrade a contract.
    /// @param _target The address of the contract that was attempted to be downgraded.
    error L2ContractsManager_DowngradeNotAllowed(address _target);

    /// @notice Error thrown when a semver string has less than 3 parts.
    error SemverComp_InvalidSemverParts();

    /// @notice Executes the upgrade for all predeploys.
    /// @dev This function MUST be called via DELEGATECALL from the L2ProxyAdmin.
    function upgrade() external;

    /// @notice Constructor for the L2ContractsManager contract.
    /// @param _implementations The implementation struct containing the new implementation addresses for the L2
    /// predeploys.
    function __constructor__(L2ContractsManagerTypes.Implementations memory _implementations) external;

    /// @notice Returns the implementation address of the StorageSetter contract.
    function storageSetterImpl() external view returns (address);

    /// @notice Returns the implementation address of the GasPriceOracle contract.
    function gasPriceOracleImpl() external view returns (address);

    /// @notice Returns the implementation address of the L2CrossDomainMessenger contract.
    function l2CrossDomainMessengerImpl() external view returns (address);

    /// @notice Returns the implementation address of the L2StandardBridge contract.
    function l2StandardBridgeImpl() external view returns (address);

    /// @notice Returns the implementation address of the SequencerFeeWallet contract.
    function sequencerFeeWalletImpl() external view returns (address);

    /// @notice Returns the implementation address of the OptimismMintableERC20Factory contract.
    function optimismMintableERC20FactoryImpl() external view returns (address);

    /// @notice Returns the implementation address of the L2ERC721Bridge contract.
    function l2ERC721BridgeImpl() external view returns (address);

    /// @notice Returns the implementation address of the L1Block contract.
    function l1BlockImpl() external view returns (address);

    /// @notice Returns the implementation address of the L1Block contract for custom gas token networks.
    function l1BlockCGTImpl() external view returns (address);

    /// @notice Returns the implementation address of the L2ToL1MessagePasser contract.
    function l2ToL1MessagePasserImpl() external view returns (address);

    /// @notice Returns the implementation address of the L2ToL1MessagePasser contract for custom gas token networks.
    function l2ToL1MessagePasserCGTImpl() external view returns (address);

    /// @notice Returns the implementation address of the OptimismMintableERC721Factory contract.
    function optimismMintableERC721FactoryImpl() external view returns (address);

    /// @notice Returns the implementation address of the ProxyAdmin contract.
    function proxyAdminImpl() external view returns (address);

    /// @notice Returns the implementation address of the BaseFeeVault contract.
    function baseFeeVaultImpl() external view returns (address);

    /// @notice Returns the implementation address of the L1FeeVault contract.
    function l1FeeVaultImpl() external view returns (address);

    /// @notice Returns the implementation address of the OperatorFeeVault contract.
    function operatorFeeVaultImpl() external view returns (address);

    /// @notice Returns the implementation address of the SchemaRegistry contract.
    function schemaRegistryImpl() external view returns (address);

    /// @notice Returns the implementation address of the EAS contract.
    function easImpl() external view returns (address);

    /// @notice Returns the implementation address of the CrossL2Inbox contract.
    function crossL2InboxImpl() external view returns (address);

    /// @notice Returns the implementation address of the L2ToL2CrossDomainMessenger contract.
    function l2ToL2CrossDomainMessengerImpl() external view returns (address);

    /// @notice Returns the implementation address of the SuperchainETHBridge contract.
    function superchainETHBridgeImpl() external view returns (address);

    /// @notice Returns the implementation address of the ETHLiquidity contract.
    function ethLiquidityImpl() external view returns (address);

    /// @notice Returns the implementation address of the OptimismSuperchainERC20Factory contract.
    function optimismSuperchainERC20FactoryImpl() external view returns (address);

    /// @notice Returns the implementation address of the OptimismSuperchainERC20Beacon contract.
    function optimismSuperchainERC20BeaconImpl() external view returns (address);

    /// @notice Returns the implementation address of the SuperchainTokenBridge contract.
    function superchainTokenBridgeImpl() external view returns (address);

    /// @notice Returns the implementation address of the NativeAssetLiquidity contract.
    function nativeAssetLiquidityImpl() external view returns (address);

    /// @notice Returns the implementation address of the LiquidityController contract.
    function liquidityControllerImpl() external view returns (address);

    /// @notice Returns the implementation address of the FeeSplitter contract.
    function feeSplitterImpl() external view returns (address);

    /// @notice Returns the implementation address of the ConditionalDeployer contract.
    function conditionalDeployerImpl() external view returns (address);
}

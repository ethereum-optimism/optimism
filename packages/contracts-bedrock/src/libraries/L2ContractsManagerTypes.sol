// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Types } from "src/libraries/Types.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IStandardBridge } from "interfaces/universal/IStandardBridge.sol";
import { IERC721Bridge } from "interfaces/universal/IERC721Bridge.sol";
import { ISharesCalculator } from "interfaces/L2/ISharesCalculator.sol";

/// @title L2ContractsManagerTypes
/// @notice Type definitions for L2ContractsManager upgrade operations.
library L2ContractsManagerTypes {
    /// @notice Configuration for L2CrossDomainMessenger.
    struct CrossDomainMessengerConfig {
        ICrossDomainMessenger otherMessenger;
    }

    /// @notice Configuration for L2StandardBridge.
    struct StandardBridgeConfig {
        IStandardBridge otherBridge;
    }

    /// @notice Configuration for L2ERC721Bridge.
    struct ERC721BridgeConfig {
        IERC721Bridge otherBridge;
    }

    /// @notice Configuration for OptimismMintableERC20Factory.
    struct MintableERC20FactoryConfig {
        address bridge;
    }

    /// @notice Configuration for OptimismMintableERC721Factory.
    struct MintableERC721FactoryConfig {
        address bridge;
        uint256 remoteChainID;
    }

    /// @notice Configuration for a FeeVault contract.
    struct FeeVaultConfig {
        address recipient;
        uint256 minWithdrawalAmount;
        Types.WithdrawalNetwork withdrawalNetwork;
    }

    /// @notice Configuration for LiquidityController.
    struct LiquidityControllerConfig {
        address owner;
        string gasPayingTokenName;
        string gasPayingTokenSymbol;
    }

    /// @notice Configuration for FeeSplitter.
    struct FeeSplitterConfig {
        ISharesCalculator sharesCalculator;
    }

    /// @notice Full network-specific configuration gathered from existing predeploys.
    ///         These values are read before upgrade and passed to initializers after.
    struct FullConfig {
        CrossDomainMessengerConfig crossDomainMessenger;
        StandardBridgeConfig standardBridge;
        ERC721BridgeConfig erc721Bridge;
        MintableERC20FactoryConfig mintableERC20Factory;
        MintableERC721FactoryConfig mintableERC721Factory;
        FeeVaultConfig sequencerFeeVault;
        FeeVaultConfig baseFeeVault;
        FeeVaultConfig l1FeeVault;
        FeeVaultConfig operatorFeeVault;
        LiquidityControllerConfig liquidityController;
        FeeSplitterConfig feeSplitter;
        bool isCustomGasToken;
    }

    /// @notice The current implementation addresses for the L2 predeploys.
    struct Implementations {
        address storageSetterImpl;
        address l2CrossDomainMessengerImpl;
        address gasPriceOracleImpl;
        address l2StandardBridgeImpl;
        address sequencerFeeWalletImpl;
        address optimismMintableERC20FactoryImpl;
        address l2ERC721BridgeImpl;
        address l1BlockImpl;
        address l1BlockCGTImpl;
        address l2ToL1MessagePasserImpl;
        address l2ToL1MessagePasserCGTImpl;
        address optimismMintableERC721FactoryImpl;
        address proxyAdminImpl;
        address baseFeeVaultImpl;
        address l1FeeVaultImpl;
        address operatorFeeVaultImpl;
        address schemaRegistryImpl;
        address easImpl;
        address crossL2InboxImpl;
        address l2ToL2CrossDomainMessengerImpl;
        address superchainETHBridgeImpl;
        address ethLiquidityImpl;
        address optimismSuperchainERC20FactoryImpl;
        address optimismSuperchainERC20BeaconImpl;
        address superchainTokenBridgeImpl;
        address nativeAssetLiquidityImpl;
        address liquidityControllerImpl;
        address feeSplitterImpl;
        address conditionalDeployerImpl;
        address l2DevFeatureFlagsImpl;
    }
}

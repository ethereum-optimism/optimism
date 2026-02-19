// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title L2ContractsManagerTypes
/// @notice Type definitions for L2ContractsManager upgrade operations.
library L2ContractsManagerTypes {
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
    }
}

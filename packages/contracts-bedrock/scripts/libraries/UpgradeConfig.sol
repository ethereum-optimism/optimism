// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Fork } from "scripts/libraries/Config.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

/// @title UpgradeConfig
/// @notice Configuration library for L2 hardfork upgrade transaction generation.
/// @dev Provides gas limits and transaction counts for upgrade bundle generation.
library UpgradeConfig {
    /// @notice The number of predeploy implementations always deployed in every upgrade.
    ///         This is the base count that applies to all forks and configurations.
    ///         21 predeploys + 1 StorageSetter (deployed separately) = 22 total base implementations.
    ///         Additional implementations may be deployed conditionally based on:
    ///         - Fork version (e.g., INTEROP adds CrossL2Inbox, L2ToL2CrossDomainMessenger)
    ///         - Custom gas token flag (adds NativeAssetLiquidity, LiquidityController)
    uint256 internal constant IMPLEMENTATION_COUNT = 22;

    /// @notice Gas limits for different types of upgrade transactions.
    /// @param conditionalDeployerDeployment Gas for deploying ConditionalDeployer
    /// @param conditionalDeployerUpgrade Gas for upgrading ConditionalDeployer proxy
    /// @param proxyAdminUpgrade Gas for upgrading ProxyAdmin implementation
    /// @param l2cmDeployment Gas for deploying L2ContractsManager
    /// @param upgradeExecution Gas for L2ProxyAdmin.upgradePredeploys() call
    struct GasLimits {
        // Fixed
        uint64 storageSetterDeployment;
        uint64 l2cmDeployment;
        uint64 upgradeExecution;
        // Jovian
        uint64 conditionalDeployerDeployment;
        uint64 conditionalDeployerUpgrade;
        uint64 proxyAdminUpgrade;
    }

    /// @notice Calculates the total number of transactions for a given fork.
    function calculateTransactionCount(Fork _fork, bool _useCustomGasToken) public pure returns (uint256 txnCount_) {
        txnCount_ = IMPLEMENTATION_COUNT + 2; // Implementations + L2CM deployment + Upgrade Predeploys call

        if (_fork == Fork.JOVIAN) {
            txnCount_ += 3; // ConditionalDeployer (deployment + upgrade) + ProxyAdmin upgrade
        }
        if (_useCustomGasToken) {
            txnCount_ += 2; // NativeAssetLiquidity & LiquidityController deployment
        }
        if (_fork >= Fork.INTEROP) {
            txnCount_ += 2; // CrossL2Inbox & L2ToL2CrossDomainMessenger deployment
        }
    }

    /// @notice Returns the gas limits for all upgrade transaction types.
    /// @dev Gas limits are chosen to provide sufficient headroom while being
    ///      conservative enough to fit within the upgrade block gas allocation.
    ///      Rationale for each limit:
    ///      - [complete rationale here]
    /// @return Gas limits struct.
    function gasLimits() internal pure returns (GasLimits memory) {
        return GasLimits({
            // Fixed
            storageSetterDeployment: 375_000,
            l2cmDeployment: 375_000,
            upgradeExecution: type(uint64).max,
            // Jovian
            conditionalDeployerDeployment: 375_000,
            conditionalDeployerUpgrade: 50_000,
            proxyAdminUpgrade: 50_000
        });
    }
}

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Interfaces
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title IL2ProxyAdmin
interface IL2ProxyAdmin is IProxyAdmin, ISemver {
    /// @notice Emitted when the predeploys are upgraded.
    /// @param xForkL2ContractsManager Address of the xForkL2ContractsManager contract.
    event PredeploysUpgraded(address indexed xForkL2ContractsManager);

    /// @notice Thrown when the caller is not the depositor account.
    error L2ProxyAdmin__Unauthorized();

    /// @notice Thrown when the upgrade fails.
    error L2ProxyAdmin__UpgradeFailed(bytes data);

    function __constructor__(address _owner) external;

    /// @notice Upgrades the predeploys via delegatecall to the xForkL2ContractsManager contract.
    /// @param xForkL2ContractsManager Address of the xForkL2ContractsManager contract.
    function upgradePredeploys(address xForkL2ContractsManager) external;
}

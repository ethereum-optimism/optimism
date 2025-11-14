// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";
import { IL1ChugSplashProxy } from "interfaces/legacy/IL1ChugSplashProxy.sol";
import { IAddressManager } from "interfaces/legacy/IAddressManager.sol";
import { Constants } from "src/libraries/Constants.sol";

/// @notice Base contract for L2 Contracts Manager, responsible for orquestrating the upgrades
///         of the L2 contracts during hardforks.
abstract contract L2ContractsManager {
    /// @notice Struct representing the data for an upgrade.
    struct ProxyUpgrade {
        address proxy;
        address implementation;
    }

    /// @notice Executes the NUT with before/after hooks.
    function execute() external {
        _beforeExecution();
        _performUpgrades();
        _afterExecution();
    }

    /// @notice Hook called before execution.
    function _beforeExecution() internal virtual;

    /// @notice Hook called after execution.
    function _afterExecution() internal virtual;

    /// @notice Performs the proxy upgrades logic.
    function _performUpgrades() internal virtual { }
}

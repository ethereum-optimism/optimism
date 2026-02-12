// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Interfaces
import { IL2ContractsManager } from "interfaces/L2/IL2ContractsManager.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";

// Contracts
import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";

/// @custom:proxied true
/// @custom:predeploy 0x4200000000000000000000000000000000000018
/// @title L2ProxyAdmin
/// @notice The L2ProxyAdmin is the administrative contract responsible for managing proxy upgrades
///         for L2 predeploy contracts.
/// @dev    It extends the standard ProxyAdmin with an upgradePredeploys() function that  orchestates
///         batch upgrades of multiple predeploys by delegating to an L2ContractsManager contract.
contract L2ProxyAdmin is ProxyAdmin, ISemver {
    /// @notice Emitted when the predeploys are upgraded.
    /// @param l2ContractsManager Address of the L2ContractsManager contract.
    event PredeploysUpgraded(address indexed l2ContractsManager);

    /// @notice Thrown when the caller is not the depositor account.
    error L2ProxyAdmin__Unauthorized();

    /// @notice Thrown when the upgrade fails.
    error L2ProxyAdmin__UpgradeFailed(bytes data);

    /// @notice The semantic version of the contract.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice The constructor for the L2ProxyAdmin contract.
    /// @param _owner Address of the initial owner of this contract.
    constructor(address _owner) ProxyAdmin(_owner) { }

    /// @notice Upgrades the predeploys via delegatecall to the l2ContractsManager contract.
    /// @param _l2ContractsManager Address of the l2ContractsManager contract.
    function upgradePredeploys(address _l2ContractsManager) external {
        if (msg.sender != Constants.DEPOSITOR_ACCOUNT) revert L2ProxyAdmin__Unauthorized();

        (bool success, bytes memory data) =
            _l2ContractsManager.delegatecall(abi.encodeCall(IL2ContractsManager.upgrade, ()));

        if (!success) revert L2ProxyAdmin__UpgradeFailed(data);

        emit PredeploysUpgraded(_l2ContractsManager);
    }
}

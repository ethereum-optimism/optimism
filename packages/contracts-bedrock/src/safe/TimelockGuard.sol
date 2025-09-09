// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Safe
import { GnosisSafe as Safe } from "safe-contracts/GnosisSafe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import { GuardManager, Guard as IGuard } from "safe-contracts/base/GuardManager.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title TimelockGuard
/// @notice This guard provides timelock functionality for Safe transactions
/// @dev This is a singleton contract. To use it:
///      1. The Safe must first enable this guard using GuardManager.setGuard()
///      2. The Safe must then configure the guard by calling configureTimelockGuard()
contract TimelockGuard is IGuard, ISemver {
    /// @notice Configuration for a Safe's timelock guard
    struct GuardConfig {
        uint256 timelockDelay;
    }

    /// @notice Mapping from Safe address to its guard configuration
    mapping(address => GuardConfig) public safeConfigs;

    /// @notice Mapping from Safe address to its current cancellation threshold
    mapping(address => uint256) public safeCancellationThreshold;

    /// @notice Error for when guard is not enabled for the Safe
    error TimelockGuard_GuardNotEnabled();

    /// @notice Error for when Safe is not configured for this guard
    error TimelockGuard_GuardNotConfigured();

    /// @notice Error for when attempt to clear guard while it is still enabled for the Safe
    error TimelockGuard_GuardStillEnabled();

    /// @notice Error for invalid timelock delay
    error TimelockGuard_InvalidTimelockDelay();

    /// @notice Emitted when a Safe configures the guard
    event GuardConfigured(address indexed safe, uint256 timelockDelay);

    /// @notice Emitted when a Safe clears the guard configuration
    event GuardCleared(address indexed safe);

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Returns the timelock delay for a given Safe
    /// @dev MUST never revert
    /// @param _safe The Safe address to query
    /// @return The timelock delay in seconds
    function viewTimelockGuardConfiguration(address _safe) public view returns (uint256) {
        // Q: What should this return if the guard is not enabled?
        return safeConfigs[_safe].timelockDelay;
    }

    /// @notice Configure the contract as a timelock guard by setting the timelock delay
    /// @dev MUST allow an arbitrary number of Safe contracts to use the contract as a guard
    /// @dev The contract MUST be enabled as a guard for the Safe
    /// @dev MUST revert if timelock_delay is longer than 1 year
    /// @dev MUST set the caller as a Safe
    /// @dev MUST take timelock_delay as a parameter and store it as related to the Safe
    /// @dev MUST emit a GuardConfigured event with at least timelock_delay as a parameter
    /// @param _timelockDelay The timelock delay in seconds
    function configureTimelockGuard(uint256 _timelockDelay) external {
        // Validate timelock delay - must be non-zero and not longer than 1 year
        if (_timelockDelay == 0 || _timelockDelay > 365 days) {
            revert TimelockGuard_InvalidTimelockDelay();
        }

        // Check that this guard is enabled on the calling Safe
        if (_getGuard(msg.sender) != address(this)) {
            revert TimelockGuard_GuardNotEnabled();
        }

        // Store the configuration for this safe
        safeConfigs[msg.sender].timelockDelay = _timelockDelay;

        // Initialize cancellation threshold to 1
        safeCancellationThreshold[msg.sender] = 1;

        emit GuardConfigured(msg.sender, _timelockDelay);
    }

    /// @notice Remove the timelock guard configuration by a previously enabled Safe
    /// @dev The contract MUST NOT be enabled as a guard for the Safe
    /// @dev MUST erase the existing timelock_delay data related to the calling Safe
    /// @dev MUST emit a GuardCleared event
    function clearTimelockGuard() external {
        // Check if the calling safe has configuration set
        if (safeConfigs[msg.sender].timelockDelay == 0) {
            revert TimelockGuard_GuardNotConfigured();
        }

        // Check that this guard is NOT enabled on the calling Safe
        if (_getGuard(msg.sender) == address(this)) {
            revert TimelockGuard_GuardStillEnabled();
        }

        // Erase the configuration data for this safe
        delete safeConfigs[msg.sender];
        delete safeCancellationThreshold[msg.sender];

        emit GuardCleared(msg.sender);
    }

    /// @notice Returns the cancellation threshold for a given safe
    /// @dev MUST NOT revert
    /// @dev MUST return 0 if the contract is not enabled as a guard for the safe
    /// @param _safe The Safe address to query
    /// @return The current cancellation threshold
    function cancellationThreshold(address _safe) public view returns (uint256) {
        // Return 0 if guard is not enabled
        if (_getGuard(_safe) != address(this)) {
            return 0;
        }

        // Return 0 if not configured
        if (safeConfigs[_safe].timelockDelay == 0) {
            return 0;
        }

        uint256 threshold = safeCancellationThreshold[_safe];
        if (threshold == 0) {
            // NOTE: not sure if this is the right thing to do.
            //    defaulting to one is good to prevent us from forgetting to set it to one elsewhere.
            // Default to 1 if not set
            return 1;
        }
        return threshold;
    }

    /// @notice Internal helper to get the guard address from a Safe
    /// @param _safe The Safe address
    /// @return The current guard address
    function _getGuard(address _safe) internal view returns (address) {
        // keccak256("guard_manager.guard.address") from GuardManager
        bytes32 guardSlot = 0x4a204f620c8c5ccdca3fd54d003badd85ba500436a431f0cbda4f558c93c34c8;
        Safe safe = Safe(payable(_safe));
        return abi.decode(safe.getStorageAt({ offset: uint256(guardSlot), length: 1 }), (address));
    }

    /// @notice Called by the Safe before executing a transaction
    /// @dev Implementation of IGuard interface
    function checkTransaction(
        address to,
        uint256 value,
        bytes memory data,
        Enum.Operation operation,
        uint256 safeTxGas,
        uint256 baseGas,
        uint256 gasPrice,
        address gasToken,
        address payable refundReceiver,
        bytes memory signatures,
        address msgSender
    )
        external
        override
    {
        // Empty implementation for now - will be filled in when implementing checkTransaction
    }

    /// @notice Called by the Safe after executing a transaction
    /// @dev Implementation of IGuard interface
    function checkAfterExecution(bytes32 txHash, bool success) external override {
        // Empty implementation for now
    }
}

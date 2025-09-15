// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Safe
import { GnosisSafe as Safe } from "safe-contracts/GnosisSafe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import { OwnerManager } from "safe-contracts/base/OwnerManager.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title LivenessModule2
/// @notice This module allows challenge-based ownership transfer to a fallback owner
///         when the Safe becomes unresponsive. The fallback owner can initiate a challenge,
///         and if the Safe doesn't respond within the challenge period, ownership transfers
///         to the fallback owner.
/// @dev This is a singleton contract. To use it:
///      1. The Safe must first enable this module using ModuleManager.enableModule()
///      2. The Safe must then configure the module by calling configure() with params
contract LivenessModule2 is ISemver {
    /// @notice Configuration for a Safe's liveness module
    struct ModuleConfig {
        uint256 livenessResponsePeriod;
        address fallbackOwner;
    }

    /// @notice Mapping from Safe address to its configuration
    mapping(address => ModuleConfig) public safeConfigs;

    /// @notice Mapping from Safe address to active challenge start time (0 if none)
    mapping(address => uint256) public challengeStartTime;

    /// @notice Reserved address used as previous owner to the first owner in a Safe
    address internal constant SENTINEL_OWNER = address(0x1);

    /// @notice Error for when module is not enabled for the Safe
    error LivenessModule2_ModuleNotEnabled();

    /// @notice Error for when Safe is not configured for this module
    error LivenessModule2_ModuleNotConfigured();

    /// @notice Error for when a challenge already exists
    error LivenessModule2_ChallengeAlreadyExists();

    /// @notice Error for when no challenge exists
    error LivenessModule2_ChallengeDoesNotExist();

    /// @notice Error for when trying to cancel a challenge after response period has ended
    error LivenessModule2_ResponsePeriodEnded();

    /// @notice Error for when trying to execute ownership transfer while response period is active
    error LivenessModule2_ResponsePeriodActive();

    /// @notice Error for when caller is not authorized
    error LivenessModule2_UnauthorizedCaller();

    /// @notice Error for invalid response period
    error LivenessModule2_InvalidResponsePeriod();

    /// @notice Error for invalid fallback owner
    error LivenessModule2_InvalidFallbackOwner();

    /// @notice Error for when trying to clear configuration while module is enabled
    error LivenessModule2_ModuleStillEnabled();

    /// @notice Error for when ownership transfer verification fails
    error LivenessModule2_OwnershipTransferFailed();

    /// @notice Emitted when a Safe enables the module
    event ModuleEnabled(address indexed safe, uint256 livenessResponsePeriod, address fallbackOwner);

    /// @notice Emitted when a Safe disables the module
    event ModuleDisabled(address indexed safe);

    /// @notice Emitted when a challenge is started
    event ChallengeStarted(address indexed safe, uint256 challengeStartTime);

    /// @notice Emitted when a challenge is cancelled
    event ChallengeCancelled(address indexed safe);

    /// @notice Emitted when ownership is transferred to the fallback owner
    event ChallengeSucceeded(address indexed safe, address fallbackOwner);

    /// @notice Semantic version.
    /// @custom:semver 2.0.0
    string public constant version = "2.0.0";

    /// @notice Returns challenge_start_time + liveness_response_period if challenge exists, or
    /// 0 if not
    /// @param _safe The Safe address to query
    /// @return The challenge end timestamp, or 0 if no challenge
    function getChallengePeriodEnd(address _safe) public view returns (uint256) {
        uint256 startTime = challengeStartTime[_safe];
        if (startTime == 0) {
            return 0;
        }
        ModuleConfig storage config = safeConfigs[_safe];
        return startTime + config.livenessResponsePeriod;
    }

    /// @notice Configures the module for a Safe that has already enabled it
    /// @param _config The configuration parameters for the module
    function configure(ModuleConfig memory _config) external {
        // Validate configuration parameters to ensure module can function properly.
        // livenessResponsePeriod must be > 0 to allow time for Safe owners to respond.
        if (_config.livenessResponsePeriod == 0) {
            revert LivenessModule2_InvalidResponsePeriod();
        }
        // fallbackOwner must not be zero address to have a valid ownership recipient.
        if (_config.fallbackOwner == address(0)) {
            revert LivenessModule2_InvalidFallbackOwner();
        }

        // Check that this module is enabled on the calling Safe.
        _assertModuleEnabled(msg.sender);

        // Store the configuration for this safe
        safeConfigs[msg.sender] = _config;

        // Clear any existing challenge when configuring/re-configuring.
        // This is necessary because changing the configuration (especially livenessResponsePeriod)
        // would invalidate any ongoing challenge timing, creating inconsistent state.
        // For example, if a challenge was started with a 7-day period and we reconfigure to
        // 1 day, the challenge timing becomes ambiguous. Canceling ensures clean state.
        _cancelChallenge(msg.sender);

        emit ModuleEnabled(msg.sender, _config.livenessResponsePeriod, _config.fallbackOwner);
    }

    /// @notice Clears the module configuration for a Safe
    /// @dev Note: Clearing the configuration also cancels any ongoing challenges
    function clear() external {
        // Check if the calling safe has configuration set
        _assertModuleConfigured(msg.sender);
        // Check that this module is NOT enabled on the calling Safe
        // This prevents clearing configuration while module is still enabled
        _assertModuleNotEnabled(msg.sender);

        // Erase the configuration data for this safe
        delete safeConfigs[msg.sender];
        // Also clear any active challenge
        _cancelChallenge(msg.sender);
        emit ModuleDisabled(msg.sender);
    }

    /// @notice Challenges an enabled safe
    /// @param _safe The Safe to challenge
    function challenge(address _safe) external {
        _assertModuleConfigured(_safe);

        // Check that the module is still enabled on the target Safe
        _assertModuleEnabled(_safe);

        if (msg.sender != safeConfigs[_safe].fallbackOwner) {
            revert LivenessModule2_UnauthorizedCaller();
        }

        if (challengeStartTime[_safe] != 0) {
            revert LivenessModule2_ChallengeAlreadyExists();
        }

        challengeStartTime[_safe] = block.timestamp;
        emit ChallengeStarted(_safe, block.timestamp);
    }

    /// @notice Responds to a challenge for an enabled safe, canceling it
    function respond() external {
        // Check that this module is enabled on the calling Safe
        _assertModuleEnabled(msg.sender);

        // Check if the calling safe has configuration set
        _assertModuleConfigured(msg.sender);

        uint256 startTime = challengeStartTime[msg.sender];
        if (startTime == 0) {
            revert LivenessModule2_ChallengeDoesNotExist();
        }

        // Cancel the challenge without checking if response period has expired
        // This allows the Safe to respond at any time, providing more flexibility
        _cancelChallenge(msg.sender);
    }

    /// @notice With successful challenge, removes all current owners from enabled safe,
    /// appoints fallback as sole owner, and sets its quorum to 1
    /// @param _safe The Safe to transfer ownership of
    function changeOwnershipToFallback(address _safe) external {
        // Ensure Safe is configured with this module to prevent unauthorized execution
        _assertModuleConfigured(_safe);

        // Verify module is still enabled to ensure Safe hasn't disabled it mid-challenge
        _assertModuleEnabled(_safe);

        // Only fallback owner can execute ownership transfer (per specs update)
        if (msg.sender != safeConfigs[_safe].fallbackOwner) {
            revert LivenessModule2_UnauthorizedCaller();
        }

        // Verify active challenge exists - without challenge, ownership transfer not allowed
        uint256 startTime = challengeStartTime[_safe];
        if (startTime == 0) {
            revert LivenessModule2_ChallengeDoesNotExist();
        }

        // Ensure response period has fully expired before allowing ownership transfer.
        // This gives Safe owners full configured time to demonstrate liveness.
        if (block.timestamp < getChallengePeriodEnd(_safe)) {
            revert LivenessModule2_ResponsePeriodActive();
        }

        Safe targetSafe = Safe(payable(_safe));

        // Get current owners
        address[] memory owners = targetSafe.getOwners();

        // Remove all owners after the first one
        // Note: This loop is safe as real-world Safes have limited owners (typically < 10)
        // Gas limits would only be a concern with hundreds/thousands of owners
        while (owners.length > 1) {
            targetSafe.execTransactionFromModule({
                to: _safe,
                value: 0,
                operation: Enum.Operation.Call,
                data: abi.encodeCall(OwnerManager.removeOwner, (SENTINEL_OWNER, owners[0], 1))
            });
            owners = targetSafe.getOwners();
        }

        // Now swap the remaining single owner with the fallback owner
        targetSafe.execTransactionFromModule({
            to: _safe,
            value: 0,
            operation: Enum.Operation.Call,
            data: abi.encodeCall(OwnerManager.swapOwner, (SENTINEL_OWNER, owners[0], safeConfigs[_safe].fallbackOwner))
        });

        // Sanity check: verify the fallback owner is now the only owner
        address[] memory finalOwners = targetSafe.getOwners();
        if (finalOwners.length != 1 || finalOwners[0] != safeConfigs[_safe].fallbackOwner) {
            revert LivenessModule2_OwnershipTransferFailed();
        }

        // Reset the challenge state to allow a new challenge
        delete challengeStartTime[_safe];

        emit ChallengeSucceeded(_safe, safeConfigs[_safe].fallbackOwner);
    }

    function _assertModuleConfigured(address _safe) internal view {
        ModuleConfig storage config = safeConfigs[_safe];
        if (config.fallbackOwner == address(0)) {
            revert LivenessModule2_ModuleNotConfigured();
        }
    }

    function _assertModuleEnabled(address _safe) internal view {
        Safe safe = Safe(payable(_safe));
        if (!safe.isModuleEnabled(address(this))) {
            revert LivenessModule2_ModuleNotEnabled();
        }
    }

    /// @notice Internal function to cancel a challenge and emit the appropriate event
    /// @param _safe The Safe address for which to cancel the challenge
    function _cancelChallenge(address _safe) internal {
        // Early return if no challenge exists
        if (challengeStartTime[_safe] == 0) return;

        delete challengeStartTime[_safe];
        emit ChallengeCancelled(_safe);
    }

    function _assertModuleNotEnabled(address _safe) internal view {
        Safe safe = Safe(payable(_safe));
        if (safe.isModuleEnabled(address(this))) {
            revert LivenessModule2_ModuleStillEnabled();
        }
    }
}

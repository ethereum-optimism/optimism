// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Safe
import { GnosisSafe as Safe } from "safe-contracts/GnosisSafe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import { OwnerManager } from "safe-contracts/base/OwnerManager.sol";

// Interfaces
import { ILivenessModule2 } from "interfaces/safe/ILivenessModule2.sol";

/// @title LivenessModule2
/// @notice This module allows for challenge-based ownership transfer to a fallback owner
///         when the Safe becomes unresponsive. The fallback owner can initiate a challenge,
///         and if the Safe doesn't respond within the challenge period, ownership transfers
///         to the fallback owner.
/// @dev This is a singleton contract. To use it:
///      1. The Safe must first enable this module using ModuleManager.enableModule()
///      2. The Safe must then configure the module by calling enableModule() with parameters
contract LivenessModule2 is ILivenessModule2 {
    /// @notice Reserved address used as the previous owner to the first owner in a Safe
    address public constant SENTINEL_OWNER = address(0x1);

    /// @notice Mapping from Safe address to its configuration
    mapping(address => ModuleConfig) public safeConfigs;

    /// @notice Mapping from Safe address to active challenge start time (0 if no challenge)
    mapping(address => uint256) public challengeStartTime;

    /// @notice Semantic version.
    /// @custom:semver 2.0.0
    string public constant version = "2.0.0";

    /// @notice Configures the module for a Safe that has already enabled it
    /// @dev MUST only be callable by a Safe that has enabled this module
    /// @param _config The configuration parameters for the module
    function configure(ModuleConfig memory _config) external {
        if (_config.livenessResponsePeriod == 0 || _config.fallbackOwner == address(0)) {
            revert LivenessModule2_InvalidParameters();
        }

        // Check that this module is enabled on the calling Safe
        Safe safe = Safe(payable(msg.sender));
        if (!safe.isModuleEnabled(address(this))) {
            revert LivenessModule2_ModuleNotEnabled();
        }

        // Store the configuration for this safe
        safeConfigs[msg.sender] = _config;

        // Clear any existing challenge when configuring/re-configuring
        delete challengeStartTime[msg.sender];

        emit ModuleEnabled(msg.sender, _config.livenessResponsePeriod, _config.fallbackOwner);
    }

    /// @notice Clears the module configuration for a Safe
    /// @dev MUST only be executable by a Safe that has DISABLED this module first
    /// @dev MUST erase the existing liveness_response_period and fallback_owner data related to the calling safe
    /// @dev Note: Clearing the configuration also cancels any ongoing challenges
    function clear() external {
        // Check if the calling safe has configuration set
        ModuleConfig storage config = safeConfigs[msg.sender];
        if (config.fallbackOwner == address(0)) {
            revert LivenessModule2_ModuleNotConfigured();
        }

        // Check that this module is NOT enabled on the calling Safe
        // This prevents clearing configuration while module is still enabled
        Safe safe = Safe(payable(msg.sender));
        if (safe.isModuleEnabled(address(this))) {
            revert LivenessModule2_ModuleStillEnabled();
        }

        // Erase the configuration data for this safe
        delete safeConfigs[msg.sender];
        // Also clear any active challenge
        delete challengeStartTime[msg.sender];
        emit ModuleDisabled(msg.sender);
    }

    /// @notice Returns challenge_start_time + liveness_response_period if there is a challenge for the given safe, or
    /// 0 if not
    /// @dev MUST never revert
    /// @param _safe The Safe address to query
    /// @return The challenge end timestamp, or 0 if no challenge
    function isChallenged(address _safe) external view returns (uint256) {
        uint256 startTime = challengeStartTime[_safe];
        if (startTime == 0) {
            return 0;
        }
        ModuleConfig storage config = safeConfigs[_safe];
        return startTime + config.livenessResponsePeriod;
    }

    /// @notice Challenges an enabled safe
    /// @dev MUST only be executable by fallback owner of the challenged safe
    /// @dev MUST revert if there is a challenge for the safe
    /// @dev MUST set challenge_start_time to the current block time
    /// @dev MUST emit the ChallengeStarted event
    /// @param _safe The Safe to challenge
    function challenge(address _safe) external {
        ModuleConfig storage config = safeConfigs[_safe];

        if (config.fallbackOwner == address(0)) {
            revert LivenessModule2_ModuleNotConfigured();
        }

        // Check that the module is still enabled on the target Safe
        Safe safe = Safe(payable(_safe));
        if (!safe.isModuleEnabled(address(this))) {
            revert LivenessModule2_ModuleNotEnabled();
        }

        if (msg.sender != config.fallbackOwner) {
            revert LivenessModule2_UnauthorizedCaller();
        }

        if (challengeStartTime[_safe] != 0) {
            revert LivenessModule2_ChallengeAlreadyExists();
        }

        challengeStartTime[_safe] = block.timestamp;
        emit ChallengeStarted(_safe, block.timestamp);
    }

    /// @notice Responds to a challenge for an enabled safe, canceling it
    /// @dev MUST only be executable by an enabled safe
    /// @dev MUST revert if there isn't a challenge for the calling safe
    /// @dev MUST revert if there is a challenge for the calling safe but the response period has expired
    /// @dev MUST emit the ChallengeCancelled event
    function respond() external {
        // Check that this module is enabled on the calling Safe
        Safe safe = Safe(payable(msg.sender));
        if (!safe.isModuleEnabled(address(this))) {
            revert LivenessModule2_ModuleNotEnabled();
        }

        ModuleConfig storage config = safeConfigs[msg.sender];

        // Check if the calling safe has configuration set
        if (config.fallbackOwner == address(0)) {
            revert LivenessModule2_ModuleNotConfigured();
        }

        uint256 startTime = challengeStartTime[msg.sender];
        if (startTime == 0) {
            revert LivenessModule2_ChallengeDoesNotExist();
        }

        // Check if response period has expired
        if (block.timestamp >= startTime + config.livenessResponsePeriod) {
            revert LivenessModule2_ResponsePeriodEnded();
        }

        delete challengeStartTime[msg.sender];
        emit ChallengeCancelled(msg.sender);
    }

    /// @notice With a successful challenge, removes all current owners from an enabled safe, appoints fallback as its
    /// sole owner, and sets its quorum to 1
    /// @dev MUST be executable by anyone
    /// @dev MUST revert if the given safe hasn't enabled the module
    /// @dev MUST revert if there isn't a successful challenge for the given safe
    /// @dev MUST enable the module to start a new challenge
    /// @dev MUST emit the ChallengeExecuted event
    /// @param _safe The Safe to transfer ownership of
    function changeOwnershipToFallback(address _safe) external {
        ModuleConfig storage config = safeConfigs[_safe];

        if (config.fallbackOwner == address(0)) {
            revert LivenessModule2_ModuleNotConfigured();
        }

        // Check that the module is still enabled on the target Safe
        Safe safe = Safe(payable(_safe));
        if (!safe.isModuleEnabled(address(this))) {
            revert LivenessModule2_ModuleNotEnabled();
        }

        uint256 startTime = challengeStartTime[_safe];
        if (startTime == 0) {
            revert LivenessModule2_ChallengeDoesNotExist();
        }

        // Check if response period has expired
        if (block.timestamp < startTime + config.livenessResponsePeriod) {
            revert LivenessModule2_ResponsePeriodActive();
        }

        Safe targetSafe = Safe(payable(_safe));

        // Get current owners
        address[] memory owners = targetSafe.getOwners();

        // Remove all owners after the first one
        while (owners.length > 1) {
            // removeOwner automatically updates the threshold, so we don't need to do it manually
            targetSafe.execTransactionFromModule({
                to: _safe,
                value: 0,
                operation: Enum.Operation.Call,
                data: abi.encodeCall(OwnerManager.removeOwner, (SENTINEL_OWNER, owners[0], 1))
            });
            // Get updated owners list after removal
            owners = targetSafe.getOwners();
        }

        // Now swap the remaining single owner with the fallback owner
        targetSafe.execTransactionFromModule({
            to: _safe,
            value: 0,
            operation: Enum.Operation.Call,
            data: abi.encodeCall(OwnerManager.swapOwner, (SENTINEL_OWNER, owners[0], config.fallbackOwner))
        });

        // Reset the challenge state to allow a new challenge
        delete challengeStartTime[_safe];

        emit ChallengeExecuted(_safe, config.fallbackOwner);
    }
}

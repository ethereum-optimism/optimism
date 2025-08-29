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
    /// @notice Configuration for a Safe's liveness module
    struct SafeConfig {
        uint256 livenessResponsePeriod;
        address fallbackOwner;
        uint256 challengeStartTime;
    }

    /// @notice Mapping from Safe address to its configuration
    mapping(address => SafeConfig) public safeConfigs;

    /// @notice Semantic version.
    /// @custom:semver 2.0.0
    string public constant version = "2.0.0";

    /// @notice Enables the module by the multisig to be challenged and sets the liveness_response_period and
    /// fallback_owner
    /// @dev MUST set the caller as a safe
    /// @dev MUST take as parameters liveness_response_period and fallback_owner and store them as related to the safe
    /// @dev MUST accept an arbitrary number of independent safe contracts to enable the module
    /// @param _livenessResponsePeriod The period in seconds for a liveness response
    /// @param _fallbackOwner The address that will become owner if challenge succeeds
    function enableModule(uint256 _livenessResponsePeriod, address _fallbackOwner) external {
        if (_livenessResponsePeriod == 0 || _fallbackOwner == address(0)) {
            revert LivenessModule2_InvalidParameters();
        }

        // Set the caller as a safe and store its configuration
        SafeConfig storage config = safeConfigs[msg.sender];

        // Store the parameters related to this safe
        config.livenessResponsePeriod = _livenessResponsePeriod;
        config.fallbackOwner = _fallbackOwner;
        config.challengeStartTime = 0;

        emit ModuleEnabled(msg.sender, _livenessResponsePeriod, _fallbackOwner);
    }

    /// @notice Disables the module by an enabled safe
    /// @dev MUST only be executable by an enabled safe
    /// @dev MUST erase the existing liveness_response_period and fallback_owner data related to the calling safe
    /// @dev Note: Disabling the module also cancels any ongoing challenges
    function disableModule() external {
        SafeConfig storage config = safeConfigs[msg.sender];
        // Check if the calling safe has the module enabled
        if (config.fallbackOwner == address(0)) {
            revert LivenessModule2_ModuleNotEnabled();
        }

        // Erase the configuration data for this safe
        delete safeConfigs[msg.sender];
        emit ModuleDisabled(msg.sender);
    }

    /// @notice Returns the liveness_response_period and fallback_owner for a given safe
    /// @dev MUST never revert
    /// @param _safe The Safe address to query
    /// @return livenessResponsePeriod The response period
    /// @return fallbackOwner The fallback owner address
    function viewConfiguration(address _safe) external view returns (uint256, address) {
        SafeConfig storage config = safeConfigs[_safe];
        return (config.livenessResponsePeriod, config.fallbackOwner);
    }

    /// @notice Returns challenge_start_time + liveness_response_period if there is a challenge for the given safe, or
    /// 0 if not
    /// @dev MUST never revert
    /// @param _safe The Safe address to query
    /// @return The challenge end timestamp, or 0 if no challenge
    function isChallenged(address _safe) external view returns (uint256) {
        SafeConfig storage config = safeConfigs[_safe];
        if (config.challengeStartTime == 0) {
            return 0;
        }
        return config.challengeStartTime + config.livenessResponsePeriod;
    }

    /// @notice Challenges an enabled safe
    /// @dev MUST only be executable by fallback owner of the challenged safe
    /// @dev MUST revert if there is a challenge for the safe
    /// @dev MUST set challenge_start_time to the current block time
    /// @dev MUST emit the ChallengeStarted event
    /// @param _safe The Safe to challenge
    function startChallenge(address _safe) external {
        SafeConfig storage config = safeConfigs[_safe];

        if (config.fallbackOwner == address(0)) {
            revert LivenessModule2_ModuleNotEnabled();
        }

        if (msg.sender != config.fallbackOwner) {
            revert LivenessModule2_UnauthorizedCaller();
        }

        if (config.challengeStartTime != 0) {
            revert LivenessModule2_ChallengeAlreadyExists();
        }

        config.challengeStartTime = block.timestamp;
        emit ChallengeStarted(_safe, block.timestamp);
    }

    /// @notice Cancels a challenge for an enabled safe
    /// @dev MUST only be executable by an enabled safe
    /// @dev MUST revert if there isn't a challenge for the calling safe
    /// @dev MUST revert if there is a challenge for the calling safe but the challenge is successful
    /// @dev MUST emit the ChallengeCancelled event
    function cancelChallenge() external {
        SafeConfig storage config = safeConfigs[msg.sender];

        if (config.fallbackOwner == address(0)) {
            revert LivenessModule2_ModuleNotEnabled();
        }

        if (config.challengeStartTime == 0) {
            revert LivenessModule2_ChallengeDoesNotExist();
        }

        // Check if response period has expired
        if (block.timestamp >= config.challengeStartTime + config.livenessResponsePeriod) {
            revert LivenessModule2_ResponsePeriodExpired();
        }

        config.challengeStartTime = 0;
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
        SafeConfig storage config = safeConfigs[_safe];

        if (config.fallbackOwner == address(0)) {
            revert LivenessModule2_ModuleNotEnabled();
        }

        if (config.challengeStartTime == 0) {
            revert LivenessModule2_ChallengeDoesNotExist();
        }

        // Check if response period has expired
        if (block.timestamp < config.challengeStartTime + config.livenessResponsePeriod) {
            revert LivenessModule2_ResponsePeriodActive();
        }

        Safe safe = Safe(payable(_safe));

        // Get current owners
        address[] memory currentOwners = safe.getOwners();

        if (currentOwners.length > 0) {
            // Remove all owners except the first one from the front
            // removeOwner automatically updates the threshold, so we don't need to do it manually
            address sentinel = address(0x1); // Sentinel value for the first owner
            
            for (uint256 i = currentOwners.length - 1; i > 0; i--) {
                address ownerToRemove = currentOwners[0]; // Always remove the first owner
                safe.execTransactionFromModule({
                    to: _safe,
                    value: 0,
                    operation: Enum.Operation.Call,
                    data: abi.encodeCall(OwnerManager.removeOwner, (sentinel, ownerToRemove, 1))
                });
                // Get updated owners list after removal
                currentOwners = safe.getOwners();
            }

            // Now swap the remaining single owner with the fallback owner
            safe.execTransactionFromModule({
                to: _safe,
                value: 0,
                operation: Enum.Operation.Call,
                data: abi.encodeCall(OwnerManager.swapOwner, (sentinel, currentOwners[0], config.fallbackOwner))
            });
        }

        // Reset the challenge state to allow a new challenge
        config.challengeStartTime = 0;

        emit ChallengeExecuted(_safe, config.fallbackOwner);
    }

}

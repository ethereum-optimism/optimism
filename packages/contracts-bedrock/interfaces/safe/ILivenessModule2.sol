// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title ILivenessModule2
/// @notice Interface for LivenessModule2, a singleton module for challenge-based ownership transfer
interface ILivenessModule2 is ISemver {

    /// @notice Error for when module is not enabled for the Safe
    error LivenessModule2_ModuleNotEnabled();

    /// @notice Error for when module is already enabled for the Safe
    error LivenessModule2_ModuleAlreadyEnabled();

    /// @notice Error for when a challenge already exists
    error LivenessModule2_ChallengeAlreadyExists();

    /// @notice Error for when no challenge exists
    error LivenessModule2_ChallengeDoesNotExist();

    /// @notice Error for when challenge is not successful
    error LivenessModule2_ChallengeNotSuccessful();

    /// @notice Error for when caller is not authorized
    error LivenessModule2_UnauthorizedCaller();

    /// @notice Error for invalid parameters
    error LivenessModule2_InvalidParameters();

    /// @notice Error for when an owner is not found in the Safe's owner list
    error LivenessModule2_OwnerNotFound();

    /// @notice Emitted when a Safe enables the module
    event ModuleEnabled(address indexed safe, uint256 livenessChallengePeriod, address fallbackOwner);

    /// @notice Emitted when a Safe disables the module
    event ModuleDisabled(address indexed safe);

    /// @notice Emitted when a challenge is started
    event ChallengeStarted(address indexed safe, uint256 challengeStartTime);

    /// @notice Emitted when a challenge is cancelled
    event ChallengeCancelled(address indexed safe);

    /// @notice Emitted when ownership is transferred to the fallback owner
    event ChallengeExecuted(address indexed safe, address fallbackOwner);

    /// @notice Returns the configuration for a Safe
    /// @return livenessChallengePeriod The challenge period
    /// @return fallbackOwner The fallback owner address
    /// @return challengeStartTime The challenge start time (0 if no challenge)
    function safeConfigs(address) external view returns (uint256 livenessChallengePeriod, address fallbackOwner, uint256 challengeStartTime);

    /// @notice Semantic version
    /// @return version The contract version
    function version() external view returns (string memory);

    /// @notice Enables the module by the multisig to be challenged
    /// @param _livenessChallengePeriod The period in seconds for a liveness challenge
    /// @param _fallbackOwner The address that will become owner if challenge succeeds
    function enableModule(uint256 _livenessChallengePeriod, address _fallbackOwner) external;

    /// @notice Disables the module by an enabled safe
    function disableModule() external;

    /// @notice Returns the liveness_challenge_period and fallback_owner for a given safe
    /// @param _safe The Safe address to query
    /// @return livenessChallengePeriod The challenge period
    /// @return fallbackOwner The fallback owner address
    function viewConfiguration(address _safe) external view returns (uint256, address);

    /// @notice Returns challenge_start_time + liveness_challenge_period if there is a challenge, or 0 if not
    /// @param _safe The Safe address to query
    /// @return The challenge end timestamp, or 0 if no challenge
    function isChallenged(address _safe) external view returns (uint256);

    /// @notice Challenges an enabled safe
    /// @param _safe The Safe to challenge
    function startChallenge(address _safe) external;

    /// @notice Cancels a challenge for an enabled safe
    function cancelChallenge() external;

    /// @notice Removes all current owners from an enabled safe and appoints fallback as sole owner
    /// @param _safe The Safe to transfer ownership of
    function changeOwnershipToFallback(address _safe) external;
}

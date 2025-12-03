// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface ISafe { }

/// @title ILivenessModule2
/// @notice Interface for LivenessModule2, a singleton module for challenge-based ownership transfer
interface ILivenessModule2 {
    /// @notice Configuration for a Safe's liveness module
    struct ModuleConfig {
        uint256 livenessResponsePeriod;
        address fallbackOwner;
    }

    event ChallengeCancelled(address indexed safe);
    event ChallengeStarted(address indexed safe, uint256 challengeStartTime);
    event ChallengeSucceeded(address indexed safe, address fallbackOwner);
    event ModuleCleared(address indexed safe);
    event ModuleConfigured(address indexed safe, uint256 livenessResponsePeriod, address fallbackOwner);

    error LivenessModule2_ChallengeAlreadyExists();
    error LivenessModule2_ChallengeDoesNotExist();
    error LivenessModule2_InvalidFallbackOwner();
    error LivenessModule2_InvalidResponsePeriod();
    error LivenessModule2_InvalidVersion();
    error LivenessModule2_ModuleNotConfigured();
    error LivenessModule2_ModuleNotEnabled();
    error LivenessModule2_ModuleStillEnabled();
    error LivenessModule2_OwnershipTransferFailed();
    error LivenessModule2_ResponsePeriodActive();
    error LivenessModule2_ResponsePeriodEnded();
    error LivenessModule2_UnauthorizedCaller();
    error SemverComp_InvalidSemverParts();

    /// @notice Returns the configuration for a Safe
    /// @param _safe The Safe to query
    /// @return The ModuleConfig for the Safe
    function livenessSafeConfiguration(ISafe _safe) external view returns (ModuleConfig memory);

    /// @notice Returns the challenge start time for a Safe (0 if no challenge)
    /// @return The challenge start timestamp
    function challengeStartTime(ISafe) external view returns (uint256);

    /// @notice Configures the module for a Safe that has already enabled it
    /// @param _config The configuration parameters for the module
    function configureLivenessModule(ModuleConfig memory _config) external;

    /// @notice Clears the module configuration for a Safe
    function clearLivenessModule() external;

    /// @notice Returns challenge_start_time + liveness_response_period if there is a challenge, or 0 if not
    /// @param _safe The Safe address to query
    /// @return The challenge end timestamp, or 0 if no challenge
    function getChallengePeriodEnd(ISafe _safe) external view returns (uint256);

    /// @notice Challenges an enabled safe
    /// @param _safe The Safe to challenge
    function challenge(ISafe _safe) external;

    /// @notice Responds to a challenge for an enabled safe, canceling it
    function respond() external;

    /// @notice Removes all current owners from an enabled safe and appoints fallback as sole owner
    /// @param _safe The Safe to transfer ownership of
    function changeOwnershipToFallback(ISafe _safe) external;
}

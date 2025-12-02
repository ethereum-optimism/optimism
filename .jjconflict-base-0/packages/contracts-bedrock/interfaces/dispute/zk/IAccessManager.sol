// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IAccessManager
/// @notice Interface for the AccessManager contract that manages permissions for dispute game proposers and challengers.
interface IAccessManager {
    /// @notice Event emitted when proposer permissions are updated.
    event ProposerPermissionUpdated(address indexed proposer, bool allowed);

    /// @notice Event emitted when challenger permissions are updated.
    event ChallengerPermissionUpdated(address indexed challenger, bool allowed);

    function proposers(address) external view returns (bool);
    function challengers(address) external view returns (bool);
    function FALLBACK_TIMEOUT() external view returns (uint256);
    function DISPUTE_GAME_FACTORY() external view returns (address);
    function DEPLOYMENT_TIMESTAMP() external view returns (uint256);

    function setProposer(address _proposer, bool _allowed) external;
    function setChallenger(address _challenger, bool _allowed) external;
    function getLastProposalTimestamp() external view returns (uint256);
    function isAllowedProposer(address _proposer) external view returns (bool allowed_);
    function isAllowedChallenger(address _challenger) external view returns (bool allowed_);
    function isProposalPermissionlessMode() external view returns (bool);
}

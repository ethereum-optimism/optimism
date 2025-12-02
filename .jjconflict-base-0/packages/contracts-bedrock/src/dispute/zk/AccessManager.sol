// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { GameType, GameTypes } from "src/dispute/lib/Types.sol";
import { Timestamp } from "src/dispute/lib/LibUDT.sol";

/// @title AccessManager
/// @notice Manages permissions for dispute game proposers and challengers.
contract AccessManager is Ownable {
    ////////////////////////////////////////////////////////////////
    //                         Events                             //
    ////////////////////////////////////////////////////////////////

    /// @notice Event emitted when proposer permissions are updated.
    event ProposerPermissionUpdated(address indexed proposer, bool allowed);

    /// @notice Event emitted when challenger permissions are updated.
    event ChallengerPermissionUpdated(address indexed challenger, bool allowed);

    ////////////////////////////////////////////////////////////////
    //                         State Vars                         //
    ////////////////////////////////////////////////////////////////

    /// @notice Tracks whitelisted proposers.
    mapping(address => bool) public proposers;

    /// @notice Tracks whitelisted challengers.
    mapping(address => bool) public challengers;

    /// @notice The timeout (in seconds) after which permissionless proposing is allowed (immutable).
    // nosemgrep: sol-safety-no-immutable-variables
    uint256 public immutable FALLBACK_TIMEOUT;

    /// @notice The dispute game factory address.
    // nosemgrep: sol-safety-no-immutable-variables
    IDisputeGameFactory public immutable DISPUTE_GAME_FACTORY;

    /// @notice The timestamp of this contract's creation. Used for permissionless fallback proposals.
    // nosemgrep: sol-safety-no-immutable-variables
    uint256 public immutable DEPLOYMENT_TIMESTAMP;

    ////////////////////////////////////////////////////////////////
    //                      Constructor                           //
    ////////////////////////////////////////////////////////////////

    /// @notice Constructor sets the fallback timeout and initializes timestamp.
    /// @param _fallbackTimeout The timeout in seconds after last proposal when permissionless mode activates.
    /// @param _disputeGameFactory The dispute game factory address.
    constructor(uint256 _fallbackTimeout, IDisputeGameFactory _disputeGameFactory) {
        FALLBACK_TIMEOUT = _fallbackTimeout;
        DISPUTE_GAME_FACTORY = _disputeGameFactory;
        DEPLOYMENT_TIMESTAMP = block.timestamp;
    }

    ////////////////////////////////////////////////////////////////
    //                      Functions                             //
    ////////////////////////////////////////////////////////////////

    /// @notice Allows the owner to whitelist or un-whitelist proposers.
    /// @param _proposer The address to set in the proposers mapping.
    /// @param _allowed True if whitelisting, false otherwise.
    function setProposer(address _proposer, bool _allowed) external onlyOwner {
        proposers[_proposer] = _allowed;
        emit ProposerPermissionUpdated(_proposer, _allowed);
    }

    /// @notice Allows the owner to whitelist or un-whitelist challengers.
    /// @param _challenger The address to set in the challengers mapping.
    /// @param _allowed True if whitelisting, false otherwise.
    function setChallenger(address _challenger, bool _allowed) external onlyOwner {
        challengers[_challenger] = _allowed;
        emit ChallengerPermissionUpdated(_challenger, _allowed);
    }

    /// @notice Returns the last proposal timestamp.
    /// @return The last proposal timestamp.
    function getLastProposalTimestamp() public view returns (uint256) {
        // Get the latest game to check its timestamp.
        GameType gameType = GameTypes.OPTIMISTIC_ZK_GAME_TYPE;
        uint256 numGames = DISPUTE_GAME_FACTORY.gameCount();

        // Early return if no games exist.
        if (numGames == 0) {
            return DEPLOYMENT_TIMESTAMP;
        }

        // Iterate backwards through games to find the most recent of our type.
        // This avoids the memory allocation of findLatestGames for a single result.
        uint256 i = numGames - 1;
        while (true) {
            (GameType gameTypeAtIndex, Timestamp timestamp,) = DISPUTE_GAME_FACTORY.gameAtIndex(i);

            // If we found a game of the correct type, return its timestamp.
            if (gameTypeAtIndex.raw() == gameType.raw()) {
                return uint256(timestamp.raw());
            }

            // If we've reached index 0, break out of the loop
            if (i == 0) {
                break;
            }

            unchecked {
                --i;
            }
        }

        // If we've checked all games without finding our type, return deployment timestamp.
        return DEPLOYMENT_TIMESTAMP;
    }

    /// @notice Checks if an address is allowed to propose.
    /// @param _proposer The address to check.
    /// @return allowed_ Whether the address is allowed to propose.
    function isAllowedProposer(address _proposer) external view returns (bool allowed_) {
        // If address(0) is allowed, then it's permissionless.
        // If the fallback timeout has elapsed since last proposal, anyone can propose.

        uint256 lastProposalTimestamp = getLastProposalTimestamp();

        allowed_ = proposers[address(0)] || proposers[_proposer]
            || (block.timestamp - lastProposalTimestamp > FALLBACK_TIMEOUT);
    }

    /// @notice Checks if an address is allowed to challenge.
    /// @param _challenger The address to check.
    /// @return allowed_ Whether the address is allowed to challenge.
    function isAllowedChallenger(address _challenger) external view returns (bool allowed_) {
        // If address(0) is allowed, then it's permissionless.
        allowed_ = challengers[address(0)] || challengers[_challenger];
    }

    /// @notice Returns whether proposal fallback timeout has elapsed.
    /// @return Whether permissionless proposing is active.
    function isProposalPermissionlessMode() external view returns (bool) {
        uint256 lastProposalTimestamp = getLastProposalTimestamp();
        return block.timestamp - lastProposalTimestamp > FALLBACK_TIMEOUT || proposers[address(0)];
    }
}

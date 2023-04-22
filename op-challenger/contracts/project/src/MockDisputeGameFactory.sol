// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import { GameType, Claim } from "./Types.sol";
import { IDisputeGame } from "./IDisputeGame.sol";
import { MockAttestationDisputeGame } from "./MockAttestationDisputeGame.sol";

/// @title MockDisputeGameFactory
/// @dev Tests the `op-challenger` on a local devnet.
contract MockDisputeGameFactory {
    mapping(bytes32 => IDisputeGame) internal innerGames;

    event DisputeGameCreated(address indexed disputeProxy, GameType indexed gameType, Claim indexed rootClaim);

    /// @notice `games` is a mapping that maps the hash of `gameType ++ rootClaim ++ extraData` to the deployed
    ///         `DisputeGame` clone.
    /// @dev `++` equates to concatenation.
    /// @param gameType The type of the DisputeGame - used to decide the proxy implementation
    /// @param rootClaim The root claim of the DisputeGame.
    /// @param extraData Any extra data that should be provided to the created dispute game.
    /// @return _proxy The clone of the `DisputeGame` created with the given parameters. address(0) if nonexistent.
    function games(GameType gameType, Claim rootClaim, bytes calldata extraData)
        external
        view
        returns (IDisputeGame _proxy) {
        _proxy = innerGames[keccak256(abi.encodePacked(gameType, rootClaim, extraData))];
        }

    /// @notice Creates a new DisputeGame proxy contract.
    /// @param gameType The type of the DisputeGame - used to decide the proxy implementation
    /// @param rootClaim The root claim of the DisputeGame.
    /// @param extraData Any extra data that should be provided to the created dispute game.
    function create(GameType gameType, Claim rootClaim, bytes calldata extraData) external returns (MockAttestationDisputeGame mock) {
        uint256 l2BlockNumber = abi.decode(extraData, (uint256));
        mock = new MockAttestationDisputeGame(rootClaim, l2BlockNumber, msg.sender);
        emit DisputeGameCreated(address(mock), gameType, rootClaim);
        innerGames[keccak256(abi.encodePacked(gameType, rootClaim, extraData))] = IDisputeGame(address(mock));
    }
}
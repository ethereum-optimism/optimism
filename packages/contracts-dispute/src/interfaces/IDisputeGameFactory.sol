// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

import { Claim } from "src/types/Types.sol";
import { GameType } from "src/types/Types.sol";

import { IDisputeGame } from "src/interfaces/IDisputeGame.sol";

/// @title IDisputeGameFactory
/// @author clabby <https://github.com/clabby>
/// @notice The interface for a DisputeGameFactory contract.
interface IDisputeGameFactory {
    /// @notice Emitted when a new dispute game is created
    /// @param disputeProxy The address of the dispute game proxy
    /// @param gameType The type of the dispute game proxy's implementation
    /// @param rootClaim The root claim of the dispute game
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
        returns (IDisputeGame _proxy);

    /// @notice Gets the `IDisputeGame` for a given `GameType`.
    /// @param gameType The type of the dispute game.
    /// @return _impl The address of the implementation of the game type. Will be cloned on creation.
    function getImplementation(GameType gameType) external view returns (IDisputeGame _impl);

    /// @notice The owner of the contract.
    /// @dev Owner Permissions:
    ///      - Update the implementation contracts for a given game type.
    /// @return _owner The owner of the contract.
    function owner() external view returns (address _owner);

    /// @notice Creates a new DisputeGame proxy contract.
    /// @param gameType The type of the DisputeGame - used to decide the proxy implementation
    /// @param rootClaim The root claim of the DisputeGame.
    /// @param extraData Any extra data that should be provided to the created dispute game.
    function create(GameType gameType, Claim rootClaim, bytes calldata extraData)
        external
        returns (IDisputeGame proxy);

    /// @notice Sets the implementation contract for a specific `GameType`
    /// @param gameType The type of the DisputeGame
    /// @param impl The implementation contract for the given `GameType`
    function setImplementation(GameType gameType, IDisputeGame impl) external;
}

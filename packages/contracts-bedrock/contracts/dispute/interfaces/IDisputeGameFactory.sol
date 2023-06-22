// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "../../libraries/DisputeTypes.sol";

import { IDisputeGame } from "./IDisputeGame.sol";

/// @title IDisputeGameFactory
/// @notice The interface for a DisputeGameFactory contract.
interface IDisputeGameFactory {
    /// @notice Emitted when a new dispute game is created
    /// @param disputeProxy The address of the dispute game proxy
    /// @param gameType The type of the dispute game proxy's implementation
    /// @param rootClaim The root claim of the dispute game
    event DisputeGameCreated(
        address indexed disputeProxy,
        GameType indexed gameType,
        Claim indexed rootClaim
    );

    /// @notice Emitted when a new game implementation added to the factory
    /// @param impl The implementation contract for the given `GameType`.
    /// @param gameType The type of the DisputeGame.
    event ImplementationSet(address indexed impl, GameType indexed gameType);

    /// @notice The total number of dispute games created by this factory.
    /// @return _gameCount The total number of dispute games created by this factory.
    function gameCount() external view returns (uint256 _gameCount);

    /// @notice `games` queries an internal mapping that maps the hash of
    ///         `gameType ++ rootClaim ++ extraData` to the deployed `DisputeGame` clone.
    /// @dev `++` equates to concatenation.
    /// @param gameType The type of the DisputeGame - used to decide the proxy implementation
    /// @param rootClaim The root claim of the DisputeGame.
    /// @param extraData Any extra data that should be provided to the created dispute game.
    /// @return _proxy The clone of the `DisputeGame` created with the given parameters.
    ///         Returns `address(0)` if nonexistent.
    /// @return _timestamp The timestamp of the creation of the dispute game.
    function games(
        GameType gameType,
        Claim rootClaim,
        bytes calldata extraData
    ) external view returns (IDisputeGame _proxy, uint256 _timestamp);

    /// @notice `gameAtIndex` returns the dispute game contract address and its creation timestamp
    ///          at the given index. Each created dispute game increments the underlying index.
    /// @param _index The index of the dispute game.
    /// @return _proxy The clone of the `DisputeGame` created with the given parameters.
    ///         Returns `address(0)` if nonexistent.
    /// @return _timestamp The timestamp of the creation of the dispute game.
    function gameAtIndex(uint256 _index)
        external
        view
        returns (IDisputeGame _proxy, uint256 _timestamp);

    /// @notice `gameImpls` is a mapping that maps `GameType`s to their respective
    ///         `IDisputeGame` implementations.
    /// @param gameType The type of the dispute game.
    /// @return _impl The address of the implementation of the game type.
    ///         Will be cloned on creation of a new dispute game with the given `gameType`.
    function gameImpls(GameType gameType) external view returns (IDisputeGame _impl);

    /// @notice Creates a new DisputeGame proxy contract.
    /// @param gameType The type of the DisputeGame - used to decide the proxy implementation.
    /// @param rootClaim The root claim of the DisputeGame.
    /// @param extraData Any extra data that should be provided to the created dispute game.
    /// @return proxy The address of the created DisputeGame proxy.
    function create(
        GameType gameType,
        Claim rootClaim,
        bytes calldata extraData
    ) external returns (IDisputeGame proxy);

    /// @notice Sets the implementation contract for a specific `GameType`.
    /// @dev May only be called by the `owner`.
    /// @param gameType The type of the DisputeGame.
    /// @param impl The implementation contract for the given `GameType`.
    function setImplementation(GameType gameType, IDisputeGame impl) external;

    /// @notice Returns a unique identifier for the given dispute game parameters.
    /// @dev Hashes the concatenation of `gameType . rootClaim . extraData`
    ///      without expanding memory.
    /// @param gameType The type of the DisputeGame.
    /// @param rootClaim The root claim of the DisputeGame.
    /// @param extraData Any extra data that should be provided to the created dispute game.
    /// @return _uuid The unique identifier for the given dispute game parameters.
    function getGameUUID(
        GameType gameType,
        Claim rootClaim,
        bytes memory extraData
    ) external pure returns (Hash _uuid);
}

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import { Claim } from "src/types/Types.sol";
import { GameType } from "src/types/Types.sol";
import { Owner } from "src/util/Owner.sol";
import { Clone } from "src/util/Clone.sol";
import { Initializable } from "src/util/Initializable.sol";
import { IDisputeGame } from "src/interfaces/IDisputeGame.sol";
import { IDisputeGameFactory } from "src/interfaces/IDisputeGameFactory.sol";
import { ClonesWithImmutableArgs } from "cwia/ClonesWithImmutableArgs.sol";

/// @title DisputeGameFactory
/// @author refcell <https://github.com/refcell>
/// @notice A factory contract for creating [`DisputeGame`] contracts.
contract DisputeGameFactory is IDisputeGameFactory, Owner {
    using ClonesWithImmutableArgs for address;

    /// @notice Mapping of GameType to the `DisputeGame` proxy contract.
    /// @dev The GameType id is computed as the hash of `gameType . rootClaim . extraData`.
    mapping(GameType => IDisputeGame) internal disputeGames;

    /// @notice Constructs a new DisputeGameFactory contract.
    /// @param _owner The owner of the contract.
    constructor(address _owner) Owner(_owner) { }

    /// @notice Retrieves the hash of `gameType . rootClaim . extraData` to the deployed `DisputeGame` clone.
    /// @dev Note: `.` denotes concatenation.
    /// @param gameType The type of the DisputeGame - used to decide the proxy implementation
    /// @param rootClaim The root claim of the DisputeGame.
    /// @param extraData Any extra data that should be provided to the created dispute game.
    /// @return _proxy The clone of the `DisputeGame` created with the given parameters. address(0) if nonexistent.
    function games(GameType gameType, Claim rootClaim, bytes calldata extraData)
        external
        view
        returns (IDisputeGame _proxy)
    {
        return disputeGames[GameType.wrap(getGameID(gameType, rootClaim, extraData))];
    }

    /// @notice The owner of the contract.
    /// @notice The owner can update the implementation contracts for a given GameType.
    /// @return _owner The owner of the contract.
    function owner() external view returns (address _owner) {
        return _owner;
    }

    /// @notice Returns a game id for the given dispute game parameters.
    function getGameID(GameType gameType, Claim rootClaim, bytes calldata extraData) public pure returns (bytes32) {
        return keccak256(abi.encode(gameType, rootClaim, extraData));
    }

    /// @notice Gets the `IDisputeGame` for a given `GameType`.
    /// @dev Notice, we can just use the `games` mapping to get the implementation.
    /// @dev This works since clones are mapped using a hash of `gameType . rootClaim . extraData`.
    /// @param gameType The type of the dispute game.
    /// @return _impl The address of the implementation of the game type. Will be cloned on creation.
    function getImplementation(GameType gameType) external view returns (IDisputeGame _impl) {
        return disputeGames[gameType];
    }

    /// @notice Creates a new DisputeGame proxy contract.
    /// @notice If a dispute game with the given parameters already exists, it will be returned.
    /// @param gameType The type of the DisputeGame - used to decide the proxy implementation
    /// @param rootClaim The root claim of the DisputeGame.
    /// @param extraData Any extra data that should be provided to the created dispute game.
    /// @return proxy The clone of the `DisputeGame` created with the given parameters.
    function create(GameType gameType, Claim rootClaim, bytes calldata extraData)
        external
        returns (IDisputeGame proxy)
    {
        bytes32 gameID = getGameID(gameType, rootClaim, extraData);
        GameType id = GameType.wrap(gameID);
        proxy = disputeGames[id];
        if (address(proxy) == address(0)) {
            IDisputeGame impl = disputeGames[gameType];
            bytes memory data = abi.encodePacked(gameType, rootClaim, extraData);
            proxy = IDisputeGame(address(impl).clone(data));
            proxy.initialize();
            disputeGames[id] = proxy;
            emit DisputeGameCreated(address(proxy), gameType, rootClaim);
        }
        return proxy;
    }

    /// @notice Sets the implementation contract for a specific `GameType`
    /// @param gameType The type of the DisputeGame
    /// @param impl The implementation contract for the given `GameType`
    function setImplementation(GameType gameType, IDisputeGame impl) external onlyOwner {
        disputeGames[gameType] = impl;
    }
}

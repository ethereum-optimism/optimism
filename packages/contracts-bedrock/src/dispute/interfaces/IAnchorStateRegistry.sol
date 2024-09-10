// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IFaultDisputeGame } from "src/dispute/interfaces/IFaultDisputeGame.sol";
import { IDisputeGameFactory } from "src/dispute/interfaces/IDisputeGameFactory.sol";

import "src/dispute/lib/Types.sol";

/// @title IAnchorStateRegistry
/// @notice Describes a contract that stores the anchor state for each game type.
interface IAnchorStateRegistry {
    /// @notice Returns the anchor state for the given game type.
    /// @param _gameType The game type to get the anchor state for.
    /// @return The anchor state for the given game type.
    function anchors(GameType _gameType) external view returns (Hash, uint256);

    /// @notice Returns the DisputeGameFactory address.
    /// @return DisputeGameFactory address.
    function disputeGameFactory() external view returns (IDisputeGameFactory);

    /// @notice Callable by FaultDisputeGame contracts to update the anchor state. Pulls the anchor state directly from
    ///         the FaultDisputeGame contract and stores it in the registry if the new anchor state is valid and the
    ///         state is newer than the current anchor state.
    function tryUpdateAnchorState() external;

    /// @notice Sets the anchor state given the game.
    /// @param _game The game to set the anchor state for.
    function setAnchorState(IFaultDisputeGame _game) external;
}

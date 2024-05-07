// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IDisputeGameFactory, IDisputeGame } from "src/dispute/interfaces/IDisputeGameFactory.sol";

import "src/dispute/lib/Types.sol";

/// @title IAnchorStateRegistry
/// @notice Describes a contract that stores the anchor state for each game type.
interface IAnchorStateRegistry {
    /// @notice Returns the anchor state for the given game type.
    /// @param _gameType The game type to get the anchor state for.
    /// @return outputRoot_ The output root of the anchor state for the given game type.
    /// @return blockNumber_ The L2 block number at which the output root was generated.
    function anchors(GameType _gameType) external view returns (Hash outputRoot_, uint256 blockNumber_);

    /// @notice Returns true if the passed dispute game has been verified.
    /// @param _disputeGame The game type to check.
    /// @return isVerified_ True if the dispute game has been verified.
    function verifiedGames(IDisputeGame _disputeGame) external view returns (bool isVerified_);

    /// @notice Returns the DisputeGameFactory address.
    /// @return factory_ DisputeGameFactory address.
    function disputeGameFactory() external view returns (IDisputeGameFactory factory_);
}

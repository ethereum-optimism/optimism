// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "src/libraries/DisputeTypes.sol";
import "src/dispute/interfaces/IDisputeGame.sol";

/// @title LibGameId
/// @notice Utility functions for packing and unpacking GameIds.
library LibGameId {
    /// @notice Packs values into a 32 byte GameId type.
    /// @param _gameType The game type.
    /// @param _timestamp The timestamp of the game's creation.
    /// @param _gameProxy The game proxy address.
    /// @return gameId_ The packed GameId.
    function pack(
        GameType _gameType,
        Timestamp _timestamp,
        IDisputeGame _gameProxy
    )
        internal
        pure
        returns (GameId gameId_)
    {
        assembly {
            gameId_ := or(or(shl(248, _gameType), shl(184, _timestamp)), _gameProxy)
        }
    }

    /// @notice Unpacks values from a 32 byte GameId type.
    /// @param _gameId The packed GameId.
    /// @return gameType_ The game type.
    /// @return timestamp_ The timestamp of the game's creation.
    /// @return gameProxy_ The game proxy address.
    function unpack(GameId _gameId)
        internal
        pure
        returns (GameType gameType_, Timestamp timestamp_, IDisputeGame gameProxy_)
    {
        assembly {
            gameType_ := shr(248, _gameId)
            timestamp_ := shr(184, and(_gameId, not(shl(248, 0xff))))
            gameProxy_ := and(_gameId, 0xffffffffffffffffffffffffffffffffffffffff)
        }
    }
}

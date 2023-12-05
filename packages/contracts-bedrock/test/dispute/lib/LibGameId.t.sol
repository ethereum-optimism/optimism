// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { Test } from "forge-std/Test.sol";

import { LibGameId } from "src/dispute/lib/LibGameId.sol";
import { IDisputeGame } from "src/dispute/interfaces/IDisputeGame.sol";
import "src/libraries/DisputeTypes.sol";

contract LibGameId_Test is Test {
    /// @dev Tests that a round trip of packing and unpacking a GameId maintains the same values.
    function testFuzz_gameId_roundTrip_succeeds(
        GameType _gameType,
        Timestamp _timestamp,
        IDisputeGame _gameProxy
    )
        public
    {
        GameId gameId = LibGameId.pack(_gameType, _timestamp, _gameProxy);
        (GameType gameType_, Timestamp timestamp_, IDisputeGame gameProxy_) = LibGameId.unpack(gameId);

        assertEq(GameType.unwrap(gameType_), GameType.unwrap(_gameType));
        assertEq(Timestamp.unwrap(timestamp_), Timestamp.unwrap(_timestamp));
        assertEq(address(gameProxy_), address(_gameProxy));
    }
}

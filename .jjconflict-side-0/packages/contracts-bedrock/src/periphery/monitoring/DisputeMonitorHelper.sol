// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

// Libraries
import { LibString } from "@solady/utils/LibString.sol";

// Interfaces
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { Timestamp, GameType, Claim } from "src/dispute/lib/Types.sol";

/// @title DisputeMonitorHelper
/// @notice Peripheral contract that can help to monitor dispute games. Supplements offchain tools
///         by simplifying certain queries about dispute games.
contract DisputeMonitorHelper {
    /// @notice Thrown when the end index is less than the start index.
    error DisputeMonitorHelper_InvalidSearchRange();

    /// @notice Enum representing the direction of the search.
    enum SearchDirection {
        OLDER_THAN_OR_EQ,
        NEWER_THAN_OR_EQ
    }

    /// @notice Checks if a game was created by the provided factory.
    /// @param _factory The factory of the dispute games.
    /// @param _game The game to check.
    /// @return isValid_ True if the game was created by the factory, false otherwise.
    function isGameRegistered(IDisputeGameFactory _factory, IDisputeGame _game) public view returns (bool isValid_) {
        // Grab the game and game data.
        (GameType gameType, Claim rootClaim, bytes memory extraData) = _game.gameData();

        // Grab the verified address of the game based on the game data.
        (IDisputeGame _factoryRegisteredGame,) =
            _factory.games({ _gameType: gameType, _rootClaim: rootClaim, _extraData: extraData });

        // Return whether the game is factory registered.
        isValid_ = address(_factoryRegisteredGame) == address(_game);
    }

    /// @notice Finds all unresolved games in a given time range.
    /// @param _factory The factory of the dispute games.
    /// @param _creationRangeStart Start of the range of game creation timestamps.
    /// @param _creationRangeEnd End of the range of game creation timestamps.
    /// @return unresolvedGames_ The array of unresolved games.
    function getUnresolvedGames(
        IDisputeGameFactory _factory,
        uint256 _creationRangeStart,
        uint256 _creationRangeEnd
    )
        public
        view
        returns (IDisputeGame[] memory unresolvedGames_)
    {
        // Check that the max
        if (_creationRangeEnd < _creationRangeStart) {
            revert DisputeMonitorHelper_InvalidSearchRange();
        }

        // If there are no games, return an empty array. In theory we could error here too but it's
        // easier for offchain tooling if this case just returns empty. Either is fine, but this is
        // likely to be a common standard case and it'd be nicer if it didn't error.
        if (_factory.gameCount() == 0) {
            return new IDisputeGame[](0);
        }

        // Try to find a suitable start and end index. If startIdx is type(uint256).max then we did
        // not find any newer games and the creation range start must be after the timestamp of the
        // latest game. Similarly, if endIdx is type(uint256).max then we did not find any older
        // games and the creation range end must be before the timestamp of the earliest game. In
        // either case, we can return an empty array.
        uint256 startIdx = search(_factory, _creationRangeStart, SearchDirection.NEWER_THAN_OR_EQ);
        uint256 endIdx = search(_factory, _creationRangeEnd, SearchDirection.OLDER_THAN_OR_EQ);
        if (startIdx == type(uint256).max || endIdx == type(uint256).max) {
            return new IDisputeGame[](0);
        }

        // Additionally, if the end index is less than the start index, then the range is between
        // two dispute games. We return an empty array in this case.
        if (endIdx < startIdx) {
            return new IDisputeGame[](0);
        }

        // Allocate the array and fill it
        unresolvedGames_ = new IDisputeGame[](endIdx - startIdx + 1);
        uint256 unresolvedGameCount = 0;
        for (uint256 i = startIdx; i <= endIdx; i++) {
            (,, IDisputeGame game) = _factory.gameAtIndex(i);
            if (game.resolvedAt().raw() == 0) {
                unresolvedGames_[unresolvedGameCount] = game;
                unresolvedGameCount++;
            }
        }

        // Clobber the size of the array to return the right size.
        assembly {
            mstore(unresolvedGames_, unresolvedGameCount)
        }
    }

    /// @notice Searches for a game by timestamp, returning the index of the game that best matches the
    ///         given search direction.
    /// @param _factory         The factory of the dispute games.
    /// @param _targetTimestamp The timestamp to search for.
    /// @param _direction       The direction to search in (older or newer).
    /// @return index_          The index of the matching game, if it exists.
    function search(
        IDisputeGameFactory _factory,
        uint256 _targetTimestamp,
        SearchDirection _direction
    )
        public
        view
        returns (uint256 index_)
    {
        uint256 gameCount = _factory.gameCount();
        // If there are no games, return max to indicate "not found."
        if (gameCount == 0) {
            return type(uint256).max;
        }

        uint256 left = 0;
        uint256 right = gameCount - 1;

        // We'll store the candidate here. If it remains max, no suitable game was found.
        index_ = type(uint256).max;

        while (left <= right) {
            uint256 mid = left + (right - left) / 2;
            (, Timestamp timestamp,) = _factory.gameAtIndex(mid);
            uint256 gameTimestamp = uint64(timestamp.raw());

            if (_direction == SearchDirection.OLDER_THAN_OR_EQ) {
                // Rightmost index where timestamp <= _targetTimestamp
                if (gameTimestamp <= _targetTimestamp) {
                    index_ = mid;
                    left = mid + 1;
                } else {
                    if (mid == 0) {
                        // Prevent underflow
                        break;
                    }
                    right = mid - 1;
                }
            } else {
                // Leftmost index where timestamp >= _targetTimestamp
                if (gameTimestamp >= _targetTimestamp) {
                    index_ = mid;
                    if (mid == 0) {
                        // Prevent underflow
                        break;
                    }
                    right = mid - 1;
                } else {
                    left = mid + 1;
                }
            }
        }
    }

    /// @notice Converts a uint256 to an RPC hex string.
    /// @param _value The value to convert.
    /// @return hexString_ The hex string.
    function toRpcHexString(uint256 _value) public pure returns (string memory hexString_) {
        hexString_ = LibString.toMinimalHexString(_value);
    }
}

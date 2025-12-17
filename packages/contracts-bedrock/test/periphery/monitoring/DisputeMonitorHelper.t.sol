// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { DisputeGameFactory_TestInit } from "test/dispute/DisputeGameFactory.t.sol";
import { _changeClaimStatus } from "test/dispute/FaultDisputeGame.t.sol";

// Contracts
import { DisputeMonitorHelper } from "src/periphery/monitoring/DisputeMonitorHelper.sol";
import { GameTypes, Claim, VMStatuses } from "src/dispute/lib/Types.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";

/// @title DisputeMonitorHelper_TestInit
/// @notice Reusable test initialization for `DisputeMonitorHelper` tests.
abstract contract DisputeMonitorHelper_TestInit is DisputeGameFactory_TestInit {
    DisputeMonitorHelper helper;

    function setUp() public override {
        super.setUp();

        helper = new DisputeMonitorHelper();

        // Skip everything for forked networks. Tests here involve carefully controlling the list
        // of games in the factory, which is not possible on forked networks.
        skipIfForkTest("DisputeMonitorHelper tests are not applicable to forked networks");

        Claim absolutePrestate = _changeClaimStatus(Claim.wrap(keccak256(abi.encode(0))), VMStatuses.UNFINISHED);

        setupFaultDisputeGame(absolutePrestate);
    }

    /// @notice Helper to create a game with a specific timestamp.
    /// @param _timestamp The timestamp to set for the game creation.
    /// @param _claim The claim for the game.
    /// @return gameIndex_ The index of the created game.
    function createGameWithTimestamp(uint256 _timestamp, bytes32 _claim) internal returns (uint256 gameIndex_) {
        // Store current timestamp to restore later.
        uint256 currentTimestamp = block.timestamp;

        // Warp to the desired timestamp.
        vm.warp(_timestamp);

        // Create the game.
        disputeGameFactory.create{ value: disputeGameFactory.initBonds(GameTypes.CANNON) }(
            GameTypes.CANNON, Claim.wrap(_claim), abi.encode(999999)
        );

        // Get the game index.
        gameIndex_ = disputeGameFactory.gameCount() - 1;

        // Restore the original timestamp.
        vm.warp(currentTimestamp);
    }
}

/// @title DisputeMonitorHelper_IsGameRegistered_Test
/// @notice Tests the `isGameRegistered` function of the `DisputeMonitorHelper` contract.
contract DisputeMonitorHelper_IsGameRegistered_Test is DisputeMonitorHelper_TestInit {
    /// @notice Test that a game created through the factory is registered.
    function test_isGameRegistered_validGame_succeeds() external {
        // Create a game through the factory
        uint256 gameIndex = createGameWithTimestamp(block.timestamp, bytes32(uint256(1)));

        // Get the game address
        (,, IDisputeGame game) = disputeGameFactory.gameAtIndex(gameIndex);

        // Check that the game is registered
        bool isRegistered = helper.isGameRegistered(disputeGameFactory, game);
        assertTrue(isRegistered, "Game should be registered");
    }

    /// @notice Test that a random address is not registered as a game.
    function test_isGameRegistered_invalidGame_fails() external {
        // Create a random address that is not a registered game
        address randomAddress = address(uint160(uint256(keccak256(abi.encodePacked(block.timestamp)))));
        IDisputeGame fakeGame = IDisputeGame(randomAddress);

        // Mock the gameData call on the fake game to return something
        vm.mockCall(
            randomAddress,
            abi.encodeCall(IDisputeGame.gameData, ()),
            abi.encode(GameTypes.CANNON, Claim.wrap(bytes32(uint256(1))), abi.encode(999999))
        );

        // Check that the random address is not registered
        bool isRegistered = helper.isGameRegistered(disputeGameFactory, fakeGame);
        assertFalse(isRegistered, "Random address should not be registered as a game");
    }
}

/// @title DisputeMonitorHelper_GetUnresolvedGames_Test
/// @notice Tests the `getUnresolvedGames` function of the `DisputeMonitorHelper` contract.
contract DisputeMonitorHelper_GetUnresolvedGames_Test is DisputeMonitorHelper_TestInit {
    /// @notice Fuzz test for searching for unresolved games.
    /// @param _numGames Number of games to create.
    /// @param _resolvedPercent Percentage of games to mark as resolved.
    function testFuzz_getUnresolvedGames_succeeds(uint8 _numGames, uint8 _resolvedPercent) external {
        // Want _resolvedPercent to have 5% steps.
        _resolvedPercent = _resolvedPercent % 20;

        // Create an array to store game timestamps and indices.
        bool[] memory gameStatuses = new bool[](_numGames);
        uint256[] memory gameTimestamps = new uint256[](_numGames);
        uint256[] memory gameIndices = new uint256[](_numGames);

        // Start with a base timestamp.
        uint256 currentTimestamp = 1000;

        // Create games with increasing timestamps.
        for (uint256 i = 0; i < _numGames; i++) {
            // Generate a random timestamp increase (between 0 and 1000).
            uint256 timestampIncrease = vm.randomUint(0, 1000);
            currentTimestamp += timestampIncrease;

            // Store the timestamp.
            gameTimestamps[i] = currentTimestamp;

            // Create the game and store its index.
            gameIndices[i] = createGameWithTimestamp(currentTimestamp, bytes32(i + 1));

            // Decide if the game should be resolved or not.
            if (_resolvedPercent != 0 && vm.randomUint(0, 20) <= _resolvedPercent) {
                // Winner winner!
                // Mock the resolvedAt timestamp to anything but 0.
                gameStatuses[i] = true;
                (,, IDisputeGame game) = disputeGameFactory.gameAtIndex(i);
                vm.mockCall(address(game), abi.encodeCall(game.resolvedAt, ()), abi.encode(block.timestamp));
            } else {
                gameStatuses[i] = false;
            }
        }

        // If we have no games, expect an empty array always
        if (_numGames == 0) {
            uint256 creationRangeStart = vm.randomUint(0, block.timestamp + 1000000);
            uint256 creationRangeEnd = vm.randomUint(creationRangeStart, creationRangeStart + 1000000);
            IDisputeGame[] memory results =
                helper.getUnresolvedGames(disputeGameFactory, creationRangeStart, creationRangeEnd);
            assertEq(results.length, 0, "empty case returned games");
        } else {
            // Do 10 random tests inside the range we just created.
            for (uint256 i = 0; i < 10; i++) {
                // Pick a random timestamp that might be outside of the bounds of the available
                // timestamps. We'll use cases that fall outside of the available bounds to make sure
                // that errors are working as expected.
                uint256 rangeStart = gameTimestamps[0];
                uint256 rangeEnd = gameTimestamps[gameTimestamps.length - 1];
                uint256 randomRangeStart = vm.randomUint(rangeStart - 500, rangeEnd + 500);
                uint256 randomRangeEnd = vm.randomUint(rangeStart - 500, rangeEnd + 500);

                // Different assertions for different cases.
                if (randomRangeStart > randomRangeEnd) {
                    // If the boundaries are invalid, expect an error.
                    vm.expectRevert(DisputeMonitorHelper.DisputeMonitorHelper_InvalidSearchRange.selector);
                    helper.getUnresolvedGames(disputeGameFactory, randomRangeStart, randomRangeEnd);
                } else if (randomRangeEnd < rangeStart || randomRangeStart > rangeEnd) {
                    // If the boundaries are valid but the range is outside of the range of
                    // timestamps created by the array of games, expect an empty array.
                    IDisputeGame[] memory results =
                        helper.getUnresolvedGames(disputeGameFactory, randomRangeStart, randomRangeEnd);
                    assertEq(results.length, 0, "results should be empty");
                } else {
                    // Otherwise, we expect a number of results equal to the number of games that
                    // are unresolved within the range. Start by allocating an array with the total
                    // number of games, though actual size will be less.
                    IDisputeGame[] memory expected = new IDisputeGame[](_numGames);

                    // Create the array of expected results.
                    uint256 insertedCount = 0;
                    for (uint256 j = 0; j < _numGames; j++) {
                        if (
                            gameStatuses[j] == false && gameTimestamps[j] >= randomRangeStart
                                && gameTimestamps[j] <= randomRangeEnd
                        ) {
                            (,, IDisputeGame game) = disputeGameFactory.gameAtIndex(j);
                            expected[insertedCount] = game;
                            insertedCount++;
                        }
                    }

                    // Perform the search with our function.
                    IDisputeGame[] memory results =
                        helper.getUnresolvedGames(disputeGameFactory, randomRangeStart, randomRangeEnd);

                    // Should have a number of results equal to the elements inserted.
                    assertEq(results.length, insertedCount, "unexpected results length");

                    // Each element should match.
                    for (uint256 j = 0; j < results.length; j++) {
                        assertEq(address(results[j]), address(expected[j]));
                    }
                }
            }
        }
    }

    /// @notice Test that getting unresolved games with no games returns an empty array.
    /// @param _creationRangeStart The start of the creation range.
    /// @param _creationRangeEnd The end of the creation range.
    function testFuzz_getUnresolvedGames_noGames_succeeds(
        uint256 _creationRangeStart,
        uint256 _creationRangeEnd
    )
        external
        view
    {
        // Make sure the boundaries are valid.
        _creationRangeEnd = bound(_creationRangeEnd, _creationRangeStart, type(uint256).max);

        // Get the unresolved games.
        IDisputeGame[] memory results =
            helper.getUnresolvedGames(disputeGameFactory, _creationRangeStart, _creationRangeEnd);
        assertEq(results.length, 0, "empty case returned games");
    }

    /// @notice Test that getting unresolved games between two timestamps returns an empty array.
    function test_getUnresolvedGames_betweenTimestamps_succeeds() external {
        // Select two timestamps.
        uint256 timestamp1 = 100;
        uint256 timestamp2 = 200;

        // Create two games.
        createGameWithTimestamp(timestamp1, bytes32(uint256(1)));
        createGameWithTimestamp(timestamp2, bytes32(uint256(2)));

        // Select a range that falls between the two timestamps.
        uint256 rangeStart = timestamp1 + 1;
        uint256 rangeEnd = timestamp2 - 1;

        // Get the unresolved games.
        IDisputeGame[] memory results = helper.getUnresolvedGames(disputeGameFactory, rangeStart, rangeEnd);
        assertEq(results.length, 0, "expected 0 games");
    }

    /// @notice Fuzz test for getting unresolved games with bad boundaries.
    /// @param _creationRangeStart The start of the creation range.
    /// @param _creationRangeEnd The end of the creation range.
    function testFuzz_getUnresolvedGames_badBoundaries_reverts(
        uint256 _creationRangeStart,
        uint256 _creationRangeEnd
    )
        external
    {
        // Make sure the boundaries are deliberately invalid.
        _creationRangeStart = bound(_creationRangeStart, 1, type(uint256).max);
        _creationRangeEnd = bound(_creationRangeEnd, 0, _creationRangeStart - 1);

        // Get the unresolved games.
        vm.expectRevert(DisputeMonitorHelper.DisputeMonitorHelper_InvalidSearchRange.selector);
        helper.getUnresolvedGames(disputeGameFactory, _creationRangeStart, _creationRangeEnd);
    }

    /// @notice Fuzz test for getting unresolved games when all games are resolved.
    /// @param _numGames Number of games to create.
    function testFuzz_getUnresolvedGames_allResolved_succeeds(uint8 _numGames) external {
        // Create 5 games.
        for (uint256 i = 0; i < _numGames; i++) {
            createGameWithTimestamp(1000, bytes32(uint256(i + 1)));
            (,, IDisputeGame game) = disputeGameFactory.gameAtIndex(i);
            vm.mockCall(address(game), abi.encodeCall(game.resolvedAt, ()), abi.encode(block.timestamp));
        }

        // Get the unresolved games.
        IDisputeGame[] memory results = helper.getUnresolvedGames(disputeGameFactory, 0, type(uint256).max);
        assertEq(results.length, 0, "expected 0 games");
    }

    /// @notice Fuzz test for getting unresolved games when all games are unresolved.
    /// @param _numGames Number of games to create.
    function testFuzz_getUnresolvedGames_allUnresolved_succeeds(uint8 _numGames) external {
        // Create 5 games.
        for (uint256 i = 0; i < _numGames; i++) {
            createGameWithTimestamp(1000, bytes32(uint256(i + 1)));
        }

        // Get the unresolved games.
        IDisputeGame[] memory results = helper.getUnresolvedGames(disputeGameFactory, 0, type(uint256).max);
        assertEq(results.length, _numGames, "expected 5 games");
    }
}

contract DisputeMonitorHelper_toRpcHexString_Test is DisputeMonitorHelper_TestInit {
    /// @notice Test that the toRpcHexString function converts a uint256 to a hex string that
    ///         starts with 0x and doesn't have any leading zeros.
    function test_toRpcHexString_succeeds() external view {
        uint256 value = 1234567890;
        string memory hexString = helper.toRpcHexString(value);
        assertEq(hexString, "0x499602d2");
    }

    /// @notice Test that the toRpcHexString function converts a uint256 to a hex string that
    ///         doesn't have any leading zeros.
    function test_toRpcHexString_noLeadingZero_succeeds() external view {
        uint256 value = 136210625;
        string memory hexString = helper.toRpcHexString(value);
        assertEq(hexString, "0x81e68c1");
    }
}

/// @title DisputeMonitorHelper_Search_Test
/// @notice Tests the `search` function of the `DisputeMonitorHelper` contract.
contract DisputeMonitorHelper_Search_Test is DisputeMonitorHelper_TestInit {
    /// @notice Fuzz test for searching with random timestamps and directions.
    /// @param _numGames Number of games to generate for the test.
    /// @param _searchOlderThan Search direction.
    function testFuzz_search_succeeds(uint8 _numGames, bool _searchOlderThan) external {
        // Convert into search direction.
        DisputeMonitorHelper.SearchDirection direction = _searchOlderThan
            ? DisputeMonitorHelper.SearchDirection.OLDER_THAN_OR_EQ
            : DisputeMonitorHelper.SearchDirection.NEWER_THAN_OR_EQ;

        // Create an array to store game timestamps and indices.
        uint256[] memory gameTimestamps = new uint256[](_numGames);
        uint256[] memory gameIndices = new uint256[](_numGames);

        // Start with a base timestamp.
        uint256 currentTimestamp = 1000;

        // Create games with increasing timestamps.
        for (uint256 i = 0; i < _numGames; i++) {
            // Generate a random timestamp increase (between 0 and 1000). Games can have the same
            // exact timestamp. If this happens, we expect the earliest index in the NEWER_THAN
            // case or the latest index in the OLDER_THAN case.
            uint256 timestampIncrease = vm.randomUint(0, 1000);
            currentTimestamp += timestampIncrease;

            // Store the timestamp.
            gameTimestamps[i] = currentTimestamp;

            // Create the game and store its index.
            gameIndices[i] = createGameWithTimestamp(currentTimestamp, bytes32(i + 1));
        }

        // Verify the game count.
        assertEq(disputeGameFactory.gameCount(), _numGames, "wrong number of created games");

        // If we have no games, expect the NoGames error no matter the timestamp.
        if (_numGames == 0) {
            uint256 foundIndex =
                helper.search(disputeGameFactory, vm.randomUint(0, block.timestamp + 1000000), direction);
            assertEq(foundIndex, type(uint256).max, "found index should be max");
        } else {
            // Do 10 random tests inside the range we just created.
            for (uint256 i = 0; i < 10; i++) {
                // Pick a random timestamp that might be outside of the bounds of the available
                // timestamps. We'll use cases that fall outside of the available bounds to make
                // sure that errors are working as expected.
                uint256 rangeStart = gameTimestamps[0];
                uint256 rangeEnd = gameTimestamps[gameTimestamps.length - 1];
                uint256 randomTimestamp = vm.randomUint(rangeStart - 500, rangeEnd + 500);

                // Different assertions for different cases.
                if (
                    (direction == DisputeMonitorHelper.SearchDirection.OLDER_THAN_OR_EQ && randomTimestamp < rangeStart)
                        || (
                            direction == DisputeMonitorHelper.SearchDirection.NEWER_THAN_OR_EQ && randomTimestamp > rangeEnd
                        )
                ) {
                    // If we fall outside of the range, expect the max index representing that no
                    // valid game was found.
                    uint256 foundIndex = helper.search(disputeGameFactory, randomTimestamp, direction);
                    assertEq(foundIndex, type(uint256).max, "found index should be max");
                } else {
                    // Otherwise, we expect a valid index. Manual linear search to figure out what
                    // the right answer should be.
                    uint256 targetIndex;
                    for (uint256 j = 0; j < _numGames; j++) {
                        if (direction == DisputeMonitorHelper.SearchDirection.OLDER_THAN_OR_EQ) {
                            if (gameTimestamps[j] <= randomTimestamp) {
                                // Need to find newer indices for the OLDER_THAN_OR_EQ case.
                                targetIndex = j;
                            }
                        } else {
                            if (gameTimestamps[j] >= randomTimestamp) {
                                // Only find the first index for the NEWER_THAN_OR_EQ case.
                                targetIndex = j;
                                break;
                            }
                        }
                    }

                    // Perform the search with our function.
                    uint256 foundIndex = helper.search(disputeGameFactory, randomTimestamp, direction);

                    // Indices should match.
                    assertEq(foundIndex, targetIndex, "found incorrect index");
                }
            }
        }
    }

    /// @notice Test that searching for a game with no games returns the max index.
    /// @param _timestamp The timestamp to search for.
    /// @param _searchOlderThan Whether to search for an older or newer game.
    function testFuzz_search_noGames_succeeds(uint256 _timestamp, bool _searchOlderThan) external view {
        // Convert into search direction.
        DisputeMonitorHelper.SearchDirection direction = _searchOlderThan
            ? DisputeMonitorHelper.SearchDirection.OLDER_THAN_OR_EQ
            : DisputeMonitorHelper.SearchDirection.NEWER_THAN_OR_EQ;

        // Search for a game with no games.
        uint256 foundIndex = helper.search(disputeGameFactory, _timestamp, direction);

        // Should return the max index.
        assertEq(foundIndex, type(uint256).max, "found index should be max");
    }

    /// @notice Test that searching for a game older than all games returns the max index.
    function test_search_olderThanEverything_succeeds() external {
        // Select a target timestamp.
        uint256 targetTimestamp = vm.randomUint(1, 100);

        // Create one game.
        createGameWithTimestamp(targetTimestamp, bytes32(uint256(1)));

        // Search by providing a timestamp that is before all games.
        uint256 foundIndex = helper.search(
            disputeGameFactory, targetTimestamp - 1, DisputeMonitorHelper.SearchDirection.OLDER_THAN_OR_EQ
        );
        assertEq(foundIndex, type(uint256).max, "found index should be max");
    }

    /// @notice Test that searching for a game newer than all games returns the max index.
    function test_search_newerThanEverything_succeeds() external {
        // Select a target timestamp.
        uint256 targetTimestamp = vm.randomUint(1, 100);

        // Create one game.
        createGameWithTimestamp(targetTimestamp, bytes32(uint256(1)));

        // Search by providing a timestamp that is after all games.
        uint256 foundIndex = helper.search(
            disputeGameFactory, targetTimestamp + 1, DisputeMonitorHelper.SearchDirection.NEWER_THAN_OR_EQ
        );
        assertEq(foundIndex, type(uint256).max, "found index should be max");
    }
}

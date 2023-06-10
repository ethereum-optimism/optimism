// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { Test } from "forge-std/Test.sol";
import { DisputeGameFactory } from "../dispute/DisputeGameFactory.sol";
import { FaultDisputeGame } from "../dispute/FaultDisputeGame.sol";

import "../libraries/DisputeTypes.sol";
import { LibClock } from "../dispute/lib/LibClock.sol";
import { LibPosition } from "../dispute/lib/LibPosition.sol";

contract FaultDisputeGame_Test is Test {
    /**
     * @dev The root claim of the game.
     */
    Claim internal constant ROOT_CLAIM = Claim.wrap(bytes32(uint256(10)));
    /**
     * @dev The extra data passed to the game for initialization.
     */
    bytes internal constant EXTRA_DATA = abi.encode(1);
    /**
     * @dev The type of the game being tested.
     */
    GameType internal constant GAME_TYPE = GameType.FAULT;
    /**
     * @dev The current version of the `FaultDisputeGame` contract.
     */
    string internal constant VERSION = "0.0.1";

    /**
     * @dev The factory that will be used to create the game.
     */
    DisputeGameFactory internal factory;
    /**
     * @dev The implementation of the game.
     */
    FaultDisputeGame internal gameImpl;
    /**
     * @dev The `Clone` proxy of the game.
     */
    FaultDisputeGame internal gameProxy;

    function setUp() public {
        // Deploy a new dispute game factory.
        factory = new DisputeGameFactory(address(this));
        // Deploy an implementation of the fault game
        gameImpl = new FaultDisputeGame();
        // Register the game implementation with the factory.
        factory.setImplementation(GAME_TYPE, gameImpl);
        // Create a new game.
        gameProxy = FaultDisputeGame(address(factory.create(GAME_TYPE, ROOT_CLAIM, EXTRA_DATA)));

        // Label the proxy
        vm.label(address(gameProxy), "FaultDisputeGame_Clone");
    }

    ////////////////////////////////////////////////////////////////
    //            `IDisputeGame` Implementation Tests             //
    ////////////////////////////////////////////////////////////////

    /**
     * @dev Tests that the game's root claim is set correctly.
     */
    function test_rootClaim_succeeds() public {
        assertEq(Claim.unwrap(gameProxy.rootClaim()), Claim.unwrap(ROOT_CLAIM));
    }

    /**
     * @dev Tests that the game's extra data is set correctly.
     */
    function test_extraData_succeeds() public {
        assertEq(gameProxy.extraData(), EXTRA_DATA);
    }

    /**
     * @dev Tests that the game's version is set correctly.
     */
    function test_version_succeeds() public {
        assertEq(gameProxy.version(), VERSION);
    }

    /**
     * @dev Tests that the game's status is set correctly.
     */
    function test_gameStart_succeeds() public {
        assertEq(Timestamp.unwrap(gameProxy.gameStart()), block.timestamp);
    }

    /**
     * @dev Tests that the game's type is set correctly.
     */
    function test_gameType_succeeds() public {
        assertEq(uint256(gameProxy.gameType()), uint256(GAME_TYPE));
    }

    /**
     * @dev Tests that the game's data is set correctly.
     */
    function test_gameData_succeeds() public {
        (GameType gameType, Claim rootClaim, bytes memory extraData) =
            gameProxy.gameData();

        assertEq(uint256(gameType), uint256(GAME_TYPE));
        assertEq(Claim.unwrap(rootClaim), Claim.unwrap(ROOT_CLAIM));
        assertEq(extraData, EXTRA_DATA);
    }

    ////////////////////////////////////////////////////////////////
    //          `IFaultDisputeGame` Implementation Tests          //
    ////////////////////////////////////////////////////////////////

    /**
     * @dev Tests that the root claim's data is set correctly when the game is initialized.
     */
    function test_initialRootClaimData_succeeds() public {
        (
            uint32 parentIndex,
            uint32 rc,
            bool countered,
            Claim claim,
            Position position,
            Clock clock
        ) = gameProxy.claimData(0);

        assertEq(parentIndex, type(uint32).max);
        assertEq(rc, 0);
        assertEq(countered, false);
        assertEq(Claim.unwrap(claim), Claim.unwrap(ROOT_CLAIM));
        assertEq(Position.unwrap(position), 0);
        assertEq(Clock.unwrap(clock), Clock.unwrap(LibClock.wrap(Duration.wrap(0), Timestamp.wrap(uint64(block.timestamp)))));
    }

    /**
     * @dev Static unit test for the correctness of an opening attack.
     */
    function test_simpleAttack_succeeds() public {
        // Warp ahead 5 seconds.
        vm.warp(block.timestamp + 5);

        // Perform the attack.
        gameProxy.attack(0, Claim.wrap(bytes32(uint(5))));

        // Grab the claim data of the attack.
        (
            uint32 parentIndex,
            uint32 rc,
            bool countered,
            Claim claim,
            Position position,
            Clock clock
        ) = gameProxy.claimData(1);

        // Assert correctness of the attack claim's data.
        assertEq(parentIndex, 0);
        assertEq(rc, 0);
        assertEq(countered, false);
        assertEq(Claim.unwrap(claim), Claim.unwrap(Claim.wrap(bytes32(uint(5)))));
        assertEq(Position.unwrap(position), Position.unwrap(LibPosition.attack(Position.wrap(0))));
        assertEq(Clock.unwrap(clock), Clock.unwrap(LibClock.wrap(Duration.wrap(5), Timestamp.wrap(uint64(block.timestamp)))));

        // Grab the claim data of the parent.
        (
            parentIndex,
            rc,
            countered,
            claim,
            position,
            clock
        ) = gameProxy.claimData(0);

        // Assert correctness of the parent claim's data.
        assertEq(parentIndex, type(uint32).max);
        assertEq(rc, 1);
        assertEq(countered, true);
        assertEq(Claim.unwrap(claim), Claim.unwrap(ROOT_CLAIM));
        assertEq(Position.unwrap(position), 0);
        assertEq(Clock.unwrap(clock), Clock.unwrap(LibClock.wrap(Duration.wrap(0), Timestamp.wrap(uint64(block.timestamp - 5)))));

        // Resolve the game.
        assertEq(uint256(gameProxy.resolve()), uint256(GameStatus.CHALLENGER_WINS));
    }
}

/**
 * @title BigStepper
 * @notice A mock fault proof processor contract for testing purposes.
 *⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
 *⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣼⠶⢅⠒⢄⢔⣶⡦⣤⡤⠄⣀⠀⠀⠀⠀⠀⠀⠀
 *⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠨⡏⠀⠀⠈⠢⣙⢯⣄⠀⢨⠯⡺⡘⢄⠀⠀⠀⠀⠀
 *⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣶⡆⠀⠀⠀⠀⠈⠓⠬⡒⠡⣀⢙⡜⡀⠓⠄⠀⠀⠀
 *⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⡷⠿⣧⣀⡀⠀⠀⠀⠀⠀⠀⠉⠣⣞⠩⠥⠀⠼⢄⠀⠀
 *⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⡇⠀⠀⠀⠉⢹⣶⠒⠒⠂⠈⠉⠁⠘⡆⠀⣿⣿⠫⡄⠀
 *⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⢶⣤⣀⡀⠀⠀⢸⡿⠀⠀⠀⠀⠀⢀⠞⠀⠀⢡⢨⢀⡄⠀
 *⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⡒⣿⢿⡤⠝⡣⠉⠁⠚⠛⠀⠤⠤⣄⡰⠁⠀⠀⠀⠉⠙⢸⠀⠀
 *⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⡤⢯⡌⡿⡇⠘⡷⠀⠁⠀⠀⢀⣰⠢⠲⠛⣈⣸⠦⠤⠶⠴⢬⣐⣊⡂⠀
 *⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣤⡪⡗⢫⠞⠀⠆⣀⠻⠤⠴⠐⠚⣉⢀⠦⠂⠋⠁⠀⠁⠀⠀⠀⠀⢋⠉⠇⠀
 *⠀⠀⠀⠀⣀⡤⠐⠒⠘⡹⠉⢸⠇⠸⠀⠀⠀⠀⣀⣤⠴⠚⠉⠈⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠼⠀⣾⠀
 *⠀⠀⠀⡰⠀⠉⠉⠀⠁⠀⠀⠈⢇⠈⠒⠒⠘⠈⢀⢡⡂⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢰⠀⢸⡄
 *⠀⠀⠸⣿⣆⠤⢀⡀⠀⠀⠀⠀⢘⡌⠀⠀⣀⣀⣀⡈⣤⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⠀⢸⡇
 *⠀⠀⢸⣀⠀⠉⠒⠐⠛⠋⠭⠭⠍⠉⠛⠒⠒⠒⠀⠒⠚⠛⠛⠛⠩⠭⠭⠭⠭⠤⠤⠤⠤⠤⠭⠭⠉⠓⡆
 *⠀⠀⠘⠿⣷⣶⣤⣤⣀⣀⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⣤⣄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⡇
 *⠀⠀⠀⠀⠀⠉⠙⠛⠛⠻⠿⢿⣿⣿⣷⣶⣶⣶⣤⣤⣀⣁⣛⣃⣒⠿⠿⠿⠤⠠⠄⠤⠤⢤⣛⣓⣂⣻⡇
 *⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠉⠉⠉⠙⠛⠻⠿⠿⠿⢿⣿⣿⣿⣷⣶⣶⣾⣿⣿⣿⣿⠿⠟⠁
 *⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠈⠉⠉⠉⠉⠁⠀⠀⠀⠀⠀⠀
 */
contract BigStepper {
    /**
     * @notice Steps from the `preState` to the `postState` by adding 1 to the `preState`.
     * @param preState The pre state to start from
     * @return postState The state stepped to
     */
    function step(Claim preState) external pure returns (Claim postState) {
        postState = Claim.wrap(bytes32(uint256(Claim.unwrap(preState)) + 1));
    }
}

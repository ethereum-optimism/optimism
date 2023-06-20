// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { Test } from "forge-std/Test.sol";
import { DisputeGameFactory_Init } from "./DisputeGameFactory.t.sol";
import { DisputeGameFactory } from "../dispute/DisputeGameFactory.sol";
import { FaultDisputeGame } from "../dispute/FaultDisputeGame.sol";

import "../libraries/DisputeTypes.sol";
import "../libraries/DisputeErrors.sol";
import { LibClock } from "../dispute/lib/LibClock.sol";
import { LibPosition } from "../dispute/lib/LibPosition.sol";

contract FaultDisputeGame_Test is DisputeGameFactory_Init {
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
    GameType internal constant GAME_TYPE = GameType.wrap(0);
    /**
     * @dev The current version of the `FaultDisputeGame` contract.
     */
    string internal constant VERSION = "0.0.1";

    /**
     * @dev The implementation of the game.
     */
    FaultDisputeGame internal gameImpl;
    /**
     * @dev The `Clone` proxy of the game.
     */
    FaultDisputeGame internal gameProxy;

    event Move(uint256 indexed parentIndex, Claim indexed pivot, address indexed claimant);

    function setUp() public override {
        super.setUp();
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
        assertEq(GameType.unwrap(gameProxy.gameType()), GameType.unwrap(GAME_TYPE));
    }

    /**
     * @dev Tests that the game's data is set correctly.
     */
    function test_gameData_succeeds() public {
        (GameType gameType, Claim rootClaim, bytes memory extraData) = gameProxy.gameData();

        assertEq(GameType.unwrap(gameType), GameType.unwrap(GAME_TYPE));
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
            bool countered,
            Claim claim,
            Position position,
            Clock clock
        ) = gameProxy.claimData(0);

        assertEq(parentIndex, type(uint32).max);
        assertEq(countered, false);
        assertEq(Claim.unwrap(claim), Claim.unwrap(ROOT_CLAIM));
        assertEq(Position.unwrap(position), 1);
        assertEq(
            Clock.unwrap(clock),
            Clock.unwrap(LibClock.wrap(Duration.wrap(0), Timestamp.wrap(uint64(block.timestamp))))
        );
    }

    /**
     * @dev Tests that a move while the game status is not `IN_PROGRESS` causes the call to revert
     *      with the `GameNotInProgress` error
     */
    function test_move_gameNotInProgress_reverts() public {
        uint256 chalWins = uint256(GameStatus.CHALLENGER_WINS);

        // Replace the game status in storage. It exists in slot 0 at offset 8.
        uint256 slot = uint256(vm.load(address(gameProxy), bytes32(0)));
        uint256 offset = (8 << 3);
        uint256 mask = 0xFF << offset;
        // Replace the byte in the slot value with the challenger wins status.
        slot = (slot & ~mask) | (chalWins << offset);
        vm.store(address(gameProxy), bytes32(uint256(0)), bytes32(slot));

        // Ensure that the game status was properly updated.
        GameStatus status = gameProxy.status();
        assertEq(uint256(status), chalWins);

        // Attempt to make a move. Should revert.
        vm.expectRevert(GameNotInProgress.selector);
        gameProxy.attack(0, Claim.wrap(0));
    }

    /**
     * @dev Tests that an attempt to defend the root claim reverts with the `CannotDefendRootClaim` error.
     */
    function test_defendRoot_reverts() public {
        vm.expectRevert(CannotDefendRootClaim.selector);
        gameProxy.defend(0, Claim.wrap(bytes32(uint256(5))));
    }

    /**
     * @dev Tests that an attempt to move against a claim that does not exist reverts with the
     *      `ParentDoesNotExist` error.
     */
    function test_moveAgainstNonexistentParent_reverts() public {
        Claim claim = Claim.wrap(bytes32(uint256(5)));

        // Expect an out of bounds revert for an attack
        vm.expectRevert(abi.encodeWithSignature("Panic(uint256)", 0x32));
        gameProxy.attack(1, claim);

        // Expect an out of bounds revert for an attack
        vm.expectRevert(abi.encodeWithSignature("Panic(uint256)", 0x32));
        gameProxy.defend(1, claim);
    }

    /**
     * @dev Tests that an attempt to move at the maximum game depth reverts with the
     *      `GameDepthExceeded` error.
     */
    function test_gameDepthExceeded_reverts() public {
        Claim claim = Claim.wrap(bytes32(uint256(5)));

        for (uint256 i = 0; i < 63; i++) {
            // At the max game depth, the `_move` function should revert with
            // the `GameDepthExceeded` error.
            if (i == 62) {
                vm.expectRevert(GameDepthExceeded.selector);
            }
            gameProxy.attack(i, claim);
        }
    }

    /**
     * @dev Tests that a move made after the clock time has exceeded reverts with the
     *      `ClockTimeExceeded` error.
     */
    function test_clockTimeExceeded_reverts() public {
        // Warp ahead past the clock time for the first move (3 1/2 days)
        vm.warp(block.timestamp + 3 days + 12 hours + 1);
        vm.expectRevert(ClockTimeExceeded.selector);
        gameProxy.attack(0, Claim.wrap(bytes32(uint256(5))));
    }

    /**
     * @dev Tests that an identical claim cannot be made twice. The duplicate claim attempt should
     *      revert with the `ClaimAlreadyExists` error.
     */
    function test_duplicateClaim_reverts() public {
        Claim claim = Claim.wrap(bytes32(uint256(5)));

        // Make the first move. This should succeed.
        gameProxy.attack(0, claim);

        // Attempt to make the same move again.
        vm.expectRevert(ClaimAlreadyExists.selector);
        gameProxy.attack(0, claim);
    }

    /**
     * @dev Static unit test for the correctness of an opening attack.
     */
    function test_simpleAttack_succeeds() public {
        // Warp ahead 5 seconds.
        vm.warp(block.timestamp + 5);

        Claim counter = Claim.wrap(bytes32(uint256(5)));

        // Perform the attack.
        vm.expectEmit(true, true, true, false);
        emit Move(0, counter, address(this));
        gameProxy.attack(0, counter);

        // Grab the claim data of the attack.
        (
            uint32 parentIndex,
            bool countered,
            Claim claim,
            Position position,
            Clock clock
        ) = gameProxy.claimData(1);

        // Assert correctness of the attack claim's data.
        assertEq(parentIndex, 0);
        assertEq(countered, false);
        assertEq(Claim.unwrap(claim), Claim.unwrap(counter));
        assertEq(Position.unwrap(position), Position.unwrap(Position.wrap(1).attack()));
        assertEq(
            Clock.unwrap(clock),
            Clock.unwrap(LibClock.wrap(Duration.wrap(5), Timestamp.wrap(uint64(block.timestamp))))
        );

        // Grab the claim data of the parent.
        (parentIndex, countered, claim, position, clock) = gameProxy.claimData(0);

        // Assert correctness of the parent claim's data.
        assertEq(parentIndex, type(uint32).max);
        assertEq(countered, true);
        assertEq(Claim.unwrap(claim), Claim.unwrap(ROOT_CLAIM));
        assertEq(Position.unwrap(position), 1);
        assertEq(
            Clock.unwrap(clock),
            Clock.unwrap(
                LibClock.wrap(Duration.wrap(0), Timestamp.wrap(uint64(block.timestamp - 5)))
            )
        );
    }

    /**
     * @dev Static unit test for the correctness an uncontested root resolution.
     */
    function test_resolve_rootUncontested() public {
        GameStatus status = gameProxy.resolve();
        assertEq(uint8(status), uint8(GameStatus.DEFENDER_WINS));
        assertEq(uint8(gameProxy.status()), uint8(GameStatus.DEFENDER_WINS));
    }

    /**
     * @dev Static unit test asserting that resolve reverts when the game is not in progress.
     */
    function test_resolve_reverts() public {
        gameProxy.resolve();
        vm.expectRevert(GameNotInProgress.selector);
        gameProxy.resolve();
    }

    /**
     * @dev Static unit test for the correctness of resolving a single attack game state.
     */
    function test_resolve_rootContested() public {
        gameProxy.attack(0, Claim.wrap(bytes32(uint256(5))));

        GameStatus status = gameProxy.resolve();
        assertEq(uint8(status), uint8(GameStatus.CHALLENGER_WINS));
        assertEq(uint8(gameProxy.status()), uint8(GameStatus.CHALLENGER_WINS));
    }

    /**
     * @dev Static unit test for the correctness of resolving a game with a contested challenge claim.
     */
    function test_resolve_challengeContested() public {
        gameProxy.attack(0, Claim.wrap(bytes32(uint256(5))));
        gameProxy.defend(1, Claim.wrap(bytes32(uint256(6))));

        GameStatus status = gameProxy.resolve();
        assertEq(uint8(status), uint8(GameStatus.DEFENDER_WINS));
        assertEq(uint8(gameProxy.status()), uint8(GameStatus.DEFENDER_WINS));
    }

    /**
     * @dev Static unit test for the correctness of resolving a game with multiplayer moves.
     */
    function test_resolve_teamDeathmatch() public {
        gameProxy.attack(0, Claim.wrap(bytes32(uint256(5))));
        gameProxy.attack(0, Claim.wrap(bytes32(uint256(4))));
        gameProxy.defend(1, Claim.wrap(bytes32(uint256(6))));
        gameProxy.defend(1, Claim.wrap(bytes32(uint256(7))));

        GameStatus status = gameProxy.resolve();
        assertEq(uint8(status), uint8(GameStatus.CHALLENGER_WINS));
        assertEq(uint8(gameProxy.status()), uint8(GameStatus.CHALLENGER_WINS));
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
 *⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠈⠉⠉⠉⠉⠁⠀⠀⠀⠀⠀
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

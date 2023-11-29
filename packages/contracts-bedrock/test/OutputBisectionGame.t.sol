// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { Test } from "forge-std/Test.sol";
import { Vm } from "forge-std/Vm.sol";
import { DisputeGameFactory_Init } from "test/DisputeGameFactory.t.sol";
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { OutputBisectionGame } from "src/dispute/OutputBisectionGame.sol";
import { L2OutputOracle } from "src/L1/L2OutputOracle.sol";
import { BlockOracle } from "src/dispute/BlockOracle.sol";
import { PreimageOracle } from "src/cannon/PreimageOracle.sol";
import { PreimageKeyLib } from "src/cannon/PreimageKeyLib.sol";

import "src/libraries/DisputeTypes.sol";
import "src/libraries/DisputeErrors.sol";
import { Types } from "src/libraries/Types.sol";
import { LibClock } from "src/dispute/lib/LibClock.sol";
import { LibPosition } from "src/dispute/lib/LibPosition.sol";
import { IBigStepper, IPreimageOracle } from "src/dispute/interfaces/IBigStepper.sol";
import { AlphabetVM } from "test/mocks/AlphabetVM.sol";

contract OutputBisectionGame_Init is DisputeGameFactory_Init {
    /// @dev The type of the game being tested.
    GameType internal constant GAME_TYPE = GameType.wrap(0);
    /// @dev The L2 Block Number for the game's proposed output (the root claim)
    uint256 internal constant L2_BLOCK_NUMBER = 0xFFFF;

    uint256 internal constant GENESIS_BLOCK_NUMBER = 0;

    /// @dev The implementation of the game.
    OutputBisectionGame internal gameImpl;
    /// @dev The `Clone` proxy of the game.
    OutputBisectionGame internal gameProxy;

    /// @dev The extra data passed to the game for initialization.
    bytes internal extraData;

    event Move(uint256 indexed parentIndex, Claim indexed pivot, address indexed claimant);

    function init(Claim rootClaim, Claim absolutePrestate) public {
        // Set the time to a realistic date.
        vm.warp(1690906994);

        // Set the extra data for the game creation
        extraData = abi.encode(L2_BLOCK_NUMBER);

        // Deploy an implementation of the fault game
        gameImpl = new OutputBisectionGame(
            GAME_TYPE,
            absolutePrestate,
            GENESIS_BLOCK_NUMBER,
            4,
            2,
            Duration.wrap(7 days),
            new AlphabetVM(absolutePrestate)
        );
        // Register the game implementation with the factory.
        factory.setImplementation(GAME_TYPE, gameImpl);
        // Create a new game.
        gameProxy = OutputBisectionGame(address(factory.create(GAME_TYPE, rootClaim, extraData)));

        // Label the proxy
        vm.label(address(gameProxy), "OutputBisectionGame_Clone");
    }
}

contract OutputBisectionGame_Test is OutputBisectionGame_Init {
    /// @dev The root claim of the game.
    Claim internal constant ROOT_CLAIM = Claim.wrap(bytes32((uint256(1) << 248) | uint256(10)));
    /// @dev The absolute prestate of the trace.
    Claim internal constant ABSOLUTE_PRESTATE = Claim.wrap(bytes32((uint256(3) << 248) | uint256(0)));

    function setUp() public override {
        super.setUp();
        super.init(ROOT_CLAIM, ABSOLUTE_PRESTATE);
    }

    ////////////////////////////////////////////////////////////////
    //            `IDisputeGame` Implementation Tests             //
    ////////////////////////////////////////////////////////////////

    /// @dev Tests that the game's root claim is set correctly.
    function test_rootClaim_succeeds() public {
        assertEq(Claim.unwrap(gameProxy.rootClaim()), Claim.unwrap(ROOT_CLAIM));
    }

    /// @dev Tests that the game's extra data is set correctly.
    function test_extraData_succeeds() public {
        assertEq(gameProxy.extraData(), extraData);
    }

    /// @dev Tests that the game's starting timestamp is set correctly.
    function test_createdAt_succeeds() public {
        assertEq(Timestamp.unwrap(gameProxy.createdAt()), block.timestamp);
    }

    /// @dev Tests that the game's type is set correctly.
    function test_gameType_succeeds() public {
        assertEq(GameType.unwrap(gameProxy.gameType()), GameType.unwrap(GAME_TYPE));
    }

    /// @dev Tests that the game's data is set correctly.
    function test_gameData_succeeds() public {
        (GameType gameType, Claim rootClaim, bytes memory _extraData) = gameProxy.gameData();

        assertEq(GameType.unwrap(gameType), GameType.unwrap(GAME_TYPE));
        assertEq(Claim.unwrap(rootClaim), Claim.unwrap(ROOT_CLAIM));
        assertEq(_extraData, extraData);
    }

    ////////////////////////////////////////////////////////////////
    //          `IOutputBisectionGame` Implementation Tests          //
    ////////////////////////////////////////////////////////////////

    /// @dev Tests that the game is initialized with the correct data.
    function test_initialize_correctData_succeeds() public {
        // Assert that the root claim is initialized correctly.
        (uint32 parentIndex, bool countered, Claim claim, Position position, Clock clock) = gameProxy.claimData(0);
        assertEq(parentIndex, type(uint32).max);
        assertEq(countered, false);
        assertEq(Claim.unwrap(claim), Claim.unwrap(ROOT_CLAIM));
        assertEq(Position.unwrap(position), 1);
        assertEq(
            Clock.unwrap(clock), Clock.unwrap(LibClock.wrap(Duration.wrap(0), Timestamp.wrap(uint64(block.timestamp))))
        );

        // Assert that the `createdAt` timestamp is correct.
        assertEq(Timestamp.unwrap(gameProxy.createdAt()), block.timestamp);

        // Assert that the blockhash provided is correct.
        assertEq(Hash.unwrap(gameProxy.l1Head()), blockhash(block.number - 1));
    }

    /// @dev Tests that a move while the game status is not `IN_PROGRESS` causes the call to revert
    ///      with the `GameNotInProgress` error
    function test_move_gameNotInProgress_reverts() public {
        uint256 chalWins = uint256(GameStatus.CHALLENGER_WINS);

        // Replace the game status in storage. It exists in slot 0 at offset 16.
        uint256 slot = uint256(vm.load(address(gameProxy), bytes32(0)));
        uint256 offset = 16 << 3;
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

    /// @dev Tests that an attempt to defend the root claim reverts with the `CannotDefendRootClaim` error.
    function test_move_defendRoot_reverts() public {
        vm.expectRevert(CannotDefendRootClaim.selector);
        gameProxy.defend(0, Claim.wrap(bytes32(uint256(5))));
    }

    /// @dev Tests that an attempt to move against a claim that does not exist reverts with the
    ///      `ParentDoesNotExist` error.
    function test_move_nonExistentParent_reverts() public {
        Claim claim = Claim.wrap(bytes32(uint256(5)));

        // Expect an out of bounds revert for an attack
        vm.expectRevert(abi.encodeWithSignature("Panic(uint256)", 0x32));
        gameProxy.attack(1, claim);

        // Expect an out of bounds revert for an attack
        vm.expectRevert(abi.encodeWithSignature("Panic(uint256)", 0x32));
        gameProxy.defend(1, claim);
    }

    /// @dev Tests that an attempt to move at the maximum game depth reverts with the
    ///      `GameDepthExceeded` error.
    function test_move_gameDepthExceeded_reverts() public {
        Claim claim = Claim.wrap(bytes32(uint256(5)));

        uint256 maxDepth = gameProxy.MAX_GAME_DEPTH();

        for (uint256 i = 0; i <= maxDepth; i++) {
            // At the max game depth, the `_move` function should revert with
            // the `GameDepthExceeded` error.
            if (i == maxDepth) {
                vm.expectRevert(GameDepthExceeded.selector);
            } else if (i == 2) {
                claim = changeClaimStatus(claim, VMStatuses.PANIC);
            }
            gameProxy.attack(i, claim);
        }
    }

    /// @dev Tests that a move made after the clock time has exceeded reverts with the
    ///      `ClockTimeExceeded` error.
    function test_move_clockTimeExceeded_reverts() public {
        // Warp ahead past the clock time for the first move (3 1/2 days)
        vm.warp(block.timestamp + 3 days + 12 hours + 1);
        vm.expectRevert(ClockTimeExceeded.selector);
        gameProxy.attack(0, Claim.wrap(bytes32(uint256(5))));
    }

    /// @notice Static unit test for the correctness of the chess clock incrementation.
    function test_move_clockCorrectness_succeeds() public {
        (,,,, Clock clock) = gameProxy.claimData(0);
        assertEq(
            Clock.unwrap(clock), Clock.unwrap(LibClock.wrap(Duration.wrap(0), Timestamp.wrap(uint64(block.timestamp))))
        );

        Claim claim = Claim.wrap(bytes32(uint256(5)));

        vm.warp(block.timestamp + 15);
        gameProxy.attack(0, claim);
        (,,,, clock) = gameProxy.claimData(1);
        assertEq(
            Clock.unwrap(clock), Clock.unwrap(LibClock.wrap(Duration.wrap(15), Timestamp.wrap(uint64(block.timestamp))))
        );

        vm.warp(block.timestamp + 10);
        gameProxy.attack(1, claim);
        (,,,, clock) = gameProxy.claimData(2);
        assertEq(
            Clock.unwrap(clock), Clock.unwrap(LibClock.wrap(Duration.wrap(10), Timestamp.wrap(uint64(block.timestamp))))
        );

        // We are at the split depth, so we need to set the status byte of the claim
        // for the next move.
        claim = changeClaimStatus(claim, VMStatuses.PANIC);

        vm.warp(block.timestamp + 10);
        gameProxy.attack(2, claim);
        (,,,, clock) = gameProxy.claimData(3);
        assertEq(
            Clock.unwrap(clock), Clock.unwrap(LibClock.wrap(Duration.wrap(25), Timestamp.wrap(uint64(block.timestamp))))
        );

        vm.warp(block.timestamp + 10);
        gameProxy.attack(3, claim);
        (,,,, clock) = gameProxy.claimData(4);
        assertEq(
            Clock.unwrap(clock), Clock.unwrap(LibClock.wrap(Duration.wrap(20), Timestamp.wrap(uint64(block.timestamp))))
        );
    }

    /// @dev Tests that an identical claim cannot be made twice. The duplicate claim attempt should
    ///      revert with the `ClaimAlreadyExists` error.
    function test_move_duplicateClaim_reverts() public {
        Claim claim = Claim.wrap(bytes32(uint256(5)));

        // Make the first move. This should succeed.
        gameProxy.attack(0, claim);

        // Attempt to make the same move again.
        vm.expectRevert(ClaimAlreadyExists.selector);
        gameProxy.attack(0, claim);
    }

    /// @dev Static unit test asserting that identical claims at the same position can be made in different subgames.
    function test_move_duplicateClaimsDifferentSubgames_succeeds() public {
        Claim claimA = Claim.wrap(bytes32(uint256(5)));
        Claim claimB = Claim.wrap(bytes32(uint256(6)));

        // Make the first move. This should succeed.
        gameProxy.attack(0, claimA);
        gameProxy.attack(0, claimB);

        gameProxy.attack(1, claimB);
        gameProxy.attack(2, claimA);
    }

    /// @dev Static unit test for the correctness of an opening attack.
    function test_move_simpleAttack_succeeds() public {
        // Warp ahead 5 seconds.
        vm.warp(block.timestamp + 5);

        Claim counter = Claim.wrap(bytes32(uint256(5)));

        // Perform the attack.
        vm.expectEmit(true, true, true, false);
        emit Move(0, counter, address(this));
        gameProxy.attack(0, counter);

        // Grab the claim data of the attack.
        (uint32 parentIndex, bool countered, Claim claim, Position position, Clock clock) = gameProxy.claimData(1);

        // Assert correctness of the attack claim's data.
        assertEq(parentIndex, 0);
        assertEq(countered, false);
        assertEq(Claim.unwrap(claim), Claim.unwrap(counter));
        assertEq(Position.unwrap(position), Position.unwrap(Position.wrap(1).move(true)));
        assertEq(
            Clock.unwrap(clock), Clock.unwrap(LibClock.wrap(Duration.wrap(5), Timestamp.wrap(uint64(block.timestamp))))
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
            Clock.unwrap(LibClock.wrap(Duration.wrap(0), Timestamp.wrap(uint64(block.timestamp - 5))))
        );
    }

    /// @dev Static unit test for the correctness an uncontested root resolution.
    function test_resolve_rootUncontested_succeeds() public {
        vm.warp(block.timestamp + 3 days + 12 hours + 1 seconds);
        gameProxy.resolveClaim(0);
        assertEq(uint8(gameProxy.resolve()), uint8(GameStatus.DEFENDER_WINS));
    }

    /// @dev Static unit test for the correctness an uncontested root resolution.
    function test_resolve_rootUncontestedClockNotExpired_succeeds() public {
        vm.warp(block.timestamp + 3 days + 12 hours);
        vm.expectRevert(ClockNotExpired.selector);
        gameProxy.resolveClaim(0);
    }

    /// @dev Static unit test asserting that resolve reverts when the absolute root
    ///      subgame has not been resolved.
    function test_resolve_rootUncontestedButUnresolved_reverts() public {
        vm.warp(block.timestamp + 3 days + 12 hours + 1 seconds);
        vm.expectRevert(OutOfOrderResolution.selector);
        gameProxy.resolve();
    }

    /// @dev Static unit test asserting that resolve reverts when the game state is
    ///      not in progress.
    function test_resolve_notInProgress_reverts() public {
        uint256 chalWins = uint256(GameStatus.CHALLENGER_WINS);

        // Replace the game status in storage. It exists in slot 0 at offset 16.
        uint256 slot = uint256(vm.load(address(gameProxy), bytes32(0)));
        uint256 offset = 16 << 3;
        uint256 mask = 0xFF << offset;
        // Replace the byte in the slot value with the challenger wins status.
        slot = (slot & ~mask) | (chalWins << offset);

        vm.store(address(gameProxy), bytes32(uint256(0)), bytes32(slot));
        vm.expectRevert(GameNotInProgress.selector);
        gameProxy.resolveClaim(0);
    }

    /// @dev Static unit test for the correctness of resolving a single attack game state.
    function test_resolve_rootContested_succeeds() public {
        gameProxy.attack(0, Claim.wrap(bytes32(uint256(5))));

        vm.warp(block.timestamp + 3 days + 12 hours + 1 seconds);

        gameProxy.resolveClaim(0);
        assertEq(uint8(gameProxy.resolve()), uint8(GameStatus.CHALLENGER_WINS));
    }

    /// @dev Static unit test for the correctness of resolving a game with a contested challenge claim.
    function test_resolve_challengeContested_succeeds() public {
        gameProxy.attack(0, Claim.wrap(bytes32(uint256(5))));
        gameProxy.defend(1, Claim.wrap(bytes32(uint256(6))));

        vm.warp(block.timestamp + 3 days + 12 hours + 1 seconds);

        gameProxy.resolveClaim(1);
        gameProxy.resolveClaim(0);
        assertEq(uint8(gameProxy.resolve()), uint8(GameStatus.DEFENDER_WINS));
    }

    /// @dev Static unit test for the correctness of resolving a game with multiplayer moves.
    function test_resolve_teamDeathmatch_succeeds() public {
        gameProxy.attack(0, Claim.wrap(bytes32(uint256(5))));
        gameProxy.attack(0, Claim.wrap(bytes32(uint256(4))));
        gameProxy.defend(1, Claim.wrap(bytes32(uint256(6))));
        gameProxy.defend(1, Claim.wrap(bytes32(uint256(7))));

        vm.warp(block.timestamp + 3 days + 12 hours + 1 seconds);

        gameProxy.resolveClaim(1);
        gameProxy.resolveClaim(0);
        assertEq(uint8(gameProxy.resolve()), uint8(GameStatus.CHALLENGER_WINS));
    }

    /// @dev Static unit test for the correctness of resolving a game that reaches max game depth.
    function test_resolve_stepReached_succeeds() public {
        Claim dummyClaim = Claim.wrap(bytes32(uint256(5)));
        gameProxy.attack(0, dummyClaim);
        gameProxy.attack(1, dummyClaim);

        dummyClaim = changeClaimStatus(dummyClaim, VMStatuses.PANIC);

        gameProxy.attack(2, dummyClaim);
        gameProxy.attack(3, dummyClaim);

        vm.warp(block.timestamp + 3 days + 12 hours + 1 seconds);

        // resolving claim at 4 isn't necessary
        gameProxy.resolveClaim(3);
        gameProxy.resolveClaim(2);
        gameProxy.resolveClaim(1);
        gameProxy.resolveClaim(0);
        assertEq(uint8(gameProxy.resolve()), uint8(GameStatus.DEFENDER_WINS));
    }

    /// @dev Static unit test asserting that resolve reverts when attempting to resolve a subgame multiple times
    function test_resolve_claimAlreadyResolved_reverts() public {
        gameProxy.attack(0, Claim.wrap(bytes32(uint256(5))));
        gameProxy.attack(1, Claim.wrap(bytes32(uint256(5))));

        vm.warp(block.timestamp + 3 days + 12 hours + 1 seconds);

        gameProxy.resolveClaim(1);
        vm.expectRevert(ClaimAlreadyResolved.selector);
        gameProxy.resolveClaim(1);
    }

    /// @dev Static unit test asserting that resolve reverts when attempting to resolve a subgame at max depth
    function test_resolve_claimAtMaxDepthAlreadyResolved_reverts() public {
        Claim dummyClaim = Claim.wrap(bytes32(uint256(5)));
        gameProxy.attack(0, dummyClaim);
        gameProxy.attack(1, dummyClaim);

        dummyClaim = changeClaimStatus(dummyClaim, VMStatuses.PANIC);

        gameProxy.attack(2, dummyClaim);
        gameProxy.attack(3, dummyClaim);

        vm.warp(block.timestamp + 3 days + 12 hours + 1 seconds);

        vm.expectRevert(ClaimAlreadyResolved.selector);
        gameProxy.resolveClaim(4);
    }

    /// @dev Static unit test asserting that resolve reverts when attempting to resolve subgames out of order
    function test_resolve_outOfOrderResolution_reverts() public {
        gameProxy.attack(0, Claim.wrap(bytes32(uint256(5))));
        gameProxy.attack(1, Claim.wrap(bytes32(uint256(5))));

        vm.warp(block.timestamp + 3 days + 12 hours + 1 seconds);

        vm.expectRevert(OutOfOrderResolution.selector);
        gameProxy.resolveClaim(0);
    }

    /// @dev Tests that adding local data with an out of bounds identifier reverts.
    function testFuzz_addLocalData_oob_reverts(uint256 _ident) public {
        // Get a claim below the split depth so that we can add local data for an execution trace subgame.
        gameProxy.attack(0, Claim.wrap(bytes32(uint256(5))));
        gameProxy.attack(1, Claim.wrap(bytes32(uint256(5))));
        gameProxy.attack(2, ROOT_CLAIM);

        // [1, 5] are valid local data identifiers.
        if (_ident <= 5) _ident = 0;

        vm.expectRevert(InvalidLocalIdent.selector);
        gameProxy.addLocalData(_ident, 3, 0);
    }

    /// @dev Helper to get the localized key for an identifier in the context of the game proxy.
    function _getKey(uint256 _ident, bytes32 _localContext) internal view returns (bytes32) {
        bytes32 h = keccak256(abi.encode(_ident | (1 << 248), address(gameProxy), _localContext));
        return bytes32((uint256(h) & ~uint256(0xFF << 248)) | (1 << 248));
    }

    /// @dev Helper to change the VM status byte of a claim.
    function changeClaimStatus(Claim _claim, VMStatus _status) public pure returns (Claim out_) {
        assembly {
            out_ := or(and(not(shl(248, 0xFF)), _claim), shl(248, _status))
        }
    }
}

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { Test } from "forge-std/Test.sol";
import { Vm } from "forge-std/Vm.sol";
import { DisputeGameFactory_Init } from "./DisputeGameFactory.t.sol";
import { DisputeGameFactory } from "../dispute/DisputeGameFactory.sol";
import { FaultDisputeGame } from "../dispute/FaultDisputeGame.sol";

import "../libraries/DisputeTypes.sol";
import "../libraries/DisputeErrors.sol";
import { LibClock } from "../dispute/lib/LibClock.sol";
import { LibPosition } from "../dispute/lib/LibPosition.sol";
import { IBigStepper } from "../dispute/interfaces/IBigStepper.sol";

contract FaultDisputeGame_Init is DisputeGameFactory_Init {
    /// @dev The extra data passed to the game for initialization.
    bytes internal constant EXTRA_DATA = abi.encode(1);
    /// @dev The type of the game being tested.
    GameType internal constant GAME_TYPE = GameType.wrap(0);

    /// @dev The implementation of the game.
    FaultDisputeGame internal gameImpl;
    /// @dev The `Clone` proxy of the game.
    FaultDisputeGame internal gameProxy;

    event Move(uint256 indexed parentIndex, Claim indexed pivot, address indexed claimant);

    function init(Claim rootClaim, Claim absolutePrestate) public {
        super.setUp();
        // Deploy an implementation of the fault game
        gameImpl = new FaultDisputeGame(absolutePrestate, 4, new AlphabetVM(absolutePrestate));
        // Register the game implementation with the factory.
        factory.setImplementation(GAME_TYPE, gameImpl);
        // Create a new game.
        gameProxy = FaultDisputeGame(address(factory.create(GAME_TYPE, rootClaim, EXTRA_DATA)));

        // Label the proxy
        vm.label(address(gameProxy), "FaultDisputeGame_Clone");
    }
}

contract FaultDisputeGame_Test is FaultDisputeGame_Init {
    /// @dev The root claim of the game.
    Claim internal constant ROOT_CLAIM = Claim.wrap(bytes32(uint256(10)));
    /// @dev The absolute prestate of the trace.
    Claim internal constant ABSOLUTE_PRESTATE = Claim.wrap(bytes32(uint256(0)));

    function setUp() public override {
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
        assertEq(gameProxy.extraData(), EXTRA_DATA);
    }

    /// @dev Tests that the game's status is set correctly.
    function test_gameStart_succeeds() public {
        assertEq(Timestamp.unwrap(gameProxy.gameStart()), block.timestamp);
    }

    /// @dev Tests that the game's type is set correctly.
    function test_gameType_succeeds() public {
        assertEq(GameType.unwrap(gameProxy.gameType()), GameType.unwrap(GAME_TYPE));
    }

    /// @dev Tests that the game's data is set correctly.
    function test_gameData_succeeds() public {
        (GameType gameType, Claim rootClaim, bytes memory extraData) = gameProxy.gameData();

        assertEq(GameType.unwrap(gameType), GameType.unwrap(GAME_TYPE));
        assertEq(Claim.unwrap(rootClaim), Claim.unwrap(ROOT_CLAIM));
        assertEq(extraData, EXTRA_DATA);
    }

    ////////////////////////////////////////////////////////////////
    //          `IFaultDisputeGame` Implementation Tests          //
    ////////////////////////////////////////////////////////////////

    /// @dev Tests that the root claim's data is set correctly when the game is initialized.
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

    /// @dev Tests that a move while the game status is not `IN_PROGRESS` causes the call to revert
    ///      with the `GameNotInProgress` error
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

    /// @dev Tests that an attempt to defend the root claim reverts with the `CannotDefendRootClaim` error.
    function test_defendRoot_invalidMove_reverts() public {
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

    /// @dev Static unit test for the correctness of an opening attack.
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
        assertEq(Position.unwrap(position), Position.unwrap(Position.wrap(1).move(true)));
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

    /// @dev Static unit test for the correctness an uncontested root resolution.
    function test_resolve_rootUncontested() public {
        GameStatus status = gameProxy.resolve();
        assertEq(uint8(status), uint8(GameStatus.DEFENDER_WINS));
        assertEq(uint8(gameProxy.status()), uint8(GameStatus.DEFENDER_WINS));
    }

    /// @dev Static unit test asserting that resolve reverts when the game state is
    ///      not in progress.
    function test_resolve_notInProgress_reverts() public {
        uint256 chalWins = uint256(GameStatus.CHALLENGER_WINS);

        // Replace the game status in storage. It exists in slot 0 at offset 8.
        uint256 slot = uint256(vm.load(address(gameProxy), bytes32(0)));
        uint256 offset = (8 << 3);
        uint256 mask = 0xFF << offset;
        // Replace the byte in the slot value with the challenger wins status.
        slot = (slot & ~mask) | (chalWins << offset);

        vm.store(address(gameProxy), bytes32(uint256(0)), bytes32(slot));
        vm.expectRevert(GameNotInProgress.selector);
        gameProxy.resolve();
    }

    /// @dev Static unit test for the correctness of resolving a single attack game state.
    function test_resolve_rootContested() public {
        gameProxy.attack(0, Claim.wrap(bytes32(uint256(5))));

        GameStatus status = gameProxy.resolve();
        assertEq(uint8(status), uint8(GameStatus.CHALLENGER_WINS));
        assertEq(uint8(gameProxy.status()), uint8(GameStatus.CHALLENGER_WINS));
    }

    /// @dev Static unit test for the correctness of resolving a game with a contested challenge claim.
    function test_resolve_challengeContested() public {
        gameProxy.attack(0, Claim.wrap(bytes32(uint256(5))));
        gameProxy.defend(1, Claim.wrap(bytes32(uint256(6))));

        GameStatus status = gameProxy.resolve();
        assertEq(uint8(status), uint8(GameStatus.DEFENDER_WINS));
        assertEq(uint8(gameProxy.status()), uint8(GameStatus.DEFENDER_WINS));
    }

    /// @dev Static unit test for the correctness of resolving a game with multiplayer moves.
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

/// @notice A generic game player actor with a configurable trace.
/// @dev This actor always responds rationally with respect to their trace. The
///      `play` function can be overridden to change this behavior.
contract GamePlayer {
    bool public failedToStep;
    FaultDisputeGame public gameProxy;
    bytes public trace;

    GamePlayer internal counterParty;
    Vm internal vm;
    uint256 internal maxDepth;

    /// @notice Initializes the player
    function init(
        FaultDisputeGame _gameProxy,
        GamePlayer _counterParty,
        Vm _vm
    ) public {
        gameProxy = _gameProxy;
        counterParty = _counterParty;
        vm = _vm;
        maxDepth = _gameProxy.MAX_GAME_DEPTH();
    }

    /// @notice Perform the next move in the game.
    function play(uint256 _parentIndex) public virtual {
        // Grab the claim data at the parent index.
        (uint32 grandparentIndex, , Claim parentClaim, Position parentPos, ) = gameProxy.claimData(
            _parentIndex
        );

        // The position to move to.
        Position movePos;
        // May or may not be used.
        Position movePos2;
        // Signifies whether the move is an attack or not.
        bool isAttack;

        if (grandparentIndex == type(uint32).max) {
            // If the parent claim is the root claim, begin by attacking.
            movePos = parentPos.move(true);
            // Flag the move as an attack.
            isAttack = true;
        } else {
            // If the parent claim is not the root claim, check if we disagree with it and/or its grandparent
            // to determine our next move(s).

            // Fetch our claim at the parent's position.
            Claim ourParentClaim = claimAt(parentPos);

            // Fetch our claim at the grandparent's position.
            (, , Claim grandparentClaim, Position grandparentPos, ) = gameProxy.claimData(
                grandparentIndex
            );
            Claim ourGrandparentClaim = claimAt(grandparentPos);

            if (Claim.unwrap(ourParentClaim) != Claim.unwrap(parentClaim)) {
                // Attack parent.
                movePos = parentPos.move(true);
                // If we also disagree with the grandparent, attack it as well.
                if (Claim.unwrap(ourGrandparentClaim) != Claim.unwrap(grandparentClaim)) {
                    movePos2 = grandparentPos.move(true);
                }

                // Flag the move as an attack.
                isAttack = true;
            } else if (
                Claim.unwrap(ourParentClaim) == Claim.unwrap(parentClaim) &&
                Claim.unwrap(ourGrandparentClaim) == Claim.unwrap(grandparentClaim)
            ) {
                movePos = parentPos.move(false);
            }
        }

        // If we are past the maximum depth, break the recursion and step.
        if (movePos.depth() > maxDepth) {
            uint256 stateIndex;
            bytes memory preStateTrace;

            // First, we need to find the pre/post state index depending on whether we
            // are making an attack step or a defense step. If the index at depth of the
            // move position is 0, the prestate is the absolute prestate and we need to
            // do nothing.
            if (movePos.indexAtDepth() > 0) {
                Position leafPos = isAttack
                    ? Position.wrap(Position.unwrap(parentPos) - 1)
                    : Position.wrap(Position.unwrap(parentPos) + 1);
                Position statePos = leafPos;

                // Walk up until the valid position that commits to the prestate's
                // trace index is found.
                while (
                    Position.unwrap(statePos.parent().rightIndex(maxDepth)) ==
                    Position.unwrap(leafPos)
                ) {
                    statePos = statePos.parent();
                }

                // Now, search for the index of the claim that commits to the prestate's trace
                // index.
                uint256 len = claimDataLen();
                for (uint256 i = 0; i < len; i++) {
                    (, , , Position pos, ) = gameProxy.claimData(i);
                    if (Position.unwrap(pos) == Position.unwrap(statePos)) {
                        stateIndex = i;
                        break;
                    }
                }

                // Grab the trace up to the prestate's trace index.
                if (isAttack) {
                    preStateTrace = abi.encode(statePos.traceIndex(maxDepth), traceAt(statePos));
                } else {
                    preStateTrace = abi.encode(parentPos.traceIndex(maxDepth), traceAt(parentPos));
                }
            }

            // Perform the step and halt recursion.
            try gameProxy.step(stateIndex, _parentIndex, isAttack, preStateTrace, hex"") {
                // Do nothing, step succeeded.
            } catch {
                failedToStep = true;
            }
        } else {
            // Find the trace index that our next claim must commit to.
            uint256 traceIndex = movePos.traceIndex(maxDepth);
            // Grab the claim that we need to make from the helper.
            Claim ourClaim = claimAt(traceIndex);

            if (isAttack) {
                // Attack the parent claim.
                gameProxy.attack(_parentIndex, ourClaim);
                // Call out to our counter party to respond.
                counterParty.play(claimDataLen() - 1);

                // If we have a second move position, attack the grandparent.
                if (Position.unwrap(movePos2) != 0) {
                    (, , , Position grandparentPos, ) = gameProxy.claimData(grandparentIndex);
                    Claim ourGrandparentClaim = claimAt(grandparentPos.move(true));

                    gameProxy.attack(grandparentIndex, ourGrandparentClaim);
                    counterParty.play(claimDataLen() - 1);
                }
            } else {
                // Defend the parent claim.
                gameProxy.defend(_parentIndex, ourClaim);
                // Call out to our counter party to respond.
                counterParty.play(claimDataLen() - 1);
            }
        }
    }

    /// @notice Returns the length of the claim data array.
    function claimDataLen() internal view returns (uint256 len_) {
        return uint256(vm.load(address(gameProxy), bytes32(uint256(1))));
    }

    /// @notice Returns the state at the trace index within the player's trace.
    function traceAt(Position _position) public view returns (uint256 state_) {
        return traceAt(_position.traceIndex(maxDepth));
    }

    /// @notice Returns the state at the trace index within the player's trace.
    function traceAt(uint256 _traceIndex) public view returns (uint256 state_) {
        return uint256(uint8(trace[_traceIndex]));
    }

    /// @notice Returns the player's claim that commits to a given trace index.
    function claimAt(uint256 _traceIndex) public view returns (Claim claim_) {
        return Claim.wrap(keccak256(abi.encode(_traceIndex, traceAt(_traceIndex))));
    }

    /// @notice Returns the player's claim that commits to a given trace index.
    function claimAt(Position _position) public view returns (Claim claim_) {
        return claimAt(_position.traceIndex(maxDepth));
    }
}

contract OneVsOne_Arena is FaultDisputeGame_Init {
    /// @dev The absolute prestate of the trace.
    Claim internal constant ABSOLUTE_PRESTATE = Claim.wrap(bytes32(uint256(15)));
    /// @dev The defender.
    GamePlayer internal defender;
    /// @dev The challenger.
    GamePlayer internal challenger;

    function init(GamePlayer _defender, GamePlayer _challenger) public {
        Claim rootClaim = Claim.wrap(keccak256(abi.encode(15, _defender.traceAt(15))));
        super.init(rootClaim, ABSOLUTE_PRESTATE);
        defender = _defender;
        challenger = _challenger;

        // Set the counterparties.
        defender.init(gameProxy, challenger, vm);
        challenger.init(gameProxy, defender, vm);

        // Label actors for trace.
        vm.label(address(challenger), "Challenger");
        vm.label(address(defender), "Defender");
    }
}

contract FaultDisputeGame_ResolvesCorrectly_IncorrectRoot is OneVsOne_Arena {
    function setUp() public override {
        GamePlayer honest = new HonestPlayer(ABSOLUTE_PRESTATE);
        GamePlayer dishonest = new FullyDivergentPlayer(ABSOLUTE_PRESTATE);
        super.init(dishonest, honest);
    }

    function test_resolvesCorrectly_succeeds() public {
        // Play the game until a step is forced.
        challenger.play(0);

        // Resolve the game and assert that the honest player challenged the root
        // claim successfully.
        assertEq(uint8(gameProxy.resolve()), uint8(GameStatus.CHALLENGER_WINS));
        assertFalse(defender.failedToStep());
    }
}

contract FaultDisputeGame_ResolvesCorrectly_CorrectRoot is OneVsOne_Arena {
    function setUp() public override {
        GamePlayer honest = new HonestPlayer(ABSOLUTE_PRESTATE);
        GamePlayer dishonest = new FullyDivergentPlayer(ABSOLUTE_PRESTATE);
        super.init(honest, dishonest);
    }

    function test_resolvesCorrectly_succeeds() public {
        // Play the game until a step is forced.
        challenger.play(0);

        // Resolve the game and assert that the dishonest player challenged the root
        // claim unsuccessfully.
        assertEq(uint8(gameProxy.resolve()), uint8(GameStatus.DEFENDER_WINS));
        assertTrue(challenger.failedToStep());
    }
}

contract FaultDisputeGame_ResolvesCorrectly_IncorrectRoot2 is OneVsOne_Arena {
    function setUp() public override {
        GamePlayer honest = new HonestPlayer(ABSOLUTE_PRESTATE);
        GamePlayer dishonest = new HalfDivergentPlayer(ABSOLUTE_PRESTATE);
        super.init(dishonest, honest);
    }

    function test_resolvesCorrectly_succeeds() public {
        // Play the game until a step is forced.
        challenger.play(0);

        // Resolve the game and assert that the honest player challenged the root
        // claim successfully.
        assertEq(uint8(gameProxy.resolve()), uint8(GameStatus.CHALLENGER_WINS));
        assertFalse(defender.failedToStep());
    }
}

contract FaultDisputeGame_ResolvesCorrectly_CorrectRoot2 is OneVsOne_Arena {
    function setUp() public override {
        GamePlayer honest = new HonestPlayer(ABSOLUTE_PRESTATE);
        GamePlayer dishonest = new HalfDivergentPlayer(ABSOLUTE_PRESTATE);
        super.init(honest, dishonest);
    }

    function test_resolvesCorrectly_succeeds() public {
        // Play the game until a step is forced.
        challenger.play(0);

        // Resolve the game and assert that the dishonest player challenged the root
        // claim unsuccessfully.
        assertEq(uint8(gameProxy.resolve()), uint8(GameStatus.DEFENDER_WINS));
        assertTrue(challenger.failedToStep());
    }
}

contract FaultDisputeGame_ResolvesCorrectly_IncorrectRoot3 is OneVsOne_Arena {
    function setUp() public override {
        GamePlayer honest = new HonestPlayer(ABSOLUTE_PRESTATE);
        GamePlayer dishonest = new EarlyDivergentPlayer(ABSOLUTE_PRESTATE);
        super.init(dishonest, honest);
    }

    function test_resolvesCorrectly_succeeds() public {
        // Play the game until a step is forced.
        challenger.play(0);

        // Resolve the game and assert that the honest player challenged the root
        // claim successfully.
        assertEq(uint8(gameProxy.resolve()), uint8(GameStatus.CHALLENGER_WINS));
        assertFalse(defender.failedToStep());
    }
}

contract FaultDisputeGame_ResolvesCorrectly_CorrectRoot3 is OneVsOne_Arena {
    function setUp() public override {
        GamePlayer honest = new HonestPlayer(ABSOLUTE_PRESTATE);
        GamePlayer dishonest = new EarlyDivergentPlayer(ABSOLUTE_PRESTATE);
        super.init(honest, dishonest);
    }

    function test_resolvesCorrectly_succeeds() public {
        // Play the game until a step is forced.
        challenger.play(0);

        // Resolve the game and assert that the dishonest player challenged the root
        // claim unsuccessfully.
        assertEq(uint8(gameProxy.resolve()), uint8(GameStatus.DEFENDER_WINS));
        assertTrue(challenger.failedToStep());
    }
}

////////////////////////////////////////////////////////////////
//                           ACTORS                           //
////////////////////////////////////////////////////////////////

contract HonestPlayer is GamePlayer {
    constructor(Claim _absolutePrestate) {
        uint8 absolutePrestate = uint8(uint256(Claim.unwrap(_absolutePrestate)));
        bytes memory honestTrace = new bytes(16);
        for (uint8 i = 0; i < honestTrace.length; i++) {
            honestTrace[i] = bytes1(absolutePrestate + i + 1);
        }
        trace = honestTrace;
    }
}

contract FullyDivergentPlayer is GamePlayer {
    constructor(Claim _absolutePrestate) {
        uint8 absolutePrestate = uint8(uint256(Claim.unwrap(_absolutePrestate)));
        bytes memory dishonestTrace = new bytes(16);
        for (uint8 i = 0; i < dishonestTrace.length; i++) {
            // Offset the honest trace by 1.
            dishonestTrace[i] = bytes1(absolutePrestate + i);
        }
        trace = dishonestTrace;
    }
}

contract HalfDivergentPlayer is GamePlayer {
    constructor(Claim _absolutePrestate) {
        uint8 absolutePrestate = uint8(uint256(Claim.unwrap(_absolutePrestate)));
        bytes memory dishonestTrace = new bytes(16);
        for (uint8 i = 0; i < dishonestTrace.length; i++) {
            // Offset the trace after the first half.
            dishonestTrace[i] = i > 7 ? bytes1(i) : bytes1(absolutePrestate + i + 1);
        }
        trace = dishonestTrace;
    }
}

contract EarlyDivergentPlayer is GamePlayer {
    constructor(Claim _absolutePrestate) {
        uint8 absolutePrestate = uint8(uint256(Claim.unwrap(_absolutePrestate)));
        bytes memory dishonestTrace = new bytes(16);
        for (uint8 i = 0; i < dishonestTrace.length; i++) {
            // Offset the trace after the first half.
            dishonestTrace[i] = i > 2 ? bytes1(i) : bytes1(absolutePrestate + i + 1);
        }
        trace = dishonestTrace;
    }
}

////////////////////////////////////////////////////////////////
//                          MOCK VMS                          //
////////////////////////////////////////////////////////////////

contract AlphabetVM is IBigStepper {
    Claim internal immutable ABSOLUTE_PRESTATE;

    constructor(Claim _absolutePrestate) {
        ABSOLUTE_PRESTATE = _absolutePrestate;
    }

    /// @inheritdoc IBigStepper
    function step(bytes calldata _stateData, bytes calldata)
        external
        view
        returns (bytes32 postState_)
    {
        uint256 traceIndex;
        uint256 claim;
        if (_stateData.length == 0) {
            // If the state data is empty, then the absolute prestate is the claim.
            traceIndex = 0;
            claim = uint256(Claim.unwrap(ABSOLUTE_PRESTATE));
        } else {
            // Otherwise, decode the state data.
            (traceIndex, claim) = abi.decode(_stateData, (uint256, uint256));
            traceIndex++;
        }
        // STF: n -> n + 1
        postState_ = keccak256(abi.encode(traceIndex, claim + 1));
    }
}

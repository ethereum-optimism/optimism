// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { CommonBase } from "forge-std/Base.sol";

import { OutputBisectionGame } from "src/dispute/OutputBisectionGame.sol";
import { IOutputBisectionGame } from "src/dispute/interfaces/IOutputBisectionGame.sol";

import "src/libraries/DisputeTypes.sol";

/// @title GameSolver
/// @notice The `GameSolver` contract is a contract that can produce an array of available
///         moves for a given `OutputBisectionGame` contract, from the eyes of an honest
///         actor. The `GameSolver` does not implement functionality for acting on the `Move`s
///         it suggests.
abstract contract GameSolver is CommonBase {
    /// @notice The `OutputBisectionGame` proxy that the `GameSolver` will be solving.
    OutputBisectionGame public immutable GAME;
    /// @notice The maximum length of the execution trace, in bytes. Enforced by the depth of the
    ///         execution trace bisection game.
    uint256 internal immutable MAX_TRACE_LENGTH;
    /// @notice The split depth of the game
    uint256 internal immutable SPLIT_DEPTH;
    /// @notice The max depth of the game
    uint256 internal immutable MAX_DEPTH;
    /// @notice The starting L2 Block Number
    uint256 internal immutable STARTING_L2_BLOCK_NUMBER;

    /// @notice The execution trace that the `GameSolver` will be representing.
    bytes public trace;
    /// @notice The raw absolute prestate data.
    bytes public absolutePrestateData;
    /// @notice The offset of previously processed claims in the `GAME` contract's `claimData` array.
    ///         Starts at 0 and increments by 1 for each claim processed.
    uint256 public processedBuf;
    /// @notice Signals whether or not the `GameSolver` agrees with the root claim of the
    ///         `GAME` contract.
    bool public agreeWithRoot;

    /// @notice The `MoveKind` enum represents a kind of interaction with the `OutputBisectionGame` contract.
    enum MoveKind {
        Attack,
        Defend,
        Step,
        AddLocalData,
        ResolveLocal,
        ResolveGlobal
    }

    /// @notice The `Move` struct represents a move in the game, and contains information
    ///         about the kind of move, the sender of the move, and the calldata to be sent
    ///         to the `OutputBisectionGame` contract by a consumer of this contract.
    struct Move {
        MoveKind kind;
        bytes data;
    }

    constructor(
        OutputBisectionGame _gameProxy,
        bytes memory _trace,
        bytes memory _preStateData,
        uint256 _startingL2BlockNumber
    ) {
        GAME = _gameProxy;
        MAX_TRACE_LENGTH = 2 ** (_gameProxy.MAX_GAME_DEPTH() - _gameProxy.SPLIT_DEPTH());
        SPLIT_DEPTH = GAME.SPLIT_DEPTH();
        MAX_DEPTH = GAME.MAX_GAME_DEPTH();
        STARTING_L2_BLOCK_NUMBER = _startingL2BlockNumber;
        trace = _trace;
        absolutePrestateData = _preStateData;
    }

    /// @notice Returns an array of `Move`s that can be taken from the perspective of an honest
    ///         actor in the `OutputBisectionGame` contract.
    function solveGame() external virtual returns (Move[] memory moves_);
}

/// @title HonestGameSolver
/// @notice The `HonestGameSolver` is an implementation of `GameSolver` which responds accordingly depending
///         on the state of the `OutputBisectionGame` contract in relation to their local opinion of the correct
///         order of output roots and the execution trace between each block `n` -> `n + 1` state transition.
contract HonestGameSolver is GameSolver {
    /// @notice The `Direction` enum represents the direction of a proposed move in the game,
    ///         or a lack thereof.
    enum Direction {
        Defend,
        Attack,
        Noop
    }

    constructor(
        OutputBisectionGame _gameProxy,
        bytes memory _trace,
        bytes memory _preStateData,
        uint256 _startingL2BlockNumber
    )
        GameSolver(_gameProxy, _trace, _preStateData, _startingL2BlockNumber)
    {
        // Mark agreement with the root claim if the local opinion of the root claim is the same as the
        // observed root claim.
        if (Claim.unwrap(outputAt(MAX_TRACE_LENGTH)) == Claim.unwrap(_gameProxy.rootClaim())) {
            agreeWithRoot = true;
        }
    }

    ////////////////////////////////////////////////////////////////
    //                          EXTERNAL                          //
    ////////////////////////////////////////////////////////////////

    /// @notice Returns an array of `Move`s that can be taken from the perspective of an honest
    ///         actor in the `OutputBisectionGame` contract.
    function solveGame() external override returns (Move[] memory moves_) {
        uint256 numClaims = GAME.claimDataLen();

        // Pre-allocate the `moves_` array to the maximum possible length. Test environment, so
        // over-allocation is fine, and more easy to read than making a linked list in asm.
        uint256 movesLen = 0;
        moves_ = new Move[](numClaims - processedBuf);

        for (uint256 i = processedBuf; i < numClaims; i++) {
            // Grab the observed claim.
            IOutputBisectionGame.ClaimData memory observed = getClaimData(i);

            // Determine the direction of the next move to be taken.
            (Direction moveDirection, Position movePos) = determineDirection(observed);

            // Continue if there is no move to be taken against the observed claim.
            if (moveDirection == Direction.Noop) continue;

            if (movePos.depth() <= SPLIT_DEPTH) {
                // output bisection
                moves_[movesLen++] = handleOutputBisectionMove(moveDirection, movePos, i);
            } else if (movePos.depth() <= MAX_DEPTH) {
                // execution trace bisection
                moves_[movesLen++] = handleExecutionTraceBisectionMove(moveDirection, movePos, i);
            } else {
                // instruction step
                moves_[movesLen++] = handleStepMove(moveDirection, observed.position, movePos, i);
            }
        }

        // Update the length of the `moves_` array to the number of moves that were added.
        assembly {
            mstore(moves_, movesLen)
        }

        // Increment `processedBuf` by the number of claims processed, so that next time around,
        // we don't attempt to process the same claims again.
        processedBuf += numClaims - processedBuf;
    }

    ////////////////////////////////////////////////////////////////
    //                          INTERNAL                          //
    ////////////////////////////////////////////////////////////////

    /// @dev Helper function to determine the direction of the next move to be taken.
    function determineDirection(IOutputBisectionGame.ClaimData memory _claimData)
        internal
        view
        returns (Direction direction_, Position movePos_)
    {
        bool rightLevel = isRightLevel(_claimData.position);
        if (_claimData.parentIndex == type(uint32).max) {
            // If we agree with the parent claim and it is on a level we agree with, ignore it.
            if (Claim.unwrap(claimAt(_claimData.position)) == Claim.unwrap(_claimData.claim) && rightLevel) {
                return (Direction.Noop, Position.wrap(0));
            }

            // The parent claim is the root claim. We must attack if we disagree per the game rules.
            direction_ = Direction.Attack;
            movePos_ = _claimData.position.move(true);
        } else {
            // If the parent claim is not the root claim, check if we disagree with it and/or its grandparent
            // to determine our next move.

            // Fetch the local opinion of the parent claim.
            Claim localParent = claimAt(_claimData.position);

            bool localAgree = Claim.unwrap(localParent) == Claim.unwrap(_claimData.claim);
            if (rightLevel) {
                // Never move against a claim on the right level. Even if it's wrong, if it's uncountered, it furthers
                // our goals.
                return (Direction.Noop, Position.wrap(0));
            } else {
                // NOTE: Poison not handled yet.
                if (!localAgree) {
                    // If we disagree with the observed claim, we must attack it.
                    movePos_ = _claimData.position.move(true);
                    direction_ = Direction.Attack;
                } else {
                    // If we agree with the observed claim, but disagree with the grandparent claim, we must defend
                    // the observed claim.
                    movePos_ = _claimData.position.move(false);
                    direction_ = Direction.Defend;
                }
            }
        }
    }

    /// @notice Returns a `Move` struct that represents an attack or defense move in the output bisection
    ///         portion of the dispute game.
    /// @dev Note: This function assumes that the `movePos` and `challengeIndex` are valid within the
    ///            output bisection context. This is enforced by the `solveGame` function.
    function handleOutputBisectionMove(
        Direction _direction,
        Position _movePos,
        uint256 _challengeIndex
    )
        internal
        view
        returns (Move memory move_)
    {
        bool isAttack = _direction == Direction.Attack;
        move_ = Move({
            kind: isAttack ? MoveKind.Attack : MoveKind.Defend,
            data: abi.encodeCall(OutputBisectionGame.move, (_challengeIndex, claimAt(_movePos), isAttack))
        });
    }

    /// @notice Returns a `Move` struct that represents an attack or defense move in the execution trace
    ///         bisection portion of the dispute game.
    /// @dev Note: This function assumes that the `movePos` and `challengeIndex` are valid within the
    ///            execution trace bisection context. This is enforced by the `solveGame` function.
    function handleExecutionTraceBisectionMove(
        Direction _direction,
        Position _movePos,
        uint256 _challengeIndex
    )
        internal
        view
        returns (Move memory move_)
    {
        bool isAttack = _direction == Direction.Attack;
        move_ = Move({
            kind: isAttack ? MoveKind.Attack : MoveKind.Defend,
            data: abi.encodeCall(OutputBisectionGame.move, (_challengeIndex, claimAt(_movePos), isAttack))
        });
    }

    /// @notice Returns a `Move` struct that represents a step move in the execution trace
    ///         bisection portion of the dispute game.
    /// @dev Note: This function assumes that the `movePos` and `challengeIndex` are valid within the
    ///            execution trace bisection context. This is enforced by the `solveGame` function.
    function handleStepMove(
        Direction _direction,
        Position _parentPos,
        Position _movePos,
        uint256 _challengeIndex
    )
        internal
        view
        returns (Move memory move_)
    {
        bool isAttack = _direction == Direction.Attack;
        bytes memory preStateTrace;

        // First, we need to find the pre/post state index depending on whether we
        // are making an attack step or a defense step. If the index at depth of the
        // move position is 0, the prestate is the absolute prestate and we need to
        // do nothing.
        if ((_movePos.indexAtDepth() % (2 ** (MAX_DEPTH - SPLIT_DEPTH))) > 0) {
            // Grab the trace up to the prestate's trace index.
            if (isAttack) {
                Position leafPos = Position.wrap(Position.unwrap(_parentPos) - 1);
                preStateTrace = abi.encode(leafPos.traceIndex(MAX_DEPTH), stateAt(leafPos));
            } else {
                preStateTrace = abi.encode(_parentPos.traceIndex(MAX_DEPTH), stateAt(_parentPos));
            }
        } else {
            preStateTrace = absolutePrestateData;
        }

        move_ = Move({
            kind: MoveKind.Step,
            data: abi.encodeCall(OutputBisectionGame.step, (_challengeIndex, isAttack, preStateTrace, hex""))
        });
    }

    ////////////////////////////////////////////////////////////////
    //                          HELPERS                           //
    ////////////////////////////////////////////////////////////////

    /// @dev Helper function to get the `ClaimData` struct at a given index in the `GAME` contract's
    ///      `claimData` array.
    function getClaimData(uint256 _claimIndex)
        internal
        view
        returns (IOutputBisectionGame.ClaimData memory claimData_)
    {
        // thanks, solc
        (uint32 parentIndex, bool countered, Claim claim, Position position, Clock clock) = GAME.claimData(_claimIndex);
        claimData_ = IOutputBisectionGame.ClaimData({
            parentIndex: parentIndex,
            countered: countered,
            claim: claim,
            position: position,
            clock: clock
        });
    }

    /// @notice Returns the player's claim that commits to a given position, swapping between
    ///         output bisection claims and execution trace bisection claims depending on the depth
    function claimAt(Position _position) public view returns (Claim claim_) {
        return _position.depth() > SPLIT_DEPTH ? statehashAt(_position) : outputAt(_position);
    }

    /// @notice Returns the mock output at the given position.
    function outputAt(Position _position) public view returns (Claim claim_) {
        // Don't allow for positions that are deeper than the split depth.
        if (_position.depth() > SPLIT_DEPTH) {
            revert("GameSolver: invalid position depth");
        }

        return outputAt(_position.traceIndex(SPLIT_DEPTH) + 1);
    }

    /// @notice Returns the mock output at the given L2 block number.
    function outputAt(uint256 _l2BlockNumber) public view returns (Claim claim_) {
        return Claim.wrap(bytes32(STARTING_L2_BLOCK_NUMBER + _l2BlockNumber));
    }

    /// @notice Returns the player's claim that commits to a given trace index.
    function statehashAt(uint256 _traceIndex) public view returns (Claim claim_) {
        bytes32 hash =
            keccak256(abi.encode(_traceIndex >= trace.length ? trace.length - 1 : _traceIndex, stateAt(_traceIndex)));
        assembly {
            claim_ := or(and(hash, not(shl(248, 0xFF))), shl(248, 1))
        }
    }

    /// @notice Returns the player's claim that commits to a given trace index.
    function statehashAt(Position _position) public view returns (Claim claim_) {
        return statehashAt(_position.traceIndex(MAX_DEPTH));
    }

    /// @notice Returns the state at the trace index within the player's trace.
    function stateAt(Position _position) public view returns (uint256 state_) {
        return stateAt(_position.traceIndex(MAX_DEPTH));
    }

    /// @notice Returns the state at the trace index within the player's trace.
    function stateAt(uint256 _traceIndex) public view returns (uint256 state_) {
        return uint256(uint8(_traceIndex >= trace.length ? trace[trace.length - 1] : trace[_traceIndex]));
    }

    /// @notice Returns whether or not the position is on a level which opposes the local opinion of the
    ///         root claim.
    function isRightLevel(Position _position) public view returns (bool isRightLevel_) {
        isRightLevel_ = agreeWithRoot == (_position.depth() % 2 == 0);
    }
}

/// @title DisputeActor
/// @notice The `DisputeActor` contract is an abstract contract that represents an actor
///         that consumes the suggested moves from a `GameSolver` contract.
abstract contract DisputeActor {
    /// @notice The `GameSolver` contract used to determine the moves to be taken.
    GameSolver public solver;

    /// @notice Performs all available moves deemed by the attached solver.
    /// @return numMoves_ The number of moves that the actor took.
    /// @return success_ True if all moves were successful, false otherwise.
    function move() external virtual returns (uint256 numMoves_, bool success_);
}

/// @title HonestDisputeActor
/// @notice An actor that consumes the suggested moves from an `HonestGameSolver` contract. Note
///         that this actor *can* be dishonest if the trace is faulty, but it will always follow
///         the rules of the honest actor.
contract HonestDisputeActor is DisputeActor {
    OutputBisectionGame public immutable GAME;

    constructor(
        OutputBisectionGame _gameProxy,
        bytes memory _trace,
        bytes memory _preStateData,
        uint256 _startingL2BlockNumber
    ) {
        GAME = _gameProxy;
        solver = GameSolver(new HonestGameSolver(_gameProxy, _trace, _preStateData, _startingL2BlockNumber));
    }

    /// @inheritdoc DisputeActor
    function move() external override returns (uint256 numMoves_, bool success_) {
        GameSolver.Move[] memory moves = solver.solveGame();
        numMoves_ = moves.length;

        success_ = true;
        for (uint256 i = 0; i < moves.length; i++) {
            (bool innerSuccess,) = address(GAME).call(moves[i].data);
            success_ = success_ && innerSuccess;
        }
    }
}

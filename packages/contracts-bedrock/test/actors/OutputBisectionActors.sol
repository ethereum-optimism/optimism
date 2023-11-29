// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { Vm } from "forge-std/Vm.sol";

import { OutputBisectionGame } from "src/dispute/OutputBisectionGame.sol";
import { IOutputBisectionGame } from "src/dispute/interfaces/IOutputBisectionGame.sol";

import "src/libraries/DisputeTypes.sol";

/// @title GameSolver
/// @notice The `GameSolver` contract is a contract that can produce an array of available
///         moves for a given `OutputBisectionGame` contract, from the eyes of an honest
///         actor. The `GameSolver` does not implement functionality for acting on the `Move`s
///         it suggests.
abstract contract GameSolver {
    /// @notice The HEVM cheatcode address.
    Vm internal immutable VM;
    /// @notice The `OutputBisectionGame` proxy that the `GameSolver` will be solving.
    OutputBisectionGame public immutable GAME;
    /// @notice The maximum length of the execution trace, in bytes. Enforced by the depth of the
    ///         execution trace bisection game.
    uint256 internal immutable MAX_TRACE_LENGTH;

    /// @notice The execution trace that the `GameSolver` will be representing.
    bytes public trace;
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
        address sender;
        bytes data;
    }

    constructor(Vm _vm, OutputBisectionGame _gameProxy, bytes memory _trace) {
        VM = _vm;
        GAME = _gameProxy;
        MAX_TRACE_LENGTH = 2 ** (_gameProxy.MAX_GAME_DEPTH() - _gameProxy.SPLIT_DEPTH());
        trace = _trace;
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

    constructor(Vm _vm, OutputBisectionGame _gameProxy, bytes memory _trace) GameSolver(_vm, _gameProxy, _trace) {
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

            if (movePos.depth() <= GAME.SPLIT_DEPTH()) {
                // output bisection
                moves_[movesLen++] = handleOutputBisectionMove(moveDirection, movePos, i);
            } else if (movePos.depth() <= GAME.MAX_GAME_DEPTH()) {
                // execution trace bisection
                moves_[movesLen++] = handleExecutionTraceBisectionMove(moveDirection, movePos, i);
            } else {
                // instruction step
                moves_[movesLen++] = handleStepMove(moveDirection, movePos, i);
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
        if (_claimData.parentIndex == type(uint32).max) {
            // If we agree with the parent claim and it is on a level we agree with, ignore it.
            if (Claim.unwrap(claimAt(_claimData.position)) == Claim.unwrap(_claimData.claim)) {
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

            // Fetch the local opinion of the grandparent claim and the grandparent claim's metadata.
            IOutputBisectionGame.ClaimData memory grandparentClaimData = getClaimData(_claimData.parentIndex);
            Claim localGrandparent = claimAt(grandparentClaimData.position);

            if (Claim.unwrap(localParent) != Claim.unwrap(_claimData.claim)) {
                // If we disagree with the observed claim, we must attack it.
                movePos_ = _claimData.position.move(true);
                direction_ = Direction.Attack;
            } else if (
                Claim.unwrap(localParent) == Claim.unwrap(_claimData.claim)
                    && Claim.unwrap(localGrandparent) != Claim.unwrap(grandparentClaimData.claim)
            ) {
                // Never defend a claim that the solver would have made.
                if (isRightLevel(_claimData.position)) {
                    return (Direction.Noop, Position.wrap(0));
                }

                // If we agree with the observed claim, but disagree with the grandparent claim, we must defend
                // the observed claim.
                movePos_ = _claimData.position.move(false);
                direction_ = Direction.Defend;
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
            sender: address(this), // TODO: Change to allow for alt actors?
            data: abi.encodeCall(OutputBisectionGame.move, (_challengeIndex, outputAt(_movePos), isAttack))
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
            sender: address(this), // TODO: Change to allow for alt actors?
            data: abi.encodeCall(OutputBisectionGame.move, (_challengeIndex, claimAt(_movePos), isAttack))
        });
    }

    /// @notice Returns a `Move` struct that represents a step move in the execution trace
    ///         bisection portion of the dispute game.
    /// @dev Note: This function assumes that the `movePos` and `challengeIndex` are valid within the
    ///            execution trace bisection context. This is enforced by the `solveGame` function.
    /// @dev TODO: Handle new format for `AlphabetVM` once it's been refactored for Output Bisection.
    function handleStepMove(
        Direction _direction,
        Position _movePos,
        uint256 _challengeIndex
    )
        internal
        view
        returns (Move memory move_)
    {
        bool isAttack = _direction == Direction.Attack;
        Position parentPos = _movePos.parent();
        bytes memory preStateTrace;

        // First, we need to find the pre/post state index depending on whether we
        // are making an attack step or a defense step. If the index at depth of the
        // move position is 0, the prestate is the absolute prestate and we need to
        // do nothing.
        if (_movePos.indexAtDepth() > 0) {
            Position leafPos =
                isAttack ? Position.wrap(Position.unwrap(parentPos) - 1) : Position.wrap(Position.unwrap(parentPos) + 1);
            Position statePos = leafPos.traceAncestor();

            // Grab the trace up to the prestate's trace index.
            if (isAttack) {
                preStateTrace = abi.encode(statePos.traceIndex(GAME.MAX_GAME_DEPTH()), traceAt(statePos));
            } else {
                preStateTrace = abi.encode(parentPos.traceIndex(GAME.MAX_GAME_DEPTH()), traceAt(parentPos));
            }
        } else {
            // TODO: Prestate trace value.
            preStateTrace = abi.encode(0xdead);
        }

        move_ = Move({
            kind: MoveKind.Step,
            sender: address(this), // TODO: Change to allow for alt actors?
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

    /// @notice Returns the mock output at the given position.
    /// @dev TODO: Variability for malicious actors?
    function outputAt(Position _position) public view returns (Claim claim_) {
        // Don't allow for positions that are deeper than the split depth.
        if (_position.depth() > GAME.SPLIT_DEPTH()) {
            revert("GameSolver: invalid position depth");
        }

        return outputAt(_position.traceIndex(GAME.SPLIT_DEPTH()));
    }

    /// @notice Returns the mock output at the given L2 block number.
    /// @dev TODO: Variability for malicious actors?
    function outputAt(uint256 _l2BlockNumber) public pure returns (Claim claim_) {
        return Claim.wrap(bytes32(_l2BlockNumber));
    }

    /// @notice Returns the state at the trace index within the player's trace.
    /// @dev TODO: Separate traces per execution trace game.
    function traceAt(Position _position) public view returns (uint256 state_) {
        return traceAt(_position.traceIndex(GAME.MAX_GAME_DEPTH()));
    }

    /// @notice Returns the state at the trace index within the player's trace.
    /// @dev TODO: Separate traces per execution trace game.
    function traceAt(uint256 _traceIndex) public view returns (uint256 state_) {
        return uint256(uint8(_traceIndex >= trace.length ? trace[trace.length - 1] : trace[_traceIndex]));
    }

    /// @notice Returns the player's claim that commits to a given trace index.
    function claimAt(uint256 _traceIndex) public view returns (Claim claim_) {
        bytes32 hash =
            keccak256(abi.encode(_traceIndex >= trace.length ? trace.length - 1 : _traceIndex, traceAt(_traceIndex)));
        assembly {
            claim_ := or(and(hash, not(shl(248, 0xFF))), shl(248, 1))
        }
    }

    /// @notice Returns the player's claim that commits to a given trace index.
    function claimAt(Position _position) public view returns (Claim claim_) {
        return claimAt(_position.traceIndex(GAME.MAX_GAME_DEPTH()));
    }

    /// @notice Returns whether or not the position is on a level which opposes the local opinion of the
    ///         root claim.
    function isRightLevel(Position _position) public view returns (bool isRightLevel_) {
        return _position.depth() % 2 == 0 && agreeWithRoot;
    }
}

// TODO: `DishonestGameSolver`. Can remove a lot of the cruft and just throw bad claims
//        at the wall.
// TODO: Actors that utilize the `HonestGameSolver`

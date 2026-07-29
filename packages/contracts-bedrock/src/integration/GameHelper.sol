// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IDisputeGameFactory } from "../../interfaces/dispute/IDisputeGameFactory.sol";
import { IFaultDisputeGame } from "../../interfaces/dispute/IFaultDisputeGame.sol";

// Libraries
import { Claim, Position, GameType } from "src/dispute/lib/Types.sol";

/// @title GameHelper
/// @notice GameHelper is a util contract for testing to perform multiple moves in a dispute game in a single
/// transaction. Note that it is unsafe to use in production as the bonds paid cannot be recovered.
contract GameHelper {
    struct Move {
        uint256 parentIdx;
        Claim claim;
        bool attack;
    }

    /// @notice Performs the specified set of moves in the supplied dispute game.
    /// @param _game the game to perform moves in.
    /// @param _moves the moves to perform.
    function performMoves(IFaultDisputeGame _game, Move[] calldata _moves) public payable {
        uint256 movesLen = _moves.length;
        for (uint256 i = 0; i < movesLen; i++) {
            Move memory move = _moves[i];
            (,,,, Claim pClaim, Position pPosition,) = _game.claimData(move.parentIdx);
            uint256 requiredBond = _game.getRequiredBond(pPosition.move(move.attack));
            _game.move{ value: requiredBond }(pClaim, move.parentIdx, move.claim, move.attack);
        }
    }

    /// @notice Creates a new game and performs the specified moves in it.
    /// @param _dgf the DisputeGameFactory to create a game in.
    /// @param _gameType the type of game to create.
    /// @param _rootClaim the root claim of the new game.
    /// @param _extraData the extra data for the new game.
    /// @param _moves the array of moves to perform in the new game.
    /// @return gameAddr_ the address of the newly created game.
    function createGameWithClaims(
        IDisputeGameFactory _dgf,
        GameType _gameType,
        Claim _rootClaim,
        bytes memory _extraData,
        Move[] calldata _moves
    )
        external
        payable
        returns (address gameAddr_)
    {
        uint256 initBond = _dgf.initBonds(_gameType);
        gameAddr_ = address(_dgf.create{ value: initBond }(_gameType, _rootClaim, _extraData));
        IFaultDisputeGame game = IFaultDisputeGame(gameAddr_);
        performMoves(game, _moves);
    }

    // @notice Allows funds to be sent to this contract or to use it in a 7702 authorization.
    receive() external payable { }
}

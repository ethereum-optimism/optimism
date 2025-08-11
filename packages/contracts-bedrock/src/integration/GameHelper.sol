// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IDisputeGameFactory } from "../../interfaces/dispute/IDisputeGameFactory.sol";
import { IFaultDisputeGame } from "../../interfaces/dispute/IFaultDisputeGame.sol";

// Libraries
import { Claim, Position, GameType } from "src/dispute/lib/Types.sol";

contract GameHelper {
    struct Move {
        uint256 parentIdx;
        Claim claim;
        bool attack;
    }

    function performMoves(IFaultDisputeGame _game, Move[] calldata _moves) public payable {
        uint256 movesLen = _moves.length;
        for (uint256 i = 0; i < movesLen; i++) {
            Move memory move = _moves[i];
            (,,,, Claim pClaim, Position pPosition,) = _game.claimData(move.parentIdx);
            uint256 requiredBond = _game.getRequiredBond(pPosition.move(move.attack));
            _game.move{ value: requiredBond }(pClaim, move.parentIdx, move.claim, move.attack);
        }
    }

    function createGameWithClaims(
        IDisputeGameFactory _dgf,
        GameType _gameType,
        Claim _rootClaim,
        bytes memory _extraData,
        Move[] calldata _moves
    )
        external
        payable
        returns (address)
    {
        uint256 initBond = _dgf.initBonds(_gameType);
        IFaultDisputeGame game =
            IFaultDisputeGame(address(_dgf.create{ value: initBond }(_gameType, _rootClaim, _extraData)));
        performMoves(game, _moves);
        return address(game);
    }
}

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

import { Claim } from "src/types/Types.sol";
import { GameType } from "src/types/Types.sol";
import { GameStatus } from "src/types/Types.sol";

import { LibGames } from "src/lib/LibGames.sol";
import { IBondManager } from "src/interfaces/IBondManager.sol";
import { IDisputeGame } from "src/interfaces/IDisputeGame.sol";

// TODO: you wish this was fully implemented :P

/// @title ValidityDisputeGame
/// @author refcell <https://github.com/refcell>
/// @notice A validity-based dispute game.
contract ValidityDisputeGame is IDisputeGame {
    function initialize() external override { }

    function version() external pure override returns (string memory) {
        return "0.0.0";
    }

    function status() external pure override returns (GameStatus) {
        return GameStatus.IN_PROGRESS;
    }

    function gameType() external pure override returns (GameType) {
        return LibGames.ValidityGameType;
    }

    function extraData() external pure override returns (bytes memory) {
        return bytes("");
    }

    function bondManager() external pure override returns (IBondManager) {
        return IBondManager(address(0));
    }

    function rootClaim() external pure override returns (Claim _rootClaim) {
        return Claim.wrap(0);
    }

    function resolve() external pure override returns (GameStatus) {
        return GameStatus.IN_PROGRESS;
    }
}

// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { FaultDisputeGame } from "src/dispute/FaultDisputeGame.sol";
import { PermissionedDisputeGame } from "src/dispute/PermissionedDisputeGame.sol";
import { SuperFaultDisputeGame } from "src/dispute/SuperFaultDisputeGame.sol";
import { SuperPermissionedDisputeGame } from "src/dispute/SuperPermissionedDisputeGame.sol";
import { ZKDisputeGame } from "src/dispute/zk/ZKDisputeGame.sol";

// Libraries
import { Duration, GameStatus, Timestamp } from "src/dispute/lib/Types.sol";
import { ClaimAlreadyResolved, GameNotInProgress } from "src/dispute/lib/Errors.sol";

// Testing
import { KontrolUtils } from "./utils/KontrolUtils.sol";

contract FaultDisputeGameStatus_Harness is FaultDisputeGame {
    constructor()
        FaultDisputeGame(
            GameConstructorParams({
                maxGameDepth: 4,
                splitDepth: 2,
                clockExtension: Duration.wrap(0),
                maxClockDuration: Duration.wrap(0)
            })
        )
    { }

    function setChallengerWins(uint64 _resolvedAt) external {
        status = GameStatus.CHALLENGER_WINS;
        resolvedAt = Timestamp.wrap(_resolvedAt);
        initialized = true;
    }
}

contract PermissionedDisputeGameStatus_Harness is PermissionedDisputeGame {
    constructor()
        PermissionedDisputeGame(
            GameConstructorParams({
                maxGameDepth: 4,
                splitDepth: 2,
                clockExtension: Duration.wrap(0),
                maxClockDuration: Duration.wrap(0)
            })
        )
    { }

    function setChallengerWins(uint64 _resolvedAt) external {
        status = GameStatus.CHALLENGER_WINS;
        resolvedAt = Timestamp.wrap(_resolvedAt);
        initialized = true;
    }
}

contract SuperFaultDisputeGameStatus_Harness is SuperFaultDisputeGame {
    constructor()
        SuperFaultDisputeGame(
            GameConstructorParams({
                maxGameDepth: 4,
                splitDepth: 2,
                clockExtension: Duration.wrap(0),
                maxClockDuration: Duration.wrap(0)
            })
        )
    { }

    function setChallengerWins(uint64 _resolvedAt) external {
        status = GameStatus.CHALLENGER_WINS;
        resolvedAt = Timestamp.wrap(_resolvedAt);
        initialized = true;
    }
}

contract ZKDisputeGameStatus_Harness is ZKDisputeGame {
    function setChallengerWins(uint64 _resolvedAt) external {
        status = GameStatus.CHALLENGER_WINS;
        resolvedAt = Timestamp.wrap(_resolvedAt);
        initialized = true;
    }
}

contract SuperPermissionedDisputeGameStatus_Harness is SuperPermissionedDisputeGame {
    function setChallengerWins(uint64 _resolvedAt) external {
        status = GameStatus.CHALLENGER_WINS;
        resolvedAt = Timestamp.wrap(_resolvedAt);
        initialized = true;
    }
}

/// @notice Proves that a game whose stored outcome is `CHALLENGER_WINS` cannot later change its
///         stored outcome to `DEFENDER_WINS` through its resolution entry point. Each proof uses
///         an arbitrary resolution timestamp, so preserving it also proves that an old timestamp
///         cannot be paired with a newly favorable outcome. In the production implementations,
///         `resolve` is the only post-initialization writer of `status` and `resolvedAt`, making
///         this one-step property inductive over any number of subsequent calls.
contract DisputeGameStatusKontrol is KontrolUtils {
    function prove_faultDisputeGame_challengerWinsIsTerminal(uint64 _resolvedAt) external {
        FaultDisputeGameStatus_Harness game = new FaultDisputeGameStatus_Harness();
        game.setChallengerWins(_resolvedAt);

        vm.expectRevert(GameNotInProgress.selector);
        game.resolve();

        assert(game.status() == GameStatus.CHALLENGER_WINS);
        assert(game.resolvedAt().raw() == _resolvedAt);
    }

    function prove_permissionedDisputeGame_challengerWinsIsTerminal(uint64 _resolvedAt) external {
        PermissionedDisputeGameStatus_Harness game = new PermissionedDisputeGameStatus_Harness();
        game.setChallengerWins(_resolvedAt);

        vm.expectRevert(GameNotInProgress.selector);
        game.resolve();

        assert(game.status() == GameStatus.CHALLENGER_WINS);
        assert(game.resolvedAt().raw() == _resolvedAt);
    }

    function prove_superFaultDisputeGame_challengerWinsIsTerminal(uint64 _resolvedAt) external {
        SuperFaultDisputeGameStatus_Harness game = new SuperFaultDisputeGameStatus_Harness();
        game.setChallengerWins(_resolvedAt);

        vm.expectRevert(GameNotInProgress.selector);
        game.resolve();

        assert(game.status() == GameStatus.CHALLENGER_WINS);
        assert(game.resolvedAt().raw() == _resolvedAt);
    }

    function prove_zkDisputeGame_challengerWinsIsTerminal(uint64 _resolvedAt) external {
        ZKDisputeGameStatus_Harness game = new ZKDisputeGameStatus_Harness();
        game.setChallengerWins(_resolvedAt);

        vm.expectRevert(ClaimAlreadyResolved.selector);
        game.resolve();

        assert(game.status() == GameStatus.CHALLENGER_WINS);
        assert(game.resolvedAt().raw() == _resolvedAt);
    }

    function prove_superPermissionedDisputeGame_challengerWinsStorageIsTerminal(uint64 _resolvedAt) external {
        SuperPermissionedDisputeGameStatus_Harness game = new SuperPermissionedDisputeGameStatus_Harness();
        game.setChallengerWins(_resolvedAt);

        // SuperPermissionedDisputeGame resolves to DEFENDER_WINS during initialization. Its
        // resolve function is a pure compatibility no-op that returns DEFENDER_WINS without
        // modifying the stored status or resolution timestamp.
        assert(game.resolve() == GameStatus.DEFENDER_WINS);
        assert(game.status() == GameStatus.CHALLENGER_WINS);
        assert(game.resolvedAt().raw() == _resolvedAt);
    }
}

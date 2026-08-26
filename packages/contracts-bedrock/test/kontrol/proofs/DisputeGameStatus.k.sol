// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { FaultDisputeGame } from "src/dispute/FaultDisputeGame.sol";
import { PermissionedDisputeGame } from "src/dispute/PermissionedDisputeGame.sol";
import { SuperFaultDisputeGame } from "src/dispute/SuperFaultDisputeGame.sol";
import { SuperPermissionedDisputeGame } from "src/dispute/SuperPermissionedDisputeGame.sol";
import { ZKDisputeGame } from "src/dispute/zk/ZKDisputeGame.sol";
import { LibClone } from "@solady/utils/LibClone.sol";

// Libraries
import { Claim, Clock, Duration, GameStatus, GameType, GameTypes, Hash, Timestamp } from "src/dispute/lib/Types.sol";
import { Position } from "src/dispute/lib/LibPosition.sol";
import { Types } from "src/libraries/Types.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { ClaimAlreadyResolved, GameNotInProgress } from "src/dispute/lib/Errors.sol";

// Testing
import { KontrolUtils } from "./utils/KontrolUtils.sol";

contract SuperPermissionedRegistry_Harness {
    function getAnchorRoot() external pure returns (Hash, uint256) {
        return (Hash.wrap(bytes32(uint256(1))), 0);
    }

    function respectedGameType() external pure returns (GameType) {
        return GameTypes.SUPER_PERMISSIONED;
    }
}

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

    function prepareResolution(address _counteredBy) external {
        initialized = true;
        status = GameStatus.IN_PROGRESS;
        resolvedAt = Timestamp.wrap(0);
        claimData.push(
            ClaimData({
                parentIndex: 0,
                counteredBy: _counteredBy,
                claimant: address(0),
                bond: 0,
                claim: Claim.wrap(bytes32(0)),
                position: Position.wrap(1),
                clock: Clock.wrap(0)
            })
        );
        resolvedSubgames[0] = true;
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

    function prepareResolution(address _counteredBy) external {
        initialized = true;
        status = GameStatus.IN_PROGRESS;
        resolvedAt = Timestamp.wrap(0);
        claimData.push(
            ClaimData({
                parentIndex: 0,
                counteredBy: _counteredBy,
                claimant: address(0),
                bond: 0,
                claim: Claim.wrap(bytes32(0)),
                position: Position.wrap(1),
                clock: Clock.wrap(0)
            })
        );
        resolvedSubgames[0] = true;
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

    function prepareResolution(address _counteredBy) external {
        initialized = true;
        status = GameStatus.IN_PROGRESS;
        resolvedAt = Timestamp.wrap(0);
        claimData.push(
            ClaimData({
                parentIndex: 0,
                counteredBy: _counteredBy,
                claimant: address(0),
                bond: 0,
                claim: Claim.wrap(bytes32(0)),
                position: Position.wrap(1),
                clock: Clock.wrap(0)
            })
        );
        resolvedSubgames[0] = true;
    }
}

contract ZKDisputeGameStatus_Harness is ZKDisputeGame {
    function setChallengerWins(uint64 _resolvedAt) external {
        status = GameStatus.CHALLENGER_WINS;
        resolvedAt = Timestamp.wrap(_resolvedAt);
        initialized = true;
    }

    function prepareUnchallengedResolution() external {
        initialized = true;
        status = GameStatus.IN_PROGRESS;
        resolvedAt = Timestamp.wrap(0);
        claimData = ClaimData({
            parentIndex: type(uint32).max,
            status: ProposalStatus.Unchallenged,
            challenger: address(0),
            prover: address(0),
            deadline: Timestamp.wrap(0),
            claim: Claim.wrap(bytes32(0))
        });
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
    using LibClone for address;

    address internal constant SUPER_PERMISSIONED_PROPOSER = address(0xA11CE);

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

    /// @notice Proves that a successful favorable resolution cannot reuse an old timestamp: the
    ///         terminal status and `resolvedAt = block.timestamp` are written by the same call.
    function prove_faultDisputeGame_defenderResolutionStartsAirgap(uint64 _resolutionTime) external {
        vm.assume(_resolutionTime > 0);
        FaultDisputeGameStatus_Harness game = new FaultDisputeGameStatus_Harness();
        game.prepareResolution(address(0));
        vm.warp(_resolutionTime);

        assert(game.resolve() == GameStatus.DEFENDER_WINS);
        assert(game.status() == GameStatus.DEFENDER_WINS);
        assert(game.resolvedAt().raw() == _resolutionTime);
    }

    function prove_permissionedDisputeGame_defenderResolutionStartsAirgap(uint64 _resolutionTime) external {
        vm.assume(_resolutionTime > 0);
        PermissionedDisputeGameStatus_Harness game = new PermissionedDisputeGameStatus_Harness();
        game.prepareResolution(address(0));
        vm.warp(_resolutionTime);

        assert(game.resolve() == GameStatus.DEFENDER_WINS);
        assert(game.status() == GameStatus.DEFENDER_WINS);
        assert(game.resolvedAt().raw() == _resolutionTime);
    }

    function prove_superFaultDisputeGame_defenderResolutionStartsAirgap(uint64 _resolutionTime) external {
        vm.assume(_resolutionTime > 0);
        SuperFaultDisputeGameStatus_Harness game = new SuperFaultDisputeGameStatus_Harness();
        game.prepareResolution(address(0));
        vm.warp(_resolutionTime);

        assert(game.resolve() == GameStatus.DEFENDER_WINS);
        assert(game.status() == GameStatus.DEFENDER_WINS);
        assert(game.resolvedAt().raw() == _resolutionTime);
    }

    function prove_zkDisputeGame_defenderResolutionStartsAirgap(uint64 _resolutionTime) external {
        vm.assume(_resolutionTime > 0);
        ZKDisputeGameStatus_Harness implementation = new ZKDisputeGameStatus_Harness();
        // Only the static parent-index getter is needed on this path. A root-game sentinel makes
        // the actual implementation treat the already trusted anchor as DEFENDER_WINS.
        bytes memory immutableArgs =
            abi.encodePacked(address(this), bytes32(0), bytes32(0), uint32(0), type(uint32).max);
        ZKDisputeGameStatus_Harness game =
            ZKDisputeGameStatus_Harness(payable(address(implementation).clone(immutableArgs)));
        game.prepareUnchallengedResolution();
        vm.warp(_resolutionTime);

        assert(game.resolve() == GameStatus.DEFENDER_WINS);
        assert(game.status() == GameStatus.DEFENDER_WINS);
        assert(game.resolvedAt().raw() == _resolutionTime);
    }

    function prove_superPermissionedDisputeGame_initializationStartsAirgap(uint64 _resolutionTime) external {
        vm.assume(_resolutionTime > 0);
        // Keep the immutable clone runtime concrete for Kontrol's jump-destination analysis. The
        // state transition being proved is independent of proposer identity.
        vm.assume(tx.origin == SUPER_PERMISSIONED_PROPOSER);
        vm.warp(_resolutionTime);

        Types.OutputRootWithChainId[] memory outputRoots = new Types.OutputRootWithChainId[](1);
        outputRoots[0] = Types.OutputRootWithChainId({ chainId: 1, root: bytes32(uint256(2)) });
        Types.SuperRootProof memory proof =
            Types.SuperRootProof({ version: bytes1(uint8(1)), timestamp: 1, outputRoots: outputRoots });
        bytes memory extraData = Encoding.encodeSuperRootProof(proof);
        Claim rootClaim = Claim.wrap(Hashing.hashSuperRootProof(proof));
        SuperPermissionedRegistry_Harness registry = new SuperPermissionedRegistry_Harness();
        SuperPermissionedDisputeGameStatus_Harness implementation = new SuperPermissionedDisputeGameStatus_Harness();
        bytes memory immutableArgs = abi.encodePacked(
            address(this),
            rootClaim,
            bytes32(0),
            GameTypes.SUPER_PERMISSIONED,
            extraData,
            address(registry),
            SUPER_PERMISSIONED_PROPOSER
        );
        SuperPermissionedDisputeGameStatus_Harness game =
            SuperPermissionedDisputeGameStatus_Harness(payable(address(implementation).clone(immutableArgs)));

        game.initialize();

        assert(game.status() == GameStatus.DEFENDER_WINS);
        assert(game.createdAt().raw() == _resolutionTime);
        assert(game.resolvedAt().raw() == _resolutionTime);
    }
}

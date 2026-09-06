// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { OptimismPortal2 } from "src/L1/OptimismPortal2.sol";
import { KontrolUtils } from "./utils/KontrolUtils.sol";

// Libraries
import { GameStatus, Timestamp } from "src/dispute/lib/Types.sol";

// Interfaces
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";

/// @notice Dependency model for component proofs, not a verified dispute game.
contract WithdrawalGame_Harness {
    Timestamp public createdAt;
    GameStatus public status;

    function configure(uint64 _createdAt, GameStatus _status) external {
        createdAt = Timestamp.wrap(_createdAt);
        status = _status;
    }
}

/// @notice Models registry responses for component checks.
contract WithdrawalRegistry_Harness {
    bool internal valid;
    bool internal blacklisted;

    function configure(bool _valid, bool _blacklisted) external {
        valid = _valid;
        blacklisted = _blacklisted;
    }

    function isGameClaimValid(IDisputeGame) external view returns (bool) {
        return valid;
    }

    function isGameBlacklisted(IDisputeGame) external view returns (bool) {
        return blacklisted;
    }
}

/// @notice Establishes component-proof preconditions without overriding production functions.
/// @dev These setters are setup only. They are not transitions of the production protocol.
contract WithdrawalPortal_Harness is OptimismPortal2 {
    constructor(IAnchorStateRegistry _registry) OptimismPortal2(7 days) {
        anchorStateRegistry = _registry;
    }

    function record(bytes32 _hash, address _submitter, IDisputeGame _game, uint64 _provenAt) external {
        provenWithdrawals[_hash][_submitter] = ProvenWithdrawal(_game, _provenAt);
        proofSubmitters[_hash].push(_submitter);
    }

    function setFinalized(bytes32 _hash, bool _finalized) external {
        finalizedWithdrawals[_hash] = _finalized;
    }
}

/// @notice Portal component lemmas. See withdrawal-verification.md for remaining obligations.
/// @dev Stored proof provenance and registry correctness are deliberately not conclusions here.
contract WithdrawalAuthorizationKontrol is KontrolUtils {
    struct DeletionCase {
        bytes32 hash;
        address submitter;
        bytes32 otherHash;
        address otherSubmitter;
        address caller;
        uint64 provenAt;
        uint64 otherProvenAt;
        bool finalized;
        bool challengerWins;
        bool blacklisted;
    }

    WithdrawalPortal_Harness internal portal;
    WithdrawalRegistry_Harness internal registry;
    WithdrawalGame_Harness internal game;

    function setUp() public {
        registry = new WithdrawalRegistry_Harness();
        game = new WithdrawalGame_Harness();
        portal = new WithdrawalPortal_Harness(IAnchorStateRegistry(address(registry)));
    }

    /// @notice Acceptance and rejection under the fixture's authorization inputs.
    /// @dev Uses a fixed seven-day maturity delay.
    function prove_checkWithdrawal_equivalence(
        bytes32 _hash,
        address _submitter,
        uint64 _createdAt,
        uint64 _provenAt,
        uint64 _now,
        bool _finalized,
        bool _valid
    )
        external
    {
        game.configure(_createdAt, GameStatus.IN_PROGRESS);
        registry.configure(_valid, false);
        portal.record(_hash, _submitter, IDisputeGame(address(game)), _provenAt);
        portal.setFinalized(_hash, _finalized);
        vm.warp(_now);

        bool eligible = !_finalized && _provenAt != 0 && _provenAt > _createdAt && _now >= _provenAt
            && uint256(_now) - uint256(_provenAt) > 7 days && _valid;
        (bool accepted,) = address(portal).staticcall(abi.encodeCall(portal.checkWithdrawal, (_hash, _submitter)));

        assert(accepted == eligible);
        _assertRecord(_hash, _submitter, IDisputeGame(address(game)), _provenAt);
        assert(portal.finalizedWithdrawals(_hash) == _finalized);
    }

    /// @notice Deletion eligibility and preservation of a distinct record.
    /// @dev The record keys may share either hash or submitter. Dependencies return normally.
    function prove_deleteProvenWithdrawal_preservesOtherRecord(DeletionCase memory _case) external {
        vm.assume(_case.hash != _case.otherHash || _case.submitter != _case.otherSubmitter);
        IDisputeGame gameAddress = IDisputeGame(address(game));
        game.configure(0, _case.challengerWins ? GameStatus.CHALLENGER_WINS : GameStatus.DEFENDER_WINS);
        registry.configure(false, _case.blacklisted);
        portal.record(_case.hash, _case.submitter, gameAddress, _case.provenAt);
        portal.record(_case.otherHash, _case.otherSubmitter, gameAddress, _case.otherProvenAt);
        portal.setFinalized(_case.hash, _case.finalized);
        uint256 submittersBefore = portal.numProofSubmitters(_case.hash);
        uint256 otherSubmittersBefore = portal.numProofSubmitters(_case.otherHash);

        vm.prank(_case.caller);
        (bool deleted,) =
            address(portal).call(abi.encodeCall(portal.deleteProvenWithdrawal, (_case.hash, _case.submitter)));

        assert(deleted == (_case.provenAt != 0 && (_case.challengerWins || _case.blacklisted)));
        _assertRecord(
            _case.hash, _case.submitter, deleted ? IDisputeGame(address(0)) : gameAddress, deleted ? 0 : _case.provenAt
        );
        _assertRecord(_case.otherHash, _case.otherSubmitter, gameAddress, _case.otherProvenAt);
        assert(portal.finalizedWithdrawals(_case.hash) == _case.finalized);
        assert(portal.numProofSubmitters(_case.hash) == submittersBefore);
        assert(portal.numProofSubmitters(_case.otherHash) == otherSubmittersBefore);
    }

    function _assertRecord(bytes32 _hash, address _submitter, IDisputeGame _game, uint64 _provenAt) internal view {
        (IDisputeGame storedGame, uint64 storedAt) = portal.provenWithdrawals(_hash, _submitter);
        assert(storedGame == _game);
        assert(storedAt == _provenAt);
    }
}

// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { DeploymentSummaryFaultProofs } from "./utils/DeploymentSummaryFaultProofs.sol";
import { KontrolUtils } from "./utils/KontrolUtils.sol";

// Libraries
import { Claim, GameStatus, GameType, Timestamp } from "src/dispute/lib/Types.sol";
import { Types } from "src/libraries/Types.sol";

// Interfaces
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";

/// @notice Game reports and pause input, not a proof of game resolution or pause administration.
contract WithdrawalGame_Harness {
    Timestamp public createdAt;
    Timestamp public resolvedAt;
    GameStatus public status;
    bool public wasRespectedGameTypeWhenCreated;
    bool public paused;

    function configure(uint64 _created, uint64 _resolved, GameStatus _status, bool _respected, bool _paused) external {
        createdAt = Timestamp.wrap(_created);
        resolvedAt = Timestamp.wrap(_resolved);
        status = _status;
        wasRespectedGameTypeWhenCreated = _respected;
        paused = _paused;
    }

    function gameData() external pure returns (GameType, Claim, bytes memory) {
        return (GameType.wrap(0), Claim.wrap(bytes32(0)), abi.encode(uint256(1)));
    }
}

/// @notice Authorization through production proxies, Portal, registry and factory lookup code.
///         Records are seeded preconditions; inclusion and history preservation remain separate.
contract WithdrawalAuthorizationKontrol is DeploymentSummaryFaultProofs, KontrolUtils {
    struct AuthorizationCase {
        bytes32 withdrawalHash;
        address submitter;
        uint64 createdAt;
        uint64 resolvedAt;
        uint64 provenAt;
        uint64 retiredAt;
        uint64 now;
        uint8 status;
        uint8 finalized;
        uint32 registrationType;
        uint64 registrationTimestamp;
        address registeredGame;
        bool respected;
        uint8 blacklisted;
        bool paused;
    }

    IOptimismPortal2 internal portal;
    IAnchorStateRegistry internal registry;
    IDisputeGameFactory internal factory;
    WithdrawalGame_Harness internal game;
    bytes32 internal registrationSlot;
    uint256 internal proofDelay;
    uint256 internal gameDelay;

    function setUp() public {
        portal = IOptimismPortal2(payable(optimismPortalProxyAddress));
        registry = portal.anchorStateRegistry();
        factory = registry.disputeGameFactory();
        proofDelay = portal.proofMaturityDelaySeconds();
        gameDelay = registry.disputeGameFinalityDelaySeconds();
        game = new WithdrawalGame_Harness();

        // Slots/packing follow snapshots/storageLayout; assertions detect incorrect setup.
        bytes32 initialized = vm.load(address(registry), bytes32(0)) & bytes32(uint256(0xffff));
        vm.store(address(registry), bytes32(0), initialized | bytes32(uint256(uint160(address(game))) << 16));
        bytes32 uuid = keccak256(abi.encode(GameType.wrap(0), Claim.wrap(bytes32(0)), abi.encode(uint256(1))));
        registrationSlot = keccak256(abi.encode(uuid, uint256(103)));
        assert(address(portal.anchorStateRegistry()) == address(registry));
        assert(address(registry.systemConfig()) == address(game));
        assert(address(registry.disputeGameFactory()) == address(factory));
    }

    /// @notice Acceptance is equivalent to the combined Portal and registry eligibility predicate.
    function prove_checkWithdrawal_equivalence(AuthorizationCase memory _case) external {
        _check(_case);
    }

    /// @notice A concrete eligible witness prevents an always-reverting fixture from passing.
    function prove_checkWithdrawal_eligible_succeeds() external {
        assert(_check(_eligibleCase()));
    }

    /// @notice Neither finalizer can commit when the exact withdrawal's selected record is ineligible.
    ///         The dynamic bytes field has symbolic length; no concrete length annotation is used.
    function prove_finalizeWithdrawal_ineligible_reverts(
        AuthorizationCase memory _case,
        Types.WithdrawalTransaction memory _tx,
        address _caller,
        bool _externalProof
    )
        external
    {
        _case.withdrawalHash = _withdrawalHash(_tx);
        if (!_externalProof) _case.submitter = _caller;
        // This is the rejection premise, checked against the independent eligibility expression in _check.
        vm.assume(!_check(_case));
        assert(!_finalize(_tx, _case.submitter, _caller, _externalProof));
    }

    /// @notice Both entry points admit an eligible example, so an always-reverting fixture is insufficient.
    function prove_finalizeWithdrawal_eligible_succeeds(bool _externalProof) external {
        AuthorizationCase memory example = _eligibleCase();
        Types.WithdrawalTransaction memory withdrawal;
        withdrawal.sender = address(0x1234);
        withdrawal.target = address(0x5678);
        withdrawal.gasLimit = 100_000;
        example.submitter = address(0x9ABC);
        example.withdrawalHash = _withdrawalHash(withdrawal);
        assert(_check(example));
        assert(_finalize(withdrawal, example.submitter, example.submitter, _externalProof));
        assert(portal.finalizedWithdrawals(example.withdrawalHash));
    }

    function _finalize(
        Types.WithdrawalTransaction memory _tx,
        address _submitter,
        address _caller,
        bool _externalProof
    )
        internal
        returns (bool accepted_)
    {
        bytes memory callData = _externalProof
            ? abi.encodeCall(portal.finalizeWithdrawalTransactionExternalProof, (_tx, _submitter))
            : abi.encodeCall(portal.finalizeWithdrawalTransaction, (_tx));
        vm.prank(_caller);
        (accepted_,) = address(portal).call(callData);
    }

    function _eligibleCase() internal view returns (AuthorizationCase memory example) {
        example.createdAt = 1 days;
        example.provenAt = 2 days;
        example.resolvedAt = 3 days;
        example.now = 30 days;
        example.status = uint8(GameStatus.DEFENDER_WINS);
        example.registeredGame = address(game);
        example.respected = true;
    }

    function _withdrawalHash(Types.WithdrawalTransaction memory _tx) internal pure returns (bytes32) {
        return keccak256(abi.encode(_tx.nonce, _tx.sender, _tx.target, _tx.value, _tx.gasLimit, _tx.data));
    }

    function _check(AuthorizationCase memory _case) internal returns (bool accepted_) {
        vm.assume(_case.status <= uint8(GameStatus.DEFENDER_WINS));
        // Preserve the boolean domain without branching to convert flags for storage setup.
        vm.assume(_case.finalized <= 1);
        vm.assume(_case.blacklisted <= 1);
        game.configure(_case.createdAt, _case.resolvedAt, GameStatus(_case.status), _case.respected, _case.paused);
        // Independent 32/64/160-bit fields span every possible packed registration word.
        bytes32 registration = bytes32(
            (uint256(_case.registrationType) << 224) | (uint256(_case.registrationTimestamp) << 160)
                | uint256(uint160(_case.registeredGame))
        );
        vm.store(address(factory), registrationSlot, registration);
        vm.store(address(registry), bytes32(uint256(6)), bytes32(uint256(_case.retiredAt) << 32));
        vm.store(
            address(registry), keccak256(abi.encode(address(game), uint256(5))), bytes32(uint256(_case.blacklisted))
        );
        bytes32 recordSlot =
            keccak256(abi.encode(_case.submitter, keccak256(abi.encode(_case.withdrawalHash, uint256(57)))));
        vm.store(
            address(portal), recordSlot, bytes32(uint256(uint160(address(game))) | (uint256(_case.provenAt) << 160))
        );
        bytes32 finalizedSlot = keccak256(abi.encode(_case.withdrawalHash, uint256(51)));
        vm.store(address(portal), finalizedSlot, bytes32(uint256(_case.finalized)));
        vm.warp(_case.now);

        (accepted_,) =
            address(portal).staticcall(abi.encodeCall(portal.checkWithdrawal, (_case.withdrawalHash, _case.submitter)));
        bool eligible = _case.finalized == 0 && _case.provenAt != 0 && _case.provenAt > _case.createdAt
            && _case.now >= _case.provenAt && uint256(_case.now) - _case.provenAt > proofDelay
            && _case.registeredGame == address(game) && _case.blacklisted == 0 && _case.createdAt > _case.retiredAt
            && !_case.paused && _case.respected && _case.status == uint8(GameStatus.DEFENDER_WINS) && _case.resolvedAt != 0
            && _case.now >= _case.resolvedAt && uint256(_case.now) - _case.resolvedAt > gameDelay;
        assert(accepted_ == eligible);
        // STATICCALL also prevents writes in every dependency, including reverted executions.
        assert(
            vm.load(address(portal), recordSlot)
                == bytes32(uint256(uint160(address(game))) | (uint256(_case.provenAt) << 160))
        );
        assert(portal.finalizedWithdrawals(_case.withdrawalHash) == (_case.finalized != 0));
    }
}

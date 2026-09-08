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

/// @notice Normal game reports for an acceptance witness; game resolution itself remains upstream.
contract WithdrawalProofGame_Harness {
    Claim public immutable rootClaim;
    GameType public immutable gameType;
    Claim private immutable perChainRoot;
    uint256 private immutable chainId;
    Timestamp public resolvedAt;
    GameStatus public status;
    bool public constant wasRespectedGameTypeWhenCreated = true;

    constructor(Claim _outputRoot, uint256 _chainId, bool _superGame) {
        perChainRoot = _outputRoot;
        chainId = _chainId;
        gameType = GameType.wrap(_superGame ? 4 : 0);
        // Single-chain Super Root v1: version, timestamp, chain ID, output root.
        rootClaim = _superGame
            ? Claim.wrap(keccak256(abi.encodePacked(bytes1(0x01), uint64(1), _chainId, Claim.unwrap(_outputRoot))))
            : _outputRoot;
    }

    function createdAt() external pure returns (Timestamp) {
        return Timestamp.wrap(uint64(1 days));
    }

    function rootClaimByChainId(uint256 _chainId) external view returns (Claim) {
        require(_chainId == chainId);
        return perChainRoot;
    }

    function gameData() external view returns (GameType, Claim, bytes memory) {
        return (gameType, rootClaim, abi.encode(uint256(1)));
    }

    function resolve() external {
        status = GameStatus.DEFENDER_WINS;
        resolvedAt = Timestamp.wrap(uint64(block.timestamp));
    }
}

/// @notice Authorization through production proxies, Portal, registry and factory lookup code.
///         Eligibility proofs seed records; the inclusion witness creates its record through the Portal.
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
        kevm.setGas(30_000_000);
        AuthorizationCase memory example = _eligibleCase();
        Types.WithdrawalTransaction memory withdrawal;
        withdrawal.sender = address(0x1234);
        withdrawal.target = address(0x5678);
        withdrawal.gasLimit = 100_000;
        example.submitter = address(0x9ABC);
        example.withdrawalHash = _withdrawalHash(withdrawal);
        assert(_check(example));
        assert(
            _finalize(
                withdrawal, example.submitter, _externalProof ? address(0xBEEF) : example.submitter, _externalProof
            )
        );
        assert(portal.finalizedWithdrawals(example.withdrawalHash));
    }

    /// @notice Deletion removes exactly an eligible record and preserves an unrelated record and finalized status.
    function prove_deleteProvenWithdrawal_equivalence(
        AuthorizationCase memory _case,
        bytes32 _otherHash,
        address _otherSubmitter,
        bytes32 _otherRecord,
        bool _otherFinalized,
        address _caller
    )
        external
    {
        vm.assume(_otherHash != _case.withdrawalHash || _otherSubmitter != _case.submitter);
        bytes32 recordSlot = _seed(_case);
        bytes32 beforeRecord = vm.load(address(portal), recordSlot);
        bytes32 otherSlot = keccak256(abi.encode(_otherSubmitter, keccak256(abi.encode(_otherHash, uint256(57)))));
        vm.store(address(portal), otherSlot, _otherRecord);
        bytes32 finalizedSlot = keccak256(abi.encode(_otherHash, uint256(51)));
        bytes32 beforeFinalized = bytes32(uint256(_otherFinalized ? 1 : 0));
        vm.store(address(portal), finalizedSlot, beforeFinalized);

        vm.prank(_caller);
        (bool deleted,) =
            address(portal).call(abi.encodeCall(portal.deleteProvenWithdrawal, (_case.withdrawalHash, _case.submitter)));
        bool eligible =
            _case.provenAt != 0 && (_case.status == uint8(GameStatus.CHALLENGER_WINS) || _case.blacklisted != 0);
        assert(deleted == eligible);
        assert(vm.load(address(portal), recordSlot) == (deleted ? bytes32(0) : beforeRecord));
        assert(vm.load(address(portal), otherSlot) == _otherRecord);
        assert(vm.load(address(portal), finalizedSlot) == beforeFinalized);
    }

    /// @notice A blacklisted proven record can be deleted.
    function prove_deleteProvenWithdrawal_eligible_succeeds() external {
        AuthorizationCase memory example = _eligibleCase();
        example.blacklisted = 1;
        bytes32 recordSlot = _seed(example);
        portal.deleteProvenWithdrawal(example.withdrawalHash, example.submitter);
        assert(vm.load(address(portal), recordSlot) == bytes32(0));
    }

    /// @notice A concrete inclusion witness creates a record that either finalizer can consume.
    ///         This acceptance sequence is not a universal inclusion or history theorem.
    function prove_proveAndFinalize_eligible_succeeds(bool _externalProof, bool _superGame) external {
        kevm.setGas(30_000_000);
        Types.WithdrawalTransaction memory withdrawal;
        withdrawal.sender = address(0x1234);
        withdrawal.target = address(0x5678);
        withdrawal.gasLimit = 100_000;
        bytes32 withdrawalHash = _withdrawalHash(withdrawal);
        address submitter = address(0x9ABC);
        bytes32 recordSlot = keccak256(abi.encode(submitter, keccak256(abi.encode(withdrawalHash, uint256(57)))));
        WithdrawalProofGame_Harness candidate;
        {
            bytes32 secureKey = keccak256(abi.encode(keccak256(abi.encode(withdrawalHash, uint256(0)))));
            bytes[] memory witness = new bytes[](1);
            // Canonical RLP leaf: [hex-prefix(complete 32-byte key, leaf), storage value 0x01].
            witness[0] = abi.encodePacked(hex"e3a120", secureKey, hex"01");
            Types.OutputRootProof memory outputRoot;
            outputRoot.messagePasserStorageRoot = keccak256(witness[0]);
            bytes32 commitment = keccak256(
                abi.encode(
                    outputRoot.version,
                    outputRoot.stateRoot,
                    outputRoot.messagePasserStorageRoot,
                    outputRoot.latestBlockhash
                )
            );
            vm.warp(1 days);
            candidate =
                new WithdrawalProofGame_Harness(Claim.wrap(commitment), portal.systemConfig().l2ChainId(), _superGame);
            // A wrong root getter must not accidentally satisfy the acceptance witness.
            assert(!_superGame || Claim.unwrap(candidate.rootClaim()) != commitment);
            bytes32 uuid = keccak256(abi.encode(candidate.gameType(), candidate.rootClaim(), abi.encode(uint256(1))));
            bytes32 registration = bytes32(
                (uint256(GameType.unwrap(candidate.gameType())) << 224) | (uint256(1 days) << 160)
                    | uint256(uint160(address(candidate)))
            );
            vm.store(address(factory), keccak256(abi.encode(uuid, uint256(103))), registration);
            vm.store(address(factory), bytes32(uint256(104)), bytes32(uint256(1)));
            vm.store(address(factory), keccak256(abi.encode(uint256(104))), registration);
            vm.store(address(registry), bytes32(uint256(6)), bytes32(0));

            assert(vm.load(address(portal), recordSlot) == bytes32(0));
            vm.warp(2 days);
            vm.prank(submitter);
            portal.proveWithdrawalTransaction(withdrawal, 0, outputRoot, witness);
        }
        assert(
            vm.load(address(portal), recordSlot)
                == bytes32(uint256(uint160(address(candidate))) | (uint256(2 days) << 160))
        );
        vm.warp(3 days);
        candidate.resolve();
        vm.warp(3 days + proofDelay + gameDelay + 1);
        assert(_finalize(withdrawal, submitter, _externalProof ? address(0xBEEF) : submitter, _externalProof));
        assert(portal.finalizedWithdrawals(withdrawalHash));
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

    function _seed(AuthorizationCase memory _case) internal returns (bytes32 recordSlot) {
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
        recordSlot = keccak256(abi.encode(_case.submitter, keccak256(abi.encode(_case.withdrawalHash, uint256(57)))));
        vm.store(
            address(portal), recordSlot, bytes32(uint256(uint160(address(game))) | (uint256(_case.provenAt) << 160))
        );
        bytes32 finalizedSlot = keccak256(abi.encode(_case.withdrawalHash, uint256(51)));
        vm.store(address(portal), finalizedSlot, bytes32(uint256(_case.finalized)));
        vm.warp(_case.now);
    }

    function _check(AuthorizationCase memory _case) internal returns (bool accepted_) {
        bytes32 recordSlot = _seed(_case);
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

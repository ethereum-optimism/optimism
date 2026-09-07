// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { OptimismPortal2 } from "src/L1/OptimismPortal2.sol";
import { AnchorStateRegistry } from "src/dispute/AnchorStateRegistry.sol";
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { Proxy } from "src/universal/Proxy.sol";
import { KontrolUtils } from "./utils/KontrolUtils.sol";

// Libraries
import { Claim, GameStatus, GameType, Timestamp } from "src/dispute/lib/Types.sol";

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
contract WithdrawalAuthorizationKontrol is KontrolUtils {
    struct AuthorizationCase {
        bytes32 withdrawalHash;
        address submitter;
        uint64 createdAt;
        uint64 resolvedAt;
        uint64 provenAt;
        uint64 retiredAt;
        uint64 now;
        uint8 status;
        bool finalized;
        bool registered;
        bool respected;
        bool blacklisted;
        bool paused;
    }

    OptimismPortal2 internal portal;
    AnchorStateRegistry internal registry;
    DisputeGameFactory internal factory;
    WithdrawalGame_Harness internal game;
    bytes32 internal registrationSlot;

    function setUp() public {
        portal = OptimismPortal2(payable(_proxy(address(new OptimismPortal2(7 days)))));
        registry = AnchorStateRegistry(_proxy(address(new AnchorStateRegistry(3.5 days))));
        factory = DisputeGameFactory(_proxy(address(new DisputeGameFactory())));
        game = new WithdrawalGame_Harness();

        // Slots/packing follow snapshots/storageLayout; assertions detect incorrect setup.
        vm.store(address(portal), bytes32(uint256(62)), bytes32(uint256(uint160(address(registry)))));
        vm.store(address(registry), bytes32(0), bytes32(uint256(uint160(address(game))) << 16));
        vm.store(address(registry), bytes32(uint256(1)), bytes32(uint256(uint160(address(factory)))));
        bytes32 uuid = keccak256(abi.encode(GameType.wrap(0), Claim.wrap(bytes32(0)), abi.encode(uint256(1))));
        registrationSlot = keccak256(abi.encode(uuid, uint256(103)));
        assert(address(portal.anchorStateRegistry()) == address(registry));
        assert(address(registry.systemConfig()) == address(game));
        assert(address(registry.disputeGameFactory()) == address(factory));
    }

    /// @notice Acceptance is equivalent to the combined Portal and registry eligibility predicate.
    function prove_checkWithdrawal_equivalence(AuthorizationCase memory _case) external {
        vm.assume(_case.status <= uint8(GameStatus.DEFENDER_WINS));
        _check(_case);
    }

    /// @notice A concrete eligible witness prevents an always-reverting fixture from passing.
    function prove_checkWithdrawal_eligible_succeeds() external {
        AuthorizationCase memory example;
        example.createdAt = 1 days;
        example.provenAt = 2 days;
        example.resolvedAt = 3 days;
        example.now = 30 days;
        example.status = uint8(GameStatus.DEFENDER_WINS);
        example.registered = true;
        example.respected = true;
        assert(_check(example));
    }

    function _check(AuthorizationCase memory _case) internal returns (bool accepted_) {
        game.configure(_case.createdAt, _case.resolvedAt, GameStatus(_case.status), _case.respected, _case.paused);
        vm.store(address(factory), registrationSlot, bytes32(uint256(_case.registered ? uint160(address(game)) : 0)));
        vm.store(address(registry), bytes32(uint256(6)), bytes32(uint256(_case.retiredAt) << 32));
        vm.store(
            address(registry),
            keccak256(abi.encode(address(game), uint256(5))),
            bytes32(uint256(_case.blacklisted ? 1 : 0))
        );
        bytes32 recordSlot =
            keccak256(abi.encode(_case.submitter, keccak256(abi.encode(_case.withdrawalHash, uint256(57)))));
        vm.store(
            address(portal), recordSlot, bytes32(uint256(uint160(address(game))) | (uint256(_case.provenAt) << 160))
        );
        bytes32 finalizedSlot = keccak256(abi.encode(_case.withdrawalHash, uint256(51)));
        vm.store(address(portal), finalizedSlot, bytes32(uint256(_case.finalized ? 1 : 0)));
        vm.warp(_case.now);

        bool eligible = !_case.finalized && _case.provenAt != 0 && _case.provenAt > _case.createdAt
            && _case.now >= _case.provenAt && uint256(_case.now) - _case.provenAt > 7 days && _case.registered
            && !_case.blacklisted && _case.createdAt > _case.retiredAt && !_case.paused && _case.respected
            && _case.status == uint8(GameStatus.DEFENDER_WINS) && _case.resolvedAt != 0 && _case.now >= _case.resolvedAt
            && uint256(_case.now) - _case.resolvedAt > 3.5 days;
        (accepted_,) =
            address(portal).staticcall(abi.encodeCall(portal.checkWithdrawal, (_case.withdrawalHash, _case.submitter)));
        assert(accepted_ == eligible);
        // STATICCALL also prevents writes in every dependency, including reverted executions.
        assert(
            vm.load(address(portal), recordSlot)
                == bytes32(uint256(uint160(address(game))) | (uint256(_case.provenAt) << 160))
        );
        assert(portal.finalizedWithdrawals(_case.withdrawalHash) == _case.finalized);
    }

    function _proxy(address _implementation) internal returns (address) {
        Proxy proxy = new Proxy(address(this));
        proxy.upgradeTo(_implementation);
        return address(proxy);
    }
}

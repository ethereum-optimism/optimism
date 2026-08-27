// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import { DeploymentSummaryFaultProofs } from "./utils/DeploymentSummaryFaultProofs.sol";
import { KontrolUtils } from "./utils/KontrolUtils.sol";
import { Types } from "src/libraries/Types.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Timestamp } from "src/dispute/lib/Types.sol";
import { IOptimismPortal2 as OptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";
import { ISuperchainConfig as SuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { Features } from "src/libraries/Features.sol";
import { Constants } from "src/libraries/Constants.sol";
import { GameStatus } from "src/dispute/lib/Types.sol";
import { OptimismPortal2 as OptimismPortal2Implementation } from "src/L1/OptimismPortal2.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import {
    AirgapFactory_Harness,
    AirgapGame_Harness,
    AirgapSystemConfig_Harness,
    AnchorStateRegistryAirgap_Harness
} from "./utils/AirgapHarnesses.sol";

contract OptimismPortalAirgap_Harness is OptimismPortal2Implementation {
    constructor(uint256 _proofMaturityDelay) OptimismPortal2Implementation(_proofMaturityDelay) { }

    function configure(IAnchorStateRegistry _registry) external {
        anchorStateRegistry = _registry;
    }

    function recordWithdrawal(
        bytes32 _withdrawalHash,
        address _proofSubmitter,
        IDisputeGame _game,
        uint64 _provenAt
    )
        external
    {
        provenWithdrawals[_withdrawalHash][_proofSubmitter] =
            ProvenWithdrawal({ disputeGameProxy: _game, timestamp: _provenAt });
    }
}

contract OptimismPortal2Kontrol is DeploymentSummaryFaultProofs, KontrolUtils {
    address internal constant WITHDRAWAL_SENDER = address(0x1111);
    address internal constant WITHDRAWAL_TARGET = address(0xBEEF);
    bytes32 internal constant WITHDRAWAL_HASH = 0x39a1f3252bf31160a6debb698e5f9b4afceabcfbb8657ef197fadffcc584d7d6;
    uint256 internal constant PROVEN_WITHDRAWALS_SLOT = 57;
    uint256 internal constant L2_SENDER_SLOT = 50;
    uint256 internal constant SYSTEM_CONFIG_SLOT = 55;
    uint256 internal constant ANCHOR_STATE_REGISTRY_SLOT = 62;
    uint256 internal constant ETH_LOCKBOX_SLOT = 63;
    uint256 internal constant SYSTEM_CONFIG_FEATURES_SLOT = 109;
    uint256 internal constant LOCKBOX_BALANCE = type(uint128).max;
    uint64 internal constant PROVEN_AT = 1;
    uint64 internal constant PROOF_MATURITY_BOUNDARY = PROVEN_AT + 7 days;
    uint64 internal constant FIRST_MATURE_TIMESTAMP = PROOF_MATURITY_BOUNDARY + 1;

    OptimismPortal optimismPortal;
    SuperchainConfig superchainConfig;
    IAnchorStateRegistry anchorStateRegistry;
    IDisputeGame disputeGame;
    IETHLockbox ethLockbox;

    /// @dev Inlined setUp function for faster Kontrol performance
    ///      Tracking issue: https://github.com/runtimeverification/kontrol/issues/282
    function setUpInlined() public {
        optimismPortal = OptimismPortal(payable(optimismPortalProxyAddress));
        superchainConfig = SuperchainConfig(superchainConfigProxyAddress);
        anchorStateRegistry = optimismPortal.anchorStateRegistry();
        disputeGame = IDisputeGame(address(0xD15));
        ethLockbox = IETHLockbox(payable(eTHLockboxProxyAddress));
    }

    /// @notice Proves that a withdrawal recorded at time T cannot be finalized at any symbolic
    ///         time less than or equal to T plus the Portal's proof-maturity delay. The registry is
    ///         mocked to authorize finalization, so no external contract is trusted by this proof.
    /// @param _finalizationTimestamp Symbolic time, constrained to the inclusive maturity boundary.
    function prove_provenWithdrawal_cannotFinalizeBeforeProofMaturity(uint64 _finalizationTimestamp) external {
        setUpInlined();

        Types.WithdrawalTransaction memory withdrawal = _withdrawal();
        vm.warp(PROVEN_AT);
        _recordProvenWithdrawal(WITHDRAWAL_HASH, PROVEN_AT);

        // This is the exact Portal storage postcondition of a successful proving call. The
        // existing end-to-end Foundry delay-edge test exercises the valid Merkle-trie proof path;
        // this Kontrol lemma quantifies over every time in the resulting maturity interval.
        assert(Hashing.hashWithdrawal(withdrawal) == WITHDRAWAL_HASH);
        (IDisputeGame provenGame, uint64 storedAt) = optimismPortal.provenWithdrawals(WITHDRAWAL_HASH, address(this));
        assert(provenGame == disputeGame);
        assert(storedAt == PROVEN_AT);

        // Model a maximally adversarial registry that always authorizes the game. The Portal must
        // reject before consulting this result for every time in the proof-maturity interval.
        vm.mockCall(
            address(anchorStateRegistry),
            abi.encodeCall(anchorStateRegistry.isGameClaimValid, (disputeGame)),
            abi.encode(true)
        );
        uint256 proofMaturityDelay = optimismPortal.proofMaturityDelaySeconds();
        assert(proofMaturityDelay == 7 days);
        vm.assume(_finalizationTimestamp >= PROVEN_AT);
        vm.assume(_finalizationTimestamp <= PROOF_MATURITY_BOUNDARY);
        vm.warp(_finalizationTimestamp);

        vm.expectRevert(OptimismPortal.OptimismPortal_ProofNotOldEnough.selector);
        optimismPortal.finalizeWithdrawalTransaction(withdrawal);
        assert(!optimismPortal.finalizedWithdrawals(WITHDRAWAL_HASH));
    }

    /// @notice Proves the Portal's trust boundary by modeling a compromised AnchorStateRegistry.
    ///         If the registry authorizes the game, the Portal has no independent air-gap check and
    ///         considers the withdrawal authorized as soon as its own proof-maturity delay elapses.
    function prove_checkWithdrawal_trustsAnchorStateRegistry() external {
        setUpInlined();

        vm.warp(PROVEN_AT);
        _recordProvenWithdrawal(WITHDRAWAL_HASH, PROVEN_AT);

        // A compromised registry can authorize the claim without consulting resolvedAt. The
        // Portal's own proof-maturity check still cannot be bypassed.
        vm.mockCall(
            address(anchorStateRegistry),
            abi.encodeCall(anchorStateRegistry.isGameClaimValid, (disputeGame)),
            abi.encode(true)
        );
        vm.expectRevert(OptimismPortal.OptimismPortal_ProofNotOldEnough.selector);
        optimismPortal.checkWithdrawal(WITHDRAWAL_HASH, address(this));

        assert(optimismPortal.proofMaturityDelaySeconds() == 7 days);
        vm.warp(FIRST_MATURE_TIMESTAMP);
        optimismPortal.checkWithdrawal(WITHDRAWAL_HASH, address(this));
    }

    /// @notice Composes the production Portal and AnchorStateRegistry implementations. Whenever
    ///         the Portal's authorization predicate succeeds, both independently recorded L1
    ///         clocks must be strictly beyond their seven-day boundaries.
    function prove_checkWithdrawal_successRequiresBothAirgaps(
        uint64 _finalizationTime,
        uint64 _provenAt,
        uint64 _resolvedAt
    )
        external
    {
        vm.assume(_provenAt > 1);
        vm.assume(_resolvedAt > 0);
        vm.assume(_finalizationTime >= _provenAt);
        vm.assume(_finalizationTime >= _resolvedAt);

        AirgapGame_Harness game = new AirgapGame_Harness();
        game.setState(1, _resolvedAt, GameStatus.DEFENDER_WINS);
        AirgapFactory_Harness factory = new AirgapFactory_Harness(IDisputeGame(address(game)));
        AirgapSystemConfig_Harness systemConfig = new AirgapSystemConfig_Harness();
        AnchorStateRegistryAirgap_Harness registry = new AnchorStateRegistryAirgap_Harness(7 days);
        registry.configure(ISystemConfig(address(systemConfig)), IDisputeGameFactory(address(factory)));
        OptimismPortalAirgap_Harness portal = new OptimismPortalAirgap_Harness(7 days);
        portal.configure(IAnchorStateRegistry(address(registry)));
        portal.recordWithdrawal(WITHDRAWAL_HASH, address(this), IDisputeGame(address(game)), _provenAt);
        vm.warp(_finalizationTime);

        (bool authorized,) =
            address(portal).staticcall(abi.encodeCall(portal.checkWithdrawal, (WITHDRAWAL_HASH, address(this))));
        if (authorized) {
            assert(uint256(_finalizationTime) - uint256(_provenAt) > 7 days);
            assert(uint256(_finalizationTime) - uint256(_resolvedAt) > 7 days);
            assert(game.status() == GameStatus.DEFENDER_WINS);
        }
    }

    /// @notice The following proofs partition all current state-changing Portal entry points.
    ///         Each uses canonical ABI calldata from this attacker contract, proves that the
    ///         authorization invariant is preserved, and then attempts both finalization paths
    ///         for a challenger-backed withdrawal in the same timestamp. The external-ABI manifest
    ///         fails if a new selector is introduced or an existing view becomes state-mutating.
    function prove_depositPreCall_cannotFinalizeChallengerWithdrawal() external {
        _provePreCallCannotFinalizeChallengerWithdrawal(
            abi.encodeCall(
                OptimismPortal2Implementation.depositTransaction, (address(0xD3E0), 0, 21_000, false, bytes(""))
            )
        );
    }

    function prove_donatePreCall_cannotFinalizeChallengerWithdrawal() external {
        _provePreCallCannotFinalizeChallengerWithdrawal(abi.encodeCall(OptimismPortal2Implementation.donateETH, ()));
    }

    function prove_receivePreCall_cannotFinalizeChallengerWithdrawal() external {
        _provePreCallCannotFinalizeChallengerWithdrawal(hex"");
    }

    function prove_initializePreCall_cannotFinalizeChallengerWithdrawal() external {
        _provePreCallCannotFinalizeChallengerWithdrawal(
            abi.encodeCall(
                OptimismPortal2Implementation.initialize,
                (ISystemConfig(address(0)), IAnchorStateRegistry(address(0)), IETHLockbox(address(0)))
            )
        );
    }

    function prove_migrateLiquidityPreCall_cannotFinalizeChallengerWithdrawal() external {
        _provePreCallCannotFinalizeChallengerWithdrawal(
            abi.encodeCall(OptimismPortal2Implementation.migrateLiquidity, ())
        );
    }

    function prove_migrateGamePreCall_cannotFinalizeChallengerWithdrawal() external {
        _provePreCallCannotFinalizeChallengerWithdrawal(
            abi.encodeCall(
                OptimismPortal2Implementation.migrateToSharedDisputeGame,
                (IETHLockbox(address(0)), IAnchorStateRegistry(address(0)))
            )
        );
    }

    function prove_proveWithdrawalPreCall_cannotFinalizeChallengerWithdrawal() external {
        bytes[] memory withdrawalProof = new bytes[](0);
        _provePreCallCannotFinalizeChallengerWithdrawal(
            abi.encodeCall(
                OptimismPortal2Implementation.proveWithdrawalTransaction,
                (
                    _withdrawal(),
                    0,
                    Types.OutputRootProof(bytes32(0), bytes32(0), bytes32(0), bytes32(0)),
                    withdrawalProof
                )
            )
        );
    }

    function _provePreCallCannotFinalizeChallengerWithdrawal(bytes memory _preCallData) internal {
        AirgapGame_Harness game = new AirgapGame_Harness();
        game.setState(1, 2, GameStatus.CHALLENGER_WINS);
        AirgapFactory_Harness factory = new AirgapFactory_Harness(IDisputeGame(address(game)));
        AirgapSystemConfig_Harness systemConfig = new AirgapSystemConfig_Harness();
        AnchorStateRegistryAirgap_Harness registry = new AnchorStateRegistryAirgap_Harness(7 days);
        registry.configure(ISystemConfig(address(systemConfig)), IDisputeGameFactory(address(factory)));

        OptimismPortal2Implementation portal = new OptimismPortal2Implementation(7 days);
        vm.store(address(portal), bytes32(L2_SENDER_SLOT), bytes32(uint256(uint160(Constants.DEFAULT_L2_SENDER))));
        vm.store(address(portal), bytes32(SYSTEM_CONFIG_SLOT), bytes32(uint256(uint160(address(systemConfig)))));
        vm.store(address(portal), bytes32(ANCHOR_STATE_REGISTRY_SLOT), bytes32(uint256(uint160(address(registry)))));
        _recordProvenWithdrawalOn(address(portal), WITHDRAWAL_HASH, address(this), IDisputeGame(address(game)), 2);
        vm.warp(2 + 7 days + 1);

        // The preparatory call may succeed or revert. In either case, it must preserve every
        // authorization-critical field and cannot create a path around the registry decision.
        (bool preCallSucceeded,) = address(portal).call(_preCallData);
        if (preCallSucceeded) {
            _assertChallengerAuthorizationInvariant(portal, registry, game);
        } else {
            _assertChallengerAuthorizationInvariant(portal, registry, game);
        }

        Types.WithdrawalTransaction memory withdrawal = _withdrawal();
        assert(Hashing.hashWithdrawal(withdrawal) == WITHDRAWAL_HASH);
        (bool finalized,) = address(portal).call(abi.encodeCall(portal.finalizeWithdrawalTransaction, (withdrawal)));

        assert(!finalized);
        assert(!portal.finalizedWithdrawals(WITHDRAWAL_HASH));

        (bool finalizedWithExternalProof,) = address(portal).call(
            abi.encodeCall(portal.finalizeWithdrawalTransactionExternalProof, (withdrawal, address(this)))
        );
        assert(!finalizedWithExternalProof);
        assert(!portal.finalizedWithdrawals(WITHDRAWAL_HASH));
    }

    function _assertChallengerAuthorizationInvariant(
        OptimismPortal2Implementation _portal,
        AnchorStateRegistryAirgap_Harness _registry,
        AirgapGame_Harness _game
    )
        internal
        view
    {
        (IDisputeGame storedGame, uint64 storedAt) = _portal.provenWithdrawals(WITHDRAWAL_HASH, address(this));
        assert(storedGame == IDisputeGame(address(_game)));
        assert(storedAt == 2);
        assert(!_portal.finalizedWithdrawals(WITHDRAWAL_HASH));
        assert(_portal.anchorStateRegistry() == IAnchorStateRegistry(address(_registry)));
        assert(_portal.l2Sender() == Constants.DEFAULT_L2_SENDER);
        assert(_game.status() == GameStatus.CHALLENGER_WINS);
        assert(_game.resolvedAt().raw() == 2);
    }

    /// @notice Proves for every symbolic value that the real ETHLockbox transfers exactly that
    ///         amount to its authorized Portal and cannot debit more than requested.
    function prove_ethLockbox_unlocksExactValue(uint96 _value) external {
        setUpInlined();
        vm.assume(_value > 0);
        _enableLockbox();

        vm.deal(address(optimismPortal), 0);
        vm.deal(address(ethLockbox), LOCKBOX_BALANCE);

        vm.prank(address(optimismPortal));
        ethLockbox.unlockETH(_value);

        assert(address(ethLockbox).balance == LOCKBOX_BALANCE - _value);
        assert(address(optimismPortal).balance == _value);
    }

    /// @notice Proves for every symbolic value that unlocking and then locking the same amount
    ///         restores both the real ETHLockbox and Portal balances exactly.
    function prove_ethLockbox_roundTripPreservesValue(uint96 _value) external {
        setUpInlined();
        vm.assume(_value > 0);
        _enableLockbox();

        vm.deal(address(optimismPortal), 0);
        vm.deal(address(ethLockbox), LOCKBOX_BALANCE);

        vm.prank(address(optimismPortal));
        ethLockbox.unlockETH(_value);
        vm.prank(address(optimismPortal));
        ethLockbox.lockETH{ value: _value }();

        assert(address(ethLockbox).balance == LOCKBOX_BALANCE);
        assert(address(optimismPortal).balance == 0);
    }

    function _recordProvenWithdrawal(bytes32 _withdrawalHash, uint64 _provenAt) internal {
        bytes32 outerSlot = keccak256(abi.encode(_withdrawalHash, PROVEN_WITHDRAWALS_SLOT));
        bytes32 provenWithdrawalSlot = keccak256(abi.encode(address(this), outerSlot));
        uint256 packedProvenWithdrawal = uint256(uint160(address(disputeGame))) | uint256(_provenAt) << 160;
        vm.store(address(optimismPortal), provenWithdrawalSlot, bytes32(packedProvenWithdrawal));

        vm.mockCall(
            address(disputeGame), abi.encodeCall(disputeGame.createdAt, ()), abi.encode(Timestamp.wrap(uint64(0)))
        );
        vm.mockCall(
            address(anchorStateRegistry),
            abi.encodeCall(anchorStateRegistry.isGameClaimValid, (disputeGame)),
            abi.encode(true)
        );
    }

    function _recordProvenWithdrawalOn(
        address _portal,
        bytes32 _withdrawalHash,
        address _proofSubmitter,
        IDisputeGame _game,
        uint64 _provenAt
    )
        internal
    {
        bytes32 outerSlot = keccak256(abi.encode(_withdrawalHash, PROVEN_WITHDRAWALS_SLOT));
        bytes32 provenWithdrawalSlot = keccak256(abi.encode(_proofSubmitter, outerSlot));
        uint256 packedProvenWithdrawal = uint256(uint160(address(_game))) | uint256(_provenAt) << 160;
        vm.store(_portal, provenWithdrawalSlot, bytes32(packedProvenWithdrawal));
    }

    function _enableLockbox() internal {
        // Preserve any packed values above the lockbox address in slot 63.
        uint256 portalLockboxSlot = uint256(vm.load(address(optimismPortal), bytes32(ETH_LOCKBOX_SLOT)));
        portalLockboxSlot = portalLockboxSlot & ~uint256(type(uint160).max) | uint160(address(ethLockbox));
        vm.store(address(optimismPortal), bytes32(ETH_LOCKBOX_SLOT), bytes32(portalLockboxSlot));

        bytes32 featureSlot = keccak256(abi.encode(Features.ETH_LOCKBOX, SYSTEM_CONFIG_FEATURES_SLOT));
        vm.store(address(optimismPortal.systemConfig()), featureSlot, bytes32(uint256(1)));

        assert(optimismPortal.ethLockbox() == ethLockbox);
        assert(ethLockbox.authorizedPortals(optimismPortal));
    }

    function _withdrawal() internal pure returns (Types.WithdrawalTransaction memory) {
        return _withdrawalWithValue(WITHDRAWAL_TARGET, 0);
    }

    function _withdrawalWithValue(
        address _target,
        uint256 _value
    )
        internal
        pure
        returns (Types.WithdrawalTransaction memory)
    {
        return Types.WithdrawalTransaction({
            nonce: 0,
            sender: WITHDRAWAL_SENDER,
            target: _target,
            value: _value,
            gasLimit: 100_000,
            data: hex""
        });
    }

    function prove_finalizeWithdrawalTransaction_paused(Types.WithdrawalTransaction calldata _tx) external {
        setUpInlined();

        // Pause Optimism Portal
        vm.prank(optimismPortal.guardian());
        superchainConfig.pause(address(0));

        vm.expectRevert(OptimismPortal.OptimismPortal_CallPaused.selector);
        optimismPortal.finalizeWithdrawalTransaction(_tx);
    }

    /// @dev Function containing the logic for prove_proveWithdrawalTransaction_paused
    ///      The reason for this is that we want the _withdrawalProof to range in size from
    ///      0 to 10. These 11 proofs will exercise the same logic, which is contained in this function
    function prove_proveWithdrawalTransaction_paused_internal(
        Types.WithdrawalTransaction memory _tx,
        uint256 _l2OutputIndex,
        Types.OutputRootProof calldata _outputRootProof,
        bytes[] memory _withdrawalProof
    )
        internal
    {
        setUpInlined();

        // Pause Optimism Portal
        vm.prank(optimismPortal.guardian());
        superchainConfig.pause(address(0));

        vm.expectRevert(OptimismPortal.OptimismPortal_CallPaused.selector);
        optimismPortal.proveWithdrawalTransaction(_tx, _l2OutputIndex, _outputRootProof, _withdrawalProof);
    }

    /// @custom:kontrol-array-length-equals _withdrawalProof: 10,
    /// @custom:kontrol-bytes-length-equals _withdrawalProof: 600,
    function prove_proveWithdrawalTransaction_paused10(
        Types.WithdrawalTransaction memory _tx,
        uint256 _l2OutputIndex,
        Types.OutputRootProof calldata _outputRootProof,
        bytes[] calldata _withdrawalProof
    )
        external
    {
        prove_proveWithdrawalTransaction_paused_internal(_tx, _l2OutputIndex, _outputRootProof, _withdrawalProof);
    }

    /// @custom:kontrol-array-length-equals _withdrawalProof: 9,
    /// @custom:kontrol-bytes-length-equals _withdrawalProof: 600,
    function prove_proveWithdrawalTransaction_paused9(
        Types.WithdrawalTransaction memory _tx,
        uint256 _l2OutputIndex,
        Types.OutputRootProof calldata _outputRootProof,
        bytes[] calldata _withdrawalProof
    )
        external
    {
        prove_proveWithdrawalTransaction_paused_internal(_tx, _l2OutputIndex, _outputRootProof, _withdrawalProof);
    }

    /// @custom:kontrol-array-length-equals _withdrawalProof: 8,
    /// @custom:kontrol-bytes-length-equals _withdrawalProof: 600,
    function prove_proveWithdrawalTransaction_paused8(
        Types.WithdrawalTransaction memory _tx,
        uint256 _l2OutputIndex,
        Types.OutputRootProof calldata _outputRootProof,
        bytes[] calldata _withdrawalProof
    )
        external
    {
        prove_proveWithdrawalTransaction_paused_internal(_tx, _l2OutputIndex, _outputRootProof, _withdrawalProof);
    }

    /// @custom:kontrol-array-length-equals _withdrawalProof: 7,
    /// @custom:kontrol-bytes-length-equals _withdrawalProof: 600,
    function prove_proveWithdrawalTransaction_paused7(
        Types.WithdrawalTransaction memory _tx,
        uint256 _l2OutputIndex,
        Types.OutputRootProof calldata _outputRootProof,
        bytes[] calldata _withdrawalProof
    )
        external
    {
        prove_proveWithdrawalTransaction_paused_internal(_tx, _l2OutputIndex, _outputRootProof, _withdrawalProof);
    }

    /// @custom:kontrol-array-length-equals _withdrawalProof: 6,
    /// @custom:kontrol-bytes-length-equals _withdrawalProof: 600,
    function prove_proveWithdrawalTransaction_paused6(
        Types.WithdrawalTransaction memory _tx,
        uint256 _l2OutputIndex,
        Types.OutputRootProof calldata _outputRootProof,
        bytes[] calldata _withdrawalProof
    )
        external
    {
        prove_proveWithdrawalTransaction_paused_internal(_tx, _l2OutputIndex, _outputRootProof, _withdrawalProof);
    }

    /// @custom:kontrol-array-length-equals _withdrawalProof: 5,
    /// @custom:kontrol-bytes-length-equals _withdrawalProof: 600,
    function prove_proveWithdrawalTransaction_paused5(
        Types.WithdrawalTransaction memory _tx,
        uint256 _l2OutputIndex,
        Types.OutputRootProof calldata _outputRootProof,
        bytes[] calldata _withdrawalProof
    )
        external
    {
        prove_proveWithdrawalTransaction_paused_internal(_tx, _l2OutputIndex, _outputRootProof, _withdrawalProof);
    }

    /// @custom:kontrol-array-length-equals _withdrawalProof: 4,
    /// @custom:kontrol-bytes-length-equals _withdrawalProof: 600,
    function prove_proveWithdrawalTransaction_paused4(
        Types.WithdrawalTransaction memory _tx,
        uint256 _l2OutputIndex,
        Types.OutputRootProof calldata _outputRootProof,
        bytes[] calldata _withdrawalProof
    )
        external
    {
        prove_proveWithdrawalTransaction_paused_internal(_tx, _l2OutputIndex, _outputRootProof, _withdrawalProof);
    }

    /// @custom:kontrol-array-length-equals _withdrawalProof: 3,
    /// @custom:kontrol-bytes-length-equals _withdrawalProof: 600,
    function prove_proveWithdrawalTransaction_paused3(
        Types.WithdrawalTransaction memory _tx,
        uint256 _l2OutputIndex,
        Types.OutputRootProof calldata _outputRootProof,
        bytes[] calldata _withdrawalProof
    )
        external
    {
        prove_proveWithdrawalTransaction_paused_internal(_tx, _l2OutputIndex, _outputRootProof, _withdrawalProof);
    }

    /// @custom:kontrol-array-length-equals _withdrawalProof: 2,
    /// @custom:kontrol-bytes-length-equals _withdrawalProof: 600,
    function prove_proveWithdrawalTransaction_paused2(
        Types.WithdrawalTransaction memory _tx,
        uint256 _l2OutputIndex,
        Types.OutputRootProof calldata _outputRootProof,
        bytes[] calldata _withdrawalProof
    )
        external
    {
        prove_proveWithdrawalTransaction_paused_internal(_tx, _l2OutputIndex, _outputRootProof, _withdrawalProof);
    }

    /// @custom:kontrol-array-length-equals _withdrawalProof: 1,
    /// @custom:kontrol-bytes-length-equals _withdrawalProof: 600,
    function prove_proveWithdrawalTransaction_paused1(
        Types.WithdrawalTransaction memory _tx,
        uint256 _l2OutputIndex,
        Types.OutputRootProof calldata _outputRootProof,
        bytes[] calldata _withdrawalProof
    )
        external
    {
        prove_proveWithdrawalTransaction_paused_internal(_tx, _l2OutputIndex, _outputRootProof, _withdrawalProof);
    }

    function prove_proveWithdrawalTransaction_paused0(
        Types.WithdrawalTransaction memory _tx,
        uint256 _l2OutputIndex,
        Types.OutputRootProof calldata _outputRootProof
    )
        external
    {
        bytes[] memory _withdrawalProof = new bytes[](0);
        prove_proveWithdrawalTransaction_paused_internal(_tx, _l2OutputIndex, _outputRootProof, _withdrawalProof);
    }
}

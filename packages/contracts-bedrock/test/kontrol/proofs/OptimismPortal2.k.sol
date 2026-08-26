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

contract OptimismPortal2Kontrol is DeploymentSummaryFaultProofs, KontrolUtils {
    address internal constant WITHDRAWAL_SENDER = address(0x1111);
    address internal constant WITHDRAWAL_TARGET = address(0xBEEF);
    bytes32 internal constant WITHDRAWAL_HASH = 0x39a1f3252bf31160a6debb698e5f9b4afceabcfbb8657ef197fadffcc584d7d6;
    uint256 internal constant PROVEN_WITHDRAWALS_SLOT = 57;
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

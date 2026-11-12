// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { ComplianceIntegration_TestInit } from "test/setup/ComplianceIntegrationTestInit.sol";

// Libraries
import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Types } from "src/libraries/Types.sol";

// Interfaces
import { IComplianceModule } from "interfaces/compliance/IComplianceModule.sol";
import { IVisionPolicyV1 } from "interfaces/compliance/IVisionPolicyV1.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";

/// @title ComplianceWithdrawals_CallRecorder
/// @notice A withdrawal target that records the call it receives, so that two runs of the same
///         withdrawal can be compared.
contract ComplianceWithdrawals_CallRecorder {
    IOptimismPortal2 internal immutable PORTAL;

    /// @notice Hash of the caller, the Portal's l2Sender, the value and the calldata of the last call.
    bytes32 public lastCall;
    uint256 public calls;

    constructor(IOptimismPortal2 _portal) {
        PORTAL = _portal;
    }

    receive() external payable {
        _record();
    }

    fallback() external payable {
        _record();
    }

    function _record() internal {
        lastCall = keccak256(abi.encode(msg.sender, PORTAL.l2Sender(), msg.value, msg.data));
        calls++;
    }
}

/// @title ComplianceWithdrawals_Integration_Test
/// @notice Tests withdrawals proven and finalized through the Portal while the ComplianceModule
///         is its hook and VisionPolicyV1 is its policy.
///
///         VisionPolicyV1 pays a withdrawal that has a clear verdict. It rejects one without a
///         verdict until HOLD_DELAY after the withdrawal became finalizable, and holds it after.
contract ComplianceWithdrawals_Integration_Test is ComplianceIntegration_TestInit {
    /// @notice Runtime code that always reverts.
    bytes internal constant REVERTING_CODE = hex"60006000fd";

    ComplianceWithdrawals_CallRecorder internal recorder;

    function setUp() public override {
        super.setUp();
        recorder = new ComplianceWithdrawals_CallRecorder(optimismPortal2);
    }

    /// @notice Tests that a held ETH withdrawal completes with the same call to its target as a
    ///         withdrawal that is paid at finalization, and that its ETH waits in the module.
    function test_ethWithdrawal_heldThenCompleted_succeeds() external {
        Types.WithdrawalTransaction memory withdrawal = _withdrawal(0, alice, address(recorder), AMOUNT);
        (bytes32 id,) = _itemOf(withdrawal);
        bytes32 withdrawalHash = _proveAndWarpToHoldable(withdrawal);

        uint256 snapshot = vm.snapshotState();
        _clear(id);
        optimismPortal2.finalizeWithdrawalTransaction(withdrawal);
        bytes32 direct = recorder.lastCall();
        vm.revertToState(snapshot);

        (bytes32 heldId, BridgeHookTypes.Item memory item) = _holdWithdrawal(withdrawal);

        assertEq(heldId, id);
        _assertHeldWithdrawal(id);
        assertTrue(optimismPortal2.finalizedWithdrawals(withdrawalHash));
        assertEq(recorder.calls(), 0);
        assertEq(address(module).balance, AMOUNT);

        _clear(id);
        vm.expectEmit(address(optimismPortal2));
        emit WithdrawalFinalized(withdrawalHash, true);
        vm.prank(anyone);
        module.completeWithdrawal(item);

        assertEq(recorder.lastCall(), direct);
        assertEq(recorder.calls(), 1);
        assertEq(address(recorder).balance, AMOUNT);
        assertEq(address(module).balance, 0);
        _assertReleased(id);
    }

    /// @notice Tests that a held ETH withdrawal through the bridge completes through the Portal,
    ///         the messenger and the bridge, and pays its recipient the same amount as a
    ///         withdrawal that is paid at finalization.
    function test_bridgeETHWithdrawal_heldThenCompleted_succeeds() external {
        Types.WithdrawalTransaction memory withdrawal = _bridgeETHWithdrawal(0, alice, bob, AMOUNT);
        (bytes32 id,) = _itemOf(withdrawal);
        _proveAndWarpToHoldable(withdrawal);
        uint256 bobBalance = bob.balance;

        uint256 snapshot = vm.snapshotState();
        _clear(id);
        optimismPortal2.finalizeWithdrawalTransaction(withdrawal);
        uint256 direct = bob.balance;
        vm.revertToState(snapshot);

        (bytes32 heldId, BridgeHookTypes.Item memory item) = _holdWithdrawal(withdrawal);

        assertEq(heldId, id);
        _assertHeldWithdrawal(id);
        assertEq(module.parties(item).sender, alice);
        assertEq(module.parties(item).recipient, bob);
        assertEq(address(module).balance, AMOUNT);
        assertEq(bob.balance, bobBalance);

        _clear(id);
        vm.prank(anyone);
        module.completeWithdrawal(item);

        assertEq(bob.balance, direct);
        assertEq(direct, bobBalance + AMOUNT);
        assertEq(address(module).balance, 0);
        assertTrue(l1CrossDomainMessenger.successfulMessages(keccak256(withdrawal.data)));
        _assertReleased(id);
    }

    /// @notice Tests that a held token withdrawal moves its tokens from the bridge's escrow to
    ///         the module and pays its recipient at completion, as a cleared one does at
    ///         finalization.
    function test_erc20Withdrawal_heldThenCompleted_succeeds() external {
        _escrow(AMOUNT);
        Types.WithdrawalTransaction memory withdrawal = _erc20Withdrawal(0, alice, bob, AMOUNT);
        (bytes32 id,) = _tokenItemOf(withdrawal);
        _proveAndWarpToHoldable(withdrawal);

        uint256 snapshot = vm.snapshotState();
        _clear(id);
        optimismPortal2.finalizeWithdrawalTransaction(withdrawal);
        assertEq(L1Token.balanceOf(bob), AMOUNT);
        _assertTokenCustody({ _escrowed: 0, _held: 0 });
        vm.revertToState(snapshot);

        (bytes32 heldId, BridgeHookTypes.Item memory item) = _holdWithdrawal(withdrawal);

        assertEq(heldId, id);
        _assertHeldWithdrawal(id);
        _assertTokenCustody({ _escrowed: 0, _held: AMOUNT });
        assertEq(L1Token.balanceOf(bob), 0);

        _clear(id);
        vm.prank(anyone);
        module.completeWithdrawal(item);

        assertEq(L1Token.balanceOf(bob), AMOUNT);
        _assertTokenCustody({ _escrowed: 0, _held: 0 });
        assertTrue(l1CrossDomainMessenger.successfulMessages(keccak256(withdrawal.data)));
        _assertReleased(id);
    }

    /// @notice Tests that a held message with no value completes with the same call to its
    ///         target as a message that is relayed at finalization.
    function test_messageWithdrawal_heldThenCompleted_succeeds() external {
        Types.WithdrawalTransaction memory withdrawal = _messageWithdrawal(0, alice, address(recorder), 0, hex"1234");
        (bytes32 id,) = _itemOf(withdrawal);
        _proveAndWarpToHoldable(withdrawal);

        uint256 snapshot = vm.snapshotState();
        _clear(id);
        optimismPortal2.finalizeWithdrawalTransaction(withdrawal);
        bytes32 direct = recorder.lastCall();
        vm.revertToState(snapshot);

        (bytes32 heldId, BridgeHookTypes.Item memory item) = _holdWithdrawal(withdrawal);

        assertEq(heldId, id);
        _assertHeldWithdrawal(id);
        assertEq(recorder.calls(), 0);

        _clear(id);
        vm.prank(anyone);
        module.completeWithdrawal(item);

        assertEq(recorder.lastCall(), direct);
        assertEq(recorder.calls(), 1);
        _assertReleased(id);
    }

    /// @notice Tests that the verdict writer clears and completes held ETH and token withdrawals
    ///         in one transaction.
    function test_clearAndCompleteWithdrawal_batch_succeeds() external {
        _escrow(AMOUNT);
        Types.WithdrawalTransaction memory ethWithdrawal = _withdrawal(0, alice, bob, AMOUNT);
        Types.WithdrawalTransaction memory erc20Withdrawal = _erc20Withdrawal(1, alice, bob, AMOUNT);
        bytes[] memory calls = new bytes[](2);
        bytes32[] memory ids = new bytes32[](2);
        BridgeHookTypes.Item memory item;

        _prove(ethWithdrawal);
        _proveAndWarpToHoldable(erc20Withdrawal);
        (ids[0], item) = _holdWithdrawal(ethWithdrawal);
        calls[0] = abi.encodeCall(policy.clearAndCompleteWithdrawal, (item, _expiry()));
        (ids[1], item) = _holdWithdrawal(erc20Withdrawal);
        calls[1] = abi.encodeCall(policy.clearAndCompleteWithdrawal, (item, _expiry()));
        uint256 bobBalance = bob.balance;

        vm.prank(verdictWriter);
        policy.multicall(calls);

        _assertReleased(ids[0]);
        _assertReleased(ids[1]);
        assertEq(bob.balance, bobBalance + AMOUNT);
        assertEq(L1Token.balanceOf(bob), AMOUNT);
        assertEq(address(module).balance, 0);
        _assertTokenCustody({ _escrowed: 0, _held: 0 });
    }

    /// @notice Tests the path of a clean user: finalizing without a verdict before the hold delay
    ///         has passed reverts and consumes nothing, and the same withdrawal is paid once it
    ///         has a verdict.
    function test_withdrawal_clearedAfterRejection_succeeds() external {
        Types.WithdrawalTransaction memory withdrawal = _withdrawal(0, alice, bob, AMOUNT);
        (bytes32 id,) = _itemOf(withdrawal);
        bytes32 withdrawalHash = _prove(withdrawal);
        uint256 holdableAt = _warpToFinalizable(withdrawalHash) + HOLD_DELAY;
        uint256 bobBalance = bob.balance;

        vm.warp(holdableAt - 1);
        vm.expectRevert(abi.encodeWithSelector(IVisionPolicyV1.VisionPolicyV1_NotYetHoldable.selector, holdableAt));
        optimismPortal2.finalizeWithdrawalTransaction(withdrawal);

        assertFalse(optimismPortal2.finalizedWithdrawals(withdrawalHash));

        _clear(id);
        optimismPortal2.finalizeWithdrawalTransaction(withdrawal);

        assertEq(bob.balance, bobBalance + AMOUNT);
        assertFalse(optimismPortal2.heldWithdrawals(id));
        assertEq(address(module).balance, 0);
    }

    /// @notice Tests that a withdrawal to a denied recipient is held at the first attempt, with no
    ///         hold delay, and completes only once the entry is removed.
    function test_completeWithdrawal_afterRecipientUndenied_succeeds() external {
        Types.WithdrawalTransaction memory withdrawal = _withdrawal(0, alice, bob, AMOUNT);
        _warpToFinalizable(_prove(withdrawal));
        _deny(bob);
        uint256 bobBalance = bob.balance;

        (bytes32 id, BridgeHookTypes.Item memory item) = _holdWithdrawal(withdrawal);
        _assertHeldWithdrawal(id);

        _clear(id);
        vm.expectRevert(IComplianceModule.ComplianceModule_PolicyRefused.selector);
        module.completeWithdrawal(item);

        _assertHeldWithdrawal(id);
        assertEq(address(module).balance, AMOUNT);

        _undeny(bob);
        module.completeWithdrawal(item);

        assertEq(bob.balance, bobBalance + AMOUNT);
        _assertReleased(id);
    }

    /// @notice Tests that a withdrawal whose verdict expired is held at the first attempt, with no
    ///         hold delay, and completes on a new verdict.
    function test_completeWithdrawal_afterVerdictRenewed_succeeds() external {
        Types.WithdrawalTransaction memory withdrawal = _withdrawal(0, alice, bob, AMOUNT);
        (bytes32 id,) = _itemOf(withdrawal);
        _warpToFinalizable(_prove(withdrawal));
        uint256 bobBalance = bob.balance;

        vm.prank(verdictWriter);
        policy.clear(id, uint64(block.timestamp + 1));
        vm.warp(block.timestamp + 1);
        (bytes32 heldId, BridgeHookTypes.Item memory item) = _holdWithdrawal(withdrawal);

        assertEq(heldId, id);
        _assertHeldWithdrawal(id);
        assertEq(bob.balance, bobBalance);

        _clear(id);
        module.completeWithdrawal(item);

        assertEq(bob.balance, bobBalance + AMOUNT);
        _assertReleased(id);
    }

    /// @notice Tests that a Fee Vault's withdrawal is paid with no verdict and no allowlist entry.
    function test_withdrawal_feeVault_succeeds() external {
        Types.WithdrawalTransaction memory withdrawal = _withdrawal(0, Predeploys.SEQUENCER_FEE_WALLET, bob, AMOUNT);
        _warpToFinalizable(_prove(withdrawal));
        uint256 bobBalance = bob.balance;

        optimismPortal2.finalizeWithdrawalTransaction(withdrawal);

        assertEq(bob.balance, bobBalance + AMOUNT);
        assertEq(address(module).balance, 0);
    }

    /// @notice Tests that a held withdrawal runs once. It cannot be finalized again while held or
    ///         completed again after completion, even with a new verdict.
    function test_completeWithdrawal_twice_reverts() external {
        Types.WithdrawalTransaction memory withdrawal = _withdrawal(0, alice, address(recorder), AMOUNT);
        _proveAndWarpToHoldable(withdrawal);
        (bytes32 id, BridgeHookTypes.Item memory item) = _holdWithdrawal(withdrawal);

        vm.expectRevert(IOptimismPortal2.OptimismPortal_AlreadyFinalized.selector);
        optimismPortal2.finalizeWithdrawalTransaction(withdrawal);

        _clear(id);
        module.completeWithdrawal(item);

        _clear(id);
        vm.expectRevert(IComplianceModule.ComplianceModule_ItemNotHeld.selector);
        module.completeWithdrawal(item);

        assertEq(recorder.calls(), 1);
        assertEq(address(recorder).balance, AMOUNT);
    }

    /// @notice Tests that a completion whose target call fails reverts and changes nothing. The
    ///         withdrawal keeps its hold, its verdict and its ETH, and completes once the target
    ///         accepts the call.
    function test_completeWithdrawal_afterTargetFixed_succeeds() external {
        address target = makeAddr("target");
        vm.etch(target, REVERTING_CODE);
        Types.WithdrawalTransaction memory withdrawal = _withdrawal(0, alice, target, AMOUNT);
        _proveAndWarpToHoldable(withdrawal);
        (bytes32 id, BridgeHookTypes.Item memory item) = _holdWithdrawal(withdrawal);
        _clear(id);
        uint64 expiry = _expiry();

        vm.expectRevert(IOptimismPortal2.OptimismPortal_TargetCallFailed.selector);
        module.completeWithdrawal(item);

        _assertHeldWithdrawal(id);
        assertEq(module.items(id).expiresAt, expiry);
        assertEq(address(module).balance, AMOUNT);

        vm.etch(target, hex"");
        module.completeWithdrawal(item);

        assertEq(target.balance, AMOUNT);
        _assertReleased(id);
    }

    /// @notice Tests that a held withdrawal whose relay fails in the messenger still completes.
    ///         Its ETH moves from the module to the messenger, and anyone can replay the message
    ///         there. The replay does not reach the hook, so the deny list no longer applies.
    function test_completeWithdrawal_messengerRelayFails_succeeds() external {
        address target = makeAddr("target");
        vm.etch(target, REVERTING_CODE);
        Types.WithdrawalTransaction memory withdrawal = _messageWithdrawal(0, alice, target, AMOUNT, hex"");
        bytes32 messageHash = keccak256(withdrawal.data);
        _proveAndWarpToHoldable(withdrawal);
        (bytes32 id, BridgeHookTypes.Item memory item) = _holdWithdrawal(withdrawal);

        _clear(id);
        module.completeWithdrawal(item);

        _assertReleased(id);
        assertTrue(l1CrossDomainMessenger.failedMessages(messageHash));
        assertEq(address(module).balance, 0);
        assertEq(address(l1CrossDomainMessenger).balance, AMOUNT);

        vm.etch(target, hex"");
        _deny(target);
        vm.prank(anyone);
        l1CrossDomainMessenger.relayMessage(withdrawal.nonce, alice, target, AMOUNT, MESSAGE_GAS_LIMIT, hex"");

        assertTrue(l1CrossDomainMessenger.successfulMessages(messageHash));
        assertEq(target.balance, AMOUNT);
    }

    /// @notice Tests that a pause stops the completion of held withdrawals but not of held
    ///         deposits, and that the withdrawal completes after the pause ends.
    function test_completeWithdrawal_afterUnpause_succeeds() external {
        Types.WithdrawalTransaction memory withdrawal = _withdrawal(0, alice, bob, AMOUNT);
        _proveAndWarpToHoldable(withdrawal);
        (bytes32 id, BridgeHookTypes.Item memory item) = _holdWithdrawal(withdrawal);
        _clear(id);
        _depositETH(alice, bob, AMOUNT);
        (bytes32 depositId, BridgeHookTypes.Item memory deposit) = _heldDeposit();
        _clear(depositId);
        uint256 bobBalance = bob.balance;

        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(address(0));

        vm.expectRevert(IComplianceModule.ComplianceModule_Paused.selector);
        module.completeWithdrawal(item);
        _completeDeposit(deposit);
        _assertHeldWithdrawal(id);

        vm.prank(superchainConfig.guardian());
        superchainConfig.unpause(address(0));
        module.completeWithdrawal(item);

        assertEq(bob.balance, bobBalance + AMOUNT);
        _assertReleased(id);
    }

    /// @notice Escrows L1Token in the bridge with a deposit that proceeds, so that a token
    ///         withdrawal has something to pay from.
    function _escrow(uint256 _amount) internal {
        _allowlist(depositor);
        _depositERC20(depositor, depositor, _amount);
    }
}

// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test, Vm } from "forge-std/Test.sol";

// Contracts
import { MockOptimismPortal2 } from "scripts/deploy/mocks/MockOptimismPortal2.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { Features } from "src/libraries/Features.sol";
import { Types } from "src/libraries/Types.sol";
import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";
import { BridgeHookHashing } from "src/libraries/BridgeHookHashing.sol";

// Interfaces
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IBridgeHook } from "interfaces/L1/IBridgeHook.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";

/// @title StubBridgeHook
/// @notice Minimal hook standing in for the ComplianceModule, which lands in a later change. It
///         records what it was handed so the tests can assert custody actually moved.
contract StubBridgeHook is IBridgeHook {
    bool public passDeposits = true;
    bool public rejectDeposits;
    bytes32 public lastHeldId;

    error StubBridgeHook_Rejected();

    function setPassDeposits(bool _pass) external {
        passDeposits = _pass;
    }

    function setRejectDeposits(bool _reject) external {
        rejectDeposits = _reject;
    }

    function screenDeposit(BridgeHookTypes.Item calldata) external view returns (bool pass_) {
        if (rejectDeposits) revert StubBridgeHook_Rejected();
        pass_ = passDeposits;
    }

    function holdDeposit(BridgeHookTypes.Item calldata _item) external payable {
        lastHeldId = BridgeHookHashing.hashItem(_item);
    }

    bool public passWithdrawals = true;
    uint256 public lastFinalizableAt;

    function setPassWithdrawals(bool _pass) external {
        passWithdrawals = _pass;
    }

    function screenWithdrawal(BridgeHookTypes.Item calldata, uint256 _finalizableAt) external returns (bool pass_) {
        lastFinalizableAt = _finalizableAt;
        pass_ = passWithdrawals;
    }

    function holdWithdrawal(BridgeHookTypes.Item calldata _item) external payable {
        lastHeldId = BridgeHookHashing.hashItem(_item);
    }

    /// @notice Completes a held withdrawal, which only the hook may do.
    function completeWithdrawal(MockOptimismPortal2 _portal, BridgeHookTypes.Item calldata _item) external {
        _portal.completeWithdrawalTransaction{ value: _item.ethAmount }(_item);
    }

    /// @notice Completes a held deposit, which only the hook may do.
    function complete(MockOptimismPortal2 _portal, BridgeHookTypes.Item calldata _item) external {
        _portal.completeDepositTransaction{ value: _item.ethAmount }(_item);
    }

    receive() external payable { }
}

/// @title MockOptimismPortal2_Harness
/// @notice Writes the state that initialize would otherwise set behind a proxy.
contract MockOptimismPortal2_Harness is MockOptimismPortal2 {
    constructor() MockOptimismPortal2(1, 1000) {
        // initialize seeds this, and the harness writes state directly instead.
        l2Sender = Constants.DEFAULT_L2_SENDER;
    }

    function setSystemConfig(ISystemConfig _systemConfig) external {
        systemConfig = _systemConfig;
    }

    function setBridgeHookUnchecked(IBridgeHook _hook) external {
        bridgeHook = _hook;
    }

    function setETHLockbox(IETHLockbox _lockbox) external {
        ethLockbox = _lockbox;
    }
}

/// @title MockOptimismPortal2_TestInit
/// @notice Shared setup for the mock Portal deposit tests.
contract MockOptimismPortal2_TestInit is Test {
    MockOptimismPortal2_Harness internal portal;
    StubBridgeHook internal hook;

    address internal systemConfig = makeAddr("systemConfig");
    address internal messenger = makeAddr("messenger");
    address internal lockbox = makeAddr("lockbox");
    address internal bridge = makeAddr("bridge");
    address internal alice = makeAddr("alice");
    address internal target = makeAddr("target");

    /// @notice Mirrors of the Portal's events, which 0.8.15 cannot reference by qualified name.
    event WithdrawalProven(bytes32 indexed withdrawalHash, address indexed from, address indexed to);
    event WithdrawalProvenExtension1(bytes32 indexed withdrawalHash, address indexed proofSubmitter);
    event WithdrawalFinalized(bytes32 indexed withdrawalHash, bool success);
    event DepositHeld(bytes32 indexed id);
    event WithdrawalHeld(bytes32 indexed id);

    bytes32 internal constant DEPOSITED_TOPIC = keccak256("TransactionDeposited(address,address,uint256,bytes)");
    bytes32 internal constant FINALIZED_TOPIC = keccak256("WithdrawalFinalized(bytes32,bool)");

    uint64 internal constant GAS_LIMIT = 100_000;

    function setUp() public virtual {
        portal = new MockOptimismPortal2_Harness();
        portal.setSystemConfig(ISystemConfig(systemConfig));
        portal.setETHLockbox(IETHLockbox(payable(lockbox)));
        hook = new StubBridgeHook();

        vm.mockCall(lockbox, abi.encodeCall(IETHLockbox.paused, ()), abi.encode(false));

        // The lockbox is off, so a deposit's only event is TransactionDeposited. The lockbox
        // ordering is covered by the end-to-end choreography tests.
        vm.mockCall(
            systemConfig, abi.encodeCall(ISystemConfig.isFeatureEnabled, (Features.ETH_LOCKBOX)), abi.encode(false)
        );
        vm.mockCall(
            systemConfig, abi.encodeCall(ISystemConfig.isFeatureEnabled, (Features.CUSTOM_GAS_TOKEN)), abi.encode(false)
        );
        vm.mockCall(systemConfig, abi.encodeCall(ISystemConfig.l1StandardBridge, ()), abi.encode(bridge));
        vm.mockCall(systemConfig, abi.encodeCall(ISystemConfig.l1CrossDomainMessenger, ()), abi.encode(messenger));

        vm.deal(alice, 100 ether);
        vm.deal(address(portal), 100 ether);
    }

    /// @notice Counts the TransactionDeposited events in the recorded logs.
    function _countDeposited(Vm.Log[] memory _logs) internal pure returns (uint256 count_) {
        count_ = _countTopic(_logs, DEPOSITED_TOPIC);
    }

    /// @notice Counts the logs carrying the given topic0.
    function _countTopic(Vm.Log[] memory _logs, bytes32 _topic) internal pure returns (uint256 count_) {
        for (uint256 i; i < _logs.length; i++) {
            if (_logs[i].topics.length > 0 && _logs[i].topics[0] == _topic) count_++;
        }
    }

    /// @notice Builds a withdrawal paying `_value` to `_target`.
    function _withdrawal(address _target, uint256 _value) internal pure returns (Types.WithdrawalTransaction memory) {
        return Types.WithdrawalTransaction({
            nonce: 7, sender: address(0xbeef), target: _target, value: _value, gasLimit: GAS_LIMIT, data: ""
        });
    }

    /// @notice The item the Portal screens for the given withdrawal.
    function _withdrawalItem(Types.WithdrawalTransaction memory _tx)
        internal
        pure
        returns (BridgeHookTypes.Item memory)
    {
        return BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Withdrawal,
            asset: BridgeHookTypes.Asset.ETH,
            from: _tx.sender,
            aliased: false,
            to: _tx.target,
            ethAmount: _tx.value,
            l2Value: 0,
            gasLimit: _tx.gasLimit,
            isCreation: false,
            data: _tx.data,
            nonce: _tx.nonce
        });
    }

    /// @notice Proves a withdrawal and advances past its maturity so it can be finalized.
    function _proveAndMature(Types.WithdrawalTransaction memory _tx) internal {
        portal.proveWithdrawalTransaction(_tx, 0, Types.OutputRootProof(0, 0, 0, 0), new bytes[](0));
        vm.warp(block.timestamp + portal.proofMaturityDelaySeconds() + 1);
    }

    /// @notice Builds the item the Portal would screen for an ETH deposit from alice.
    function _ethItem(uint256 _value, uint256 _nonce) internal view returns (BridgeHookTypes.Item memory) {
        return BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ETH,
            from: alice,
            aliased: false,
            to: target,
            ethAmount: _value,
            l2Value: 0,
            gasLimit: GAS_LIMIT,
            isCreation: false,
            data: "",
            nonce: _nonce
        });
    }
}

/// @title MockOptimismPortal2_depositTransaction_Test
/// @notice Tests the depositTransaction function of the MockOptimismPortal2 contract.
contract MockOptimismPortal2_depositTransaction_Test is MockOptimismPortal2_TestInit {
    /// @notice Tests that a deposit emits TransactionDeposited when no hook is configured, so a
    ///         chain that has not opted in behaves as it does today.
    function test_depositTransaction_noHook_succeeds() public {
        vm.recordLogs();
        vm.prank(alice, alice);
        portal.depositTransaction{ value: 1 ether }(target, 0, GAS_LIMIT, false, "");

        assertEq(_countDeposited(vm.getRecordedLogs()), 1);
        assertEq(portal.depositCount(), 0);
    }

    /// @notice Tests that a deposit the hook lets through takes the normal path unchanged.
    function test_depositTransaction_hookProceeds_succeeds() public {
        portal.setBridgeHookUnchecked(hook);

        vm.recordLogs();
        vm.prank(alice, alice);
        portal.depositTransaction{ value: 1 ether }(target, 0, GAS_LIMIT, false, "");

        assertEq(_countDeposited(vm.getRecordedLogs()), 1);
        assertEq(address(hook).balance, 0);
        assertEq(portal.depositCount(), 1);
    }

    /// @notice Tests that a held deposit emits no TransactionDeposited, moves its ETH to the hook
    ///         and commits its terms. This is the inversion the screener has to recognise.
    function test_depositTransaction_hookHolds_succeeds() public {
        portal.setBridgeHookUnchecked(hook);
        hook.setPassDeposits(false);

        vm.recordLogs();
        vm.prank(alice, alice);
        portal.depositTransaction{ value: 1 ether }(target, 0, GAS_LIMIT, false, "");

        assertEq(_countDeposited(vm.getRecordedLogs()), 0);
        assertEq(address(hook).balance, 1 ether);
        assertTrue(portal.heldDeposits(BridgeHookHashing.hashItem(_ethItem(1 ether, 0))));
    }

    /// @notice Tests that a held deposit is announced by the Portal itself, so the hold is
    ///         observable without depending on whatever hook happens to be installed.
    function test_depositTransaction_emitsDepositHeld_succeeds() public {
        portal.setBridgeHookUnchecked(hook);
        hook.setPassDeposits(false);

        vm.expectEmit(true, true, true, true, address(portal));
        emit DepositHeld(BridgeHookHashing.hashItem(_ethItem(1 ether, 0)));

        vm.prank(alice, alice);
        portal.depositTransaction{ value: 1 ether }(target, 0, GAS_LIMIT, false, "");
    }

    /// @notice Tests that a deposit which proceeds announces nothing, so the event means held
    ///         rather than screened.
    function test_depositTransaction_proceedingEmitsNoDepositHeld_succeeds() public {
        portal.setBridgeHookUnchecked(hook);

        vm.recordLogs();
        vm.prank(alice, alice);
        portal.depositTransaction{ value: 1 ether }(target, 0, GAS_LIMIT, false, "");

        assertEq(_countTopic(vm.getRecordedLogs(), keccak256("DepositHeld(bytes32)")), 0);
    }

    /// @notice Tests that a hook which reverts refuses the deposit outright, consuming nothing.
    function test_depositTransaction_hookRejects_reverts() public {
        portal.setBridgeHookUnchecked(hook);
        hook.setRejectDeposits(true);

        vm.prank(alice);
        vm.expectRevert(StubBridgeHook.StubBridgeHook_Rejected.selector);
        portal.depositTransaction{ value: 1 ether }(target, 0, GAS_LIMIT, false, "");
    }

    /// @notice Tests that two otherwise identical deposits in one block get distinct item IDs, so
    ///         a clearance for one cannot release the other.
    function test_depositTransaction_counterDistinguishesDeposits_succeeds() public {
        portal.setBridgeHookUnchecked(hook);
        hook.setPassDeposits(false);

        vm.startPrank(alice, alice);
        portal.depositTransaction{ value: 1 ether }(target, 0, GAS_LIMIT, false, "");
        portal.depositTransaction{ value: 1 ether }(target, 0, GAS_LIMIT, false, "");
        vm.stopPrank();

        bytes32 first = BridgeHookHashing.hashItem(_ethItem(1 ether, 0));
        bytes32 second = BridgeHookHashing.hashItem(_ethItem(1 ether, 1));
        assertTrue(first != second);
        assertTrue(portal.heldDeposits(first));
        assertTrue(portal.heldDeposits(second));
    }
}

/// @title MockOptimismPortal2_completeDepositTransaction_Test
/// @notice Tests the completeDepositTransaction function of the MockOptimismPortal2 contract.
contract MockOptimismPortal2_completeDepositTransaction_Test is MockOptimismPortal2_TestInit {
    BridgeHookTypes.Item internal held;

    function setUp() public override {
        super.setUp();
        portal.setBridgeHookUnchecked(hook);
        hook.setPassDeposits(false);

        vm.prank(alice, alice);
        portal.depositTransaction{ value: 1 ether }(target, 0, GAS_LIMIT, false, "");
        held = _ethItem(1 ether, 0);
    }

    /// @notice Tests that completing a held deposit emits TransactionDeposited on the committed
    ///         terms and clears the commitment.
    function test_completeDepositTransaction_succeeds() public {
        vm.recordLogs();
        hook.complete(portal, held);

        assertEq(_countDeposited(vm.getRecordedLogs()), 1);
        assertFalse(portal.heldDeposits(BridgeHookHashing.hashItem(held)));
    }

    /// @notice Tests that a held deposit completes at most once.
    function test_completeDepositTransaction_alreadyCompleted_reverts() public {
        hook.complete(portal, held);

        // Refund the hook so the second attempt is refused by the commitment check rather than
        // by running out of ETH.
        vm.deal(address(hook), 1 ether);

        vm.expectRevert(MockOptimismPortal2.OptimismPortal_NotHeld.selector);
        hook.complete(portal, held);
    }

    /// @notice Tests that only the hook may complete a held deposit.
    function test_completeDepositTransaction_callerNotHook_reverts() public {
        vm.deal(alice, 1 ether);
        vm.prank(alice);
        vm.expectRevert(MockOptimismPortal2.OptimismPortal_NotBridgeHook.selector);
        portal.completeDepositTransaction{ value: 1 ether }(held);
    }

    /// @notice Tests that an item which was never held cannot be completed, so the hook cannot
    ///         run terms the Portal never agreed to.
    function test_completeDepositTransaction_notHeld_reverts() public {
        BridgeHookTypes.Item memory forged = _ethItem(5 ether, 0);
        vm.deal(address(hook), 5 ether);

        vm.expectRevert(MockOptimismPortal2.OptimismPortal_NotHeld.selector);
        hook.complete(portal, forged);
    }

    /// @notice Tests that completion must carry exactly the ETH the item was held with.
    function test_completeDepositTransaction_valueMismatch_reverts() public {
        vm.deal(address(hook), 1 ether);
        vm.prank(address(hook));
        vm.expectRevert(MockOptimismPortal2.OptimismPortal_ValueMismatch.selector);
        portal.completeDepositTransaction{ value: 0.5 ether }(held);
    }
}

/// @title RevertingTarget
/// @notice A withdrawal target that always reverts, to exercise the failed-call branch.
contract RevertingTarget {
    error RevertingTarget_Nope();

    fallback() external payable {
        revert RevertingTarget_Nope();
    }
}

/// @title MockOptimismPortal2_proveWithdrawalTransaction_Test
/// @notice Tests the proveWithdrawalTransaction function of the MockOptimismPortal2 contract.
contract MockOptimismPortal2_proveWithdrawalTransaction_Test is MockOptimismPortal2_TestInit {
    /// @notice Tests that proving records the submitter and emits both proof events, without
    ///         verifying any merkle proof or consulting a dispute game.
    function test_proveWithdrawalTransaction_succeeds() public {
        Types.WithdrawalTransaction memory wtx = _withdrawal(target, 1 ether);
        bytes32 hash = keccak256(abi.encode(wtx.nonce, wtx.sender, wtx.target, wtx.value, wtx.gasLimit, wtx.data));

        vm.expectEmit(true, true, true, true);
        emit WithdrawalProven(hash, wtx.sender, wtx.target);
        vm.expectEmit(true, true, true, true);
        emit WithdrawalProvenExtension1(hash, address(this));

        portal.proveWithdrawalTransaction(wtx, 0, Types.OutputRootProof(0, 0, 0, 0), new bytes[](0));

        assertEq(portal.numProofSubmitters(hash), 1);
    }

    /// @notice Tests that proving refuses an unsafe target, as the real Portal does.
    function test_proveWithdrawalTransaction_unsafeTarget_reverts() public {
        vm.expectRevert(MockOptimismPortal2.OptimismPortal_BadTarget.selector);
        portal.proveWithdrawalTransaction(
            _withdrawal(address(portal), 0), 0, Types.OutputRootProof(0, 0, 0, 0), new bytes[](0)
        );
    }
}

/// @title MockOptimismPortal2_finalizeWithdrawalTransaction_Test
/// @notice Tests the finalizeWithdrawalTransaction function of the MockOptimismPortal2 contract.
contract MockOptimismPortal2_finalizeWithdrawalTransaction_Test is MockOptimismPortal2_TestInit {
    /// @notice Tests that a withdrawal the hook lets through pays its target and reports success.
    function test_finalizeWithdrawalTransaction_hookProceeds_succeeds() public {
        portal.setBridgeHookUnchecked(hook);
        Types.WithdrawalTransaction memory wtx = _withdrawal(target, 1 ether);
        _proveAndMature(wtx);

        vm.recordLogs();
        portal.finalizeWithdrawalTransaction(wtx);

        assertEq(_countTopic(vm.getRecordedLogs(), FINALIZED_TOPIC), 1);
        assertEq(target.balance, 1 ether);
    }

    /// @notice Tests that a held withdrawal emits no WithdrawalFinalized, so it is never reported
    ///         as paid, and that its value moves to the hook instead of its target.
    function test_finalizeWithdrawalTransaction_hookHolds_succeeds() public {
        portal.setBridgeHookUnchecked(hook);
        hook.setPassWithdrawals(false);
        Types.WithdrawalTransaction memory wtx = _withdrawal(target, 1 ether);
        _proveAndMature(wtx);

        vm.recordLogs();
        portal.finalizeWithdrawalTransaction(wtx);

        assertEq(_countTopic(vm.getRecordedLogs(), FINALIZED_TOPIC), 0);
        assertEq(target.balance, 0);
        assertEq(address(hook).balance, 1 ether);
        assertTrue(portal.heldWithdrawals(BridgeHookHashing.hashItem(_withdrawalItem(wtx))));
    }

    /// @notice Tests that a held withdrawal is announced by the Portal itself, the counterpart
    ///         to DepositHeld.
    function test_finalizeWithdrawalTransaction_emitsWithdrawalHeld_succeeds() public {
        portal.setBridgeHookUnchecked(hook);
        hook.setPassWithdrawals(false);
        Types.WithdrawalTransaction memory wtx = _withdrawal(target, 1 ether);
        _proveAndMature(wtx);

        vm.expectEmit(true, true, true, true, address(portal));
        emit WithdrawalHeld(BridgeHookHashing.hashItem(_withdrawalItem(wtx)));

        portal.finalizeWithdrawalTransaction(wtx);
    }

    /// @notice Tests that the hook is told when the withdrawal became finalizable, so a policy
    ///         can decide how long to wait before holding an uncleared one.
    function test_finalizeWithdrawalTransaction_passesFinalizableAt_succeeds() public {
        portal.setBridgeHookUnchecked(hook);
        Types.WithdrawalTransaction memory wtx = _withdrawal(target, 0);

        uint256 provenAt = block.timestamp;
        _proveAndMature(wtx);
        portal.finalizeWithdrawalTransaction(wtx);

        assertEq(hook.lastFinalizableAt(), provenAt + portal.proofMaturityDelaySeconds());
    }

    /// @notice Tests that a failing target still finalizes the withdrawal, reporting success as
    ///         false rather than reverting.
    function test_finalizeWithdrawalTransaction_targetReverts_succeeds() public {
        address reverter = address(new RevertingTarget());
        Types.WithdrawalTransaction memory wtx = _withdrawal(reverter, 1 ether);
        _proveAndMature(wtx);

        vm.expectEmit(true, true, true, true);
        emit WithdrawalFinalized(
            keccak256(abi.encode(wtx.nonce, wtx.sender, wtx.target, wtx.value, wtx.gasLimit, wtx.data)), false
        );
        portal.finalizeWithdrawalTransaction(wtx);
    }

    /// @notice Tests that an unproven withdrawal cannot be finalized.
    function test_finalizeWithdrawalTransaction_unproven_reverts() public {
        vm.expectRevert(MockOptimismPortal2.OptimismPortal_Unproven.selector);
        portal.finalizeWithdrawalTransaction(_withdrawal(target, 0));
    }

    /// @notice Tests that a withdrawal cannot be finalized before its proof has matured.
    function test_finalizeWithdrawalTransaction_proofTooYoung_reverts() public {
        Types.WithdrawalTransaction memory wtx = _withdrawal(target, 0);
        portal.proveWithdrawalTransaction(wtx, 0, Types.OutputRootProof(0, 0, 0, 0), new bytes[](0));

        vm.expectRevert(MockOptimismPortal2.OptimismPortal_ProofNotOldEnough.selector);
        portal.finalizeWithdrawalTransaction(wtx);
    }

    /// @notice Tests that a held withdrawal is consumed, so it cannot be finalized again to get
    ///         a second payout.
    function test_finalizeWithdrawalTransaction_heldThenRefinalized_reverts() public {
        portal.setBridgeHookUnchecked(hook);
        hook.setPassWithdrawals(false);
        Types.WithdrawalTransaction memory wtx = _withdrawal(target, 1 ether);
        _proveAndMature(wtx);
        portal.finalizeWithdrawalTransaction(wtx);

        vm.expectRevert(MockOptimismPortal2.OptimismPortal_AlreadyFinalized.selector);
        portal.finalizeWithdrawalTransaction(wtx);
    }
}

/// @title MockOptimismPortal2_completeWithdrawalTransaction_Test
/// @notice Tests the completeWithdrawalTransaction function of the MockOptimismPortal2 contract.
contract MockOptimismPortal2_completeWithdrawalTransaction_Test is MockOptimismPortal2_TestInit {
    Types.WithdrawalTransaction internal wtx;
    BridgeHookTypes.Item internal held;

    function setUp() public override {
        super.setUp();
        portal.setBridgeHookUnchecked(hook);
        hook.setPassWithdrawals(false);

        wtx = _withdrawal(target, 1 ether);
        _proveAndMature(wtx);
        portal.finalizeWithdrawalTransaction(wtx);
        held = _withdrawalItem(wtx);
    }

    /// @notice Tests that completing a held withdrawal pays the original target and reports the
    ///         withdrawal as finalized, on the committed terms.
    function test_completeWithdrawalTransaction_succeeds() public {
        vm.recordLogs();
        hook.completeWithdrawal(portal, held);

        assertEq(_countTopic(vm.getRecordedLogs(), FINALIZED_TOPIC), 1);
        assertEq(target.balance, 1 ether);
        assertFalse(portal.heldWithdrawals(BridgeHookHashing.hashItem(held)));
    }

    /// @notice Tests that the completion leaves no lingering L2 sender behind it.
    function test_completeWithdrawalTransaction_resetsL2Sender_succeeds() public {
        hook.completeWithdrawal(portal, held);
        assertEq(portal.l2Sender(), Constants.DEFAULT_L2_SENDER);
    }

    /// @notice Tests that only the hook may complete a held withdrawal.
    function test_completeWithdrawalTransaction_callerNotHook_reverts() public {
        vm.deal(alice, 1 ether);
        vm.prank(alice);
        vm.expectRevert(MockOptimismPortal2.OptimismPortal_NotBridgeHook.selector);
        portal.completeWithdrawalTransaction{ value: 1 ether }(held);
    }

    /// @notice Tests that a completion whose forwarded call fails reverts rather than consuming
    ///         the commitment, since a deleted commitment could never be retried.
    function test_completeWithdrawalTransaction_targetReverts_reverts() public {
        address reverter = address(new RevertingTarget());
        Types.WithdrawalTransaction memory badTx = _withdrawal(reverter, 1 ether);
        _proveAndMature(badTx);
        portal.finalizeWithdrawalTransaction(badTx);

        vm.expectRevert(MockOptimismPortal2.OptimismPortal_CompletionFailed.selector);
        hook.completeWithdrawal(portal, _withdrawalItem(badTx));
    }
}

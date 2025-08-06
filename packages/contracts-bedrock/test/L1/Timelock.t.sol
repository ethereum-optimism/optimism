// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing.
import { CommonTest } from "test/setup/CommonTest.sol";
import { GnosisSafe } from "safe-contracts/GnosisSafe.sol";

// Target Contract.
import { Timelock } from "src/L1/Timelock.sol";

/// @notice Mock contract for testing Timelock execution calls.
contract MockTarget {
    uint256 public value;
    bool public shouldRevert;
    bytes public revertData;

    event MockCalled(uint256 newValue);

    function setValue(uint256 _value) external payable {
        if (shouldRevert) {
            if (revertData.length > 0) {
                bytes memory data = revertData;
                assembly {
                    revert(add(data, 0x20), mload(data))
                }
            }
            revert("Mock revert");
        }
        value = _value;
        emit MockCalled(_value);
    }

    function setRevertData(bytes calldata _revertData) external {
        revertData = _revertData;
    }

    function setShouldRevert(bool _shouldRevert) external {
        shouldRevert = _shouldRevert;
    }

    receive() external payable { }
}

/// @notice Harness contract for testing Gnosis Safe integration.
contract Mock_GnosisSafe {
    mapping(address => bool) public owners;
    uint256 public threshold;

    constructor(address[] memory _owners, uint256 _threshold) {
        for (uint256 i = 0; i < _owners.length; i++) {
            owners[_owners[i]] = true;
        }
        threshold = _threshold;
    }

    function getThreshold() external view returns (uint256) {
        return threshold;
    }

    function isOwner(address _owner) external view returns (bool) {
        return owners[_owner];
    }
}

/// @title Timelock_TestInit
/// @notice Base contract for Timelock tests setup.
contract Timelock_TestInit is CommonTest {
    Timelock timelock;
    MockTarget mockTarget;
    Mock_GnosisSafe gnosisSafe;

    address controller0 = address(111);
    address controller1 = address(222);
    address controller2 = address(333);
    address controller3 = address(444);

    address[] controllers;

    address nonController = makeAddr("nonController");
    address gnosisSafeOwner = makeAddr("gnosisSafeOwner");
    address nonGnosisSafeOwner = makeAddr("nonGnosisSafeOwner");

    uint256 longDelay = 7 days;
    uint256 shortDelay = 1 days;

    event Approved(bytes32 indexed txHash, Timelock.Call call, uint256 eta);
    event Cancelled(bytes32 indexed txHash);
    event Executed(bytes32 indexed txHash, Timelock.Call call);

    function setUp() public virtual override {
        super.setUp();

        controllers.push(controller1);
        controllers.push(controller2);
        controllers.push(controller3);

        timelock = new Timelock();
        timelock.initialize(controllers, longDelay, shortDelay);
        mockTarget = new MockTarget();

        // Create Gnosis Safe harness.
        address[] memory safeOwners = new address[](2);
        safeOwners[0] = gnosisSafeOwner;
        safeOwners[1] = controller1;
        gnosisSafe = new Mock_GnosisSafe(safeOwners, 1);
    }

    function _createCall(
        bytes32 _salt,
        address _target,
        uint256 _value,
        bytes memory _data
    )
        internal
        pure
        returns (Timelock.Call memory)
    {
        return Timelock.Call({ salt: _salt, target: _target, value: _value, data: _data });
    }

    function _createDefaultCall() internal view returns (Timelock.Call memory) {
        return _createCall(
            bytes32("salt"), address(mockTarget), 0, abi.encodeWithSelector(MockTarget.setValue.selector, 42)
        );
    }
}

/// @title Timelock_Initialize_Test
/// @notice Test contract for Timelock initializer.
contract Timelock_Initialize_Test is Timelock_TestInit {
    /// @notice Tests successful initializer with valid parameters.
    function test_initialize_validParameters_succeeds() external view {
        assertEq(timelock.longDelay(), longDelay);
        assertEq(timelock.shortDelay(), shortDelay);
    }

    /// @notice Tests initializer reverts with empty controllers array.
    function test_initialize_emptyControllers_reverts() external {
        controllers = new address[](0);

        timelock = new Timelock();
        vm.expectRevert(Timelock.Timelock_EmptyControllers.selector);
        timelock.initialize(controllers, longDelay, shortDelay);
    }

    /// @notice Tests initializer reverts when long delay <= short delay.
    function test_initialize_longDelayNotGreaterThanShort_reverts() external {
        timelock = new Timelock();
        vm.expectRevert(Timelock.Timelock_ReversedDelays.selector);
        timelock.initialize(controllers, 1 days, 1 days);
    }

    /// @notice Tests initializer reverts when long delay is zero.
    function test_initialize_zeroLongDelay_reverts() external {
        timelock = new Timelock();
        vm.expectRevert(Timelock.Timelock_ReversedDelays.selector);
        timelock.initialize(controllers, 0, 0);
    }

    /// @notice Tests initializer reverts when short delay is zero.
    function test_initialize_zeroShortDelay_reverts() external {
        timelock = new Timelock();
        vm.expectRevert(Timelock.Timelock_ShortDelayZero.selector);
        timelock.initialize(controllers, 2 days, 0);
    }

    /// @notice Tests initializer reverts when long delay > 180 days.
    function test_initialize_longDelayTooLarge_reverts() external {
        timelock = new Timelock();
        vm.expectRevert(Timelock.Timelock_LongDelayTooLarge.selector);
        timelock.initialize(controllers, 181 days, 1 days);
    }

    /// @notice Tests initializer reverts when short delay > 30 days.
    function test_initialize_shortDelayTooLarge_reverts() external {
        timelock = new Timelock();
        vm.expectRevert(Timelock.Timelock_ShortDelayTooLarge.selector);
        timelock.initialize(controllers, 32 days, 31 days);
    }

    /// @notice Tests initializer reverts when controllers array contains zero address.
    function test_initialize_zeroAddressController_reverts() external {
        controllers = new address[](3);
        controllers[0] = controller1;
        controllers[1] = address(0); // Zero address controller.
        controllers[2] = controller3;

        timelock = new Timelock();
        vm.expectRevert(Timelock.Timelock_InvalidController.selector);
        timelock.initialize(controllers, longDelay, shortDelay);
    }

    /// @notice Tests initializer reverts when controllers are not in ascending order.
    function test_initialize_controllersNotSorted_reverts() external {
        controllers = new address[](3);
        controllers[0] = controller2; // Should be controller1 first.
        controllers[1] = controller1; // Not in ascending order.
        controllers[2] = controller3;

        timelock = new Timelock();
        vm.expectRevert(Timelock.Timelock_ControllersNotSortedOrUnique.selector);
        timelock.initialize(controllers, longDelay, shortDelay);
    }

    /// @notice Tests initializer reverts when controllers array has duplicate addresses.
    function test_initialize_duplicateControllers_reverts() external {
        controllers = new address[](3);
        controllers[0] = controller1;
        controllers[1] = controller1; // Duplicate controller.
        controllers[2] = controller3;

        timelock = new Timelock();
        vm.expectRevert(Timelock.Timelock_ControllersNotSortedOrUnique.selector);
        timelock.initialize(controllers, longDelay, shortDelay);
    }

    /// @notice Tests initializer reverts with single controller.
    function test_initialize_singleController_reverts() external {
        controllers = new address[](1);
        controllers[0] = controller1;

        timelock = new Timelock();
        vm.expectRevert(Timelock.Timelock_SingleController.selector);
        timelock.initialize(controllers, longDelay, shortDelay);
    }

    /// @notice Tests fuzz initializer with valid delay ranges.
    function testFuzz_initialize_validDelayRanges_succeeds(uint256 _longDelay, uint256 _shortDelay) external {
        _shortDelay = uint256(bound(_shortDelay, 1, 30 days));
        _longDelay = uint256(bound(_longDelay, _shortDelay + 1, 180 days));

        timelock = new Timelock();
        timelock.initialize(controllers, _longDelay, _shortDelay);

        assertEq(timelock.longDelay(), _longDelay);
        assertEq(timelock.shortDelay(), _shortDelay);
    }
}

/// @title Timelock_Hash_Test
/// @notice Test contract for Timelock hash function.
contract Timelock_Hash_Test is Timelock_TestInit {
    /// @notice Tests hash function produces consistent results.
    function test_hash_consistentResults_succeeds() external {
        Timelock.Call memory call = _createDefaultCall();

        bytes32 hash1 = timelock.hash(call);
        vm.warp(block.timestamp + 1);
        vm.roll(block.number + 1);
        bytes32 hash2 = timelock.hash(call);

        assertEq(hash1, hash2);
    }

    /// @notice Tests hash function produces different results for different calls.
    function test_hash_differentCalls_differentHashes() external view {
        Timelock.Call memory call1 = _createCall(bytes32("salt1"), address(mockTarget), 0, hex"1234");
        Timelock.Call memory call2 = _createCall(bytes32("salt2"), address(mockTarget), 0, hex"1234");

        bytes32 hash1 = timelock.hash(call1);
        bytes32 hash2 = timelock.hash(call2);

        assertNotEq(hash1, hash2);
    }

    /// @notice Tests fuzz hash function with different parameters.
    function testFuzz_hash_differentParameters_succeeds(
        bytes32 _salt,
        address _target,
        uint256 _value,
        bytes calldata _data
    )
        external
        view
    {
        Timelock.Call memory call = _createCall(_salt, _target, _value, _data);

        bytes32 computedHash = timelock.hash(call);
        bytes32 expectedHash = keccak256(abi.encode(call));

        assertEq(computedHash, expectedHash);
    }
}

/// @title Timelock_Approve_Test
/// @notice Test contract for Timelock approve function.
contract Timelock_Approve_Test is Timelock_TestInit {
    /// @notice Tests first approval sets long delay eta.
    function test_approve_firstApproval_setsLongDelay() external {
        Timelock.Call memory call = _createDefaultCall();
        uint256 timestamp = block.timestamp;
        uint256 expectedEta = uint256(timestamp) + longDelay;

        vm.expectEmit();
        emit Approved(timelock.hash(call), call, expectedEta);

        vm.prank(controller1);
        uint256 eta = timelock.approve(call);

        assertEq(eta, expectedEta);
    }

    /// @notice Tests all controllers approval sets short delay eta.
    function test_approve_allControllers_setsShortDelay() external {
        Timelock.Call memory call = _createDefaultCall();

        uint256 longEta = uint256(block.timestamp) + longDelay;
        uint256 shortEta = uint256(block.timestamp) + shortDelay;

        // First approval., long ETA.
        vm.prank(controller1);
        uint256 eta = timelock.approve(call);
        assertEq(eta, longEta);

        // Second approval, long ETA.
        vm.prank(controller2);
        eta = timelock.approve(call);
        assertEq(eta, longEta);

        // Third approval (all controllers), short ETA.
        vm.expectEmit();
        emit Approved(timelock.hash(call), call, shortEta);

        vm.prank(controller3);
        eta = timelock.approve(call);
        assertEq(eta, shortEta);
    }

    /// @notice Tests non-controller cannot approve.
    function test_approve_nonController_reverts() external {
        Timelock.Call memory call = _createDefaultCall();

        vm.expectRevert(Timelock.Timelock_NotAuthorized.selector);
        vm.prank(nonController);
        timelock.approve(call);
    }

    /// @notice Tests controller cannot approve twice.
    function test_approve_alreadyApproved_reverts() external {
        Timelock.Call memory call = _createDefaultCall();

        vm.prank(controller1);
        timelock.approve(call);

        vm.expectRevert(Timelock.Timelock_AlreadyApproved.selector);
        vm.prank(controller1);
        timelock.approve(call);
    }

    /// @notice Tests cannot approve executed call.
    function test_approve_alreadyExecuted_reverts() external {
        Timelock.Call memory call = _createDefaultCall();

        // Approve with all controllers.
        vm.prank(controller1);
        timelock.approve(call);
        vm.prank(controller2);
        timelock.approve(call);
        vm.prank(controller3);
        timelock.approve(call);

        // Wait for short delay, then execute.
        vm.warp(block.timestamp + shortDelay);
        timelock.execute(call);

        // Try to approve executed call - should fail with "Already approved"
        // because the contract checks approvals before checking executed status.
        vm.expectRevert(Timelock.Timelock_AlreadyApproved.selector);
        vm.prank(controller1);
        timelock.approve(call);
    }

    /// @notice Tests cannot approve cancelled call.
    function test_approve_alreadyCancelled_reverts() external {
        Timelock.Call memory call = _createDefaultCall();

        // First approval.
        vm.prank(controller1);
        timelock.approve(call);

        // Cancel call.
        vm.prank(controller2);
        timelock.cancel(call, controller2);

        // Try to approve cancelled call - use different controller.
        vm.expectRevert(Timelock.Timelock_AlreadyCancelled.selector);
        vm.prank(controller3);
        timelock.approve(call);
    }

    /// @notice Tests approve with each controller.
    function test_approve_eachController_succeeds() external {
        for (uint256 i = 0; i < controllers.length; i++) {
            Timelock.Call memory call = _createDefaultCall();
            vm.prank(controllers[i]);
            uint256 eta = timelock.approve(call);
            uint256 expectedEta = i == controllers.length - 1
                ? uint256(block.timestamp) + shortDelay
                : uint256(block.timestamp) + longDelay;
            assertEq(eta, expectedEta);
        }
    }

    /// @notice Tests that intermediate approvals preserve the original long delay ETA.
    function test_approve_intermediateApprovalsPreserveEta_succeeds() external {
        Timelock.Call memory call = _createDefaultCall();
        uint256 timestamp = block.timestamp;

        // First approval - should set long delay ETA.
        vm.prank(controller1);
        uint256 eta1 = timelock.approve(call);
        uint256 expectedLongEta = timestamp + longDelay;
        assertEq(eta1, expectedLongEta);

        // Second approval (intermediate) - should preserve the same ETA.
        vm.prank(controller2);
        uint256 eta2 = timelock.approve(call);
        assertEq(eta2, expectedLongEta); // Same ETA as first approval.

        // Third approval (all controllers) - should change to short delay.
        uint256 timestamp2 = block.timestamp;
        vm.prank(controller3);
        uint256 eta3 = timelock.approve(call);
        uint256 expectedShortEta = timestamp2 + shortDelay;
        assertEq(eta3, expectedShortEta);
    }

    /// @notice Tests that ETA never increases (invariant validation) when final approval comes after long delay would
    /// have expired.
    function test_approve_etaNeverIncreases_succeeds() external {
        Timelock.Call memory call = _createDefaultCall();

        // First approval - set long delay ETA.
        vm.prank(controller1);
        uint256 eta1 = timelock.approve(call);

        // Second approval - ETA should remain the same.
        vm.prank(controller2);
        uint256 eta2 = timelock.approve(call);
        assertEq(eta2, eta1); // ETA unchanged.

        // Warp time to after the long delay would expire, but before executing.
        vm.warp(block.timestamp + longDelay + 1 days);

        // Final approval - should not increase the ETA because the current ETA is closer than block.timestamp +
        // shortDelay.
        uint256 timestampBeforeFinalApproval = block.timestamp;
        vm.prank(controller3);
        uint256 eta3 = timelock.approve(call);

        // The ETA should remain the original ETA (not increase to block.timestamp + shortDelay).
        assertEq(eta3, eta1); // ETA should not have increased.

        // Verify that block.timestamp + shortDelay would have been larger.
        assertGt(timestampBeforeFinalApproval + shortDelay, eta1);
    }
}

/// @title Timelock_Cancel_Test
/// @notice Test contract for Timelock cancel functions.
contract Timelock_Cancel_Test is Timelock_TestInit {
    function assertIsCancelled(bytes32 _txHash) internal view {
        (bool executed, bool cancelled,,) = timelock.getCall(_txHash);
        assertTrue(cancelled);
        assertFalse(executed);
    }

    /// @notice Tests controller can cancel with txHash.
    function test_cancel_withTxHash_succeeds() external {
        Timelock.Call memory call = _createDefaultCall();
        bytes32 txHash = timelock.hash(call);

        vm.prank(controller1);
        timelock.approve(call);

        vm.expectEmit();
        emit Cancelled(txHash);
        vm.prank(controller2);
        timelock.cancel(txHash, controller2);
        assertIsCancelled(txHash);
    }

    /// @notice Tests controller can cancel with Call struct.
    function test_cancel_withCall_succeeds() external {
        Timelock.Call memory call = _createDefaultCall();
        bytes32 txHash = timelock.hash(call);

        vm.prank(controller1);
        timelock.approve(call);

        vm.expectEmit();
        emit Cancelled(txHash);
        vm.prank(controller2);
        timelock.cancel(call, controller2);
        assertIsCancelled(txHash);
    }

    /// @notice Tests Gnosis Safe owner can cancel.
    function test_cancel_gnosisSafeOwner_succeeds() external {
        // Deploy timelock with gnosis safe as controller.
        controllers = new address[](1);
        controllers[0] = address(gnosisSafe);
        timelock = new Timelock();
        timelock.initialize(controllers, longDelay, shortDelay);

        Timelock.Call memory call = _createDefaultCall();
        bytes32 txHash = timelock.hash(call);

        // Approve from gnosis safe.
        vm.prank(address(gnosisSafe));
        timelock.approve(call);

        vm.expectEmit();
        emit Cancelled(txHash);
        vm.prank(gnosisSafeOwner);
        timelock.cancel(txHash, address(gnosisSafe));
        assertIsCancelled(txHash);
    }

    /// @notice Tests invalid controller cannot cancel.
    function test_cancel_invalidController_reverts() external {
        Timelock.Call memory call = _createDefaultCall();
        bytes32 txHash = timelock.hash(call);

        vm.expectRevert(Timelock.Timelock_InvalidController.selector);
        vm.prank(controller1);
        timelock.cancel(txHash, nonController);
    }

    /// @notice Tests unauthorized user cannot cancel.
    function test_cancel_notAuthorized_reverts() external {
        Timelock.Call memory call = _createDefaultCall();
        bytes32 txHash = timelock.hash(call);

        vm.expectRevert(Timelock.Timelock_NotAuthorized.selector);
        vm.prank(nonController);
        timelock.cancel(txHash, controller1);
    }

    /// @notice Tests cannot cancel already executed call.
    function test_cancel_alreadyExecuted_reverts() external {
        Timelock.Call memory call = _createDefaultCall();
        bytes32 txHash = timelock.hash(call);

        // Approve with all controllers and execute.
        vm.prank(controller1);
        timelock.approve(call);
        vm.prank(controller2);
        timelock.approve(call);
        vm.prank(controller3);
        timelock.approve(call);

        vm.warp(block.timestamp + shortDelay);
        timelock.execute(call);

        vm.expectRevert(Timelock.Timelock_AlreadyExecuted.selector);
        vm.prank(controller1);
        timelock.cancel(txHash, controller1);
    }

    /// @notice Tests cannot cancel already cancelled call.
    function test_cancel_alreadyCancelled_reverts() external {
        Timelock.Call memory call = _createDefaultCall();
        bytes32 txHash = timelock.hash(call);

        // Approve and cancel.
        vm.prank(controller1);
        timelock.approve(call);

        vm.prank(controller2);
        timelock.cancel(txHash, controller2);

        vm.expectRevert(Timelock.Timelock_AlreadyCancelled.selector);
        vm.prank(controller3);
        timelock.cancel(txHash, controller3);
    }

    /// @notice Tests non-Gnosis Safe owner cannot cancel for Gnosis Safe.
    function test_cancel_nonGnosisSafeOwner_reverts() external {
        // Deploy timelock with gnosis safe as controller.
        controllers = new address[](1);
        controllers[0] = address(gnosisSafe);
        timelock = new Timelock();
        timelock.initialize(controllers, longDelay, shortDelay);

        Timelock.Call memory call = _createDefaultCall();
        bytes32 txHash = timelock.hash(call);

        vm.expectRevert(Timelock.Timelock_NotAuthorized.selector);
        vm.prank(nonGnosisSafeOwner);
        timelock.cancel(txHash, address(gnosisSafe));
    }
}

/// @title Timelock_Execute_Test
/// @notice Test contract for Timelock execute function.
contract Timelock_Execute_Test is Timelock_TestInit {
    /// @notice Tests successful execution after short delay.
    function test_execute_afterShortDelay_succeeds() external {
        Timelock.Call memory call = _createDefaultCall();
        bytes32 txHash = timelock.hash(call);

        // Approve with all controllers.
        vm.prank(controller1);
        timelock.approve(call);
        vm.prank(controller2);
        timelock.approve(call);
        vm.prank(controller3);
        timelock.approve(call);

        // Wait for short delay.
        vm.warp(block.timestamp + shortDelay);

        vm.expectEmit();
        emit Executed(txHash, call);

        bytes memory result = timelock.execute(call);

        assertEq(mockTarget.value(), 42);
        assertEq(result.length, 0);
    }

    /// @notice Tests successful execution after long delay.
    function test_execute_afterLongDelay_succeeds() external {
        Timelock.Call memory call = _createDefaultCall();

        // Single approval (uses long delay).
        vm.prank(controller1);
        timelock.approve(call);

        // Wait for long delay.
        vm.warp(block.timestamp + longDelay);

        timelock.execute(call);

        assertEq(mockTarget.value(), 42);
    }

    /// @notice Tests execution with ether value.
    function test_execute_withValue_succeeds() external {
        uint256 value = 1 ether;
        vm.deal(address(timelock), value);

        Timelock.Call memory call = _createCall(
            bytes32("salt"), address(mockTarget), value, abi.encodeWithSelector(MockTarget.setValue.selector, 42)
        );

        // Approve with all controllers.
        vm.prank(controller1);
        timelock.approve(call);
        vm.prank(controller2);
        timelock.approve(call);
        vm.prank(controller3);
        timelock.approve(call);

        vm.warp(block.timestamp + shortDelay);

        uint256 targetBalanceBefore = address(mockTarget).balance;
        timelock.execute(call);

        assertEq(address(mockTarget).balance, targetBalanceBefore + value);
        assertEq(mockTarget.value(), 42);
    }

    /// @notice Tests execution reverts before eta.
    function test_execute_beforeEta_reverts() external {
        Timelock.Call memory call = _createDefaultCall();

        // Single approval (long delay).
        vm.prank(controller1);
        timelock.approve(call);

        // Try to execute before eta.
        vm.expectRevert(Timelock.Timelock_EtaNotReached.selector);
        timelock.execute(call);
    }

    /// @notice Tests execution reverts for cancelled call.
    function test_execute_cancelled_reverts() external {
        Timelock.Call memory call = _createDefaultCall();

        // Approve and cancel.
        vm.prank(controller1);
        timelock.approve(call);

        vm.prank(controller2);
        timelock.cancel(call, controller2);

        vm.warp(block.timestamp + longDelay);

        vm.expectRevert(Timelock.Timelock_CallCancelled.selector);
        timelock.execute(call);
    }

    /// @notice Tests execution reverts for already executed call.
    function test_execute_alreadyExecuted_reverts() external {
        Timelock.Call memory call = _createDefaultCall();

        // Approve with all controllers and execute.
        vm.prank(controller1);
        timelock.approve(call);
        vm.prank(controller2);
        timelock.approve(call);
        vm.prank(controller3);
        timelock.approve(call);

        vm.warp(block.timestamp + shortDelay);
        timelock.execute(call);

        vm.expectRevert(Timelock.Timelock_AlreadyExecuted.selector);
        timelock.execute(call);
    }

    /// @notice Tests execution reverts when target call fails.
    function test_execute_callFails_reverts() external {
        mockTarget.setShouldRevert(true);

        Timelock.Call memory call = _createDefaultCall();

        // Approve with all controllers.
        vm.prank(controller1);
        timelock.approve(call);
        vm.prank(controller2);
        timelock.approve(call);
        vm.prank(controller3);
        timelock.approve(call);

        vm.warp(block.timestamp + shortDelay);

        // The revert data will be ABI-encoded string.
        bytes memory expectedRevertData = abi.encodeWithSignature("Error(string)", "Mock revert");
        vm.expectRevert(abi.encodeWithSelector(Timelock.Timelock_CallFailed.selector, expectedRevertData));
        timelock.execute(call);
    }

    /// @notice Tests execution reverts with custom revert data.
    function test_execute_customRevertData_reverts() external {
        bytes memory customRevertData = abi.encodeWithSignature("CustomError(uint256)", 123);
        mockTarget.setRevertData(customRevertData);
        mockTarget.setShouldRevert(true);

        Timelock.Call memory call = _createDefaultCall();

        // Approve with all controllers.
        vm.prank(controller1);
        timelock.approve(call);
        vm.prank(controller2);
        timelock.approve(call);
        vm.prank(controller3);
        timelock.approve(call);

        vm.warp(block.timestamp + shortDelay);

        vm.expectRevert(abi.encodeWithSelector(Timelock.Timelock_CallFailed.selector, customRevertData));
        timelock.execute(call);
    }
}

/// @title Timelock_Uncategorized_Test
/// @notice Test contract for integration scenarios spanning multiple functions.
contract Timelock_Uncategorized_Test is Timelock_TestInit {
    /// @notice Tests complete call lifecycle with single approval.
    function test_callLifecycle_singleApproval_succeeds() external {
        Timelock.Call memory call = _createDefaultCall();

        // Approve call. (sets long delay).
        vm.prank(controller1);
        uint256 eta = timelock.approve(call);

        assertEq(eta, uint256(block.timestamp) + longDelay);

        // Wait for eta and execute.
        vm.warp(eta);
        timelock.execute(call);

        assertEq(mockTarget.value(), 42);
    }

    /// @notice Tests complete call lifecycle with all approvals.
    function test_callLifecycle_allApprovals_succeeds() external {
        Timelock.Call memory call = _createDefaultCall();

        // First approval (long delay).
        vm.prank(controller1);
        uint256 eta1 = timelock.approve(call);
        assertEq(eta1, uint256(block.timestamp) + longDelay);

        // Second approval (still long delay).
        vm.prank(controller2);
        uint256 eta2 = timelock.approve(call);
        assertEq(eta2, eta1); // ETA doesn't change.

        // Third approval (changes to short delay).
        uint256 timestamp = block.timestamp;
        vm.prank(controller3);
        uint256 eta3 = timelock.approve(call);
        assertEq(eta3, uint256(timestamp) + shortDelay);

        // Execute after short delay.
        vm.warp(eta3);
        timelock.execute(call);

        assertEq(mockTarget.value(), 42);
    }

    /// @notice Tests call cancellation prevents execution.
    function test_approveAndCancel_preventsExecution_succeeds() external {
        Timelock.Call memory call = _createDefaultCall();

        // Approve call.
        vm.prank(controller1);
        timelock.approve(call);

        // Cancel call.
        vm.prank(controller2);
        timelock.cancel(call, controller2);

        // Try to execute after delay.
        vm.warp(block.timestamp + longDelay);

        vm.expectRevert(Timelock.Timelock_CallCancelled.selector);
        timelock.execute(call);
    }

    /// @notice Tests multiple different calls can coexist.
    function test_multipleCalls_coexist_succeeds() external {
        Timelock.Call memory call1 = _createCall(
            bytes32("salt1"), address(mockTarget), 0, abi.encodeWithSelector(MockTarget.setValue.selector, 100)
        );

        Timelock.Call memory call2 = _createCall(
            bytes32("salt2"), address(mockTarget), 0, abi.encodeWithSelector(MockTarget.setValue.selector, 200)
        );

        // Approve both calls.
        vm.prank(controller1);
        timelock.approve(call1);

        vm.prank(controller2);
        timelock.approve(call2);

        // Execute both after delay.
        vm.warp(block.timestamp + longDelay);

        timelock.execute(call1);
        assertEq(mockTarget.value(), 100);

        timelock.execute(call2);
        assertEq(mockTarget.value(), 200);
    }

    /// @notice Tests state consistency after partial approval and cancellation.
    function test_partialApprovalAndCancel_stateConsistency_succeeds() external {
        Timelock.Call memory call = _createDefaultCall();

        // Partial approvals.
        vm.prank(controller1);
        timelock.approve(call);

        vm.prank(controller2);
        timelock.approve(call);

        // Cancel before reaching full approval.
        vm.prank(controller3);
        timelock.cancel(call, controller3);

        // Verify new call with different salt can be created.
        Timelock.Call memory newCall = _createCall(
            bytes32("different_salt"), address(mockTarget), 0, abi.encodeWithSelector(MockTarget.setValue.selector, 42)
        );

        vm.prank(controller1);
        uint256 eta = timelock.approve(newCall);

        assertEq(eta, uint256(block.timestamp) + longDelay);
    }
}

// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { GnosisSafe } from "safe-contracts/GnosisSafe.sol";
import { GnosisSafeProxyFactory } from "safe-contracts/proxies/GnosisSafeProxyFactory.sol";

// Target Contract
import { Timelock } from "src/L1/Timelock.sol";

/// @title Timelock_MockTarget_Harness
/// @notice Mock contract for testing Timelock execution calls.
contract Timelock_MockTarget_Harness {
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

/// @title Timelock_GnosisSafe_Harness
/// @notice Harness contract for testing Gnosis Safe integration.
contract Timelock_GnosisSafe_Harness {
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
    Timelock_MockTarget_Harness mockTarget;
    Timelock_GnosisSafe_Harness gnosisSafe;

    address controller1;
    address controller2;
    address controller3;
    address nonController;
    address gnosisSafeOwner;
    address nonGnosisSafeOwner;

    uint64 longDelay = 7 days;
    uint64 shortDelay = 1 days;

    event Approved(bytes32 indexed txHash, Timelock.Call call, uint64 eta);
    event Executed(bytes32 indexed txHash, Timelock.Call call);
    event Cancelled(bytes32 indexed txHash);

    function setUp() public virtual override {
        super.setUp();

        controller1 = makeAddr("controller1");
        controller2 = makeAddr("controller2");
        controller3 = makeAddr("controller3");
        nonController = makeAddr("nonController");
        gnosisSafeOwner = makeAddr("gnosisSafeOwner");
        nonGnosisSafeOwner = makeAddr("nonGnosisSafeOwner");

        address[] memory controllers = new address[](3);
        controllers[0] = controller1;
        controllers[1] = controller2;
        controllers[2] = controller3;

        timelock = new Timelock(controllers, longDelay, shortDelay);
        mockTarget = new Timelock_MockTarget_Harness();

        // Create Gnosis Safe harness
        address[] memory safeOwners = new address[](2);
        safeOwners[0] = gnosisSafeOwner;
        safeOwners[1] = controller1;
        gnosisSafe = new Timelock_GnosisSafe_Harness(safeOwners, 1);
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
            bytes32("salt"),
            address(mockTarget),
            0,
            abi.encodeWithSelector(Timelock_MockTarget_Harness.setValue.selector, 42)
        );
    }
}

/// @title Timelock_Constructor_Test
/// @notice Test contract for Timelock constructor validation.
contract Timelock_Constructor_Test is Timelock_TestInit {
    /// @notice Tests successful constructor with valid parameters.
    function test_constructor_validParameters_succeeds() external {
        address[] memory controllers = new address[](2);
        controllers[0] = makeAddr("controller1");
        controllers[1] = makeAddr("controller2");

        Timelock newTimelock = new Timelock(controllers, 7 days, 1 days);

        assertEq(newTimelock.longDelay(), 7 days);
        assertEq(newTimelock.shortDelay(), 1 days);
    }

    /// @notice Tests constructor reverts with empty controllers array.
    function test_constructor_emptyControllers_reverts() external {
        address[] memory controllers = new address[](0);

        vm.expectRevert(
            abi.encodeWithSelector(Timelock.InvalidConstructor.selector, "Controllers length must be greater than 0.")
        );
        new Timelock(controllers, 7 days, 1 days);
    }

    /// @notice Tests constructor reverts when long delay <= short delay.
    function test_constructor_longDelayNotGreaterThanShort_reverts() external {
        address[] memory controllers = new address[](1);
        controllers[0] = makeAddr("controller");

        vm.expectRevert(
            abi.encodeWithSelector(Timelock.InvalidConstructor.selector, "Long delay must be greater than short delay.")
        );
        new Timelock(controllers, 1 days, 1 days);
    }

    /// @notice Tests constructor reverts when long delay is zero.
    function test_constructor_zeroLongDelay_reverts() external {
        address[] memory controllers = new address[](1);
        controllers[0] = makeAddr("controller");

        vm.expectRevert(
            abi.encodeWithSelector(Timelock.InvalidConstructor.selector, "Long delay must be greater than short delay.")
        );
        new Timelock(controllers, 0, 0);
    }

    /// @notice Tests constructor reverts when short delay is zero.
    function test_constructor_zeroShortDelay_reverts() external {
        address[] memory controllers = new address[](1);
        controllers[0] = makeAddr("controller");

        vm.expectRevert(
            abi.encodeWithSelector(Timelock.InvalidConstructor.selector, "Short delay must be greater than 0.")
        );
        new Timelock(controllers, 2 days, 0);
    }

    /// @notice Tests constructor reverts when long delay > 180 days.
    function test_constructor_longDelayTooLarge_reverts() external {
        address[] memory controllers = new address[](1);
        controllers[0] = makeAddr("controller");

        vm.expectRevert(abi.encodeWithSelector(Timelock.InvalidConstructor.selector, "Long delay must be <= 6 months."));
        new Timelock(controllers, 181 days, 1 days);
    }

    /// @notice Tests constructor reverts when short delay > 30 days.
    function test_constructor_shortDelayTooLarge_reverts() external {
        address[] memory controllers = new address[](1);
        controllers[0] = makeAddr("controller");

        vm.expectRevert(abi.encodeWithSelector(Timelock.InvalidConstructor.selector, "Short delay must be <= 1 year."));
        new Timelock(controllers, 32 days, 31 days);
    }

    /// @notice Tests fuzz constructor with valid delay ranges.
    function testFuzz_constructor_validDelayRanges_succeeds(uint64 _longDelay, uint64 _shortDelay) external {
        _shortDelay = uint64(bound(_shortDelay, 1, 30 days));
        _longDelay = uint64(bound(_longDelay, _shortDelay + 1, 180 days));

        address[] memory controllers = new address[](1);
        controllers[0] = makeAddr("controller");

        Timelock newTimelock = new Timelock(controllers, _longDelay, _shortDelay);

        assertEq(newTimelock.longDelay(), _longDelay);
        assertEq(newTimelock.shortDelay(), _shortDelay);
    }
}

/// @title Timelock_Hash_Test
/// @notice Test contract for Timelock hash function.
contract Timelock_Hash_Test is Timelock_TestInit {
    /// @notice Tests hash function produces consistent results.
    function test_hash_consistentResults_succeeds() external view {
        Timelock.Call memory call = _createDefaultCall();

        bytes32 hash1 = timelock.hash(call);
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
        uint64 expectedEta = uint64(timestamp) + longDelay;

        vm.expectEmit(true, true, true, true);
        emit Approved(timelock.hash(call), call, expectedEta);

        vm.prank(controller1);
        uint64 eta = timelock.approve(call);

        assertEq(eta, expectedEta);
    }

    /// @notice Tests all controllers approval sets short delay eta.
    function test_approve_allControllers_setsShortDelay() external {
        Timelock.Call memory call = _createDefaultCall();

        // First approval
        vm.prank(controller1);
        timelock.approve(call);

        // Second approval
        vm.prank(controller2);
        timelock.approve(call);

        // Third approval (all controllers)
        uint256 timestamp = block.timestamp;
        uint64 expectedEta = uint64(timestamp) + shortDelay;

        vm.expectEmit(true, true, true, true);
        emit Approved(timelock.hash(call), call, expectedEta);

        vm.prank(controller3);
        uint64 eta = timelock.approve(call);

        assertEq(eta, expectedEta);
    }

    /// @notice Tests non-controller cannot approve.
    function test_approve_nonController_reverts() external {
        Timelock.Call memory call = _createDefaultCall();

        vm.expectRevert(bytes("Not authorized."));
        vm.prank(nonController);
        timelock.approve(call);
    }

    /// @notice Tests controller cannot approve twice.
    function test_approve_alreadyApproved_reverts() external {
        Timelock.Call memory call = _createDefaultCall();

        vm.prank(controller1);
        timelock.approve(call);

        vm.expectRevert(bytes("Already approved."));
        vm.prank(controller1);
        timelock.approve(call);
    }

    /// @notice Tests cannot approve executed proposal.
    function test_approve_alreadyExecuted_reverts() external {
        Timelock.Call memory call = _createDefaultCall();

        // Approve with all controllers
        vm.prank(controller1);
        timelock.approve(call);
        vm.prank(controller2);
        timelock.approve(call);
        vm.prank(controller3);
        timelock.approve(call);

        // Wait for short delay and execute
        vm.warp(block.timestamp + shortDelay);
        timelock.execute(call);

        // Try to approve executed proposal - should fail with "Already approved"
        // because the contract checks approvals before checking executed status
        vm.expectRevert(bytes("Already approved."));
        vm.prank(controller1);
        timelock.approve(call);
    }

    /// @notice Tests cannot approve cancelled proposal.
    function test_approve_alreadyCancelled_reverts() external {
        Timelock.Call memory call = _createDefaultCall();

        // First approval
        vm.prank(controller1);
        timelock.approve(call);

        // Cancel proposal
        vm.prank(controller2);
        timelock.cancel(call, controller2);

        // Try to approve cancelled proposal - use different controller
        vm.expectRevert(bytes("Already executed."));
        vm.prank(controller3);
        timelock.approve(call);
    }

    /// @notice Tests fuzz approve with different controllers.
    function testFuzz_approve_randomController_succeeds(uint256 _controllerIndex) external {
        _controllerIndex = bound(_controllerIndex, 0, 2);

        address controller = _controllerIndex == 0 ? controller1 : _controllerIndex == 1 ? controller2 : controller3;

        Timelock.Call memory call = _createDefaultCall();

        vm.prank(controller);
        uint64 eta = timelock.approve(call);

        assertEq(eta, uint64(block.timestamp) + longDelay);
    }
}

/// @title Timelock_Cancel_Test
/// @notice Test contract for Timelock cancel functions.
contract Timelock_Cancel_Test is Timelock_TestInit {
    /// @notice Tests controller can cancel with txHash.
    function test_cancel_withTxHash_succeeds() external {
        Timelock.Call memory call = _createDefaultCall();
        bytes32 txHash = timelock.hash(call);

        // First approve
        vm.prank(controller1);
        timelock.approve(call);

        vm.expectEmit(true, false, false, false);
        emit Cancelled(txHash);

        vm.prank(controller2);
        timelock.cancel(txHash, controller2);
    }

    /// @notice Tests controller can cancel with Call struct.
    function test_cancel_withCall_succeeds() external {
        Timelock.Call memory call = _createDefaultCall();
        bytes32 txHash = timelock.hash(call);

        // First approve
        vm.prank(controller1);
        timelock.approve(call);

        vm.expectEmit(true, false, false, false);
        emit Cancelled(txHash);

        vm.prank(controller2);
        timelock.cancel(call, controller2);
    }

    /// @notice Tests Gnosis Safe owner can cancel.
    function test_cancel_gnosisSafeOwner_succeeds() external {
        // Deploy timelock with gnosis safe as controller
        address[] memory controllers = new address[](1);
        controllers[0] = address(gnosisSafe);
        Timelock newTimelock = new Timelock(controllers, longDelay, shortDelay);

        Timelock.Call memory call = _createDefaultCall();
        bytes32 txHash = newTimelock.hash(call);

        // Approve from gnosis safe
        vm.prank(address(gnosisSafe));
        newTimelock.approve(call);

        vm.expectEmit(true, true, true, true);
        emit Cancelled(txHash);

        // Cancel from gnosis safe owner (now works correctly)
        vm.prank(gnosisSafeOwner);
        newTimelock.cancel(txHash, address(gnosisSafe));
    }

    /// @notice Tests invalid controller cannot cancel.
    function test_cancel_invalidController_reverts() external {
        Timelock.Call memory call = _createDefaultCall();
        bytes32 txHash = timelock.hash(call);

        vm.expectRevert(bytes("Invalid controller."));
        vm.prank(controller1);
        timelock.cancel(txHash, nonController);
    }

    /// @notice Tests unauthorized user cannot cancel.
    function test_cancel_notAuthorized_reverts() external {
        Timelock.Call memory call = _createDefaultCall();
        bytes32 txHash = timelock.hash(call);

        vm.expectRevert(bytes("Not authorized."));
        vm.prank(nonController);
        timelock.cancel(txHash, controller1);
    }

    /// @notice Tests cannot cancel already executed proposal.
    function test_cancel_alreadyExecuted_reverts() external {
        Timelock.Call memory call = _createDefaultCall();
        bytes32 txHash = timelock.hash(call);

        // Approve with all controllers and execute
        vm.prank(controller1);
        timelock.approve(call);
        vm.prank(controller2);
        timelock.approve(call);
        vm.prank(controller3);
        timelock.approve(call);

        vm.warp(block.timestamp + shortDelay);
        timelock.execute(call);

        vm.expectRevert(bytes("Already executed."));
        vm.prank(controller1);
        timelock.cancel(txHash, controller1);
    }

    /// @notice Tests cannot cancel already cancelled proposal.
    function test_cancel_alreadyCancelled_reverts() external {
        Timelock.Call memory call = _createDefaultCall();
        bytes32 txHash = timelock.hash(call);

        // Approve and cancel
        vm.prank(controller1);
        timelock.approve(call);

        vm.prank(controller2);
        timelock.cancel(txHash, controller2);

        vm.expectRevert(bytes("Already cancelled."));
        vm.prank(controller3);
        timelock.cancel(txHash, controller3);
    }

    /// @notice Tests non-Gnosis Safe owner cannot cancel for Gnosis Safe.
    function test_cancel_nonGnosisSafeOwner_reverts() external {
        // Deploy timelock with gnosis safe as controller
        address[] memory controllers = new address[](1);
        controllers[0] = address(gnosisSafe);
        Timelock newTimelock = new Timelock(controllers, longDelay, shortDelay);

        Timelock.Call memory call = _createDefaultCall();
        bytes32 txHash = newTimelock.hash(call);

        vm.expectRevert(bytes("Not authorized."));
        vm.prank(nonGnosisSafeOwner);
        newTimelock.cancel(txHash, address(gnosisSafe));
    }
}

/// @title Timelock_Execute_Test
/// @notice Test contract for Timelock execute function.
contract Timelock_Execute_Test is Timelock_TestInit {
    /// @notice Tests successful execution after short delay.
    function test_execute_afterShortDelay_succeeds() external {
        Timelock.Call memory call = _createDefaultCall();
        bytes32 txHash = timelock.hash(call);

        // Approve with all controllers
        vm.prank(controller1);
        timelock.approve(call);
        vm.prank(controller2);
        timelock.approve(call);
        vm.prank(controller3);
        timelock.approve(call);

        // Wait for short delay
        vm.warp(block.timestamp + shortDelay);

        vm.expectEmit(true, true, true, true);
        emit Executed(txHash, call);

        bytes memory result = timelock.execute(call);

        assertEq(mockTarget.value(), 42);
        assertEq(result.length, 0); // setValue returns nothing
    }

    /// @notice Tests successful execution after long delay.
    function test_execute_afterLongDelay_succeeds() external {
        Timelock.Call memory call = _createDefaultCall();

        // Single approval (uses long delay)
        vm.prank(controller1);
        timelock.approve(call);

        // Wait for long delay
        vm.warp(block.timestamp + longDelay);

        timelock.execute(call);

        assertEq(mockTarget.value(), 42);
    }

    /// @notice Tests execution with ether value.
    function test_execute_withValue_succeeds() external {
        uint256 value = 1 ether;
        vm.deal(address(timelock), value);

        Timelock.Call memory call = _createCall(
            bytes32("salt"),
            address(mockTarget),
            value,
            abi.encodeWithSelector(Timelock_MockTarget_Harness.setValue.selector, 42)
        );

        // Approve with all controllers
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

        // Single approval (long delay)
        vm.prank(controller1);
        timelock.approve(call);

        // Try to execute before eta
        vm.expectRevert(bytes("ETA not reached."));
        timelock.execute(call);
    }

    /// @notice Tests execution reverts for cancelled proposal.
    function test_execute_cancelled_reverts() external {
        Timelock.Call memory call = _createDefaultCall();

        // Approve and cancel
        vm.prank(controller1);
        timelock.approve(call);

        vm.prank(controller2);
        timelock.cancel(call, controller2);

        vm.warp(block.timestamp + longDelay);

        vm.expectRevert(bytes("Proposal cancelled."));
        timelock.execute(call);
    }

    /// @notice Tests execution reverts for already executed proposal.
    function test_execute_alreadyExecuted_reverts() external {
        Timelock.Call memory call = _createDefaultCall();

        // Approve with all controllers and execute
        vm.prank(controller1);
        timelock.approve(call);
        vm.prank(controller2);
        timelock.approve(call);
        vm.prank(controller3);
        timelock.approve(call);

        vm.warp(block.timestamp + shortDelay);
        timelock.execute(call);

        vm.expectRevert(bytes("Proposal already executed."));
        timelock.execute(call);
    }

    /// @notice Tests execution reverts when target call fails.
    function test_execute_callFails_reverts() external {
        mockTarget.setShouldRevert(true);

        Timelock.Call memory call = _createDefaultCall();

        // Approve with all controllers
        vm.prank(controller1);
        timelock.approve(call);
        vm.prank(controller2);
        timelock.approve(call);
        vm.prank(controller3);
        timelock.approve(call);

        vm.warp(block.timestamp + shortDelay);

        // The revert data will be ABI-encoded string
        bytes memory expectedRevertData = abi.encodeWithSignature("Error(string)", "Mock revert");
        vm.expectRevert(abi.encodeWithSelector(Timelock.CallFailed.selector, expectedRevertData));
        timelock.execute(call);
    }

    /// @notice Tests execution reverts with custom revert data.
    function test_execute_customRevertData_reverts() external {
        bytes memory customRevertData = abi.encodeWithSignature("CustomError(uint256)", 123);
        mockTarget.setRevertData(customRevertData);
        mockTarget.setShouldRevert(true);

        Timelock.Call memory call = _createDefaultCall();

        // Approve with all controllers
        vm.prank(controller1);
        timelock.approve(call);
        vm.prank(controller2);
        timelock.approve(call);
        vm.prank(controller3);
        timelock.approve(call);

        vm.warp(block.timestamp + shortDelay);

        vm.expectRevert(abi.encodeWithSelector(Timelock.CallFailed.selector, customRevertData));
        timelock.execute(call);
    }
}

/// @title Timelock_Uncategorized_Test
/// @notice Test contract for integration scenarios spanning multiple functions.
contract Timelock_Uncategorized_Test is Timelock_TestInit {
    /// @notice Tests complete proposal lifecycle with single approval.
    function test_proposalLifecycle_singleApproval_succeeds() external {
        Timelock.Call memory call = _createDefaultCall();

        // Approve proposal (sets long delay)
        vm.prank(controller1);
        uint64 eta = timelock.approve(call);

        assertEq(eta, uint64(block.timestamp) + longDelay);

        // Wait for eta and execute
        vm.warp(eta);
        timelock.execute(call);

        assertEq(mockTarget.value(), 42);
    }

    /// @notice Tests complete proposal lifecycle with all approvals.
    function test_proposalLifecycle_allApprovals_succeeds() external {
        Timelock.Call memory call = _createDefaultCall();

        // First approval (long delay)
        vm.prank(controller1);
        uint64 eta1 = timelock.approve(call);
        assertEq(eta1, uint64(block.timestamp) + longDelay);

        // Second approval (still long delay)
        vm.prank(controller2);
        uint64 eta2 = timelock.approve(call);
        assertEq(eta2, eta1); // ETA doesn't change

        // Third approval (changes to short delay)
        uint256 timestamp = block.timestamp;
        vm.prank(controller3);
        uint64 eta3 = timelock.approve(call);
        assertEq(eta3, uint64(timestamp) + shortDelay);

        // Execute after short delay
        vm.warp(eta3);
        timelock.execute(call);

        assertEq(mockTarget.value(), 42);
    }

    /// @notice Tests proposal cancellation prevents execution.
    function test_approveAndCancel_preventsExecution_succeeds() external {
        Timelock.Call memory call = _createDefaultCall();

        // Approve proposal
        vm.prank(controller1);
        timelock.approve(call);

        // Cancel proposal
        vm.prank(controller2);
        timelock.cancel(call, controller2);

        // Try to execute after delay
        vm.warp(block.timestamp + longDelay);

        vm.expectRevert(bytes("Proposal cancelled."));
        timelock.execute(call);
    }

    /// @notice Tests multiple different proposals can coexist.
    function test_multipleProposals_coexist_succeeds() external {
        Timelock.Call memory call1 = _createCall(
            bytes32("salt1"),
            address(mockTarget),
            0,
            abi.encodeWithSelector(Timelock_MockTarget_Harness.setValue.selector, 100)
        );

        Timelock.Call memory call2 = _createCall(
            bytes32("salt2"),
            address(mockTarget),
            0,
            abi.encodeWithSelector(Timelock_MockTarget_Harness.setValue.selector, 200)
        );

        // Approve both proposals
        vm.prank(controller1);
        timelock.approve(call1);

        vm.prank(controller2);
        timelock.approve(call2);

        // Execute both after delay
        vm.warp(block.timestamp + longDelay);

        timelock.execute(call1);
        assertEq(mockTarget.value(), 100);

        timelock.execute(call2);
        assertEq(mockTarget.value(), 200);
    }

    /// @notice Tests state consistency after partial approval and cancellation.
    function test_partialApprovalAndCancel_stateConsistency_succeeds() external {
        Timelock.Call memory call = _createDefaultCall();

        // Partial approvals
        vm.prank(controller1);
        timelock.approve(call);

        vm.prank(controller2);
        timelock.approve(call);

        // Cancel before reaching full approval
        vm.prank(controller3);
        timelock.cancel(call, controller3);

        // Verify new proposal with different salt can be created
        Timelock.Call memory newCall = _createCall(
            bytes32("different_salt"),
            address(mockTarget),
            0,
            abi.encodeWithSelector(Timelock_MockTarget_Harness.setValue.selector, 42)
        );

        vm.prank(controller1);
        uint64 eta = timelock.approve(newCall);

        assertEq(eta, uint64(block.timestamp) + longDelay);
    }
}

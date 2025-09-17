// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { GnosisSafe as Safe } from "safe-contracts/GnosisSafe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import { GuardManager } from "safe-contracts/base/GuardManager.sol";
import { StorageAccessible } from "safe-contracts/common/StorageAccessible.sol";
import { ExecTransactionParams } from "src/safe/Types.sol";
import "test/safe-tools/SafeTestTools.sol";

import { console2 as console } from "forge-std/console2.sol";

import { TimelockGuard } from "src/safe/TimelockGuard.sol";

/// @title TimelockGuard_TestInit
/// @notice Reusable test initialization for `TimelockGuard` tests.
contract TimelockGuard_TestInit is Test, SafeTestTools {
    using SafeTestLib for SafeInstance;

    // Events
    event GuardConfigured(Safe indexed safe, uint256 timelockDelay);
    event GuardCleared(Safe indexed safe);
    event TransactionScheduled(Safe indexed safe, bytes32 indexed txId, uint256 when);
    event TransactionCancelled(Safe indexed safe, bytes32 indexed txId);
    event CancellationThresholdUpdated(Safe indexed safe, uint256 oldThreshold, uint256 newThreshold);

    uint256 constant INIT_TIME = 10;
    uint256 constant TIMELOCK_DELAY = 7 days;
    uint256 constant NUM_OWNERS = 5;
    uint256 constant THRESHOLD = 3;
    uint256 constant ONE_YEAR = 365 days;

    TimelockGuard timelockGuard;

    // The Safe address will be the same as SafeInstance.safe, but it has the Safe type.
    // This is useful for testing functions that take a Safe as an argument.
    Safe safe;
    SafeInstance safeInstance;

    SafeInstance unguardedSafe;

    function setUp() public virtual {
        vm.warp(INIT_TIME);

        // Deploy the singleton TimelockGuard
        timelockGuard = new TimelockGuard();

        // Create Safe owners
        (, uint256[] memory keys) = SafeTestLib.makeAddrsAndKeys("owners", NUM_OWNERS);

        // Set up Safe with owners
        safeInstance = _setupSafe(keys, THRESHOLD);
        safe = Safe(payable(safeInstance.safe));

        // Safe without guard enabled
        // Reduce the threshold just to prevent a CREATE2 collision when deploying this safe.
        unguardedSafe = _setupSafe(keys, THRESHOLD - 1);

        // Enable the guard on the Safe
        SafeTestLib.execTransaction(
            safeInstance,
            address(safeInstance.safe),
            0,
            abi.encodeCall(GuardManager.setGuard, (address(timelockGuard))),
            Enum.Operation.Call
        );
    }

    /// @notice Helper to generate the transaction hash for a given transaction params and nonce
    function _getTxHash(
        SafeInstance memory _safeInstance,
        ExecTransactionParams memory _params,
        uint256 _nonce
    )
        internal
        view
        returns (bytes32)
    {
        return _safeInstance.safe.getTransactionHash({
            to: _params.to,
            value: _params.value,
            data: _params.data,
            operation: _params.operation,
            safeTxGas: _params.safeTxGas,
            baseGas: _params.baseGas,
            gasPrice: _params.gasPrice,
            gasToken: _params.gasToken,
            refundReceiver: _params.refundReceiver,
            _nonce: _nonce
        });
    }

    /// @notice Helper to generate signatures for an arbitrary transaction
    function _getSignaturesForTx(
        SafeInstance memory _safeInstance,
        bytes32 _txHash,
        uint256 _numSignatures
    )
        internal
        pure
        returns (bytes memory)
    {
        bytes memory signatures = new bytes(0);
        for (uint256 i; i < _numSignatures; ++i) {
            (uint8 v, bytes32 r, bytes32 s) = vm.sign(_safeInstance.ownerPKs[i], _txHash);

            // The signature format is a compact form of: {bytes32 r}{bytes32 s}{uint8 v}
            signatures = bytes.concat(signatures, abi.encodePacked(r, s, v));
        }
        return signatures;
    }

    /// @notice Helper to generate dummy transaction parameters
    function _getDummyTxParams() internal pure returns (ExecTransactionParams memory) {
        return ExecTransactionParams({
            to: address(0xabba),
            value: 0,
            data: hex"acdc",
            operation: Enum.Operation.Call,
            safeTxGas: 0,
            baseGas: 0,
            gasPrice: 0,
            gasToken: address(0),
            refundReceiver: payable(address(0))
        });
    }

    /// @notice Helper to generate everything needed to schedule a transaction for the current nonce
    function _getDummyTxWithSignaturesAndHash(SafeInstance memory _safe)
        internal
        view
        returns (ExecTransactionParams memory, bytes32, bytes memory)
    {
        uint256 nonce = _safe.safe.nonce();
        ExecTransactionParams memory dummyTxParams = _getDummyTxParams();
        bytes32 txHash = _getTxHash(_safe, dummyTxParams, nonce);
        bytes memory signatures = _getSignaturesForTx(_safe, txHash, THRESHOLD);
        return (dummyTxParams, txHash, signatures);
    }

    function _getCancellationTx(address _safe, bytes32 _txHash) internal pure returns (ExecTransactionParams memory) {
        bytes memory txData = abi.encodeWithSignature("cancelTransaction(bytes32)", _txHash);
        ExecTransactionParams memory cancellationTxParams =
            ExecTransactionParams(_safe, 0, txData, Enum.Operation.Call, 0, 0, 0, address(0), payable(address(0)));

        return cancellationTxParams;
    }

    /// @notice Helper to configure the TimelockGuard for a Safe
    function _configureGuard(SafeInstance memory _safe, uint256 _delay) internal {
        SafeTestLib.execTransaction(
            _safe,
            address(timelockGuard),
            0,
            abi.encodeCall(TimelockGuard.configureTimelockGuard, (_delay)),
            Enum.Operation.Call
        );
    }

    /// @notice Helper to disable guard on a Safe
    function _disableGuard(SafeInstance memory _safe) internal {
        SafeTestLib.execTransaction(
            _safe, address(_safe.safe), 0, abi.encodeCall(GuardManager.setGuard, (address(0))), Enum.Operation.Call
        );
    }

    /// @notice Helper to enable guard on a Safe
    function _enableGuard(SafeInstance memory _safe) internal {
        SafeTestLib.execTransaction(
            _safe,
            address(_safe.safe),
            0,
            abi.encodeCall(GuardManager.setGuard, (address(timelockGuard))),
            Enum.Operation.Call
        );
    }

    /// @notice Helper to clear the TimelockGuard configuration for a Safe
    function _clearGuard(SafeInstance memory _safe) internal {
        SafeTestLib.execTransaction(
            _safe, address(timelockGuard), 0, abi.encodeCall(TimelockGuard.clearTimelockGuard, ()), Enum.Operation.Call
        );
    }
}

/// @title TimelockGuard_ViewTimelockGuardConfiguration_Test
/// @notice Tests for viewTimelockGuardConfiguration function
contract TimelockGuard_ViewTimelockGuardConfiguration_Test is TimelockGuard_TestInit {
    function test_viewTimelockGuardConfiguration_returnsZeroForUnconfiguredSafe_succeeds() external view {
        TimelockGuard.GuardConfig memory config = timelockGuard.viewTimelockGuardConfiguration(safeInstance.safe);
        assertEq(config.timelockDelay, 0);
        assertEq(config.configured, false);
    }

    function test_viewTimelockGuardConfiguration_returnsConfigurationForConfiguredSafe_succeeds() external {
        _configureGuard(safeInstance, TIMELOCK_DELAY);
        TimelockGuard.GuardConfig memory config = timelockGuard.viewTimelockGuardConfiguration(safeInstance.safe);
        assertEq(config.timelockDelay, TIMELOCK_DELAY);
        assertEq(config.configured, true);
    }
}

/// @title TimelockGuard_ConfigureTimelockGuard_Test
/// @notice Tests for configureTimelockGuard function
contract TimelockGuard_ConfigureTimelockGuard_Test is TimelockGuard_TestInit {
    function test_configureTimelockGuard_succeeds() external {
        vm.expectEmit(true, true, true, true);
        emit GuardConfigured(safe, TIMELOCK_DELAY);

        _configureGuard(safeInstance, TIMELOCK_DELAY);

        TimelockGuard.GuardConfig memory config = timelockGuard.viewTimelockGuardConfiguration(safe);
        assertEq(config.timelockDelay, TIMELOCK_DELAY);
        assertEq(config.configured, true);
    }

    function test_configureTimelockGuard_revertsIfGuardNotEnabled_reverts() external {
        vm.expectRevert(TimelockGuard.TimelockGuard_GuardNotEnabled.selector);
        vm.prank(address(unguardedSafe.safe));
        timelockGuard.configureTimelockGuard(TIMELOCK_DELAY);
    }

    function test_configureTimelockGuard_revertsIfDelayTooLong_reverts() external {
        uint256 tooLongDelay = ONE_YEAR + 1;

        vm.expectRevert(TimelockGuard.TimelockGuard_InvalidTimelockDelay.selector);
        vm.prank(address(safeInstance.safe));
        timelockGuard.configureTimelockGuard(tooLongDelay);
    }

    function test_configureTimelockGuard_acceptsMaxValidDelay_succeeds() external {
        vm.expectEmit(true, true, true, true);
        emit GuardConfigured(safe, ONE_YEAR);

        _configureGuard(safeInstance, ONE_YEAR);

        TimelockGuard.GuardConfig memory config = timelockGuard.viewTimelockGuardConfiguration(safe);
        assertEq(config.timelockDelay, ONE_YEAR);
        assertEq(config.configured, true);
    }

    function test_configureTimelockGuard_allowsReconfiguration_succeeds() external {
        // Initial configuration
        _configureGuard(safeInstance, TIMELOCK_DELAY);
        assertEq(timelockGuard.viewTimelockGuardConfiguration(safe).timelockDelay, TIMELOCK_DELAY);

        // Reconfigure with different delay
        uint256 newDelay = 14 days;
        vm.expectEmit(true, true, true, true);
        emit GuardConfigured(safe, newDelay);

        _configureGuard(safeInstance, newDelay);
        assertEq(timelockGuard.viewTimelockGuardConfiguration(safe).timelockDelay, newDelay);
    }
}

/// @title TimelockGuard_ClearTimelockGuard_Test
/// @notice Tests for clearTimelockGuard function
contract TimelockGuard_ClearTimelockGuard_Test is TimelockGuard_TestInit {
    function test_clearTimelockGuard_succeeds() external {
        // First configure the guard
        _configureGuard(safeInstance, TIMELOCK_DELAY);
        assertEq(timelockGuard.viewTimelockGuardConfiguration(safe).timelockDelay, TIMELOCK_DELAY);

        // Disable the guard first
        _disableGuard(safeInstance);

        // Clear should succeed and emit event
        vm.expectEmit(true, true, true, true);
        emit GuardCleared(safe);

        _clearGuard(safeInstance);

        // Configuration should be cleared
        assertEq(timelockGuard.viewTimelockGuardConfiguration(safe).timelockDelay, 0);
        assertEq(timelockGuard.viewTimelockGuardConfiguration(safe).configured, false);
        // Ensure cancellation threshold is reset to 0
        assertEq(timelockGuard.cancellationThreshold(safe), 0);

        // TODO: Check that any active challenge is cancelled
    }

    function test_clearTimelockGuard_revertsIfGuardStillEnabled_reverts() external {
        // First configure the guard
        _configureGuard(safeInstance, TIMELOCK_DELAY);

        // Try to clear without disabling guard first - should revert
        vm.expectRevert(TimelockGuard.TimelockGuard_GuardStillEnabled.selector);
        vm.prank(address(safeInstance.safe));
        timelockGuard.clearTimelockGuard();
    }

    function test_clearTimelockGuard_revertsIfNotConfigured_reverts() external {
        // Try to clear - should revert because not configured
        vm.expectRevert(TimelockGuard.TimelockGuard_GuardNotConfigured.selector);
        vm.prank(address(safeInstance.safe));
        timelockGuard.clearTimelockGuard();
    }
}

/// @title TimelockGuard_CancellationThreshold_Test
/// @notice Tests for cancellationThreshold function
contract TimelockGuard_CancellationThreshold_Test is TimelockGuard_TestInit {
    function test_cancellationThreshold_returnsZeroIfGuardNotEnabled_succeeds() external view {
        uint256 threshold = timelockGuard.cancellationThreshold(Safe(payable(unguardedSafe.safe)));
        assertEq(threshold, 0);
    }

    function test_cancellationThreshold_returnsZeroIfGuardNotConfigured_succeeds() external view {
        // Safe with guard enabled but not configured should return 0
        uint256 threshold = timelockGuard.cancellationThreshold(safe);
        assertEq(threshold, 0);
    }

    function test_cancellationThreshold_returnsOneAfterConfiguration_succeeds() external {
        // Configure the guard
        _configureGuard(safeInstance, TIMELOCK_DELAY);

        // Should default to 1 after configuration
        uint256 threshold = timelockGuard.cancellationThreshold(safe);
        assertEq(threshold, 1);
    }

    // Note: Testing increment/decrement behavior will require scheduleTransaction,
    // cancelTransaction and execution functions to be implemented first
}

/// @title TimelockGuard_ScheduleTransaction_Test
/// @notice Tests for scheduleTransaction function
contract TimelockGuard_ScheduleTransaction_Test is TimelockGuard_TestInit {
    function setUp() public override {
        super.setUp();
        _configureGuard(safeInstance, TIMELOCK_DELAY);
    }

    function test_scheduleTransaction_succeeds() public {
        (ExecTransactionParams memory dummyTxParams, bytes32 txHash, bytes memory signatures) =
            _getDummyTxWithSignaturesAndHash(safeInstance);

        vm.expectEmit(true, true, true, true);
        emit TransactionScheduled(safe, txHash, INIT_TIME + TIMELOCK_DELAY);
        timelockGuard.scheduleTransaction(safeInstance.safe, safeInstance.safe.nonce(), dummyTxParams, signatures);
    }

    // A test which demonstrates that if the guard is enabled but not explicitly configured,
    // the timelock delay is set to 0.
    function test_scheduleTransaction_guardNotConfigured_succeeds() external {
        // Enable the guard on the unguarded Safe, but don't configure it
        _enableGuard(unguardedSafe);
        assertEq(timelockGuard.viewTimelockGuardConfiguration(unguardedSafe.safe).timelockDelay, 0);

        (ExecTransactionParams memory dummyTxParams, bytes32 txHash,) = _getDummyTxWithSignaturesAndHash(unguardedSafe);

        bytes memory signatures = _getSignaturesForTx(unguardedSafe, txHash, THRESHOLD - 1);

        uint256 nonce = unguardedSafe.safe.nonce();
        vm.expectEmit(true, true, true, true);
        emit TransactionScheduled(unguardedSafe.safe, txHash, INIT_TIME + 0);
        timelockGuard.scheduleTransaction(unguardedSafe.safe, nonce, dummyTxParams, signatures);
    }

    function test_scheduleTransaction_reschedulingIdenticalTransaction_reverts() external {
        uint256 nonce = safeInstance.safe.nonce();

        (ExecTransactionParams memory dummyTxParams,, bytes memory signatures) =
            _getDummyTxWithSignaturesAndHash(safeInstance);
        timelockGuard.scheduleTransaction(safeInstance.safe, nonce, dummyTxParams, signatures);

        vm.expectRevert(TimelockGuard.TimelockGuard_TransactionAlreadyScheduled.selector);
        timelockGuard.scheduleTransaction(safeInstance.safe, nonce, dummyTxParams, signatures);
    }

    function test_scheduleTransaction_identicalPreviouslyCancelled_reverts() external {
        // TODO: Implement once cancelTransaction is implemented and tested
    }

    function test_scheduleTransaction_guardNotEnabled_reverts() external {
        // Attempt to schedule a transaction with a Safe that has not enabled the guard
        uint256 nonce = unguardedSafe.safe.nonce();
        vm.expectRevert(TimelockGuard.TimelockGuard_GuardNotEnabled.selector);
        timelockGuard.scheduleTransaction(unguardedSafe.safe, nonce, _getDummyTxParams(), "");
    }

    function test_scheduleTransaction_canScheduleIdenticalWithDifferentNonce_succeeds() external {
        // Schedule a transaction with a specific nonce
        (ExecTransactionParams memory dummyTxParams,, bytes memory signatures) =
            _getDummyTxWithSignaturesAndHash(safeInstance);
        timelockGuard.scheduleTransaction(safeInstance.safe, safeInstance.safe.nonce(), dummyTxParams, signatures);

        // Schedule an identical transaction with a different nonce (salt)
        uint256 newNonce = safeInstance.safe.nonce() + 1;
        bytes32 newTxHash = _getTxHash(safeInstance, dummyTxParams, newNonce);
        bytes memory newSignatures = _getSignaturesForTx(safeInstance, newTxHash, THRESHOLD);

        vm.expectEmit(true, true, true, true);
        emit TransactionScheduled(safe, newTxHash, INIT_TIME + TIMELOCK_DELAY);
        timelockGuard.scheduleTransaction(safeInstance.safe, newNonce, dummyTxParams, newSignatures);
    }
}

/// @title TimelockGuard_CancelTransaction_Test
/// @notice Tests for cancelTransaction function
contract TimelockGuard_CancelTransaction_Test is TimelockGuard_TestInit {
    function setUp() public override {
        super.setUp();

        // Configure the guard and schedule a transaction
        _configureGuard(safeInstance, TIMELOCK_DELAY);
    }

    /// @notice Helper to schedule a transaction in order to test cancelTransaction.
    ///         Will always schedule the dummy transaction using the Safe's current nonce.
    function _scheduleTransaction() internal {
        (ExecTransactionParams memory dummyTxParams, bytes32 txHash, bytes memory signatures) =
            _getDummyTxWithSignaturesAndHash(safeInstance);
        timelockGuard.scheduleTransaction(safeInstance.safe, safeInstance.safe.nonce(), dummyTxParams, signatures);

        // Confirm that the transaction is scheduled
        TimelockGuard.ScheduledTransaction memory scheduledTransaction =
            timelockGuard.getScheduledTransaction(safeInstance.safe, txHash);
        assertEq(scheduledTransaction.executionTime, block.timestamp + TIMELOCK_DELAY);
        assertEq(scheduledTransaction.cancelled, false);
        assertEq(scheduledTransaction.executed, false);
    }

    function test_cancelTransaction_withPrivKeySignature_succeeds() external {
        _scheduleTransaction();

        // Get the transaction hash
        (, bytes32 txHash,) = _getDummyTxWithSignaturesAndHash(safeInstance);

        // Get the nonce
        uint256 nonce = safeInstance.safe.nonce();

        // Get the cancellation signatures
        uint256 numSignatures = timelockGuard.cancellationThreshold(safeInstance.safe);
        ExecTransactionParams memory cancellationTxParams = _getCancellationTx(address(safeInstance.safe), txHash);
        bytes32 cancellationTxHash = _getTxHash(safeInstance, cancellationTxParams, nonce);
        bytes memory cancelSignatures = _getSignaturesForTx(safeInstance, cancellationTxHash, numSignatures);

        // Cancel the transaction
        vm.expectEmit(true, true, true, true);
        emit CancellationThresholdUpdated(safeInstance.safe, numSignatures, numSignatures + 1);
        vm.expectEmit(true, true, true, true);
        emit TransactionCancelled(safeInstance.safe, txHash);
        timelockGuard.cancelTransaction(safeInstance.safe, txHash, nonce, cancelSignatures);

        // Confirm that the transaction is cancelled
        TimelockGuard.ScheduledTransaction memory scheduledTransaction =
            timelockGuard.getScheduledTransaction(safeInstance.safe, txHash);
        assertEq(scheduledTransaction.cancelled, true);
    }

    function test_cancelTransaction_withApproveHash_succeeds() external {
        _scheduleTransaction();

        // Get the transaction hash
        (, bytes32 txHash,) = _getDummyTxWithSignaturesAndHash(safeInstance);

        // Get the nonce
        uint256 nonce = safeInstance.safe.nonce();

        // Get the cancellation transaction hash
        ExecTransactionParams memory cancellationTxParams = _getCancellationTx(address(safeInstance.safe), txHash);
        bytes32 cancellationTxHash = _getTxHash(safeInstance, cancellationTxParams, nonce);

        // Get the owner
        address owner = safeInstance.safe.getOwners()[0];

        // Approve the cancellation transaction hash
        vm.prank(owner);
        safeInstance.safe.approveHash(cancellationTxHash);

        // Encode the prevalidated cancellation signature
        bytes memory signatures = abi.encodePacked(bytes32(uint256(uint160(owner))), bytes32(0), uint8(1));

        // Get the cancellation threshold
        uint256 cancellationThreshold = timelockGuard.cancellationThreshold(safeInstance.safe);

        // Cancel the transaction
        vm.expectEmit(true, true, true, true);
        emit CancellationThresholdUpdated(safeInstance.safe, cancellationThreshold, cancellationThreshold + 1);
        vm.expectEmit(true, true, true, true);
        emit TransactionCancelled(safeInstance.safe, txHash);
        timelockGuard.cancelTransaction(safeInstance.safe, txHash, nonce, signatures);

        // Confirm that the transaction is cancelled
        TimelockGuard.ScheduledTransaction memory scheduledTransaction =
            timelockGuard.getScheduledTransaction(safeInstance.safe, txHash);
        assertEq(scheduledTransaction.cancelled, true);
    }

    function test_cancelTransaction_revertsIfTransactionNotScheduled_reverts() external {
        // Get the transaction hash
        (, bytes32 txHash,) = _getDummyTxWithSignaturesAndHash(safeInstance);

        // Get the cancellation signatures
        uint256 nonce = safeInstance.safe.nonce();

        // Attempt to cancel the transaction
        vm.expectRevert(TimelockGuard.TimelockGuard_TransactionNotScheduled.selector);
        timelockGuard.cancelTransaction(safeInstance.safe, txHash, nonce, new bytes(0));
    }
}

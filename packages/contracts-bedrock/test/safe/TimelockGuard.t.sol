// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { Safe } from "safe-contracts/Safe.sol";
import { GuardManager } from "safe-contracts/base/GuardManager.sol";
import { ITransactionGuard } from "interfaces/safe/ITransactionGuard.sol";
import "test/safe-tools/SafeTestTools.sol";
import { Reverter } from "test/mocks/Callers.sol";

import { TimelockGuard } from "src/safe/TimelockGuard.sol";
import { SaferSafes } from "src/safe/SaferSafes.sol";

using TransactionBuilder for TransactionBuilder.Transaction;

/// @title TransactionBuilder
/// @notice Facilitates the construction of transactions and signatures, and provides helper methods
///        for scheduling, executing, and cancelling transactions.
library TransactionBuilder {
    // A struct type used to construct a transaction for scheduling and execution
    struct Transaction {
        SafeInstance safeInstance;
        TimelockGuard.ExecTransactionParams params;
        uint256 nonce;
        bytes32 hash;
        bytes signatures;
    }

    address internal constant VM_ADDR = 0x7109709ECfa91a80626fF3989D68f67F5b1DD12D;

    /// @notice Sets a nonce value on the provided transaction struct.
    function setNonce(Transaction memory _tx, uint256 _nonce) internal pure {
        _tx.nonce = _nonce;
    }

    /// @notice Computes and stores the Safe transaction hash for the struct.
    function setHash(Transaction memory _tx) internal view {
        _tx.hash = _tx.safeInstance.safe.getTransactionHash({
            to: _tx.params.to,
            value: _tx.params.value,
            data: _tx.params.data,
            operation: _tx.params.operation,
            safeTxGas: _tx.params.safeTxGas,
            baseGas: _tx.params.baseGas,
            gasPrice: _tx.params.gasPrice,
            gasToken: _tx.params.gasToken,
            refundReceiver: _tx.params.refundReceiver,
            _nonce: _tx.nonce
        });
    }

    /// @notice Collects signatures from the first `_num` owners for the transaction.
    function setSignatures(Transaction memory _tx, uint256 _num) internal pure {
        bytes memory signatures = new bytes(0);
        for (uint256 i; i < _num; ++i) {
            (uint8 v, bytes32 r, bytes32 s) = Vm(VM_ADDR).sign(_tx.safeInstance.ownerPKs[i], _tx.hash);

            // The signature format is a compact form of: {bytes32 r}{bytes32 s}{uint8 v}
            signatures = bytes.concat(signatures, abi.encodePacked(r, s, v));
        }
        _tx.signatures = signatures;
    }

    /// @notice Collects enough signatures to meet the Safe threshold.
    function setSignatures(Transaction memory _tx) internal view {
        uint256 num = _tx.safeInstance.safe.getThreshold();
        setSignatures(_tx, num);
    }

    /// @notice Updates the hash and signatures for a specific approval count.
    function updateTransaction(Transaction memory _tx, uint256 _num) internal view {
        _tx.setHash();
        _tx.setSignatures(_num);
    }

    /// @notice Updates the hash and threshold-based signatures on the transaction.
    function updateTransaction(Transaction memory _tx) internal view {
        _tx.setHash();
        _tx.setSignatures();
    }

    /// @notice Schedules the transaction with the supplied TimelockGuard instance.
    function scheduleTransaction(Transaction memory _tx, TimelockGuard _timelockGuard) internal {
        _timelockGuard.scheduleTransaction(_tx.safeInstance.safe, _tx.nonce, _tx.params, _tx.signatures);
    }

    /// @notice Executes the transaction via the underlying Safe contract.
    function executeTransaction(Transaction memory _tx, address _owner) internal {
        Vm(VM_ADDR).prank(_owner);
        _tx.safeInstance.safe.execTransaction(
            _tx.params.to,
            _tx.params.value,
            _tx.params.data,
            _tx.params.operation,
            _tx.params.safeTxGas,
            _tx.params.baseGas,
            _tx.params.gasPrice,
            _tx.params.gasToken,
            _tx.params.refundReceiver,
            _tx.signatures
        );
    }

    /// @notice Returns a fresh transaction struct copy with identical fields.
    function deepCopy(Transaction memory _tx) internal pure returns (Transaction memory) {
        return Transaction({
            safeInstance: _tx.safeInstance,
            nonce: _tx.nonce,
            params: _tx.params,
            signatures: _tx.signatures,
            hash: _tx.hash
        });
    }

    /// @notice Builds the corresponding cancellation transaction for the provided data.
    function makeCancellationTransaction(
        Transaction memory _tx,
        TimelockGuard _timelockGuard
    )
        internal
        view
        returns (Transaction memory)
    {
        // Deep copy the transaction
        Transaction memory cancellation = Transaction({
            safeInstance: _tx.safeInstance,
            nonce: _tx.nonce,
            params: _tx.params,
            signatures: _tx.signatures,
            hash: _tx.hash
        });

        // Empty out the params, then set based on the cancellation transaction format
        delete cancellation.params;
        cancellation.params.to = address(_timelockGuard);
        cancellation.params.data = abi.encodeCall(TimelockGuard.signCancellation, (_tx.hash));

        // Get only the number of signatures required for the cancellation transaction
        uint256 cancellationThreshold = _timelockGuard.cancellationThreshold(_tx.safeInstance.safe);

        cancellation.updateTransaction(cancellationThreshold);
        return cancellation;
    }
}

/// @title TimelockGuard_TestInit
/// @notice Reusable test initialization for `TimelockGuard` tests.
abstract contract TimelockGuard_TestInit is Test, SafeTestTools {
    using stdStorage for StdStorage;
    // Events

    event GuardConfigured(Safe indexed safe, uint256 timelockDelay);
    event TransactionScheduled(Safe indexed safe, bytes32 indexed txId, uint256 when);
    event TransactionCancelled(Safe indexed safe, bytes32 indexed txId);
    event CancellationThresholdUpdated(Safe indexed safe, uint256 oldThreshold, uint256 newThreshold);
    event TransactionExecuted(Safe indexed safe, bytes32 txHash);
    event Message(string message);
    event TransactionsNotCancelled(Safe indexed safe, uint256 n);

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

    /// @notice Deploys test fixtures and configures default Safe instances.
    function setUp() public virtual {
        vm.warp(INIT_TIME);

        // Deploy the combined SaferSafes contract which implements TimelockGuard
        SaferSafes saferSafesImpl = new SaferSafes();
        timelockGuard = TimelockGuard(address(saferSafesImpl));

        // Set up Safe with owners
        safeInstance = _deploySafe("owners", NUM_OWNERS, THRESHOLD);
        safe = Safe(payable(safeInstance.safe));

        // Safe without guard enabled
        unguardedSafe = _deploySafe("owners-unguarded", NUM_OWNERS, THRESHOLD);

        // Enable the guard on the Safe
        _enableGuard(safeInstance);
    }

    /// @notice Set the cancellation threshold storage for a given Safe on the TimelockGuard.
    /// @param _safe The Safe for which to override the threshold.
    /// @param _value The threshold value to set.
    function _setCancellationThreshold(Safe _safe, uint256 _value) internal {
        uint256 slot = stdstore.target(address(timelockGuard)).sig("cancellationThreshold(address)").with_key(
            address(_safe)
        ).find();
        vm.store(address(timelockGuard), bytes32(slot), bytes32(uint256(_value)));
    }

    /// @notice Deploys a Safe with the given owners and threshold
    function _deploySafe(
        string memory _prefix,
        uint256 _numOwners,
        uint256 _threshold
    )
        internal
        returns (SafeInstance memory)
    {
        (, uint256[] memory keys) = SafeTestLib.makeAddrsAndKeys(_prefix, _numOwners);
        return _setupSafe(keys, _threshold);
    }

    /// @notice Builds an empty transaction wrapper for a Safe instance.
    function _createEmptyTransaction(SafeInstance memory _safeInstance)
        internal
        view
        returns (TransactionBuilder.Transaction memory)
    {
        TransactionBuilder.Transaction memory transaction;
        // transaction.params will have null values
        transaction.safeInstance = _safeInstance;
        transaction.nonce = _safeInstance.safe.nonce();
        transaction.updateTransaction();
        return transaction;
    }

    /// @notice Creates a dummy transaction populated with placeholder call data.
    function _createDummyTransaction(SafeInstance memory _safeInstance)
        internal
        view
        returns (TransactionBuilder.Transaction memory)
    {
        TransactionBuilder.Transaction memory transaction = _createEmptyTransaction(_safeInstance);
        transaction.params.to = address(0xabba);
        transaction.params.data = hex"acdc";
        transaction.updateTransaction();
        return transaction;
    }

    /// @notice Helper to configure the TimelockGuard for a Safe
    function _configureGuard(SafeInstance memory _safe, uint256 _delay) internal {
        vm.startPrank(_safe.owners[0]);
        SafeTestLib.execTransaction(
            _safe, address(timelockGuard), 0, abi.encodeCall(TimelockGuard.configureTimelockGuard, (_delay))
        );
        vm.stopPrank();
    }

    /// @notice Helper to enable guard on a Safe
    function _enableGuard(SafeInstance memory _safe) internal {
        SafeTestLib.execTransaction(
            _safe, address(_safe.safe), 0, abi.encodeCall(GuardManager.setGuard, (address(timelockGuard)))
        );
    }

    /// @notice Helper to disable guard on a Safe
    function _disableGuard(SafeInstance memory _safe) internal {
        // Create, schedule, and execute a transaction to disable the guard
        TransactionBuilder.Transaction memory disableGuardTx = _createEmptyTransaction(safeInstance);
        disableGuardTx.params.to = address(_safe.safe);
        disableGuardTx.params.data = abi.encodeCall(GuardManager.setGuard, (address(0)));
        disableGuardTx.updateTransaction();
        disableGuardTx.scheduleTransaction(timelockGuard);

        // Wait for timelock delay to pass
        vm.warp(block.timestamp + TIMELOCK_DELAY + 1);

        // Execute the disable guard transaction
        disableGuardTx.executeTransaction(_safe.owners[0]);
    }
}

/// @title TimelockGuard_TimelockDelay_Test
/// @notice Tests for TimelockDelay function
contract TimelockGuard_TimelockDelay_Test is TimelockGuard_TestInit {
    /// @notice Ensures an unconfigured Safe reports a zero timelock delay.
    function test_timelockDelay_returnsZeroForUnconfiguredSafe_succeeds() external view {
        uint256 delay = timelockGuard.timelockDelay(safeInstance.safe);
        assertEq(delay, 0);
    }

    /// @notice Fuzz test: Validates the configuration view reflects the stored timelock delay for any valid delay.
    function testFuzz_timelockDelay_returnsConfigurationForConfiguredSafe_succeeds(uint256 _delay_) external {
        _delay_ = bound(_delay_, 1, ONE_YEAR); // Restrict to valid range
        _configureGuard(safeInstance, _delay_);
        uint256 delay_ = timelockGuard.timelockDelay(safeInstance.safe);
        assertEq(delay_, _delay_);
    }
}

/// @title TimelockGuard_ConfigureTimelockGuard_Test
/// @notice Tests for configureTimelockGuard function
contract TimelockGuard_ConfigureTimelockGuard_Test is TimelockGuard_TestInit {
    /// @notice Verifies the guard can be configured with various valid delays.
    function testFuzz_configureTimelockGuard_validDelay_succeeds(uint256 _delay) external {
        _delay = bound(_delay, 1, ONE_YEAR);

        vm.expectEmit(true, true, true, true);
        emit GuardConfigured(safe, _delay);

        _configureGuard(safeInstance, _delay);

        uint256 delay = timelockGuard.timelockDelay(safe);
        assertEq(delay, _delay);
    }

    /// @notice Confirms delays above the maximum revert during configuration.
    function testFuzz_configureTimelockGuard_delayTooLong_reverts(uint256 _delay) external {
        _delay = bound(_delay, ONE_YEAR + 1, type(uint256).max);

        vm.expectRevert(TimelockGuard.TimelockGuard_InvalidTimelockDelay.selector);
        vm.prank(address(safeInstance.safe));
        timelockGuard.configureTimelockGuard(_delay);
    }

    /// @notice Checks configuration reverts when the contract is not 1.4.1.
    function test_configureTimelockGuard_withWrongVersion_reverts() external {
        // nosemgrep: sol-style-use-abi-encodecall
        vm.mockCall(address(safeInstance.safe), abi.encodeWithSignature("VERSION()"), abi.encode("1.4.0"));
        vm.expectRevert(TimelockGuard.TimelockGuard_InvalidVersion.selector, address(timelockGuard));
        vm.prank(address(safeInstance.safe));
        timelockGuard.configureTimelockGuard(TIMELOCK_DELAY);
    }

    /// @notice Checks configuration succeeds even with pre-release versions of the Safe contract.
    function test_configureTimelockGuard_withPatchReleases_succeeds() external {
        // nosemgrep: sol-style-use-abi-encodecall
        vm.mockCall(address(safeInstance.safe), abi.encodeWithSignature("VERSION()"), abi.encode("1.4.1-rc.1"));
        vm.prank(address(safeInstance.safe));
        timelockGuard.configureTimelockGuard(TIMELOCK_DELAY);
    }

    /// @notice Asserts the maximum valid delay configures successfully.
    function test_configureTimelockGuard_acceptsMaxValidDelay_succeeds() external {
        vm.expectEmit(true, true, true, true);
        emit GuardConfigured(safe, ONE_YEAR);

        _configureGuard(safeInstance, ONE_YEAR);

        uint256 delay = timelockGuard.timelockDelay(safe);
        assertEq(delay, ONE_YEAR);
    }

    /// @notice Demonstrates the guard can be reconfigured to a new delay.
    function test_configureTimelockGuard_allowsReconfiguration_succeeds() external {
        // Initial configuration
        _configureGuard(safeInstance, TIMELOCK_DELAY);
        assertEq(timelockGuard.timelockDelay(safe), TIMELOCK_DELAY);

        uint256 newDelay = TIMELOCK_DELAY + 1;

        // Setup and schedule the reconfiguration transaction
        TransactionBuilder.Transaction memory reconfigureGuardTx = _createEmptyTransaction(safeInstance);
        reconfigureGuardTx.params.to = address(timelockGuard);
        reconfigureGuardTx.params.data = abi.encodeCall(TimelockGuard.configureTimelockGuard, (newDelay));
        reconfigureGuardTx.updateTransaction();
        reconfigureGuardTx.scheduleTransaction(timelockGuard);

        vm.warp(block.timestamp + TIMELOCK_DELAY);

        // Reconfigure with different delay
        vm.expectEmit(true, true, true, true);
        emit GuardConfigured(safe, newDelay);

        _configureGuard(safeInstance, newDelay);
        assertEq(timelockGuard.timelockDelay(safe), newDelay);
    }

    /// @notice Ensures setting delay to zero clears the configuration.
    function test_configureTimelockGuard_clearConfiguration_succeeds() external {
        // First configure the guard
        _configureGuard(safeInstance, TIMELOCK_DELAY);
        assertEq(timelockGuard.timelockDelay(safe), TIMELOCK_DELAY);

        // Configure timelock delay to 0 should succeed and emit event
        vm.expectEmit(true, true, true, true);
        emit GuardConfigured(safe, 0);
        vm.prank(address(safeInstance.safe));
        timelockGuard.configureTimelockGuard(0);

        // Timelock delay should be set to 0
        assertEq(timelockGuard.timelockDelay(safe), 0);
    }

    /// @notice Checks clearing succeeds even if the guard was never configured.
    function test_configureTimelockGuard_notConfigured_succeeds() external {
        // Try to clear - should succeed even if not yet configured
        vm.expectEmit(true, true, true, true);
        emit GuardConfigured(safe, 0);
        vm.prank(address(safeInstance.safe));
        timelockGuard.configureTimelockGuard(0);
    }
}

/// @title TimelockGuard_CancellationThreshold_Test
/// @notice Tests for cancellationThreshold function
contract TimelockGuard_CancellationThreshold_Test is TimelockGuard_TestInit {
    /// @notice Ensures an enabled but unconfigured guard yields a zero threshold.
    function test_cancellationThreshold_returnsZeroIfGuardNotConfigured_succeeds() external view {
        // Safe with guard enabled but not configured should return 0
        uint256 threshold = timelockGuard.cancellationThreshold(safe);
        assertEq(threshold, 0);
    }

    /// @notice Confirms the default threshold becomes one after configuration.
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
    /// @notice Configures the guard before each scheduleTransaction test.
    function setUp() public override {
        super.setUp();
        _configureGuard(safeInstance, TIMELOCK_DELAY);
    }

    /// @notice Ensures scheduling emits the expected event and stores state.
    function test_scheduleTransaction_succeeds() public {
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(safeInstance);

        vm.expectEmit(true, true, true, true);
        emit TransactionScheduled(safe, dummyTx.hash, INIT_TIME + TIMELOCK_DELAY);
        dummyTx.scheduleTransaction(timelockGuard);
    }

    // A test which demonstrates that if the guard is enabled but not explicitly configured,
    // the timelock delay is set to 0.
    /// @notice Checks scheduling reverts if the guard lacks configuration.
    function test_scheduleTransaction_guardNotConfigured_reverts() external {
        // Enable the guard on the unguarded Safe, but don't configure it
        _enableGuard(unguardedSafe);
        assertEq(timelockGuard.timelockDelay(unguardedSafe.safe), 0);

        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(unguardedSafe);
        vm.expectRevert(TimelockGuard.TimelockGuard_GuardNotConfigured.selector);
        dummyTx.scheduleTransaction(timelockGuard);
    }

    /// @notice Verifies rescheduling an identical pending transaction reverts.
    function test_scheduleTransaction_reschedulingIdenticalTransaction_reverts() external {
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(safeInstance);

        timelockGuard.scheduleTransaction(safeInstance.safe, dummyTx.nonce, dummyTx.params, dummyTx.signatures);

        vm.expectRevert(TimelockGuard.TimelockGuard_TransactionAlreadyScheduled.selector);
        timelockGuard.scheduleTransaction(dummyTx.safeInstance.safe, dummyTx.nonce, dummyTx.params, dummyTx.signatures);
    }

    /// @notice Confirms scheduling fails when the guard has not been enabled.
    function test_scheduleTransaction_guardNotEnabled_reverts() external {
        // Attempt to schedule a transaction with a Safe that has enabled the guard but
        // has not configured it.
        _enableGuard(unguardedSafe);
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(unguardedSafe);

        vm.expectRevert(TimelockGuard.TimelockGuard_GuardNotConfigured.selector);
        dummyTx.scheduleTransaction(timelockGuard);
    }

    /// @notice Demonstrates identical payloads can be scheduled with distinct nonces.
    function test_scheduleTransaction_canScheduleIdenticalWithDifferentNonce_succeeds() external {
        // Schedule a transaction with a specific nonce
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(safeInstance);
        dummyTx.scheduleTransaction(timelockGuard);

        // Schedule an identical transaction with a different nonce (salt)
        TransactionBuilder.Transaction memory newTx = dummyTx.deepCopy();
        newTx.nonce = dummyTx.nonce + 1;
        newTx.updateTransaction();

        vm.expectEmit(true, true, true, true);
        emit TransactionScheduled(safe, newTx.hash, INIT_TIME + TIMELOCK_DELAY);
        timelockGuard.scheduleTransaction(safeInstance.safe, newTx.nonce, newTx.params, newTx.signatures);
    }
}

/// @title TimelockGuard_ScheduledTransaction_Test
/// @notice Tests for scheduledTransaction function
contract TimelockGuard_ScheduledTransaction_Test is TimelockGuard_TestInit {
    /// @notice Configures the guard before each scheduleTransaction test.
    function setUp() public override {
        super.setUp();
        _configureGuard(safeInstance, TIMELOCK_DELAY);
    }

    function test_scheduledTransaction_succeeds() external {
        // schedule a transaction
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(safeInstance);
        dummyTx.scheduleTransaction(timelockGuard);

        TimelockGuard.ScheduledTransaction memory scheduledTransaction =
            timelockGuard.scheduledTransaction(safe, dummyTx.hash);
        assertEq(scheduledTransaction.executionTime, INIT_TIME + TIMELOCK_DELAY);
        assert(scheduledTransaction.state == TimelockGuard.TransactionState.Pending);
        assertEq(keccak256(abi.encode(scheduledTransaction.params)), keccak256(abi.encode(dummyTx.params)));
    }
}

/// @title TimelockGuard_PendingTransactions_Test
/// @notice Tests for pendingTransactions function
contract TimelockGuard_PendingTransactions_Test is TimelockGuard_TestInit {
    function setUp() public override {
        super.setUp();
        _configureGuard(safeInstance, TIMELOCK_DELAY);
    }

    function test_pendingTransactions_succeeds() external {
        // schedule a transaction
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(safeInstance);
        dummyTx.scheduleTransaction(timelockGuard);

        TimelockGuard.ScheduledTransaction[] memory pendingTransactions = timelockGuard.pendingTransactions(safe);
        assertEq(pendingTransactions.length, 1);
        // ensure the hash of the transaction params are the same
        assertEq(pendingTransactions[0].params.to, dummyTx.params.to);
        assertEq(keccak256(abi.encode(pendingTransactions[0].params)), keccak256(abi.encode(dummyTx.params)));
    }

    function test_pendingTransactions_removeTransactionAfterCancellation_succeeds() external {
        // schedule a transaction
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(safeInstance);
        dummyTx.scheduleTransaction(timelockGuard);

        // cancel the transaction
        TransactionBuilder.Transaction memory cancellationTx = dummyTx.makeCancellationTransaction(timelockGuard);
        timelockGuard.cancelTransaction(safeInstance.safe, dummyTx.hash, dummyTx.nonce, cancellationTx.signatures);

        // get the pending transactions
        TimelockGuard.ScheduledTransaction[] memory pendingTransactions = timelockGuard.pendingTransactions(safe);
        assertEq(pendingTransactions.length, 0);
    }

    function test_pendingTransactions_removeTransactionAfterExecution_succeeds() external {
        // schedule a transaction
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(safeInstance);
        dummyTx.scheduleTransaction(timelockGuard);

        vm.warp(block.timestamp + TIMELOCK_DELAY);

        // execute the transaction
        dummyTx.executeTransaction(safeInstance.owners[0]);

        // get the pending transactions
        TimelockGuard.ScheduledTransaction[] memory pendingTransactions = timelockGuard.pendingTransactions(safe);
        assertEq(pendingTransactions.length, 0);
    }
}

/// @title TimelockGuard_signCancellation_Test
/// @notice Tests for signCancellation function
contract TimelockGuard_signCancellation_Test is TimelockGuard_TestInit {
    function test_signCancellation_succeeds() external {
        vm.expectEmit(true, true, true, true);
        emit Message("This function is not meant to be called, did you mean to call cancelTransaction?");
        timelockGuard.signCancellation(bytes32(0));
    }
}

contract TimelockGuard_CancelTransaction_Test is TimelockGuard_TestInit {
    /// @notice Prepares a configured guard before cancellation tests run.
    function setUp() public override {
        super.setUp();

        // Configure the guard and schedule a transaction
        _configureGuard(safeInstance, TIMELOCK_DELAY);
    }

    /// @notice Ensures cancellations succeed using owner signatures.
    function test_cancelTransaction_withPrivKeySignature_succeeds() external {
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(safeInstance);
        dummyTx.scheduleTransaction(timelockGuard);

        // Get the cancellation transaction
        TransactionBuilder.Transaction memory cancellationTx = dummyTx.makeCancellationTransaction(timelockGuard);
        uint256 cancellationThreshold = timelockGuard.cancellationThreshold(dummyTx.safeInstance.safe);

        // Cancel the transaction
        vm.expectEmit(true, true, true, true);
        emit CancellationThresholdUpdated(safeInstance.safe, cancellationThreshold, cancellationThreshold + 1);
        vm.expectEmit(true, true, true, true);
        emit TransactionCancelled(safeInstance.safe, dummyTx.hash);
        timelockGuard.cancelTransaction(safeInstance.safe, dummyTx.hash, dummyTx.nonce, cancellationTx.signatures);

        assert(
            timelockGuard.scheduledTransaction(safeInstance.safe, dummyTx.hash).state
                == TimelockGuard.TransactionState.Cancelled
        );
    }

    /// @notice Confirms pre-approved hashes can authorise cancellations.
    function test_cancelTransaction_withApproveHash_succeeds() external {
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(safeInstance);
        dummyTx.scheduleTransaction(timelockGuard);

        // Get the cancellation transaction hash
        TransactionBuilder.Transaction memory cancellationTx = dummyTx.makeCancellationTransaction(timelockGuard);

        // Get the owner
        address owner = dummyTx.safeInstance.safe.getOwners()[0];

        // Approve the cancellation transaction hash
        vm.prank(owner);
        safeInstance.safe.approveHash(cancellationTx.hash);

        // Encode the prevalidated cancellation signature
        bytes memory cancellationSignatures = abi.encodePacked(bytes32(uint256(uint160(owner))), bytes32(0), uint8(1));

        // Get the cancellation threshold
        uint256 cancellationThreshold = timelockGuard.cancellationThreshold(dummyTx.safeInstance.safe);

        // Cancel the transaction
        vm.expectEmit(true, true, true, true);
        emit CancellationThresholdUpdated(dummyTx.safeInstance.safe, cancellationThreshold, cancellationThreshold + 1);
        vm.expectEmit(true, true, true, true);
        emit TransactionCancelled(dummyTx.safeInstance.safe, dummyTx.hash);
        timelockGuard.cancelTransaction(dummyTx.safeInstance.safe, dummyTx.hash, dummyTx.nonce, cancellationSignatures);

        // Confirm that the transaction is cancelled
        TimelockGuard.ScheduledTransaction memory scheduledTransaction =
            timelockGuard.scheduledTransaction(dummyTx.safeInstance.safe, dummyTx.hash);
        assert(scheduledTransaction.state == TimelockGuard.TransactionState.Cancelled);
    }

    /// @notice Verifies cancelling an unscheduled transaction reverts.
    function test_cancelTransaction_revertsIfTransactionNotScheduled_reverts() external {
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(safeInstance);
        TransactionBuilder.Transaction memory cancellationTx = dummyTx.makeCancellationTransaction(timelockGuard);

        // Attempt to cancel the transaction
        vm.expectRevert(TimelockGuard.TimelockGuard_TransactionNotScheduled.selector);
        timelockGuard.cancelTransaction(safeInstance.safe, dummyTx.hash, dummyTx.nonce, cancellationTx.signatures);
    }
}

/// @title TimelockGuard_CheckTransaction_Test
/// @notice Tests for checkTransaction function
contract TimelockGuard_CheckTransaction_Test is TimelockGuard_TestInit {
    /// @notice Establishes the configured guard before checkTransaction tests.
    function setUp() public override {
        super.setUp();
        _configureGuard(safeInstance, TIMELOCK_DELAY);
    }

    /// @notice Test that checkTransaction updates state for successful transactions
    function test_checkTransaction_successfulTransaction_succeeds() external {
        // Schedule a transaction
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(safeInstance);
        dummyTx.scheduleTransaction(timelockGuard);

        // Advance time past timelock delay
        uint256 expectedExecutionTime = block.timestamp + TIMELOCK_DELAY;
        vm.warp(expectedExecutionTime);

        // Increment the Safe nonce to mimic pre-exec state (Safe increments before calling guard)
        vm.store(address(safeInstance.safe), bytes32(uint256(5)), bytes32(uint256(safeInstance.safe.nonce() + 1)));

        // Bump initial cancellation threshold to 2 to validate reset behavior
        _setCancellationThreshold(safeInstance.safe, 2);
        assertEq(timelockGuard.cancellationThreshold(safeInstance.safe), 2);

        // Expect TransactionExecuted event when the guard confirms execution in checkTransaction
        vm.expectEmit(true, true, true, true);
        emit TransactionExecuted(safeInstance.safe, dummyTx.hash);

        // Call checkTransaction as if from the Safe, with an owner as msgSender
        vm.prank(address(safeInstance.safe));
        timelockGuard.checkTransaction(
            dummyTx.params.to,
            dummyTx.params.value,
            dummyTx.params.data,
            dummyTx.params.operation,
            dummyTx.params.safeTxGas,
            dummyTx.params.baseGas,
            dummyTx.params.gasPrice,
            dummyTx.params.gasToken,
            dummyTx.params.refundReceiver,
            "",
            safeInstance.owners[0]
        );

        // State should reflect execution
        TimelockGuard.ScheduledTransaction memory scheduledTx =
            timelockGuard.scheduledTransaction(safeInstance.safe, dummyTx.hash);
        assertEq(uint256(scheduledTx.state), uint256(TimelockGuard.TransactionState.Executed));
        TimelockGuard.ScheduledTransaction[] memory pending = timelockGuard.pendingTransactions(safe);
        assertEq(pending.length, 0);

        // Cancellation threshold should be reset to 1
        assertEq(timelockGuard.cancellationThreshold(safeInstance.safe), 1);
    }

    /// @notice Test that checkTransaction treats failed transactions the same as successful ones
    function test_checkTransaction_failedTransaction_succeeds() external {
        // Build a transaction that will revert (call a contract that always reverts)
        TransactionBuilder.Transaction memory dummyTx = _createEmptyTransaction(safeInstance);
        Reverter reverter = new Reverter();
        dummyTx.params.to = address(reverter);
        // empty data triggers fallback, which reverts
        dummyTx.updateTransaction();
        dummyTx.scheduleTransaction(timelockGuard);

        // Advance time past timelock delay
        uint256 expectedExecutionTime = block.timestamp + TIMELOCK_DELAY;
        vm.warp(expectedExecutionTime);

        // Increment the Safe nonce to mimic pre-exec state (Safe increments before calling guard)
        vm.store(address(safeInstance.safe), bytes32(uint256(5)), bytes32(uint256(safeInstance.safe.nonce() + 1)));

        // Bump initial cancellation threshold to 2 to validate reset behavior
        _setCancellationThreshold(safeInstance.safe, 2);
        assertEq(timelockGuard.cancellationThreshold(safeInstance.safe), 2);

        // Expect TransactionExecuted event when the guard confirms execution in checkTransaction
        vm.expectEmit(true, true, true, true);
        emit TransactionExecuted(safeInstance.safe, dummyTx.hash);

        // Call checkTransaction as if from the Safe, with an owner as msgSender
        vm.prank(address(safeInstance.safe));
        timelockGuard.checkTransaction(
            dummyTx.params.to,
            dummyTx.params.value,
            dummyTx.params.data,
            dummyTx.params.operation,
            dummyTx.params.safeTxGas,
            dummyTx.params.baseGas,
            dummyTx.params.gasPrice,
            dummyTx.params.gasToken,
            dummyTx.params.refundReceiver,
            "",
            safeInstance.owners[0]
        );

        // State should reflect execution, even if the transaction failed
        TimelockGuard.ScheduledTransaction memory scheduledTx =
            timelockGuard.scheduledTransaction(safeInstance.safe, dummyTx.hash);
        assertEq(uint256(scheduledTx.state), uint256(TimelockGuard.TransactionState.Executed));
        TimelockGuard.ScheduledTransaction[] memory pending = timelockGuard.pendingTransactions(safe);
        assertEq(pending.length, 0);

        // Cancellation threshold should be reset to 1
        assertEq(timelockGuard.cancellationThreshold(safeInstance.safe), 1);
    }

    /// @notice Ensures checkTransaction returns early and does not revert when guard is enabled but unconfigured
    function test_checkTransaction_unconfiguredGuard_succeeds() external {
        // Enable guard on an otherwise unconfigured Safe
        _enableGuard(unguardedSafe);
        assertEq(timelockGuard.timelockDelay(unguardedSafe.safe), 0);

        // Build a dummy tx for the unconfigured Safe
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(unguardedSafe);

        // Should not revert: guard returns early when timelock delay is zero
        vm.prank(address(unguardedSafe.safe));
        timelockGuard.checkTransaction(
            dummyTx.params.to,
            dummyTx.params.value,
            dummyTx.params.data,
            dummyTx.params.operation,
            dummyTx.params.safeTxGas,
            dummyTx.params.baseGas,
            dummyTx.params.gasPrice,
            dummyTx.params.gasToken,
            dummyTx.params.refundReceiver,
            "",
            unguardedSafe.owners[0]
        );
    }

    /// @notice Test that checkTransaction reverts when scheduled transaction delay hasn't passed
    function test_checkTransaction_scheduledTransactionNotReady_reverts() external {
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(safeInstance);

        // Schedule the transaction but do not advance time past the timelock delay
        dummyTx.scheduleTransaction(timelockGuard);

        // Increment the nonce, as would normally happen when the transaction is executed
        vm.store(address(safeInstance.safe), bytes32(uint256(5)), bytes32(uint256(safeInstance.safe.nonce() + 1)));

        vm.expectRevert(TimelockGuard.TimelockGuard_TransactionNotReady.selector);
        vm.prank(address(safeInstance.safe));
        timelockGuard.checkTransaction(
            dummyTx.params.to,
            dummyTx.params.value,
            dummyTx.params.data,
            dummyTx.params.operation,
            dummyTx.params.safeTxGas,
            dummyTx.params.baseGas,
            dummyTx.params.gasPrice,
            dummyTx.params.gasToken,
            dummyTx.params.refundReceiver,
            "",
            safeInstance.owners[0]
        );
    }

    /// @notice Test that checkTransaction reverts when scheduled transaction was cancelled
    function test_checkTransaction_scheduledTransactionCancelled_reverts() external {
        // Schedule a transaction
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(safeInstance);
        dummyTx.scheduleTransaction(timelockGuard);

        // Cancel the transaction
        TransactionBuilder.Transaction memory cancellationTx = dummyTx.makeCancellationTransaction(timelockGuard);
        timelockGuard.cancelTransaction(safeInstance.safe, dummyTx.hash, dummyTx.nonce, cancellationTx.signatures);

        // Fast forward past the timelock delay
        vm.warp(block.timestamp + TIMELOCK_DELAY);
        // Increment the nonce, as would normally happen when the transaction is executed
        vm.store(address(safeInstance.safe), bytes32(uint256(5)), bytes32(uint256(safeInstance.safe.nonce() + 1)));

        // Should revert because transaction was cancelled
        vm.expectRevert(TimelockGuard.TimelockGuard_TransactionAlreadyCancelled.selector);
        vm.prank(address(safeInstance.safe));
        timelockGuard.checkTransaction(
            dummyTx.params.to,
            dummyTx.params.value,
            dummyTx.params.data,
            dummyTx.params.operation,
            dummyTx.params.safeTxGas,
            dummyTx.params.baseGas,
            dummyTx.params.gasPrice,
            dummyTx.params.gasToken,
            dummyTx.params.refundReceiver,
            "",
            safeInstance.owners[0]
        );
    }

    /// @notice Test that checkTransaction reverts when a transaction has not been scheduled
    function test_checkTransaction_transactionNotScheduled_reverts() external {
        // Get transaction parameters but don't schedule the transaction
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(safeInstance);

        // Should revert because transaction was not scheduled
        vm.expectRevert(TimelockGuard.TimelockGuard_TransactionNotScheduled.selector);
        vm.prank(address(safeInstance.safe));
        timelockGuard.checkTransaction(
            dummyTx.params.to,
            dummyTx.params.value,
            dummyTx.params.data,
            dummyTx.params.operation,
            dummyTx.params.safeTxGas,
            dummyTx.params.baseGas,
            dummyTx.params.gasPrice,
            dummyTx.params.gasToken,
            dummyTx.params.refundReceiver,
            "",
            safeInstance.owners[0]
        );
    }

    /// @notice Test that checkTransaction reverts when the caller is not an owner
    function testFuzz_checkTransaction_notOwner_reverts(address nonOwner) external {
        vm.assume(!safeInstance.safe.isOwner(nonOwner));
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(safeInstance);
        dummyTx.scheduleTransaction(timelockGuard);

        vm.expectRevert(TimelockGuard.TimelockGuard_NotOwner.selector);
        dummyTx.executeTransaction(nonOwner);
    }
}

/// @title TimelockGuard_MaxCancellationThreshold_Test
/// @notice Tests for the maxCancellationThreshold function in TimelockGuard
contract TimelockGuard_MaxCancellationThreshold_Test is TimelockGuard_TestInit {
    function setUp() public override {
        super.setUp();
        _configureGuard(safeInstance, TIMELOCK_DELAY);
    }

    /// @notice Test that maxCancellationThreshold returns the correct value
    function test_maxCancellationThreshold_maxThresholdIsBlockingThreshold_succeeds() external {
        // create a new Safe with 7 owners and quorum of 5 (blocking threshold is 3)
        SafeInstance memory newSafeInstance = _deploySafe("owners", 7, 5);
        _enableGuard(newSafeInstance);
        _configureGuard(newSafeInstance, TIMELOCK_DELAY);

        // Set up a dummy transaction
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(newSafeInstance);
        dummyTx.scheduleTransaction(timelockGuard);

        // Calculate expected max cancellation threshold
        uint256 blockingThreshold = newSafeInstance.safe.getOwners().length - newSafeInstance.safe.getThreshold() + 1;
        uint256 quorum = newSafeInstance.safe.getThreshold();

        // Ensure that the minimum is set by the blocking threshold
        assertGt(quorum, blockingThreshold);

        // Assert that the maxCancellationThreshold function returns the expected value
        assertEq(timelockGuard.maxCancellationThreshold(newSafeInstance.safe), blockingThreshold);
    }

    /// @notice Test that maxCancellationThreshold returns the correct value
    function test_maxCancellationThreshold_maxThresholdIsQuorum_succeeds() external {
        // create a new Safe with 7 owners and quorum of 3 (blocking threshold is 5)
        SafeInstance memory newSafeInstance = _deploySafe("owners", 7, 3);
        _enableGuard(newSafeInstance);
        _configureGuard(newSafeInstance, TIMELOCK_DELAY);

        // Set up a dummy transaction
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(newSafeInstance);
        dummyTx.scheduleTransaction(timelockGuard);

        // Calculate expected max cancellation threshold
        uint256 blockingThreshold = newSafeInstance.safe.getOwners().length - newSafeInstance.safe.getThreshold() + 1;
        uint256 quorum = newSafeInstance.safe.getThreshold();

        // Ensure that the minimum is set by quorum
        assertGt(blockingThreshold, quorum);

        // Assert that the maxCancellationThreshold function returns the expected value
        assertEq(timelockGuard.maxCancellationThreshold(newSafeInstance.safe), quorum);
    }
}

/// @title TimelockGuard_Integration_Test
/// @notice Tests for integration between TimelockGuard and Safe
contract TimelockGuard_Integration_Test is TimelockGuard_TestInit {
    using stdStorage for StdStorage;

    function setUp() public override {
        super.setUp();
        _configureGuard(safeInstance, TIMELOCK_DELAY);
    }

    /// @notice Test that scheduling a transaction and then executing it succeeds
    function test_integration_scheduleThenExecute_succeeds() external {
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(safeInstance);
        dummyTx.scheduleTransaction(timelockGuard);

        vm.warp(block.timestamp + TIMELOCK_DELAY);

        // increment the cancellation threshold so that we can test that it is reset
        uint256 slot = stdstore.target(address(timelockGuard)).sig("cancellationThreshold(address)").with_key(
            address(safeInstance.safe)
        ).find();
        vm.store(
            address(timelockGuard),
            bytes32(slot),
            bytes32(uint256(timelockGuard.cancellationThreshold(safeInstance.safe) + 1))
        );

        vm.expectEmit(true, true, true, true);
        emit TransactionExecuted(safeInstance.safe, dummyTx.hash);
        dummyTx.executeTransaction(safeInstance.owners[0]);

        // Confirm that the transaction is executed
        TimelockGuard.ScheduledTransaction memory scheduledTransaction =
            timelockGuard.scheduledTransaction(safeInstance.safe, dummyTx.hash);
        assert(scheduledTransaction.state == TimelockGuard.TransactionState.Executed);

        // Confirm that the cancellation threshold is reset
        assertEq(timelockGuard.cancellationThreshold(safeInstance.safe), 1);
    }

    /// @notice Test that scheduling a transaction and then executing it twice reverts
    function test_integration_scheduleThenExecuteTwice_reverts() external {
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(safeInstance);
        dummyTx.scheduleTransaction(timelockGuard);

        vm.warp(block.timestamp + TIMELOCK_DELAY);
        dummyTx.executeTransaction(safeInstance.owners[0]);

        vm.expectRevert("GS026");
        dummyTx.executeTransaction(safeInstance.owners[0]);
    }

    function test_integration_scheduleThenExecuteThenCancel_reverts() external {
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(safeInstance);
        dummyTx.scheduleTransaction(timelockGuard);

        vm.warp(block.timestamp + TIMELOCK_DELAY);
        dummyTx.executeTransaction(safeInstance.owners[0]);

        TransactionBuilder.Transaction memory cancellationTx = dummyTx.makeCancellationTransaction(timelockGuard);
        vm.expectRevert(TimelockGuard.TimelockGuard_TransactionAlreadyExecuted.selector);
        timelockGuard.cancelTransaction(safeInstance.safe, dummyTx.hash, dummyTx.nonce, cancellationTx.signatures);
    }

    /// @notice Test that rescheduling an identical previously cancelled transaction reverts
    function test_integration_scheduleTransactionIdenticalToPreviouslyCancelled_reverts() external {
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(safeInstance);
        dummyTx.scheduleTransaction(timelockGuard);

        TransactionBuilder.Transaction memory cancellationTx = dummyTx.makeCancellationTransaction(timelockGuard);
        timelockGuard.cancelTransaction(safeInstance.safe, dummyTx.hash, dummyTx.nonce, cancellationTx.signatures);

        vm.expectRevert(TimelockGuard.TimelockGuard_TransactionAlreadyScheduled.selector);
        dummyTx.scheduleTransaction(timelockGuard);
    }

    /// @notice Test that the guard can be reset while still enabled, and then can be disabled
    function test_integration_resetThenDisableGuard_succeeds() external {
        TransactionBuilder.Transaction memory resetGuardTx = _createEmptyTransaction(safeInstance);
        resetGuardTx.params.to = address(timelockGuard);
        resetGuardTx.params.data = abi.encodeCall(TimelockGuard.configureTimelockGuard, (0));
        resetGuardTx.updateTransaction();
        resetGuardTx.scheduleTransaction(timelockGuard);

        vm.warp(block.timestamp + TIMELOCK_DELAY);
        resetGuardTx.executeTransaction(safeInstance.owners[0]);

        TransactionBuilder.Transaction memory disableGuardTx = _createEmptyTransaction(safeInstance);
        disableGuardTx.params.to = address(safeInstance.safe);
        disableGuardTx.params.data = abi.encodeCall(GuardManager.setGuard, (address(0)));
        disableGuardTx.updateTransaction();

        vm.warp(block.timestamp + TIMELOCK_DELAY);
        disableGuardTx.executeTransaction(safeInstance.owners[0]);
    }

    /// @notice Test that the max cancellation threshold is not exceeded
    function test_integration_maxCancellationThresholdNotExceeded_succeeds() external {
        uint256 maxThreshold = timelockGuard.maxCancellationThreshold(safeInstance.safe);

        // Schedule a transaction
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(safeInstance);

        // schedule and cancel the transaction maxThreshold + 1 times
        for (uint256 i = 0; i < maxThreshold + 1; i++) {
            // modify the calldata slightly to make the txHash different
            dummyTx.params.data = bytes.concat(dummyTx.params.data, abi.encodePacked(i));
            dummyTx.updateTransaction();
            dummyTx.scheduleTransaction(timelockGuard);

            // Cancel the transaction
            TransactionBuilder.Transaction memory cancellationTx = dummyTx.makeCancellationTransaction(timelockGuard);
            timelockGuard.cancelTransaction(safeInstance.safe, dummyTx.hash, dummyTx.nonce, cancellationTx.signatures);
        }

        assertEq(timelockGuard.cancellationThreshold(safeInstance.safe), maxThreshold);
    }
}

/// @title TimelockGuard_ClearTimelockGuard_Test
/// @notice Tests for clearTimelockGuard function
contract TimelockGuard_ClearTimelockGuard_Test is TimelockGuard_TestInit {
    /// @notice Verifies that clearTimelockGuard successfully clears configuration after guard is disabled
    function test_clearTimelockGuard_succeeds() external {
        // First configure the guard
        _configureGuard(safeInstance, TIMELOCK_DELAY);

        // Schedule a transaction to create pending state
        TransactionBuilder.Transaction memory dummyTx = _createDummyTransaction(safeInstance);
        dummyTx.scheduleTransaction(timelockGuard);

        // Verify transaction is pending
        TimelockGuard.ScheduledTransaction memory scheduledTx = timelockGuard.scheduledTransaction(safe, dummyTx.hash);
        assertEq(uint256(scheduledTx.state), uint256(TimelockGuard.TransactionState.Pending));

        _disableGuard(safeInstance);

        // Clear the guard configuration
        SafeTestLib.execTransaction(
            safeInstance, address(timelockGuard), 0, abi.encodeCall(TimelockGuard.clearTimelockGuard, ())
        );

        // Verify configuration is cleared
        assertEq(timelockGuard.timelockDelay(safe), 0);
        assertEq(timelockGuard.cancellationThreshold(safe), 0);

        // Verify pending transaction doesn't exist anymore
        scheduledTx = timelockGuard.scheduledTransaction(safe, dummyTx.hash);
        assertEq(uint256(scheduledTx.state), uint256(TimelockGuard.TransactionState.NotScheduled));
    }

    /// @notice Verifies that clearTimelockGuard reverts when guard is still enabled
    function test_clearTimelockGuard_guardStillEnabled_reverts() external {
        // First configure the guard
        _configureGuard(safeInstance, TIMELOCK_DELAY);

        // Try to clear while guard is still enabled (should revert)
        vm.expectRevert(TimelockGuard.TimelockGuard_GuardStillEnabled.selector);
        vm.prank(address(safeInstance.safe));
        timelockGuard.clearTimelockGuard();
    }
}

/// @title TimelockGuard_SupportsInterface_Test
/// @notice Tests ERC165 interface support for TimelockGuard
contract TimelockGuard_SupportsInterface_Test is TimelockGuard_TestInit {
    function test_supportsInterface_iTransactionGuard_succeeds() external view {
        bytes4 interfaceId = 0xe6d7a83a; // ITransactionGuard interface ID
        assertTrue(timelockGuard.supportsInterface(interfaceId), "Should support ITransactionGuard");
    }

    function test_supportsInterface_ierc165_succeeds() external view {
        bytes4 interfaceId = 0x01ffc9a7; // IERC165 interface ID
        assertTrue(timelockGuard.supportsInterface(interfaceId), "Should support IERC165");
    }

    function test_supportsInterface_invalidInterface_fails(bytes4 _interfaceId) external view {
        vm.assume(_interfaceId != type(ITransactionGuard).interfaceId);
        vm.assume(_interfaceId != type(IERC165).interfaceId);
        assertFalse(timelockGuard.supportsInterface(_interfaceId), "Should not support invalid interface");
    }
}

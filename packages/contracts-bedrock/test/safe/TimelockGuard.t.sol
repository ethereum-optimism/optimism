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
    event GuardConfigured(address indexed safe, uint256 timelockDelay);
    event GuardCleared(address indexed safe);
    event TransactionScheduled(Safe indexed safe, bytes32 indexed txId, uint256 when);

    uint256 constant INIT_TIME = 10;
    uint256 constant TIMELOCK_DELAY = 7 days;
    uint256 constant NUM_OWNERS = 5;
    uint256 constant THRESHOLD = 3;
    uint256 constant ONE_YEAR = 365 days;

    TimelockGuard timelockGuard;
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

    /// @notice Helper to create a dummy transaction with signatures and a tx hash
    // TODO: separate into two functions: one for the params+hash, one for the signatures
    function _getDummyTx() internal view returns (ExecTransactionParams memory, bytes32) {
        // Get the nonce of the safe to sign
        uint256 nonce = safeInstance.safe.nonce();

        // Declare the dummy transaction params with an empty signature
        ExecTransactionParams memory dummyTxParams = ExecTransactionParams({
            to: address(0xabba),
            value: 0,
            data: hex"acdc",
            operation: Enum.Operation.Call,
            safeTxGas: 0,
            baseGas: 0,
            gasPrice: 0,
            gasToken: address(0),
            refundReceiver: payable(address(0)),
            signatures: new bytes(0)
        });

        // Get the tx hash
        bytes32 txHash;
            {
                txHash = safeInstance.safe.getTransactionHash({
                    to: dummyTxParams.to,
                    value: dummyTxParams.value,
                    data: dummyTxParams.data,
                    operation: dummyTxParams.operation,
                    safeTxGas: dummyTxParams.safeTxGas,
                    baseGas: dummyTxParams.baseGas,
                    gasPrice: dummyTxParams.gasPrice,
                    gasToken: dummyTxParams.gasToken,
                    refundReceiver: dummyTxParams.refundReceiver,
                    _nonce: nonce
                });
            }

        // Sign the tx hash with the owners' private keys
        for (uint256 i; i < THRESHOLD; ++i) {
            (uint8 v, bytes32 r, bytes32 s) = vm.sign(safeInstance.ownerPKs[i], txHash);

            // The signature format is a compact form of: {bytes32 r}{bytes32 s}{uint8 v}
            dummyTxParams.signatures = bytes.concat(dummyTxParams.signatures, abi.encodePacked(r, s, v));
        }

        return (dummyTxParams, txHash);
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
        uint256 delay = timelockGuard.viewTimelockGuardConfiguration(address(safeInstance.safe));
        assertEq(delay, 0);
    }
}

/// @title TimelockGuard_ConfigureTimelockGuard_Test
/// @notice Tests for configureTimelockGuard function
contract TimelockGuard_ConfigureTimelockGuard_Test is TimelockGuard_TestInit {
    function test_configureTimelockGuard_succeeds() external {
        vm.expectEmit(true, true, true, true);
        emit GuardConfigured(address(safeInstance.safe), TIMELOCK_DELAY);

        _configureGuard(safeInstance, TIMELOCK_DELAY);

        uint256 storedDelay = timelockGuard.viewTimelockGuardConfiguration(address(safeInstance.safe));
        assertEq(storedDelay, TIMELOCK_DELAY);
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
        emit GuardConfigured(address(safeInstance.safe), ONE_YEAR);

        _configureGuard(safeInstance, ONE_YEAR);

        uint256 storedDelay = timelockGuard.viewTimelockGuardConfiguration(address(safeInstance.safe));
        assertEq(storedDelay, ONE_YEAR);
    }

    function test_configureTimelockGuard_allowsReconfiguration_succeeds() external {
        // Initial configuration
        _configureGuard(safeInstance, TIMELOCK_DELAY);
        assertEq(timelockGuard.viewTimelockGuardConfiguration(address(safeInstance.safe)), TIMELOCK_DELAY);

        // Reconfigure with different delay
        uint256 newDelay = 14 days;
        vm.expectEmit(true, true, true, true);
        emit GuardConfigured(address(safeInstance.safe), newDelay);

        _configureGuard(safeInstance, newDelay);
        assertEq(timelockGuard.viewTimelockGuardConfiguration(address(safeInstance.safe)), newDelay);
    }
}

/// @title TimelockGuard_ClearTimelockGuard_Test
/// @notice Tests for clearTimelockGuard function
contract TimelockGuard_ClearTimelockGuard_Test is TimelockGuard_TestInit {
    function test_clearTimelockGuard_succeeds() external {
        // First configure the guard
        _configureGuard(safeInstance, TIMELOCK_DELAY);
        assertEq(timelockGuard.viewTimelockGuardConfiguration(address(safeInstance.safe)), TIMELOCK_DELAY);

        // Disable the guard first
        _disableGuard(safeInstance);

        // Clear should succeed and emit event
        vm.expectEmit(true, true, true, true);
        emit GuardCleared(address(safeInstance.safe));

        _clearGuard(safeInstance);

        // Configuration should be cleared
        assertEq(timelockGuard.viewTimelockGuardConfiguration(address(safeInstance.safe)), 0);
        // Ensure cancellation threshold is reset to 0
        assertEq(timelockGuard.cancellationThreshold(address(safeInstance.safe)), 0);

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
    function test_cancellationThreshold_returnsZeroIfGuardNotEnabled_succeeds() external {
        uint256 threshold = timelockGuard.cancellationThreshold(address(unguardedSafe.safe));
        assertEq(threshold, 0);
    }

    function test_cancellationThreshold_returnsZeroIfGuardNotConfigured_succeeds() external view {
        // Safe with guard enabled but not configured should return 0
        uint256 threshold = timelockGuard.cancellationThreshold(address(safeInstance.safe));
        assertEq(threshold, 0);
    }

    function test_cancellationThreshold_returnsOneAfterConfiguration_succeeds() external {
        // Configure the guard
        _configureGuard(safeInstance, TIMELOCK_DELAY);

        // Should default to 1 after configuration
        uint256 threshold = timelockGuard.cancellationThreshold(address(safeInstance.safe));
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
        (ExecTransactionParams memory dummyTxParams, bytes32 txHash) = _getDummyTx();

        vm.expectEmit(true, true, true, true);
        emit TransactionScheduled(safeInstance.safe, txHash, INIT_TIME + TIMELOCK_DELAY);
        timelockGuard.scheduleTransaction(safeInstance.safe, safeInstance.safe.nonce(), dummyTxParams);
    }

    function test_scheduleTransaction_reschedulingIdenticalTransaction_reverts() external {
        (ExecTransactionParams memory dummyTxParams,) = _getDummyTx();
        timelockGuard.scheduleTransaction(safeInstance.safe, safeInstance.safe.nonce(), dummyTxParams);

        vm.expectRevert(TimelockGuard.TimelockGuard_TransactionAlreadyScheduled.selector);
        timelockGuard.scheduleTransaction(safeInstance.safe, safeInstance.safe.nonce(), dummyTxParams);
    }

    function test_scheduleTransaction_identicalPreviouslyCancelled_reverts() external { }

    function test_scheduleTransaction_guardNotEnabled_reverts() external { }

    function test_scheduleTransaction_guardNotConfigured_reverts() external { }

    function test_scheduleTransaction_canScheduleIdenticalWithSalt_succeeds() external { }
}

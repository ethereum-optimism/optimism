// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import { GuardManager } from "safe-contracts/base/GuardManager.sol";
import { StorageAccessible } from "safe-contracts/common/StorageAccessible.sol";
import "test/safe-tools/SafeTestTools.sol";

import { TimelockGuard } from "src/safe/TimelockGuard.sol";

/// @title TimelockGuard_TestInit
/// @notice Reusable test initialization for `TimelockGuard` tests.
contract TimelockGuard_TestInit is Test, SafeTestTools {
    using SafeTestLib for SafeInstance;

    // Events
    event GuardConfigured(address indexed safe, uint256 timelockDelay);
    event GuardCleared(address indexed safe);

    uint256 constant INIT_TIME = 10;
    uint256 constant TIMELOCK_DELAY = 7 days;
    uint256 constant NUM_OWNERS = 5;
    uint256 constant THRESHOLD = 3;
    uint256 constant ONE_YEAR = 365 days;

    TimelockGuard timelockGuard;
    SafeInstance safeInstance;
    SafeInstance safeInstance2;
    address[] owners;
    uint256[] ownerPKs;

    function setUp() public virtual {
        vm.warp(INIT_TIME);

        // Deploy the singleton TimelockGuard
        timelockGuard = new TimelockGuard();

        // Create Safe owners
        (address[] memory _owners, uint256[] memory _keys) = SafeTestLib.makeAddrsAndKeys("owners", NUM_OWNERS);
        owners = _owners;
        ownerPKs = _keys;

        // Set up Safe with owners
        safeInstance = _setupSafe(ownerPKs, THRESHOLD);

        // Enable the guard on the Safe
        SafeTestLib.execTransaction(
            safeInstance,
            address(safeInstance.safe),
            0,
            abi.encodeCall(GuardManager.setGuard, (address(timelockGuard))),
            Enum.Operation.Call
        );
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
            _safe,
            address(_safe.safe),
            0,
            abi.encodeCall(GuardManager.setGuard, (address(0))),
            Enum.Operation.Call
        );
    }

    /// @notice Helper to clear the TimelockGuard configuration for a Safe
    function _clearGuard(SafeInstance memory _safe) internal {
        SafeTestLib.execTransaction(
            _safe,
            address(timelockGuard),
            0,
            abi.encodeCall(TimelockGuard.clearTimelockGuard, ()),
            Enum.Operation.Call
        );
    }
}

/// @title TimelockGuard_ViewTimelockGuardConfiguration_Test
/// @notice Tests for viewTimelockGuardConfiguration function
contract TimelockGuard_ViewTimelockGuardConfiguration_Test is TimelockGuard_TestInit {
    function test_viewTimelockGuardConfiguration_returnsZeroForUnconfiguredSafe() external view {
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

    function test_configureTimelockGuard_revertsIfGuardNotEnabled() external {
        // Create a safe without enabling the guard
        // Reduce the threshold just to prevent a CREATE2 collision when deploying this safe.
        SafeInstance memory unguardedSafe = _setupSafe(ownerPKs, THRESHOLD-1);

        vm.expectRevert(TimelockGuard.TimelockGuard_GuardNotEnabled.selector);
        vm.prank(address(unguardedSafe.safe));
        timelockGuard.configureTimelockGuard(TIMELOCK_DELAY);
    }

    function test_configureTimelockGuard_revertsIfDelayTooLong() external {
        uint256 tooLongDelay = ONE_YEAR + 1;

        vm.expectRevert(TimelockGuard.TimelockGuard_InvalidTimelockDelay.selector);
        vm.prank(address(safeInstance.safe));
        timelockGuard.configureTimelockGuard(tooLongDelay);
    }

    function test_configureTimelockGuard_revertsIfDelayZero() external {
        vm.expectRevert(TimelockGuard.TimelockGuard_InvalidTimelockDelay.selector);
        vm.prank(address(safeInstance.safe));
        timelockGuard.configureTimelockGuard(0);
    }

    function test_configureTimelockGuard_acceptsMaxValidDelay() external {
        vm.expectEmit(true, true, true, true);
        emit GuardConfigured(address(safeInstance.safe), ONE_YEAR);

        _configureGuard(safeInstance, ONE_YEAR);

        uint256 storedDelay = timelockGuard.viewTimelockGuardConfiguration(address(safeInstance.safe));
        assertEq(storedDelay, ONE_YEAR);
    }

    function test_configureTimelockGuard_allowsReconfiguration() external {
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
        // TODO: Check that any active challenge is cancelled
    }

    function test_clearTimelockGuard_revertsIfGuardStillEnabled() external {
        // First configure the guard
        _configureGuard(safeInstance, TIMELOCK_DELAY);

        // Try to clear without disabling guard first - should revert
        vm.expectRevert(TimelockGuard.TimelockGuard_GuardStillEnabled.selector);
        vm.prank(address(safeInstance.safe));
        timelockGuard.clearTimelockGuard();
    }

    function test_clearTimelockGuard_revertsIfNotConfigured() external {
        // Try to clear - should revert because not configured
        vm.expectRevert(TimelockGuard.TimelockGuard_GuardNotConfigured.selector);
        vm.prank(address(safeInstance.safe));
        timelockGuard.clearTimelockGuard();
    }
}

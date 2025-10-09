// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import "test/safe-tools/SafeTestTools.sol";

import { SaferSafes } from "src/safe/SaferSafes.sol";
import { LivenessModule2 } from "src/safe/LivenessModule2.sol";

/// @title SaferSafes_TestInit
/// @notice Reusable test initialization for `SaferSafes` tests.
contract SaferSafes_TestInit is Test, SafeTestTools {
    using SafeTestLib for SafeInstance;

    // Events
    event ModuleConfigured(address indexed safe, uint256 livenessResponsePeriod, address fallbackOwner);
    event GuardConfigured(address indexed safe, uint256 timelockDelay, uint256 cancellationThreshold);

    uint256 constant INIT_TIME = 10;
    uint256 constant NUM_OWNERS = 5;
    uint256 constant THRESHOLD = 3;

    SaferSafes saferSafes;
    SafeInstance safeInstance;
    address fallbackOwner;
    address[] owners;
    uint256[] ownerPKs;

    function setUp() public virtual {
        vm.warp(INIT_TIME);

        // Deploy the SaferSafes contract
        saferSafes = new SaferSafes();

        // Create Safe owners
        (address[] memory _owners, uint256[] memory _keys) = SafeTestLib.makeAddrsAndKeys("owners", NUM_OWNERS);
        owners = _owners;
        ownerPKs = _keys;

        // Set up Safe with owners
        safeInstance = _setupSafe(ownerPKs, THRESHOLD);

        // Set fallback owner
        fallbackOwner = makeAddr("fallbackOwner");

        // Enable the module and guard on the Safe
        safeInstance.enableModule(address(saferSafes));
        safeInstance.setGuard(address(saferSafes));
    }
}

/// @title SaferSafes_Uncategorized_Test
/// @notice Tests for SaferSafes configuration functionality.
contract SaferSafes_Uncategorized_Test is SaferSafes_TestInit {
    /// @notice Test successful configuration when liveness response period is at least 2x timelock delay.
    function test_configure_livenessModuleFirst_succeeds() public {
        uint256 timelockDelay = 7 days;
        uint256 livenessResponsePeriod = 21 days; // Much greater than 2 * 7 days = 14 days (should succeed)

        // Configure the liveness module FIRST
        LivenessModule2.ModuleConfig memory moduleConfig = LivenessModule2.ModuleConfig({
            livenessResponsePeriod: livenessResponsePeriod,
            fallbackOwner: fallbackOwner
        });

        vm.prank(address(safeInstance.safe));
        saferSafes.configureLivenessModule(moduleConfig);

        // Configure the timelock guard SECOND (this will trigger the check)
        vm.prank(address(safeInstance.safe));
        saferSafes.configureTimelockGuard(timelockDelay);

        // Verify configurations were set
        (uint256 storedLivenessResponsePeriod, address storedFallbackOwner) =
            saferSafes.livenessSafeConfiguration(address(safeInstance.safe));
        assertEq(storedLivenessResponsePeriod, livenessResponsePeriod);
        assertEq(storedFallbackOwner, fallbackOwner);
        assertEq(saferSafes.timelockConfiguration(safeInstance.safe), timelockDelay);
    }

    function test_configure_timelockGuardFirst_succeeds() public {
        uint256 timelockDelay = 7 days;
        uint256 livenessResponsePeriod = 21 days; // Much greater than 2 * 7 days = 14 days (should succeed)

        // Configure the timelock guard FIRST
        vm.prank(address(safeInstance.safe));
        saferSafes.configureTimelockGuard(timelockDelay);

        LivenessModule2.ModuleConfig memory moduleConfig = LivenessModule2.ModuleConfig({
            livenessResponsePeriod: livenessResponsePeriod,
            fallbackOwner: fallbackOwner
        });

        // Configure the liveness module SECOND (this will trigger the check)
        vm.prank(address(safeInstance.safe));
        saferSafes.configureLivenessModule(moduleConfig);

        // Verify configurations were set
        (uint256 storedLivenessResponsePeriod, address storedFallbackOwner) =
            saferSafes.livenessSafeConfiguration(address(safeInstance.safe));
        assertEq(storedLivenessResponsePeriod, livenessResponsePeriod);
        assertEq(storedFallbackOwner, fallbackOwner);
        assertEq(saferSafes.timelockConfiguration(safeInstance.safe), timelockDelay);
    }

    /// @notice Test that attempting to incorrectly configure the timelock guard after first configuring the liveness
    /// module fails.
    /// @dev This test would fail if timelock guard configuration also triggered validation
    function test_configure_livenessModuleFirstInvalidConfig_reverts() public {
        uint256 timelockDelay = 7 days;
        uint256 livenessResponsePeriod = 13 days; // This is invalid: 13 < 2*7

        // Configure liveness module first
        LivenessModule2.ModuleConfig memory moduleConfig = LivenessModule2.ModuleConfig({
            livenessResponsePeriod: livenessResponsePeriod,
            fallbackOwner: fallbackOwner
        });

        vm.prank(address(safeInstance.safe));
        saferSafes.configureLivenessModule(moduleConfig);

        // Now configure timelock guard
        vm.prank(address(safeInstance.safe));
        vm.expectRevert(SaferSafes.SaferSafes_InsufficientLivenessResponsePeriod.selector);
        saferSafes.configureTimelockGuard(timelockDelay);
    }

    function test_configure_timelockGuardFirstInvalidConfig_reverts() public {
        uint256 timelockDelay = 7 days;
        uint256 livenessResponsePeriod = 13 days; // This is invalid: 13 < 2*7

        // Configure timelock guard first
        vm.prank(address(safeInstance.safe));
        saferSafes.configureTimelockGuard(timelockDelay);

        LivenessModule2.ModuleConfig memory moduleConfig = LivenessModule2.ModuleConfig({
            livenessResponsePeriod: livenessResponsePeriod,
            fallbackOwner: fallbackOwner
        });

        // Configure liveness module second - this will trigger the check
        vm.expectRevert(SaferSafes.SaferSafes_InsufficientLivenessResponsePeriod.selector);
        vm.prank(address(safeInstance.safe));
        saferSafes.configureLivenessModule(moduleConfig);
    }
}

// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Enum } from "safe-contracts/common/Enum.sol";
import { GnosisSafe as Safe } from "safe-contracts/GnosisSafe.sol";
import "test/safe-tools/SafeTestTools.sol";

import { SaferSafes } from "src/safe/SaferSafes.sol";
import { LivenessModule2 } from "src/safe/LivenessModule2.sol";

import { GuardManager, Guard as IGuard } from "safe-contracts/base/GuardManager.sol";
import { ModuleManager } from "safe-contracts/base/ModuleManager.sol";
import { GnosisSafe as Safe } from "safe-contracts/GnosisSafe.sol";

// Import the test utils from LivenessModule2 tests
import { LivenessModule2_TestUtils } from "test/safe/LivenessModule2.t.sol";

/// @title SaferSafes_TestInit
/// @notice Reusable test initialization for `SaferSafes` tests.
abstract contract SaferSafes_TestInit is LivenessModule2_TestUtils {
    using SafeTestLib for SafeInstance;

    // Events
    event ModuleConfigured(address indexed safe, uint256 livenessResponsePeriod, address fallbackOwner);
    event GuardConfigured(address indexed safe, uint256 timelockDelay, uint256 cancellationThreshold);
    event ChallengeSucceeded(address indexed safe, address fallbackOwner);

    uint256 constant INIT_TIME = 10;
    uint256 constant NUM_OWNERS = 5;
    uint256 constant THRESHOLD = 3;
    uint256 constant CHALLENGE_PERIOD = 7 days;

    SaferSafes saferSafes;
    SafeInstance safeInstance;
    address fallbackOwner;
    address[] owners;
    uint256[] ownerPKs;

    function setUp() public virtual {
        vm.warp(INIT_TIME);

        // Deploy the SaferSafes contract
        saferSafes = new SaferSafes();
        livenessModule2 = LivenessModule2(address(saferSafes));

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
    function test_version_succeeds() external view {
        assertTrue(bytes(saferSafes.version()).length > 0);
    }

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
        LivenessModule2.ModuleConfig memory storedConfig = saferSafes.livenessSafeConfiguration(safeInstance.safe);
        assertEq(storedConfig.livenessResponsePeriod, livenessResponsePeriod);
        assertEq(storedConfig.fallbackOwner, fallbackOwner);
        assertEq(saferSafes.timelockDelay(safeInstance.safe), timelockDelay);
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
        LivenessModule2.ModuleConfig memory storedConfig = saferSafes.livenessSafeConfiguration(safeInstance.safe);
        assertEq(storedConfig.livenessResponsePeriod, livenessResponsePeriod);
        assertEq(storedConfig.fallbackOwner, fallbackOwner);
        assertEq(saferSafes.timelockDelay(safeInstance.safe), timelockDelay);
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

/// @title SaferSafes_ChangeOwnershipToFallback_Test
/// @notice Tests the ownership transfer after successful challenge
contract SaferSafes_ChangeOwnershipToFallback_Test is SaferSafes_TestInit {
    function setUp() public override {
        super.setUp();

        _enableModule(safeInstance, CHALLENGE_PERIOD, fallbackOwner);

        // enable the guard
        SafeTestLib.execTransaction(
            safeInstance,
            address(safeInstance.safe),
            0,
            abi.encodeCall(GuardManager.setGuard, (address(livenessModule2))),
            Enum.Operation.Call
        );
    }

    function _assertOwnershipChanged() internal view {
        // Verify ownership changed
        address[] memory newOwners = safeInstance.safe.getOwners();
        assertEq(newOwners.length, 1);
        assertEq(newOwners[0], fallbackOwner);
        assertEq(safeInstance.safe.getThreshold(), 1);

        // Verify challenge is reset
        uint256 challengeEndTime = livenessModule2.getChallengePeriodEnd(safeInstance.safe);
        assertEq(challengeEndTime, 0);

        // Verify guard is deactivated
        assertEq(_getGuard(safeInstance), address(0));
    }

    function test_changeOwnershipToFallback_succeeds() external {
        // Start a challenge
        vm.prank(fallbackOwner);
        livenessModule2.challenge(safeInstance.safe);

        // Warp past challenge period
        vm.warp(block.timestamp + CHALLENGE_PERIOD + 1);

        // Execute ownership transfer
        vm.expectEmit(true, true, true, true);
        emit ChallengeSucceeded(address(safeInstance.safe), fallbackOwner);

        vm.prank(fallbackOwner);
        livenessModule2.changeOwnershipToFallback(safeInstance.safe);

        _assertOwnershipChanged();
    }

    function test_changeOwnershipToFallback_withOtherGuard_succeeds() external {
        _enableModule(safeInstance, CHALLENGE_PERIOD, fallbackOwner);
        // Create a mock guard
        address dummyGuard = makeAddr("dummyGuard");
        vm.mockCall(
            dummyGuard,
            abi.encodeCall(
                IGuard.checkTransaction,
                (address(0), 0, "", Enum.Operation.Call, 0, 0, 0, address(0), payable(address(0)), "", address(0))
            ),
            ""
        );
        vm.mockCall(dummyGuard, abi.encodeCall(IGuard.checkAfterExecution, (bytes32(0), false)), "");

        // Enable the mock guard on the Safe
        SafeTestLib.execTransaction(
            safeInstance,
            address(safeInstance.safe),
            0,
            abi.encodeCall(GuardManager.setGuard, (dummyGuard)),
            Enum.Operation.Call
        );

        // Start a challenge
        vm.prank(fallbackOwner);
        livenessModule2.challenge(safeInstance.safe);

        // Warp past challenge period
        vm.warp(block.timestamp + CHALLENGE_PERIOD + 1);

        // Execute ownership transfer
        vm.expectEmit(true, true, true, true);
        emit ChallengeSucceeded(address(safeInstance.safe), fallbackOwner);

        vm.prank(fallbackOwner);
        livenessModule2.changeOwnershipToFallback(safeInstance.safe);

        // These checks include ensuring that the guard is deactivated
        _assertOwnershipChanged();
    }

    function test_changeOwnershipToFallback_moduleNotEnabled_reverts() external {
        address newSafe = makeAddr("newSafe");

        vm.prank(fallbackOwner);
        vm.expectRevert(LivenessModule2.LivenessModule2_ModuleNotConfigured.selector);
        livenessModule2.changeOwnershipToFallback(Safe(payable(newSafe)));
    }

    function test_changeOwnershipToFallback_noChallenge_reverts() external {
        vm.prank(fallbackOwner);
        vm.expectRevert(LivenessModule2.LivenessModule2_ChallengeDoesNotExist.selector);
        livenessModule2.changeOwnershipToFallback(safeInstance.safe);
    }

    function test_changeOwnershipToFallback_beforeResponsePeriod_reverts() external {
        // Start a challenge
        vm.prank(fallbackOwner);
        livenessModule2.challenge(safeInstance.safe);

        // Try to execute before response period expires
        vm.prank(fallbackOwner);
        vm.expectRevert(LivenessModule2.LivenessModule2_ResponsePeriodActive.selector);
        livenessModule2.changeOwnershipToFallback(safeInstance.safe);
    }

    function test_changeOwnershipToFallback_moduleDisabledAtSafeLevel_reverts() external {
        // Start a challenge
        vm.prank(fallbackOwner);
        livenessModule2.challenge(safeInstance.safe);

        // Warp past challenge period
        vm.warp(block.timestamp + CHALLENGE_PERIOD + 1);

        // Disable the module at Safe level
        SafeTestLib.execTransaction(
            safeInstance,
            address(safeInstance.safe),
            0,
            abi.encodeCall(ModuleManager.disableModule, (address(0x1), address(livenessModule2))),
            Enum.Operation.Call
        );

        // Try to execute ownership transfer - should revert because module is disabled at Safe level
        vm.prank(fallbackOwner);
        vm.expectRevert(LivenessModule2.LivenessModule2_ModuleNotEnabled.selector);
        livenessModule2.changeOwnershipToFallback(safeInstance.safe);
    }

    function test_changeOwnershipToFallback_onlyFallbackOwner_succeeds() external {
        _enableModule(safeInstance, CHALLENGE_PERIOD, fallbackOwner);
        // Start a challenge
        vm.prank(fallbackOwner);
        livenessModule2.challenge(safeInstance.safe);

        // Warp past challenge period
        vm.warp(block.timestamp + CHALLENGE_PERIOD + 1);

        // Try from random address - should fail
        address randomCaller = makeAddr("randomCaller");
        vm.prank(randomCaller);
        vm.expectRevert(LivenessModule2.LivenessModule2_UnauthorizedCaller.selector);
        livenessModule2.changeOwnershipToFallback(safeInstance.safe);

        // Execute from fallback owner - should succeed
        vm.prank(fallbackOwner);
        livenessModule2.changeOwnershipToFallback(safeInstance.safe);

        // Verify ownership changed
        address[] memory newOwners = safeInstance.safe.getOwners();
        assertEq(newOwners.length, 1);
        assertEq(newOwners[0], fallbackOwner);
    }

    function test_changeOwnershipToFallback_canRechallenge_succeeds() external {
        _enableModule(safeInstance, CHALLENGE_PERIOD, fallbackOwner);
        // Start and execute first challenge
        vm.prank(fallbackOwner);
        livenessModule2.challenge(safeInstance.safe);

        vm.warp(block.timestamp + CHALLENGE_PERIOD + 1);
        vm.prank(fallbackOwner);
        livenessModule2.changeOwnershipToFallback(safeInstance.safe);

        // Re-configure the module
        vm.prank(address(safeInstance.safe));
        livenessModule2.configureLivenessModule(
            LivenessModule2.ModuleConfig({ livenessResponsePeriod: CHALLENGE_PERIOD, fallbackOwner: fallbackOwner })
        );

        // Start a new challenge (as fallback owner)
        vm.prank(fallbackOwner);
        livenessModule2.challenge(safeInstance.safe);

        uint256 challengeEndTime = livenessModule2.getChallengePeriodEnd(safeInstance.safe);
        assertGt(challengeEndTime, 0);
    }
}

// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import { Safe } from "safe-contracts/Safe.sol";
import "test/safe-tools/SafeTestTools.sol";
import { Constants } from "src/libraries/Constants.sol";

import { LivenessModule2 } from "src/safe/LivenessModule2.sol";
import { SaferSafes } from "src/safe/SaferSafes.sol";
import { ModuleManager } from "safe-contracts/base/ModuleManager.sol";
import { GuardManager } from "safe-contracts/base/GuardManager.sol";
import { DummyGuard } from "test/mocks/DummyGuard.sol";

/// @title LivenessModule2_TestUtils
/// @notice Reusable helper methods for LivenessModule2 tests.
abstract contract LivenessModule2_TestUtils is Test, SafeTestTools {
    LivenessModule2 livenessModule2;

    /// @notice Helper to enable the LivenessModule2 for a Safe
    function _enableModule(SafeInstance memory _safe, uint256 _period, address _fallback) internal {
        LivenessModule2.ModuleConfig memory config =
            LivenessModule2.ModuleConfig({ livenessResponsePeriod: _period, fallbackOwner: _fallback });
        SafeTestLib.execTransaction(
            _safe,
            address(livenessModule2),
            0,
            abi.encodeCall(LivenessModule2.configureLivenessModule, (config)),
            Enum.Operation.Call
        );
    }

    /// @notice Helper to disable the LivenessModule2 for a Safe
    function _disableModule(SafeInstance memory _safe) internal {
        // First disable the module at the Safe level
        SafeTestLib.execTransaction(
            _safe,
            address(_safe.safe),
            0,
            abi.encodeCall(ModuleManager.disableModule, (address(0x1), address(livenessModule2))),
            Enum.Operation.Call
        );

        // Then clear the module configuration
        SafeTestLib.execTransaction(
            _safe,
            address(livenessModule2),
            0,
            abi.encodeCall(LivenessModule2.clearLivenessModule, ()),
            Enum.Operation.Call
        );
    }

    /// @notice Helper to respond to a challenge from a Safe
    function _respondToChallenge(SafeInstance memory _safe) internal {
        SafeTestLib.execTransaction(
            _safe, address(livenessModule2), 0, abi.encodeCall(LivenessModule2.respond, ()), Enum.Operation.Call
        );
    }

    /// @notice Helper to get the guard address from a Safe
    function _getGuard(SafeInstance memory _safe) internal view returns (address) {
        address guard = abi.decode(_safe.safe.getStorageAt(uint256(Constants.GUARD_STORAGE_SLOT), 1), (address));
        return guard;
    }
}

/// @title LivenessModule2_TestInit
/// @notice Reusable test initialization for `LivenessModule2` tests.
contract LivenessModule2_TestInit is LivenessModule2_TestUtils {
    using SafeTestLib for SafeInstance;

    // Events
    event ModuleConfigured(address indexed safe, uint256 livenessResponsePeriod, address fallbackOwner);
    event ModuleCleared(address indexed safe);
    event ChallengeStarted(address indexed safe, uint256 challengeStartTime);
    event ChallengeCancelled(address indexed safe);
    event ChallengeSucceeded(address indexed safe, address fallbackOwner);

    uint256 constant INIT_TIME = 10;
    uint256 constant CHALLENGE_PERIOD = 7 days;
    uint256 constant NUM_OWNERS = 5;
    uint256 constant THRESHOLD = 3;

    SafeInstance safeInstance;
    address fallbackOwner;
    address[] owners;
    uint256[] ownerPKs;

    function setUp() public virtual {
        vm.warp(INIT_TIME);

        // Deploy the combined SaferSafes contract which implements LivenessModule2
        livenessModule2 = new SaferSafes();

        // Create Safe owners
        (address[] memory _owners, uint256[] memory _keys) = SafeTestLib.makeAddrsAndKeys("owners", NUM_OWNERS);
        owners = _owners;
        ownerPKs = _keys;

        // Set up Safe with owners
        safeInstance = _setupSafe(ownerPKs, THRESHOLD);

        // Set fallback owner
        fallbackOwner = makeAddr("fallbackOwner");

        // Enable the module on the Safe
        SafeTestLib.enableModule(safeInstance, address(livenessModule2));
    }
}

/// @title LivenessModule2_ConfigureLivenessModule_Test
/// @notice Tests configuring the module
contract LivenessModule2_ConfigureLivenessModule_Test is LivenessModule2_TestInit {
    function test_configureLivenessModule_succeeds() external {
        vm.expectEmit(true, true, true, true);
        emit ModuleConfigured(address(safeInstance.safe), CHALLENGE_PERIOD, fallbackOwner);

        _enableModule(safeInstance, CHALLENGE_PERIOD, fallbackOwner);

        LivenessModule2.ModuleConfig memory config = livenessModule2.livenessSafeConfiguration(safeInstance.safe);
        assertEq(config.livenessResponsePeriod, CHALLENGE_PERIOD);
        assertEq(config.fallbackOwner, fallbackOwner);
        assertEq(livenessModule2.challengeStartTime(safeInstance.safe), 0);
    }

    function test_configureLivenessModule_multipleSafes_succeeds() external {
        // Test that multiple independent safes can configure the module
        (, uint256[] memory keys1) = SafeTestLib.makeAddrsAndKeys("safe1", NUM_OWNERS);
        SafeInstance memory safe1 = _setupSafe(keys1, THRESHOLD);
        SafeTestLib.enableModule(safe1, address(livenessModule2));

        (, uint256[] memory keys2) = SafeTestLib.makeAddrsAndKeys("safe2", NUM_OWNERS);
        SafeInstance memory safe2 = _setupSafe(keys2, THRESHOLD);
        SafeTestLib.enableModule(safe2, address(livenessModule2));

        (, uint256[] memory keys3) = SafeTestLib.makeAddrsAndKeys("safe3", NUM_OWNERS);
        SafeInstance memory safe3 = _setupSafe(keys3, THRESHOLD);
        SafeTestLib.enableModule(safe3, address(livenessModule2));

        address fallback1 = makeAddr("fallback1");
        address fallback2 = makeAddr("fallback2");
        address fallback3 = makeAddr("fallback3");

        // Configure module for each safe
        _enableModule(safe1, 1 days, fallback1);
        _enableModule(safe2, 2 days, fallback2);
        _enableModule(safe3, 3 days, fallback3);

        // Verify each safe has independent configuration
        LivenessModule2.ModuleConfig memory config1 = livenessModule2.livenessSafeConfiguration(safe1.safe);
        assertEq(config1.livenessResponsePeriod, 1 days);
        assertEq(config1.fallbackOwner, fallback1);

        LivenessModule2.ModuleConfig memory config2 = livenessModule2.livenessSafeConfiguration(safe2.safe);
        assertEq(config2.livenessResponsePeriod, 2 days);
        assertEq(config2.fallbackOwner, fallback2);

        LivenessModule2.ModuleConfig memory config3 = livenessModule2.livenessSafeConfiguration(safe3.safe);
        assertEq(config3.livenessResponsePeriod, 3 days);
        assertEq(config3.fallbackOwner, fallback3);
    }

    function test_configureLivenessModule_requiresSafeModuleInstallation_reverts() external {
        // Create a safe that has NOT installed the module at the Safe level
        (, uint256[] memory newKeys) = SafeTestLib.makeAddrsAndKeys("newSafe", NUM_OWNERS);
        SafeInstance memory newSafe = _setupSafe(newKeys, THRESHOLD);
        // Note: we don't call SafeTestLib.enableModule here

        // Now configure should revert because the module is not enabled at the Safe level
        vm.expectRevert(LivenessModule2.LivenessModule2_ModuleNotEnabled.selector);
        vm.prank(address(newSafe.safe));
        livenessModule2.configureLivenessModule(
            LivenessModule2.ModuleConfig({ livenessResponsePeriod: CHALLENGE_PERIOD, fallbackOwner: fallbackOwner })
        );
    }

    function test_configureLivenessModule_invalidResponsePeriod_reverts() external {
        // Test with zero period
        vm.expectRevert(LivenessModule2.LivenessModule2_InvalidResponsePeriod.selector);
        vm.prank(address(safeInstance.safe));
        livenessModule2.configureLivenessModule(
            LivenessModule2.ModuleConfig({ livenessResponsePeriod: 0, fallbackOwner: fallbackOwner })
        );
    }

    function test_configureLivenessModule_invalidFallbackOwner_reverts() external {
        // Test with zero address
        vm.expectRevert(LivenessModule2.LivenessModule2_InvalidFallbackOwner.selector);
        vm.prank(address(safeInstance.safe));
        livenessModule2.configureLivenessModule(
            LivenessModule2.ModuleConfig({ livenessResponsePeriod: CHALLENGE_PERIOD, fallbackOwner: address(0) })
        );
    }

    /// @notice Checks configuration reverts when the contract is too old.
    function test_configureLivenessModule_withWrongVersion_reverts() external {
        // nosemgrep: sol-style-use-abi-encodecall
        vm.mockCall(address(safeInstance.safe), abi.encodeWithSignature("VERSION()"), abi.encode("1.4.0"));
        vm.expectRevert(LivenessModule2.LivenessModule2_InvalidVersion.selector, address(livenessModule2));
        vm.prank(address(safeInstance.safe));
        livenessModule2.configureLivenessModule(
            LivenessModule2.ModuleConfig({ livenessResponsePeriod: CHALLENGE_PERIOD, fallbackOwner: fallbackOwner })
        );
    }

    /// @notice Checks configuration succeeds even with pre-release versions of the Safe contract.
    function test_configureLivenessModule_withPreReleaseVersion_succeeds() external {
        // nosemgrep: sol-style-use-abi-encodecall
        vm.mockCall(address(safeInstance.safe), abi.encodeWithSignature("VERSION()"), abi.encode("1.4.1-rc.1"));
        vm.prank(address(safeInstance.safe));
        livenessModule2.configureLivenessModule(
            LivenessModule2.ModuleConfig({ livenessResponsePeriod: CHALLENGE_PERIOD, fallbackOwner: fallbackOwner })
        );
    }

    function test_configureLivenessModule_cancelsExistingChallenge_succeeds() external {
        // First configure the module
        _enableModule(safeInstance, CHALLENGE_PERIOD, fallbackOwner);

        // Start a challenge
        vm.prank(fallbackOwner);
        livenessModule2.challenge(safeInstance.safe);

        // Verify challenge exists
        uint256 challengeEndTime = livenessModule2.getChallengePeriodEnd(safeInstance.safe);
        assertGt(challengeEndTime, 0);

        // Reconfigure the module, which should cancel the challenge and emit ChallengeCancelled
        vm.expectEmit(true, true, true, true);
        emit ChallengeCancelled(address(safeInstance.safe));
        vm.expectEmit(true, true, true, true);
        emit ModuleConfigured(address(safeInstance.safe), CHALLENGE_PERIOD * 2, fallbackOwner);

        vm.prank(address(safeInstance.safe));
        livenessModule2.configureLivenessModule(
            LivenessModule2.ModuleConfig({ livenessResponsePeriod: CHALLENGE_PERIOD * 2, fallbackOwner: fallbackOwner })
        );

        // Verify challenge was cancelled
        challengeEndTime = livenessModule2.getChallengePeriodEnd(safeInstance.safe);
        assertEq(challengeEndTime, 0);
    }
}

/// @title LivenessModule2_ClearLivenessModule_Test
/// @notice Tests clearing the module configuration
contract LivenessModule2_ClearLivenessModule_Test is LivenessModule2_TestInit {
    function test_clearLivenessModule_succeeds() external {
        _enableModule(safeInstance, CHALLENGE_PERIOD, fallbackOwner);

        // Start a challenge to test that clearing also cancels it
        vm.prank(fallbackOwner);
        livenessModule2.challenge(safeInstance.safe);
        assertGt(livenessModule2.challengeStartTime(safeInstance.safe), 0);

        // First disable the module at the Safe level
        SafeTestLib.execTransaction(
            safeInstance,
            address(safeInstance.safe),
            0,
            abi.encodeCall(ModuleManager.disableModule, (address(0x1), address(livenessModule2))),
            Enum.Operation.Call
        );

        // Clear should emit ChallengeCancelled then ModuleCleared
        vm.expectEmit(true, true, true, true);
        emit ChallengeCancelled(address(safeInstance.safe));
        vm.expectEmit(true, true, true, true);
        emit ModuleCleared(address(safeInstance.safe));

        // Now clear the configuration
        SafeTestLib.execTransaction(
            safeInstance,
            address(livenessModule2),
            0,
            abi.encodeCall(LivenessModule2.clearLivenessModule, ()),
            Enum.Operation.Call
        );

        LivenessModule2.ModuleConfig memory clearedConfig = livenessModule2.livenessSafeConfiguration(safeInstance.safe);
        assertEq(clearedConfig.livenessResponsePeriod, 0);
        assertEq(clearedConfig.fallbackOwner, address(0));
        assertEq(livenessModule2.challengeStartTime(safeInstance.safe), 0);
    }

    function test_clearLivenessModule_notConfigured_reverts() external {
        vm.expectRevert(LivenessModule2.LivenessModule2_ModuleNotConfigured.selector);
        vm.prank(address(safeInstance.safe));
        livenessModule2.clearLivenessModule();
    }

    function test_clearLivenessModule_moduleStillEnabled_reverts() external {
        _enableModule(safeInstance, CHALLENGE_PERIOD, fallbackOwner);

        // Try to clear while module is still enabled (should revert)
        vm.expectRevert(LivenessModule2.LivenessModule2_ModuleStillEnabled.selector);
        vm.prank(address(safeInstance.safe));
        livenessModule2.clearLivenessModule();
    }
}

/// @title LivenessModule2_Challenge_Test
/// @notice Tests the challenge mechanism
contract LivenessModule2_Challenge_Test is LivenessModule2_TestInit {
    function setUp() public override {
        super.setUp();
        _enableModule(safeInstance, CHALLENGE_PERIOD, fallbackOwner);
    }

    function test_challenge_succeeds() external {
        vm.expectEmit(true, true, true, true);
        emit ChallengeStarted(address(safeInstance.safe), block.timestamp);

        vm.prank(fallbackOwner);
        livenessModule2.challenge(safeInstance.safe);

        uint256 challengeEndTime = livenessModule2.getChallengePeriodEnd(safeInstance.safe);
        assertEq(challengeEndTime, block.timestamp + CHALLENGE_PERIOD);
    }

    function test_challenge_notFallbackOwner_reverts() external {
        address notFallback = makeAddr("notFallback");

        vm.expectRevert(LivenessModule2.LivenessModule2_UnauthorizedCaller.selector);
        vm.prank(notFallback);
        livenessModule2.challenge(safeInstance.safe);
    }

    function test_challenge_moduleNotEnabled_reverts() external {
        address newSafe = makeAddr("newSafe");

        vm.expectRevert(LivenessModule2.LivenessModule2_ModuleNotConfigured.selector);
        vm.prank(fallbackOwner);
        livenessModule2.challenge(Safe(payable(newSafe)));
    }

    function test_challenge_alreadyExists_reverts() external {
        vm.prank(fallbackOwner);
        livenessModule2.challenge(safeInstance.safe);

        vm.expectRevert(LivenessModule2.LivenessModule2_ChallengeAlreadyExists.selector);
        vm.prank(fallbackOwner);
        livenessModule2.challenge(safeInstance.safe);
    }

    function test_challenge_moduleDisabledAtSafeLevel_reverts() external {
        // Create a Safe, configure it, then disable the module at Safe level
        (, uint256[] memory newKeys) = SafeTestLib.makeAddrsAndKeys("disabledSafe", NUM_OWNERS);
        SafeInstance memory disabledSafe = _setupSafe(newKeys, THRESHOLD);

        // First enable module at Safe level
        SafeTestLib.enableModule(disabledSafe, address(livenessModule2));

        // Then configure
        _enableModule(disabledSafe, CHALLENGE_PERIOD, fallbackOwner);

        // Now disable the module at Safe level (but keep config)
        SafeTestLib.execTransaction(
            disabledSafe,
            address(disabledSafe.safe),
            0,
            abi.encodeCall(ModuleManager.disableModule, (address(0x1), address(livenessModule2))),
            Enum.Operation.Call
        );

        // Try to challenge - should revert because module is disabled at Safe level
        vm.expectRevert(LivenessModule2.LivenessModule2_ModuleNotEnabled.selector);
        vm.prank(fallbackOwner);
        livenessModule2.challenge(disabledSafe.safe);
    }

    function test_respond_succeeds() external {
        // Start a challenge
        vm.prank(fallbackOwner);
        livenessModule2.challenge(safeInstance.safe);

        // Cancel it
        vm.expectEmit(true, true, true, true);
        emit ChallengeCancelled(address(safeInstance.safe));

        _respondToChallenge(safeInstance);

        // Verify challenge is cancelled
        uint256 challengeEndTime = livenessModule2.getChallengePeriodEnd(safeInstance.safe);
        assertEq(challengeEndTime, 0);
    }

    function test_respond_noChallenge_reverts() external {
        // Module is already enabled in setUp, no challenge exists

        // Try to cancel when no challenge exists - this should fail
        // We need to use a transaction that would work if there was a challenge
        // Use safeTxGas > 0 to allow the Safe to handle the revert gracefully
        bytes memory data = abi.encodeCall(LivenessModule2.respond, ());
        bool success = SafeTestLib.execTransaction(
            safeInstance,
            address(livenessModule2),
            0,
            data,
            Enum.Operation.Call,
            100000, // safeTxGas > 0 allows transaction to fail without reverting
            0,
            0,
            address(0),
            address(0),
            ""
        );
        assertFalse(success, "Should fail to cancel non-existent challenge");
    }

    function test_respond_afterResponsePeriod_succeeds() external {
        // Start a challenge
        vm.prank(fallbackOwner);
        livenessModule2.challenge(safeInstance.safe);

        // Warp past challenge period
        vm.warp(block.timestamp + CHALLENGE_PERIOD + 1);

        // Should be able to respond even after response period (per new specs)
        vm.expectEmit(true, true, true, true);
        emit ChallengeCancelled(address(safeInstance.safe));
        vm.prank(address(safeInstance.safe));
        livenessModule2.respond();

        // Verify challenge was cancelled
        assertEq(livenessModule2.challengeStartTime(safeInstance.safe), 0);
    }

    function test_respond_moduleNotConfigured_reverts() external {
        // Create a Safe that hasn't enabled the module
        (, uint256[] memory newKeys) = SafeTestLib.makeAddrsAndKeys("safeThatDidntEnable", NUM_OWNERS);
        SafeInstance memory safeThatDidntEnable = _setupSafe(newKeys, THRESHOLD);
        // Note: we don't call SafeTestLib.enableModule here

        vm.expectRevert(LivenessModule2.LivenessModule2_ModuleNotConfigured.selector);
        vm.prank(address(safeThatDidntEnable.safe));
        livenessModule2.respond();
    }

    function test_respond_moduleNotEnabled_reverts() external {
        // Create a Safe, enable and configure the module, then disable it
        (, uint256[] memory newKeys) = SafeTestLib.makeAddrsAndKeys("configuredButDisabled", NUM_OWNERS);
        SafeInstance memory configuredSafe = _setupSafe(newKeys, THRESHOLD);

        // First enable module at Safe level
        SafeTestLib.enableModule(configuredSafe, address(livenessModule2));

        // Configure the module (this sets the configuration)
        _enableModule(configuredSafe, CHALLENGE_PERIOD, fallbackOwner);

        // Now disable the module at Safe level (but keep config)
        SafeTestLib.execTransaction(
            configuredSafe,
            address(configuredSafe.safe),
            0,
            abi.encodeCall(ModuleManager.disableModule, (address(0x1), address(livenessModule2))),
            Enum.Operation.Call
        );

        // Verify the Safe still has configuration but module is not enabled
        LivenessModule2.ModuleConfig memory configuredConfig =
            livenessModule2.livenessSafeConfiguration(configuredSafe.safe);
        assertTrue(configuredConfig.livenessResponsePeriod > 0); // Configuration exists
        assertTrue(configuredConfig.fallbackOwner != address(0)); // Configuration exists
        assertFalse(configuredSafe.safe.isModuleEnabled(address(livenessModule2))); // Module not enabled

        // Now respond() should revert because module is not enabled
        vm.expectRevert(LivenessModule2.LivenessModule2_ModuleNotEnabled.selector);
        vm.prank(address(configuredSafe.safe));
        livenessModule2.respond();
    }
}

/// @title LivenessModule2_ChangeOwnershipToFallback_Test
/// @notice Tests the ownership transfer after successful challenge
contract LivenessModule2_ChangeOwnershipToFallback_Test is LivenessModule2_TestInit {
    function setUp() public override {
        super.setUp();
        _enableModule(safeInstance, CHALLENGE_PERIOD, fallbackOwner);
    }

    /// @notice Tests that ownership successfully transfers to fallback owner after challenge period expires
    function testFuzz_changeOwnershipToFallback_succeeds(uint256 timeAfterExpiry) external {
        // Bound time to reasonable values (1 second to 365 days after expiry)
        timeAfterExpiry = bound(timeAfterExpiry, 1, 365 days);

        // Set a guard to verify it gets removed
        DummyGuard otherGuard = new DummyGuard();

        SafeTestLib.execTransaction(
            safeInstance,
            address(safeInstance.safe),
            0,
            abi.encodeCall(GuardManager.setGuard, (address(otherGuard))),
            Enum.Operation.Call
        );
        assertEq(_getGuard(safeInstance), address(otherGuard));

        // Start a challenge
        vm.prank(fallbackOwner);
        livenessModule2.challenge(safeInstance.safe);

        // Warp past challenge period
        vm.warp(block.timestamp + CHALLENGE_PERIOD + timeAfterExpiry);

        // Execute ownership transfer
        vm.expectEmit(true, true, true, true);
        emit ChallengeSucceeded(address(safeInstance.safe), fallbackOwner);

        vm.prank(fallbackOwner);
        livenessModule2.changeOwnershipToFallback(safeInstance.safe);

        // Verify ownership changed
        address[] memory newOwners = safeInstance.safe.getOwners();
        assertEq(newOwners.length, 1);
        assertEq(newOwners[0], fallbackOwner);
        assertEq(safeInstance.safe.getThreshold(), 1);

        // Verify challenge is reset
        uint256 challengeEndTime = livenessModule2.getChallengePeriodEnd(safeInstance.safe);
        assertEq(challengeEndTime, 0);

        // Verify guard was removed
        assertEq(_getGuard(safeInstance), address(0));
    }

    /// @notice Tests that changeOwnershipToFallback reverts if module is not configured
    function test_changeOwnershipToFallback_moduleNotEnabled_reverts() external {
        address newSafe = makeAddr("newSafe");

        vm.prank(fallbackOwner);
        vm.expectRevert(LivenessModule2.LivenessModule2_ModuleNotConfigured.selector);
        livenessModule2.changeOwnershipToFallback(Safe(payable(newSafe)));
    }

    /// @notice Tests that changeOwnershipToFallback reverts if no challenge exists
    function test_changeOwnershipToFallback_noChallenge_reverts() external {
        vm.prank(fallbackOwner);
        vm.expectRevert(LivenessModule2.LivenessModule2_ChallengeDoesNotExist.selector);
        livenessModule2.changeOwnershipToFallback(safeInstance.safe);
    }

    /// @notice Tests that changeOwnershipToFallback reverts if called before response period expires
    function testFuzz_changeOwnershipToFallback_beforeResponsePeriod_reverts(uint256 timeElapsed) external {
        // Bound time to be within response period (not yet expired)
        timeElapsed = bound(timeElapsed, 0, CHALLENGE_PERIOD - 1);

        // Start a challenge
        vm.prank(fallbackOwner);
        livenessModule2.challenge(safeInstance.safe);

        // Warp to a time before response period expires
        vm.warp(block.timestamp + timeElapsed);

        // Try to execute before response period expires
        vm.prank(fallbackOwner);
        vm.expectRevert(LivenessModule2.LivenessModule2_ResponsePeriodActive.selector);
        livenessModule2.changeOwnershipToFallback(safeInstance.safe);
    }

    /// @notice Tests that changeOwnershipToFallback reverts if module is disabled at Safe level
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

    /// @notice Tests that only the fallback owner can execute changeOwnershipToFallback
    function testFuzz_changeOwnershipToFallback_onlyFallbackOwner_succeeds(address _caller) external {
        vm.assume(_caller != fallbackOwner);

        // Start a challenge
        vm.prank(fallbackOwner);
        livenessModule2.challenge(safeInstance.safe);

        // Warp past challenge period
        vm.warp(block.timestamp + CHALLENGE_PERIOD + 1);

        // Try from non-fallback owner - should fail
        vm.prank(_caller);
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

    /// @notice Tests that a new challenge can be started after ownership transfer completes
    function test_changeOwnershipToFallback_canRechallenge_succeeds() external {
        // Start and execute first challenge
        vm.prank(fallbackOwner);
        livenessModule2.challenge(safeInstance.safe);

        vm.warp(block.timestamp + CHALLENGE_PERIOD + 1);
        vm.prank(fallbackOwner);
        livenessModule2.changeOwnershipToFallback(safeInstance.safe);

        // Start a new challenge (as fallback owner)
        vm.prank(fallbackOwner);
        livenessModule2.challenge(safeInstance.safe);

        uint256 challengeEndTime = livenessModule2.getChallengePeriodEnd(safeInstance.safe);
        assertGt(challengeEndTime, 0);
    }
}

/// @title LivenessModule2_GetChallengePeriodEnd_Test
/// @notice Tests the getChallengePeriodEnd function and related view functionality
contract LivenessModule2_GetChallengePeriodEnd_Test is LivenessModule2_TestInit {
    function test_safeConfigs_succeeds() external {
        // Before enabling
        LivenessModule2.ModuleConfig memory configBefore = livenessModule2.livenessSafeConfiguration(safeInstance.safe);
        assertEq(configBefore.livenessResponsePeriod, 0);
        assertEq(configBefore.fallbackOwner, address(0));
        assertEq(livenessModule2.challengeStartTime(safeInstance.safe), 0);

        // After enabling
        _enableModule(safeInstance, CHALLENGE_PERIOD, fallbackOwner);
        LivenessModule2.ModuleConfig memory configAfter = livenessModule2.livenessSafeConfiguration(safeInstance.safe);
        assertEq(configAfter.livenessResponsePeriod, CHALLENGE_PERIOD);
        assertEq(configAfter.fallbackOwner, fallbackOwner);
        assertEq(livenessModule2.challengeStartTime(safeInstance.safe), 0);
    }

    function test_getChallengePeriodEnd_succeeds() external {
        _enableModule(safeInstance, CHALLENGE_PERIOD, fallbackOwner);

        // No challenge
        assertEq(livenessModule2.getChallengePeriodEnd(safeInstance.safe), 0);

        // With challenge
        vm.prank(fallbackOwner);
        livenessModule2.challenge(safeInstance.safe);
        assertEq(livenessModule2.getChallengePeriodEnd(safeInstance.safe), block.timestamp + CHALLENGE_PERIOD);

        // After cancellation
        _respondToChallenge(safeInstance);
        assertEq(livenessModule2.getChallengePeriodEnd(safeInstance.safe), 0);
    }
}

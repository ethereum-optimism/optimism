// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import "test/safe-tools/SafeTestTools.sol";

import { LivenessModule2 } from "src/safe/LivenessModule2.sol";
import { ILivenessModule2 } from "interfaces/safe/ILivenessModule2.sol";

/// @title LivenessModule2_TestInit
/// @notice Reusable test initialization for `LivenessModule2` tests.
contract LivenessModule2_TestInit is Test, SafeTestTools {
    using SafeTestLib for SafeInstance;

    // Events
    event ModuleEnabled(address indexed safe, uint256 livenessChallengePeriod, address fallbackOwner);
    event ModuleDisabled(address indexed safe);
    event ChallengeStarted(address indexed safe, uint256 challengeStartTime);
    event ChallengeCancelled(address indexed safe);
    event ChallengeExecuted(address indexed safe, address fallbackOwner);

    uint256 constant INIT_TIME = 10;
    uint256 constant CHALLENGE_PERIOD = 7 days;
    uint256 constant NUM_OWNERS = 5;
    uint256 constant THRESHOLD = 3;

    LivenessModule2 livenessModule2;
    SafeInstance safeInstance;
    address fallbackOwner;
    address[] owners;
    uint256[] ownerPKs;

    function setUp() public virtual {
        vm.warp(INIT_TIME);

        // Deploy the singleton LivenessModule2
        livenessModule2 = new LivenessModule2();

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

    /// @notice Helper to enable the LivenessModule2 for a Safe
    function _enableModule(SafeInstance memory _safe, uint256 _period, address _fallback) internal {
        SafeTestLib.execTransaction(
            _safe,
            address(livenessModule2),
            0,
            abi.encodeCall(LivenessModule2.enableModule, (_period, _fallback)),
            Enum.Operation.Call
        );
    }

    /// @notice Helper to disable the LivenessModule2 for a Safe
    function _disableModule(SafeInstance memory _safe) internal {
        SafeTestLib.execTransaction(
            _safe, address(livenessModule2), 0, abi.encodeCall(LivenessModule2.disableModule, ()), Enum.Operation.Call
        );
    }

    /// @notice Helper to cancel a challenge from a Safe
    function _cancelChallenge(SafeInstance memory _safe) internal {
        SafeTestLib.execTransaction(
            _safe, address(livenessModule2), 0, abi.encodeCall(LivenessModule2.cancelChallenge, ()), Enum.Operation.Call
        );
    }
}

/// @title LivenessModule2_EnableModule_Test
/// @notice Tests enabling and disabling the module
contract LivenessModule2_EnableModule_Test is LivenessModule2_TestInit {
    function test_enableModule_succeeds() external {
        vm.expectEmit(true, true, true, true);
        emit ModuleEnabled(address(safeInstance.safe), CHALLENGE_PERIOD, fallbackOwner);

        _enableModule(safeInstance, CHALLENGE_PERIOD, fallbackOwner);

        (uint256 period, address fbOwner) = livenessModule2.viewConfiguration(address(safeInstance.safe));
        assertEq(period, CHALLENGE_PERIOD);
        assertEq(fbOwner, fallbackOwner);
    }

    function test_enableModule_acceptsArbitraryNumberOfSafes_succeeds() external {
        // Test that multiple independent safes can enable the module
        address safe1 = makeAddr("safe1");
        address safe2 = makeAddr("safe2");
        address safe3 = makeAddr("safe3");

        address fallback1 = makeAddr("fallback1");
        address fallback2 = makeAddr("fallback2");
        address fallback3 = makeAddr("fallback3");

        // Enable module for safe1
        vm.prank(safe1);
        livenessModule2.enableModule(1 days, fallback1);

        // Enable module for safe2
        vm.prank(safe2);
        livenessModule2.enableModule(2 days, fallback2);

        // Enable module for safe3
        vm.prank(safe3);
        livenessModule2.enableModule(3 days, fallback3);

        // Verify each safe has independent configuration
        (uint256 period1, address fb1) = livenessModule2.viewConfiguration(safe1);
        assertEq(period1, 1 days);
        assertEq(fb1, fallback1);

        (uint256 period2, address fb2) = livenessModule2.viewConfiguration(safe2);
        assertEq(period2, 2 days);
        assertEq(fb2, fallback2);

        (uint256 period3, address fb3) = livenessModule2.viewConfiguration(safe3);
        assertEq(period3, 3 days);
        assertEq(fb3, fallback3);
    }

    function test_enableModule_requiresSafeModuleInstallation_reverts() external {
        // Create a safe that has NOT installed the module at the Safe level
        (, uint256[] memory newKeys) = SafeTestLib.makeAddrsAndKeys("newSafe", NUM_OWNERS);
        SafeInstance memory newSafe = _setupSafe(newKeys, THRESHOLD);
        // Note: we don't call SafeTestLib.enableModule here

        // Configure the module (this should work even without Safe-level installation)
        _enableModule(newSafe, CHALLENGE_PERIOD, fallbackOwner);

        // Verify configuration was stored
        (uint256 period, address fb) = livenessModule2.viewConfiguration(address(newSafe.safe));
        assertEq(period, CHALLENGE_PERIOD);
        assertEq(fb, fallbackOwner);

        // But trying to execute a challenge should fail because the Safe hasn't
        // enabled this module at the Safe level (so execTransactionFromModule will fail)
        vm.prank(fallbackOwner);
        livenessModule2.startChallenge(address(newSafe.safe));

        vm.warp(block.timestamp + CHALLENGE_PERIOD + 1);

        // This should fail because the module is not enabled at the Safe level
        // The Safe will revert with "GS104" when execTransactionFromModule is called by non-enabled module
        vm.expectRevert("GS104");
        livenessModule2.changeOwnershipToFallback(address(newSafe.safe));
    }

    function test_enableModule_alreadyEnabled_reverts() external {
        _enableModule(safeInstance, CHALLENGE_PERIOD, fallbackOwner);

        // Try to enable again directly from the Safe address
        vm.expectRevert(ILivenessModule2.LivenessModule2_ModuleAlreadyEnabled.selector);
        vm.prank(address(safeInstance.safe));
        livenessModule2.enableModule(CHALLENGE_PERIOD, fallbackOwner);
    }

    function test_enableModule_invalidParameters_reverts() external {
        // Test with zero period
        vm.expectRevert(ILivenessModule2.LivenessModule2_InvalidParameters.selector);
        vm.prank(address(safeInstance.safe));
        livenessModule2.enableModule(0, fallbackOwner);

        // Test with zero address
        vm.expectRevert(ILivenessModule2.LivenessModule2_InvalidParameters.selector);
        vm.prank(address(safeInstance.safe));
        livenessModule2.enableModule(CHALLENGE_PERIOD, address(0));
    }

    function test_disableModule_succeeds() external {
        _enableModule(safeInstance, CHALLENGE_PERIOD, fallbackOwner);

        vm.expectEmit(true, true, true, true);
        emit ModuleDisabled(address(safeInstance.safe));

        _disableModule(safeInstance);

        (uint256 period, address fbOwner) = livenessModule2.viewConfiguration(address(safeInstance.safe));
        assertEq(period, 0);
        assertEq(fbOwner, address(0));
    }

    function test_disableModule_notEnabled_reverts() external {
        vm.expectRevert(ILivenessModule2.LivenessModule2_ModuleNotEnabled.selector);
        vm.prank(address(safeInstance.safe));
        livenessModule2.disableModule();
    }
}

/// @title LivenessModule2_StartChallenge_Test
/// @notice Tests the challenge mechanism
contract LivenessModule2_StartChallenge_Test is LivenessModule2_TestInit {
    function setUp() public override {
        super.setUp();
        _enableModule(safeInstance, CHALLENGE_PERIOD, fallbackOwner);
    }

    function test_startChallenge_succeeds() external {
        vm.expectEmit(true, true, true, true);
        emit ChallengeStarted(address(safeInstance.safe), block.timestamp);

        vm.prank(fallbackOwner);
        livenessModule2.startChallenge(address(safeInstance.safe));

        uint256 challengeEndTime = livenessModule2.isChallenged(address(safeInstance.safe));
        assertEq(challengeEndTime, block.timestamp + CHALLENGE_PERIOD);
    }

    function test_startChallenge_notFallbackOwner_reverts() external {
        address notFallback = makeAddr("notFallback");

        vm.expectRevert(ILivenessModule2.LivenessModule2_UnauthorizedCaller.selector);
        vm.prank(notFallback);
        livenessModule2.startChallenge(address(safeInstance.safe));
    }

    function test_startChallenge_moduleNotEnabled_reverts() external {
        address newSafe = makeAddr("newSafe");

        vm.expectRevert(ILivenessModule2.LivenessModule2_ModuleNotEnabled.selector);
        vm.prank(fallbackOwner);
        livenessModule2.startChallenge(newSafe);
    }

    function test_startChallenge_alreadyExists_reverts() external {
        vm.prank(fallbackOwner);
        livenessModule2.startChallenge(address(safeInstance.safe));

        vm.expectRevert(ILivenessModule2.LivenessModule2_ChallengeAlreadyExists.selector);
        vm.prank(fallbackOwner);
        livenessModule2.startChallenge(address(safeInstance.safe));
    }

    function test_cancelChallenge_succeeds() external {
        // Start a challenge
        vm.prank(fallbackOwner);
        livenessModule2.startChallenge(address(safeInstance.safe));

        // Cancel it
        vm.expectEmit(true, true, true, true);
        emit ChallengeCancelled(address(safeInstance.safe));

        _cancelChallenge(safeInstance);

        // Verify challenge is cancelled
        uint256 challengeEndTime = livenessModule2.isChallenged(address(safeInstance.safe));
        assertEq(challengeEndTime, 0);
    }

    function test_cancelChallenge_noChallenge_reverts() external {
        // Module is already enabled in setUp, no challenge exists

        // Try to cancel when no challenge exists - this should fail
        // We need to use a transaction that would work if there was a challenge
        // Use safeTxGas > 0 to allow the Safe to handle the revert gracefully
        bytes memory data = abi.encodeCall(LivenessModule2.cancelChallenge, ());
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

    function test_cancelChallenge_afterPeriod_reverts() external {
        // Start a challenge
        vm.prank(fallbackOwner);
        livenessModule2.startChallenge(address(safeInstance.safe));

        // Warp past challenge period
        vm.warp(block.timestamp + CHALLENGE_PERIOD + 1);

        // Try to cancel - should fail as challenge is successful
        vm.expectRevert(ILivenessModule2.LivenessModule2_ChallengeNotSuccessful.selector);
        vm.prank(address(safeInstance.safe));
        livenessModule2.cancelChallenge();
    }

    function test_cancelChallenge_moduleNotEnabled_reverts() external {
        // Create a Safe that hasn't enabled the module
        address safeThatDidntEnable = makeAddr("safeThatDidntEnable");

        vm.expectRevert(ILivenessModule2.LivenessModule2_ModuleNotEnabled.selector);
        vm.prank(safeThatDidntEnable);
        livenessModule2.cancelChallenge();
    }
}

/// @title LivenessModule2_ChangeOwnershipToFallback_Test
/// @notice Tests the ownership transfer after successful challenge
contract LivenessModule2_ChangeOwnershipToFallback_Test is LivenessModule2_TestInit {
    function setUp() public override {
        super.setUp();
        _enableModule(safeInstance, CHALLENGE_PERIOD, fallbackOwner);
    }

    function test_changeOwnershipToFallback_succeeds() external {
        // Start a challenge
        vm.prank(fallbackOwner);
        livenessModule2.startChallenge(address(safeInstance.safe));

        // Warp past challenge period
        vm.warp(block.timestamp + CHALLENGE_PERIOD + 1);

        // Execute ownership transfer
        vm.expectEmit(true, true, true, true);
        emit ChallengeExecuted(address(safeInstance.safe), fallbackOwner);

        livenessModule2.changeOwnershipToFallback(address(safeInstance.safe));

        // Verify ownership changed
        address[] memory newOwners = safeInstance.safe.getOwners();
        assertEq(newOwners.length, 1);
        assertEq(newOwners[0], fallbackOwner);
        assertEq(safeInstance.safe.getThreshold(), 1);

        // Verify challenge is reset
        uint256 challengeEndTime = livenessModule2.isChallenged(address(safeInstance.safe));
        assertEq(challengeEndTime, 0);
    }

    function test_changeOwnershipToFallback_moduleNotEnabled_reverts() external {
        address newSafe = makeAddr("newSafe");

        vm.expectRevert(ILivenessModule2.LivenessModule2_ModuleNotEnabled.selector);
        livenessModule2.changeOwnershipToFallback(newSafe);
    }

    function test_changeOwnershipToFallback_noChallenge_reverts() external {
        vm.expectRevert(ILivenessModule2.LivenessModule2_ChallengeDoesNotExist.selector);
        livenessModule2.changeOwnershipToFallback(address(safeInstance.safe));
    }

    function test_changeOwnershipToFallback_challengeNotSuccessful_reverts() external {
        // Start a challenge
        vm.prank(fallbackOwner);
        livenessModule2.startChallenge(address(safeInstance.safe));

        // Try to execute before period expires
        vm.expectRevert(ILivenessModule2.LivenessModule2_ChallengeNotSuccessful.selector);
        livenessModule2.changeOwnershipToFallback(address(safeInstance.safe));
    }

    function test_changeOwnershipToFallback_canBeCalledByAnyone_succeeds() external {
        // Start a challenge
        vm.prank(fallbackOwner);
        livenessModule2.startChallenge(address(safeInstance.safe));

        // Warp past challenge period
        vm.warp(block.timestamp + CHALLENGE_PERIOD + 1);

        // Execute from random address
        address randomCaller = makeAddr("randomCaller");
        vm.prank(randomCaller);
        livenessModule2.changeOwnershipToFallback(address(safeInstance.safe));

        // Verify ownership changed
        address[] memory newOwners = safeInstance.safe.getOwners();
        assertEq(newOwners.length, 1);
        assertEq(newOwners[0], fallbackOwner);
    }

    function test_changeOwnershipToFallback_allowsNewChallenge_succeeds() external {
        // Start and execute first challenge
        vm.prank(fallbackOwner);
        livenessModule2.startChallenge(address(safeInstance.safe));

        vm.warp(block.timestamp + CHALLENGE_PERIOD + 1);
        livenessModule2.changeOwnershipToFallback(address(safeInstance.safe));

        // Start a new challenge (as fallback owner)
        vm.prank(fallbackOwner);
        livenessModule2.startChallenge(address(safeInstance.safe));

        uint256 challengeEndTime = livenessModule2.isChallenged(address(safeInstance.safe));
        assertGt(challengeEndTime, 0);
    }
}

/// @title LivenessModule2_ViewConfiguration_Test
/// @notice Tests view functions
contract LivenessModule2_ViewConfiguration_Test is LivenessModule2_TestInit {
    function test_viewConfiguration_works() external {
        // Before enabling
        (uint256 period1, address fbOwner1) = livenessModule2.viewConfiguration(address(safeInstance.safe));
        assertEq(period1, 0);
        assertEq(fbOwner1, address(0));

        // After enabling
        _enableModule(safeInstance, CHALLENGE_PERIOD, fallbackOwner);
        (uint256 period2, address fbOwner2) = livenessModule2.viewConfiguration(address(safeInstance.safe));
        assertEq(period2, CHALLENGE_PERIOD);
        assertEq(fbOwner2, fallbackOwner);
    }

    function test_isChallenged_works() external {
        _enableModule(safeInstance, CHALLENGE_PERIOD, fallbackOwner);

        // No challenge
        assertEq(livenessModule2.isChallenged(address(safeInstance.safe)), 0);

        // With challenge
        vm.prank(fallbackOwner);
        livenessModule2.startChallenge(address(safeInstance.safe));
        assertEq(livenessModule2.isChallenged(address(safeInstance.safe)), block.timestamp + CHALLENGE_PERIOD);

        // After cancellation
        _cancelChallenge(safeInstance);
        assertEq(livenessModule2.isChallenged(address(safeInstance.safe)), 0);
    }

    function test_version_works() external view {
        assertTrue(bytes(livenessModule2.version()).length > 0);
    }
}

/// @title LivenessModule2_IsChallenged_Test
/// @notice Tests invariants from specifications
contract LivenessModule2_IsChallenged_Test is LivenessModule2_TestInit {
    /// @notice Test iLM-001: No Concurrent Challenges
    function test_invariant_noConcurrentChallenges_works() external {
        _enableModule(safeInstance, CHALLENGE_PERIOD, fallbackOwner);

        // Start first challenge
        vm.prank(fallbackOwner);
        livenessModule2.startChallenge(address(safeInstance.safe));

        // Attempt second challenge should fail
        vm.expectRevert(ILivenessModule2.LivenessModule2_ChallengeAlreadyExists.selector);
        vm.prank(fallbackOwner);
        livenessModule2.startChallenge(address(safeInstance.safe));
    }

    /// @notice Test iLM-002: Honest Users Can Recover From Temporary Key Control
    function test_invariant_honestUsersCanRecover_works() external {
        _enableModule(safeInstance, CHALLENGE_PERIOD, fallbackOwner);

        // Simulate attacker with control over less than quorum
        // Start a challenge
        vm.prank(fallbackOwner);
        livenessModule2.startChallenge(address(safeInstance.safe));

        // Honest users (with quorum) can cancel the challenge
        _cancelChallenge(safeInstance);

        // Verify Safe is still controlled by original owners
        address[] memory currentOwners = safeInstance.safe.getOwners();
        assertEq(currentOwners.length, NUM_OWNERS);
        for (uint256 i = 0; i < NUM_OWNERS; i++) {
            bool found = false;
            for (uint256 j = 0; j < owners.length; j++) {
                if (currentOwners[i] == owners[j]) {
                    found = true;
                    break;
                }
            }
            assertTrue(found);
        }
    }

    /// @notice Test iLM-003: A Quorum Of Honest Users Retains Ownership
    function test_invariant_quorumRetainsOwnership_works() external {
        _enableModule(safeInstance, CHALLENGE_PERIOD, fallbackOwner);

        // Start a challenge
        vm.prank(fallbackOwner);
        livenessModule2.startChallenge(address(safeInstance.safe));

        // Quorum of honest users cancels before period expires
        vm.warp(block.timestamp + CHALLENGE_PERIOD - 1);
        _cancelChallenge(safeInstance);

        // Verify ownership unchanged
        address[] memory currentOwners = safeInstance.safe.getOwners();
        assertEq(currentOwners.length, NUM_OWNERS);
        assertEq(safeInstance.safe.getThreshold(), THRESHOLD);
    }
}

/// @title LivenessModule2_CancelChallenge_Test
/// @notice Integration tests with Safe operations
contract LivenessModule2_CancelChallenge_Test is LivenessModule2_TestInit {
    function test_integration_fullChallengeFlow_works() external {
        // Enable module
        _enableModule(safeInstance, CHALLENGE_PERIOD, fallbackOwner);

        // Start challenge
        vm.prank(fallbackOwner);
        livenessModule2.startChallenge(address(safeInstance.safe));

        // Safe responds and cancels challenge
        _cancelChallenge(safeInstance);

        // Start another challenge
        vm.prank(fallbackOwner);
        livenessModule2.startChallenge(address(safeInstance.safe));

        // This time Safe doesn't respond
        vm.warp(block.timestamp + CHALLENGE_PERIOD + 1);

        // Execute ownership transfer
        livenessModule2.changeOwnershipToFallback(address(safeInstance.safe));

        // Verify fallback owner has control
        address[] memory newOwners = safeInstance.safe.getOwners();
        assertEq(newOwners.length, 1);
        assertEq(newOwners[0], fallbackOwner);
        assertEq(safeInstance.safe.getThreshold(), 1);
    }

    function test_integration_multipleEnable_works() external {
        // Use different key sets for different safes
        (, uint256[] memory keys1) = SafeTestLib.makeAddrsAndKeys("safe1", NUM_OWNERS);
        (, uint256[] memory keys2) = SafeTestLib.makeAddrsAndKeys("safe2", NUM_OWNERS);

        SafeInstance memory safe1 = _setupSafe(keys1, THRESHOLD);
        SafeInstance memory safe2 = _setupSafe(keys2, THRESHOLD);

        SafeTestLib.enableModule(safe1, address(livenessModule2));
        SafeTestLib.enableModule(safe2, address(livenessModule2));

        address fallback1 = makeAddr("fallback1");
        address fallback2 = makeAddr("fallback2");

        // Enable module on both safes
        _enableModule(safe1, CHALLENGE_PERIOD, fallback1);
        _enableModule(safe2, CHALLENGE_PERIOD * 2, fallback2);

        // Verify configurations are independent
        (uint256 period1, address fb1) = livenessModule2.viewConfiguration(address(safe1.safe));
        (uint256 period2, address fb2) = livenessModule2.viewConfiguration(address(safe2.safe));

        assertEq(period1, CHALLENGE_PERIOD);
        assertEq(fb1, fallback1);
        assertEq(period2, CHALLENGE_PERIOD * 2);
        assertEq(fb2, fallback2);
    }
}

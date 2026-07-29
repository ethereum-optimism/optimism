// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Contracts
import { L2DevFeatureFlags } from "src/L2/L2DevFeatureFlags.sol";

// Interfaces
import { IL2DevFeatureFlags } from "interfaces/L2/IL2DevFeatureFlags.sol";

/// @title L2DevFeatureFlags_TestInit
abstract contract L2DevFeatureFlags_TestInit is CommonTest {
    IL2DevFeatureFlags public l2DevFeatureFlags;

    /// @notice Test setup.
    function setUp() public virtual override {
        super.setUp();
        l2DevFeatureFlags = IL2DevFeatureFlags(Predeploys.L2_DEV_FEATURE_FLAGS);
    }
}

/// @title L2DevFeatureFlags_Version_Test
/// @notice Tests the `version` function of the `L2DevFeatureFlags` contract.
contract L2DevFeatureFlags_Version_Test is L2DevFeatureFlags_TestInit {
    /// @notice Tests that the `version` function returns a non-empty string.
    function test_version_succeeds() public view {
        assertEq(keccak256(bytes(l2DevFeatureFlags.version())), keccak256(bytes("1.3.0")), "Versions should match");
    }
}

/// @title L2DevFeatureFlags_SetDevFeatureBitmap_Test
/// @notice Tests the `setDevFeatureBitmap` function of the `L2DevFeatureFlags` contract.
contract L2DevFeatureFlags_SetDevFeatureBitmap_Test is L2DevFeatureFlags_TestInit {
    /// @notice Tests that `setDevFeatureBitmap` reverts when called by an unauthorized caller.
    function testFuzz_setDevFeatureBitmap_unauthorizedCaller_reverts(address _caller, bytes32 _bitmap) public {
        vm.assume(_caller != Constants.DEPOSITOR_ACCOUNT);

        vm.expectRevert(L2DevFeatureFlags.L2DevFeatureFlags_Unauthorized.selector);
        vm.prank(_caller);
        l2DevFeatureFlags.setDevFeatureBitmap(_bitmap);
    }

    /// @notice Tests that `setDevFeatureBitmap` succeeds when called by the DEPOSITOR_ACCOUNT.
    function testFuzz_setDevFeatureBitmap_depositorAccount_succeeds(bytes32 _bitmap) public {
        vm.prank(Constants.DEPOSITOR_ACCOUNT);
        l2DevFeatureFlags.setDevFeatureBitmap(_bitmap);

        assertEq(l2DevFeatureFlags.devFeatureBitmap(), _bitmap);
    }
}

/// @title L2DevFeatureFlags_IsDevFeatureEnabled_Test
/// @notice Tests the `isDevFeatureEnabled` function of the `L2DevFeatureFlags` contract.
contract L2DevFeatureFlags_IsDevFeatureEnabled_Test is L2DevFeatureFlags_TestInit {
    /// @notice Tests that `isDevFeatureEnabled` returns false when the bitmap is zero.
    function testFuzz_isDevFeatureEnabled_zeroBitmap_succeeds(bytes32 _feature) public {
        vm.assume(_feature != bytes32(0));
        // SuperRootGamesMigration is hardcoded enabled. TODO(#21662): remove with the broader migration cleanup.
        vm.assume((_feature & DevFeatures.SUPER_ROOT_GAMES_MIGRATION) != DevFeatures.SUPER_ROOT_GAMES_MIGRATION);
        // Ensure `devFeatureBitmap` contains bytes32(0)
        vm.store(address(l2DevFeatureFlags), bytes32(uint256(keccak256("l2devfeatureflags.bitmap")) - 1), bytes32(0));
        assertFalse(l2DevFeatureFlags.isDevFeatureEnabled(_feature));
    }

    /// @notice Tests that `isDevFeatureEnabled(SUPER_ROOT_GAMES_MIGRATION)` always returns true regardless of stored
    /// bitmap.
    /// @dev TODO(#21662): remove with the broader SuperRootGamesMigration cleanup.
    function test_isDevFeatureEnabled_superRootGamesMigrationAlwaysEnabled_succeeds() public view {
        assertTrue(l2DevFeatureFlags.isDevFeatureEnabled(DevFeatures.SUPER_ROOT_GAMES_MIGRATION));
    }

    /// @notice Tests that `isDevFeatureEnabled` returns false for zero feature.
    function testFuzz_isDevFeatureEnabled_zeroFeature_succeeds(bytes32 _bitmap) public {
        vm.prank(Constants.DEPOSITOR_ACCOUNT);
        l2DevFeatureFlags.setDevFeatureBitmap(_bitmap);

        assertFalse(l2DevFeatureFlags.isDevFeatureEnabled(bytes32(0)));
    }

    /// @notice Tests that `isDevFeatureEnabled` returns true when the feature bit is set.
    function testFuzz_isDevFeatureEnabledFeatureSet_succeeds(bytes32 _feature) public {
        vm.assume(_feature != bytes32(0));

        vm.prank(Constants.DEPOSITOR_ACCOUNT);
        l2DevFeatureFlags.setDevFeatureBitmap(_feature);

        assertTrue(l2DevFeatureFlags.isDevFeatureEnabled(_feature));
    }

    /// @notice Tests that `isDevFeatureEnabled` works correctly with the known OPTIMISM_PORTAL_INTEROP feature.
    function test_isDevFeatureEnabled_interop_succeeds() public {
        vm.prank(Constants.DEPOSITOR_ACCOUNT);
        l2DevFeatureFlags.setDevFeatureBitmap(DevFeatures.OPTIMISM_PORTAL_INTEROP);

        assertTrue(l2DevFeatureFlags.isDevFeatureEnabled(DevFeatures.OPTIMISM_PORTAL_INTEROP));
        assertFalse(l2DevFeatureFlags.isDevFeatureEnabled(DevFeatures.ZK_DISPUTE_GAME));
    }

    /// @notice Tests that `isDevFeatureEnabled` works correctly with multiple features set.
    function test_isDevFeatureEnabled_multipleFeatures_succeeds() public {
        bytes32 bitmap = DevFeatures.OPTIMISM_PORTAL_INTEROP | DevFeatures.ZK_DISPUTE_GAME;

        vm.prank(Constants.DEPOSITOR_ACCOUNT);
        l2DevFeatureFlags.setDevFeatureBitmap(bitmap);

        assertTrue(l2DevFeatureFlags.isDevFeatureEnabled(DevFeatures.OPTIMISM_PORTAL_INTEROP));
        assertTrue(l2DevFeatureFlags.isDevFeatureEnabled(DevFeatures.ZK_DISPUTE_GAME));
    }
}

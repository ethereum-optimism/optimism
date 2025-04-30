// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Interfaces
import { IProxy } from "interfaces/universal/IProxy.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";

/// @title SuperchainConfig Test Init
/// @notice Test initialization for SuperchainConfig tests.
contract SuperchainConfig_TestInit is CommonTest {
    function setUp() public virtual override {
        super.setUp();
        skipIfForkTest("SuperchainConfig_Init_Test: cannot test initialization on forked network");
    }
}

/// @title SuperchainConfig_Initialize_Test
/// @notice Test contract for SuperchainConfig `initialize` function.
contract SuperchainConfig_Initialize_Test is SuperchainConfig_TestInit {
    /// @notice Tests that initialization sets the correct values. These are defined in
    ///         CommonTest.sol.
    function test_initialize_unpaused_succeeds() external view {
        assertFalse(superchainConfig.paused(address(this)));
        assertEq(superchainConfig.guardian(), deploy.cfg().superchainConfigGuardian());
    }

    /// @notice Tests that it can be intialized as paused.
    function test_initialize_paused_succeeds() external {
        IProxy newProxy = IProxy(
            DeployUtils.create1({
                _name: "Proxy",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxy.__constructor__, (alice)))
            })
        );
        ISuperchainConfig newImpl = ISuperchainConfig(
            DeployUtils.create1({
                _name: "SuperchainConfig",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(ISuperchainConfig.__constructor__, ()))
            })
        );

        vm.startPrank(alice);
        newProxy.upgradeToAndCall(
            address(newImpl), abi.encodeCall(ISuperchainConfig.initialize, (deploy.cfg().superchainConfigGuardian()))
        );

        assertFalse(ISuperchainConfig(address(newProxy)).paused(address(this)));
        assertEq(ISuperchainConfig(address(newProxy)).guardian(), deploy.cfg().superchainConfigGuardian());
    }
}

/// @title SuperchainConfig_PauseExpiry_Test
/// @notice Test contract for SuperchainConfig `pauseExpiry` function.
contract SuperchainConfig_PauseExpiry_Test is SuperchainConfig_TestInit {
    /// @notice Tests that `pauseExpiry` returns the correct constant value.
    function test_pauseExpiry_succeeds() external view {
        assertEq(superchainConfig.pauseExpiry(), 7_884_000);
    }
}

/// @title SuperchainConfig_Pause_Test
/// @notice Test contract for SuperchainConfig `pause` function.
contract SuperchainConfig_Pause_Test is SuperchainConfig_TestInit {
    /// @notice Tests that `pause` reverts when called by a non-guardian.
    function test_pause_notGuardian_reverts() external {
        assertFalse(superchainConfig.paused(address(this)));

        assertTrue(superchainConfig.guardian() != alice);
        vm.expectRevert(ISuperchainConfig.SuperchainConfig_OnlyGuardian.selector);
        vm.prank(alice);
        superchainConfig.pause(address(this));

        assertFalse(superchainConfig.paused(address(this)));
    }

    /// @notice Tests that `pause` reverts when the identifier is already used.
    function test_pause_alreadyUsed_reverts() external {
        vm.startPrank(superchainConfig.guardian());
        superchainConfig.pause(address(this));

        vm.expectRevert(
            abi.encodeWithSelector(ISuperchainConfig.SuperchainConfig_AlreadyPaused.selector, address(this))
        );

        superchainConfig.pause(address(this));
    }

    /// @notice Tests that `pause` successfully pauses when called by the guardian.
    function test_pause_succeeds() external {
        assertFalse(superchainConfig.paused(address(this)));

        vm.expectEmit(address(superchainConfig));
        emit Paused(address(this));

        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(address(this));

        assertTrue(superchainConfig.paused(address(this)));
    }
}

/// @title SuperchainConfig_Unpause_Test
/// @notice Test contract for SuperchainConfig `unpause` function.
contract SuperchainConfig_Unpause_Test is SuperchainConfig_TestInit {
    /// @notice Tests that `unpause` reverts when called by a non-guardian.
    function test_unpause_notGuardian_reverts() external {
        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(address(this));
        assertTrue(superchainConfig.paused(address(this)));

        assertTrue(superchainConfig.guardian() != alice);
        vm.expectRevert(ISuperchainConfig.SuperchainConfig_OnlyGuardian.selector);
        vm.prank(alice);
        superchainConfig.unpause(address(this));

        assertTrue(superchainConfig.paused(address(this)));
    }

    /// @notice Tests that `unpause` successfully unpauses when called by the guardian.
    function test_unpause_succeeds() external {
        vm.startPrank(superchainConfig.guardian());
        superchainConfig.pause(address(this));
        assertTrue(superchainConfig.paused(address(this)));

        vm.expectEmit(address(superchainConfig));
        emit Unpaused(address(this));
        superchainConfig.unpause(address(this));

        assertFalse(superchainConfig.paused(address(this)));
    }
}

/// @title SuperchainConfig_Extend_Test
/// @notice Test contract for SuperchainConfig `extend` function.
contract SuperchainConfig_Extend_Test is SuperchainConfig_TestInit {
    /// @notice Tests that `extend` successfully resets and re-pauses an identifier.
    function test_extend_succeeds() external {
        vm.startPrank(superchainConfig.guardian());
        superchainConfig.pause(address(this));
        uint256 firstPauseTimestamp = block.timestamp;

        vm.warp(block.timestamp + 1);

        superchainConfig.extend(address(this));
        assertTrue(superchainConfig.pauseTimestamps(address(this)) > firstPauseTimestamp);
        assertTrue(superchainConfig.paused(address(this)));
    }

    /// @notice Tests that `extend` reverts when called by a non-guardian.
    function test_extend_notGuardian_reverts() external {
        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(address(this));

        vm.prank(alice);
        vm.expectRevert(ISuperchainConfig.SuperchainConfig_OnlyGuardian.selector);
        superchainConfig.extend(address(this));
    }
}

/// @title SuperchainConfig_Pausable_Test
/// @notice Test contract for SuperchainConfig `pausable` function.
contract SuperchainConfig_Pausable_Test is SuperchainConfig_TestInit {
    /// @notice Tests that `pausable` returns true when the identifier is not paused.
    function test_pausable_notPaused_succeeds() external view {
        assertTrue(superchainConfig.pausable(address(this)));
    }

    /// @notice Tests that `pausable` returns false when the identifier is paused.
    function test_pausable_paused_succeeds() external {
        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(address(this));
        assertFalse(superchainConfig.pausable(address(this)));
    }
}

/// @title SuperchainConfig_Expiration_Test
/// @notice Test contract for SuperchainConfig `expiration` function.
contract SuperchainConfig_Expiration_Test is SuperchainConfig_TestInit {
    /// @notice Tests that `expiration` returns 0 when the identifier is not paused.
    function test_expiration_notPaused_succeeds() external view {
        assertEq(superchainConfig.expiration(address(this)), 0);
    }

    /// @notice Tests that `expiration` returns the correct timestamp when the identifier is
    ///         paused.
    function test_expiration_paused_succeeds() external {
        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(address(this));
        uint256 expectedExpiration = block.timestamp + superchainConfig.pauseExpiry();
        assertEq(superchainConfig.expiration(address(this)), expectedExpiration);
    }

    /// @notice Tests that `expiration` returns the updated timestamp after extending the pause.
    function test_expiration_afterExtend_succeeds() external {
        vm.startPrank(superchainConfig.guardian());
        superchainConfig.pause(address(this));
        uint256 firstExpiration = superchainConfig.expiration(address(this));

        // Warp forward in time
        vm.warp(block.timestamp + 100);

        // Extend the pause
        superchainConfig.extend(address(this));
        uint256 newExpiration = superchainConfig.expiration(address(this));

        assertTrue(newExpiration > firstExpiration);
        assertEq(newExpiration, block.timestamp + superchainConfig.pauseExpiry());
    }
}

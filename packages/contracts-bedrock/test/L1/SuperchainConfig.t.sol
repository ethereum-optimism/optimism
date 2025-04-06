// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "test/setup/CommonTest.sol";

// Target contract dependencies
import { IProxy } from "interfaces/universal/IProxy.sol";

// Target contract
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";

import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

contract SuperchainConfig_Init_Test is CommonTest {
    function setUp() public virtual override {
        super.setUp();
        skipIfForkTest("SuperchainConfig_Init_Test: cannot test initialization on forked network");
    }

    /// @dev Tests that initialization sets the correct values. These are defined in CommonTest.sol.
    function test_initialize_unpaused_succeeds() external view {
        assertFalse(superchainConfig.paused(address(this)));
        assertEq(superchainConfig.guardian(), deploy.cfg().superchainConfigGuardian());
    }

    /// @dev Tests that it can be intialized as paused.
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
            address(newImpl), abi.encodeCall(ISuperchainConfig.initialize, (deploy.cfg().superchainConfigGuardian(), 0))
        );

        assertFalse(ISuperchainConfig(address(newProxy)).paused(address(this)));
        assertEq(ISuperchainConfig(address(newProxy)).guardian(), deploy.cfg().superchainConfigGuardian());
    }
}

contract SuperchainConfig_Pause_TestFail is CommonTest {
    /// @dev Tests that `pause` reverts when called by a non-guardian.
    function test_pause_notGuardian_reverts() external {
        assertFalse(superchainConfig.paused(address(this)));

        assertTrue(superchainConfig.guardian() != alice);
        vm.expectRevert(ISuperchainConfig.OnlyGuardian.selector);
        vm.prank(alice);
        superchainConfig.pause(address(this));

        assertFalse(superchainConfig.paused(address(this)));
    }

    /// @dev Tests that `pause` reverts when the identifier is already used.
    function test_pause_alreadyUsed_reverts() external {
        vm.startPrank(superchainConfig.guardian());
        superchainConfig.pause(address(this));

        vm.expectRevert(ISuperchainConfig.PauseAlreadyUsed.selector);
        superchainConfig.pause(address(this));
    }
}

contract SuperchainConfig_Pause_Test is CommonTest {
    /// @dev Tests that `pause` successfully pauses
    ///      when called by the guardian.
    function test_pause_succeeds() external {
        assertFalse(superchainConfig.paused(address(this)));

        vm.expectEmit(address(superchainConfig));
        emit Paused(string(abi.encodePacked(address(this))));

        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(address(this));

        assertTrue(superchainConfig.paused(address(this)));
    }
}

contract SuperchainConfig_Unpause_TestFail is CommonTest {
    /// @dev Tests that `unpause` reverts when called by a non-guardian.
    function test_unpause_notGuardian_reverts() external {
        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(address(this));
        assertTrue(superchainConfig.paused(address(this)));

        assertTrue(superchainConfig.guardian() != alice);
        vm.expectRevert(ISuperchainConfig.OnlyGuardian.selector);
        vm.prank(alice);
        superchainConfig.unpause(address(this));

        assertTrue(superchainConfig.paused(address(this)));
    }
}

contract SuperchainConfig_Unpause_Test is CommonTest {
    /// @dev Tests that `unpause` successfully unpauses
    ///      when called by the guardian.
    function test_unpause_succeeds() external {
        vm.startPrank(superchainConfig.guardian());
        superchainConfig.pause(address(this));
        assertTrue(superchainConfig.paused(address(this)));

        vm.expectEmit(address(superchainConfig));
        emit Unpaused();
        superchainConfig.unpause(address(this));

        assertFalse(superchainConfig.paused(address(this)));
    }
}

contract SuperchainConfig_Reset_Test is CommonTest {
    /// @dev Tests that `reset` successfully resets a used identifier.
    function test_reset_succeeds() external {
        vm.startPrank(superchainConfig.guardian());
        superchainConfig.pause(address(this));
        assertTrue(superchainConfig.pauseUsed(address(this)));

        superchainConfig.reset(address(this));
        assertFalse(superchainConfig.pauseUsed(address(this)));
    }

    /// @dev Tests that `reset` reverts when called by a non-guardian.
    function test_reset_notGuardian_reverts() external {
        vm.startPrank(superchainConfig.guardian());
        superchainConfig.pause(address(this));

        vm.expectRevert(ISuperchainConfig.OnlyGuardian.selector);
        vm.prank(alice);
        superchainConfig.reset(address(this));
    }
}

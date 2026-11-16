// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Scripts
import { DeployComplianceModule } from "scripts/deploy/compliance/DeployComplianceModule.s.sol";

// Interfaces
import { IProxy } from "interfaces/universal/IProxy.sol";

/// @title DeployComplianceModule_Run_Test
/// @notice Tests the deployment of the ComplianceModule before the chain exists.
contract DeployComplianceModule_Run_Test is Test {
    DeployComplianceModule internal script;

    function setUp() public {
        script = new DeployComplianceModule();
    }

    /// @notice Tests that the broadcaster is the admin of a module proxy that has no
    ///         implementation yet.
    function test_run_succeeds() public {
        DeployComplianceModule.Output memory output = script.run();

        assertGt(address(output.moduleImpl).code.length, 0, "implementation has no code");
        // The OPCM only accepts a hook with code.
        assertGt(address(output.module).code.length, 0, "module has no code");

        vm.startPrank(address(0));
        assertEq(IProxy(payable(address(output.module))).admin(), address(this), "module admin");
        assertEq(IProxy(payable(address(output.module))).implementation(), address(0), "module implementation");
        vm.stopPrank();
    }

    /// @notice Tests that the module rejects calls until it is configured, so a chain that already
    ///         uses it as its hook cannot move value through it.
    function test_run_callBeforeConfigure_reverts() public {
        DeployComplianceModule.Output memory output = script.run();

        vm.prank(makeAddr("portal"));
        vm.expectRevert("Proxy: implementation not initialized");
        output.module.portal();
    }
}

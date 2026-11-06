// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Scripts
import { DeployCompliance } from "scripts/deploy/DeployCompliance.s.sol";

// Interfaces
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IComplianceModule } from "interfaces/compliance/IComplianceModule.sol";
import { IPolicy } from "interfaces/compliance/IPolicy.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";

/// @title DeployCompliance_TestInit
/// @notice Shared setup for the compliance deployment script.
contract DeployCompliance_TestInit is CommonTest {
    DeployCompliance internal deployScript;

    address internal custodyOwner = makeAddr("custodyOwner");
    address internal moduleOwner = makeAddr("moduleOwner");

    function setUp() public virtual override {
        super.setUp();
        deployScript = new DeployCompliance();
    }

    /// @notice The default input, pointed at the test chain's contracts.
    /// @return The input parameters.
    function _input() internal view returns (DeployCompliance.Input memory) {
        return DeployCompliance.Input({
            portal: IOptimismPortal2(payable(address(optimismPortal2))),
            bridge: l1StandardBridge,
            l1CrossDomainMessenger: l1CrossDomainMessenger,
            chainProxyAdminOwner: optimismPortal2.proxyAdminOwner(),
            custodyOwner: custodyOwner,
            moduleOwner: moduleOwner
        });
    }

    /// @notice Runs the script with the default input.
    /// @return The deployed contracts.
    function _run() internal returns (DeployCompliance.Output memory) {
        return deployScript.run(_input());
    }
}

/// @title DeployCompliance_run_Test
/// @notice Covers the happy path and the ownership split it has to produce.
contract DeployCompliance_run_Test is DeployCompliance_TestInit {
    /// @notice Tests that the script wires the module, its policy and the custody admin.
    function test_run_succeeds() public {
        DeployCompliance.Output memory output = _run();

        assertEq(output.custodyProxyAdmin.owner(), custodyOwner, "custody owner mismatch");
        assertEq(output.module.owner(), moduleOwner, "module owner mismatch");
        assertEq(address(output.module.portal()), address(optimismPortal2), "portal mismatch");
        assertEq(address(output.module.bridge()), address(l1StandardBridge), "bridge mismatch");
        assertEq(address(output.module.l1CrossDomainMessenger()), address(l1CrossDomainMessenger), "messenger mismatch");
        assertEq(address(output.module.policy()), address(output.policy), "module policy mismatch");
        assertEq(address(output.policy.module()), address(output.module), "policy module mismatch");
    }

    /// @notice Tests that the module's upgrade authority is not the chain's.
    function test_run_custodyAdminIsSeparate_succeeds() public {
        DeployCompliance.Output memory output = _run();

        assertNotEq(address(output.custodyProxyAdmin), address(proxyAdmin), "custody admin is the chain admin");
        assertNotEq(output.custodyProxyAdmin.owner(), proxyAdmin.owner(), "custody owner is the chain PAO");

        vm.prank(address(0));
        assertEq(
            IProxy(payable(address(output.module))).admin(),
            address(output.custodyProxyAdmin),
            "module not admined by the custody admin"
        );
    }

    /// @notice Tests that the script doesn't touch the Portal.
    function test_run_doesNotTouchThePortal_succeeds() public {
        address hookBefore = address(optimismPortal2.bridgeHook());

        DeployCompliance.Output memory output = _run();

        assertEq(address(optimismPortal2.bridgeHook()), hookBefore, "portal bridge hook changed");
        assertNotEq(address(optimismPortal2.bridgeHook()), address(output.module), "portal points at the module");
    }

    /// @notice Tests that the module is initialized exactly once.
    function test_run_moduleInitializedOnce_succeeds() public {
        DeployCompliance.Output memory output = _run();

        vm.prank(custodyOwner);
        vm.expectRevert("Proxy: delegatecall to new implementation contract failed");
        output.custodyProxyAdmin
            .upgradeAndCall(
                payable(address(output.module)),
                address(output.moduleImpl),
                abi.encodeCall(IComplianceModule.initialize, (makeAddr("attacker"), IPolicy(address(output.policy))))
            );

        assertEq(output.module.owner(), moduleOwner, "module owner changed");
    }

    /// @notice Tests that the script makes no call to the chain's contracts, so it can run against
    ///         predicted addresses before they are deployed.
    function test_run_contractsNotDeployed_succeeds() public {
        DeployCompliance.Input memory input = _input();
        input.portal = IOptimismPortal2(payable(makeAddr("predictedPortal")));
        input.bridge = IL1StandardBridge(payable(makeAddr("predictedBridge")));
        input.l1CrossDomainMessenger = IL1CrossDomainMessenger(makeAddr("predictedMessenger"));
        input.chainProxyAdminOwner = makeAddr("predictedChainPAO");

        DeployCompliance.Output memory output = deployScript.run(input);

        assertEq(address(output.module.portal()), address(input.portal), "portal mismatch");
        assertEq(address(output.module.bridge()), address(input.bridge), "bridge mismatch");
        assertEq(
            address(output.module.l1CrossDomainMessenger()), address(input.l1CrossDomainMessenger), "messenger mismatch"
        );
    }

    /// @notice Tests that running the script twice yields independent deployments, since the proxy
    ///         and the custody admin are not deterministic.
    function test_run_twice_succeeds() public {
        DeployCompliance.Output memory first = _run();
        DeployCompliance.Output memory second = _run();

        assertNotEq(address(first.module), address(second.module), "module proxies collide");
        assertNotEq(address(first.custodyProxyAdmin), address(second.custodyProxyAdmin), "custody admins collide");
    }
}

/// @title DeployCompliance_assertValidInput_Test
/// @notice Covers the input guards.
contract DeployCompliance_assertValidInput_Test is DeployCompliance_TestInit {
    /// @notice Tests that a zero Portal is rejected.
    function test_assertValidInput_zeroPortal_reverts() public {
        DeployCompliance.Input memory input = _input();
        input.portal = IOptimismPortal2(payable(address(0)));

        vm.expectRevert("DeployCompliance: portal not set");
        deployScript.assertValidInput(input);
    }

    /// @notice Tests that a zero bridge is rejected.
    function test_assertValidInput_zeroBridge_reverts() public {
        DeployCompliance.Input memory input = _input();
        input.bridge = IL1StandardBridge(payable(address(0)));

        vm.expectRevert("DeployCompliance: bridge not set");
        deployScript.assertValidInput(input);
    }

    /// @notice Tests that a zero messenger is rejected.
    function test_assertValidInput_zeroMessenger_reverts() public {
        DeployCompliance.Input memory input = _input();
        input.l1CrossDomainMessenger = IL1CrossDomainMessenger(address(0));

        vm.expectRevert("DeployCompliance: l1CrossDomainMessenger not set");
        deployScript.assertValidInput(input);
    }

    /// @notice Tests that a zero chain ProxyAdmin owner is rejected.
    function test_assertValidInput_zeroChainProxyAdminOwner_reverts() public {
        DeployCompliance.Input memory input = _input();
        input.chainProxyAdminOwner = address(0);

        vm.expectRevert("DeployCompliance: chainProxyAdminOwner not set");
        deployScript.assertValidInput(input);
    }

    /// @notice Tests that a zero custody owner is rejected.
    function test_assertValidInput_zeroCustodyOwner_reverts() public {
        DeployCompliance.Input memory input = _input();
        input.custodyOwner = address(0);

        vm.expectRevert("DeployCompliance: custodyOwner not set");
        deployScript.assertValidInput(input);
    }

    /// @notice Tests that a zero module owner is rejected.
    function test_assertValidInput_zeroModuleOwner_reverts() public {
        DeployCompliance.Input memory input = _input();
        input.moduleOwner = address(0);

        vm.expectRevert("DeployCompliance: moduleOwner not set");
        deployScript.assertValidInput(input);
    }

    /// @notice Tests that the custody owner cannot be the chain's ProxyAdmin owner.
    function test_assertValidInput_custodyOwnerIsChainPAO_reverts() public {
        DeployCompliance.Input memory input = _input();
        input.custodyOwner = input.chainProxyAdminOwner;

        vm.expectRevert("DeployCompliance: custodyOwner must not be the chain ProxyAdmin owner");
        deployScript.assertValidInput(input);
    }
}

/// @title DeployCompliance_assertValidOutput_Test
contract DeployCompliance_assertValidOutput_Test is DeployCompliance_TestInit {
    /// @notice Tests that a module proxy whose implementation is not the output's is rejected.
    function test_assertValidOutput_implementationMismatch_reverts() public {
        DeployCompliance.Input memory input = _input();
        DeployCompliance.Output memory output = deployScript.run(input);
        output.moduleImpl = IComplianceModule(makeAddr("otherModuleImpl"));

        vm.expectRevert("DeployCompliance: module implementation mismatch");
        deployScript.assertValidOutput(input, output);
    }
}

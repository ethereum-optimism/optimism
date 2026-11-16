// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Scripts
import { ConfigureComplianceModule } from "scripts/deploy/compliance/ConfigureComplianceModule.s.sol";
import { DeployComplianceModule } from "scripts/deploy/compliance/DeployComplianceModule.s.sol";

// Libraries
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Interfaces
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IComplianceModule } from "interfaces/compliance/IComplianceModule.sol";
import { IVisionPolicyV1 } from "interfaces/compliance/IVisionPolicyV1.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";

/// @title ConfigureComplianceModule_TestInit
/// @notice Shared setup: a module proxy whose admin is this contract, which acts as the temporary
///         admin, and the test chain's Portal using it as its hook.
contract ConfigureComplianceModule_TestInit is CommonTest {
    ConfigureComplianceModule internal script;

    IComplianceModule internal module;

    address internal custodyAdmin = makeAddr("custodyAdmin");
    address internal moduleOwner = makeAddr("moduleOwner");
    address internal policyOwner = makeAddr("policyOwner");
    address internal verdictWriter = makeAddr("verdictWriter");
    address internal allowed = makeAddr("allowed");
    address internal denied = makeAddr("denied");

    function setUp() public virtual override {
        super.setUp();
        script = new ConfigureComplianceModule();

        module = IComplianceModule(
            DeployUtils.create1({
                _name: "Proxy",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxy.__constructor__, (address(this))))
            })
        );
        _useAsBridgeHook(address(module));
    }

    /// @notice Makes the test chain's Portal report `_hook` as its bridge hook.
    function _useAsBridgeHook(address _hook) internal {
        vm.mockCall(address(optimismPortal2), abi.encodeCall(IOptimismPortal2.bridgeHook, ()), abi.encode(_hook));
    }

    /// @notice Returns the module proxy's admin.
    function _adminOf(IComplianceModule _module) internal returns (address) {
        vm.prank(address(0));
        return IProxy(payable(address(_module))).admin();
    }

    /// @notice A valid input. Any contract passes the implementation's code check, and the input
    ///         tests revert before it is used.
    function _input() internal view returns (ConfigureComplianceModule.Input memory input_) {
        input_.module = module;
        input_.moduleImpl = IComplianceModule(address(optimismPortal2));
        input_.portal = IOptimismPortal2(payable(address(optimismPortal2)));
        input_.custodyAdmin = custodyAdmin;
        input_.moduleOwner = moduleOwner;
        input_.policyOwner = policyOwner;
        input_.verdictWriter = verdictWriter;
        input_.maxVerdictValidity = 1 days;
        // Within VisionPolicyV1's cap and below the test Portal's proof maturity delay.
        input_.holdDelay = 1 hours;
        input_.allowlist = new address[](1);
        input_.allowlist[0] = allowed;
        input_.denylist = new address[](1);
        input_.denylist[0] = denied;
    }

    /// @notice Deploys the module with DeployComplianceModule, points the Portal at it and
    ///         configures it.
    function _deployAndConfigure()
        internal
        returns (ConfigureComplianceModule.Input memory input_, IVisionPolicyV1 policy_)
    {
        DeployComplianceModule.Output memory deployed = new DeployComplianceModule().run();
        _useAsBridgeHook(address(deployed.module));

        input_ = _input();
        input_.module = deployed.module;
        input_.moduleImpl = deployed.moduleImpl;
        policy_ = script.run(input_);
    }
}

/// @title ConfigureComplianceModule_Run_Test
/// @notice Tests the configuration and handover of the ComplianceModule.
contract ConfigureComplianceModule_Run_Test is ConfigureComplianceModule_TestInit {
    /// @notice Tests the full flow: deploy before the chain, then configure and hand over.
    function test_run_succeeds() public {
        (ConfigureComplianceModule.Input memory input, IVisionPolicyV1 policy) = _deployAndConfigure();
        IComplianceModule deployed = input.module;

        assertEq(_adminOf(deployed), custodyAdmin, "module admin");
        assertEq(deployed.owner(), moduleOwner, "module owner");
        assertEq(address(deployed.portal()), address(optimismPortal2), "module portal");
        assertEq(address(deployed.l1CrossDomainMessenger()), address(l1CrossDomainMessenger), "module messenger");
        assertEq(address(deployed.bridge()), address(l1StandardBridge), "module bridge");
        assertEq(address(deployed.policy()), address(policy), "module policy");

        assertEq(address(policy.module()), address(deployed), "policy module");
        assertEq(policy.owner(), policyOwner, "policy owner");
        assertEq(policy.verdictWriter(), verdictWriter, "policy verdict writer");
        assertEq(policy.maxVerdictValidity(), input.maxVerdictValidity, "policy max verdict validity");
        assertEq(policy.holdDelay(), input.holdDelay, "policy hold delay");
        assertTrue(policy.allowlisted(allowed), "allowlist");
        assertTrue(policy.denylisted(denied), "denylist");
    }

    /// @notice Tests that the custody admin can upgrade the module.
    function test_run_custodyAdminUpgrades_succeeds() public {
        (ConfigureComplianceModule.Input memory input,) = _deployAndConfigure();
        address newImpl = DeployUtils.create1({
            _name: "ComplianceModule",
            _args: DeployUtils.encodeConstructor(abi.encodeCall(IComplianceModule.__constructor__, ()))
        });

        vm.prank(custodyAdmin);
        IProxy(payable(address(input.module))).upgradeTo(newImpl);

        vm.prank(address(0));
        assertEq(IProxy(payable(address(input.module))).implementation(), newImpl, "module implementation");
        // Storage is kept across the upgrade.
        assertEq(address(input.module.portal()), address(optimismPortal2), "module portal");
    }

    /// @notice Tests that a module the chain does not use as its hook is rejected.
    function test_run_notBridgeHook_reverts() public {
        _useAsBridgeHook(address(0));
        vm.expectRevert("ConfigureComplianceModule: portal does not use the module as its bridge hook");
        script.run(_input());
    }

    /// @notice Tests that only the module proxy's admin can configure the module.
    function test_run_broadcasterNotModuleAdmin_reverts() public {
        IProxy(payable(address(module))).changeAdmin(custodyAdmin);
        vm.expectRevert("ConfigureComplianceModule: broadcaster is not the module admin");
        script.run(_input());
    }

    /// @notice Tests that the custody admin cannot be the chain's ProxyAdmin owner.
    function test_run_custodyAdminIsChainProxyAdminOwner_reverts() public {
        ConfigureComplianceModule.Input memory input = _input();
        input.custodyAdmin = optimismPortal2.proxyAdminOwner();
        vm.expectRevert("ConfigureComplianceModule: custodyAdmin must not be the chain ProxyAdmin owner");
        script.run(input);
    }

    /// @notice Tests that the custody admin cannot be the chain's ProxyAdmin, whose owner could
    ///         then upgrade the module through it.
    function test_run_custodyAdminIsChainProxyAdmin_reverts() public {
        ConfigureComplianceModule.Input memory input = _input();
        input.custodyAdmin = address(optimismPortal2.proxyAdmin());
        vm.expectRevert("ConfigureComplianceModule: custodyAdmin must not be the chain ProxyAdmin");
        script.run(input);
    }

    /// @notice Tests that custody cannot stay with the temporary key.
    function test_run_custodyAdminIsBroadcaster_reverts() public {
        ConfigureComplianceModule.Input memory input = _input();
        input.custodyAdmin = address(this);
        vm.expectRevert("ConfigureComplianceModule: custodyAdmin must not be the broadcaster");
        script.run(input);
    }

    /// @notice Tests that the module cannot stay with the temporary key.
    function test_run_moduleOwnerIsBroadcaster_reverts() public {
        ConfigureComplianceModule.Input memory input = _input();
        input.moduleOwner = address(this);
        vm.expectRevert("ConfigureComplianceModule: moduleOwner must not be the broadcaster");
        script.run(input);
    }

    /// @notice Tests that every role must be set.
    function test_run_zeroRole_reverts() public {
        ConfigureComplianceModule.Input memory input = _input();
        input.custodyAdmin = address(0);
        vm.expectRevert("ConfigureComplianceModule: custodyAdmin not set");
        script.run(input);

        input = _input();
        input.moduleOwner = address(0);
        vm.expectRevert("ConfigureComplianceModule: moduleOwner not set");
        script.run(input);

        input = _input();
        input.policyOwner = address(0);
        vm.expectRevert("ConfigureComplianceModule: policyOwner not set");
        script.run(input);

        input = _input();
        input.verdictWriter = address(0);
        vm.expectRevert("ConfigureComplianceModule: verdictWriter not set");
        script.run(input);
    }

    /// @notice Tests that a hold delay at or above the Portal's proof maturity delay is rejected.
    function test_run_holdDelayNotBelowProofMaturity_reverts() public {
        ConfigureComplianceModule.Input memory input = _input();
        vm.mockCall(
            address(optimismPortal2),
            abi.encodeCall(IOptimismPortal2.proofMaturityDelaySeconds, ()),
            abi.encode(uint256(input.holdDelay))
        );
        vm.expectRevert("ConfigureComplianceModule: holdDelay must be below the Portal's proof maturity delay");
        script.run(input);
    }

    /// @notice Tests that an implementation without code is rejected.
    function test_run_implementationWithoutCode_reverts() public {
        ConfigureComplianceModule.Input memory input = _input();
        input.moduleImpl = IComplianceModule(makeAddr("noCode"));
        vm.expectRevert("ConfigureComplianceModule: moduleImpl has no code");
        script.run(input);
    }
}

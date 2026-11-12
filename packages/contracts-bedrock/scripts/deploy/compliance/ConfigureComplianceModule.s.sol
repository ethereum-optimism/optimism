// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Scripts
import { Script } from "forge-std/Script.sol";
import { console2 as console } from "forge-std/console2.sol";

// Libraries
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Interfaces
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IComplianceModule } from "interfaces/compliance/IComplianceModule.sol";
import { IPolicy } from "interfaces/compliance/IPolicy.sol";
import { IVisionPolicyV1 } from "interfaces/compliance/IVisionPolicyV1.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";

/// @title ConfigureComplianceModule
/// @notice Run by the temporary admin from DeployComplianceModule once the chain uses the module
///         as its bridge hook. Deploys the policy, initializes the module with its final owner,
///         the policy and the chain's addresses, then hands the proxy to its final admin. Until it
///         runs, the Portal's deposits and withdrawal finalizations revert, so on an existing chain
///         run it right after the upgrade that sets the hook.
contract ConfigureComplianceModule is Script {
    /// @notice Input for the configuration.
    /// @custom:field module             The ComplianceModule proxy, the chain's bridge hook.
    /// @custom:field moduleImpl         The ComplianceModule implementation.
    /// @custom:field portal             The chain's OptimismPortal. The bridge and messenger are
    ///                                  read from its SystemConfig.
    /// @custom:field custodyAdmin       Final admin of the module proxy, so the module's upgrade
    ///                                  authority. Must not be the chain's ProxyAdmin owner.
    /// @custom:field moduleOwner        Final owner of the module, who sets its policy.
    /// @custom:field policyOwner        Owner of the policy.
    /// @custom:field verdictWriter      Address that records verdicts on the policy.
    /// @custom:field maxVerdictValidity Longest validity the policy accepts for a verdict.
    /// @custom:field holdDelay          Delay before the policy lets an item be held.
    /// @custom:field allowlist          Accounts the policy allows initially.
    /// @custom:field denylist           Accounts the policy denies initially.
    struct Input {
        IComplianceModule module;
        IComplianceModule moduleImpl;
        IOptimismPortal2 portal;
        address custodyAdmin;
        address moduleOwner;
        address policyOwner;
        address verdictWriter;
        uint64 maxVerdictValidity;
        uint64 holdDelay;
        address[] allowlist;
        address[] denylist;
    }

    /// @notice ERC-7201 slot of OpenZeppelin v5's Initializable storage, which ComplianceModule uses.
    bytes32 internal constant OZ_V5_INITIALIZABLE_SLOT =
        0xf0c57e16840df040f15088dc2f81fe391c3923bec73e23a9662efc9c229c6a00;

    /// @notice Configures the module and hands over ownership.
    /// @param _input The input parameters.
    /// @return policy_ The deployed policy.
    function run(Input memory _input) public returns (IVisionPolicyV1 policy_) {
        assertValidInput(_input);

        ISystemConfig systemConfig = _input.portal.systemConfig();
        IL1StandardBridge bridge = IL1StandardBridge(payable(systemConfig.l1StandardBridge()));
        IL1CrossDomainMessenger messenger = IL1CrossDomainMessenger(systemConfig.l1CrossDomainMessenger());
        IProxy proxy = IProxy(payable(address(_input.module)));

        // Deployed first because initialize requires the policy.
        vm.broadcast(msg.sender);
        policy_ = IVisionPolicyV1(
            DeployUtils.create1({
                _name: "VisionPolicyV1",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(
                        IVisionPolicyV1.__constructor__,
                        (
                            _input.module,
                            _input.policyOwner,
                            _input.verdictWriter,
                            _input.maxVerdictValidity,
                            _input.holdDelay,
                            _input.allowlist,
                            _input.denylist
                        )
                    )
                )
            })
        );
        vm.label(address(policy_), "VisionPolicyV1");

        vm.broadcast(msg.sender);
        proxy.upgradeToAndCall(
            address(_input.moduleImpl),
            abi.encodeCall(
                IComplianceModule.initialize,
                (_input.moduleOwner, IPolicy(address(policy_)), _input.portal, messenger, bridge)
            )
        );

        vm.broadcast(msg.sender);
        proxy.changeAdmin(_input.custodyAdmin);

        assertValidOutput(_input, policy_, messenger, bridge);

        console.log("VisionPolicyV1:", address(policy_));
    }

    /// @notice Asserts that the broadcaster controls the unconfigured module, that the chain uses
    ///         it as its hook, and that custody stays separate from the chain's upgrade authority.
    /// @param _input The input parameters.
    function assertValidInput(Input memory _input) public {
        require(address(_input.module) != address(0), "ConfigureComplianceModule: module not set");
        require(address(_input.moduleImpl).code.length > 0, "ConfigureComplianceModule: moduleImpl has no code");
        require(address(_input.portal) != address(0), "ConfigureComplianceModule: portal not set");
        require(_input.custodyAdmin != address(0), "ConfigureComplianceModule: custodyAdmin not set");
        require(_input.moduleOwner != address(0), "ConfigureComplianceModule: moduleOwner not set");
        require(_input.policyOwner != address(0), "ConfigureComplianceModule: policyOwner not set");
        require(_input.verdictWriter != address(0), "ConfigureComplianceModule: verdictWriter not set");

        vm.prank(address(0));
        require(
            IProxy(payable(address(_input.module))).admin() == msg.sender,
            "ConfigureComplianceModule: broadcaster is not the module admin"
        );
        require(
            address(_input.portal.bridgeHook()) == address(_input.module),
            "ConfigureComplianceModule: portal does not use the module as its bridge hook"
        );

        // The custody admin can take all held value, so it must be neither the chain's upgrade
        // authority, directly or through its ProxyAdmin, nor the temporary key.
        require(
            _input.custodyAdmin != _input.portal.proxyAdminOwner(),
            "ConfigureComplianceModule: custodyAdmin must not be the chain ProxyAdmin owner"
        );
        require(
            _input.custodyAdmin != address(_input.portal.proxyAdmin()),
            "ConfigureComplianceModule: custodyAdmin must not be the chain ProxyAdmin"
        );
        require(
            _input.custodyAdmin != msg.sender, "ConfigureComplianceModule: custodyAdmin must not be the broadcaster"
        );
        require(_input.moduleOwner != msg.sender, "ConfigureComplianceModule: moduleOwner must not be the broadcaster");

        // A withdrawal without a verdict must become holdable before it could be finalized.
        require(
            _input.holdDelay < _input.portal.proofMaturityDelaySeconds(),
            "ConfigureComplianceModule: holdDelay must be below the Portal's proof maturity delay"
        );
    }

    /// @notice Asserts that the module is initialized and wired to the chain and the policy, and
    ///         that ownership has been handed over.
    /// @param _input     The input parameters.
    /// @param _policy    The deployed policy.
    /// @param _messenger The chain's L1CrossDomainMessenger.
    /// @param _bridge    The chain's L1StandardBridge.
    function assertValidOutput(
        Input memory _input,
        IVisionPolicyV1 _policy,
        IL1CrossDomainMessenger _messenger,
        IL1StandardBridge _bridge
    )
        public
    {
        vm.prank(address(0));
        require(
            IProxy(payable(address(_input.module))).admin() == _input.custodyAdmin,
            "ConfigureComplianceModule: module admin mismatch"
        );
        require(
            DeployUtils.assertERC1967ImplementationSet(address(_input.module)) == address(_input.moduleImpl),
            "ConfigureComplianceModule: module implementation mismatch"
        );
        DeployUtils.assertInitialized({
            _contractAddress: address(_input.module),
            _isProxy: true,
            _slot: uint256(OZ_V5_INITIALIZABLE_SLOT),
            _offset: 0
        });

        IComplianceModule module = _input.module;
        require(address(module.portal()) == address(_input.portal), "ConfigureComplianceModule: module portal mismatch");
        require(
            address(module.l1CrossDomainMessenger()) == address(_messenger),
            "ConfigureComplianceModule: module messenger mismatch"
        );
        require(address(module.bridge()) == address(_bridge), "ConfigureComplianceModule: module bridge mismatch");
        require(address(module.policy()) == address(_policy), "ConfigureComplianceModule: module policy mismatch");
        require(module.owner() == _input.moduleOwner, "ConfigureComplianceModule: module owner mismatch");

        require(address(_policy.module()) == address(module), "ConfigureComplianceModule: policy module mismatch");
        require(_policy.owner() == _input.policyOwner, "ConfigureComplianceModule: policy owner mismatch");
    }
}

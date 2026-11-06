// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

// Libraries
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Interfaces
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IComplianceModule } from "interfaces/compliance/IComplianceModule.sol";
import { IPolicy } from "interfaces/compliance/IPolicy.sol";
import { IVisionPolicyV1 } from "interfaces/compliance/IVisionPolicyV1.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";

/// @title DeployCompliance
/// @notice Deploys a chain's ComplianceModule and Policy behind a custody ProxyAdmin that
///         is deliberately separate from the chain's own ProxyAdmin. Whoever can upgrade the module
///         is the custodian of all deferred value.
contract DeployCompliance is Script {
    /// @notice Input for the deployment.
    /// @custom:field portal                 The OptimismPortal the module will be the hook for.
    /// @custom:field bridge                 The chain's L1StandardBridge.
    /// @custom:field l1CrossDomainMessenger The chain's L1CrossDomainMessenger.
    /// @custom:field chainProxyAdminOwner   The chain's ProxyAdmin owner, which the custody owner
    ///                                      must not be.
    /// @custom:field custodyOwner           The owner of the custody ProxyAdmin, and so the
    ///                                      module's upgrade authority and the custodian of all
    ///                                      deferred value.
    /// @custom:field moduleOwner            The address that can set the module's policy and
    ///                                      disable or re-enable screening.
    struct Input {
        IOptimismPortal2 portal;
        IL1StandardBridge bridge;
        IL1CrossDomainMessenger l1CrossDomainMessenger;
        address chainProxyAdminOwner;
        address custodyOwner;
        address moduleOwner;
    }

    /// @notice Output of the deployment.
    /// @custom:field custodyProxyAdmin The ProxyAdmin administering the module's proxy.
    /// @custom:field moduleImpl        The ComplianceModule implementation.
    /// @custom:field module            The ComplianceModule proxy.
    /// @custom:field policy            The VisionPolicyV1 the module consults.
    struct Output {
        IProxyAdmin custodyProxyAdmin;
        IComplianceModule moduleImpl;
        IComplianceModule module;
        IVisionPolicyV1 policy;
    }

    /// @notice Salt used for deterministic deployments.
    bytes32 internal _salt = DeployUtils.DEFAULT_SALT;

    /// @notice Runs the deployment.
    /// @param _input The input parameters.
    /// @return output_ The deployed contracts.
    function run(Input memory _input) public returns (Output memory output_) {
        assertValidInput(_input);
        deployCustodyProxyAdmin(output_);
        deployComplianceModule(_input, output_);
        deployVisionPolicyV1(output_);
        initializeComplianceModule(_input, output_);
        transferCustodyOwnership(_input, output_);
        assertValidOutput(_input, output_);
    }

    /// @notice Runs the deployment from ABI-encoded input.
    /// @param _input The ABI-encoded input parameters.
    /// @return The ABI-encoded output.
    function runWithBytes(bytes memory _input) public returns (bytes memory) {
        Input memory input = abi.decode(_input, (Input));
        Output memory output = run(input);
        return abi.encode(output);
    }

    /// @notice Deploys the custody ProxyAdmin, owned by the broadcaster while the module is wired.
    /// @param _output The output parameters.
    function deployCustodyProxyAdmin(Output memory _output) public {
        vm.broadcast(msg.sender);
        _output.custodyProxyAdmin = IProxyAdmin(
            DeployUtils.create1({
                _name: "ProxyAdmin",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxyAdmin.__constructor__, (msg.sender)))
            })
        );
        vm.label(address(_output.custodyProxyAdmin), "CustodyProxyAdmin");
    }

    /// @notice Deploys the module's implementation and its proxy.
    /// @param _input  The input parameters.
    /// @param _output The output parameters.
    function deployComplianceModule(Input memory _input, Output memory _output) public {
        _output.moduleImpl = IComplianceModule(
            DeployUtils.createDeterministic({
                _name: "ComplianceModule",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(
                        IComplianceModule.__constructor__, (_input.portal, _input.bridge, _input.l1CrossDomainMessenger)
                    )
                ),
                _salt: _salt
            })
        );
        vm.label(address(_output.moduleImpl), "ComplianceModuleImpl");

        vm.broadcast(msg.sender);
        _output.module = IComplianceModule(
            DeployUtils.create1({
                _name: "Proxy",
                _args: DeployUtils.encodeConstructor(
                    abi.encodeCall(IProxy.__constructor__, (address(_output.custodyProxyAdmin)))
                )
            })
        );
        vm.label(address(_output.module), "ComplianceModuleProxy");
    }

    /// @notice Deploys the policy against the module's proxy address.
    /// @param _output The output parameters.
    function deployVisionPolicyV1(Output memory _output) public {
        _output.policy = IVisionPolicyV1(
            DeployUtils.createDeterministic({
                _name: "VisionPolicyV1",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IVisionPolicyV1.__constructor__, (_output.module))),
                _salt: _salt
            })
        );
        vm.label(address(_output.policy), "VisionPolicyV1");
    }

    /// @notice Initializes the module with its owner and its policy in one call.
    /// @param _input  The input parameters.
    /// @param _output The output parameters.
    function initializeComplianceModule(Input memory _input, Output memory _output) public {
        vm.broadcast(msg.sender);
        _output.custodyProxyAdmin
            .upgradeAndCall(
                payable(address(_output.module)),
                address(_output.moduleImpl),
                abi.encodeCall(IComplianceModule.initialize, (_input.moduleOwner, IPolicy(address(_output.policy))))
            );
    }

    /// @notice Hands the custody ProxyAdmin to its real owner.
    /// @param _input  The input parameters.
    /// @param _output The output parameters.
    function transferCustodyOwnership(Input memory _input, Output memory _output) public {
        vm.broadcast(msg.sender);
        _output.custodyProxyAdmin.transferOwnership(_input.custodyOwner);
    }

    /// @notice Asserts that the input is valid.
    /// @param _input The input parameters.
    function assertValidInput(Input memory _input) public pure {
        require(address(_input.portal) != address(0), "DeployCompliance: portal not set");
        require(address(_input.bridge) != address(0), "DeployCompliance: bridge not set");
        require(
            address(_input.l1CrossDomainMessenger) != address(0), "DeployCompliance: l1CrossDomainMessenger not set"
        );
        require(_input.chainProxyAdminOwner != address(0), "DeployCompliance: chainProxyAdminOwner not set");
        require(_input.custodyOwner != address(0), "DeployCompliance: custodyOwner not set");
        require(_input.moduleOwner != address(0), "DeployCompliance: moduleOwner not set");

        require(
            _input.custodyOwner != _input.chainProxyAdminOwner,
            "DeployCompliance: custodyOwner must not be the chain ProxyAdmin owner"
        );
    }

    /// @notice Asserts that the deployment is wired correctly.
    /// @param _input  The input parameters.
    /// @param _output The output parameters.
    function assertValidOutput(Input memory _input, Output memory _output) public {
        require(_output.custodyProxyAdmin.owner() == _input.custodyOwner, "DeployCompliance: custody owner mismatch");
        vm.prank(address(0));
        require(
            IProxy(payable(address(_output.module))).admin() == address(_output.custodyProxyAdmin),
            "DeployCompliance: module admin mismatch"
        );
        require(
            DeployUtils.assertERC1967ImplementationSet(address(_output.module)) == address(_output.moduleImpl),
            "DeployCompliance: module implementation mismatch"
        );
        DeployUtils.assertInitialized({
            _contractAddress: address(_output.module), _isProxy: true, _slot: 0, _offset: 0
        });

        require(address(_output.module.portal()) == address(_input.portal), "DeployCompliance: portal mismatch");
        require(address(_output.module.bridge()) == address(_input.bridge), "DeployCompliance: bridge mismatch");
        require(
            address(_output.module.l1CrossDomainMessenger()) == address(_input.l1CrossDomainMessenger),
            "DeployCompliance: messenger mismatch"
        );

        // The module and the policy point to each other.
        require(_output.module.owner() == _input.moduleOwner, "DeployCompliance: module owner mismatch");
        require(address(_output.module.policy()) == address(_output.policy), "DeployCompliance: module policy mismatch");
        require(address(_output.policy.module()) == address(_output.module), "DeployCompliance: policy module mismatch");
    }
}

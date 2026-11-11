// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Scripts
import { Script } from "forge-std/Script.sol";
import { console2 as console } from "forge-std/console2.sol";

// Libraries
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Interfaces
import { IComplianceModule } from "interfaces/compliance/IComplianceModule.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";

/// @title DeployComplianceModule
/// @notice Deploys a ComplianceModule proxy and its implementation, with the deployer as the
///         proxy's admin for now. The proxy is the address to use as the chain's bridge hook. It
///         points at no implementation until ConfigureComplianceModule initializes it with the
///         chain's addresses, so any call into it reverts until then.
///         The proxy's admin can upgrade the module, so it is the custodian of all held value.
///         ConfigureComplianceModule hands it to its final address; no ProxyAdmin contract is
///         involved.
contract DeployComplianceModule is Script {
    /// @notice Output of the deployment.
    /// @custom:field moduleImpl The ComplianceModule implementation.
    /// @custom:field module     The ComplianceModule proxy, to use as the bridge hook.
    struct Output {
        IComplianceModule moduleImpl;
        IComplianceModule module;
    }

    /// @notice Deploys the contracts with the broadcaster as the proxy's admin.
    /// @return output_ The deployed contracts.
    function run() public returns (Output memory output_) {
        vm.broadcast(msg.sender);
        output_.moduleImpl = IComplianceModule(
            DeployUtils.create1({
                _name: "ComplianceModule",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IComplianceModule.__constructor__, ()))
            })
        );
        vm.label(address(output_.moduleImpl), "ComplianceModuleImpl");

        vm.broadcast(msg.sender);
        output_.module = IComplianceModule(
            DeployUtils.create1({
                _name: "Proxy",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxy.__constructor__, (msg.sender)))
            })
        );
        vm.label(address(output_.module), "ComplianceModuleProxy");

        assertValidOutput(output_);

        console.log("ComplianceModule implementation:", address(output_.moduleImpl));
        console.log("ComplianceModule proxy (bridge hook):", address(output_.module));
    }

    /// @notice Asserts that the broadcaster is the proxy's admin and the implementation exists.
    /// @param _output The deployed contracts.
    function assertValidOutput(Output memory _output) public {
        require(address(_output.moduleImpl).code.length > 0, "DeployComplianceModule: implementation has no code");

        vm.prank(address(0));
        require(
            IProxy(payable(address(_output.module))).admin() == msg.sender,
            "DeployComplianceModule: module admin mismatch"
        );
    }
}

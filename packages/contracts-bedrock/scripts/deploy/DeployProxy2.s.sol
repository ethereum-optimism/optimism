// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Forge
import { Script } from "forge-std/Script.sol";

// Libraries
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Interfaces
import { IProxy } from "interfaces/universal/IProxy.sol";

/// @title DeployProxy
contract DeployProxy2 is Script {
    struct Input {
        address owner;
    }

    struct Output {
        IProxy proxy;
    }

    function run(Input memory _input) public returns (Output memory output_) {
        assertValidInput(_input);

        deployProxySingleton(_input, output_);

        assertValidOutput(_input, output_);
    }

    function deployProxySingleton(Input memory _input, Output memory _output) internal {
        vm.broadcast(msg.sender);
        IProxy proxy = IProxy(
            DeployUtils.create1({
                _name: "Proxy",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxy.__constructor__, (_input.owner)))
            })
        );

        vm.label(address(proxy), "Proxy");
        _output.proxy = proxy;
    }

    function assertValidInput(Input memory _input) internal pure {
        require(_input.owner != address(0), "DeployProxy: owner not set");
    }

    function assertValidOutput(Input memory _input, Output memory _output) internal {
        IProxy proxy = _output.proxy;
        DeployUtils.assertValidContractAddress(address(proxy));

        vm.prank(_input.owner);
        address proxyAdmin = proxy.admin();

        require(
            proxyAdmin == _input.owner, "DeployProxy: owner of proxy does not match the owner specified in the input"
        );
    }
}

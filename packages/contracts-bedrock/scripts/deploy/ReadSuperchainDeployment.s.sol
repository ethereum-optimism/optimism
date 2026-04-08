// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

contract ReadSuperchainDeployment is Script {
    struct Input {
        // Deprecated v1 field — kept for Go ABI compatibility, ignored.
        // Remove when Go op-deployer drops v1 struct fields.
        address opcmAddress;
        ISuperchainConfig superchainConfigProxy;
    }

    struct Output {
        // Deprecated v1 ProtocolVersions fields — kept for Go ABI compatibility, always zero.
        // Remove when Go op-deployer drops v1 struct fields.
        address protocolVersionsImpl;
        address protocolVersionsProxy;
        address protocolVersionsOwner;
        bytes32 recommendedProtocolVersion;
        bytes32 requiredProtocolVersion;
        ISuperchainConfig superchainConfigImpl;
        ISuperchainConfig superchainConfigProxy;
        IProxyAdmin superchainProxyAdmin;
        address guardian;
        address superchainProxyAdminOwner;
    }

    function run(Input memory _input) public returns (Output memory output_) {
        require(
            address(_input.superchainConfigProxy).code.length > 0,
            "ReadSuperchainDeployment: superchainConfigProxy has no code"
        );

        output_.superchainConfigProxy = _input.superchainConfigProxy;
        output_.superchainProxyAdmin = IProxyAdmin(EIP1967Helper.getAdmin(address(output_.superchainConfigProxy)));

        IProxy superchainConfigProxy = IProxy(payable(address(output_.superchainConfigProxy)));

        vm.startPrank(address(0));
        output_.superchainConfigImpl = ISuperchainConfig(superchainConfigProxy.implementation());
        vm.stopPrank();

        output_.guardian = output_.superchainConfigProxy.guardian();
        output_.superchainProxyAdminOwner = output_.superchainProxyAdmin.owner();
    }

    function runWithBytes(bytes memory _input) public returns (bytes memory) {
        Input memory input = abi.decode(_input, (Input));
        Output memory output = run(input);
        return abi.encode(output);
    }
}

// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IProtocolVersions, ProtocolVersion } from "interfaces/L1/IProtocolVersions.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";

contract ReadSuperchainDeployment is Script {
    struct Input {
        IOPContractsManager opcmAddress;
    }

    struct Output {
        IProtocolVersions protocolVersionsImpl;
        IProtocolVersions protocolVersionsProxy;
        ISuperchainConfig superchainConfigImpl;
        ISuperchainConfig superchainConfigProxy;
        IProxyAdmin superchainProxyAdmin;
        address guardian;
        address protocolVersionsOwner;
        address superchainProxyAdminOwner;
        bytes32 recommendedProtocolVersion;
        bytes32 requiredProtocolVersion;
    }

    function run(Input memory _input) public returns (Output memory output_) {
        require(address(_input.opcmAddress) != address(0), "ReadSuperchainDeployment: opcmAddress not set");

        IOPContractsManager opcm = IOPContractsManager(_input.opcmAddress);

        output_.protocolVersionsProxy = IProtocolVersions(opcm.protocolVersions());
        output_.superchainConfigProxy = ISuperchainConfig(opcm.superchainConfig());
        output_.superchainProxyAdmin = IProxyAdmin(opcm.superchainProxyAdmin());

        IProxy protocolVersionsProxy = IProxy(payable(address(output_.protocolVersionsProxy)));
        IProxy superchainConfigProxy = IProxy(payable(address(output_.superchainConfigProxy)));

        vm.startPrank(address(0));
        output_.protocolVersionsImpl = IProtocolVersions(address(protocolVersionsProxy.implementation()));
        output_.superchainConfigImpl = ISuperchainConfig(address(superchainConfigProxy.implementation()));
        output_.protocolVersionsImpl = IProtocolVersions(protocolVersionsProxy.implementation());
        output_.superchainConfigImpl = ISuperchainConfig(superchainConfigProxy.implementation());
        vm.stopPrank();

        output_.guardian = output_.superchainConfigProxy.guardian();
        output_.protocolVersionsOwner = output_.protocolVersionsProxy.owner();
        output_.superchainProxyAdminOwner = output_.superchainProxyAdmin.owner();
        output_.recommendedProtocolVersion =
            bytes32(ProtocolVersion.unwrap(output_.protocolVersionsProxy.recommended()));
        output_.requiredProtocolVersion = bytes32(ProtocolVersion.unwrap(output_.protocolVersionsProxy.required()));
    }
}

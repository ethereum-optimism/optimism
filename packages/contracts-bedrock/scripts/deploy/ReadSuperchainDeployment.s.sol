// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IProtocolVersions, ProtocolVersion } from "interfaces/L1/IProtocolVersions.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

contract ReadSuperchainDeployment is Script {
    struct Input {
        IOPContractsManager opcmAddress; // TODO: Remove OPCMAddress field when OPCMv1 gets deprecated
        ISuperchainConfig superchainConfigProxy;
    }

    struct Output {
        // TODO: Remove ProtocolVersions fields when OPCMv1 gets deprecated
        IProtocolVersions protocolVersionsImpl;
        IProtocolVersions protocolVersionsProxy;
        address protocolVersionsOwner;
        bytes32 recommendedProtocolVersion;
        bytes32 requiredProtocolVersion;
        // Superchain config
        ISuperchainConfig superchainConfigImpl;
        ISuperchainConfig superchainConfigProxy;
        IProxyAdmin superchainProxyAdmin;
        address guardian;
        address superchainProxyAdminOwner;
    }

    function run(Input memory _input) public returns (Output memory output_) {
        // OPCM v2 is active when superchainConfigProxy is provided (non-zero)
        bool isOPCMV2 = address(_input.superchainConfigProxy) != address(0);

        if (isOPCMV2) {
            // OPCM v2: Use provided SuperchainConfigProxy, leave protocol versions unfilled
            output_.superchainConfigProxy = _input.superchainConfigProxy;
            output_.superchainProxyAdmin = IProxyAdmin(EIP1967Helper.getAdmin(address(output_.superchainConfigProxy)));

            IProxy superchainConfigProxy = IProxy(payable(address(output_.superchainConfigProxy)));

            vm.startPrank(address(0));
            output_.superchainConfigImpl = ISuperchainConfig(address(superchainConfigProxy.implementation()));
            vm.stopPrank();

            output_.guardian = output_.superchainConfigProxy.guardian();
            output_.superchainProxyAdminOwner = output_.superchainProxyAdmin.owner();

            // Protocol versions fields remain zero/unfilled in OPCM v2
        } else {
            // OPCM v1: Original logic using OPCM address
            require(address(_input.opcmAddress) != address(0), "ReadSuperchainDeployment: opcmAddress not set");

            IOPContractsManager opcm = IOPContractsManager(_input.opcmAddress);

            output_.protocolVersionsProxy = IProtocolVersions(opcm.protocolVersions());
            output_.superchainConfigProxy = ISuperchainConfig(opcm.superchainConfig());
            output_.superchainProxyAdmin = IProxyAdmin(EIP1967Helper.getAdmin(address(output_.superchainConfigProxy)));

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
}

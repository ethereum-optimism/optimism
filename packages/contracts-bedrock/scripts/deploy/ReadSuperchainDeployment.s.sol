// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IProtocolVersions, ProtocolVersion } from "interfaces/L1/IProtocolVersions.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";
import { IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";
import { Constants } from "src/libraries/Constants.sol";

contract ReadSuperchainDeployment is Script {
    struct Input {
        IOPContractsManager opcmAddress; // TODO(#18612): Remove OPCMAddress field when OPCMv1 gets deprecated
        ISuperchainConfig superchainConfigProxy;
    }

    struct Output {
        // TODO(#18612): Remove ProtocolVersions fields when OPCMv1 gets deprecated
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
        // Determine OPCM version by checking the semver or if the OPCM address is set. OPCM v2 starts at version 7.0.0.
        IOPContractsManager opcm = IOPContractsManager(_input.opcmAddress);
        bool isOPCMV2;
        if (address(opcm) == address(0)) {
            isOPCMV2 = true;
        } else {
            require(address(opcm).code.length > 0, "ReadSuperchainDeployment: OPCM address has no code");
            isOPCMV2 = SemverComp.gte(opcm.version(), Constants.OPCM_V2_MIN_VERSION);
        }

        if (isOPCMV2) {
            require(
                address(_input.superchainConfigProxy).code.length > 0,
                "ReadSuperchainDeployment: superchainConfigProxy has no code for OPCM v2"
            );

            // For OPCM v2, ProtocolVersions is being removed. Therefore, the ProtocolVersions-related fields
            // (protocolVersionsImpl, protocolVersionsProxy, protocolVersionsOwner, recommendedProtocolVersion,
            // requiredProtocolVersion) are intentionally left uninitialized.
            output_.superchainConfigProxy = _input.superchainConfigProxy;
            output_.superchainProxyAdmin = IProxyAdmin(EIP1967Helper.getAdmin(address(output_.superchainConfigProxy)));

            IProxy superchainConfigProxy = IProxy(payable(address(output_.superchainConfigProxy)));

            vm.startPrank(address(0));
            output_.superchainConfigImpl = ISuperchainConfig(superchainConfigProxy.implementation());
            vm.stopPrank();

            output_.guardian = output_.superchainConfigProxy.guardian();
            output_.superchainProxyAdminOwner = output_.superchainProxyAdmin.owner();
        } else {
            // When running on OPCM v1, the OPCM address is used to read the ProtocolVersions contract and
            // SuperchainConfig.
            output_.protocolVersionsProxy = IProtocolVersions(opcm.protocolVersions());
            output_.superchainConfigProxy = ISuperchainConfig(opcm.superchainConfig());
            output_.superchainProxyAdmin = IProxyAdmin(EIP1967Helper.getAdmin(address(output_.superchainConfigProxy)));

            IProxy protocolVersionsProxy = IProxy(payable(address(output_.protocolVersionsProxy)));
            IProxy superchainConfigProxy = IProxy(payable(address(output_.superchainConfigProxy)));

            vm.startPrank(address(0));
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

    function runWithBytes(bytes memory _input) public returns (bytes memory) {
        Input memory input = abi.decode(_input, (Input));
        Output memory output = run(input);
        return abi.encode(output);
    }
}

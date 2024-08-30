// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { OPStackManager } from "src/L1/OPStackManager.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { ProtocolVersions } from "src/L1/ProtocolVersions.sol";
import { ResourceMetering } from "src/L1/ResourceMetering.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { SystemConfigInterop } from "src/L1/SystemConfigInterop.sol";

/// @custom:proxied TODO this is not proxied yet.
contract OPStackManagerInterop is OPStackManager {
    constructor(
        SuperchainConfig _superchainConfig,
        ProtocolVersions _protocolVersions,
        Blueprints memory _blueprints
    )
        OPStackManager(_superchainConfig, _protocolVersions, _blueprints)
    { }

    function encodeSystemConfigInitializer(
        bytes4 selector,
        DeployInput memory _input,
        DeployOutput memory _output
    )
        internal
        pure
        virtual
        override
        returns (bytes memory)
    {
        // The `SystemConfigInterop` contract has an extra `address _dependencyManager` argument
        // that we must account for.
        (ResourceMetering.ResourceConfig memory referenceResourceConfig, SystemConfig.Addresses memory addrs) =
            defaultSystemConfigParams(selector, _input, _output);

        // TODO For now we assume that the dependency manager is the same as the proxy admin owner.
        // This is currently undefined since it's not part of the standard config, so we may need
        // to update where this value is pulled from in the future. If we want to support a
        // different dependency manager in this contract without an invasive change of redefining
        // the `Roles` struct, we can pull this from "hidden data". Solidity allows excess
        // calldata to be appended, so while in development we could simply require an extra
        // 32 bytes to be appended to the calldata and then do:
        //
        //     address dependencyManager = abi.decode(msg.data[msg.data.length - 32:], (address))
        //
        // And you don't even have really need to ABI encode the address, and can just append 20 bytes:
        //
        //     address dependencyManager = address(uint160(uint256(msg.data[msg.data.length - 20:])))
        //
        // However, once closer to production and preparing the release candidate we must properly
        // update the Roles struct.
        address dependencyManager = address(_input.roles.opChainProxyAdminOwner);

        return abi.encodeWithSelector(
            selector,
            _input.roles.systemConfigOwner,
            _input.basefeeScalar,
            _input.blobBasefeeScalar,
            bytes32(uint256(uint160(_input.roles.batcher))), // batcherHash
            30_000_000, // gasLimit
            _input.roles.unsafeBlockSigner,
            referenceResourceConfig,
            chainIdToBatchInboxAddress(_input.l2ChainId),
            addrs,
            dependencyManager
        );
    }
}

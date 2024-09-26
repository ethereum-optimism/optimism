// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { OPContractsManager } from "src/L1/OPContractsManager.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { ProtocolVersions } from "src/L1/ProtocolVersions.sol";
import { IResourceMetering } from "src/L1/interfaces/IResourceMetering.sol";
import { ResourceMetering } from "src/L1/ResourceMetering.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { ISystemConfigV160 } from "src/L1/interfaces/ISystemConfigV160.sol";
import { SystemConfigInterop } from "src/L1/SystemConfigInterop.sol";
import { Constants } from "src/libraries/Constants.sol";

/// @custom:proxied true
contract OPContractsManagerInterop is OPContractsManager {
    constructor(
        SuperchainConfig _superchainConfig,
        ProtocolVersions _protocolVersions
    )
        OPContractsManager(_superchainConfig, _protocolVersions)
    { }

    // The `SystemConfigInterop` contract has an extra `address _dependencyManager` argument
    // that we must account for.
    function encodeSystemConfigInitializer(
        bytes4 _selector,
        DeployInput memory _input,
        DeployOutput memory _output
    )
        internal
        view
        virtual
        override
        returns (bytes memory)
    {
        (
            ResourceMetering.ResourceConfig memory referenceResourceConfig,
            ISystemConfigV160.Addresses memory opChainAddrs
        ) = defaultSystemConfigInteropParams(_selector, _input, _output);

        // TODO For now we assume that the dependency manager is the same as the proxy admin owner.
        // This is currently undefined since it's not part of the standard config, so we may need
        // to update where this value is pulled from in the future. To support a different dependency
        // manager in this contract without an invasive change of redefining the `Roles` struct,
        // we will make the change described in https://github.com/ethereum-optimism/optimism/issues/11783.
        address dependencyManager = address(_input.roles.opChainProxyAdminOwner);

        return abi.encodeWithSelector(
            _selector,
            _input.roles.systemConfigOwner,
            _input.basefeeScalar,
            _input.blobBasefeeScalar,
            bytes32(uint256(uint160(_input.roles.batcher))), // batcherHash
            30_000_000, // gasLimit TODO make this configurable?
            _input.roles.unsafeBlockSigner,
            referenceResourceConfig,
            chainIdToBatchInboxAddress(_input.l2ChainId),
            opChainAddrs,
            dependencyManager
        );
    }

    /// @notice Returns default, standard config arguments for the SystemConfig initializer.
    /// This is used by subclasses to reduce code duplication.
    function defaultSystemConfigInteropParams(
        bytes4, /* selector */
        DeployInput memory, /* _input */
        DeployOutput memory _output
    )
        internal
        view
        returns (
            ResourceMetering.ResourceConfig memory resourceConfig_,
            ISystemConfigV160.Addresses memory opChainAddrs_
        )
    {
        // We use assembly to easily convert from IResourceMetering.ResourceConfig to ResourceMetering.ResourceConfig.
        // This is required because we have not yet fully migrated the codebase to be interface-based.
        IResourceMetering.ResourceConfig memory resourceConfig = Constants.DEFAULT_RESOURCE_CONFIG();
        assembly ("memory-safe") {
            resourceConfig_ := resourceConfig
        }

        opChainAddrs_ = ISystemConfigV160.Addresses({
            l1CrossDomainMessenger: address(_output.l1CrossDomainMessengerProxy),
            l1ERC721Bridge: address(_output.l1ERC721BridgeProxy),
            l1StandardBridge: address(_output.l1StandardBridgeProxy),
            disputeGameFactory: address(_output.disputeGameFactoryProxy),
            optimismPortal: address(_output.optimismPortalProxy),
            optimismMintableERC20Factory: address(_output.optimismMintableERC20FactoryProxy)
        });

        assertValidContractAddress(opChainAddrs_.l1CrossDomainMessenger);
        assertValidContractAddress(opChainAddrs_.l1ERC721Bridge);
        assertValidContractAddress(opChainAddrs_.l1StandardBridge);
        assertValidContractAddress(opChainAddrs_.disputeGameFactory);
        assertValidContractAddress(opChainAddrs_.optimismPortal);
        assertValidContractAddress(opChainAddrs_.optimismMintableERC20Factory);
    }
}

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { SystemConfig } from "src/L1/SystemConfig.sol";
import { ISemver } from "src/universal/interfaces/ISemver.sol";

/// @notice Represents the roles that can be set when deploying a standard OP Stack chain.
struct Roles {
    address proxyAdminOwner;
    address systemConfigOwner;
    address batcher;
    address unsafeBlockSigner;
    address proposer;
    address challenger;
}

/// @title IOPStackManager
/// @notice Interface for the OPStackManager contract.
interface IOPStackManager is ISemver {
    /// @notice Emitted when a new OP Stack chain is deployed.
    /// @param l2ChainId The chain ID of the new chain.
    /// @param systemConfig The address of the new chain's SystemConfig contract.
    event Deployed(uint256 indexed l2ChainId, SystemConfig indexed systemConfig);

    /// @notice Thrown when an invalid `l2ChainId` is provided to `deploy`.
    error InvalidChainId();

    /// @notice Thrown when a deployment fails.
    error DeploymentFailed(string reason);

    /// @notice Temporary error since the deploy method is not yet implemented.
    error NotImplemented();

    function deploy(
        uint256 _l2ChainId,
        uint32 _basefeeScalar,
        uint32 _blobBasefeeScalar,
        Roles calldata _roles
    )
        external
        view // This is only here to silence the compiler warning until the function is fully implemented.
        returns (SystemConfig systemConfig_);
}

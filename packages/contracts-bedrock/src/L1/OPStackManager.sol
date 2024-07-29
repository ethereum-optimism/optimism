// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ISemver } from "src/universal/ISemver.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";

contract OPStackManager is ISemver {
    /// @custom:semver 1.0.0-beta.1
    string public constant version = "1.0.0-beta.1";

    /// @notice Represents the roles that can be set when deploying a standard OP Stack chain.
    struct Roles {
        address proxyAdminOwner;
        address systemConfigOwner;
        address batcher;
        address unsafeBlockSigner;
        address proposer;
        address challenger;
    }

    /// @notice Emitted when a new OP Stack chain is deployed.
    /// @param l2ChainId The chain ID of the new chain.
    /// @param systemConfig The address of the new chain's SystemConfig contract.
    event Deployed(uint256 indexed l2ChainId, SystemConfig indexed systemConfig);

    /// @notice Thrown when an invalid `l2ChainId` is provided to `deploy`.
    error InvalidChainId();

    /// @notice Thrown when a deployment fails.
    error DeploymentFailed(string reason);

    function deploy(
        uint256 l2ChainId,
        Roles calldata roles,
        uint32 basefeeScalar,
        uint32 blobBasefeeScalar
    )
        external
        view // This is only here to silence the compiler warning until the function is fully implemented.
        returns (SystemConfig systemConfig)
    {
        if (l2ChainId == 0 || l2ChainId == block.chainid) revert InvalidChainId();

        // Silence compiler warnings.
        roles;
        basefeeScalar;
        blobBasefeeScalar;
        systemConfig;

        revert("Not implemented");
    }
}

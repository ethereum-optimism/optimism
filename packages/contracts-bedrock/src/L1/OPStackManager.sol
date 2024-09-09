// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ISemver } from "src/universal/interfaces/ISemver.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";

/// @custom:proxied true
contract OPStackManager is ISemver {
    /// @custom:semver 1.0.0-beta.2
    string public constant version = "1.0.0-beta.2";

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
        returns (SystemConfig systemConfig_)
    {
        if (_l2ChainId == 0 || _l2ChainId == block.chainid) revert InvalidChainId();

        // Silence compiler warnings.
        _roles;
        _basefeeScalar;
        _blobBasefeeScalar;
        systemConfig_;

        revert NotImplemented();
    }

    /// @notice Maps an L2 chain ID to an L1 batch inbox address as defined by the standard
    /// configuration's convention. This convention is `versionByte || keccak256(bytes32(chainId))[:19]`,
    /// where || denotes concatenation`, versionByte is 0x00, and chainId is a uint256.
    /// https://specs.optimism.io/protocol/configurability.html#consensus-parameters
    function chainIdToBatchInboxAddress(uint256 _l2ChainId) internal pure returns (address) {
        bytes1 versionByte = 0x00;
        bytes32 hashedChainId = keccak256(bytes.concat(bytes32(_l2ChainId)));
        bytes19 first19Bytes = bytes19(hashedChainId);
        return address(uint160(bytes20(bytes.concat(versionByte, first19Bytes))));
    }
}

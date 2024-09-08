// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { SystemConfig } from "src/L1/SystemConfig.sol";
import { IOPStackManager, Roles } from "src/L1/interfaces/IOPStackManager.sol";

/// @custom:proxied true
contract OPStackManager is IOPStackManager {
    /// @custom:semver 1.0.0-beta.2
    string public constant version = "1.0.0-beta.2";

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

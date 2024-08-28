// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IERC721Bridge } from "src/universal/interfaces/IERC721Bridge.sol";
import { ISemver } from "src/universal/interfaces/ISemver.sol";

/// @title IL1ERC721Bridge
/// @notice Interface for the L1ERC721Bridge contract.
interface IL1ERC721Bridge is IERC721Bridge, ISemver {
    function deposits(address, address, uint256) external view returns (bool);
    function finalizeBridgeERC721(
        address _localToken,
        address _remoteToken,
        address _from,
        address _to,
        uint256 _tokenId,
        bytes memory _extraData
    )
        external;
    function initialize(address _messenger, address _superchainConfig) external;
    function superchainConfig() external view returns (address);
}

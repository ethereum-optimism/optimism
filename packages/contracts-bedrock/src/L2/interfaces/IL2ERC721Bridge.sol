// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IL2ERC721Bridge
/// @notice Interface for the L2ERC721Bridge contract.
interface IL2ERC721Bridge {
    function finalizeBridgeERC721(
        address _localToken,
        address _remoteToken,
        address _from,
        address _to,
        uint256 _tokenId,
        bytes calldata _extraData
    )
        external;
}

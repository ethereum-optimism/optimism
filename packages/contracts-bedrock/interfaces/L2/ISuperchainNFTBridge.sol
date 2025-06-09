// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title ISuperchainNFTBridge
/// @notice Interface for the SuperchainNFTBridge contract.
interface ISuperchainNFTBridge is ISemver {
    error ZeroAddress();
    error Unauthorized();
    error SuperchainNFTBridge_InvalidCrossDomainSender();
    error SuperchainNFTBridge_InvalidCrosschainERC721();

    event SendERC721(
        address indexed token, address indexed from, address indexed to, uint256 tokenId, uint256 destination
    );

    event RelayERC721(address indexed token, address indexed from, address indexed to, uint256 tokenId, uint256 source);

    function sendERC721(
        address _token,
        address _to,
        uint256 _tokenId,
        uint256 _chainId
    )
        external
        returns (bytes32 msgHash_);

    function relayERC721(address _token, address _from, address _to, uint256 _tokenId) external;

    function __constructor__() external;
}

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IERC165 } from "@openzeppelin/contracts/interfaces/IERC165.sol";

/// @title ICrosschainERC721
/// @notice Defines the interface for crosschain ERC721 transfers.
interface ICrosschainERC721 is IERC165 {
    /// @notice Emitted when a crosschain transfer mints an NFT token.
    /// @param to       Address of the account the token is being minted for.
    /// @param tokenId  Token ID being minted.
    /// @param sender   Address of the account that finalized the crosschain transfer.
    event CrosschainMint(address indexed to, uint256 tokenId, address indexed sender);

    /// @notice Emitted when a crosschain transfer burns an NFT token.
    /// @param from     Address of the account the token is being burned from.
    /// @param tokenId  Token ID being burned.
    /// @param sender   Address of the account that initiated the crosschain transfer.
    event CrosschainBurn(address indexed from, uint256 tokenId, address indexed sender);

    /// @notice Mint the NFT token through a crosschain transfer.
    /// @param _to      Address to mint the token to.
    /// @param _tokenId Token ID being minted.
    function crosschainMint(address _to, uint256 _tokenId) external;

    /// @notice Burn the NFT token through a crosschain transfer.
    /// @param _from    Address to burn the token from.
    /// @param _tokenId Token ID being burned.
    function crosschainBurn(address _from, uint256 _tokenId) external;
}

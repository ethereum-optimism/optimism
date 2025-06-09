// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

// Contracts
import { ERC721 } from "@solady-v0.0.245/tokens/ERC721.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Unauthorized } from "src/libraries/errors/CommonErrors.sol";

// Interfaces
import { IERC721 } from "@openzeppelin/contracts/token/ERC721/IERC721.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { ICrosschainERC721, IERC165 } from "interfaces/L2/ICrosschainERC721.sol";

/// @title SuperchainERC721
/// @notice A standard ERC721 extension implementing ICrosschainERC721 for unified cross-chain ERC721
///         transfers across the Superchain. Gives the SuperchainNFTBridge mint and burn permissions.
/// @dev    This contract inherits from Solady@v0.0.245 ERC721. Carefully review Solady's,
///         documentation including all warnings, comments and natSpec, before extending or
///         interacting with this contract.
abstract contract SuperchainERC721 is ERC721, ICrosschainERC721, ISemver {
    /// @notice Semantic version.
    /// @custom:semver 0.0.1
    function version() external view virtual returns (string memory) {
        return "0.0.1";
    }

    /// @notice Allows the SuperchainNFTBridge to mint an NFT token.
    /// @param _to Address to mint an NFT token to.
    /// @param _tokenId Token ID to mint.
    function crosschainMint(address _to, uint256 _tokenId) external {
        if (msg.sender != Predeploys.SUPERCHAIN_NFT_BRIDGE) revert Unauthorized();

        _mint(_to, _tokenId);

        emit CrosschainMint(_to, _tokenId, msg.sender);
    }

    /// @notice Allows the SuperchainNFTBridge to burn an NFT token.
    /// @param _from Address to burn an NFT token from.
    /// @param _tokenId Token ID to burn.
    function crosschainBurn(address _from, uint256 _tokenId) external {
        if (msg.sender != Predeploys.SUPERCHAIN_NFT_BRIDGE) revert Unauthorized();

        _burn(_from, _tokenId);

        emit CrosschainBurn(_from, _tokenId, msg.sender);
    }

    /// @inheritdoc IERC165
    function supportsInterface(bytes4 _interfaceId) public view virtual override(ERC721, IERC165) returns (bool) {
        return _interfaceId == type(ICrosschainERC721).interfaceId || _interfaceId == type(IERC721).interfaceId
            || _interfaceId == type(IERC165).interfaceId;
    }
}

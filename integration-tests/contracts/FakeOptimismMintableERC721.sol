// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { ERC721 } from "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import { OptimismMintableERC721 } from "@eth-optimism/contracts-bedrock/contracts/universal/OptimismMintableERC721.sol";
import { IERC165 } from "@openzeppelin/contracts/utils/introspection/IERC165.sol";

contract FakeOptimismMintableERC721 is OptimismMintableERC721 {

    constructor(
        address _bridge,
        address _remoteToken,
        uint256 _remoteChainId
    ) OptimismMintableERC721(
        _bridge,
        _remoteChainId,
        _remoteToken,
        "FakeERC721",
        "FAKE"
    ) {}

    function safeMint(address to, uint256 tokenId) external override {
        _safeMint(to, tokenId);
    }

    // Burn will be called by the L2 Bridge to burn the NFT we are bridging to L1
    function burn(address, uint256 tokenId) external override {
        _burn(tokenId);
    }
}

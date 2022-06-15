// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { ERC721 } from "@openzeppelin/contracts/token/ERC721/ERC721.sol";

contract FakeOptimismMintableERC721 is ERC721 {

    address public immutable remoteToken;
    address public immutable bridge;

    constructor(address _remoteToken, address _bridge) ERC721("FakeERC721", "FAKE") {
        remoteToken = _remoteToken;
        bridge = _bridge;
    }

    function mint(address to, uint256 tokenId) public {
        _mint(to, tokenId);
    }

    // Burn will be called by the L2 Bridge to burn the NFT we are bridging to L1
    function burn(address, uint256) external {}
}

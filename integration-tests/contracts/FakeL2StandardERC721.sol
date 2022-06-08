// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { ERC721 } from "@openzeppelin/contracts/token/ERC721/ERC721.sol";

contract FakeL2StandardERC721 is ERC721 {

    address public immutable l1Token;
    address public immutable l2Bridge;

    constructor(address _l1Token, address _l2Bridge) ERC721("FakeERC721", "FAKE") {
        l1Token = _l1Token;
        l2Bridge = _l2Bridge;
    }

    function mint(address to, uint256 tokenId) public {
        _mint(to, tokenId);
    }

    // Burn will be called by the L2 Bridge to burn the NFT we are bridging to L1
    function burn(address, uint256) external {}
}

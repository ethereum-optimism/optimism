// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ERC721 } from "@rari-capital/solmate/src/tokens/ERC721.sol";

contract TestERC721 is ERC721 {
    constructor() ERC721("TEST", "TST") { }

    function mint(address to, uint256 tokenId) public {
        _mint(to, tokenId);
    }

    function tokenURI(uint256) public pure virtual override returns (string memory) { }
}

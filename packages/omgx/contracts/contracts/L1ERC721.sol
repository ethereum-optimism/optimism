// SPDX-License-Identifier: MIT
pragma solidity >0.5.0;

import '@openzeppelin/contracts/token/ERC721/ERC721.sol';
import "@openzeppelin/contracts/access/Ownable.sol";

/**
 * @title ERC721
 * @dev A super simple ERC721 implementation!
 */
contract L1ERC721 is Ownable, ERC721 {
    constructor(
        string memory name,
        string memory symbol
    )
        public
        ERC721(
            name,
            symbol
        ) {
    }

    function mint(address _to, uint256 _tokenId) public onlyOwner {
        _safeMint(_to, _tokenId);
    }
}

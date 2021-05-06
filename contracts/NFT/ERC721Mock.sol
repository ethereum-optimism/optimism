// SPDX-License-Identifier: MIT

pragma solidity >=0.7.6;

import "./ERC721.sol";
import "./utils/Counters.sol";

/**
 * @title ERC721Mock
 * This mock just provides a public safeMint, mint, and burn functions for testing purposes
 */
contract ERC721Mock is ERC721 {

    using Counters for Counters.Counter;
    Counters.Counter private _tokenIds;

    constructor (string memory name, string memory symbol) ERC721(name, symbol) { }
/*
    function mintNFT(address recipient, string memory tokenURI) public returns (uint256)
    {
        _tokenIds.increment();

        uint256 newItemId = _tokenIds.current();
        
        safeMint(recipient, newItemId);

        setTokenURI(newItemId, tokenURI);

        return newItemId;
    }
*/
    function mintNFT(address recipient, uint256 tokenId, string memory tokenURI) public returns (uint256)
    {
        safeMint(recipient, tokenId);

        setTokenURI(tokenId, tokenURI);

        return tokenId;
    }

    function baseURI() public view override returns (string memory) {
        return baseURI();
    }

    function setBaseURI(string calldata newBaseURI) public {
        _setBaseURI(newBaseURI);
    }

    function setTokenURI(uint256 tokenId, string memory _tokenURI) public {
        _setTokenURI(tokenId, _tokenURI);
    }

    function getTokenURI(uint256 tokenId) public view returns (string memory) {
        return tokenURI(tokenId);
    }

    function exists(uint256 tokenId) public view returns (bool) {
        return _exists(tokenId);
    }

    function mint(address to, uint256 tokenId) public {
        _mint(to, tokenId);
    }

    function safeMint(address to, uint256 tokenId) public {
        _safeMint(to, tokenId);
    }

    function safeMint(address to, uint256 tokenId, bytes memory _data) public {
        _safeMint(to, tokenId, _data);
    }

    function burn(uint256 tokenId) public {
        _burn(tokenId);
    }
}

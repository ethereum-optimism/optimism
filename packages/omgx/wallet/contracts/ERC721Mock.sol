// SPDX-License-Identifier: MIT

pragma solidity >=0.7.6;
import "@openzeppelin/contracts/token/ERC721/ERC721.sol";
/**
 * @title ERC721Mock
 * This mock just provides a public safeMint, mint, and burn functions for testing purposes
 */
contract ERC721Mock is ERC721 {

    uint256 tID;

    constructor (string memory name, string memory symbol, uint256 tID_start) 
        ERC721(name, symbol) {
        tID = tID_start;
    }

    function mintNFT(address recipient, string memory tokenURI) public returns (uint256)
    {
        safeMint(recipient, tID);
        setTokenURI(tID, tokenURI);
        tID += 1;
        return tID;
    }

    function getLastTID() public view returns(uint256) {
        return tID;
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

    //for a specific tokenId, get the associated NFT
    function getTokenURI(uint256 tokenId) public view returns (string memory) {
        return tokenURI(tokenId);
    }

    function getName() public view returns (string memory) {
        return name();
    }

    function getSymbol() public view returns (string memory) {
        return symbol();
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

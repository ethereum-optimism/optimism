// SPDX-License-Identifier: MIT
pragma solidity >0.5.0;

import "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

/**
 * @title ERC721Mock
 * 
 */
contract ERC721Mock is Ownable, ERC721 {

    uint256 tID;

    // Ancestral NFT
    struct Ancestor { 
       address cAddress;
       string id;
       string chain;
    }
  
    Ancestor genesis;

    constructor (
        string memory name, 
        string memory symbol, 
        uint256 tID_start, 
        address origin_cAddress,
        string memory origin_id,
        string memory origin_chain
    ) 
        ERC721(name, symbol) {
        _setBaseURI('');
        tID = tID_start;
        genesis = Ancestor(
            origin_cAddress,
            origin_id,
            origin_chain
        );
    }

    function mintNFT(address recipient, string memory tokenURI) public onlyOwner returns (uint256)
    {
        mint(recipient, tID);
        setTokenURI(tID, tokenURI);
        tID += 1;
        return tID;
    }

    function getLastTID() public view returns(uint256) {
        return tID;
    }

    function getGenesis() public view returns (
        address, 
        string memory, 
        string memory) {  
        return(genesis.cAddress, genesis.id, genesis.chain);  
    } 

    function setTokenURI(uint256 tokenId, string memory _tokenURI) public {
        _setTokenURI(tokenId, _tokenURI);
    }

    //for a specific tokenId, get the associated NFT
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

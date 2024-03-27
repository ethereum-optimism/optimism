// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { ERC721 } from "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";

contract SuperchainSubwayTicket is ERC721, Ownable {
    uint256 public currentTokenId;
    string public _baseTokenURI;

    event BaseURIChanged(string newBaseURI);

    constructor(string memory name, string memory symbol, string memory baseURI) ERC721(name, symbol) {
        setBaseURI(baseURI);
    }

    function mint(address _recipient) public returns (uint256) {
        uint256 newItemId = ++currentTokenId;
        _safeMint(_recipient, newItemId);
        return newItemId;
    }

    function setBaseURI(string memory _newBaseURI) public onlyOwner {
        _baseTokenURI = _newBaseURI;
        emit BaseURIChanged(_newBaseURI);
    }

    function _baseURI() internal view override returns (string memory) {
        return _baseTokenURI;
    }
}

// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import "@rari-capital/solmate/src/tokens/ERC721.sol";
import "@openzeppelin/contracts/utils/Strings.sol";
import "./SocialContract.sol";

contract Optimist is ERC721 {
    SocialContract public sc;
    address public admin;

    constructor(
        string memory _name,
        string memory _symbol,
        address _admin
    ) ERC721(_name, _symbol) {
        admin = _admin;
    }

    function mint(address recipient) public returns (uint256) {
        uint256 newItemId = uint256(uint160(recipient));
        _safeMint(recipient, newItemId);
        return newItemId;
    }

    function transferFrom(
        address,
        address,
        uint256
    ) public override {
        revert("Optimist::transferFrom: SOUL_BOUND");
    }

    function tokenURI(uint256 tokenId) public view virtual override returns (string memory) {
        if (ownerOf(tokenId) == address(0)) {
            revert("Optimist:::tokenURI: TOKEN_URI_DNE");
        }
        return Strings.toString(uint256(uint160(ownerOf(tokenId))));
    }

    function baseURI() public view returns (bytes memory) {
        return sc.attestations(admin, address(this), keccak256("opnft.optimistNftBaseURI"));
    }
}

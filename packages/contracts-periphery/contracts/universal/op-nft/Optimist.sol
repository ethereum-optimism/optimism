// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import "@openzeppelin/contracts-upgradeable/token/ERC721/ERC721Upgradeable.sol";
import "@openzeppelin/contracts-upgradeable/token/ERC721/extensions/ERC721BurnableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import "@openzeppelin/contracts-upgradeable/utils/StringsUpgradeable.sol";
import "./SocialContract.sol";

contract Optimist is
    Initializable,
    ERC721Upgradeable,
    ERC721BurnableUpgradeable,
    OwnableUpgradeable
{
    SocialContract public sc;
    address public admin;

    function initialize(
        string calldata _name,
        string calldata _symbol,
        address _admin,
        address _sc
    ) public initializer {
        __ERC721_init(_name, _symbol);
        __ERC721Burnable_init();
        __Ownable_init();
        sc = SocialContract(_sc);
        admin = _admin;
    }

    function mint(address recipient) public onlyOwner {
        uint256 tokenId = uint256(uint160(recipient));
        _safeMint(recipient, tokenId);
    }

    function tokenURI(uint256 tokenId) public view virtual override returns (string memory) {
        if (ownerOf(tokenId) == address(0)) {
            revert("Optimist::tokenURI: TOKEN_URI_DNE");
        }
        return StringsUpgradeable.toString(uint256(uint160(ownerOf(tokenId))));
    }

    function baseURI() public view returns (bytes memory) {
        return sc.attestations(admin, address(this), keccak256("opnft.optimistNftBaseURI"));
    }

    function _beforeTokenTransfer(
        address from,
        address to,
        uint256
    ) internal pure override {
        require(
            from == address(0) || to == address(0),
            "Optimist::_beforeTokenTransfer: SOUL_BOUND"
        );
    }
}

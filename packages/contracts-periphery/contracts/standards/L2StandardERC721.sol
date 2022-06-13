// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { ERC721 } from "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import { IERC165 } from "@openzeppelin/contracts/utils/introspection/IERC165.sol";
import { Strings } from "@openzeppelin/contracts/utils/Strings.sol";
import { Lib_Strings } from "../libraries/utils/Lib_Strings.sol";
import "./IL2StandardERC721.sol";

contract L2StandardERC721 is IL2StandardERC721, ERC721 {
    address public l1Token;
    address public l2Bridge;
    string public baseTokenURI;

    /**
     * @param _l2Bridge Address of the L2 standard bridge.
     * @param _l1Token Address of the corresponding L1 token.
     * @param _name ERC721 name.
     * @param _symbol ERC721 symbol.
     */
    constructor(
        address _l2Bridge,
        address _l1Token,
        string memory _name,
        string memory _symbol
    ) ERC721(_name, _symbol) {
        l1Token = _l1Token;
        l2Bridge = _l2Bridge;

        // Creates a base URI in the format specified by EIP-681:
        // https://eips.ethereum.org/EIPS/eip-681
        baseTokenURI = string(
            abi.encodePacked(
                "ethereum:0x",
                Lib_Strings.addressToString(_l1Token),
                "@",
                Strings.toString(block.chainid),
                "/tokenURI?uint256="
            )
        );
    }

    modifier onlyL2Bridge() {
        require(msg.sender == l2Bridge, "Only L2 Bridge can mint and burn");
        _;
    }

    // slither-disable-next-line external-function
    function supportsInterface(bytes4 _interfaceId)
        public
        view
        override(ERC721, IERC165)
        returns (bool)
    {
        bytes4 iface1 = type(IERC165).interfaceId;
        bytes4 iface2 = type(IL2StandardERC721).interfaceId;
        return
            _interfaceId == iface1 ||
            _interfaceId == iface2 ||
            super.supportsInterface(_interfaceId);
    }

    // slither-disable-next-line external-function
    function mint(address _to, uint256 _tokenId) public virtual onlyL2Bridge {
        _mint(_to, _tokenId);

        emit Mint(_to, _tokenId);
    }

    // slither-disable-next-line external-function
    function burn(address _from, uint256 _tokenId) public virtual onlyL2Bridge {
        _burn(_tokenId);

        emit Burn(_from, _tokenId);
    }

    function _baseURI() internal view virtual override returns (string memory) {
        return baseTokenURI;
    }
}

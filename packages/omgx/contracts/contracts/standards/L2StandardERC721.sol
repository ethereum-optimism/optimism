// SPDX-License-Identifier: MIT
pragma solidity 0.7.6;

import "@openzeppelin/contracts/token/ERC721/ERC721.sol";

import "./IL2StandardERC721.sol";

contract L2StandardERC721 is IL2StandardERC721, ERC721 {
    address public override l1Contract;
    address public l2Bridge;

    /**
     * @param _l2Bridge Address of the L2 standard bridge.
     * @param _l1Contract Address of the corresponding L1 NFT contract.
     * @param _name ERC721 name.
     * @param _symbol ERC721 symbol.
     */
    constructor(
        address _l2Bridge,
        address _l1Contract,
        string memory _name,
        string memory _symbol
    )
        ERC721(_name, _symbol) {
        l1Contract = _l1Contract;
        l2Bridge = _l2Bridge;
    }

    modifier onlyL2Bridge {
        require(msg.sender == l2Bridge, "Only L2 Bridge can mint and burn");
        _;
    }

    // ERC165 check interface
    function supportsInterface(bytes4 _interfaceId) public override(ERC165, IERC165) pure returns (bool) {
        bytes4 firstSupportedInterface = bytes4(keccak256("supportsInterface(bytes4)")); // ERC165
        bytes4 secondSupportedInterface = IL2StandardERC721.l1Contract.selector
            ^ IL2StandardERC721.mint.selector
            ^ IL2StandardERC721.burn.selector;
        return _interfaceId == firstSupportedInterface || _interfaceId == secondSupportedInterface;
    }

    function mint(address _to, uint256 _tokenId) public virtual override onlyL2Bridge {
        _mint(_to, _tokenId);

        emit Mint(_to, _tokenId);
    }

    function burn(uint256 _tokenId) public virtual override onlyL2Bridge {
        _burn(_tokenId);

        emit Burn(_tokenId);
    }
}

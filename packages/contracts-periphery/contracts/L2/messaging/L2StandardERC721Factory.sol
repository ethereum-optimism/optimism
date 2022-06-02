// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Contract Imports */
import { L2StandardERC721 } from "../../standards/L2StandardERC721.sol";

/**
 * @title L2StandardERC721Factory
 * @dev Factory contract for creating standard L2 ERC721 representations of L1 ERC721s
 * compatible with and working on the NFT bridge.
 */
contract L2StandardERC721Factory {
    event StandardL2ERC721Created(address indexed _l1Token, address indexed _l2Token);

    // Address of the L2 ERC721 Bridge.
    address public l2ERC721Bridge;

    // Maps an L2 ERC721 token address to a boolean that returns true if the token was created
    // with the L2StandardERC721Factory.
    mapping(address => bool) public isStandardERC721;

    // Maps an L1 ERC721 token address to an L2 Standard ERC721 token address. This mapping can
    // only be updated once per L1 ERC721 token.
    mapping(address => address) public standardERC721Mapping;

    constructor(address _l2ERC721Bridge) {
        l2ERC721Bridge = _l2ERC721Bridge;
    }

    /**
     * @dev Creates an instance of the standard ERC721 token on L2.
     * @param _l1Token Address of the corresponding L1 token.
     * @param _name ERC721 name.
     * @param _symbol ERC721 symbol.
     */
    function createStandardL2ERC721(
        address _l1Token,
        string memory _name,
        string memory _symbol
    ) external {
        require(_l1Token != address(0), "Must provide L1 token address");

        // Only one L2 Standard Token can exist for each L1 Token
        require(
            standardERC721Mapping[_l1Token] == address(0),
            "L2 Standard Token already exists for this L1 Token"
        );

        L2StandardERC721 l2Token = new L2StandardERC721(l2ERC721Bridge, _l1Token, _name, _symbol);

        isStandardERC721[address(l2Token)] = true;
        standardERC721Mapping[_l1Token] = address(l2Token);
        emit StandardL2ERC721Created(_l1Token, address(l2Token));
    }
}

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { ERC721 } from "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import { IOptimismMintableERC721 } from "@eth-optimism/contracts-periphery/contracts/universal/op-erc721/IOptimismMintableERC721.sol";
import { IERC165 } from "@openzeppelin/contracts/utils/introspection/IERC165.sol";

contract FakeOptimismMintableERC721 is ERC721 {

    address public immutable remoteToken;
    address public immutable bridge;

    constructor(address _remoteToken, address _bridge) ERC721("FakeERC721", "FAKE") {
        remoteToken = _remoteToken;
        bridge = _bridge;
    }

    function mint(address to, uint256 tokenId) public {
        _mint(to, tokenId);
    }

    // Burn will be called by the L2 Bridge to burn the NFT we are bridging to L1
    function burn(address, uint256 tokenId) external {
        _burn(tokenId);
    }

    // Returns true when queried with the interface ID for OptimismMintableERC721.
    function supportsInterface(bytes4 _interfaceId)
        public
        pure
        override
        returns (bool)
    {
        bytes4 iface1 = type(IERC165).interfaceId;
        bytes4 iface2 = type(IOptimismMintableERC721).interfaceId;
        return _interfaceId == iface1 || _interfaceId == iface2;
    }
}

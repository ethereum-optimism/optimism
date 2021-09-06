// SPDX-License-Identifier: MIT
pragma solidity 0.7.6;

import { IERC721 } from "@openzeppelin/contracts/token/ERC721/IERC721.sol";
import { IERC165 } from "@openzeppelin/contracts/introspection/IERC165.sol";

interface IL2StandardERC721 is IERC165, IERC721 {
    function l1Contract() external returns (address);

    function mint(address _to, uint256 _tokenId) external;

    function burn(uint256 _tokenId) external;

    event Mint(address indexed _account, uint256 _tokenId);
    event Burn(uint256 _tokenId);
}

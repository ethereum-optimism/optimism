// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

import { SuperchainERC721 } from "src/L2/SuperchainERC721.sol";

contract MockSuperchainERC721Implementation is SuperchainERC721 {
    function name() public pure override returns (string memory) {
        return "SuperchainERC721";
    }

    /// @dev Returns the token collection symbol.
    function symbol() public pure override returns (string memory) {
        return "SNFT";
    }

    /// @dev Returns the Uniform Resource Identifier (URI) for token `id`.
    function tokenURI(uint256 id) public pure override returns (string memory) {
        return string(abi.encode(id));
    }
}

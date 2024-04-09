// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { WETH98 } from "src/dispute/weth/WETH98.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { L1Block } from "src/L2/L1Block.sol";

/// @title WETH contract that reads the name and symbol from the L1Block contract
contract WETH is WETH98 {
    /// @notice Returns the name of the token from the L1Block contract
    function name() external view override returns (string memory) {
        return string.concat(
            "Wrapped ",
            _trimRightPaddedZeroes(
                string(abi.encodePacked(L1Block(Predeploys.L1_BLOCK_ATTRIBUTES).gasPayingTokenName()))
            )
        );
    }

    /// @notice Returns the symbol of the token from the L1Block contract
    function symbol() external view override returns (string memory) {
        return string.concat(
            "W",
            _trimRightPaddedZeroes(
                string(abi.encodePacked(L1Block(Predeploys.L1_BLOCK_ATTRIBUTES).gasPayingTokenSymbol()))
            )
        );
    }

    /// @notice Helper function to trim zero bytes from the end of a string.
    function _trimRightPaddedZeroes(string memory str) internal pure returns (string memory) {
        bytes memory strBytes = bytes(str);
        uint256 newLength = strBytes.length;
        for (; newLength > 0; newLength--) {
            if (strBytes[newLength - 1] != 0) {
                break;
            }
        }
        bytes memory trimmedBytes = new bytes(newLength);
        for (uint256 i = 0; i < newLength; i++) {
            trimmedBytes[i] = strBytes[i];
        }
        return string(trimmedBytes);
    }
}

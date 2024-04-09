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
        uint256 length = strBytes.length;

        // Find the index where the non-zero byte is last seen.
        uint256 lastIndex = 0; // We'll use this to mark the end of non-zero bytes.
        for (uint256 i = 0; i < length; i++) {
            if (strBytes[i] != 0) {
                lastIndex = i;
            }
        }

        // If no non-zero byte is found, return an empty string.
        // This also covers the case where the string is fully padded with zeroes.
        if (lastIndex == 0 && strBytes[0] == 0) {
            return "";
        }

        // Create a new bytes array of the appropriate length.
        // +1 because lastIndex represents the index of the last non-zero byte, and array indices are 0-based.
        bytes memory trimmedBytes = new bytes(lastIndex + 1);
        for (uint256 i = 0; i <= lastIndex; i++) {
            trimmedBytes[i] = strBytes[i];
        }

        return string(trimmedBytes);
    }
}

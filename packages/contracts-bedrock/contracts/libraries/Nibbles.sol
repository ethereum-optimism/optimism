// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/**
 * @title Nibbles
 * @notice Nibbles is a simple library for dealing with nibble arrays.
 */
library Nibbles {
    /**
     * @notice Converts a byte array into a nibble array by splitting each byte into two nibbles.
     *         Resulting nibble array will be exactly twice as long as the input byte array.
     *
     * @param _bytes Input byte array to convert.
     *
     * @return Resulting nibble array.
     */
    function toNibbles(bytes memory _bytes) internal pure returns (bytes memory) {
        bytes memory nibbles = new bytes(_bytes.length * 2);
        for (uint256 i = 0; i < _bytes.length; i++) {
            nibbles[i * 2] = _bytes[i] >> 4;
            nibbles[i * 2 + 1] = bytes1(uint8(_bytes[i]) % 16);
        }
        return nibbles;
    }

    /**
     * @notice Generates a byte array from a nibble array by joining each set of two nibbles into a
     *         single byte. Resulting byte array will be half as long as the input byte array.
     *
     * @param _bytes Input nibble array to convert.
     *
     * @return Resulting byte array.
     */
    function fromNibbles(bytes memory _bytes) internal pure returns (bytes memory) {
        bytes memory ret = new bytes(_bytes.length / 2);
        for (uint256 i = 0; i < ret.length; i++) {
            ret[i] = (_bytes[i * 2] << 4) | (_bytes[i * 2 + 1]);
        }
        return ret;
    }
}

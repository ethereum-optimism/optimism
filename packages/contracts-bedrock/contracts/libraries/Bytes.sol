// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/**
 * @title Bytes
 * @notice Bytes is a library for manipulating byte arrays.
 */
library Bytes {
    /**
     * @custom:attribution https://github.com/GNSPS/solidity-bytes-utils
     * @notice Slices a byte array with a given starting index and length. Returns a new byte array
     *         as opposed to a pointer to the original array. Will throw if trying to slice more
     *         bytes than exist in the array.
     *
     * @param _bytes Byte array to slice.
     * @param _start Starting index of the slice.
     * @param _length Length of the slice.
     *
     * @return Slice of the input byte array.
     */
    function slice(
        bytes memory _bytes,
        uint256 _start,
        uint256 _length
    ) internal pure returns (bytes memory) {
        unchecked {
            require(_length + 31 >= _length, "slice_overflow");
            require(_start + _length >= _start, "slice_overflow");
            require(_bytes.length >= _start + _length, "slice_outOfBounds");
        }

        bytes memory tempBytes;

        assembly {
            switch iszero(_length)
            case 0 {
                // Get a location of some free memory and store it in tempBytes as
                // Solidity does for memory variables.
                tempBytes := mload(0x40)

                // The first word of the slice result is potentially a partial
                // word read from the original array. To read it, we calculate
                // the length of that partial word and start copying that many
                // bytes into the array. The first word we copy will start with
                // data we don't care about, but the last `lengthmod` bytes will
                // land at the beginning of the contents of the new array. When
                // we're done copying, we overwrite the full first word with
                // the actual length of the slice.
                let lengthmod := and(_length, 31)

                // The multiplication in the next line is necessary
                // because when slicing multiples of 32 bytes (lengthmod == 0)
                // the following copy loop was copying the origin's length
                // and then ending prematurely not copying everything it should.
                let mc := add(add(tempBytes, lengthmod), mul(0x20, iszero(lengthmod)))
                let end := add(mc, _length)

                for {
                    // The multiplication in the next line has the same exact purpose
                    // as the one above.
                    let cc := add(add(add(_bytes, lengthmod), mul(0x20, iszero(lengthmod))), _start)
                } lt(mc, end) {
                    mc := add(mc, 0x20)
                    cc := add(cc, 0x20)
                } {
                    mstore(mc, mload(cc))
                }

                mstore(tempBytes, _length)

                //update free-memory pointer
                //allocating the array padded to 32 bytes like the compiler does now
                mstore(0x40, and(add(mc, 31), not(31)))
            }
            //if we want a zero-length slice let's just return a zero-length array
            default {
                tempBytes := mload(0x40)

                //zero out the 32 bytes slice we are about to return
                //we need to do it because Solidity does not garbage collect
                mstore(tempBytes, 0)

                mstore(0x40, add(tempBytes, 0x20))
            }
        }

        return tempBytes;
    }

    /**
     * @notice Slices a byte array with a given starting index up to the end of the original byte
     *         array. Returns a new array rathern than a pointer to the original.
     *
     * @param _bytes Byte array to slice.
     * @param _start Starting index of the slice.
     *
     * @return Slice of the input byte array.
     */
    function slice(bytes memory _bytes, uint256 _start) internal pure returns (bytes memory) {
        if (_start >= _bytes.length) {
            return bytes("");
        }
        return slice(_bytes, _start, _bytes.length - _start);
    }

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

    /**
     * @notice Compares two byte arrays by comparing their keccak256 hashes.
     *
     * @param _bytes First byte array to compare.
     * @param _other Second byte array to compare.
     *
     * @return True if the two byte arrays are equal, false otherwise.
     */
    function equal(bytes memory _bytes, bytes memory _other) internal pure returns (bool) {
        return keccak256(_bytes) == keccak256(_other);
    }
}

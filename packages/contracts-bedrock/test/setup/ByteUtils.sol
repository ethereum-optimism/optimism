// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

library ByteUtils {
    /// @notice Overwrite bytes in-place at a specific offset.
    function overwriteAtOffset(bytes memory _bytes, uint256 _offset, bytes memory _value) internal pure {
        for (uint256 i = 0; i < _value.length; i++) {
            uint256 dataOffset = _offset + i;
            if (dataOffset >= _bytes.length) {
                // Stop writing bytes when we get to the end of _bytes
                break;
            }
            _bytes[dataOffset] = _value[i];
        }
    }
}

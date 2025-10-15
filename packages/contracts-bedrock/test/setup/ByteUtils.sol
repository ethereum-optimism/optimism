// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

library BytesMutator {
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

library BytesFactory {
    /// @notice Create a bytes array from a bytes32
    function fromBytes32(bytes32 value) internal pure returns (bytes memory out_) {
        out_ = new bytes(32);
        for (uint256 i = 0; i < 32; i++) {
            out_[i] = value[i];
        }
    }
}

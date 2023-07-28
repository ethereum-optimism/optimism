// SPDX-License-Identifier: BSD
pragma solidity ^0.8.15;

/// @title Clone
/// @author zefram.eth, Saw-mon & Natalie, clabby
/// @notice Provides helper functions for reading immutable args from calldata
/// @dev Original:
///      https://github.com/Saw-mon-and-Natalie/clones-with-immutable-args/
///      blob/105efee1b9127ed7f6fedf139e1fc796ce8791f2/src/Clone.sol
/// @dev MODIFICATIONS:
///      - Added `_getArgDynBytes` function.
contract Clone {
    uint256 private constant ONE_WORD = 0x20;

    /// @notice Reads an immutable arg with type address
    /// @param argOffset The offset of the arg in the packed data
    /// @return arg The arg value
    function _getArgAddress(uint256 argOffset) internal pure returns (address arg) {
        uint256 offset = _getImmutableArgsOffset();
        assembly {
            arg := shr(0x60, calldataload(add(offset, argOffset)))
        }
    }

    /// @notice Reads an immutable arg with type uint256
    /// @param argOffset The offset of the arg in the packed data
    /// @return arg The arg value
    function _getArgUint256(uint256 argOffset) internal pure returns (uint256 arg) {
        uint256 offset = _getImmutableArgsOffset();
        assembly {
            arg := calldataload(add(offset, argOffset))
        }
    }

    /// @notice Reads an immutable arg with type bytes32
    /// @param argOffset The offset of the arg in the packed data
    /// @return arg The arg value
    function _getArgFixedBytes(uint256 argOffset) internal pure returns (bytes32 arg) {
        uint256 offset = _getImmutableArgsOffset();
        assembly {
            arg := calldataload(add(offset, argOffset))
        }
    }

    /// @notice Reads a uint256 array stored in the immutable args.
    /// @param argOffset The offset of the arg in the packed data
    /// @param arrLen Number of elements in the array
    /// @return arr The array
    function _getArgUint256Array(uint256 argOffset, uint64 arrLen)
        internal
        pure
        returns (uint256[] memory arr)
    {
        uint256 offset = _getImmutableArgsOffset() + argOffset;
        arr = new uint256[](arrLen);

        assembly {
            calldatacopy(add(arr, ONE_WORD), offset, shl(5, arrLen))
        }
    }

    /// @notice Reads a dynamic bytes array stored in the immutable args.
    /// @param argOffset The offset of the arg in the packed data
    /// @param arrLen Number of elements in the array
    /// @return arr The array
    function _getArgDynBytes(uint256 argOffset, uint64 arrLen)
        internal
        pure
        returns (bytes memory arr)
    {
        uint256 offset = _getImmutableArgsOffset() + argOffset;
        arr = new bytes(arrLen);

        assembly {
            calldatacopy(add(arr, ONE_WORD), offset, arrLen)
        }
    }

    /// @notice Reads an immutable arg with type uint64
    /// @param argOffset The offset of the arg in the packed data
    /// @return arg The arg value
    function _getArgUint64(uint256 argOffset) internal pure returns (uint64 arg) {
        uint256 offset = _getImmutableArgsOffset();
        assembly {
            arg := shr(0xc0, calldataload(add(offset, argOffset)))
        }
    }

    /// @notice Reads an immutable arg with type uint8
    /// @param argOffset The offset of the arg in the packed data
    /// @return arg The arg value
    function _getArgUint8(uint256 argOffset) internal pure returns (uint8 arg) {
        uint256 offset = _getImmutableArgsOffset();
        assembly {
            arg := shr(0xf8, calldataload(add(offset, argOffset)))
        }
    }

    /// @return offset The offset of the packed immutable args in calldata
    function _getImmutableArgsOffset() internal pure returns (uint256 offset) {
        assembly {
            offset := sub(calldatasize(), shr(0xf0, calldataload(sub(calldatasize(), 2))))
        }
    }
}

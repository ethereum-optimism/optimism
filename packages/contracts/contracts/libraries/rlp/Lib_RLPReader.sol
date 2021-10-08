// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/**
 * @title Lib_RLPReader
 * @dev Adapted from "RLPReader" by Hamdi Allam (hamdi.allam97@gmail.com).
 */
library Lib_RLPReader {
    /*************
     * Constants *
     *************/

    uint256 internal constant MAX_LIST_LENGTH = 32;

    /*********
     * Enums *
     *********/

    enum RLPItemType {
        DATA_ITEM,
        LIST_ITEM
    }

    /***********
     * Structs *
     ***********/

    struct RLPItem {
        uint256 length;
        uint256 ptr;
    }

    /**********************
     * Internal Functions *
     **********************/

    /**
     * Converts bytes to a reference to memory position and length.
     * @param _in Input bytes to convert.
     * @return Output memory reference.
     */
    function toRLPItem(bytes memory _in) internal pure returns (RLPItem memory) {
        uint256 ptr;
        assembly {
            ptr := add(_in, 32)
        }

        return RLPItem({ length: _in.length, ptr: ptr });
    }

    /**
     * Reads an RLP list value into a list of RLP items.
     * @param _in RLP list value.
     * @return Decoded RLP list items.
     */
    function readList(RLPItem memory _in) internal pure returns (RLPItem[] memory) {
        (uint256 listOffset, , RLPItemType itemType) = _decodeLength(_in);

        require(itemType == RLPItemType.LIST_ITEM, "Invalid RLP list value.");

        // Solidity in-memory arrays can't be increased in size, but *can* be decreased in size by
        // writing to the length. Since we can't know the number of RLP items without looping over
        // the entire input, we'd have to loop twice to accurately size this array. It's easier to
        // simply set a reasonable maximum list length and decrease the size before we finish.
        RLPItem[] memory out = new RLPItem[](MAX_LIST_LENGTH);

        uint256 itemCount = 0;
        uint256 offset = listOffset;
        while (offset < _in.length) {
            require(itemCount < MAX_LIST_LENGTH, "Provided RLP list exceeds max list length.");

            (uint256 itemOffset, uint256 itemLength, ) = _decodeLength(
                RLPItem({ length: _in.length - offset, ptr: _in.ptr + offset })
            );

            out[itemCount] = RLPItem({ length: itemLength + itemOffset, ptr: _in.ptr + offset });

            itemCount += 1;
            offset += itemOffset + itemLength;
        }

        // Decrease the array size to match the actual item count.
        assembly {
            mstore(out, itemCount)
        }

        return out;
    }

    /**
     * Reads an RLP list value into a list of RLP items.
     * @param _in RLP list value.
     * @return Decoded RLP list items.
     */
    function readList(bytes memory _in) internal pure returns (RLPItem[] memory) {
        return readList(toRLPItem(_in));
    }

    /**
     * Reads an RLP bytes value into bytes.
     * @param _in RLP bytes value.
     * @return Decoded bytes.
     */
    function readBytes(RLPItem memory _in) internal pure returns (bytes memory) {
        (uint256 itemOffset, uint256 itemLength, RLPItemType itemType) = _decodeLength(_in);

        require(itemType == RLPItemType.DATA_ITEM, "Invalid RLP bytes value.");

        return _copy(_in.ptr, itemOffset, itemLength);
    }

    /**
     * Reads an RLP bytes value into bytes.
     * @param _in RLP bytes value.
     * @return Decoded bytes.
     */
    function readBytes(bytes memory _in) internal pure returns (bytes memory) {
        return readBytes(toRLPItem(_in));
    }

    /**
     * Reads an RLP string value into a string.
     * @param _in RLP string value.
     * @return Decoded string.
     */
    function readString(RLPItem memory _in) internal pure returns (string memory) {
        return string(readBytes(_in));
    }

    /**
     * Reads an RLP string value into a string.
     * @param _in RLP string value.
     * @return Decoded string.
     */
    function readString(bytes memory _in) internal pure returns (string memory) {
        return readString(toRLPItem(_in));
    }

    /**
     * Reads an RLP bytes32 value into a bytes32.
     * @param _in RLP bytes32 value.
     * @return Decoded bytes32.
     */
    function readBytes32(RLPItem memory _in) internal pure returns (bytes32) {
        require(_in.length <= 33, "Invalid RLP bytes32 value.");

        (uint256 itemOffset, uint256 itemLength, RLPItemType itemType) = _decodeLength(_in);

        require(itemType == RLPItemType.DATA_ITEM, "Invalid RLP bytes32 value.");

        uint256 ptr = _in.ptr + itemOffset;
        bytes32 out;
        assembly {
            out := mload(ptr)

            // Shift the bytes over to match the item size.
            if lt(itemLength, 32) {
                out := div(out, exp(256, sub(32, itemLength)))
            }
        }

        return out;
    }

    /**
     * Reads an RLP bytes32 value into a bytes32.
     * @param _in RLP bytes32 value.
     * @return Decoded bytes32.
     */
    function readBytes32(bytes memory _in) internal pure returns (bytes32) {
        return readBytes32(toRLPItem(_in));
    }

    /**
     * Reads an RLP uint256 value into a uint256.
     * @param _in RLP uint256 value.
     * @return Decoded uint256.
     */
    function readUint256(RLPItem memory _in) internal pure returns (uint256) {
        return uint256(readBytes32(_in));
    }

    /**
     * Reads an RLP uint256 value into a uint256.
     * @param _in RLP uint256 value.
     * @return Decoded uint256.
     */
    function readUint256(bytes memory _in) internal pure returns (uint256) {
        return readUint256(toRLPItem(_in));
    }

    /**
     * Reads an RLP bool value into a bool.
     * @param _in RLP bool value.
     * @return Decoded bool.
     */
    function readBool(RLPItem memory _in) internal pure returns (bool) {
        require(_in.length == 1, "Invalid RLP boolean value.");

        uint256 ptr = _in.ptr;
        uint256 out;
        assembly {
            out := byte(0, mload(ptr))
        }

        require(out == 0 || out == 1, "Lib_RLPReader: Invalid RLP boolean value, must be 0 or 1");

        return out != 0;
    }

    /**
     * Reads an RLP bool value into a bool.
     * @param _in RLP bool value.
     * @return Decoded bool.
     */
    function readBool(bytes memory _in) internal pure returns (bool) {
        return readBool(toRLPItem(_in));
    }

    /**
     * Reads an RLP address value into a address.
     * @param _in RLP address value.
     * @return Decoded address.
     */
    function readAddress(RLPItem memory _in) internal pure returns (address) {
        if (_in.length == 1) {
            return address(0);
        }

        require(_in.length == 21, "Invalid RLP address value.");

        return address(uint160(readUint256(_in)));
    }

    /**
     * Reads an RLP address value into a address.
     * @param _in RLP address value.
     * @return Decoded address.
     */
    function readAddress(bytes memory _in) internal pure returns (address) {
        return readAddress(toRLPItem(_in));
    }

    /**
     * Reads the raw bytes of an RLP item.
     * @param _in RLP item to read.
     * @return Raw RLP bytes.
     */
    function readRawBytes(RLPItem memory _in) internal pure returns (bytes memory) {
        return _copy(_in);
    }

    /*********************
     * Private Functions *
     *********************/

    /**
     * Decodes the length of an RLP item.
     * @param _in RLP item to decode.
     * @return Offset of the encoded data.
     * @return Length of the encoded data.
     * @return RLP item type (LIST_ITEM or DATA_ITEM).
     */
    function _decodeLength(RLPItem memory _in)
        private
        pure
        returns (
            uint256,
            uint256,
            RLPItemType
        )
    {
        require(_in.length > 0, "RLP item cannot be null.");

        uint256 ptr = _in.ptr;
        uint256 prefix;
        assembly {
            prefix := byte(0, mload(ptr))
        }

        if (prefix <= 0x7f) {
            // Single byte.

            return (0, 1, RLPItemType.DATA_ITEM);
        } else if (prefix <= 0xb7) {
            // Short string.

            uint256 strLen = prefix - 0x80;

            require(_in.length > strLen, "Invalid RLP short string.");

            return (1, strLen, RLPItemType.DATA_ITEM);
        } else if (prefix <= 0xbf) {
            // Long string.
            uint256 lenOfStrLen = prefix - 0xb7;

            require(_in.length > lenOfStrLen, "Invalid RLP long string length.");

            uint256 strLen;
            assembly {
                // Pick out the string length.
                strLen := div(mload(add(ptr, 1)), exp(256, sub(32, lenOfStrLen)))
            }

            require(_in.length > lenOfStrLen + strLen, "Invalid RLP long string.");

            return (1 + lenOfStrLen, strLen, RLPItemType.DATA_ITEM);
        } else if (prefix <= 0xf7) {
            // Short list.
            uint256 listLen = prefix - 0xc0;

            require(_in.length > listLen, "Invalid RLP short list.");

            return (1, listLen, RLPItemType.LIST_ITEM);
        } else {
            // Long list.
            uint256 lenOfListLen = prefix - 0xf7;

            require(_in.length > lenOfListLen, "Invalid RLP long list length.");

            uint256 listLen;
            assembly {
                // Pick out the list length.
                listLen := div(mload(add(ptr, 1)), exp(256, sub(32, lenOfListLen)))
            }

            require(_in.length > lenOfListLen + listLen, "Invalid RLP long list.");

            return (1 + lenOfListLen, listLen, RLPItemType.LIST_ITEM);
        }
    }

    /**
     * Copies the bytes from a memory location.
     * @param _src Pointer to the location to read from.
     * @param _offset Offset to start reading from.
     * @param _length Number of bytes to read.
     * @return Copied bytes.
     */
    function _copy(
        uint256 _src,
        uint256 _offset,
        uint256 _length
    ) private pure returns (bytes memory) {
        bytes memory out = new bytes(_length);
        if (out.length == 0) {
            return out;
        }

        uint256 src = _src + _offset;
        uint256 dest;
        assembly {
            dest := add(out, 32)
        }

        // Copy over as many complete words as we can.
        for (uint256 i = 0; i < _length / 32; i++) {
            assembly {
                mstore(dest, mload(src))
            }

            src += 32;
            dest += 32;
        }

        // Pick out the remaining bytes.
        uint256 mask;
        unchecked {
            mask = 256**(32 - (_length % 32)) - 1;
        }

        assembly {
            mstore(dest, or(and(mload(src), not(mask)), and(mload(dest), mask)))
        }
        return out;
    }

    /**
     * Copies an RLP item into bytes.
     * @param _in RLP item to copy.
     * @return Copied bytes.
     */
    function _copy(RLPItem memory _in) private pure returns (bytes memory) {
        return _copy(_in.ptr, 0, _in.length);
    }
}

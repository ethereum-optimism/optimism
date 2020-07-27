pragma solidity ^0.5.0;

/**
 * @title RLPReader
 * @author Hamdi Allam hamdi.allam97@gmail.com
 */
library RLPReader {
    /*
     * Data Structures
     */

    struct RLPItem {
        uint len;
        uint memPtr;
    }


    /*
     * Contract Constants
     */

    uint8 constant private STRING_SHORT_START = 0x80;
    uint8 constant private STRING_LONG_START  = 0xb8;
    uint8 constant private LIST_SHORT_START   = 0xc0;
    uint8 constant private LIST_LONG_START    = 0xf8;
    uint8 constant private WORD_SIZE = 32;


    /*
     * Internal Functions
     */

    /**
     * @param item RLP encoded bytes
     */
    function toRlpItem(
        bytes memory item
    )
        internal
        pure
        returns (RLPItem memory)
    {
        uint memPtr;
        assembly {
            memPtr := add(item, 0x20)
        }

        return RLPItem(item.length, memPtr);
    }

    /**
     * @param item RLP encoded bytes
     */
    function rlpLen(
        RLPItem memory item
    )
        internal
        pure
        returns (uint)
    {
        return item.len;
    }

    /**
     * @param item RLP encoded bytes
     */
    function payloadLen(
        RLPItem memory item
    )
        internal
        pure
        returns (uint)
    {
        return item.len - _payloadOffset(item.memPtr);
    }

    /**
     * @param item RLP encoded list in bytes
     */
    function toList(
        RLPItem memory item
    )
        internal
        pure
        returns (RLPItem[] memory result)
    {
        require(isList(item));

        uint items = numItems(item);
        result = new RLPItem[](items);

        uint memPtr = item.memPtr + _payloadOffset(item.memPtr);
        uint dataLen;
        for (uint i = 0; i < items; i++) {
            dataLen = _itemLength(memPtr);
            result[i] = RLPItem(dataLen, memPtr);
            memPtr = memPtr + dataLen;
        }
    }

    // @return indicator whether encoded payload is a list. negate this function call for isData.
    function isList(
        RLPItem memory item
    )
        internal
        pure
        returns (bool)
    {
        if (item.len == 0) return false;

        uint8 byte0;
        uint memPtr = item.memPtr;
        assembly {
            byte0 := byte(0, mload(memPtr))
        }

        if (byte0 < LIST_SHORT_START)
            return false;
        return true;
    }

    /** RLPItem conversions into data types **/

    // @returns raw rlp encoding in bytes
    function toRlpBytes(
        RLPItem memory item
    )
        internal
        pure
        returns (bytes memory)
    {
        bytes memory result = new bytes(item.len);
        if (result.length == 0) return result;

        uint ptr;
        assembly {
            ptr := add(0x20, result)
        }

        copy(item.memPtr, ptr, item.len);
        return result;
    }

    // any non-zero byte is considered true
    function toBoolean(
        RLPItem memory item
    )
        internal
        pure
        returns (bool)
    {
        require(item.len == 1);
        uint result;
        uint memPtr = item.memPtr;
        assembly {
            result := byte(0, mload(memPtr))
        }

        return result == 0 ? false : true;
    }

    function toAddress(
        RLPItem memory item
    )
        internal
        pure
        returns (address)
    {
        // 1 byte for the length prefix
        require(item.len == 21);

        return address(toUint(item));
    }

    function toUint(
        RLPItem memory item
    )
        internal
        pure
        returns (uint)
    {
        require(item.len > 0 && item.len <= 33);

        uint offset = _payloadOffset(item.memPtr);
        uint len = item.len - offset;

        uint result;
        uint memPtr = item.memPtr + offset;
        assembly {
            result := mload(memPtr)

            // shfit to the correct location if neccesary
            if lt(len, 32) {
                result := div(result, exp(256, sub(32, len)))
            }
        }

        return result;
    }

    // enforces 32 byte length
    function toUintStrict(
        RLPItem memory item
    )
        internal
        pure
        returns (uint)
    {
        // one byte prefix
        require(item.len == 33);

        uint result;
        uint memPtr = item.memPtr + 1;
        assembly {
            result := mload(memPtr)
        }

        return result;
    }

    function toBytes(
        RLPItem memory item
    )
        internal
        pure
        returns (bytes memory)
    {
        require(item.len > 0);

        uint offset = _payloadOffset(item.memPtr);
        uint len = item.len - offset; // data length
        bytes memory result = new bytes(len);

        uint destPtr;
        assembly {
            destPtr := add(0x20, result)
        }

        copy(item.memPtr + offset, destPtr, len);
        return result;
    }


    /*
     * Private Functions
     */

    // @return number of payload items inside an encoded list.
    function numItems(
        RLPItem memory item
    )
        private
        pure
        returns (uint)
    {
        if (item.len == 0) return 0;

        uint count = 0;
        uint currPtr = item.memPtr + _payloadOffset(item.memPtr);
        uint endPtr = item.memPtr + item.len;
        while (currPtr < endPtr) {
           currPtr = currPtr + _itemLength(currPtr); // skip over an item
           count++;
        }

        return count;
    }

    // @return entire rlp item byte length
    function _itemLength(
        uint memPtr
    )
        private
        pure
        returns (uint len)
    {
        uint byte0;
        assembly {
            byte0 := byte(0, mload(memPtr))
        }

        if (byte0 < STRING_SHORT_START)
            return 1;
        
        else if (byte0 < STRING_LONG_START)
            return byte0 - STRING_SHORT_START + 1;

        else if (byte0 < LIST_SHORT_START) {
            assembly {
                let byteLen := sub(byte0, 0xb7) // number of bytes the actual length is
                memPtr := add(memPtr, 1) // skip over the first byte
                
                /* 32 byte word size */
                let dataLen := div(mload(memPtr), exp(256, sub(32, byteLen))) // right shifting to get the len
                len := add(dataLen, add(byteLen, 1))
            }
        }

        else if (byte0 < LIST_LONG_START) {
            return byte0 - LIST_SHORT_START + 1;
        } 

        else {
            assembly {
                let byteLen := sub(byte0, 0xf7)
                memPtr := add(memPtr, 1)

                let dataLen := div(mload(memPtr), exp(256, sub(32, byteLen))) // right shifting to the correct length
                len := add(dataLen, add(byteLen, 1))
            }
        }
    }

    // @return number of bytes until the data
    function _payloadOffset(
        uint memPtr
    )
        private
        pure
        returns (uint)
    {
        uint byte0;
        assembly {
            byte0 := byte(0, mload(memPtr))
        }

        if (byte0 < STRING_SHORT_START) 
            return 0;
        else if (byte0 < STRING_LONG_START || (byte0 >= LIST_SHORT_START && byte0 < LIST_LONG_START))
            return 1;
        else if (byte0 < LIST_SHORT_START)  // being explicit
            return byte0 - (STRING_LONG_START - 1) + 1;
        else
            return byte0 - (LIST_LONG_START - 1) + 1;
    }

    /*
    * @param src Pointer to source
    * @param dest Pointer to destination
    * @param len Amount of memory to copy from the source
    */
    function copy(
        uint src,
        uint dest,
        uint len
    )
        private
        pure
    {
        if (len == 0) return;

        // copy as many word sizes as possible
        for (; len >= WORD_SIZE; len -= WORD_SIZE) {
            assembly {
                mstore(dest, mload(src))
            }

            src += WORD_SIZE;
            dest += WORD_SIZE;
        }

        // left over bytes. Mask is used to remove unwanted bytes from the word
        uint mask = 256 ** (WORD_SIZE - len) - 1;
        assembly {
            let srcpart := and(mload(src), not(mask)) // zero out src
            let destpart := and(mload(dest), mask) // retrieve the bytes
            mstore(dest, or(destpart, srcpart))
        }
    }
}

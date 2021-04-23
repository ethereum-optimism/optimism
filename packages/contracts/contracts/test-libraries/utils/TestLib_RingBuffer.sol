// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_RingBuffer } from "../../optimistic-ethereum/libraries/utils/Lib_RingBuffer.sol";

/**
 * @title TestLib_RingBuffer
 */
contract TestLib_RingBuffer {
    using Lib_RingBuffer for Lib_RingBuffer.RingBuffer;

    Lib_RingBuffer.RingBuffer internal buf;

    function push(
        bytes32 _value,
        bytes27 _extraData
    )
        public
    {
        buf.push(
            _value,
            _extraData
        );
    }

    function get(
        uint256 _index
    )
        public
        view
        returns (
            bytes32
        )
    {
        return buf.get(_index);
    }

    function deleteElementsAfterInclusive(
        uint40 _index,
        bytes27 _extraData
    )
        internal
    {
        return buf.deleteElementsAfterInclusive(
            _index,
            _extraData
        );
    }

    function getLength()
        internal
        view
        returns (
            uint40
        )
    {
        return buf.getLength();
    }

    function getExtraData()
        internal
        view
        returns (
            bytes27
        )
    {
        return buf.getExtraData();
    }
}

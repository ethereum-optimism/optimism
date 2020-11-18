// SPDX-License-Identifier: MIT
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_RingBuffer, iRingBufferOverwriter } from "../../optimistic-ethereum/libraries/utils/Lib_RingBuffer.sol";

/**
 * @title TestLib_RingBuffer
 */
contract TestLib_RingBuffer {
    using Lib_RingBuffer for Lib_RingBuffer.RingBuffer;
    
    Lib_RingBuffer.RingBuffer internal buf;

    function init(
        uint256 _initialBufferSize,
        bytes32 _id,
        iRingBufferOverwriter _overwriter
    )
        public
    {
        buf.init(
            _initialBufferSize,
            _id,
            _overwriter
        );
    }

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

    function push2(
        bytes32 _valueA,
        bytes32 _valueB,
        bytes27 _extraData
    )
        public
    {
        buf.push2(
            _valueA,
            _valueB,
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

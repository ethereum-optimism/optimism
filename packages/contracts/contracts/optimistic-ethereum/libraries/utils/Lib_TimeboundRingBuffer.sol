// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;

/* Logging */
import { console } from "@nomiclabs/buidler/console.sol";

struct TimeboundRingBuffer {
    mapping(uint=>bytes32) elements;
    bytes32 context;
    uint32 maxSize;
    uint32 maxSizeIncrementAmount;
    uint firstElementTimestamp;
    uint timeout;
}

/**
 * @title Lib_TimeboundRingBuffer
 */
library Lib_TimeboundRingBuffer {
    function init(
        TimeboundRingBuffer storage _self,
        uint32 _startingSize,
        uint32 _maxSizeIncrementAmount,
        uint _timeout
    )
        internal
    {
        _self.maxSize = _startingSize;
        _self.maxSizeIncrementAmount = _maxSizeIncrementAmount;
        _self.timeout = _timeout;
        _self.firstElementTimestamp = block.timestamp;
    }

    function push(
        TimeboundRingBuffer storage _self,
        bytes32 _ele,
        bytes28 _extraData
    )
        internal
    {
        uint length = _getLength(_self.context);
        uint maxSize = _self.maxSize;
        if (length == maxSize) {
            if (block.timestamp < _self.firstElementTimestamp + _self.timeout) {
                _self.maxSize += _self.maxSizeIncrementAmount;
                maxSize = _self.maxSize;
            }
        }
        _self.elements[length % maxSize] = _ele;
        _self.context = makeContext(uint32(length+1), _extraData);
    }

    function push2(
        TimeboundRingBuffer storage _self,
        bytes32 _ele1,
        bytes32 _ele2,
        bytes28 _extraData
    )
        internal
    {
        uint length = _getLength(_self.context);
        uint maxSize = _self.maxSize;
        if (length + 1 >= maxSize) {
            if (block.timestamp < _self.firstElementTimestamp + _self.timeout) {
                // Because this is a push2 we need to at least increment by 2
                _self.maxSize += _self.maxSizeIncrementAmount > 1 ? _self.maxSizeIncrementAmount : 2;
                maxSize = _self.maxSize;
            }
        }
        _self.elements[length % maxSize] = _ele1;
        _self.elements[(length + 1) % maxSize] = _ele2;
        _self.context = makeContext(uint32(length+2), _extraData);
    }

    function makeContext(
        uint32 _length,
        bytes28 _extraData
    )
        internal
        pure
        returns(
            bytes32
        )
    {
        return bytes32(_extraData) | bytes32(uint256(_length));
    }

    function getLength(
        TimeboundRingBuffer storage _self
    )
        internal
        view
        returns(
            uint32
        )
    {
        return _getLength(_self.context);
    }

    function _getLength(
        bytes32 context
    )
        internal
        pure
        returns(
            uint32
        )
    {
        bytes32 lengthMask = 0x00000000000000000000000000000000000000000000000000000000ffffffff;
        return uint32(uint256(context & lengthMask));
    }

    function getExtraData(
        TimeboundRingBuffer storage _self
    )
        internal
        view
        returns(
            bytes28
        )
    {
        bytes32 extraDataMask = 0xffffffffffffffffffffffffffffffffffffffffffffffffffffffff00000000;
        return bytes28(_self.context & extraDataMask);
    }

    function get(
        TimeboundRingBuffer storage _self,
        uint32 _index
    )
        internal
        view
        returns(
            bytes32
        )
    {
        uint length = _getLength(_self.context);
        require(_index < length, "Index too large.");
        require(length - _index <= _self.maxSize, "Index too old & has been overridden.");
        return _self.elements[_index % _self.maxSize];
    }
}
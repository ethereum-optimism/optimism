// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Logging */
import { console } from "@nomiclabs/buidler/console.sol";

/* Library Imports */
import { TimeboundRingBuffer, Lib_TimeboundRingBuffer } from "../../optimistic-ethereum/libraries/utils/Lib_TimeboundRingBuffer.sol";

/**
 * @title TestLib_TimeboundRingBuffer
 */
contract TestLib_TimeboundRingBuffer {
    using Lib_TimeboundRingBuffer for TimeboundRingBuffer;
    
    TimeboundRingBuffer public list;

    constructor (
        uint32 _startingSize,
        uint32 _maxSizeIncrementAmount,
        uint _timeout
    )
        public
    {
        list.init(_startingSize, _maxSizeIncrementAmount, _timeout);
    }

    function push(bytes32 _ele, bytes28 _extraData) public {
        list.push(_ele, _extraData);
    }

    function push2(bytes32 _ele1, bytes32 _ele2, bytes28 _extraData) public {
        list.push2(_ele1, _ele2, _extraData);
    }

    function get(uint32 index) public view returns(bytes32) {
        return list.get(index);
    }

    function getLength() public view returns(uint32) {
        return list.getLength();
    }

    function getExtraData() public view returns(bytes28) {
        return list.getExtraData();
    }

    function getMaxSize() public view returns(uint32) {
        return list.maxSize;
    }

    function getMaxSizeIncrementAmount() public view returns(uint32) {
        return list.maxSizeIncrementAmount;
    }

    function getFirstElementTimestamp() public view returns(uint) {
        return list.firstElementTimestamp;
    }

    function getTimeout() public view returns(uint) {
        return list.timeout;
    }
}
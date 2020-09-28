// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";

/* Interface Imports */
import { iOVM_BaseQueue } from "../../iOVM/queue/iOVM_BaseQueue.sol";

/**
 * @title OVM_BaseQueue
 */
contract OVM_BaseQueue is iOVM_BaseQueue {

    /****************************************
     * Contract Variables: Internal Storage *
     ****************************************/

    Lib_OVMCodec.QueueElement[] internal queue;
    uint256 internal front;


    /**********************
     * Function Modifiers *
     **********************/

    /**
     * Asserts that the queue is not empty.
     */
    modifier notEmpty() {
        require(
            size() > 0,
            "Queue is empty."
        );
        _;
    }


    /**********************************
     * Public Functions: Queue Access *
     **********************************/

    /**
     * Gets the size of the queue.
     * @return _size Number of elements in the queue.
     */
    function size()
        override
        public
        view
        returns (
            uint256 _size
        )
    {
        return front >= queue.length ? 0 : queue.length - front;
    }

    /**
     * Gets the top element of the queue.
     * @return _element First element in the queue.
     */
    function peek()
        override
        public
        view
        notEmpty
        returns (
            Lib_OVMCodec.QueueElement memory _element
        )
    {
        return queue[front];
    }


    /******************************************
     * Internal Functions: Queue Manipulation *
     ******************************************/

    /**
     * Adds an element to the queue.
     * @param _element Queue element to add to the queue.
     */
    function _enqueue(
        Lib_OVMCodec.QueueElement memory _element
    )
        internal
    {
        queue.push(_element);
    }

    /**
     * Pops an element from the queue.
     * @return _element Queue element popped from the queue.
     */
    function _dequeue()
        internal
        notEmpty
        returns (
            Lib_OVMCodec.QueueElement memory _element
        )
    {
        _element = queue[front];
        delete queue[front];
        front += 1;
        return _element;
    }
}

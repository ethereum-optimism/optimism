// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";

/* Interface Imports */
import { iOVM_BaseQueue } from "./iOVM_BaseQueue.sol";

/**
 * @title iOVM_L1ToL2TransactionQueue
 */
interface iOVM_L1ToL2TransactionQueue is iOVM_BaseQueue {

    /****************************************
     * Public Functions: Queue Manipulation *
     ****************************************/

    function enqueue(Lib_OVMCodec.QueueElement memory _element) external;
    function dequeue() external returns (Lib_OVMCodec.QueueElement memory _element);
}

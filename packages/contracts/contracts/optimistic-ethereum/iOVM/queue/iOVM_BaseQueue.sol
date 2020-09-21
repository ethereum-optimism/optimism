// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";

/**
 * @title iOVM_BaseQueue
 */
interface iOVM_BaseQueue {

    /**********************************
     * Public Functions: Queue Access *
     **********************************/

    function size() external view returns (uint256 _size);
    function peek() external view returns (Lib_OVMCodec.QueueElement memory _element);
}

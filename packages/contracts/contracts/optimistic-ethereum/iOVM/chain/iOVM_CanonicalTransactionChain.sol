// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_BaseChain } from "./iOVM_BaseChain.sol";

/**
 * @title iOVM_CanonicalTransactionChain
 */
interface iOVM_CanonicalTransactionChain is iOVM_BaseChain {

    /****************************************
     * Public Functions: Batch Manipulation *
     ****************************************/

    function enqueue(address _target, uint256 _gasLimit, bytes memory _data) external;
    function appendQueueBatch() external;
    function appendSequencerBatch(bytes[] calldata _batch, uint256 _timestamp) external;
}

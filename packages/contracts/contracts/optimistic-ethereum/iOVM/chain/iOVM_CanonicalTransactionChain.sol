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

    function appendQueueBatch() external;
    function appendSequencerBatch(bytes[] calldata _batch, uint256 _timestamp) external;
}

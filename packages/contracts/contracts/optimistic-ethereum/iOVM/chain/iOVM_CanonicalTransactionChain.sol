// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";

/* Interface Imports */
import { iOVM_BaseChain } from "./iOVM_BaseChain.sol";

/**
 * @title iOVM_CanonicalTransactionChain
 */
interface iOVM_CanonicalTransactionChain is iOVM_BaseChain {

    /**********
     * Events *
     **********/

    event TransactionEnqueued(
        address _l1TxOrigin,
        address _target,
        uint256 _gasLimit,
        bytes _data,
        uint256 _queueIndex,
        uint256 _timestamp
    );

    event QueueBatchAppended(
        uint256 _startingQueueIndex,
        uint256 _numQueueElements
    );

    event SequencerBatchAppended(
        uint256 _startingQueueIndex,
        uint256 _numQueueElements
    );


    /***********
     * Structs *
     ***********/

    struct BatchContext {
        uint256 numSequencedTransactions;
        uint256 numSubsequentQueueTransactions;
        uint256 timestamp;
        uint256 blockNumber;
        uint256 index;
    }

    struct TransactionChainElement {
        bool isSequenced;
        uint256 queueIndex;  // QUEUED TX ONLY
        uint256 timestamp;   // SEQUENCER TX ONLY
        uint256 blockNumber; // SEQUENCER TX ONLY
        bytes txData;        // SEQUENCER TX ONLY
    }


    /********************
     * Public Functions *
     ********************/

    /**
     * Gets the queue element at a particular index.
     * @param _index Index of the queue element to access.
     * @return _element Queue element at the given index.
     */
    function getQueueElement(
        uint256 _index
    )
        external
        view
        returns (
            Lib_OVMCodec.QueueElement memory _element
        );

    /**
     * Adds a transaction to the queue.
     * @param _target Target contract to send the transaction to.
     * @param _gasLimit Gas limit for the given transaction.
     * @param _data Transaction data.
     */
    function enqueue(
        address _target,
        uint256 _gasLimit,
        bytes memory _data
    ) external;

    /**
     * Appends a given number of queued transactions as a single batch.
     * @param _numQueuedTransactions Number of transactions to append.
     */
    function appendQueueBatch(
        uint256 _numQueuedTransactions
    ) external;

    // /**
    //  * Allows the sequencer to append a batch of transactions.
    //  * @param _transactions Array of raw transaction data.
    //  * @param _contexts Array of batch contexts.
    //  * @param _shouldStartAtBatch Specific batch we expect to start appending to.
    //  * @param _totalElementsToAppend Total number of batch elements we expect to append.
    //  */
    function appendSequencerBatch(
        // uint256 _shouldStartAtBatch,
        // uint _totalElementsToAppend,
        // BatchContext[] memory _contexts,
        // bytes[] memory _transactions
    ) external;
}

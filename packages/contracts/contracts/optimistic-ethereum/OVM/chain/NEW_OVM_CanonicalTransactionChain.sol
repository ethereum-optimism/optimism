// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";
import { Lib_AddressResolver } from "../../libraries/resolver/Lib_AddressResolver.sol";
import { Lib_MerkleUtils } from "../../libraries/utils/Lib_MerkleUtils.sol";

/* Interface Imports */
import { iOVM_CanonicalTransactionChain } from "../../iOVM/chain/iOVM_CanonicalTransactionChain.sol";
import { iOVM_L1ToL2TransactionQueue } from "../../iOVM/queue/iOVM_L1ToL2TransactionQueue.sol";

/* Contract Imports */
import { OVM_BaseChain } from "./OVM_BaseChain.sol";

/**
 * @title OVM_CanonicalTransactionChain
 */
contract NEW_OVM_CanonicalTransactionChain is OVM_BaseChain, Lib_AddressResolver { // TODO: re-add iOVM_CanonicalTransactionChain

    struct SeqeuncerBatchContext {
        uint numSequencedTransactions;
        uint numSubsequentQueueTransactions;
        uint timestamp;
        uint blocknumber;
    }

    struct TransactionChainElement {
        bool isSequenced;
        uint queueIndex; // unused if isSequenced
        uint timestamp; // unused if !isSequenced
        uint blocknumber; // unused if !isSequenced
        bytes txData; // unused if !isSequenced
    }

    /*******************************************
     * Contract Variables: Contract References *
     *******************************************/
    
    iOVM_L1ToL2TransactionQueue internal ovmL1ToL2TransactionQueue;


    /*******************************************
     * Contract Variables: Internal Accounting *
     *******************************************/

    uint256 internal forceInclusionPeriodSeconds;
    uint256 internal lastOVMTimestamp;


    /***************
     * Constructor *
     ***************/

    /**
     * @param _libAddressManager Address of the Address Manager.
     * @param _forceInclusionPeriodSeconds Period during which only the sequencer can submit.
     */
    constructor(
        address _libAddressManager,
        uint256 _forceInclusionPeriodSeconds
    )
        Lib_AddressResolver(_libAddressManager)
    {
        ovmL1ToL2TransactionQueue = iOVM_L1ToL2TransactionQueue(resolve("OVM_L1ToL2TransactionQueue"));
        forceInclusionPeriodSeconds = _forceInclusionPeriodSeconds;
    }


    /****************************************
     * Public Functions: Batch Manipulation *
     ****************************************/


// TODO: allow the sequencer to append queue batches at any time


    /**
     * Appends a sequencer batch.
     */
    function appendSequencerBatches(
        bytes[] memory _rawTransactions,
        SeqeuncerBatchContext[] memory _multiBatchContext,
        uint256 _shouldStartAtBatch,
        uint _totalElementsToAppend
    )
        // override
        public // TODO: can we make external?  Hopefully so
    {
        require(
            _shouldStartAtBatch == getTotalBatches(),
            "Batch submission failed: chain length has become larger than expected"
        );

        require(
            msg.sender == resolve("Sequencer"),
            "Function can only be called by the Sequencer."
        );

        if (ovmL1ToL2TransactionQueue.size() > 0) {
            require(
                block.timestamp < ovmL1ToL2TransactionQueue.peek().timestamp + forceInclusionPeriodSeconds,
                "Older queue batches must be processed before a new sequencer batch."
            );
        }

        bytes32[] memory leaves = new bytes32[](_totalElementsToAppend);

        uint numBatches = _multiBatchContext.length;
        uint transactionIndex = 0;
        uint numSequencerTransactionsProcessed = 0;
        for (uint batchIndex = 0; batchIndex < numBatches; batchIndex++) {
            SeqeuncerBatchContext memory curBatch = _multiBatchContext[batchIndex];
            uint numSequencedTransactions = curBatch.numSequencedTransactions;
            for (uint txIndex = 0; txIndex < numSequencedTransactions; txIndex++) {
                TransactionChainElement memory element = TransactionChainElement({
                    isSequenced: true,
                    queueIndex: 0,
                    timestamp: curBatch.timestamp,
                    blocknumber: curBatch.blocknumber,
                    txData: _rawTransactions[numSequencerTransactionsProcessed]
                });
                leaves[transactionIndex] = _hashTransactionChainElement(element);
                numSequencerTransactionsProcessed++;
                transactionIndex++;
            }

            uint numQueuedTransactions = curBatch.numSubsequentQueueTransactions;
            for (uint queueTxIndex = 0; queueTxIndex < numQueuedTransactions; queueTxIndex++) {
                TransactionChainElement memory element = TransactionChainElement({
                    isSequenced: false,
                    queueIndex: ovmL1ToL2TransactionQueue.size(),
                    timestamp: 0,
                    blocknumber: 0,
                    txData: hex""
                });
                leaves[transactionIndex] = _hashTransactionChainElement(element);
                transactionIndex++;
                // todo: dequeue however it works now (peeked?)
            }
        }

        bytes32 root;
        // todo: get root from merkle utils on leaves
        _appendQueueBatch(root, _batch.length);
    }


    /******************************************
     * Internal Functions: Batch Manipulation *
     ******************************************/

    /**
     * Appends a queue batch to the chain.
     * @param _queueElement Queue element to append.
     * @param _batchSize Number of elements in the batch.
     */
    function _appendQueueBatch(
        bytes32 _batchRoot,
        uint256 _batchSize
    )
        internal
    {
        Lib_OVMCodec.ChainBatchHeader memory batchHeader = Lib_OVMCodec.ChainBatchHeader({
            batchIndex: getTotalBatches(),
            batchRoot: _batchRoot,
            batchSize: _batchSize,
            prevTotalElements: getTotalElements(),
            extraData: hex""
        });

        _appendBatch(batchHeader);
        lastOVMTimestamp = _queueElement.timestamp;
    }

    // TODO docstring
    function _hashTransactionChainElement(
        TransactionChainElement memory _element
    )
        internal
        returns(bytes32)
    {
        return keccak256(abi.encode(
            _element.isSequenced,
            _element.queueIndex,
            _element.timestamp,
            _element.blocknumber,
            _element.txData
        ));
    }
}
// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";
import { Lib_AddressResolver } from "../../libraries/resolver/Lib_AddressResolver.sol";
import { Lib_MerkleUtils } from "../../libraries/utils/Lib_MerkleUtils.sol";
import { TimeboundRingBuffer, Lib_TimeboundRingBuffer } from "../../libraries/utils/Lib_TimeboundRingBuffer.sol";
import { console } from "@nomiclabs/buidler/console.sol";

/* Interface Imports */
import { iOVM_CanonicalTransactionChain } from "../../iOVM/chain/iOVM_CanonicalTransactionChain.sol";
import { iOVM_L1ToL2TransactionQueue } from "../../iOVM/queue/iOVM_L1ToL2TransactionQueue.sol";

/* Contract Imports */
import { OVM_BaseChain } from "./OVM_BaseChain.sol";

/**
 * @title OVM_CanonicalTransactionChain
 */
contract NEW_OVM_CanonicalTransactionChain is OVM_BaseChain, Lib_AddressResolver { // TODO: re-add iOVM_CanonicalTransactionChain

    /*************************************************
     * Contract Variables: Transaction Restrinctions *
     *************************************************/

    uint constant MAX_ROLLUP_TX_SIZE = 10000;
    uint constant L2_GAS_DISCOUNT_DIVISOR = 10;

    using Lib_TimeboundRingBuffer for TimeboundRingBuffer;
    TimeboundRingBuffer internal queue;

    struct MultiBatchContext {
        uint numSequencedTransactions;
        uint numSubsequentQueueTransactions;
        uint timestamp;
        uint blocknumber;
    }

    struct TransactionChainElement {
        bool isSequenced;
        uint queueIndex;  // QUEUED TX ONLY
        uint timestamp;   // SEQUENCER TX ONLY
        uint blocknumber; // SEQUENCER TX ONLY
        bytes txData;     // SEQUENCER TX ONLY
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
    address internal sequencerAddress;


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
        sequencerAddress = resolve("OVM_Sequencer");
        forceInclusionPeriodSeconds = _forceInclusionPeriodSeconds;
        queue.init(100, 50, 0); // TODO: Update once we have arbitrary condition
    }


    /***************************************
     * Public Functions: Transaction Queue *
     **************************************/

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
    )
        public
    {
        require(
            _data.length <= MAX_ROLLUP_TX_SIZE,
            "Transaction exceeds maximum rollup data size."
        );
        require(_gasLimit >= 20000, "Gas limit too low.");

        // Consume l1 gas to pay for l2 gas on l1
        uint gasToConsume = _gasLimit/L2_GAS_DISCOUNT_DIVISOR;
        uint startingGas = gasleft();
        uint i;
        while(startingGas - gasleft() > gasToConsume) {
            i++; // TODO: Replace this dumb work with minting gas token.
        }

        Lib_OVMCodec.QueueElement memory element = Lib_OVMCodec.QueueElement({
            timestamp: block.timestamp,
            batchRoot: keccak256(abi.encodePacked(
                _target,
                _gasLimit,
                _data
            )),
            isL1ToL2Batch: true
        });

        // bytes32 extraData = timestamp + blockNumber
        // queue.push2(batchRoot, );
    }


    /****************************************
     * Public Functions: Batch Manipulation *
     ****************************************/


    // TODO: allow the sequencer to append queue batches at any time


    /**
     * Appends a sequencer batch.
     */
    function appendSequencerMultiBatch(
        bytes[] memory _rawTransactions,                // 2 byte prefix for how many elements, per element 3 byte prefix.
        MultiBatchContext[] memory _multiBatchContexts, // 2 byte prefix for how many elements, fixed size elements
        uint256 _shouldStartAtBatch,                    // 6 bytes
        uint _totalElementsToAppend                     // 2 btyes
    )
        // override
        public // TODO: can we make external?  Hopefully so
    {
        require(
            _shouldStartAtBatch == getTotalBatches(),
            "Batch submission failed: chain length has become larger than expected"
        );

        require(
            msg.sender == sequencerAddress,
            "Function can only be called by the Sequencer."
        );

        if (ovmL1ToL2TransactionQueue.size() > 0) {
            require(
                block.timestamp < ovmL1ToL2TransactionQueue.peek().timestamp + forceInclusionPeriodSeconds,
                "Older queue batches must be processed before a new sequencer batch."
            );
        }

        // Initialize an array which will contain the leaves of the merkle tree commitment
        bytes32[] memory leaves = new bytes32[](_totalElementsToAppend);
        uint numBatchContexts = _multiBatchContexts.length;
        uint transactionIndex = 0;
        uint numSequencerTransactionsProcessed = 0;
        for (uint batchContextIndex = 0; batchContextIndex < numBatchContexts; batchContextIndex++) {

            // Process Sequencer Transactions
            MultiBatchContext memory curContext = _multiBatchContexts[batchContextIndex];
            uint numSequencedTransactions = curContext.numSequencedTransactions;
            for (uint txIndex = 0; txIndex < numSequencedTransactions; txIndex++) {
                TransactionChainElement memory element = TransactionChainElement({
                    isSequenced: true,
                    queueIndex: 0,
                    timestamp: curContext.timestamp,
                    blocknumber: curContext.blocknumber,
                    txData: _rawTransactions[numSequencerTransactionsProcessed]
                });
                leaves[transactionIndex] = _hashTransactionChainElement(element);
                numSequencerTransactionsProcessed++;
                transactionIndex++;
                console.log("Processed a sequencer transaction");
                console.logBytes32(_hashTransactionChainElement(element));
            }

            // Process Queue Transactions
            uint numQueuedTransactions = curContext.numSubsequentQueueTransactions;
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
                ovmL1ToL2TransactionQueue.dequeue();
            }
        }

        console.log("We reached the end!");

        bytes32 root;

        // Make sure the correct number of leaves were calculated
        require(transactionIndex == _totalElementsToAppend, "Not enough transactions supplied!");

        // TODO: get root from merkle utils on leaves
        // merklize(leaves);
        // _appendQueueBatch(root, _batch.length);
    }


    /******************************************
     * Internal Functions: Batch Manipulation *
     ******************************************/

    /**
     * Appends a queue batch to the chain.
     * @param _batchRoot Root of the batch
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
        // lastOVMTimestamp = _queueElement.timestamp;
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
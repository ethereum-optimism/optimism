// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Proxy Imports */
import { Proxy_Resolver } from "../../proxy/Proxy_Resolver.sol";

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";
import { Lib_MerkleUtils } from "../../libraries/utils/Lib_MerkleUtils.sol";

/* Interface Imports */
import { iOVM_CanonicalTransactionChain } from "../../iOVM/chain/iOVM_CanonicalTransactionChain.sol";
import { iOVM_L1ToL2TransactionQueue } from "../../iOVM/queue/iOVM_L1ToL2TransactionQueue.sol";

/* Contract Imports */
import { OVM_BaseChain } from "./OVM_BaseChain.sol";

/**
 * @title OVM_CanonicalTransactionChain
 */
contract OVM_CanonicalTransactionChain is iOVM_CanonicalTransactionChain, OVM_BaseChain, Proxy_Resolver {

    /*******************************************
     * Contract Variables: Contract References *
     *******************************************/
    
    iOVM_L1ToL2TransactionQueue internal ovmL1ToL2TransactionQueue;
    address internal sequencer;


    /*******************************************
     * Contract Variables: Internal Accounting *
     *******************************************/

    uint256 internal forceInclusionPeriodSeconds;
    uint256 internal lastOVMTimestamp;


    /***************
     * Constructor *
     ***************/

    /**
     * @param _proxyManager Address of the Proxy_Manager.
     * @param _forceInclusionPeriodSeconds Period during which only the sequencer can submit.
     */
    constructor(
        address _proxyManager,
        uint256 _forceInclusionPeriodSeconds
    )
        Proxy_Resolver(_proxyManager)
    {
        ovmL1ToL2TransactionQueue = iOVM_L1ToL2TransactionQueue(resolve("OVM_L1ToL2TransactionQueue"));
        sequencer = resolve("Sequencer");

        forceInclusionPeriodSeconds = _forceInclusionPeriodSeconds;
    }


    /****************************************
     * Public Functions: Batch Manipulation *
     ****************************************/

    /**
     * Appends a batch from the L1ToL2TransactionQueue.
     */
    function appendQueueBatch()
        override
        public
    {
        require(
            ovmL1ToL2TransactionQueue.size() > 0 == true,
            "No batches are currently queued to be appended."
        );

        Lib_OVMCodec.QueueElement memory queueElement = ovmL1ToL2TransactionQueue.peek();
        
        require(
            queueElement.timestamp + forceInclusionPeriodSeconds <= block.timestamp,
            "Cannot append until the inclusion delay period has elapsed."
        );

        _appendQueueBatch(queueElement, 1);
        ovmL1ToL2TransactionQueue.dequeue();
    }

    /**
     * Appends a sequencer batch.
     * @param _batch Batch of transactions to append.
     * @param _timestamp Timestamp for the provided batch.
     */
    function appendSequencerBatch(
        bytes[] memory _batch,
        uint256 _timestamp
    )
        override
        public
    {
        require(
            msg.sender == sequencer,
            "Function can only be called by the sequencer."
        );

        require(
            _timestamp >= lastOVMTimestamp,
            "Batch timestamp must be later than the last OVM timestamp."
        );

        if (ovmL1ToL2TransactionQueue.size() > 0) {
            require(
                _timestamp <= ovmL1ToL2TransactionQueue.peek().timestamp,
                "Older queue batches must be processed before a newer sequencer batch."
            );
        }

        Lib_OVMCodec.QueueElement memory queueElement = Lib_OVMCodec.QueueElement({
            timestamp: _timestamp,
            batchRoot: Lib_MerkleUtils.getMerkleRoot(_batch),
            isL1ToL2Batch: false
        });
        _appendQueueBatch(queueElement, _batch.length);
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
        Lib_OVMCodec.QueueElement memory _queueElement,
        uint256 _batchSize
    )
        internal
    {
        Lib_OVMCodec.ChainBatchHeader memory batchHeader = Lib_OVMCodec.ChainBatchHeader({
            batchIndex: getTotalBatches(),
            batchRoot: _queueElement.batchRoot,
            batchSize: _batchSize,
            prevTotalElements: getTotalElements(),
            extraData: abi.encodePacked(
                _queueElement.timestamp,
                _queueElement.isL1ToL2Batch
            )
        });

        _appendBatch(batchHeader);
        lastOVMTimestamp = _queueElement.timestamp;
    }
}

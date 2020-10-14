// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";
import { Lib_AddressResolver } from "../../libraries/resolver/Lib_AddressResolver.sol";
import { Lib_MerkleUtils } from "../../libraries/utils/Lib_MerkleUtils.sol";
import { Lib_MerkleRoot } from "../../libraries/utils/Lib_MerkleRoot.sol";
import { TimeboundRingBuffer, Lib_TimeboundRingBuffer } from "../../libraries/utils/Lib_TimeboundRingBuffer.sol";

/* Interface Imports */
import { iOVM_BaseChain } from "../../iOVM/chain/iOVM_BaseChain.sol";
import { iOVM_CanonicalTransactionChain } from "../../iOVM/chain/iOVM_CanonicalTransactionChain.sol";

/* Contract Imports */
import { OVM_BaseChain } from "./OVM_BaseChain.sol";

/* Logging Imports */
import { console } from "@nomiclabs/buidler/console.sol";

/**
 * @title OVM_CanonicalTransactionChain
 */
contract OVM_CanonicalTransactionChain is iOVM_CanonicalTransactionChain, OVM_BaseChain, Lib_AddressResolver {

    /*************
     * Constants *
     *************/

    uint256 constant public MIN_ROLLUP_TX_GAS = 20000;
    uint256 constant public MAX_ROLLUP_TX_SIZE = 10000;
    uint256 constant public L2_GAS_DISCOUNT_DIVISOR = 10;


    /*************
     * Variables *
     *************/

    uint256 internal forceInclusionPeriodSeconds;
    uint256 internal lastOVMTimestamp;
    address internal sequencer;

    using Lib_TimeboundRingBuffer for TimeboundRingBuffer;
    TimeboundRingBuffer internal queue;
    TimeboundRingBuffer internal chain;


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
        sequencer = resolve("OVM_Sequencer");
        forceInclusionPeriodSeconds = _forceInclusionPeriodSeconds;

        queue.init(100, 50, 10000000000); // TODO: Update once we have arbitrary condition
        batches.init(100, 50, 10000000000); // TODO: Update once we have arbitrary condition
    }


    /********************
     * Public Functions *
     ********************/

    function getTotalElements()
        override(OVM_BaseChain, iOVM_BaseChain) 
        public
        view
        returns (
            uint256 _totalElements
        )
    {
        (uint40 totalElements,) = _getLatestBatchContext();
        return uint256(totalElements);
    }

    /**
     * @inheritdoc iOVM_CanonicalTransactionChain
     */
    function getQueueElement(
        uint256 _index
    )
        override
        public
        view
        returns (
            Lib_OVMCodec.QueueElement memory _element
        )
    {
        uint32 trueIndex = uint32(_index * 2);
        bytes32 queueRoot = queue.get(trueIndex);
        bytes32 timestampAndBlockNumber = queue.get(trueIndex + 1);

        uint40 elementTimestamp;
        uint32 elementBlockNumber;
        assembly {
            elementTimestamp := and(timestampAndBlockNumber, 0x000000000000000000000000000000000000000000000000000000ffffffffff)
            elementBlockNumber := shr(40, and(timestampAndBlockNumber, 0xffffffffffffffffffffffffffffffffffffffffffffffffffffff0000000000))
        }

        return Lib_OVMCodec.QueueElement({
            queueRoot: queueRoot,
            timestamp: elementTimestamp,
            blockNumber: elementBlockNumber
        });
    }

    /**
     * @inheritdoc iOVM_CanonicalTransactionChain
     */
    function enqueue(
        address _target,
        uint256 _gasLimit,
        bytes memory _data
    )
        override
        public
    {
        require(
            _data.length <= MAX_ROLLUP_TX_SIZE,
            "Transaction exceeds maximum rollup data size."
        );

        require(
            _gasLimit >= MIN_ROLLUP_TX_GAS,
            "Layer 2 gas limit too low to enqueue."
        );

        uint256 gasToConsume = _gasLimit/L2_GAS_DISCOUNT_DIVISOR;
        uint256 startingGas = gasleft();

        // Although this check is not necessary (burn below will run out of gas if not true), it
        // gives the user an explicit reason as to why the enqueue attempt failed.
        require(
            startingGas > gasToConsume,
            "Insufficient gas for L2 rate limiting burn."
        );

        // We need to consume some amount of L1 gas in order to rate limit transactions going into
        // L2. However, L2 is cheaper than L1 so we only need to burn some small proportion of the
        // provided L1 gas.
        //
        // Here we do some "dumb" work in order to burn gas, although we should probably replace
        // this with something like minting gas token later on.
        uint256 i;
        while(startingGas - gasleft() < gasToConsume) {
            i++;
        }

        bytes memory transaction = abi.encode(
            msg.sender,
            _target,
            _gasLimit,
            _data
        );

        bytes32 transactionHash = keccak256(transaction);
        bytes32 timestampAndBlockNumber;
        assembly {
            timestampAndBlockNumber := or(timestamp(), shl(40, number()))
        }

        queue.push2(transactionHash, timestampAndBlockNumber, bytes28(0));

        (, uint32 nextQueueIndex) = _getLatestBatchContext();
        // TODO: Evaluate if we need timestamp
        emit TransactionEnqueued(
            msg.sender,
            _target,
            _gasLimit,
            _data,
            nextQueueIndex - 1,
            block.timestamp
        );
    }

    /**
     * @inheritdoc iOVM_CanonicalTransactionChain
     */
    function appendQueueBatch(
        uint _numQueuedTransactions
    )
        override
        public
    {
        require(
            _numQueuedTransactions > 0,
            "Must append more than zero transactions."
        );

        (uint40 totalElements, uint32 nextQueueIndex) = _getLatestBatchContext();

        bytes32[] memory leaves = new bytes32[](_numQueuedTransactions);
        for (uint i = 0; i < _numQueuedTransactions; i++) {
            leaves[i] = _getQueueLeafHash(nextQueueIndex);
            nextQueueIndex++;
        }

        _appendBatch(
            Lib_MerkleRoot.getMerkleRoot(leaves),
            _numQueuedTransactions,
            _numQueuedTransactions
        );

        emit QueueBatchAppended(
            nextQueueIndex - _numQueuedTransactions,
            _numQueuedTransactions
        );
    }

    /**
     * @inheritdoc iOVM_CanonicalTransactionChain
     */
    function appendSequencerBatch(
        bytes[] memory _transactions,
        BatchContext[] memory _contexts,
        uint256 _shouldStartAtBatch,
        uint _totalElementsToAppend
    )
        override
        public
    {
        require(
            _shouldStartAtBatch == getTotalBatches(),
            "Actual batch start index does not match expected start index."
        );

        require(
            msg.sender == sequencer,
            "Function can only be called by the Sequencer."
        );

        require(
            _contexts.length > 0,
            "Must provide at least one batch context."
        );

        require(
            _totalElementsToAppend > 0,
            "Must append at least one element."
        );

        bytes32[] memory leaves = new bytes32[](_totalElementsToAppend);
        uint32 transactionIndex = 0;
        uint32 numSequencerTransactionsProcessed = 0;
        (, uint32 nextQueueIndex) = _getLatestBatchContext();

        for (uint32 i = 0; i < _contexts.length; i++) {
            BatchContext memory context = _contexts[i];
            _validateBatchContext(context, nextQueueIndex);

            for (uint32 i = 0; i < context.numSequencedTransactions; i++) {
                leaves[transactionIndex] = _hashTransactionChainElement(
                    TransactionChainElement({
                        isSequenced: true,
                        queueIndex: 0,
                        timestamp: context.timestamp,
                        blockNumber: context.blockNumber,
                        txData: _transactions[numSequencerTransactionsProcessed]
                    })
                );
                numSequencerTransactionsProcessed++;
                transactionIndex++;
            }

            for (uint32 i = 0; i < context.numSubsequentQueueTransactions; i++) {
                leaves[transactionIndex] = _getQueueLeafHash(nextQueueIndex);
                nextQueueIndex++;
                transactionIndex++;
            }
        }

        require(
            numSequencerTransactionsProcessed == _transactions.length,
            "Not all sequencer transactions were processed."
        );
        require(
            transactionIndex == _totalElementsToAppend,
            "Actual transaction index does not match expected total elements to append."
        );

        uint256 numQueuedTransactions = _totalElementsToAppend - numSequencerTransactionsProcessed;
        _appendBatch(
            Lib_MerkleRoot.getMerkleRoot(leaves),
            _totalElementsToAppend,
            numQueuedTransactions
        );

        emit SequencerBatchAppended(
            nextQueueIndex - numQueuedTransactions,
            numQueuedTransactions
        );
    }


    /**********************
     * Internal Functions *
     **********************/

    /**
     * Parses the batch context from the extra data.
     * @return _totalElements Total number of elements submitted.
     * @return _nextQueueIndex Index of the next queue element.
     */
    function _getLatestBatchContext()
        internal
        view
        returns (
            uint40 _totalElements,
            uint32 _nextQueueIndex
        )
    {
        bytes28 extraData = batches.getExtraData();

        uint40 totalElements;
        uint32 nextQueueIndex;
        assembly {
            totalElements := and(shr(32, extraData), 0x000000000000000000000000000000000000000000000000000000ffffffffff)
            nextQueueIndex := shr(40, and(shr(32, extraData), 0xffffffffffffffffffffffffffffffffffffffffffffffffffffff0000000000))
        }

        return (totalElements, nextQueueIndex);
    }

    /**
     * Encodes the batch context for the extra data.
     * @param _totalElements Total number of elements submitted.
     * @param _nextQueueIndex Index of the next queue element.
     * @return _context Encoded batch context.
     */
    function _makeLatestBatchContext(
        uint40 _totalElements,
        uint32 _nextQueueIndex
    )
        internal
        view
        returns (
            bytes28 _context
        )
    {
        bytes28 totalElementsAndNextQueueIndex;
        assembly {
            totalElementsAndNextQueueIndex := shl(32, or(_totalElements, shl(40, _nextQueueIndex)))
        }

        return totalElementsAndNextQueueIndex;
    }

    /**
     * Retrieves the hash of a queue element.
     * @param _index Index of the queue element to retrieve a hash for.
     * @return _queueLeafHash Hash of the queue element.
     */
    function _getQueueLeafHash(
        uint _index
    )
        internal
        view
        returns (
            bytes32 _queueLeafHash
        )
    {
        Lib_OVMCodec.QueueElement memory element = getQueueElement(_index);

        require(
            msg.sender == sequencer
            || element.timestamp + forceInclusionPeriodSeconds <= block.timestamp,
            "Queue transactions cannot be submitted during the sequencer inclusion period."
        );

        return _hashTransactionChainElement(
            TransactionChainElement({
                isSequenced: false,
                queueIndex: _index,
                timestamp: 0,
                blockNumber: 0,
                txData: hex""
            })
        );
    }

    /**
     * Inserts a batch into the chain of batches.
     * @param _transactionRoot Root of the transaction tree for this batch.
     * @param _batchSize Number of elements in the batch.
     * @param _numQueuedTransactions Number of queue transactions in the batch.
     */
    function _appendBatch(
        bytes32 _transactionRoot,
        uint _batchSize,
        uint _numQueuedTransactions
    )
        internal
    {
        (uint40 totalElements, uint32 nextQueueIndex) = _getLatestBatchContext();

        Lib_OVMCodec.ChainBatchHeader memory header = Lib_OVMCodec.ChainBatchHeader({
            batchIndex: batches.getLength(),
            batchRoot: _transactionRoot,
            batchSize: _batchSize,
            prevTotalElements: totalElements,
            extraData: hex""
        });

        bytes32 batchHeaderHash = _hashBatchHeader(header);
        bytes28 latestBatchContext = _makeLatestBatchContext(
            totalElements + uint40(header.batchSize),
            nextQueueIndex + uint32(_numQueuedTransactions)
        );

        batches.push(batchHeaderHash, latestBatchContext);
    }

    /**
     * Checks that a given batch context is valid.
     * @param _context Batch context to validate.
     * @param _nextQueueIndex Index of the next queue element to process.
     */
    function _validateBatchContext(
        BatchContext memory _context,
        uint32 _nextQueueIndex
    )
        internal
    {
        if (queue.getLength() == 0) {
            return;
        }

        Lib_OVMCodec.QueueElement memory nextQueueElement = getQueueElement(_nextQueueIndex);

        require(
            block.timestamp < nextQueueElement.timestamp + forceInclusionPeriodSeconds,
            "Older queue batches must be processed before a new sequencer batch."
        );

        require(
            _context.timestamp <= nextQueueElement.timestamp,
            "Sequencer transactions timestamp too high."
        );

        require(
            _context.blockNumber <= nextQueueElement.blockNumber,
            "Sequencer transactions blockNumber too high."
        );
    }

    /**
     * Hashes a transaction chain element.
     * @param _element Chain element to hash.
     * @return _hash Hash of the chain element.
     */
    function _hashTransactionChainElement(
        TransactionChainElement memory _element
    )
        internal
        pure
        returns (
            bytes32 _hash
        )
    {
        return keccak256(
            abi.encode(
                _element.isSequenced,
                _element.queueIndex,
                _element.timestamp,
                _element.blockNumber,
                _element.txData
            )
        );
    }
}

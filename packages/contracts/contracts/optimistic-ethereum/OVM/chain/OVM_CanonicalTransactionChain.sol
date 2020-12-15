// SPDX-License-Identifier: MIT
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";
import { Lib_AddressResolver } from "../../libraries/resolver/Lib_AddressResolver.sol";
import { Lib_MerkleUtils } from "../../libraries/utils/Lib_MerkleUtils.sol";
import { Lib_RingBuffer, iRingBufferOverwriter } from "../../libraries/utils/Lib_RingBuffer.sol";

/* Interface Imports */
import { iOVM_CanonicalTransactionChain } from "../../iOVM/chain/iOVM_CanonicalTransactionChain.sol";

/* Contract Imports */
import { OVM_ExecutionManager } from "../execution/OVM_ExecutionManager.sol";


library Math {
    function min(uint x, uint y) internal pure returns (uint z) {
        if (x < y) {
            return x;
        }
        return y;
    }
}


/**
 * @title OVM_CanonicalTransactionChain
 */
contract OVM_CanonicalTransactionChain is iOVM_CanonicalTransactionChain, Lib_AddressResolver {
    using Lib_RingBuffer for Lib_RingBuffer.RingBuffer;


    /*************
     * Constants *
     *************/

    uint256 constant public MIN_ROLLUP_TX_GAS = 20000;
    uint256 constant public MAX_ROLLUP_TX_SIZE = 10000;
    uint256 constant public L2_GAS_DISCOUNT_DIVISOR = 10;

    // Encoding constants (all in bytes)
    uint256 constant internal BATCH_CONTEXT_SIZE = 16;
    uint256 constant internal BATCH_CONTEXT_LENGTH_POS = 12;
    uint256 constant internal BATCH_CONTEXT_START_POS = 15;
    uint256 constant internal TX_DATA_HEADER_SIZE = 3;
    uint256 constant internal BYTES_TILL_TX_DATA = 65;


    /*************
     * Variables *
     *************/

    uint256 internal forceInclusionPeriodSeconds;
    uint256 internal lastOVMTimestamp;
    Lib_RingBuffer.RingBuffer internal batches;
    Lib_RingBuffer.RingBuffer internal queue;


    /***************
     * Constructor *
     ***************/

    constructor(
        address _libAddressManager,
        uint256 _forceInclusionPeriodSeconds
    )
        Lib_AddressResolver(_libAddressManager)
    {
        forceInclusionPeriodSeconds = _forceInclusionPeriodSeconds;
    }


    /********************
     * Public Functions *
     ********************/

    /**
     * @inheritdoc iOVM_CanonicalTransactionChain
     */
    function init()
        override
        public
    {
        batches.init(
            16,
            Lib_OVMCodec.RING_BUFFER_CTC_BATCHES,
            iRingBufferOverwriter(resolve("OVM_StateCommitmentChain"))
        );

        queue.init(
            16,
            Lib_OVMCodec.RING_BUFFER_CTC_QUEUE,
            iRingBufferOverwriter(resolve("OVM_StateCommitmentChain"))
        );
    }

    /**
     * @inheritdoc iOVM_CanonicalTransactionChain
     */
    function getTotalElements()
        override
        public
        view
        returns (
            uint256 _totalElements
        )
    {
        (uint40 totalElements,) = _getBatchExtraData();
        return uint256(totalElements);
    }

    /**
     * @inheritdoc iOVM_CanonicalTransactionChain
     */
    function getTotalBatches()
        override
        public
        view
        returns (
            uint256 _totalBatches
        )
    {
        return uint256(batches.getLength());
    }

    /**
     * @inheritdoc iOVM_CanonicalTransactionChain
     */
    function getNextQueueIndex()
        override
        public
        view
        returns (
            uint40
        )
    {
        (, uint40 nextQueueIndex) = _getBatchExtraData();
        return nextQueueIndex;
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
        uint40 trueIndex = uint40(_index * 2);
        bytes32 queueRoot = queue.get(trueIndex);
        bytes32 timestampAndBlockNumber = queue.get(trueIndex + 1);

        uint40 elementTimestamp;
        uint40 elementBlockNumber;
        assembly {
            elementTimestamp   :=         and(timestampAndBlockNumber, 0x000000000000000000000000000000000000000000000000000000FFFFFFFFFF)
            elementBlockNumber := shr(40, and(timestampAndBlockNumber, 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF0000000000))
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
    function getNumPendingQueueElements()
        override
        public
        view
        returns (
            uint40
        )
    {
        return  _getQueueLength() - getNextQueueIndex();
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
            "Transaction exceeds maximum rollup transaction data size."
        );

        require(
            _gasLimit >= MIN_ROLLUP_TX_GAS,
            "Transaction gas limit too low to enqueue."
        );

        // We need to consume some amount of L1 gas in order to rate limit transactions going into
        // L2. However, L2 is cheaper than L1 so we only need to burn some small proportion of the
        // provided L1 gas.
        uint256 gasToConsume = _gasLimit/L2_GAS_DISCOUNT_DIVISOR;
        uint256 startingGas = gasleft();

        // Although this check is not necessary (burn below will run out of gas if not true), it
        // gives the user an explicit reason as to why the enqueue attempt failed.
        require(
            startingGas > gasToConsume,
            "Insufficient gas for L2 rate limiting burn."
        );

        // Here we do some "dumb" work in order to burn gas, although we should probably replace
        // this with something like minting gas token later on.
        uint256 i;
        while(startingGas - gasleft() < gasToConsume) {
            i++;
        }

        bytes32 transactionHash = keccak256(
            abi.encode(
                msg.sender,
                _target,
                _gasLimit,
                _data
            )
        );

        bytes32 timestampAndBlockNumber;
        assembly {
            timestampAndBlockNumber := timestamp()
            timestampAndBlockNumber := or(timestampAndBlockNumber, shl(40, number()))
        }

        queue.push2(
            transactionHash,
            timestampAndBlockNumber
        );

        uint40 queueIndex = queue.getLength() / 2;
        emit TransactionEnqueued(
            msg.sender,
            _target,
            _gasLimit,
            _data,
            queueIndex - 1,
            block.timestamp
        );
    }

    /**
     * @inheritdoc iOVM_CanonicalTransactionChain
     */
    function appendQueueBatch(
        uint256 _numQueuedTransactions
    )
        override
        public
    {
        _numQueuedTransactions = Math.min(_numQueuedTransactions, getNumPendingQueueElements());
        require(
            _numQueuedTransactions > 0,
            "Must append more than zero transactions."
        );

        bytes32[] memory leaves = new bytes32[](_numQueuedTransactions);
        uint40 nextQueueIndex = getNextQueueIndex();

        for (uint256 i = 0; i < _numQueuedTransactions; i++) {
            if (msg.sender != resolve("OVM_Sequencer")) {
                Lib_OVMCodec.QueueElement memory el = getQueueElement(nextQueueIndex);
                require(
                    el.timestamp + forceInclusionPeriodSeconds < block.timestamp,
                    "Queue transactions cannot be submitted during the sequencer inclusion period."
                );
            }
            leaves[i] = _getQueueLeafHash(nextQueueIndex);
            nextQueueIndex++;
        }

        _appendBatch(
            Lib_MerkleUtils.getMerkleRoot(leaves),
            _numQueuedTransactions,
            _numQueuedTransactions
        );

        emit QueueBatchAppended(
            nextQueueIndex - _numQueuedTransactions,
            _numQueuedTransactions,
            getTotalElements()
        );
    }

    /**
     * @inheritdoc iOVM_CanonicalTransactionChain
     */
    function appendSequencerBatch()
        override
        public
    {
        uint40 shouldStartAtBatch;
        uint24 totalElementsToAppend;
        uint24 numContexts;
        assembly {
            shouldStartAtBatch    := shr(216, calldataload(4))
            totalElementsToAppend := shr(232, calldataload(9))
            numContexts           := shr(232, calldataload(12))
        }

        require(
            shouldStartAtBatch == getTotalElements(),
            "Actual batch start index does not match expected start index."
        );

        require(
            msg.sender == resolve("OVM_Sequencer"),
            "Function can only be called by the Sequencer."
        );

        require(
            numContexts > 0,
            "Must provide at least one batch context."
        );

        require(
            totalElementsToAppend > 0,
            "Must append at least one element."
        );

        uint40 nextTransactionPtr = uint40(BATCH_CONTEXT_START_POS + BATCH_CONTEXT_SIZE * numContexts);
        uint256 calldataSize;
        assembly {
            calldataSize := calldatasize()
        }

        require(
            calldataSize >= nextTransactionPtr,
            "Not enough BatchContexts provided."
        );

        bytes32[] memory leaves = new bytes32[](totalElementsToAppend);
        uint32 transactionIndex = 0;
        uint32 numSequencerTransactionsProcessed = 0;
        uint40 nextQueueIndex = getNextQueueIndex();
        uint40 queueLength = _getQueueLength();

        for (uint32 i = 0; i < numContexts; i++) {
            BatchContext memory context = _getBatchContext(i);

            for (uint32 j = 0; j < context.numSequencedTransactions; j++) {
                uint256 txDataLength;
                assembly {
                    txDataLength := shr(232, calldataload(nextTransactionPtr))
                }

                leaves[transactionIndex] = _getSequencerLeafHash(context, nextTransactionPtr, txDataLength);
                nextTransactionPtr += uint40(TX_DATA_HEADER_SIZE + txDataLength);
                numSequencerTransactionsProcessed++;
                transactionIndex++;
            }

            for (uint32 j = 0; j < context.numSubsequentQueueTransactions; j++) {
                require(nextQueueIndex < queueLength, "Not enough queued transactions to append.");
                leaves[transactionIndex] = _getQueueLeafHash(nextQueueIndex);
                nextQueueIndex++;
                transactionIndex++;
            }
        }

        require(
            calldataSize == nextTransactionPtr,
            "Not all sequencer transactions were processed."
        );

        require(
            transactionIndex == totalElementsToAppend,
            "Actual transaction index does not match expected total elements to append."
        );

        uint40 numQueuedTransactions = totalElementsToAppend - numSequencerTransactionsProcessed;
        _appendBatch(
            Lib_MerkleUtils.getMerkleRoot(leaves),
            totalElementsToAppend,
            numQueuedTransactions
        );

        emit SequencerBatchAppended(
            nextQueueIndex - numQueuedTransactions,
            numQueuedTransactions,
            getTotalElements()
        );
    }

    /**
     * @inheritdoc iOVM_CanonicalTransactionChain
     */
    function verifyTransaction(
        Lib_OVMCodec.Transaction memory _transaction,
        Lib_OVMCodec.TransactionChainElement memory _txChainElement,
        Lib_OVMCodec.ChainBatchHeader memory _batchHeader,
        Lib_OVMCodec.ChainInclusionProof memory _inclusionProof
    )
        override
        public
        view
        returns (
            bool
        )
    {
        if (_txChainElement.isSequenced == true) {
            return _verifySequencerTransaction(
                _transaction,
                _txChainElement,
                _batchHeader,
                _inclusionProof
            );
        } else {
            return _verifyQueueTransaction(
                _transaction,
                _txChainElement.queueIndex,
                _batchHeader,
                _inclusionProof
            );
        }
    }


    /**********************
     * Internal Functions *
     **********************/

    /**
     * Returns the BatchContext located at a particular index.
     * @param _index The index of the BatchContext
     * @return The BatchContext at the specified index.
     */
    function _getBatchContext(
        uint256 _index
    )
        internal
        pure
        returns (
            BatchContext memory
        )
    {
        uint256 contextPtr = 15 + _index * BATCH_CONTEXT_SIZE;
        uint256 numSequencedTransactions;
        uint256 numSubsequentQueueTransactions;
        uint256 ctxTimestamp;
        uint256 ctxBlockNumber;

        assembly {
            numSequencedTransactions       := shr(232, calldataload(contextPtr))
            numSubsequentQueueTransactions := shr(232, calldataload(add(contextPtr, 3)))
            ctxTimestamp                   := shr(216, calldataload(add(contextPtr, 6)))
            ctxBlockNumber                 := shr(216, calldataload(add(contextPtr, 11)))
        }

        return BatchContext({
            numSequencedTransactions: numSequencedTransactions,
            numSubsequentQueueTransactions: numSubsequentQueueTransactions,
            timestamp: ctxTimestamp,
            blockNumber: ctxBlockNumber
        });
    }

    /**
     * Parses the batch context from the extra data.
     * @return Total number of elements submitted.
     * @return Index of the next queue element.
     */
    function _getBatchExtraData()
        internal
        view
        returns (
            uint40,
            uint40
        )
    {
        bytes27 extraData = batches.getExtraData();

        uint40 totalElements;
        uint40 nextQueueIndex;
        assembly {
            extraData      := shr(40, extraData)
            totalElements  :=         and(extraData, 0x000000000000000000000000000000000000000000000000000000FFFFFFFFFF)
            nextQueueIndex := shr(40, and(extraData, 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF0000000000))
        }

        return (
            totalElements,
            nextQueueIndex
        );
    }

    /**
     * Encodes the batch context for the extra data.
     * @param _totalElements Total number of elements submitted.
     * @param _nextQueueIndex Index of the next queue element.
     * @return Encoded batch context.
     */
    function _makeBatchExtraData(
        uint40 _totalElements,
        uint40 _nextQueueIndex
    )
        internal
        pure
        returns (
            bytes27
        )
    {
        bytes27 extraData;
        assembly {
            extraData := _totalElements
            extraData := or(extraData, shl(40, _nextQueueIndex))
            extraData := shl(40, extraData)
        }

        return extraData;
    }

    /**
     * Retrieves the hash of a queue element.
     * @param _index Index of the queue element to retrieve a hash for.
     * @return Hash of the queue element.
     */
    function _getQueueLeafHash(
        uint256 _index
    )
        internal
        view
        returns (
            bytes32
        )
    {
        return _hashTransactionChainElement(
            Lib_OVMCodec.TransactionChainElement({
                isSequenced: false,
                queueIndex: _index,
                timestamp: 0,
                blockNumber: 0,
                txData: hex""
            })
        );
    }

    /**
     * Retrieves the length of the queue.
     * @return Length of the queue.
     */
    function _getQueueLength()
        internal
        view
        returns (
            uint40
        )
    {
        // The underlying queue data structure stores 2 elements
        // per insertion, so to get the real queue length we need
        // to divide by 2. See the usage of `push2(..)`.
        return queue.getLength() / 2;
    }

    /**
     * Retrieves the hash of a sequencer element.
     * @param _context Batch context for the given element.
     * @param _nextTransactionPtr Pointer to the next transaction in the calldata.
     * @param _txDataLength Length of the transaction item.
     * @return Hash of the sequencer element.
     */
    function _getSequencerLeafHash(
        BatchContext memory _context,
        uint256 _nextTransactionPtr,
        uint256 _txDataLength
    )
        internal
        pure
        returns (
            bytes32
        )
    {

        bytes memory chainElement = new bytes(BYTES_TILL_TX_DATA + _txDataLength);
        uint256 ctxTimestamp = _context.timestamp;
        uint256 ctxBlockNumber = _context.blockNumber;

        bytes32 leafHash;
        assembly {
            let chainElementStart := add(chainElement, 0x20)

            // Set the first byte equal to `1` to indicate this is a sequencer chain element.
            // This distinguishes sequencer ChainElements from queue ChainElements because
            // all queue ChainElements are ABI encoded and the first byte of ABI encoded
            // elements is always zero
            mstore8(chainElementStart, 1)

            mstore(add(chainElementStart, 1), ctxTimestamp)
            mstore(add(chainElementStart, 33), ctxBlockNumber)

            calldatacopy(add(chainElementStart, BYTES_TILL_TX_DATA), add(_nextTransactionPtr, 3), _txDataLength)

            leafHash := keccak256(chainElementStart, add(BYTES_TILL_TX_DATA, _txDataLength))
        }

        return leafHash;
    }

    /**
     * Retrieves the hash of a sequencer element.
     * @param _txChainElement The chain element which is hashed to calculate the leaf.
     * @return Hash of the sequencer element.
     */
    function _getSequencerLeafHash(
        Lib_OVMCodec.TransactionChainElement memory _txChainElement
    )
        internal
        view
        returns(
            bytes32
        )
    {
        bytes memory txData = _txChainElement.txData;
        uint256 txDataLength = _txChainElement.txData.length;

        bytes memory chainElement = new bytes(BYTES_TILL_TX_DATA + txDataLength);
        uint256 ctxTimestamp = _txChainElement.timestamp;
        uint256 ctxBlockNumber = _txChainElement.blockNumber;

        bytes32 leafHash;
        assembly {
            let chainElementStart := add(chainElement, 0x20)

            // Set the first byte equal to `1` to indicate this is a sequencer chain element.
            // This distinguishes sequencer ChainElements from queue ChainElements because
            // all queue ChainElements are ABI encoded and the first byte of ABI encoded
            // elements is always zero
            mstore8(chainElementStart, 1)

            mstore(add(chainElementStart, 1), ctxTimestamp)
            mstore(add(chainElementStart, 33), ctxBlockNumber)

            pop(staticcall(gas(), 0x04, add(txData, 0x20), txDataLength, add(chainElementStart, BYTES_TILL_TX_DATA), txDataLength))

            leafHash := keccak256(chainElementStart, add(BYTES_TILL_TX_DATA, txDataLength))
        }

        return leafHash;
    }

    /**
     * Inserts a batch into the chain of batches.
     * @param _transactionRoot Root of the transaction tree for this batch.
     * @param _batchSize Number of elements in the batch.
     * @param _numQueuedTransactions Number of queue transactions in the batch.
     */
    function _appendBatch(
        bytes32 _transactionRoot,
        uint256 _batchSize,
        uint256 _numQueuedTransactions
    )
        internal
    {
        (uint40 totalElements, uint40 nextQueueIndex) = _getBatchExtraData();

        Lib_OVMCodec.ChainBatchHeader memory header = Lib_OVMCodec.ChainBatchHeader({
            batchIndex: batches.getLength(),
            batchRoot: _transactionRoot,
            batchSize: _batchSize,
            prevTotalElements: totalElements,
            extraData: hex""
        });

        emit TransactionBatchAppended(
            header.batchIndex,
            header.batchRoot,
            header.batchSize,
            header.prevTotalElements,
            header.extraData
        );

        bytes32 batchHeaderHash = Lib_OVMCodec.hashBatchHeader(header);
        bytes27 latestBatchContext = _makeBatchExtraData(
            totalElements + uint40(header.batchSize),
            nextQueueIndex + uint40(_numQueuedTransactions)
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
        uint40 _nextQueueIndex
    )
        internal
        view
    {
        if (getNumPendingQueueElements() == 0) {
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
     * @return Hash of the chain element.
     */
    function _hashTransactionChainElement(
        Lib_OVMCodec.TransactionChainElement memory _element
    )
        internal
        pure
        returns (
            bytes32
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

    /**
     * Verifies a sequencer transaction, returning true if it was indeed included in the CTC
     * @param _transaction The transaction we are verifying inclusion of.
     * @param _txChainElement The chain element that the transaction is claimed to be a part of.
     * @param _batchHeader Header of the batch the transaction was included in.
     * @param _inclusionProof An inclusion proof into the CTC at a particular index.
     * @return True if the transaction was included in the specified location, else false.
     */
    function _verifySequencerTransaction(
        Lib_OVMCodec.Transaction memory _transaction,
        Lib_OVMCodec.TransactionChainElement memory _txChainElement,
        Lib_OVMCodec.ChainBatchHeader memory _batchHeader,
        Lib_OVMCodec.ChainInclusionProof memory _inclusionProof
    )
        internal
        view
        returns (
            bool
        )
    {
        OVM_ExecutionManager ovmExecutionManager = OVM_ExecutionManager(resolve("OVM_ExecutionManager"));
        uint256 gasLimit = ovmExecutionManager.getMaxTransactionGasLimit();
        bytes32 leafHash = _getSequencerLeafHash(_txChainElement);

        require(
            _verifyElement(
                leafHash,
                _batchHeader,
                _inclusionProof
            ),
            "Invalid Sequencer transaction inclusion proof."
        );

        require(
            _transaction.blockNumber        == _txChainElement.blockNumber
            && _transaction.timestamp       == _txChainElement.timestamp
            && _transaction.entrypoint      == resolve("OVM_DecompressionPrecompileAddress")
            && _transaction.gasLimit        == gasLimit
            && _transaction.l1TxOrigin      == address(0)
            && _transaction.l1QueueOrigin   == Lib_OVMCodec.QueueOrigin.SEQUENCER_QUEUE
            && keccak256(_transaction.data) == keccak256(_txChainElement.txData),
            "Invalid Sequencer transaction."
        );

        return true;
    }

    /**
     * Verifies a queue transaction, returning true if it was indeed included in the CTC
     * @param _transaction The transaction we are verifying inclusion of.
     * @param _queueIndex The queueIndex of the queued transaction.
     * @param _batchHeader Header of the batch the transaction was included in.
     * @param _inclusionProof An inclusion proof into the CTC at a particular index (should point to queue tx).
     * @return True if the transaction was included in the specified location, else false.
     */
    function _verifyQueueTransaction(
        Lib_OVMCodec.Transaction memory _transaction,
        uint256 _queueIndex,
        Lib_OVMCodec.ChainBatchHeader memory _batchHeader,
        Lib_OVMCodec.ChainInclusionProof memory _inclusionProof
    )
        internal
        view
        returns (
            bool
        )
    {
        bytes32 leafHash = _getQueueLeafHash(_queueIndex);

        require(
            _verifyElement(
                leafHash,
                _batchHeader,
                _inclusionProof
            ),
            "Invalid Queue transaction inclusion proof."
        );

        bytes32 transactionHash = keccak256(
            abi.encode(
                _transaction.l1TxOrigin,
                _transaction.entrypoint,
                _transaction.gasLimit,
                _transaction.data
            )
        );

        Lib_OVMCodec.QueueElement memory el = getQueueElement(_queueIndex);
        require(
            el.queueRoot      == transactionHash
            && el.timestamp   == _transaction.timestamp
            && el.blockNumber == _transaction.blockNumber,
            "Invalid Queue transaction."
        );

        return true;
    }

    /**
     * Verifies a batch inclusion proof.
     * @param _element Hash of the element to verify a proof for.
     * @param _batchHeader Header of the batch in which the element was included.
     * @param _proof Merkle inclusion proof for the element.
     */
    function _verifyElement(
        bytes32 _element,
        Lib_OVMCodec.ChainBatchHeader memory _batchHeader,
        Lib_OVMCodec.ChainInclusionProof memory _proof
    )
        internal
        view
        returns (
            bool
        )
    {
        require(
            Lib_OVMCodec.hashBatchHeader(_batchHeader) == batches.get(uint32(_batchHeader.batchIndex)),
            "Invalid batch header."
        );

        require(
            Lib_MerkleUtils.verify(
                _batchHeader.batchRoot,
                _element,
                _proof.index,
                _proof.siblings
            ),
            "Invalid inclusion proof."
        );

        return true;
    }
}

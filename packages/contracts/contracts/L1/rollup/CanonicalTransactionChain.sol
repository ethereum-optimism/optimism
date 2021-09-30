// SPDX-License-Identifier: MIT
pragma solidity ^0.8.8;

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";
import { Lib_AddressResolver } from "../../libraries/resolver/Lib_AddressResolver.sol";

/* Interface Imports */
import { ICanonicalTransactionChain } from "./ICanonicalTransactionChain.sol";
import { IChainStorageContainer } from "./IChainStorageContainer.sol";

/**
 * @title CanonicalTransactionChain
 * @dev The Canonical Transaction Chain (CTC) contract is an append-only log of transactions
 * which must be applied to the rollup state. It defines the ordering of rollup transactions by
 * writing them to the 'CTC:batches' instance of the Chain Storage Container.
 * The CTC also allows any account to 'enqueue' an L2 transaction, which will require that the
 * Sequencer will eventually append it to the rollup state.
 *
 * Runtime target: EVM
 */
contract CanonicalTransactionChain is ICanonicalTransactionChain, Lib_AddressResolver {

    /*************
     * Constants *
     *************/

    // L2 tx gas-related
    uint256 constant public MIN_ROLLUP_TX_GAS = 100000;
    uint256 constant public MAX_ROLLUP_TX_SIZE = 50000;
    uint256 immutable public L2_GAS_DISCOUNT_DIVISOR;
    uint256 immutable public ENQUEUE_GAS_COST;
    uint256 immutable public ENQUEUE_L2_GAS_PREPAID;

    // Encoding-related (all in bytes)
    uint256 constant internal BATCH_CONTEXT_SIZE = 16;
    uint256 constant internal BATCH_CONTEXT_LENGTH_POS = 12;
    uint256 constant internal BATCH_CONTEXT_START_POS = 15;
    uint256 constant internal TX_DATA_HEADER_SIZE = 3;
    uint256 constant internal BYTES_TILL_TX_DATA = 65;


    /*************
     * Variables *
     *************/

    uint256 public maxTransactionGasLimit;


    /***************
     * Constructor *
     ***************/

    constructor(
        address _libAddressManager,
        uint256 _maxTransactionGasLimit,
        uint256 _l2GasDiscountDivisor,
        uint256 _enqueueGasCost
    )
        Lib_AddressResolver(_libAddressManager)
    {
        maxTransactionGasLimit = _maxTransactionGasLimit;
        L2_GAS_DISCOUNT_DIVISOR = _l2GasDiscountDivisor;
        ENQUEUE_GAS_COST  = _enqueueGasCost;
        ENQUEUE_L2_GAS_PREPAID = _l2GasDiscountDivisor * _enqueueGasCost;
    }


    /********************
     * Public Functions *
     ********************/

    /**
     * Accesses the batch storage container.
     * @return Reference to the batch storage container.
     */
    function batches()
        public
        view
        returns (
            IChainStorageContainer
        )
    {
        return IChainStorageContainer(
            resolve("ChainStorageContainer-CTC-batches")
        );
    }

    /**
     * Accesses the queue storage container.
     * @return Reference to the queue storage container.
     */
    function queue()
        public
        view
        returns (
            IChainStorageContainer
        )
    {
        return IChainStorageContainer(
            resolve("ChainStorageContainer-CTC-queue")
        );
    }

    /**
     * Retrieves the total number of elements submitted.
     * @return _totalElements Total submitted elements.
     */
    function getTotalElements()
        public
        view
        returns (
            uint256 _totalElements
        )
    {
        (uint40 totalElements,,,) = _getBatchExtraData();
        return uint256(totalElements);
    }

    /**
     * Retrieves the total number of batches submitted.
     * @return _totalBatches Total submitted batches.
     */
    function getTotalBatches()
        public
        view
        returns (
            uint256 _totalBatches
        )
    {
        return batches().length();
    }

    /**
     * Returns the index of the next element to be enqueued.
     * @return Index for the next queue element.
     */
    function getNextQueueIndex()
        public
        view
        returns (
            uint40
        )
    {
        (,uint40 nextQueueIndex,,) = _getBatchExtraData();
        return nextQueueIndex;
    }

    /**
     * Returns the timestamp of the last transaction.
     * @return Timestamp for the last transaction.
     */
    function getLastTimestamp()
        public
        view
        returns (
            uint40
        )
    {
        (,,uint40 lastTimestamp,) = _getBatchExtraData();
        return lastTimestamp;
    }

    /**
     * Returns the blocknumber of the last transaction.
     * @return Blocknumber for the last transaction.
     */
    function getLastBlockNumber()
        public
        view
        returns (
            uint40
        )
    {
        (,,,uint40 lastBlockNumber) = _getBatchExtraData();
        return lastBlockNumber;
    }

    /**
     * Gets the queue element at a particular index.
     * @param _index Index of the queue element to access.
     * @return _element Queue element at the given index.
     */
    function getQueueElement(
        uint256 _index
    )
        public
        view
        returns (
            Lib_OVMCodec.QueueElement memory _element
        )
    {
        return _getQueueElement(
            _index,
            queue()
        );
    }

    /**
     * Get the number of queue elements which have not yet been included.
     * @return Number of pending queue elements.
     */
    function getNumPendingQueueElements()
        public
        view
        returns (
            uint40
        )
    {
        return getQueueLength() - getNextQueueIndex();
    }

   /**
     * Retrieves the length of the queue, including
     * both pending and canonical transactions.
     * @return Length of the queue.
     */
    function getQueueLength()
        public
        view
        returns (
            uint40
        )
    {
        return _getQueueLength(
            queue()
        );
    }

    /**
     * Adds a transaction to the queue.
     * @param _target Target L2 contract to send the transaction to.
     * @param _gasLimit Gas limit for the enqueued L2 transaction.
     * @param _data Transaction data.
     */
    function enqueue(
        address _target,
        uint256 _gasLimit,
        bytes memory _data
    )
        external
    {
        require(
            _data.length <= MAX_ROLLUP_TX_SIZE,
            "Transaction data size exceeds maximum for rollup transaction."
        );

        require(
            _gasLimit <= maxTransactionGasLimit,
            "Transaction gas limit exceeds maximum for rollup transaction."
        );

        require(
            _gasLimit >= MIN_ROLLUP_TX_GAS,
            "Transaction gas limit too low to enqueue."
        );

        // Transactions submitted to the queue lack a method for paying gas fees to the Sequencer.
        // So we need to prevent spam attacks by ensuring that the cost of enqueueing a transaction
        // from L1 to L2 is not underpriced. Therefore, we define 'ENQUEUE_L2_GAS_PREPAID' as a
        // threshold. If the _gasLimit for the enqueued transaction is above this threshold, then we
        // 'charge' to user by burning additional L1 gas. Since gas is cheaper on L2 than L1, we
        // only need to burn a fraction of the provided L1 gas, which is determined by the
        // L2_GAS_DISCOUNT_DIVISOR.
        if(_gasLimit > ENQUEUE_L2_GAS_PREPAID) {
            uint256 gasToConsume = (_gasLimit - ENQUEUE_L2_GAS_PREPAID) / L2_GAS_DISCOUNT_DIVISOR;
            uint256 startingGas = gasleft();

            // Although this check is not necessary (burn below will run out of gas if not true), it
            // gives the user an explicit reason as to why the enqueue attempt failed.
            require(
                startingGas > gasToConsume,
                "Insufficient gas for L2 rate limiting burn."
            );

            uint256 i;
            while(startingGas - gasleft() < gasToConsume) {
                i++;
            }
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

        IChainStorageContainer queueRef = queue();

        queueRef.push(transactionHash);
        queueRef.push(timestampAndBlockNumber);

        // The underlying queue data structure stores 2 elements
        // per insertion, so to get the real queue length we need
        // to divide by 2 and subtract 1.
        uint256 queueIndex = queueRef.length() / 2 - 1;
        emit TransactionEnqueued(
            msg.sender,
            _target,
            _gasLimit,
            _data,
            queueIndex,
            block.timestamp
        );
    }

    /**
     * Allows the sequencer to append a batch of transactions.
     * @dev This function uses a custom encoding scheme for efficiency reasons.
     * .param _shouldStartAtElement Specific batch we expect to start appending to.
     * .param _totalElementsToAppend Total number of batch elements we expect to append.
     * .param _contexts Array of batch contexts.
     * .param _transactionDataFields Array of raw transaction data.
     */
    function appendSequencerBatch()
        external
    {
        uint40 shouldStartAtElement;
        uint24 totalElementsToAppend;
        uint24 numContexts;
        assembly {
            shouldStartAtElement  := shr(216, calldataload(4))
            totalElementsToAppend := shr(232, calldataload(9))
            numContexts           := shr(232, calldataload(12))
        }

        require(
            shouldStartAtElement == getTotalElements(),
            "Actual batch start index does not match expected start index."
        );

        require(
            msg.sender == resolve("OVM_Sequencer"),
            "Function can only be called by the Sequencer."
        );

        uint40 nextTransactionPtr = uint40(
            BATCH_CONTEXT_START_POS + BATCH_CONTEXT_SIZE * numContexts
        );

        require(
            msg.data.length >= nextTransactionPtr,
            "Not enough BatchContexts provided."
        );

        // Take a reference to the queue and its length so we don't have to keep resolving it.
        // Length isn't going to change during the course of execution, so it's fine to simply
        // resolve this once at the start. Saves gas.
        IChainStorageContainer queueRef = queue();
        uint40 queueLength = _getQueueLength(queueRef);

        // Counter for number of sequencer transactions appended so far.
        uint32 numSequencerTransactions = 0;

        // We will sequentially append leaves which are pointers to the queue.
        // The initial queue index is what is currently in storage.
        uint40 nextQueueIndex = getNextQueueIndex();

        BatchContext memory curContext;
        for (uint32 i = 0; i < numContexts; i++) {
            BatchContext memory nextContext = _getBatchContext(i);

            // Now we can update our current context.
            curContext = nextContext;

            // Process sequencer transactions first.
            numSequencerTransactions += uint32(curContext.numSequencedTransactions);

            // Now process any subsequent queue transactions.
            nextQueueIndex += uint40(curContext.numSubsequentQueueTransactions);

        }
        require(
            nextQueueIndex <= queueLength,
            "Attempted to append more elements than are available in the queue."
        );

        // Generate the required metadata that we need to append this batch
        uint40 numQueuedTransactions = totalElementsToAppend - numSequencerTransactions;
        uint40 blockTimestamp;
        uint40 blockNumber;
        if (curContext.numSubsequentQueueTransactions == 0) {
            // The last element is a sequencer tx, therefore pull timestamp and block number from
            // the last context.
            blockTimestamp = uint40(curContext.timestamp);
            blockNumber = uint40(curContext.blockNumber);
        } else {
            // The last element is a queue tx, therefore pull timestamp and block number from the
            // queue element.
            // curContext.numSubsequentQueueTransactions > 0 which means that we've processed at
            // least one queue element. We increment nextQueueIndex after processing each queue
            // element, so the index of the last element we processed is nextQueueIndex - 1.
            Lib_OVMCodec.QueueElement memory lastElement = _getQueueElement(
                nextQueueIndex - 1,
                queueRef
            );

            blockTimestamp = lastElement.timestamp;
            blockNumber = lastElement.blockNumber;
        }

        // Cache the previous blockhash to ensure all transaction data can be retrieved efficiently.
        _appendBatch(
            blockhash(block.number-1),
            totalElementsToAppend,
            numQueuedTransactions,
            blockTimestamp,
            blockNumber
        );

        emit SequencerBatchAppended(
            nextQueueIndex - numQueuedTransactions,
            numQueuedTransactions,
            getTotalElements()
        );
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
            uint40,
            uint40,
            uint40
        )
    {
        bytes27 extraData = batches().getGlobalMetadata();

        uint40 totalElements;
        uint40 nextQueueIndex;
        uint40 lastTimestamp;
        uint40 lastBlockNumber;

        // solhint-disable max-line-length
        assembly {
            extraData       :=  shr(40, extraData)
            totalElements   :=  and(extraData, 0x000000000000000000000000000000000000000000000000000000FFFFFFFFFF)
            nextQueueIndex  :=  shr(40, and(extraData, 0x00000000000000000000000000000000000000000000FFFFFFFFFF0000000000))
            lastTimestamp   :=  shr(80, and(extraData, 0x0000000000000000000000000000000000FFFFFFFFFF00000000000000000000))
            lastBlockNumber :=  shr(120, and(extraData, 0x000000000000000000000000FFFFFFFFFF000000000000000000000000000000))
        }
        // solhint-enable max-line-length

        return (
            totalElements,
            nextQueueIndex,
            lastTimestamp,
            lastBlockNumber
        );
    }

    /**
     * Encodes the batch context for the extra data.
     * @param _totalElements Total number of elements submitted.
     * @param _nextQueueIndex Index of the next queue element.
     * @param _timestamp Timestamp for the last batch.
     * @param _blockNumber Block number of the last batch.
     * @return Encoded batch context.
     */
    function _makeBatchExtraData(
        uint40 _totalElements,
        uint40 _nextQueueIndex,
        uint40 _timestamp,
        uint40 _blockNumber
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
            extraData := or(extraData, shl(80, _timestamp))
            extraData := or(extraData, shl(120, _blockNumber))
            extraData := shl(40, extraData)
        }

        return extraData;
    }

    /**
     * Gets the queue element at a particular index.
     * @param _index Index of the queue element to access.
     * @return _element Queue element at the given index.
     */
    function _getQueueElement(
        uint256 _index,
        IChainStorageContainer _queueRef
    )
        internal
        view
        returns (
            Lib_OVMCodec.QueueElement memory _element
        )
    {
        // The underlying queue data structure stores 2 elements
        // per insertion, so to get the actual desired queue index
        // we need to multiply by 2.
        uint40 trueIndex = uint40(_index * 2);
        bytes32 transactionHash = _queueRef.get(trueIndex);
        bytes32 timestampAndBlockNumber = _queueRef.get(trueIndex + 1);

        uint40 elementTimestamp;
        uint40 elementBlockNumber;
        // solhint-disable max-line-length
        assembly {
            elementTimestamp   :=         and(timestampAndBlockNumber, 0x000000000000000000000000000000000000000000000000000000FFFFFFFFFF)
            elementBlockNumber := shr(40, and(timestampAndBlockNumber, 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF0000000000))
        }
        // solhint-enable max-line-length

        return Lib_OVMCodec.QueueElement({
            transactionHash: transactionHash,
            timestamp: elementTimestamp,
            blockNumber: elementBlockNumber
        });
    }

    /**
     * Retrieves the length of the queue.
     * @return Length of the queue.
     */
    function _getQueueLength(
        IChainStorageContainer _queueRef
    )
        internal
        view
        returns (
            uint40
        )
    {
        // The underlying queue data structure stores 2 elements
        // per insertion, so to get the real queue length we need
        // to divide by 2.
        return uint40(_queueRef.length() / 2);
    }

    /**
     * Inserts a batch into the chain of batches.
     * @param _transactionRoot Root of the transaction tree for this batch.
     * @param _batchSize Number of elements in the batch.
     * @param _numQueuedTransactions Number of queue transactions in the batch.
     * @param _timestamp The latest batch timestamp.
     * @param _blockNumber The latest batch blockNumber.
     */
    function _appendBatch(
        bytes32 _transactionRoot,
        uint256 _batchSize,
        uint256 _numQueuedTransactions,
        uint40 _timestamp,
        uint40 _blockNumber
    )
        internal
    {
        IChainStorageContainer batchesRef = batches();
        (uint40 totalElements, uint40 nextQueueIndex,,) = _getBatchExtraData();

        Lib_OVMCodec.ChainBatchHeader memory header = Lib_OVMCodec.ChainBatchHeader({
            batchIndex: batchesRef.length(),
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
            nextQueueIndex + uint40(_numQueuedTransactions),
            _timestamp,
            _blockNumber
        );

        batchesRef.push(batchHeaderHash, latestBatchContext);
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
}

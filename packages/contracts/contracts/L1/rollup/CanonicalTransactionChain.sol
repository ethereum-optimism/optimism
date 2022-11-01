// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Library Imports */
import { AddressAliasHelper } from "../../standards/AddressAliasHelper.sol";
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
 */
contract CanonicalTransactionChain is ICanonicalTransactionChain, Lib_AddressResolver {
    /*************
     * Constants *
     *************/

    // L2 tx gas-related
    uint256 public constant MIN_ROLLUP_TX_GAS = 100000;
    uint256 public constant MAX_ROLLUP_TX_SIZE = 50000;

    // The approximate cost of calling the enqueue function
    uint256 public enqueueGasCost;
    // The ratio of the cost of L1 gas to the cost of L2 gas
    uint256 public l2GasDiscountDivisor;
    // The amount of L2 gas which can be forwarded to L2 without spam prevention via 'gas burn'.
    // Calculated as the product of l2GasDiscountDivisor * enqueueGasCost.
    // See comments in enqueue() for further detail.
    uint256 public enqueueL2GasPrepaid;

    // Encoding-related (all in bytes)
    uint256 internal constant BATCH_CONTEXT_SIZE = 16;
    // slither-disable-next-line unused-state
    uint256 internal constant BATCH_CONTEXT_LENGTH_POS = 12;
    uint256 internal constant BATCH_CONTEXT_START_POS = 15;
    // slither-disable-next-line unused-state
    uint256 internal constant TX_DATA_HEADER_SIZE = 3;
    // slither-disable-next-line unused-state
    uint256 internal constant BYTES_TILL_TX_DATA = 65;

    /*************
     * Variables *
     *************/

    uint256 public maxTransactionGasLimit;

    /***************
     * Queue State *
     ***************/

    uint40 private _nextQueueIndex; // index of the first queue element not yet included
    Lib_OVMCodec.QueueElement[] queueElements;

    /***************
     * Constructor *
     ***************/

    constructor(
        address _libAddressManager,
        uint256 _maxTransactionGasLimit,
        uint256 _l2GasDiscountDivisor,
        uint256 _enqueueGasCost
    ) Lib_AddressResolver(_libAddressManager) {
        maxTransactionGasLimit = _maxTransactionGasLimit;
        l2GasDiscountDivisor = _l2GasDiscountDivisor;
        enqueueGasCost = _enqueueGasCost;
        enqueueL2GasPrepaid = _l2GasDiscountDivisor * _enqueueGasCost;
    }

    /**********************
     * Function Modifiers *
     **********************/

    /**
     * Modifier to enforce that, if configured, only the Burn Admin may
     * successfully call a method.
     */
    modifier onlyBurnAdmin() {
        require(msg.sender == libAddressManager.owner(), "Only callable by the Burn Admin.");
        _;
    }

    /*******************************
     * Authorized Setter Functions *
     *******************************/

    /**
     * Allows the Burn Admin to update the parameters which determine the amount of gas to burn.
     * The value of enqueueL2GasPrepaid is immediately updated as well.
     * @param _l2GasDiscountDivisor The ratio of the cost of L1 gas to the cost of L2 gas
     * @param _enqueueGasCost The approximate cost of calling the enqueue function
     */
    function setGasParams(uint256 _l2GasDiscountDivisor, uint256 _enqueueGasCost)
        external
        onlyBurnAdmin
    {
        enqueueGasCost = _enqueueGasCost;
        l2GasDiscountDivisor = _l2GasDiscountDivisor;
        // See the comment in enqueue() for the rationale behind this formula.
        enqueueL2GasPrepaid = _l2GasDiscountDivisor * _enqueueGasCost;

        emit L2GasParamsUpdated(l2GasDiscountDivisor, enqueueGasCost, enqueueL2GasPrepaid);
    }

    /********************
     * Public Functions *
     ********************/

    /**
     * Accesses the batch storage container.
     * @return Reference to the batch storage container.
     */
    function batches() public view returns (IChainStorageContainer) {
        return IChainStorageContainer(resolve("ChainStorageContainer-CTC-batches"));
    }

    /**
     * Retrieves the total number of elements submitted.
     * @return _totalElements Total submitted elements.
     */
    function getTotalElements() public view returns (uint256 _totalElements) {
        (uint40 totalElements, , , ) = _getBatchExtraData();
        return uint256(totalElements);
    }

    /**
     * Retrieves the total number of batches submitted.
     * @return _totalBatches Total submitted batches.
     */
    // slither-disable-next-line external-function
    function getTotalBatches() public view returns (uint256 _totalBatches) {
        return batches().length();
    }

    /**
     * Returns the index of the next element to be enqueued.
     * @return Index for the next queue element.
     */
    // slither-disable-next-line external-function
    function getNextQueueIndex() public view returns (uint40) {
        return _nextQueueIndex;
    }

    /**
     * Returns the timestamp of the last transaction.
     * @return Timestamp for the last transaction.
     */
    // slither-disable-next-line external-function
    function getLastTimestamp() public view returns (uint40) {
        (, , uint40 lastTimestamp, ) = _getBatchExtraData();
        return lastTimestamp;
    }

    /**
     * Returns the blocknumber of the last transaction.
     * @return Blocknumber for the last transaction.
     */
    // slither-disable-next-line external-function
    function getLastBlockNumber() public view returns (uint40) {
        (, , , uint40 lastBlockNumber) = _getBatchExtraData();
        return lastBlockNumber;
    }

    /**
     * Gets the queue element at a particular index.
     * @param _index Index of the queue element to access.
     * @return _element Queue element at the given index.
     */
    // slither-disable-next-line external-function
    function getQueueElement(uint256 _index)
        public
        view
        returns (Lib_OVMCodec.QueueElement memory _element)
    {
        return queueElements[_index];
    }

    /**
     * Get the number of queue elements which have not yet been included.
     * @return Number of pending queue elements.
     */
    // slither-disable-next-line external-function
    function getNumPendingQueueElements() public view returns (uint40) {
        return uint40(queueElements.length) - _nextQueueIndex;
    }

    /**
     * Retrieves the length of the queue, including
     * both pending and canonical transactions.
     * @return Length of the queue.
     */
    // slither-disable-next-line external-function
    function getQueueLength() public view returns (uint40) {
        return uint40(queueElements.length);
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
    ) external {
        require(
            _data.length <= MAX_ROLLUP_TX_SIZE,
            "Transaction data size exceeds maximum for rollup transaction."
        );

        require(
            _gasLimit <= maxTransactionGasLimit,
            "Transaction gas limit exceeds maximum for rollup transaction."
        );

        require(_gasLimit >= MIN_ROLLUP_TX_GAS, "Transaction gas limit too low to enqueue.");

        // Transactions submitted to the queue lack a method for paying gas fees to the Sequencer.
        // So we need to prevent spam attacks by ensuring that the cost of enqueueing a transaction
        // from L1 to L2 is not underpriced. For transaction with a high L2 gas limit, we do this by
        // burning some extra gas on L1. Of course there is also some intrinsic cost to enqueueing a
        // transaction, so we want to make sure not to over-charge (by burning too much L1 gas).
        // Therefore, we define 'enqueueL2GasPrepaid' as the L2 gas limit above which we must burn
        // additional gas on L1. This threshold is the product of two inputs:
        // 1. enqueueGasCost: the base cost of calling this function.
        // 2. l2GasDiscountDivisor: the ratio between the cost of gas on L1 and L2. This is a
        //    positive integer, meaning we assume L2 gas is always less costly.
        // The calculation below for gasToConsume can be seen as converting the difference (between
        // the specified L2 gas limit and the prepaid L2 gas limit) to an L1 gas amount.
        if (_gasLimit > enqueueL2GasPrepaid) {
            uint256 gasToConsume = (_gasLimit - enqueueL2GasPrepaid) / l2GasDiscountDivisor;
            uint256 startingGas = gasleft();

            // Although this check is not necessary (burn below will run out of gas if not true), it
            // gives the user an explicit reason as to why the enqueue attempt failed.
            require(startingGas > gasToConsume, "Insufficient gas for L2 rate limiting burn.");

            uint256 i;
            while (startingGas - gasleft() < gasToConsume) {
                i++;
            }
        }

        // Apply an aliasing unless msg.sender == tx.origin. This prevents an attack in which a
        // contract on L1 has the same address as a contract on L2 but doesn't have the same code.
        // We can safely ignore this for EOAs because they're guaranteed to have the same "code"
        // (i.e. no code at all). This also makes it possible for users to interact with contracts
        // on L2 even when the Sequencer is down.
        address sender;
        if (msg.sender == tx.origin) {
            sender = msg.sender;
        } else {
            sender = AddressAliasHelper.applyL1ToL2Alias(msg.sender);
        }

        bytes32 transactionHash = keccak256(abi.encode(sender, _target, _gasLimit, _data));

        queueElements.push(
            Lib_OVMCodec.QueueElement({
                transactionHash: transactionHash,
                timestamp: uint40(block.timestamp),
                blockNumber: uint40(block.number)
            })
        );
        uint256 queueIndex = queueElements.length - 1;
        emit TransactionEnqueued(sender, _target, _gasLimit, _data, queueIndex, block.timestamp);
    }

    /**
     * Allows the sequencer to append a batch of transactions.
     * @dev This function uses a custom encoding scheme for efficiency reasons.
     * .param _shouldStartAtElement Specific batch we expect to start appending to.
     * .param _totalElementsToAppend Total number of batch elements we expect to append.
     * .param _contexts Array of batch contexts.
     * .param _transactionDataFields Array of raw transaction data.
     */
    function appendSequencerBatch() external {
        uint40 shouldStartAtElement;
        uint24 totalElementsToAppend;
        uint24 numContexts;
        assembly {
            shouldStartAtElement := shr(216, calldataload(4))
            totalElementsToAppend := shr(232, calldataload(9))
            numContexts := shr(232, calldataload(12))
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

        require(msg.data.length >= nextTransactionPtr, "Not enough BatchContexts provided.");

        // Counter for number of sequencer transactions appended so far.
        uint32 numSequencerTransactions = 0;

        // Cache the _nextQueueIndex storage variable to a temporary stack variable.
        // This is safe as long as nothing reads or writes to the storage variable
        // until it is updated by the temp variable.
        uint40 nextQueueIndex = _nextQueueIndex;

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
            nextQueueIndex <= queueElements.length,
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
            Lib_OVMCodec.QueueElement memory lastElement = queueElements[nextQueueIndex - 1];

            blockTimestamp = lastElement.timestamp;
            blockNumber = lastElement.blockNumber;
        }

        // Cache the previous blockhash to ensure all transaction data can be retrieved efficiently.
        // slither-disable-next-line reentrancy-no-eth, reentrancy-events
        _appendBatch(
            blockhash(block.number - 1),
            totalElementsToAppend,
            numQueuedTransactions,
            blockTimestamp,
            blockNumber
        );

        // slither-disable-next-line reentrancy-events
        emit SequencerBatchAppended(
            nextQueueIndex - numQueuedTransactions,
            numQueuedTransactions,
            getTotalElements()
        );

        // Update the _nextQueueIndex storage variable.
        // slither-disable-next-line reentrancy-no-eth
        _nextQueueIndex = nextQueueIndex;
    }

    /**********************
     * Internal Functions *
     **********************/

    /**
     * Returns the BatchContext located at a particular index.
     * @param _index The index of the BatchContext
     * @return The BatchContext at the specified index.
     */
    function _getBatchContext(uint256 _index) internal pure returns (BatchContext memory) {
        uint256 contextPtr = 15 + _index * BATCH_CONTEXT_SIZE;
        // slither-disable-next-line similar-names
        uint256 numSequencedTransactions;
        uint256 numSubsequentQueueTransactions;
        uint256 ctxTimestamp;
        uint256 ctxBlockNumber;

        assembly {
            numSequencedTransactions := shr(232, calldataload(contextPtr))
            numSubsequentQueueTransactions := shr(232, calldataload(add(contextPtr, 3)))
            ctxTimestamp := shr(216, calldataload(add(contextPtr, 6)))
            ctxBlockNumber := shr(216, calldataload(add(contextPtr, 11)))
        }

        return
            BatchContext({
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
     * @return Timestamp for the last transaction
     * @return Block number for the last transaction.
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
            extraData := shr(40, extraData)
            totalElements := and(
                extraData,
                0x000000000000000000000000000000000000000000000000000000FFFFFFFFFF
            )
            nextQueueIndex := shr(
                40,
                and(extraData, 0x00000000000000000000000000000000000000000000FFFFFFFFFF0000000000)
            )
            lastTimestamp := shr(
                80,
                and(extraData, 0x0000000000000000000000000000000000FFFFFFFFFF00000000000000000000)
            )
            lastBlockNumber := shr(
                120,
                and(extraData, 0x000000000000000000000000FFFFFFFFFF000000000000000000000000000000)
            )
        }
        // solhint-enable max-line-length

        return (totalElements, nextQueueIndex, lastTimestamp, lastBlockNumber);
    }

    /**
     * Encodes the batch context for the extra data.
     * @param _totalElements Total number of elements submitted.
     * @param _nextQueueIdx Index of the next queue element.
     * @param _timestamp Timestamp for the last batch.
     * @param _blockNumber Block number of the last batch.
     * @return Encoded batch context.
     */
    function _makeBatchExtraData(
        uint40 _totalElements,
        uint40 _nextQueueIdx,
        uint40 _timestamp,
        uint40 _blockNumber
    ) internal pure returns (bytes27) {
        bytes27 extraData;
        assembly {
            extraData := _totalElements
            extraData := or(extraData, shl(40, _nextQueueIdx))
            extraData := or(extraData, shl(80, _timestamp))
            extraData := or(extraData, shl(120, _blockNumber))
            extraData := shl(40, extraData)
        }

        return extraData;
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
    ) internal {
        IChainStorageContainer batchesRef = batches();
        (uint40 totalElements, uint40 nextQueueIndex, , ) = _getBatchExtraData();

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

        // slither-disable-next-line reentrancy-no-eth, reentrancy-events
        batchesRef.push(batchHeaderHash, latestBatchContext);
    }
}

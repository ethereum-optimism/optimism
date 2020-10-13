// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_OVMCodec } from "../../libraries/codec/Lib_OVMCodec.sol";
import { Lib_AddressResolver } from "../../libraries/resolver/Lib_AddressResolver.sol";
import { Lib_MerkleUtils } from "../../libraries/utils/Lib_MerkleUtils.sol";
import { Lib_MerkleRoot } from "../../libraries/utils/Lib_MerkleRoot.sol";
import { TimeboundRingBuffer, Lib_TimeboundRingBuffer } from "../../libraries/utils/Lib_TimeboundRingBuffer.sol";
import { console } from "@nomiclabs/buidler/console.sol";

/* Interface Imports */
import { iOVM_CanonicalTransactionChain } from "../../iOVM/chain/iOVM_CanonicalTransactionChain.sol";

/* Contract Imports */
import { OVM_BaseChain } from "./OVM_BaseChain.sol";

/**
 * @title OVM_CanonicalTransactionChain
 */
contract OVM_CanonicalTransactionChain is OVM_BaseChain, Lib_AddressResolver { // TODO: re-add iOVM_CanonicalTransactionChain

    /**********
     * Events *
     *********/
    event queueTransactionAppended(bytes _queueTransaction, bytes32 timestampAndBlockNumber);
    event chainBatchAppended(uint _startingQueueIndex, uint _numQueueElements);


    /*************************************************
     * Contract Variables: Transaction Restrinctions *
     *************************************************/

    uint constant MAX_ROLLUP_TX_SIZE = 10000;
    uint constant L2_GAS_DISCOUNT_DIVISOR = 10;

    using Lib_TimeboundRingBuffer for TimeboundRingBuffer;
    TimeboundRingBuffer internal queue;
    TimeboundRingBuffer internal chain;

    struct BatchContext {
        uint numSequencedTransactions;
        uint numSubsequentQueueTransactions;
        uint timestamp;
        uint blockNumber;
    }

    struct TransactionChainElement {
        bool isSequenced;
        uint queueIndex;  // QUEUED TX ONLY
        uint timestamp;   // SEQUENCER TX ONLY
        uint blockNumber; // SEQUENCER TX ONLY
        bytes txData;     // SEQUENCER TX ONLY
    }

    /*******************************************
     * Contract Variables: Contract References *
     *******************************************/
    

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
        sequencerAddress = resolve("OVM_Sequencer");
        forceInclusionPeriodSeconds = _forceInclusionPeriodSeconds;
        queue.init(100, 50, 10000000000); // TODO: Update once we have arbitrary condition
        batches.init(100, 50, 10000000000); // TODO: Update once we have arbitrary condition
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
        require(_gasLimit >= 20000, "Layer 2 gas limit too low to enqueue.");

        // Consume l1 gas rate limit queued transactions
        uint gasToConsume = _gasLimit/L2_GAS_DISCOUNT_DIVISOR;
        uint startingGas = gasleft();
        uint i;
        while(startingGas - gasleft() > gasToConsume) {
            i++; // TODO: Replace this dumb work with minting gas token. (not today)
        }

        bytes memory queueTx = abi.encode(
            msg.sender,
            _target,
            _gasLimit,
            _data
        );
        bytes32 queueRoot = keccak256(queueTx);
        // bytes is left aligned, uint is right aligned - use this to encode them together
        bytes32 timestampAndBlockNumber = bytes32(bytes4(uint32(block.number))) | bytes32(uint256(uint40(block.timestamp)));
        // bytes32 timestampAndBlockNumber = bytes32(bytes4(uint32(999))) | bytes32(uint256(uint40(777)));
        queue.push2(queueRoot, timestampAndBlockNumber, bytes28(0));

        emit queueTransactionAppended(queueTx, timestampAndBlockNumber);
    }

    function getQueueElement(uint queueIndex) public view returns(Lib_OVMCodec.QueueElement memory) {
        uint32 trueIndex = uint32(queueIndex * 2);
        bytes32 queueRoot = queue.get(trueIndex);
        bytes32 timestampAndBlockNumber = queue.get(trueIndex + 1);
        uint40 timestamp = uint40(uint256(timestampAndBlockNumber & 0x000000000000000000000000000000000000000000000000000000ffffffffff));
        uint32 blockNumber = uint32(bytes4(timestampAndBlockNumber & 0xffffffffffffffffffffffffffffffffffffffffffffffffffffff0000000000));
        return Lib_OVMCodec.QueueElement({
            queueRoot: queueRoot,
            timestamp: timestamp,
            blockNumber: blockNumber
        });
    }

    function getLatestBatchContext() public view returns(uint40 totalElements, uint32 nextQueueIndex) {
        bytes28 extraData = batches.getExtraData();
        totalElements = uint40(uint256(uint224(extraData & 0x0000000000000000000000000000000000000000000000ffffffffff)));
        nextQueueIndex = uint32(bytes4(extraData & 0xffffffffffffffffffffffffffffffffffffffffffffff0000000000));
        return (totalElements, nextQueueIndex);
    }

    function makeLatestBatchContext(uint40 totalElements, uint32 nextQueueIndex) public view returns(bytes28) {
        bytes28 totalElementsAndNextQueueIndex = bytes28(bytes4(uint32(nextQueueIndex))) | bytes28(uint224(uint40(totalElements)));
        return totalElementsAndNextQueueIndex;
    }

    /****************************************
     * Public Functions: Batch Manipulation *
     ****************************************/

    /**
     * Appends a sequencer batch.
     */
    function appendQueueBatch(uint numQueuedTransactions)
        public
    {
        // Get all of the leaves
        (uint40 totalElements, uint32 nextQueueIndex) = getLatestBatchContext();
        bytes32[] memory leaves = new bytes32[](numQueuedTransactions);
        for (uint i = 0; i < numQueuedTransactions; i++) {
            leaves[i] = _getQueueLeafHash(nextQueueIndex);
            nextQueueIndex++;
        }

        bytes32 root = _getRoot(leaves);
        _appendBatch(
            root,
            numQueuedTransactions,
            numQueuedTransactions
        );
    }

    function _getQueueLeafHash(
        uint queueIndex
    )
        internal
        view
        returns(bytes32)
    {
        // TODO: Improve this require statement (the `queueIndex*2` is ugly)
        require(queueIndex*2 != queue.getLength(), "Queue index too large.");

        TransactionChainElement memory element = TransactionChainElement({
            isSequenced: false,
            queueIndex: queueIndex,
            timestamp: 0,
            blockNumber: 0,
            txData: hex""
        });
        require(
            msg.sender == sequencerAddress || element.timestamp + forceInclusionPeriodSeconds <= block.timestamp,
            "Message sender does not have permission to append this batch"
        );

        return _hashTransactionChainElement(element);
    }

    function _appendBatch(
        bytes32 transactionRoot,
        uint batchSize,
        uint numQueuedTransactions
    )
        internal
    {
        (uint40 totalElements, uint32 nextQueueIndex) = getLatestBatchContext();

        Lib_OVMCodec.ChainBatchHeader memory header = Lib_OVMCodec.ChainBatchHeader({
            batchIndex: batches.getLength(),
            batchRoot: transactionRoot,
            batchSize: batchSize,
            prevTotalElements: totalElements,
            extraData: hex""
        });
        bytes32 batchHeaderHash = _hashBatchHeader(header);

        bytes28 latestBatchContext = makeLatestBatchContext(
            totalElements + uint40(header.batchSize),
            nextQueueIndex + uint32(numQueuedTransactions)
        );
        batches.push(batchHeaderHash, latestBatchContext);
    }

    /**
     * Appends a sequencer batch.
     */
    function appendSequencerBatch(
        bytes[] memory _rawTransactions,                // 2 byte prefix for how many elements, per element 3 byte prefix.
        BatchContext[] memory _batchContexts,           // 2 byte prefix for how many elements, fixed size elements
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

        // Initialize an array which will contain the leaves of the merkle tree commitment
        bytes32[] memory leaves = new bytes32[](_totalElementsToAppend);
        uint32 transactionIndex = 0;
        uint32 numSequencerTransactionsProcessed = 0;
        (, uint32 nextQueueIndex) = getLatestBatchContext();
        for (uint32 batchContextIndex = 0; batchContextIndex < _batchContexts.length; batchContextIndex++) {
            //////////////////// Process Sequencer Transactions \\\\\\\\\\\\\\\\\\\\
            BatchContext memory curContext = _batchContexts[batchContextIndex];
            _validateBatchContext(curContext, nextQueueIndex);
            uint numSequencedTransactions = curContext.numSequencedTransactions;
            for (uint32 i = 0; i < numSequencedTransactions; i++) {
                leaves[transactionIndex] = keccak256(abi.encode(
                    false,
                    0,
                    curContext.timestamp,
                    curContext.blockNumber,
                    _rawTransactions[numSequencerTransactionsProcessed]
                ));
                numSequencerTransactionsProcessed++;
                transactionIndex++;
            }

            //////////////////// Process Queue Transactions \\\\\\\\\\\\\\\\\\\\
            uint numQueuedTransactions = curContext.numSubsequentQueueTransactions;
            for (uint i = 0; i < numQueuedTransactions; i++) {
                leaves[transactionIndex] = _getQueueLeafHash(nextQueueIndex);
                nextQueueIndex++;
                transactionIndex++;
            }
        }

        // Make sure the correct number of leaves were calculated
        require(transactionIndex == _totalElementsToAppend, "Not enough transactions supplied!");

        bytes32 root = _getRoot(leaves);
        uint numQueuedTransactions = _totalElementsToAppend - numSequencerTransactionsProcessed;
        _appendBatch(
            root,
            _totalElementsToAppend,
            numQueuedTransactions
        );

        emit chainBatchAppended(nextQueueIndex-numQueuedTransactions, numQueuedTransactions);
    }

    function _validateBatchContext(BatchContext memory context, uint32 nextQueueIndex) internal {
        if (nextQueueIndex == 0) {
            return;
        }
        Lib_OVMCodec.QueueElement memory nextQueueElement = getQueueElement(nextQueueIndex);
        require(
            block.timestamp < nextQueueElement.timestamp + forceInclusionPeriodSeconds,
            "Older queue batches must be processed before a new sequencer batch."
        );
        require(
            context.timestamp <= nextQueueElement.timestamp,
            "Sequencer transactions timestamp too high"
        );
        require(
            context.blockNumber <= nextQueueElement.blockNumber,
            "Sequencer transactions blockNumber too high"
        );
    }

    function getTotalElements()
        override
        public
        view
        returns (
            uint256 _totalElements
        )
    {
        (uint40 totalElements, uint32 nextQueueIndex) = getLatestBatchContext();
        return uint256(totalElements);
    }


    /******************************************
     * Internal Functions: Batch Manipulation *
     ******************************************/

    // TODO docstring
    function _hashTransactionChainElement(
        TransactionChainElement memory _element
    )
        internal
        pure
        returns(bytes32)
    {
        return keccak256(abi.encode(
            _element.isSequenced,
            _element.queueIndex,
            _element.timestamp,
            _element.blockNumber,
            _element.txData
        ));
    }

    function _getRoot(bytes32[] memory leaves) internal returns(bytes32) {
        // TODO: Require that leaves is even (if not this lib doesn't work maybe?)
        return Lib_MerkleRoot.getMerkleRoot(leaves);
    }
}
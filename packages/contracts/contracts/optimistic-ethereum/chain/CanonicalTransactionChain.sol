pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Contract Imports */
import { L1ToL2TransactionQueue } from "../queue/L1ToL2TransactionQueue.sol";
import { SafetyTransactionQueue } from "../queue/SafetyTransactionQueue.sol";

/* Library Imports */
import { ContractResolver } from "../utils/resolvers/ContractResolver.sol";
import { DataTypes } from "../utils/libraries/DataTypes.sol";
import { RollupMerkleUtils } from "../utils/libraries/RollupMerkleUtils.sol";

/**
 * @title CanonicalTransactionChain
 */
contract CanonicalTransactionChain is ContractResolver {
    /*
     * Events
     */

    event L1ToL2BatchAppended( bytes32 _batchHeaderHash);
    event SafetyQueueBatchAppended( bytes32 _batchHeaderHash);
    event SequencerBatchAppended(bytes32 _batchHeaderHash);


    /*
     * Contract Variables
     */

    address public sequencer;
    uint public forceInclusionPeriodSeconds;
    uint public forceInclusionPeriodBlocks;
    uint public cumulativeNumElements;
    bytes32[] public batches;
    uint public lastOVMTimestamp;
    uint public lastOVMBlockNumber;


    /*
     * Constructor
     */

    /**
     * @param _addressResolver Address of the AddressResolver contract.
     * @param _sequencer Address of the sequencer.
     * @param _forceInclusionPeriodSeconds Timeout in seconds when a transaction must be published.
     */
    constructor(
        address _addressResolver,
        address _sequencer,
        uint _forceInclusionPeriodSeconds
    )
        public
        ContractResolver(_addressResolver)
    {
        sequencer = _sequencer;
        forceInclusionPeriodSeconds = _forceInclusionPeriodSeconds;
        forceInclusionPeriodBlocks = _forceInclusionPeriodSeconds / 13;
        lastOVMTimestamp = 0;
    }


    /*
     * Public Functions
     */

    /**
     * @return Total number of published transaction batches.
     */
    function getBatchesLength()
        public
        view
        returns (uint)
    {
       return batches.length;
    }

    /**
     * Computes the hash of a batch header.
     * @param _batchHeader Header to hash.
     * @return Hash of the provided header.
     */
    function hashBatchHeader(
        DataTypes.TxChainBatchHeader memory _batchHeader
    )
        public
        pure
        returns (bytes32)
    {
        return keccak256(abi.encodePacked(
            _batchHeader.timestamp,
            _batchHeader.blockNumber,
            _batchHeader.isL1ToL2Tx, // TODO REPLACE WITH QUEUE ORIGIN (if you are a PR reviewer please lmk!)
            _batchHeader.elementsMerkleRoot,
            _batchHeader.numElementsInBatch,
            _batchHeader.cumulativePrevElements
        ));
    }

    /**
     * Checks whether an address is the sequencer.
     * @param _sender Address to check.
     * @return Whether or not the address is the sequencer.
     */
    function isSequencer(
        address _sender
    )
        public
        view
        returns (bool)
    {
        return _sender == sequencer;
    }

    /**
     * Attempts to append a transaction batch from pending L1 transactions.
     */
    function appendL1ToL2Batch()
        public
    {
        L1ToL2TransactionQueue l1ToL2Queue = resolveL1ToL2TransactionQueue();
        SafetyTransactionQueue safetyQueue = resolveSafetyTransactionQueue();

        DataTypes.TimestampedHash memory l1ToL2Header = l1ToL2Queue.peek();

        require(
            safetyQueue.isEmpty() || l1ToL2Header.timestamp <= safetyQueue.peekTimestamp(),
            "Must process older SafetyQueue batches first to enforce OVM timestamp monotonicity"
        );

        _appendQueueBatch(l1ToL2Header, true);
        l1ToL2Queue.dequeue();
    }

    /**
     * Attempts to append a transaction batch from the safety queue.
     */
    function appendSafetyBatch()
        public
    {
        L1ToL2TransactionQueue l1ToL2Queue = resolveL1ToL2TransactionQueue();
        SafetyTransactionQueue safetyQueue = resolveSafetyTransactionQueue();

        DataTypes.TimestampedHash memory safetyHeader = safetyQueue.peek();

        require(
            l1ToL2Queue.isEmpty() || safetyHeader.timestamp <= l1ToL2Queue.peekTimestamp(),
            "Must process older L1ToL2Queue batches first to enforce OVM timestamp monotonicity"
        );

        _appendQueueBatch(safetyHeader, false);
        safetyQueue.dequeue();
    }

    /**
     * Attempts to append a batch provided by the sequencer.
     * @param _txBatch Transaction batch to append.
     * @param _timestamp Timestamp for the batch.
     */
    function appendSequencerBatch(
        bytes[] memory _txBatch,
        uint _timestamp,
        uint _blockNumber
    )
        public
    {
        L1ToL2TransactionQueue l1ToL2Queue = resolveL1ToL2TransactionQueue();
        SafetyTransactionQueue safetyQueue = resolveSafetyTransactionQueue();

        require(
            isSequencer(msg.sender),
            "Message sender does not have permission to append a batch"
        );

        require(
            _txBatch.length > 0,
            "Cannot submit an empty batch"
        );

        require(
            _timestamp + forceInclusionPeriodSeconds > now,
            "Cannot submit a batch with a timestamp older than the sequencer inclusion period"
        );

        require(
            _blockNumber + forceInclusionPeriodBlocks > block.number,
            "Cannot submit a batch with a blockNumber older than the sequencer inclusion period"
        );

        require(
            _timestamp <= now,
            "Cannot submit a batch with a timestamp in the future"
        );

        require(
            _blockNumber <= block.number,
            "Cannot submit a batch with a blockNumber in the future"
        );

        if (!l1ToL2Queue.isEmpty()) {
            require(
                _timestamp <= l1ToL2Queue.peekTimestamp(),
                "Must process older L1ToL2Queue batches first to enforce OVM timestamp monotonicity"
            );

            require(
                _blockNumber <= l1ToL2Queue.peekBlockNumber(),
                "Must process older L1ToL2Queue batches first to enforce OVM blockNumber monotonicity"
            );
        }

        if (!safetyQueue.isEmpty()) {
            require(
                _timestamp <= safetyQueue.peekTimestamp(),
                "Must process older SafetyQueue batches first to enforce OVM timestamp monotonicity"
            );

            require(
                _blockNumber <= safetyQueue.peekBlockNumber(),
                "Must process older SafetyQueue batches first to enforce OVM blockNumber monotonicity"
            );
        }

        require(
            _timestamp >= lastOVMTimestamp,
            "Timestamps must monotonically increase"
        );

        require(
            _blockNumber >= lastOVMBlockNumber,
            "BlockNumbers must monotonically increase"
        );

        lastOVMTimestamp = _timestamp;
        lastOVMBlockNumber = _blockNumber;

        RollupMerkleUtils merkleUtils = resolveRollupMerkleUtils();
        bytes32 batchHeaderHash = keccak256(abi.encodePacked(
            _timestamp,
            _blockNumber,
            false, // isL1ToL2Tx TODO: replace with queue origin
            merkleUtils.getMerkleRoot(_txBatch), // elementsMerkleRoot
            _txBatch.length, // numElementsInBatch
            cumulativeNumElements // cumulativeNumElements
        ));

        batches.push(batchHeaderHash);
        cumulativeNumElements += _txBatch.length;

        emit SequencerBatchAppended(batchHeaderHash);
    }

    /**
     * Checks that an element is included within a published batch.
     * @param _element Element to prove within the batch.
     * @param _position Index of the element within the batch.
     * @param _inclusionProof Inclusion proof for the element/batch.
     */
    function verifyElement(
        bytes memory _element,
        uint _position,
        DataTypes.TxElementInclusionProof memory _inclusionProof
    )
        public
        view
        returns (bool)
    {
        // For convenience, store the batchHeader
        DataTypes.TxChainBatchHeader memory batchHeader = _inclusionProof.batchHeader;

        // make sure absolute position equivalent to relative positions
        if (_position != _inclusionProof.indexInBatch +
            batchHeader.cumulativePrevElements) {
            return false;
        }

        // verify elementsMerkleRoot
        RollupMerkleUtils merkleUtils = resolveRollupMerkleUtils();
        if (!merkleUtils.verify(
            batchHeader.elementsMerkleRoot,
            _element,
            _inclusionProof.indexInBatch,
            _inclusionProof.siblings
        )) {
            return false;
        }

        //compare computed batch header with the batch header in the list.
        return hashBatchHeader(batchHeader) == batches[_inclusionProof.batchIndex];
    }


    /*
     * Internal Functions
     */

    /**
     * Appends a batch.
     * @param _timestampedHash Timestamped transaction hash.
     * @param _isL1ToL2Tx Whether or not this is an L1-L2 transaction.
     */
    function _appendQueueBatch(
        DataTypes.TimestampedHash memory _timestampedHash,
        bool _isL1ToL2Tx
    )
        internal
    {
        uint timestamp = _timestampedHash.timestamp;
        uint blockNumber = _timestampedHash.blockNumber;

        require(
            timestamp + forceInclusionPeriodSeconds <= now || isSequencer(msg.sender),
            "Message sender does not have permission to append this batch"
        );

        lastOVMTimestamp = timestamp;
        lastOVMBlockNumber = blockNumber;
        bytes32 elementsMerkleRoot = _timestampedHash.txHash;
        uint numElementsInBatch = 1;

        bytes32 batchHeaderHash = keccak256(abi.encodePacked(
            timestamp,
            blockNumber,
            _isL1ToL2Tx,
            elementsMerkleRoot,
            numElementsInBatch,
            cumulativeNumElements // cumulativePrevElements
        ));

        batches.push(batchHeaderHash);
        cumulativeNumElements += numElementsInBatch;

        if (_isL1ToL2Tx) {
            emit L1ToL2BatchAppended(batchHeaderHash);
        } else {
            emit SafetyQueueBatchAppended(batchHeaderHash);
        }
    }


    /*
     * Contract Resolution
     */

    function resolveL1ToL2TransactionQueue()
        internal
        view
        returns (L1ToL2TransactionQueue)
    {
        return L1ToL2TransactionQueue(resolveContract("L1ToL2TransactionQueue"));
    }

    function resolveSafetyTransactionQueue()
        internal
        view
        returns (SafetyTransactionQueue)
    {
        return SafetyTransactionQueue(resolveContract("SafetyTransactionQueue"));
    }

    function resolveRollupMerkleUtils()
        internal
        view
        returns (RollupMerkleUtils)
    {
        return RollupMerkleUtils(resolveContract("RollupMerkleUtils"));
    }
}

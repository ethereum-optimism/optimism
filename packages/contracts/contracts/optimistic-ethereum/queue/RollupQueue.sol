pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { DataTypes } from "../utils/libraries/DataTypes.sol";

/**
 * @title RollupQueue
 */
contract RollupQueue {
    /*
    * Events
    */
    event CalldataTxEnqueued();
    event L1ToL2TxEnqueued(bytes _tx);


    /*
    * Contract Variables
    */

    DataTypes.TimestampedHash[] public batchHeaders;
    uint256 public front;


    /*
    * Public Functions
    */

    /**
     * Gets the total number of batches.
     * @return Total submitted batches.
     */
    function getBatchHeadersLength()
        public
        view
        returns (uint)
    {
        return batchHeaders.length;
    }

    /**
     * Checks if the queue is empty.
     * @return Whether or not the queue is empty.
     */
    function isEmpty()
        public
        view
        returns (bool)
    {
        return front >= batchHeaders.length;
    }

    /**
     * Peeks the front element on the queue.
     * @return Front queue element.
     */
    function peek()
        public
        view
        returns (DataTypes.TimestampedHash memory)
    {
        require(!isEmpty(), "Queue is empty, no element to peek at");
        return batchHeaders[front];
    }

    /**
     * Peeks the timestamp of the front element on the queue.
     * @return Front queue element timestamp.
     */
    function peekTimestamp()
        public
        view
        returns (uint)
    {
        DataTypes.TimestampedHash memory frontBatch = peek();
        return frontBatch.timestamp;
    }

    /**
     * Checks whether a sender is allowed to enqueue.
     * @param _sender Sender address to check.
     * @return Whether or not the sender can enqueue.
     */
    function authenticateEnqueue(
        address _sender
    )
        public
        view
        returns (bool)
    {
        return true;
    }

    /**
     * Checks whether a sender is allowed to dequeue.
     * @param _sender Sender address to check.
     * @return Whether or not the sender can dequeue.
     */
    function authenticateDequeue(
        address _sender
    )
        public
        view
        returns (bool)
    {
        return true;
    }

    /**
     * Checks if this is a calldata transaction queue.
     * @return Whether or not this is a calldata tx queue.
     */
    function isCalldataTxQueue()
        public
        returns (bool)
    {
        return true;
    }

    /**
     * Attempts to enqueue a transaction.
     * @param _tx Transaction data to enqueue.
     */
    function enqueueTx(
        bytes memory _tx
    )
        public
    {
        // Authentication.
        require(
            authenticateEnqueue(msg.sender),
            "Message sender does not have permission to enqueue"
        );

        bytes32 txHash = keccak256(_tx);

        batchHeaders.push(DataTypes.TimestampedHash({
            timestamp: now,
            txHash: txHash
        }));

        if (isCalldataTxQueue()) {
            emit CalldataTxEnqueued();
        } else {
            emit L1ToL2TxEnqueued(_tx);
        }
    }

    /**
     * Attempts to dequeue a transaction.
     */
    function dequeue()
        public
    {
        // Authentication.
        require(
            authenticateDequeue(msg.sender),
            "Message sender does not have permission to dequeue"
        );

        require(front < batchHeaders.length, "Cannot dequeue from an empty queue");

        delete batchHeaders[front];
        front++;
    }
}

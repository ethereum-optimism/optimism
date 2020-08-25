pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { DataTypes } from "../utils/libraries/DataTypes.sol";

/**
 * @title RollupQueue
 */
contract RollupQueue {
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
     * @return Front queue element timestamp (lowest in queue).
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
     * Peeks the blocknumber of the front element on the queue.
     * @return Front queue element blocknumber (lowest in queue).
     */
    function peekBlocknumber()
        public
        view
        returns (uint)
    {
        DataTypes.TimestampedHash memory frontBatch = peek();
        return frontBatch.blocknumber;
    }
    
    /*
    * Internal Functions
    */

    /**
     * Attempts to enqueue a single data block (i.e. will not be merklized).
     * @param _data Transaction data to enqueue.
     */
    function _enqueue(
        bytes memory _data
    )
        internal
    {
        bytes32 txHash = keccak256(_data);

        batchHeaders.push(DataTypes.TimestampedHash({
            txHash: txHash,
            timestamp: now,
            blocknumber: block.number
        }));
    }

    /**
     * Attempts to dequeue a transaction.
     */
    function _dequeue()
        internal
    {
        require(front < batchHeaders.length, "Cannot dequeue from an empty queue");

        delete batchHeaders[front];
        front++;
    }
}

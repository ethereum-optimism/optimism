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

    function getBatchHeadersLength()
        public
        view
        returns (uint)
    {
        return batchHeaders.length;
    }

    function isEmpty()
        public
        view
        returns (bool)
    {
        return front >= batchHeaders.length;
    }

    function peek()
        public
        view
        returns (DataTypes.TimestampedHash memory)
    {
        require(!isEmpty(), "Queue is empty, no element to peek at");
        return batchHeaders[front];
    }

    function peekTimestamp()
        public
        view
        returns (uint)
    {
        DataTypes.TimestampedHash memory frontBatch = peek();
        return frontBatch.timestamp;
    }

    function authenticateEnqueue(
        address _sender
    )
        public
        view
        returns (bool)
    {
        return true;
    }

    function authenticateDequeue(
        address _sender
    )
        public
        view
        returns (bool)
    {
        return true;
    }

    function isCalldataTxQueue()
        public
        returns (bool)
    {
        return true;
    }

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

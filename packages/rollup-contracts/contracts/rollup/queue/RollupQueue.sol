pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {DataTypes as dt} from "../utils/DataTypes.sol";
import {RollupMerkleUtils} from "../utils/RollupMerkleUtils.sol";

contract RollupQueue {
  // List of batch header hashes
  dt.TimestampedHash[] public batchHeaders;
  uint256 public front; //Index of the first batchHeaderHash in the list

  // The Rollup Merkle Tree library (currently a contract for ease of testing)
  RollupMerkleUtils merkleUtils;

  /***************
   * Constructor *
   **************/
  constructor(address _rollupMerkleUtilsAddress) public {
    merkleUtils = RollupMerkleUtils(_rollupMerkleUtilsAddress);
    front = 0;
  }

  function getBatchHeadersLength() public view returns (uint) {
    return batchHeaders.length;
  }

  function isEmpty() public view returns (bool) {
    return front >= batchHeaders.length;
  }

  function peek() public view returns (dt.TimestampedHash memory) {
    require(!isEmpty(), "Queue is empty, no element to peek at");
    return batchHeaders[front];
  }

  function peekTimestamp() public view returns (uint) {
    dt.TimestampedHash memory frontBatch = peek();
    return frontBatch.timestamp;
  }

  function authenticateEnqueue(address _sender) public view returns (bool) { return true; }
  function authenticateDequeue(address _sender) public view returns (bool) { return true; }

  function enqueueTx(bytes memory _tx) public {
    require(authenticateEnqueue(msg.sender), "Message sender does not have permission to enqueue");
    dt.TimestampedHash memory timestampedHash = dt.TimestampedHash(
      now,
      keccak256(_tx)
    );
    batchHeaders.push(timestampedHash);
  }

  function dequeue() public {
    require(authenticateDequeue(msg.sender), "Message sender does not have permission to dequeue");
    require(front < batchHeaders.length, "Cannot dequeue from an empty queue");
    delete batchHeaders[front];
    front++;
  }
}

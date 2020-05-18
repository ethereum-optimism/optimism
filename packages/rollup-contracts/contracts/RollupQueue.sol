pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {DataTypes as dt} from "./DataTypes.sol";
import {RollupMerkleUtils} from "./RollupMerkleUtils.sol";

contract RollupQueue {
  // How many elements in total have been appended
  uint public cumulativeNumElements;
  // List of batch header hashes
  bytes32[] public batches;
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
  // for testing: returns length of batch list
  function getBatchesLength() public view returns (uint) {
    return batches.length;
  }

  function authenticateEnqueue(address _sender) public view returns (bool) { return true; }
  function authenticateDequeue(address _sender) public view returns (bool) { return true; }

  // appends to the current list of batches
  function enqueueBatch(bytes[] memory _rollupBatch) public {
    //Check that msg.sender is authorized to append
    require(authenticateEnqueue(msg.sender), "Message sender does not have permission to enqueue");
    require(_rollupBatch.length > 0, "Cannot submit an empty batch");
    // calculate batch header
    bytes32 batchHeaderHash = keccak256(
      abi.encodePacked(
        merkleUtils.getMerkleRoot(_rollupBatch), // elementsMerkleRoot
        _rollupBatch.length // numElementsInBatch
      )
    );
    // store batch header
    batches.push(batchHeaderHash);
    // update cumulative elements
    cumulativeNumElements += _rollupBatch.length;
  }

  // dequeues all batches including and before the given batch index
  function dequeueBeforeInclusive(uint _batchIndex) public {
    //Check that msg.sender is authorized to delete
    require(authenticateDequeue(msg.sender), "Message sender does not have permission to dequeue");
    //batchIndex is between first and last batches
    require(_batchIndex >= front && _batchIndex < batches.length, "Cannot delete batches outside of valid range");
    //delete all batch headers before and including batchIndex
    for (uint i = front; i <= _batchIndex; i++) {
        delete batches[i];
    }
    //keep track of new head of list
    front = _batchIndex + 1;
    // Note: keep in mind that front can point to a non-existent batch if the list is empty.
  }
}

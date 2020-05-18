pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {DataTypes as dt} from "./DataTypes.sol";
import {RollupMerkleUtils} from "./RollupMerkleUtils.sol";

contract RollupQueue {
  // How many elements in total have been appended
  uint public cumulativeNumElements;

  // List of block header hashes
  bytes32[] public blocks;

  uint256 public front; //Index of the first blockHeaderHash in the list

  // The Rollup Merkle Tree library (currently a contract for ease of testing)
  RollupMerkleUtils merkleUtils;

  /***************
   * Constructor *
   **************/
  constructor(address _rollupMerkleUtilsAddress) public {
    merkleUtils = RollupMerkleUtils(_rollupMerkleUtilsAddress);
    front = 0;
  }
  // for testing: returns length of block list
  function getBlocksLength() public view returns (uint) {
    return blocks.length;
  }

  function authenticateEnqueue(address _sender) public view returns (bool) { return true; }
  function authenticateDequeue(address _sender) public view returns (bool) { return true; }

  // appends to the current list of blocks
  function enqueueBlock(bytes[] memory _rollupBlock) public {
    //Check that msg.sender is authorized to append
    require(authenticateEnqueue(msg.sender), "Message sender does not have permission to enqueue");
    require(_rollupBlock.length > 0, "Cannot submit an empty block");
    // calculate block header
    bytes32 blockHeaderHash = keccak256(
      abi.encodePacked(
        merkleUtils.getMerkleRoot(_rollupBlock), // elementsMerkleRoot
        _rollupBlock.length // numElementsInBlock
      )
    );
    // store block header
    blocks.push(blockHeaderHash);
    // update cumulative elements
    cumulativeNumElements += _rollupBlock.length;
  }

  // dequeues all blocks including and before the given block index
  function dequeueBeforeInclusive(uint _blockIndex) public {
    //Check that msg.sender is authorized to delete
    require(authenticateDequeue(msg.sender), "Message sender does not have permission to dequeue");
    //blockIndex is between first and last blocks
    require(_blockIndex >= front && _blockIndex < blocks.length, "Cannot delete blocks outside of valid range");
    //delete all block headers before and including blockIndex
    for (uint i = front; i <= _blockIndex; i++) {
        delete blocks[i];
    }
    //keep track of new head of list
    front = _blockIndex + 1;
    // Note: keep in mind that front can point to a non-existent block if the list is empty.
  }
}

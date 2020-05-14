pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {DataTypes as dt} from "./DataTypes.sol";
import {RollupMerkleUtils} from "./RollupMerkleUtils.sol";

contract RollupList {
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

  function hashBlockHeader(
    dt.BlockHeader memory _blockHeader
  ) public pure returns (bytes32) {
    return keccak256(abi.encodePacked(
      _blockHeader.ethBlockNumber,
      _blockHeader.elementsMerkleRoot,
      _blockHeader.numElementsInBlock,
      _blockHeader.cumulativePrevElements
    ));
  }

  function authenticateEnqueue(address _sender) public view returns (bool) { return true; }
  function authenticateDelete(address _sender) public view returns (bool) { return true; }
  function authenticateDequeue(address _sender) public view returns (bool) { return true; }

  // appends to the current list of blocks
  function enqueueBlock(bytes[] memory _rollupBlock) public {
    //Check that msg.sender is authorized to append
    require(authenticateEnqueue(msg.sender), "Message sender does not have permission to enqueue");
    require(_rollupBlock.length > 0, "Cannot submit an empty block");
    // calculate block header
    bytes32 blockHeaderHash = keccak256(abi.encodePacked(
      block.number, // ethBlockNumber
      merkleUtils.getMerkleRoot(_rollupBlock), // elementsMerkleRoot
      _rollupBlock.length, // numElementsInBlock
      cumulativeNumElements // cumulativeNumElements
    ));
    // store block header
    blocks.push(blockHeaderHash);
    // update cumulative elements
    cumulativeNumElements += _rollupBlock.length;
  }

  // verifies an element is in the current list at the given position
  function verifyElement(
     bytes memory _element, // the element of the list being proven
     uint _position, // the position in the list of the element being proven
     dt.ElementInclusionProof memory _inclusionProof  // inclusion proof in the rollup block
  ) public view returns (bool) {
    // For convenience, store the blockHeader
    dt.BlockHeader memory blockHeader = _inclusionProof.blockHeader;
    // make sure absolute position equivalent to relative positions
    if(_position != _inclusionProof.indexInBlock +
      blockHeader.cumulativePrevElements)
      return false;

    // verify elementsMerkleRoot
    if (!merkleUtils.verify(
      blockHeader.elementsMerkleRoot,
      _element,
      _inclusionProof.indexInBlock,
      _inclusionProof.siblings
    )) return false;
    //compare computed block header with the block header in the list.
    return hashBlockHeader(blockHeader) == blocks[_inclusionProof.blockIndex];
  }

  // deletes all blocks including and after the given block number
  // TODO: rename to popAfterInclusive?
  function deleteAfterInclusive(
     uint _blockIndex, // delete this block index and those following
     dt.BlockHeader memory _blockHeader
  ) public {
    //Check that msg.sender is authorized to delete
    require(authenticateDelete(msg.sender), "Message sender does not have permission to delete blocks");
    //blockIndex is between first and last blocks
    require(_blockIndex >= front && _blockIndex < blocks.length, "Cannot delete blocks outside of valid range");
    // make sure the provided state to revert to is correct
    bytes32 calculatedBlockHeaderHash = hashBlockHeader(_blockHeader);
    require(calculatedBlockHeaderHash == blocks[_blockIndex], "Calculated block header is different than expected block header");
    // revert back to the state as specified
    blocks.length = _blockIndex;
    cumulativeNumElements = _blockHeader.cumulativePrevElements;
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
    //TODO Note: keep in mind that front can point to a non-existent block if the list is empty.
  }
}

pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {DataTypes as dt} from "./DataTypes.sol";
import {RollupMerkleUtils} from "./RollupMerkleUtils.sol";

contract CanonicalTransactionChain {
  // The Rollup Merkle Tree library (currently a contract for ease of testing)
  RollupMerkleUtils merkleUtils;
  address public sequencer;

  // How many elements in total have been appended
  uint public cumulativeNumElements;
  // List of block header hashes
  bytes32[] public blocks;


  constructor(
    address _rollupMerkleUtilsAddress,
    address _sequencer
  ) public {
    merkleUtils = RollupMerkleUtils(_rollupMerkleUtilsAddress);
    sequencer = _sequencer;
  }

  // for testing: returns length of block list
  function getBlocksLength() public view returns (uint) {
    return blocks.length;
  }

  function hashBlockHeader(
    dt.BlockHeader memory _blockHeader
  ) public pure returns (bytes32) {
    return keccak256(abi.encodePacked(
      _blockHeader.timestamp,
      _blockHeader.isL1ToL2Tx,
      _blockHeader.elementsMerkleRoot,
      _blockHeader.numElementsInBlock,
      _blockHeader.cumulativePrevElements
    ));
  }

  function authenticateEnqueue(address _sender) public view returns (bool) {
    return _sender == sequencer;
  }
  function authenticateDequeue(address _sender) public view returns (bool) { return false; }
  function authenticateDelete(address _sender) public view returns (bool) { return false; }

  // appends to the current list of blocks
  function appendTransactionBatch(bytes[] memory _txBatch, uint _timestamp) public {
    //Check that msg.sender is authorized to append
    require(authenticateEnqueue(msg.sender), "Message sender does not have permission to enqueue");
    require(_txBatch.length > 0, "Cannot submit an empty block");

    // require(_timestamp > lastOVMTimestamp, "timestamps must monotonically increase");
    // lastOVMTimestamp = _timestamp;
    // require dist(_timestamp, block.timestamp) < sequencerLivenessAssumption
    // require(L1ToL2Queue.ageOfOldestQueuedBlock() < sequencerLivenessAssumption, "must process all L1->L2 blocks older than liveness assumption before processing L2 blocks.")

    // calculate block header
    bytes32 blockHeaderHash = keccak256(abi.encodePacked(
      _timestamp,
      false, // isL1ToL2Tx
      merkleUtils.getMerkleRoot(_txBatch), // elementsMerkleRoot
      _txBatch.length, // numElementsInBlock
      cumulativeNumElements // cumulativeNumElements
    ));
    // store block header
    blocks.push(blockHeaderHash);
    // update cumulative elements
    cumulativeNumElements += _txBatch.length;
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
}

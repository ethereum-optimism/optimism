pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {RollupList} from "./RollupList.sol";

contract CanonicalTransactionChain is RollupList {
  address public sequencer;
  address public canonicalTransactionChain;

  constructor(
    address _rollupMerkleUtilsAddress,
    address _sequencer,
    address _canonicalTransactionChain
  ) RollupList(_rollupMerkleUtilsAddress) public {
    sequencer = _sequencer;
    canonicalTransactionChain = _canonicalTransactionChain;
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



    // // calculate block header
    // bytes32 blockHeaderHash = keccak256(abi.encodePacked(
    //   _timestamp, //timestamp, duh
    //   false, //isL1ToL2Tx
    //   merkleUtils.getMerkleRoot(_txBatch), // elementsMerkleRoot
    //   _txBatch.length, // numElementsInBlock
    //   cumulativeNumElements // cumulativePrevElements
    // ));
    // // store block header
    // blocks.push(blockHeaderHash);
    // // update cumulative elements
    // cumulativeNumElements += _txBatch.length;
  }
}

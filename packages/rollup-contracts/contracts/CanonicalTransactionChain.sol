pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {DataTypes as dt} from "./DataTypes.sol";
import {RollupMerkleUtils} from "./RollupMerkleUtils.sol";
import {L1ToL2TransactionQueue} from "./L1ToL2TransactionQueue.sol";
import {SafetyTransactionQueue} from "./SafetyTransactionQueue.sol";

contract CanonicalTransactionChain {
  address public sequencer;
  uint public sequencerLivenessAssumption;
  RollupMerkleUtils public merkleUtils;
  L1ToL2TransactionQueue public l1ToL2Queue;
  SafetyTransactionQueue public safetyQueue;
  uint public cumulativeNumElements;
  bytes32[] public batches;
  uint public lastOVMTimestamp;

  constructor(
    address _rollupMerkleUtilsAddress,
    address _sequencer,
    address _l1ToL2TransactionPasserAddress,
    uint _sequencerLivenessAssumption
  ) public {
    merkleUtils = RollupMerkleUtils(_rollupMerkleUtilsAddress);
    sequencer = _sequencer;
    l1ToL2Queue = new L1ToL2TransactionQueue(_rollupMerkleUtilsAddress, _l1ToL2TransactionPasserAddress, address(this));
    safetyQueue = new SafetyTransactionQueue(_rollupMerkleUtilsAddress, address(this));
    sequencerLivenessAssumption =_sequencerLivenessAssumption;
    lastOVMTimestamp = 0;
  }

  function getBatchesLength() public view returns (uint) {
    return batches.length;
  }

  function hashBatchHeader(
    dt.TxChainBatchHeader memory _batchHeader
  ) public pure returns (bytes32) {
    return keccak256(abi.encodePacked(
      _batchHeader.timestamp,
      _batchHeader.isL1ToL2Tx,
      _batchHeader.elementsMerkleRoot,
      _batchHeader.numElementsInBatch,
      _batchHeader.cumulativePrevElements
    ));
  }

  function authenticateAppend(address _sender) public view returns (bool) {
    return _sender == sequencer;
  }

  function appendL1ToL2Batch() public {
    dt.TimestampedHash memory l1ToL2Header = l1ToL2Queue.peek();
    if(!safetyQueue.isEmpty()) {
       require(l1ToL2Header.timestamp <= safetyQueue.peekTimestamp(), "Must process older SafetyQueue batches first to enforce timestamp monotonicity");
    }
    _appendQueueBatch(l1ToL2Header, true);
    l1ToL2Queue.dequeue();
  }

  function appendSafetyBatch() public {
    dt.TimestampedHash memory safetyHeader = safetyQueue.peek();
    if(!l1ToL2Queue.isEmpty()) {
       require(safetyHeader.timestamp <= l1ToL2Queue.peekTimestamp(), "Must process older L1ToL2Queue batches first to enforce timestamp monotonicity");
    }
    _appendQueueBatch(safetyHeader, false);
    safetyQueue.dequeue();
  }

  function _appendQueueBatch(
    dt.TimestampedHash memory timestampedHash,
    bool isL1ToL2Tx
  ) internal {
    uint timestamp = timestampedHash.timestamp;
    if (timestamp + sequencerLivenessAssumption > now) {
      require(authenticateAppend(msg.sender), "Message sender does not have permission to append this batch");
    }
    lastOVMTimestamp = timestamp;
    bytes32 elementsMerkleRoot = timestampedHash.txHash;
    uint numElementsInBatch = 1;
    bytes32 batchHeaderHash = keccak256(abi.encodePacked(
      timestamp,
      isL1ToL2Tx, // isL1ToL2Tx
      elementsMerkleRoot,
      numElementsInBatch,
      cumulativeNumElements // cumulativePrevElements
    ));
    batches.push(batchHeaderHash);
    cumulativeNumElements += numElementsInBatch;
  }

  function appendTransactionBatch(bytes[] memory _txBatch, uint _timestamp) public {
    require(authenticateAppend(msg.sender), "Message sender does not have permission to append a batch");
    require(_txBatch.length > 0, "Cannot submit an empty batch");
    require(_timestamp + sequencerLivenessAssumption > now, "Cannot submit a batch with a timestamp older than the sequencer liveness assumption");
    require(_timestamp <= now, "Cannot submit a batch with a timestamp in the future");
    if(!l1ToL2Queue.isEmpty()) {
      require(_timestamp <= l1ToL2Queue.peekTimestamp(), "Must process older L1ToL2Queue batches first to enforce timestamp monotonicity");
    }
    if(!safetyQueue.isEmpty()) {
      require(_timestamp <= safetyQueue.peekTimestamp(), "Must process older SafetyQueue batches first to enforce timestamp monotonicity");
    }
    require(_timestamp >= lastOVMTimestamp, "Timestamps must monotonically increase");
    lastOVMTimestamp = _timestamp;
    bytes32 batchHeaderHash = keccak256(abi.encodePacked(
      _timestamp,
      false, // isL1ToL2Tx
      merkleUtils.getMerkleRoot(_txBatch), // elementsMerkleRoot
      _txBatch.length, // numElementsInBatch
      cumulativeNumElements // cumulativeNumElements
    ));
    batches.push(batchHeaderHash);
    cumulativeNumElements += _txBatch.length;
  }

  // verifies an element is in the current list at the given position
  function verifyElement(
     bytes memory _element, // the element of the list being proven
     uint _position, // the position in the list of the element being proven
     dt.ElementInclusionProof memory _inclusionProof  // inclusion proof in the rollup batch
  ) public view returns (bool) {
    // For convenience, store the batchHeader
    dt.TxChainBatchHeader memory batchHeader = _inclusionProof.batchHeader;
    // make sure absolute position equivalent to relative positions
    if(_position != _inclusionProof.indexInBatch +
      batchHeader.cumulativePrevElements)
      return false;
    // verify elementsMerkleRoot
    if (!merkleUtils.verify(
      batchHeader.elementsMerkleRoot,
      _element,
      _inclusionProof.indexInBatch,
      _inclusionProof.siblings
    )) return false;
    //compare computed batch header with the batch header in the list.
    return hashBatchHeader(batchHeader) == batches[_inclusionProof.batchIndex];
  }
}

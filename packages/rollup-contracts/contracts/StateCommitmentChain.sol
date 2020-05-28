pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {DataTypes as dt} from "./DataTypes.sol";
import {RollupMerkleUtils} from "./RollupMerkleUtils.sol";
import {CanonicalTransactionChain} from "./CanonicalTransactionChain.sol";

contract StateCommitmentChain {
  CanonicalTransactionChain canonicalTransactionChain;
  RollupMerkleUtils public merkleUtils;
  uint public cumulativeNumElements;
  bytes32[] public batches;

  constructor(
    address _rollupMerkleUtilsAddress,
    address _canonicalTransactionChain
  ) public {
    merkleUtils = RollupMerkleUtils(_rollupMerkleUtilsAddress);
    canonicalTransactionChain = CanonicalTransactionChain(_canonicalTransactionChain);
  }

  function getBatchesLength() public view returns (uint) {
    return batches.length;
  }

  function hashBatchHeader(
    dt.StateChainBatchHeader memory _batchHeader
  ) public pure returns (bytes32) {
    return keccak256(abi.encodePacked(
      _batchHeader.elementsMerkleRoot,
      _batchHeader.numElementsInBatch,
      _batchHeader.cumulativePrevElements
    ));
  }

  function appendStateBatch(bytes[] memory _stateBatch) public {
    require(cumulativeNumElements + _stateBatch.length <= canonicalTransactionChain.cumulativeNumElements(),
      "Cannot append more state commitments than total number of transactions in CanonicalTransactionChain");
    require(_stateBatch.length > 0, "Cannot submit an empty state commitment batch");
    bytes32 batchHeaderHash = keccak256(abi.encodePacked(
      merkleUtils.getMerkleRoot(_stateBatch), // elementsMerkleRoot
      _stateBatch.length, // numElementsInBatch
      cumulativeNumElements // cumulativeNumElements
    ));
    batches.push(batchHeaderHash);
    cumulativeNumElements += _stateBatch.length;
  }

  // verifies an element is in the current list at the given position
  function verifyElement(
     bytes memory _element, // the element of the list being proven
     uint _position, // the position in the list of the element being proven
     dt.StateElementInclusionProof memory _inclusionProof  // inclusion proof in the rollup batch
  ) public view returns (bool) {
    // For convenience, store the batchHeader
    dt.StateChainBatchHeader memory batchHeader = _inclusionProof.batchHeader;
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

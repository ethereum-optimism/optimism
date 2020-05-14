pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {RollupList} from "./RollupList.sol";

contract RollupTransactionQueue is RollupList {
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
  function authenticateDequeue(address _sender) public view returns (bool) {
    return _sender == canonicalTransactionChain;
  }
  function authenticateDelete(address _sender) public view returns (bool) { return false; }
}

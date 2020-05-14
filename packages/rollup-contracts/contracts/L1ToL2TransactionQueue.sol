pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {RollupList} from "./RollupList.sol";

contract L1ToL2TransactionQueue is RollupList {
  address public l1ToL2TransactionPasser;
  address public canonicalTransactionChain;

  constructor(
    address _rollupMerkleUtilsAddress,
    address _l1ToL2TransactionPasser,
    address _canonicalTransactionChain
  ) RollupList(_rollupMerkleUtilsAddress) public {
    l1ToL2TransactionPasser = _l1ToL2TransactionPasser;
    canonicalTransactionChain = _canonicalTransactionChain;
  }

  function authenticateEnqueue(address _sender) public view returns (bool) {
    return _sender == l1ToL2TransactionPasser;
  }
  function authenticateDequeue(address _sender) public view returns (bool) {
    return _sender == canonicalTransactionChain;
  }
  function authenticateDelete(address _sender) public view returns (bool) { return false; }
}

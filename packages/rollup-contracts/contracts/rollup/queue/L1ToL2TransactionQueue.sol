pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import { RollupQueue } from "./RollupQueue.sol";

contract L1ToL2TransactionQueue is RollupQueue {
  address public l1ToL2TransactionPasser;
  address public canonicalTransactionChain;

  constructor(
    address _l1ToL2TransactionPasser,
    address _canonicalTransactionChain
  ) public {
    l1ToL2TransactionPasser = _l1ToL2TransactionPasser;
    canonicalTransactionChain = _canonicalTransactionChain;
  }

  function authenticateEnqueue(address _sender) public view returns (bool) {
    return _sender == l1ToL2TransactionPasser;
  }

  function authenticateDequeue(address _sender) public view returns (bool) {
    return _sender == canonicalTransactionChain;
  }
}

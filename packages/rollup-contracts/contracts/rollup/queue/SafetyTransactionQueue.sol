pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import { RollupQueue } from "./RollupQueue.sol";

contract SafetyTransactionQueue is RollupQueue {
  address public canonicalTransactionChain;

  constructor(
    address _canonicalTransactionChain
  ) public {
    canonicalTransactionChain = _canonicalTransactionChain;
  }

  function authenticateDequeue(address _sender) public view returns (bool) {
    return _sender == canonicalTransactionChain;
  }
}

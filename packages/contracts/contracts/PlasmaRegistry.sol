pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* External Imports */
import "openzeppelin-solidity/contracts/token/ERC20/ERC20.sol";

/* Internal Imports */
import { Aggregator } from "./Aggregator.sol";

contract PlasmaRegistry {
  address[] public aggregators;
  uint counter;

  constructor () public {
    counter = 0;
  }

  function addAggregator(address _authenticationAddress) public returns (Aggregator newAggregator) {
    counter += 1;
    Aggregator aggregator = new Aggregator(_authenticationAddress, counter);
    aggregators.push(address(aggregator));
    return aggregator;
  }
}
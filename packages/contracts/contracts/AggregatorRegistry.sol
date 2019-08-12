pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* External Imports */
import "openzeppelin-solidity/contracts/token/ERC20/ERC20.sol";

/* Internal Imports */
import { Aggregator } from "./Aggregator.sol";

/* Maintains the list of aggregators */
contract AggregatorRegistry {
  address[] public aggregators;

  /* Add a new aggregator to this contract */
  function addAggregator(address _authenticationAddress) public returns (Aggregator newAggregator) {
    uint id = aggregators.length + 1;
    Aggregator aggregator = new Aggregator(_authenticationAddress, id);
    aggregators.push(address(aggregator));
    return aggregator;
  }

  /* Get the current number of aggregators registered */
  function getAggregatorCount() public view returns (uint256) {
    return aggregators.length;
  }
}
pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* External Imports */
import "openzeppelin-solidity/contracts/token/ERC20/ERC20.sol";

/* Internal Imports */
// import { Aggregator } from "./Aggregator.sol";
import { DummyAggregator } from "./DummyAggregator.sol";


contract PlasmaRegistry {
  address[] public aggregators;

  function addAggregator(address _authenticationAddress) public returns (DummyAggregator newAggregator) {
    uint id = aggregators.length + 1;
    DummyAggregator aggregator = new DummyAggregator();
    // DummyAggregator aggregator = new DummyAggregator(_authenticationAddress, id);
    aggregators.push(address(aggregator));
    return aggregator;
  }

  function getAggregatorCount() public returns (uint count) {
    return aggregators.length;
  }
}
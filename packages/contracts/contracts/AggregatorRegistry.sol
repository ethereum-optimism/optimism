pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* External Imports */
import "openzeppelin-solidity/contracts/token/ERC20/ERC20.sol";

/* Internal Imports */
import { Aggregator } from "./Aggregator.sol";

/**
 * @notice Maintains the list of aggregators
 */
contract AggregatorRegistry {
  address[] public aggregators;

  /**
   * @notice Add a new aggregator to this contract
   * @param _authenticationAddress The authenticationAddress for this aggregator
   * @return newly created aggregator
   */
  function addAggregator(address _authenticationAddress) public returns (Aggregator newAggregator) {
    uint id = aggregators.length + 1;
    Aggregator aggregator = new Aggregator(_authenticationAddress, id);
    aggregators.push(address(aggregator));
    return aggregator;
  }

  /**
   * @notice Get the current number of aggregators registered
   * @return number of aggregators
   */
  function getAggregatorCount() public view returns (uint256) {
    return aggregators.length;
  }
}
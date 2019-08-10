pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* External Imports */
import "openzeppelin-solidity/contracts/token/ERC20/ERC20.sol";

/* Internal Imports */
import { Aggregator } from "./Aggregator.sol";
import { AggregatorRegistry } from "./AggregatorRegistry.sol";

contract AggregatorWithIPCreationProxy {
  address payable public owner;
  AggregatorRegistry public aggregatorRegistry;

  constructor(AggregatorRegistry _aggregatorRegistry, address _authenticationAddress, string memory data) public {
    owner = msg.sender;
    aggregatorRegistry = _aggregatorRegistry;
    aggregatorRegistry.addAggregator(_authenticationAddress);
  }

  function deleteThisContract() public {
    selfdestruct(owner);
  }
}
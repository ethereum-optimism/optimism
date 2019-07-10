pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* External Imports */
import "openzeppelin-solidity/contracts/token/ERC20/ERC20.sol";

/* Internal Imports */
import { Aggregator } from "./Aggregator.sol";
import { PlasmaRegistry } from "./PlasmaRegistry.sol";

contract AggregatorWithIPCreationProxy {
  // Aggregator aggregator;
  PlasmaRegistry plasmaRegistry;

  // constructor() public {
  //   // aggregator = new Aggregator();
  //   plasmaRegistry = new PlasmaRegistry();
  // }

  constructor(address _authenticationAddress, string memory data) public {
    plasmaRegistry = new PlasmaRegistry();
    Aggregator newAggregator = plasmaRegistry.addAggregator(_authenticationAddress);
    newAggregator.setMetadata("ip", data);
  }
}
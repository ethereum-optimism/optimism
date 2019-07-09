pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* External Imports */
import "openzeppelin-solidity/contracts/token/ERC20/ERC20.sol";

/* Internal Imports */
import { Aggregator } from "./Aggregator.sol";

contract PlasmaRegistry {
  Aggregator aggregator;

  constructor() public {
    aggregator = new Aggregator();
  }
}
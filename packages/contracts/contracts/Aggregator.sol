pragma solidity ^0.5.1;

import "openzeppelin-solidity/contracts/token/ERC20/ERC20.sol";

/* Internal Imports */
import { CommitmentChain } from "./CommitmentChain.sol";
import { Deposit } from "./Deposit.sol";

contract Aggregator {
  address public authenticationAddress;
  CommitmentChain public commitmentContract;
  mapping(address => Deposit) depositContracts;
  uint public id;
  mapping(string => string) metadata;

  constructor() public {
  }
}
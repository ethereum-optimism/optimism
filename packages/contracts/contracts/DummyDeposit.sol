pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* External Imports */
import "openzeppelin-solidity/contracts/token/ERC20/ERC20.sol";

/* Internal Imports */
import {CommitmentChain} from "./CommitmentChain.sol";

contract DummyDeposit {
  ERC20 public erc20;
  CommitmentChain public commitmentChain;

  constructor(address _erc20, address _commitmentChain) public {
      erc20 = ERC20(_erc20);
      commitmentChain = CommitmentChain(_commitmentChain);
  }
}
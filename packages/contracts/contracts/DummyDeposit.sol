pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* External Imports */
import "openzeppelin-solidity/contracts/token/ERC20/ERC20.sol";

/* Internal Imports */
import { CommitmentChain } from "./CommitmentChain.sol";


contract DummyDeposit {
  ERC20 public erc20;
  CommitmentChain public commitmentContract;

  constructor(address _erc20, address _commitmentContract) public {
    erc20 = ERC20(_erc20);
    commitmentContract = CommitmentChain(_commitmentContract);
  }
}

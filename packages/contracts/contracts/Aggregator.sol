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
  mapping(string => string) public metadata;

  constructor(address _authenticationAddress, uint _id) public {
    authenticationAddress = _authenticationAddress;
    commitmentContract = new CommitmentChain();
    id = _id;
  }

  /**
    Causes RuntimeError: VM Exception while processing transaction: revert
   */
  // function addDepositContract(ERC20 _erc20, address _walletAddress) public returns (Deposit newDepositContract) {
  //   require(msg.sender == authenticationAddress, "addDepositContract can only be called by authenticated address.");
  //   // Deposit depositContract = new Deposit(address(_erc20), address(commitmentContract));
  //   Deposit depositContract = new Deposit();
  //   depositContracts[_walletAddress] = depositContract;
  //   return depositContract;
  // }

  function setMetadata(string memory _ip, string memory _data) public {
    // require(msg.sender == authenticationAddress, "setMetadata can only be called by authenticated address.");
    metadata[_ip] = _data;
  }

  function deleteMetadata(string memory _ip) public {
    require(msg.sender == authenticationAddress, "deleteMetadata can only be called by authenticated address.");
    delete metadata[_ip];
  }
}
pragma solidity ^0.5.1;

import "openzeppelin-solidity/contracts/token/ERC20/ERC20.sol";

/* Internal Imports */
import { CommitmentChain } from "./CommitmentChain.sol";
import {DataTypes as types} from "./DataTypes.sol";
// import { Deposit } from "./Deposit.sol";
import { DummyDeposit } from "./DummyDeposit.sol";

contract Aggregator {
  address public authenticationAddress;
  CommitmentChain public commitmentContract;
  mapping(address => DummyDeposit) public depositContracts;
    // mapping(address => Deposit) public depositContracts;
  uint public id;
  mapping(string => string) public metadata;

  constructor(address _authenticationAddress, uint _id) public {
    authenticationAddress = _authenticationAddress;
    commitmentContract = new CommitmentChain();
    id = _id;
  }

  // function addDepositContract(address _erc20, address _commitmentContract) public returns (Deposit newDepositContract) {
  function addDepositContract(address _erc20, address _commitmentContract) public returns (DummyDeposit newDepositContract) {
    require(msg.sender == authenticationAddress, "addDepositContract can only be called by authenticated address.");
    // Deposit depositContract = new Deposit(_erc20, _commitmentContract);
    DummyDeposit depositContract = new DummyDeposit(_erc20, _commitmentContract);
    depositContracts[address(depositContract)] = depositContract;
    return depositContract;
  }

  function setMetadata(string memory _ip, string memory _data) public {
    // require(msg.sender == authenticationAddress, "setMetadata can only be called by authenticated address.");
    metadata[_ip] = _data;
  }

  function deleteMetadata(string memory _ip) public {
    require(msg.sender == authenticationAddress, "deleteMetadata can only be called by authenticated address.");
    delete metadata[_ip];
  }
}
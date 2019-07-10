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
    id = _id;
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
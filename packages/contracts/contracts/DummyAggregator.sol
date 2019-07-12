pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

contract DummyAggregator {
  address authenticationAddress;
  uint id;

  constructor (address _authenticationAddress, uint _id) public {
    authenticationAddress = _authenticationAddress;
    id = _id;
  }
}
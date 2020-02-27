pragma solidity ^0.5.0;

contract Dummy {
  bytes32 someState;

  constructor() public {
    someState = keccak256("Wooooooo!");
  }

  function getState() public view returns(bytes32) {
    return someState;
  }

  function updateState(bytes32 newValue) public {
    someState = newValue;
  }
}
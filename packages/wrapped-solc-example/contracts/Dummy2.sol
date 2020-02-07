pragma solidity ^0.5.0;

contract Dummy2 {
  bytes32 someState;

  constructor() public {
    someState = keccak256("Derp Derp Derp!");
  }

  function getState() public view returns(bytes32) {
    return someState;
  }

  function updateState(bytes32 newValue) public {
    someState = newValue;
  }
}

contract Dummy3 {
  bytes32 someState;

  constructor() public {
    someState = keccak256("!!!!!!!!!!!");
  }

  function getState() public view returns(bytes32) {
    return someState;
  }

  function updateState(bytes32 newValue) public {
    someState = newValue;
  }
}
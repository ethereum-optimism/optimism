pragma solidity ^0.5.0;

contract Counter {
  event Increment (uint256 by);

  uint256 public value;

  constructor (uint256 initialValue) public {
    value = initialValue;
  }

  function increment (uint256 by) public {
    // NOTE: You should use SafeMath in production code
    value += by;
    emit Increment(by);
  }
  
  function getCount() public returns(uint256) {
    return value;
  }
}
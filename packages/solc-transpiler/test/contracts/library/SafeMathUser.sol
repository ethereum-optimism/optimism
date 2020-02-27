pragma solidity ^0.5.0;

import {SimpleSafeMath} from  './SimpleSafeMath.sol';

contract SafeMathUser {
  function use() public pure returns (uint) {
    return SimpleSafeMath.addUint(2, 3);
  }
  function use2() public pure returns(uint) {
    return SimpleSafeMath.addUint(16, 18);
  }
}

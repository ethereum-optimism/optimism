pragma solidity ^0.5.0;

import {SimpleSafeMath} from  './SimpleSafeMath.sol';

contract SafeMathUser {
  function use() public returns (uint) {
    return SimpleSafeMath.addUint(2, 3);
  }
}

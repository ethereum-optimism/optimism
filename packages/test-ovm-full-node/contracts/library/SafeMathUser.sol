pragma solidity ^0.5.0;

import {SimpleSafeMath} from  './SimpleSafeMath.sol';
import {SimpleUnsafeMath} from  './SimpleUnsafeMath.sol';

contract SafeMathUser {
  function useLib() public pure returns (uint) {
    return SimpleSafeMath.addUint(2, 3);
  }
  function use2Libs() public pure returns(uint) {
    return SimpleUnsafeMath.addUint(SimpleSafeMath.addUint(1, 2), 3);
  }
}

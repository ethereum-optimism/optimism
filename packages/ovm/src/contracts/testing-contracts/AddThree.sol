pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

contract AddThree {
   function addThree(uint256 a) public pure returns (uint256) {
     return a + 3;
      }
}
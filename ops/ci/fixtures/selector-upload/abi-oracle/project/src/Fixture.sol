// SPDX-License-Identifier: MIT
pragma solidity 0.8.28;
library Empty {}
contract Fixture { struct Pair { uint256 x; address recipient; }
error Problem(uint256 value); event Pong(uint256 value);
function ping(uint256 value) external pure returns (uint256) { if (value==0) revert Problem(value); return value; }
function transform(Pair calldata pair) external pure returns (uint256) { return pair.x; } }

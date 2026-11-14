// SPDX-License-Identifier: MIT
pragma solidity 0.8.28;
library Empty {}
contract Fixture { struct Pair { uint256 x; address recipient; }
error Problem(uint256 value); event Pong(uint256 value);
function ping(uint256 _value) external pure returns (uint256) { if (_value==0) revert Problem(_value); return _value; }
function transform(Pair calldata _pair) external pure returns (uint256) { return _pair.x; } }

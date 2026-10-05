// SPDX-License-Identifier: MIT
pragma solidity 0.8.28;
import {Shared} from "./Shared.sol";
contract ZRestricted { function z(uint256 x) external pure returns (uint256) { return Shared.value(x); } }

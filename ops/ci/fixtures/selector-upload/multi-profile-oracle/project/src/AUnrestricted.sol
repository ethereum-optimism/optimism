// SPDX-License-Identifier: MIT
pragma solidity 0.8.28;
import {Shared} from "./Shared.sol";
contract AUnrestricted { function a(uint256 x) external pure returns (uint256) { return Shared.value(x); } }

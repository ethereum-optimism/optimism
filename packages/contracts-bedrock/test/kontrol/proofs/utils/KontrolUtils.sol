// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Vm } from "forge-std/Vm.sol";
import { KontrolCheats } from "kontrol-cheatcodes/KontrolCheats.sol";

/// @notice Tests inheriting this contract cannot be run with forge
abstract contract KontrolUtils is KontrolCheats {
    Vm internal constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));
}

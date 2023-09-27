// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Target contract
import { Storage } from "src/libraries/Storage.sol";
import { Test } from "forge-std/Test.sol";

contract Slot_Getters_Test is Test {
    function test_setGetUint_succeeds(bytes32 slot, uint256 num) external {
        Storage.setUint(slot, num);
        assertEq(Storage.getUint(slot), num);
        assertEq(num, uint256(vm.load(address(this), slot)));
    }

    function test_setGetAddress_succeeds(bytes32 slot, address addr) external {
        Storage.setAddress(slot, addr);
        assertEq(Storage.getAddress(slot), addr);
        assertEq(addr, address(uint160(uint256(vm.load(address(this), slot)))));
    }

    function test_setGetBytes32_succeeds(bytes32 slot, bytes32 hash) external {
        Storage.setBytes32(slot, hash);
        assertEq(Storage.getBytes32(slot), hash);
        assertEq(hash, vm.load(address(this), slot));
    }
}

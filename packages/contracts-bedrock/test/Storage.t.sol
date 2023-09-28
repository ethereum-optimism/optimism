// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Target contract
import { Storage } from "src/libraries/Storage.sol";
import { Test } from "forge-std/Test.sol";
import { console } from "forge-std/console.sol";

/// @title StorageWrapper
/// @notice StorageWrapper wraps the Storage library for testing purposes.
///         It exists to prevent storage collisions with the `Test` contract.
contract StorageWrapper {
    function getAddress(bytes32 _slot) external view returns (address) {
        return Storage.getAddress(_slot);
    }

    function setAddress(bytes32 _slot, address _address) external {
        Storage.setAddress(_slot, _address);
    }

    function getUint(bytes32 _slot) external view returns (uint256) {
        return Storage.getUint(_slot);
    }

    function setUint(bytes32 _slot, uint256 _value) external {
        Storage.setUint(_slot, _value);
    }

    function getBytes32(bytes32 _slot) external view returns (bytes32) {
        return Storage.getBytes32(_slot);
    }

    function setBytes32(bytes32 _slot, bytes32 _value) external {
        Storage.setBytes32(_slot, _value);
    }
}

contract Storage_Roundtrip_Test is Test {
    StorageWrapper wrapper;

    function setUp() external {
        wrapper = new StorageWrapper();
    }

    function test_setGetUint_succeeds(bytes32 slot, uint256 num) external {
        wrapper.setUint(slot, num);
        assertEq(wrapper.getUint(slot), num);
        assertEq(num, uint256(vm.load(address(wrapper), slot)));
    }

    function test_setGetAddress_succeeds(bytes32 slot, address addr) external {
        wrapper.setAddress(slot, addr);
        assertEq(wrapper.getAddress(slot), addr);
        assertEq(addr, address(uint160(uint256(vm.load(address(wrapper), slot)))));
    }

    function test_setGetBytes32_succeeds(bytes32 slot, bytes32 hash) external {
        wrapper.setBytes32(slot, hash);
        assertEq(wrapper.getBytes32(slot), hash);
        assertEq(hash, vm.load(address(wrapper), slot));
    }
}

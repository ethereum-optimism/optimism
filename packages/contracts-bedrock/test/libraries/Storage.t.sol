// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Target contract
import { StorageSetter } from "src/universal/StorageSetter.sol";
import { Test } from "forge-std/Test.sol";

/// @title Storage_TestInit
/// @notice Reusable test initialization for `Storage` tests.
abstract contract Storage_TestInit is Test {
    StorageSetter setter;

    function setUp() public {
        setter = new StorageSetter();
    }
}

/// @title Storage_GetAddress_Test
/// @notice Tests the `getAddress` function of the `Storage` library.
contract Storage_GetAddress_Test is Storage_TestInit {
    /// @notice Test that getAddress returns the correct address value from storage.
    /// @param _slot The storage slot to test with.
    /// @param _addr The address value to test with.
    function testFuzz_getAddress_succeeds(bytes32 _slot, address _addr) external {
        setter.setAddress(_slot, _addr);
        assertEq(setter.getAddress(_slot), _addr);
        assertEq(_addr, address(uint160(uint256(vm.load(address(setter), _slot)))));
    }
}

/// @title Storage_SetAddress_Test
/// @notice Tests the `setAddress` function of the `Storage` library.
contract Storage_SetAddress_Test is Storage_TestInit {
    /// @notice Test that setAddress correctly stores address values in arbitrary slots.
    /// @param _slot The storage slot to test with.
    /// @param _addr The address value to test with.
    function testFuzz_setAddress_succeeds(bytes32 _slot, address _addr) external {
        setter.setAddress(_slot, _addr);
        assertEq(address(uint160(uint256(vm.load(address(setter), _slot)))), _addr);
    }
}

/// @title Storage_GetUint_Test
/// @notice Tests the `getUint` function of the `Storage` library.
contract Storage_GetUint_Test is Storage_TestInit {
    /// @notice Test that getUint returns the correct uint256 value from storage.
    /// @param _slot The storage slot to test with.
    /// @param _value The uint256 value to test with.
    function testFuzz_getUint_succeeds(bytes32 _slot, uint256 _value) external {
        setter.setUint(_slot, _value);
        assertEq(setter.getUint(_slot), _value);
        assertEq(_value, uint256(vm.load(address(setter), _slot)));
    }
}

/// @title Storage_SetUint_Test
/// @notice Tests the `setUint` function of the `Storage` library.
contract Storage_SetUint_Test is Storage_TestInit {
    /// @notice Test that setUint correctly stores uint256 values in arbitrary slots.
    /// @param _slot The storage slot to test with.
    /// @param _value The uint256 value to test with.
    function testFuzz_setUint_succeeds(bytes32 _slot, uint256 _value) external {
        setter.setUint(_slot, _value);
        assertEq(uint256(vm.load(address(setter), _slot)), _value);
    }
}

/// @title Storage_GetBytes32_Test
/// @notice Tests the `getBytes32` function of the `Storage` library.
contract Storage_GetBytes32_Test is Storage_TestInit {
    /// @notice A set of storage slots to pass to `setBytes32`.
    StorageSetter.Slot[] slots;
    /// @notice Used to deduplicate slots passed to `setBytes32`.
    mapping(bytes32 => bool) keys;

    /// @notice Test that getBytes32 returns the correct bytes32 value from storage.
    /// @param _slot The storage slot to test with.
    /// @param _value The bytes32 value to test with.
    function testFuzz_getBytes32_succeeds(bytes32 _slot, bytes32 _value) external {
        setter.setBytes32(_slot, _value);
        assertEq(setter.getBytes32(_slot), _value);
        assertEq(_value, vm.load(address(setter), _slot));
    }

    /// @notice Test that multiple bytes32 values can be set and retrieved correctly.
    /// @param _slots Array of storage slots and values to test with.
    function testFuzz_getBytes32_multiSlot_succeeds(StorageSetter.Slot[] calldata _slots) external {
        for (uint256 i; i < _slots.length; i++) {
            if (keys[_slots[i].key]) {
                continue;
            }
            slots.push(_slots[i]);
            keys[_slots[i].key] = true;
        }

        setter.setBytes32(slots);
        for (uint256 i; i < slots.length; i++) {
            assertEq(setter.getBytes32(slots[i].key), slots[i].value);
            assertEq(slots[i].value, vm.load(address(setter), slots[i].key));
        }
    }
}

/// @title Storage_SetBytes32_Test
/// @notice Tests the `setBytes32` function of the `Storage` library.
contract Storage_SetBytes32_Test is Storage_TestInit {
    /// @notice Test that setBytes32 correctly stores bytes32 values in arbitrary slots.
    /// @param _slot The storage slot to test with.
    /// @param _value The bytes32 value to test with.
    function testFuzz_setBytes32_succeeds(bytes32 _slot, bytes32 _value) external {
        setter.setBytes32(_slot, _value);
        assertEq(vm.load(address(setter), _slot), _value);
    }
}

/// @title Storage_SetBool_Test
/// @notice Tests the `setBool` function of the `Storage` library.
contract Storage_SetBool_Test is Storage_TestInit {
    /// @notice Test that setBool correctly stores bool values in arbitrary slots.
    /// @param _slot The storage slot to test with.
    /// @param _value The bool value to test with.
    function testFuzz_setBool_succeeds(bytes32 _slot, bool _value) external {
        setter.setBool(_slot, _value);
        assertEq(vm.load(address(setter), _slot) == bytes32(uint256(1)), _value);
    }
}

/// @title Storage_GetBool_Test
/// @notice Tests the `getBool` function of the `Storage` library.
contract Storage_GetBool_Test is Storage_TestInit {
    /// @notice Test that getBool returns the correct bool value from storage.
    /// @param _slot The storage slot to test with.
    /// @param _value The bool value to test with.
    function testFuzz_getBool_succeeds(bytes32 _slot, bool _value) external {
        setter.setBool(_slot, _value);
        assertEq(setter.getBool(_slot), _value);
        assertEq(_value, vm.load(address(setter), _slot) == bytes32(uint256(1)));
    }
}

// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Target contract
import { Storage } from "src/libraries/Storage.sol";
import { StorageSetter } from "src/universal/StorageSetter.sol";
import { Test } from "forge-std/Test.sol";

/// @title Storage_Roundtrip_Test
/// @notice Tests the storage setting and getting through the StorageSetter contract.
///         This contract simply wraps the Storage library, this is required as to
///         not poison the storage of the `Test` contract.
contract Storage_Roundtrip_Test is Test {
    StorageSetter setter;

    function setUp() external {
        setter = new StorageSetter();
    }

    function test_setGetUint_succeeds(bytes32 slot, uint256 num) external {
        setter.setUint(slot, num);
        assertEq(setter.getUint(slot), num);
        assertEq(num, uint256(vm.load(address(setter), slot)));
    }

    function test_setGetAddress_succeeds(bytes32 slot, address addr) external {
        setter.setAddress(slot, addr);
        assertEq(setter.getAddress(slot), addr);
        assertEq(addr, address(uint160(uint256(vm.load(address(setter), slot)))));
    }

    function test_setGetBytes32_succeeds(bytes32 slot, bytes32 hash) external {
        setter.setBytes32(slot, hash);
        assertEq(setter.getBytes32(slot), hash);
        assertEq(hash, vm.load(address(setter), slot));
    }
}

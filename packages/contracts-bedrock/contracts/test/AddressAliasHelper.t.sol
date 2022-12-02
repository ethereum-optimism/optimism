// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { AddressAliasHelper } from "../vendor/AddressAliasHelper.sol";

contract AddressAliasHelper_Test is Test {
    function testFuzz_roundtrip_succeeds(address _address) external {
        address aliased = AddressAliasHelper.applyL1ToL2Alias(_address);
        address unaliased = AddressAliasHelper.undoL1ToL2Alias(aliased);
        assertEq(_address, unaliased);
    }
}

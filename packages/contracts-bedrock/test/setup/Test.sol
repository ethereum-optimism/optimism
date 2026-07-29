// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Forge
import { Test as ForgeTest } from "forge-std/Test.sol";

/// @title Test
/// @notice Test is a minimal extension of the Test contract with op-specific tweaks.
abstract contract Test is ForgeTest {
    /// @notice Makes an address without a private key, labels it, and cleans it.
    /// @param _name The name of the address.
    /// @return The address.
    function makeAddr(string memory _name) internal virtual override returns (address) {
        address addr = address(uint160(uint256(keccak256(abi.encode(_name)))));
        destroyAccount(addr, address(0));
        vm.label(addr, _name);
        return addr;
    }
}

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

    /// @notice Also excludes the P256VERIFY precompile (EIP-7951) at 0x100, which forge-std's
    ///         0x01-0xff range misses.
    /// @param _addr The address to check.
    /// @param _chainId The chain whose precompiles to exclude.
    function assumeNotPrecompile(address _addr, uint256 _chainId) internal pure virtual override {
        super.assumeNotPrecompile(_addr, _chainId);
        vm.assume(_addr != address(0x100));
    }

    /// @notice Counts calldata tokens as EIP-7623 defines them: one per zero byte, four per
    ///         non-zero byte.
    /// @param _data The calldata to count.
    /// @return tokens_ The token count.
    function calldataTokens(bytes memory _data) internal pure returns (uint256 tokens_) {
        for (uint256 i = 0; i < _data.length; i++) {
            tokens_ += _data[i] == bytes1(0) ? 1 : 4;
        }
    }
}

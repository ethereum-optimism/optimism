// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Libraries
import { PortalAllowlist } from "src/libraries/PortalAllowlist.sol";

/// @title PortalAllowlist_TestInit
/// @notice Shared helpers for the `PortalAllowlist` tests.
abstract contract PortalAllowlist_TestInit is Test {
    /// @notice Returns whether the chain carries an allowlist.
    function _isRestricted(uint256 _chainId) internal pure returns (bool) {
        return _chainId == 1 || _chainId == 11155111;
    }

    /// @notice Picks Mainnet or Sepolia from a fuzzed boolean.
    function _restrictedChainId(bool _mainnet) internal pure returns (uint256) {
        return _mainnet ? 1 : 11155111;
    }
}

/// @title PortalAllowlist_AllowedAddresses_Test
/// @notice Tests the `allowedAddresses` function of the `PortalAllowlist` library.
contract PortalAllowlist_AllowedAddresses_Test is PortalAllowlist_TestInit {
    /// @notice Tests that Mainnet carries the two allowed addresses.
    function test_allowedAddresses_mainnet_succeeds() external {
        vm.chainId(1);
        address[] memory allowed = PortalAllowlist.allowedAddresses();
        assertEq(allowed.length, 2);
        assertEq(allowed[0], 0x6307ed2483af904D67cc7EB74d5Bf97BfE3152be);
        assertEq(allowed[1], 0x93Cd74489167f9885f2D7B5112f9e19224050220);
    }

    /// @notice Tests that Sepolia carries the four allowed addresses.
    function test_allowedAddresses_sepolia_succeeds() external {
        vm.chainId(11155111);
        address[] memory allowed = PortalAllowlist.allowedAddresses();
        assertEq(allowed.length, 4);
        assertEq(allowed[0], 0xE5327a59e33CAB54AcFFFBEba6bb167AaEDCD953);
        assertEq(allowed[1], 0x2EA11DcDAEE45D82aAe2Ad6e7255f8d5bb70AbFF);
        assertEq(allowed[2], 0x7d9F14FCd14fdf7A47BC2657949e087aA24f62bA);
        assertEq(allowed[3], 0xcE38EA73Aac620E9aF3d45b53088B9E7294C5a2C);
    }

    /// @notice Tests that every other chain has no list.
    function testFuzz_allowedAddresses_otherChain_succeeds(uint64 _chainId) external {
        vm.assume(!_isRestricted(_chainId));
        vm.chainId(_chainId);
        assertEq(PortalAllowlist.allowedAddresses().length, 0);
    }
}

/// @title PortalAllowlist_IsAllowed_Test
/// @notice Tests the `isAllowed` function of the `PortalAllowlist` library.
contract PortalAllowlist_IsAllowed_Test is PortalAllowlist_TestInit {
    /// @notice Tests that every listed address is allowed on Mainnet and Sepolia.
    function testFuzz_isAllowed_listed_succeeds(bool _mainnet) external {
        vm.chainId(_restrictedChainId(_mainnet));
        address[] memory allowed = PortalAllowlist.allowedAddresses();
        for (uint256 i = 0; i < allowed.length; i++) {
            assertTrue(PortalAllowlist.isAllowed(allowed[i]));
        }
    }

    /// @notice Tests that an unlisted address is not allowed on Mainnet or Sepolia.
    function testFuzz_isAllowed_unlisted_fails(bool _mainnet, address _address) external {
        vm.chainId(_restrictedChainId(_mainnet));
        address[] memory allowed = PortalAllowlist.allowedAddresses();
        for (uint256 i = 0; i < allowed.length; i++) {
            vm.assume(_address != allowed[i]);
        }
        assertFalse(PortalAllowlist.isAllowed(_address));
    }

    /// @notice Tests that any address is allowed on a chain without a list.
    function testFuzz_isAllowed_otherChain_succeeds(uint64 _chainId, address _address) external {
        vm.assume(!_isRestricted(_chainId));
        vm.chainId(_chainId);
        assertTrue(PortalAllowlist.isAllowed(_address));
    }
}

// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Libraries
import { PortalAllowlist } from "src/libraries/PortalAllowlist.sol";

/// @title PortalAllowlist_AllowedAddresses_Test
/// @notice Tests the `allowedAddresses` function of the `PortalAllowlist` library.
contract PortalAllowlist_AllowedAddresses_Test is Test {
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
        vm.assume(_chainId != 11155111);
        vm.chainId(_chainId);
        assertEq(PortalAllowlist.allowedAddresses().length, 0);
    }
}

/// @title PortalAllowlist_IsAllowed_Test
/// @notice Tests the `isAllowed` function of the `PortalAllowlist` library.
contract PortalAllowlist_IsAllowed_Test is Test {
    /// @notice Tests that every listed address is allowed on Sepolia.
    function test_isAllowed_sepoliaListed_succeeds() external {
        vm.chainId(11155111);
        address[] memory allowed = PortalAllowlist.allowedAddresses();
        for (uint256 i = 0; i < allowed.length; i++) {
            assertTrue(PortalAllowlist.isAllowed(allowed[i]));
        }
    }

    /// @notice Tests that an unlisted address is not allowed on Sepolia.
    function testFuzz_isAllowed_sepoliaUnlisted_fails(address _address) external {
        vm.chainId(11155111);
        address[] memory allowed = PortalAllowlist.allowedAddresses();
        for (uint256 i = 0; i < allowed.length; i++) {
            vm.assume(_address != allowed[i]);
        }
        assertFalse(PortalAllowlist.isAllowed(_address));
    }

    /// @notice Tests that any address is allowed on a chain without a list.
    function testFuzz_isAllowed_otherChain_succeeds(uint64 _chainId, address _address) external {
        vm.assume(_chainId != 11155111);
        vm.chainId(_chainId);
        assertTrue(PortalAllowlist.isAllowed(_address));
    }
}

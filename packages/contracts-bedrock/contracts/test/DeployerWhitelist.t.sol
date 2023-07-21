// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "./CommonTest.t.sol";

// Target contract
import { DeployerWhitelist } from "../legacy/DeployerWhitelist.sol";

contract DeployerWhitelist_Test is CommonTest {
    DeployerWhitelist list;

    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        list = new DeployerWhitelist();
    }

    /// @dev Tests that `owner` is initialized to the zero address.
    function test_owner_succeeds() external {
        assertEq(list.owner(), address(0));
    }

    /// @dev Tests that `setOwner` correctly sets the contract owner.
    function test_storageSlots_succeeds() external {
        vm.prank(list.owner());
        list.setOwner(address(1));

        assertEq(bytes32(uint256(1)), vm.load(address(list), bytes32(uint256(0))));
    }
}

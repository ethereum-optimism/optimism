// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Target contract dependencies
import { Predeploys } from "src/libraries/Predeploys.sol";

// Target contract
import { LegacyERC20ETH } from "src/legacy/LegacyERC20ETH.sol";

contract LegacyERC20ETH_Test is CommonTest {
    LegacyERC20ETH eth;

    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();
        eth = new LegacyERC20ETH();
    }

    /// @dev Tests that the default metadata was set correctly.
    function test_metadata_succeeds() external {
        assertEq(eth.name(), "Ether");
        assertEq(eth.symbol(), "ETH");
        assertEq(eth.decimals(), 18);
    }

    /// @dev Tests that `l2Bridge` and `l1Token` return the correct values.
    function test_crossDomain_succeeds() external {
        assertEq(eth.l2Bridge(), Predeploys.L2_STANDARD_BRIDGE);
        assertEq(eth.l1Token(), address(0));
    }

    /// @dev Tests that `transfer` reverts since it does not exist.
    function test_transfer_doesNotExist_reverts() external {
        vm.expectRevert("LegacyERC20ETH: transfer is disabled");
        eth.transfer(alice, 100);
    }

    /// @dev Tests that `approve` reverts since it does not exist.
    function test_approve_doesNotExist_reverts() external {
        vm.expectRevert("LegacyERC20ETH: approve is disabled");
        eth.approve(alice, 100);
    }

    /// @dev Tests that `transferFrom` reverts since it does not exist.
    function test_transferFrom_doesNotExist_reverts() external {
        vm.expectRevert("LegacyERC20ETH: transferFrom is disabled");
        eth.transferFrom(bob, alice, 100);
    }

    /// @dev Tests that `increaseAllowance` reverts since it does not exist.
    function test_increaseAllowance_doesNotExist_reverts() external {
        vm.expectRevert("LegacyERC20ETH: increaseAllowance is disabled");
        eth.increaseAllowance(alice, 100);
    }

    /// @dev Tests that `decreaseAllowance` reverts since it does not exist.
    function test_decreaseAllowance_doesNotExist_reverts() external {
        vm.expectRevert("LegacyERC20ETH: decreaseAllowance is disabled");
        eth.decreaseAllowance(alice, 100);
    }

    /// @dev Tests that `mint` reverts since it does not exist.
    function test_mint_doesNotExist_reverts() external {
        vm.expectRevert("LegacyERC20ETH: mint is disabled");
        eth.mint(alice, 100);
    }

    /// @dev Tests that `burn` reverts since it does not exist.
    function test_burn_doesNotExist_reverts() external {
        vm.expectRevert("LegacyERC20ETH: burn is disabled");
        eth.burn(alice, 100);
    }
}

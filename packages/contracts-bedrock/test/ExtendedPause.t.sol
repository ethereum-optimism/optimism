// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "test/setup/CommonTest.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";

contract ExtendedPause_Test is CommonTest {
    /// @dev Tests that other contracts are paused when the superchain config is paused
    ///      This test is somewhat redundant with tests in the SuperchainConfig and other pausable contracts, however
    ///      it is worthwhile to pull them into one location to ensure that the behavior is consistent.
    function test_pause_fullSystem_succeeds() external {
        assertFalse(superchainConfig.paused());
        assertEq(l1CrossDomainMessenger.paused(), superchainConfig.paused());

        vm.prank(superchainConfig.guardian());
        superchainConfig.pause("identifier");

        assertTrue(superchainConfig.paused());
        assertEq(l1CrossDomainMessenger.paused(), superchainConfig.paused());

        assertTrue(l1StandardBridge.paused());
        assertEq(l1StandardBridge.paused(), superchainConfig.paused());

        //assertTrue(l1ERC721Bridge.paused());
        //assertEq(l1ERC721Bridge.paused(), superchainConfig.paused());
    }
}

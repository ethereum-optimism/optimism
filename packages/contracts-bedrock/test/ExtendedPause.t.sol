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

        // The following is hacky approach which ensures that this test will fail if the paused() function is
        // added to the L1StandardBridge or the L1ERC721Bridge. At that point this test should be updated to include
        // those methods.
        try SuperchainConfig(address(l1StandardBridge)).paused() {
            revert("The L1StandardBridge has a paused() function, but is not tested as part of the ExtendedPause");
        } catch (bytes memory) {
            assertTrue(true);
        }
        try SuperchainConfig(address(l1ERC721Bridge)).paused() {
            revert("The L1ERC721Bridge has a paused() function, but is not tested as part of the ExtendedPause");
        } catch (bytes memory) {
            assertTrue(true);
        }
    }
}

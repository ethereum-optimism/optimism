// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { SuperchainConfig_Initializer } from "./CommonTest.t.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { Types } from "src/libraries/Types.sol";
import { Hashing } from "src/libraries/Hashing.sol";

// Target contract dependencies
import { Proxy } from "src/universal/Proxy.sol";

// Target contract
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";

contract SuperchainConfig_Init_Test is SuperchainConfig_Initializer {
    /// @dev Tests that initialization sets the correct values. These are defined in CommonTest.sol.
    function test_initialize_values_succeeds() external {
        assertFalse(supConf.paused());
        assertEq(supConf.guardian(), guardian);
    }
}

contract SuperchainConfig_Pause_TestFail is SuperchainConfig_Initializer {
    /// @dev Tests that `pause` reverts when called by a non-guardian.
    function test_pause_notGuardian_reverts() external {
        assertFalse(supConf.paused());

        assertTrue(supConf.guardian() != alice);
        vm.expectRevert("SuperchainConfig: only guardian can pause");
        vm.prank(alice);
        supConf.pause("identifier");

        assertFalse(supConf.paused());
    }
}

contract SuperchainConfig_Pause_Test is SuperchainConfig_Initializer {
    /// @dev Tests that `pause` successfully pauses
    ///      when called by the guardian.
    function test_pause_succeeds() external {
        assertFalse(supConf.paused());

        vm.expectEmit(address(supConf));
        emit Paused("identifier");

        vm.prank(guardian);
        supConf.pause("identifier");

        assertTrue(supConf.paused());
    }
}

contract SuperchainConfig_Unpause_TestFail is SuperchainConfig_Initializer {
    /// @dev Tests that `unpause` reverts when called by a non-guardian.
    function test_unpause_notGuardian_reverts() external {
        _pause();

        assertTrue(supConf.guardian() != alice);
        vm.expectRevert("SuperchainConfig: only guardian can unpause");
        vm.prank(alice);
        supConf.unpause();

        assertTrue(supConf.paused());
    }
}

contract SuperchainConfig_Unpause_Test is SuperchainConfig_Initializer {
    /// @dev Tests that `unpause` successfully unpauses
    ///      when called by the guardian.
    function test_unpause_succeeds() external {
        _pause();

        vm.expectEmit(address(supConf));
        emit Unpaused();
        vm.prank(guardian);
        supConf.unpause();

        assertFalse(supConf.paused());
    }
}

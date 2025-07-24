// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { CheckTrue } from "src/periphery/drippie/dripchecks/CheckTrue.sol";

/// @title CheckTrue_TestInit
/// @notice Reusable test initialization for `CheckTrue` tests.
contract CheckTrue_TestInit is Test {
    /// @notice An instance of the CheckTrue contract.
    CheckTrue c;

    /// @notice Deploy the `CheckTrue` contract.
    function setUp() external {
        c = new CheckTrue();
    }
}

/// @title CheckTrue_Check_Test
/// @notice Tests the `check` function of the `CheckTrue` contract.
contract CheckTrue_Check_Test is CheckTrue_TestInit {
    /// @notice Fuzz the `check` function and assert that it always returns true.
    function testFuzz_check_alwaysTrue_succeeds(bytes memory input) external view {
        assertEq(c.check(input), true);
    }
}

/// @title CheckTrue_Unclassified_Test
/// @notice General tests that are not testing any function directly of the `CheckTrue` contract or
///         are testing multiple functions at once.
contract CheckTrue_Unclassified_Test is CheckTrue_TestInit {
    /// @notice Test that the `name` function returns the correct value.
    function test_name_succeeds() external view {
        assertEq(c.name(), "CheckTrue");
    }
}

// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { CheckBalanceLow } from "src/periphery/drippie/dripchecks/CheckBalanceLow.sol";

/// @title CheckBalanceLow_TestInit
/// @notice Reusable test initialization for `CheckBalanceLow` tests.
abstract contract CheckBalanceLow_TestInit is Test {
    /// @notice An instance of the CheckBalanceLow contract.
    CheckBalanceLow c;

    /// @notice Deploy the `CheckBalanceLow` contract.
    function setUp() external {
        c = new CheckBalanceLow();
    }
}

/// @title CheckBalanceLow_Check_Test
/// @notice Tests the `check` function of the `CheckBalanceLow` contract.
contract CheckBalanceLow_Check_Test is CheckBalanceLow_TestInit {
    /// @notice Fuzz the `check` function and assert that it always returns true when the target's
    ///         balance is smaller than the threshold.
    function testFuzz_check_succeeds(address _target, uint256 _threshold) external {
        CheckBalanceLow.Params memory p = CheckBalanceLow.Params({ target: _target, threshold: _threshold });

        if (_target.balance >= p.threshold) {
            if (_threshold == 0) p.threshold = 1;
            vm.deal(_target, p.threshold - 1);
        }

        assertEq(c.check(abi.encode(p)), true);
    }

    /// @notice Fuzz the `check` function and assert that it always returns false when the target's
    ///         balance is larger than the threshold.
    function testFuzz_check_highBalance_fails(address _target, uint256 _threshold) external {
        CheckBalanceLow.Params memory p = CheckBalanceLow.Params({ target: _target, threshold: _threshold });

        // prevent overflows
        _threshold = bound(_threshold, 0, type(uint256).max - 1);
        vm.deal(_target, _threshold + 1);

        assertEq(c.check(abi.encode(p)), false);
    }
}

/// @title CheckBalanceLow_Uncategorized_Test
/// @notice General tests that are not testing any function directly of the `CheckBalanceLow`
///         contract or are testing multiple functions at once.
contract CheckBalanceLow_Uncategorized_Test is CheckBalanceLow_TestInit {
    /// @notice Test that the `name` function returns the correct value.
    function test_name_succeeds() external view {
        assertEq(c.name(), "CheckBalanceLow");
    }
}

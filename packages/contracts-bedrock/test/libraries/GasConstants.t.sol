// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Libraries
import "src/libraries/GasConstants.sol" as GasConstants;

/// @title GasConstants_Uncategorized_Test
/// @notice Pins the relay gas budgets in `GasConstants.sol`.
contract GasConstants_Uncategorized_Test is Test {
    /// @notice Tests the L1 relay gas budgets.
    function test_l1RelayGas_values_succeeds() external pure {
        assertEq(GasConstants.L1_RELAY_CONSTANT_OVERHEAD, 200_000);
        assertEq(GasConstants.L1_RELAY_CALL_OVERHEAD, 40_000);
        assertEq(GasConstants.L1_RELAY_RESERVED_GAS, 135_000);
        assertEq(GasConstants.L1_RELAY_GAS_CHECK_BUFFER, 5_000);
    }

    /// @notice Tests the L2 relay gas budgets.
    function test_l2RelayGas_values_succeeds() external pure {
        assertEq(GasConstants.L2_RELAY_CONSTANT_OVERHEAD, 200_000);
        assertEq(GasConstants.L2_RELAY_CALL_OVERHEAD, 40_000);
        assertEq(GasConstants.L2_RELAY_RESERVED_GAS, 40_000);
        assertEq(GasConstants.L2_RELAY_GAS_CHECK_BUFFER, 5_000);
    }
}

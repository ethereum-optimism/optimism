// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Types } from "src/libraries/Types.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";

/// @title OperatorFeeVault_Version_Test
/// @notice Tests the `version` function of the `OperatorFeeVault` contract.
contract OperatorFeeVault_Version_Test is CommonTest {
    /// @notice Tests that version returns a valid semver string.
    function test_version_validFormat_succeeds() external view {
        SemverComp.parse(operatorFeeVault.version());
    }
}

/// @title OperatorFeeVault_Constructor_Test
/// @notice Tests the `constructor` of the `OperatorFeeVault` contract.
contract OperatorFeeVault_Constructor_Test is CommonTest {
    /// @notice Tests that the constructor sets the correct values.
    function test_constructor_succeeds() external view {
        assertEq(operatorFeeVault.RECIPIENT(), Predeploys.BASE_FEE_VAULT);
        assertEq(operatorFeeVault.recipient(), Predeploys.BASE_FEE_VAULT);
        assertEq(operatorFeeVault.MIN_WITHDRAWAL_AMOUNT(), 0);
        assertEq(operatorFeeVault.minWithdrawalAmount(), 0);
        assertEq(uint8(operatorFeeVault.WITHDRAWAL_NETWORK()), uint8(Types.WithdrawalNetwork.L2));
        assertEq(uint8(operatorFeeVault.withdrawalNetwork()), uint8(Types.WithdrawalNetwork.L2));
    }
}

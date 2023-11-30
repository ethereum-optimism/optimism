// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { Bridge_Initializer } from "test/setup/Bridge_Initializer.sol";

// Target contract
import { FeeVault } from "src/universal/FeeVault.sol";

// Test the implementations of the FeeVault
contract FeeVault_Test is Bridge_Initializer {
    /// @dev Tests that the constructor sets the correct values.
    function test_constructor_l1FeeVault_succeeds() external {
        assertEq(l1FeeVault.RECIPIENT(), deploy.cfg().l1FeeVaultRecipient());
        assertEq(l1FeeVault.MIN_WITHDRAWAL_AMOUNT(), deploy.cfg().l1FeeVaultMinimumWithdrawalAmount());
        assertEq(uint8(l1FeeVault.WITHDRAWAL_NETWORK()), uint8(FeeVault.WithdrawalNetwork.L1));
    }

    /// @dev Tests that the constructor sets the correct values.
    function test_constructor_baseFeeVault_succeeds() external {
        assertEq(baseFeeVault.RECIPIENT(), deploy.cfg().baseFeeVaultRecipient());
        assertEq(baseFeeVault.MIN_WITHDRAWAL_AMOUNT(), deploy.cfg().baseFeeVaultMinimumWithdrawalAmount());
        assertEq(uint8(baseFeeVault.WITHDRAWAL_NETWORK()), uint8(FeeVault.WithdrawalNetwork.L1));
    }
}

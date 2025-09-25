// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

/// @title L1FeeVault_Constructor_Test
/// @notice Tests the `constructor` of the `L1FeeVault` contract.
contract L1FeeVault_Constructor_Test is CommonTest {
    /// @notice Tests that the constructor sets the correct values.
    function test_constructor_l1FeeVault_succeeds() external view {
        assertEq(l1FeeVault.RECIPIENT(), deploy.cfg().l1FeeVaultRecipient());
        assertEq(l1FeeVault.recipient(), deploy.cfg().l1FeeVaultRecipient());
        assertEq(l1FeeVault.MIN_WITHDRAWAL_AMOUNT(), deploy.cfg().l1FeeVaultMinimumWithdrawalAmount());
        assertEq(l1FeeVault.minWithdrawalAmount(), deploy.cfg().l1FeeVaultMinimumWithdrawalAmount());
        assertEq(uint8(l1FeeVault.WITHDRAWAL_NETWORK()), deploy.cfg().l1FeeVaultWithdrawalNetwork());
        assertEq(uint8(l1FeeVault.withdrawalNetwork()), deploy.cfg().l1FeeVaultWithdrawalNetwork());
    }
}

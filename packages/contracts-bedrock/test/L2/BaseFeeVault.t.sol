// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Types } from "src/libraries/Types.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";

/// @title BaseFeeVault_Constructor_Test
/// @notice Tests the `constructor` of the `BaseFeeVault` contract.
contract BaseFeeVault_Constructor_Test is CommonTest {
    /// @notice Tests that the constructor sets the correct values.
    function test_constructor_succeeds() external view {
        assertEq(baseFeeVault.RECIPIENT(), deploy.cfg().baseFeeVaultRecipient());
        assertEq(baseFeeVault.recipient(), deploy.cfg().baseFeeVaultRecipient());
        assertEq(baseFeeVault.MIN_WITHDRAWAL_AMOUNT(), deploy.cfg().baseFeeVaultMinimumWithdrawalAmount());
        assertEq(baseFeeVault.minWithdrawalAmount(), deploy.cfg().baseFeeVaultMinimumWithdrawalAmount());
        assertEq(uint8(baseFeeVault.WITHDRAWAL_NETWORK()), uint8(Types.WithdrawalNetwork.L1));
        assertEq(uint8(baseFeeVault.withdrawalNetwork()), uint8(Types.WithdrawalNetwork.L1));
    }
}

/// @title BaseFeeVault_Version_Test
/// @notice Tests the `version` function of the `BaseFeeVault` contract.
contract BaseFeeVault_Version_Test is CommonTest {
    /// @notice Tests that version returns a valid semver string.
    function test_version_validFormat_succeeds() external view {
        SemverComp.parse(baseFeeVault.version());
    }
}

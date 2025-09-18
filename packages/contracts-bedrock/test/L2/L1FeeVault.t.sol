// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Interfaces
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { FeeVault_Uncategorized_Test } from "test/L2/FeeVault.t.sol";

/// @title L1FeeVault_Uncategorized_Test
/// @notice Test contract for the L1FeeVault contract's functionality
contract L1FeeVault_Uncategorized_Test is FeeVault_Uncategorized_Test {
    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();
        recipient = deploy.cfg().l1FeeVaultRecipient();
        feeVaultName = "L1FeeVault";
        minWithdrawalAmount = deploy.cfg().l1FeeVaultMinimumWithdrawalAmount();
        feeVault = IFeeVault(payable(Predeploys.L1_FEE_VAULT));
    }
}

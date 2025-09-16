// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Interfaces
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";

// Libraries
import { Types } from "src/libraries/Types.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { FeeVault_Test } from "test/L2/FeeVault.t.sol";

/// @title BaseFeeVault_Test
/// @notice Test contract for the BaseFeeVault contract's functionality
contract BaseFeeVault_Test is FeeVault_Test {
    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();
        recipient = deploy.cfg().baseFeeVaultRecipient();
        feeVaultName = "BaseFeeVault";
        minWithdrawalAmount = deploy.cfg().baseFeeVaultMinimumWithdrawalAmount();
        feeVault = IFeeVault(payable(Predeploys.BASE_FEE_VAULT));
    }
}

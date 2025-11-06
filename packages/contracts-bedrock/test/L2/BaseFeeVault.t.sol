// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Interfaces
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { FeeVault_Uncategorized_Test } from "test/L2/FeeVault.t.sol";
import { Types } from "src/libraries/Types.sol";

/// @title BaseFeeVault_Uncategorized_Test
/// @notice Test contract for the BaseFeeVault contract's functionality
contract BaseFeeVault_Uncategorized_Test is FeeVault_Uncategorized_Test {
    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();
        recipient = deploy.cfg().baseFeeVaultRecipient();
        feeVaultName = "BaseFeeVault";
        minWithdrawalAmount = deploy.cfg().baseFeeVaultMinimumWithdrawalAmount();
        feeVault = IFeeVault(payable(Predeploys.BASE_FEE_VAULT));
        withdrawalNetwork = Types.WithdrawalNetwork(uint8(deploy.cfg().baseFeeVaultWithdrawalNetwork()));
        // Current recipient is a contract that reverts when receiving fees, so etching empty bytes to it
        vm.etch(recipient, hex"");
    }
}

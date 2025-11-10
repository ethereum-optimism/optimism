// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Interfaces
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";
import { ISequencerFeeVault } from "interfaces/L2/ISequencerFeeVault.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { FeeVault_Uncategorized_Test } from "test/L2/FeeVault.t.sol";
import { Types } from "src/libraries/Types.sol";

/// @title SequencerFeeVault_Uncategorized_Test
/// @notice Test contract for the SequencerFeeVault contract's functionality
contract SequencerFeeVault_Uncategorized_Test is FeeVault_Uncategorized_Test {
    /// @dev Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();
        recipient = deploy.cfg().sequencerFeeVaultRecipient();
        feeVaultName = "SequencerFeeVault";
        minWithdrawalAmount = deploy.cfg().sequencerFeeVaultMinimumWithdrawalAmount();
        feeVault = IFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET));
        withdrawalNetwork = Types.WithdrawalNetwork(uint8(deploy.cfg().sequencerFeeVaultWithdrawalNetwork()));
    }

    function test_constructor_l1FeeWallet_succeeds() external view {
        assertEq(ISequencerFeeVault(payable(address(feeVault))).l1FeeWallet(), recipient);
    }
}

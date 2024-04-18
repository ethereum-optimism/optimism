// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Target contract dependencies
import { FeeVault } from "src/universal/FeeVault.sol";
import { SequencerFeeVault } from "src/L2/SequencerFeeVault.sol";
import { BaseFeeVault } from "src/L2/BaseFeeVault.sol";
import { L1FeeVault } from "src/L2/L1FeeVault.sol";

// Target contract
import { RevenueSharer } from "src/L2/RevenueSharer.sol";

contract RevenueSharer_Test is CommonTest {
    address recipient;
    address remainderRecipient;

    /// @dev Sets up the test suite.
    function setUp() public override {
        super.setUp();
        recipient = deploy.cfg().revenueShareRecipient();
        remainderRecipient = deploy.cfg().revenueShareRemainderRecipient();
    }

    /// @dev Tests that the l1 fee wallet is correct.
    function test_constructor_succeeds() external {
        // TODO
    }

    /// @dev Tests that the fee vault is able to receive ETH.
    function test_execute_succeeds() external {
        // Deal some ETH to the fee vaults, sanity check the results
        vm.deal(address(sequencerFeeVault), 120);
        assertEq(address(sequencerFeeVault).balance, 120);

        vm.deal(address(l1FeeVault), 70);
        assertEq(address(l1FeeVault).balance, 70);

        vm.deal(address(baseFeeVault), 110);
        assertEq(address(baseFeeVault).balance, 110);

        // Setup assertion that an event will be emitted
        // vm.expectEmit(Predeploys.SEQUENCER_FEE_WALLET);
        // emit FeesDisbursed(45, 300);

        // Execute
        revenueSharer.execute();

        // Assert 15% of revenue flows to beneficiary
        assertEq(recipient.balance, 45);

        // Assert 85% of revenue flows to other party
        assertEq(remainderRecipient.balance, 255);

        // Assert RevenueSharer does not accumulate ETH
        assertEq(address(revenueSharer).balance, 0);

        // Assert FeeVaults are depleted
        assertEq(address(sequencerFeeVault).balance, 0);
        assertEq(address(l1FeeVault).balance, 0);
        assertEq(address(baseFeeVault).balance, 0);
    }
}

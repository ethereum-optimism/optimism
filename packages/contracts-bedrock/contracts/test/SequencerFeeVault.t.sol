// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Bridge_Initializer } from "./CommonTest.t.sol";

import { SequencerFeeVault } from "../L2/SequencerFeeVault.sol";
import { StandardBridge } from "../universal/StandardBridge.sol";
import { Predeploys } from "../libraries/Predeploys.sol";

contract SequencerFeeVault_Test is Bridge_Initializer {
    SequencerFeeVault vault = SequencerFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET));
    address constant recipient = address(256);

    event Withdrawal(uint256 value, address to, address from);

    function setUp() public override {
        super.setUp();
        vm.etch(Predeploys.SEQUENCER_FEE_WALLET, address(new SequencerFeeVault(recipient)).code);
        vm.label(Predeploys.SEQUENCER_FEE_WALLET, "SequencerFeeVault");
    }

    function test_minWithdrawalAmount_succeeds() external {
        assertEq(vault.MIN_WITHDRAWAL_AMOUNT(), 10 ether);
    }

    function test_constructor_succeeds() external {
        assertEq(vault.l1FeeWallet(), recipient);
    }

    function test_receive_succeeds() external {
        uint256 balance = address(vault).balance;

        vm.prank(alice);
        (bool success, ) = address(vault).call{ value: 100 }(hex"");

        assertEq(success, true);
        assertEq(address(vault).balance, balance + 100);
    }

    function test_withdraw_notEnough_reverts() external {
        assert(address(vault).balance < vault.MIN_WITHDRAWAL_AMOUNT());

        vm.expectRevert(
            "FeeVault: withdrawal amount must be greater than minimum withdrawal amount"
        );
        vault.withdraw();
    }

    function test_withdraw_succeeds() external {
        uint256 amount = vault.MIN_WITHDRAWAL_AMOUNT() + 1;
        vm.deal(address(vault), amount);

        // No ether has been withdrawn yet
        assertEq(vault.totalProcessed(), 0);

        vm.expectEmit(true, true, true, true, address(Predeploys.SEQUENCER_FEE_WALLET));
        emit Withdrawal(address(vault).balance, vault.RECIPIENT(), address(this));

        // The entire vault's balance is withdrawn
        vm.expectCall(
            Predeploys.L2_STANDARD_BRIDGE,
            address(vault).balance,
            abi.encodeWithSelector(
                StandardBridge.bridgeETHTo.selector,
                vault.l1FeeWallet(),
                20000,
                bytes("")
            )
        );

        vault.withdraw();

        // The withdrawal was successful
        assertEq(vault.totalProcessed(), amount);
    }
}
